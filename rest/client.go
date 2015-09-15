// Copyright 2015 The Govisor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use file except in compliance with the License.
// You may obtain a copy of the license at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rest

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
)

type LogInfo struct {
	name    string
	etag    string
	Records []LogRecord
}

type Client struct {
	user      string // HTTP Basic-Auth
	pass      string
	base      string // URI to root of tree on server
	auth      bool
	client    *http.Client
	transport *http.Transport

	// Cached data
	manager  *ManagerInfo
	services map[string]*ServiceInfo // service entries
	names    []string                // service names
	etag     string                  // etag for list of services
	logs     map[string]*LogInfo
	cv       *sync.Cond
	lock     sync.Mutex
}

func (c *Client) SetAuth(user string, pass string) {
	c.user = user
	c.pass = pass
	c.auth = true
}

func (c *Client) url(name string) string {
	if name == "" {
		return c.base + "/services"
	}
	return c.base + "/services/" + url.QueryEscape(name)
}

func (c *Client) Watch(ctx context.Context, etag string) (string, error) {
	var e error
	c.lock.Lock()
	if c.manager != nil && etag == "" {
		etag = c.manager.etag
		c.lock.Unlock()
		return etag, nil
	}
	c.lock.Unlock()

	minfo := &ManagerInfo{}
	if minfo.etag, e = c.poll(ctx, c.base, etag, 300, minfo); e != nil {
		return "", e
	}
	if minfo.etag != "" {
		c.lock.Lock()
		if c.manager == nil || c.manager.etag != minfo.etag {
			c.manager = minfo
		}
		c.lock.Unlock()
		etag = minfo.etag
	}
	return etag, nil
}

func (c *Client) pollServices(ctx context.Context, secs int) ([]string, error) {

	var e error
	v := []string{}

	c.lock.Lock()
	otag := c.etag
	etag := ""
	onames := c.names
	c.lock.Unlock()

	if etag, e = c.poll(ctx, c.url(""), otag, secs, &v); e != nil {
		return nil, e
	}
	if etag == "" || etag == otag {
		return onames, nil
	}
	services := make(map[string]*ServiceInfo)

	c.lock.Lock()
	c.etag = etag
	c.names = v
	// save the services we found
	for _, n := range v {
		if svc, ok := c.services[n]; ok {
			services[n] = svc
			delete(c.services, n)
		}
	}
	c.services = services
	// we let GC clean up the old hash
	c.lock.Unlock()

	return v, nil
}

// Services returns a list of service names known to the implementation
func (c *Client) Services() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return c.pollServices(ctx, 0)
}

func (c *Client) pollService(ctx context.Context, name string, secs int, last *ServiceInfo) (*ServiceInfo, error) {

	v := &ServiceInfo{}
	c.lock.Lock()
	osvc, ok := c.services[name]
	c.lock.Unlock()

	otag := ""
	if last == nil {
		secs = 0
	} else if ok && last.etag != osvc.etag {
		// If we asked for a check against a value, and the cached
		// value is not the same, then we can return the cached value.
		return osvc, nil
	} else {
		// Either we didn't have a value cached, or they are the same.
		otag = last.etag
	}

	etag, e := c.poll(ctx, c.url(name), otag, secs, v)
	if e != nil {
		c.lock.Lock()
		delete(c.services, name)
		c.lock.Unlock()
		return nil, e
	}
	if etag == "" {
		return osvc, nil
	}
	v.etag = etag
	c.lock.Lock()
	c.services[name] = v
	c.lock.Unlock()
	return v, nil
}

func (c *Client) GetService(name string) (*ServiceInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return c.pollService(ctx, name, 0, nil)
}

func (c *Client) WatchService(ctx context.Context, name string, last *ServiceInfo) (*ServiceInfo, error) {
	return c.pollService(ctx, name, 300, last)
}

// poll issues an HTTP GET against the URL, optionally checking for a cache,
// including optionally issuing a long poll that tries to wait until the
// value changes.  The return values are the new Etag and any error.  If the
// value did not change, then the returned etag will be "", but the error will
// be nil.
type chanResp struct {
	r *http.Response
	e error
}

func (c *Client) poll(ctx context.Context, url string, etag string, wait int, v interface{}) (string, error) {

	req, e := http.NewRequest("GET", url, nil)
	if e != nil {
		return "", e
	}
	if c.auth {
		req.SetBasicAuth(c.user, c.pass)
	}
	client := c.client
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
		if wait > 0 {
			req.Header.Set(PollEtagHeader, etag)
			req.Header.Set(PollTimeHeader, strconv.Itoa(wait))
		}
	}

	ch := make(chan chanResp)
	go func() {
		res, e := client.Do(req)
		ch <- chanResp{r: res, e: e}
	}()

	var res *http.Response
	select {
	case <-ctx.Done():
		c.transport.CancelRequest(req)
		<-ch // wait for the Do to finish (or be canceled)
		return "", ctx.Err()
	case cr := <-ch:
		res = cr.r
		e = cr.e
	}
	if e != nil {
		return "", e
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotModified {
		return "", nil
	}
	if res.StatusCode != http.StatusOK {
		return "", &Error{Code: res.StatusCode, Message: res.Status}
	}
	body, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return "", e
	}
	if e := json.Unmarshal(body, v); e != nil {
		return "", e
	}
	return res.Header.Get("Etag"), nil
}

func (c *Client) post(url string) error {
	req, e := http.NewRequest("POST", url, strings.NewReader(""))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "text/plain") // we don't really care
	if c.auth {
		req.SetBasicAuth(c.user, c.pass)
	}
	res, e := c.client.Do(req)
	if e != nil {
		return e
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return &Error{Code: res.StatusCode, Message: res.Status}
	}
	return nil
}

func (c *Client) postService(name string, action string) error {
	return c.post(c.url(name) + "/" + action)
}

func (c *Client) EnableService(name string) error {
	return c.postService(name, "enable")
}

func (c *Client) DisableService(name string) error {
	return c.postService(name, "disable")
}

func (c *Client) ClearService(name string) error {
	return c.postService(name, "clear")
}

func (c *Client) RestartService(name string) error {
	return c.postService(name, "restart")
}

func (c *Client) pollLog(ctx context.Context, name string, secs int, last *LogInfo) (*LogInfo, error) {

	v := &LogInfo{}

	c.lock.Lock()
	cached, ok := c.logs[name]
	c.lock.Unlock()

	otag := ""

	if last == nil {
		secs = 0
	} else if ok && last.etag != cached.etag {
		// TODO: We should modify this to validate the cached etag
		// VERIFY THIS
		//
		secs = 0
		otag = cached.etag
		//
		// If we asked for a check against a value, and the cached
		// value is not the same, then we can return the cached value.
		//return cached, nil
	} else {
		// Either we didn't have a value cached, or they are the same.
		otag = last.etag
	}

	url := c.url(name) + "/log"
	if name == "" {
		url = c.base + "/log"
	}

	etag, e := c.poll(ctx, url, otag, secs, &v.Records)
	if e != nil {
		c.lock.Lock()
		delete(c.logs, name)
		c.lock.Unlock()
		return nil, e
	}
	if etag == "" {
		return cached, nil
	}
	v.etag = etag
	c.lock.Lock()
	c.logs[name] = v
	c.lock.Unlock()

	return v, nil
}

func (c *Client) WatchLog(ctx context.Context, name string, last *LogInfo) (*LogInfo, error) {

	// Let the poll wait for up to 300 secs (5 minutes).
	return c.pollLog(ctx, name, 300, last)
}

func (c *Client) GetLog(name string) (*LogInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return c.pollLog(ctx, name, 0, nil)
}

// GetServiceLog returns the log, utilizing caching checks.  It does not
// wait for changes to the log.
//func (c *Client) GetServiceLog(name string) ([]LogRecord, error) {
//	return c.GetLog(name)
//}

// NewClient returns a Client handle.  The transport maybe nil to use
// a default transport, but it may also be adjusted to support additional
// options such as TLS.  baseURI is the base URL to use.
func NewClient(t *http.Transport, baseURI string) *Client {
	if t == nil {
		t = &http.Transport{}
	}
	c := &Client{
		transport: t,
		base:      baseURI,
		client:    &http.Client{Transport: t},
		logs:      make(map[string]*LogInfo),
	}
	return c
}
