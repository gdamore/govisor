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
	"strings"
	"sync"
	"time"
)

type Client struct {
	user    string // HTTP Basic-Auth
	pass    string
	base    string // URI to root of tree on server
	auth    bool
	client  *http.Client
	pclient *http.Client

	// Cached data
	manager  *ManagerInfo
	services map[string]*ServiceInfo // service entries
	names    []string                // service names
	etag     string                  // etag for list of services
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

// Watch just monitors for a change in the global serial number.  This can
// be done in a client loop that runs, and watches for updates.  In other
// words, use this for long polling.  It does update the cached ManagerInfo.
func (c *Client) Watch() error {
	var e error
	c.lock.Lock()
	otag := ""
	wait := true
	if c.manager != nil {
		otag = c.manager.etag
		wait = true
	}
	c.lock.Unlock()

	minfo := &ManagerInfo{}
	if minfo.etag, e = c.poll(c.base, otag, wait, minfo); e != nil {
		return e
	}
	c.lock.Lock()
	if minfo.etag != "" && minfo.etag != otag {
		c.manager = minfo
	}
	c.lock.Unlock()
	return nil
}

// Services returns a list of service names known to the implementation
func (c *Client) Services() ([]string, error) {
	var e error
	v := []string{}
	c.lock.Lock()
	otag := c.etag
	etag := ""
	onames := c.names
	c.lock.Unlock()

	if etag, e = c.poll(c.url(""), otag, false, &v); e != nil {
		return nil, e
	}
	if etag == "" || etag == otag {
		return onames, nil
	}
	services := make(map[string]*ServiceInfo)
	c.lock.Lock()
	c.etag = etag
	c.names = v
	// move over the ones we found
	for _, n := range v {
		if svc, ok := c.services[n]; ok {
			services[n] = svc
			delete(c.services, n)
		}
	}
	// delete remaining ones -- does this buy us anything?
	for n := range c.services {
		delete(c.services, n)
	}
	c.services = services
	c.lock.Unlock()
	return v, nil
}

func (c *Client) GetService(name string) (*ServiceInfo, error) {
	var e error
	v := &ServiceInfo{}
	c.lock.Lock()
	osvc, ok := c.services[name]
	c.lock.Unlock()

	otag := ""
	etag := ""
	if ok {
		otag = osvc.etag
	}
	if etag, e = c.poll(c.url(name), otag, false, v); e != nil {
		c.lock.Lock()
		delete(c.services, name)
		c.lock.Unlock()
		return nil, e
	}
	if etag == "" {
		return osvc, nil
	}
	c.lock.Lock()
	if s, ok := c.services[name]; ok && s == osvc {
		c.services[name] = v
	}
	c.lock.Unlock()
	return v, nil
}

// poll issues an HTTP GET against the URL, optionally checking for a cache,
// including optionally issuing a long poll that tries to wait until the
// value changes.  The return values are the new Etag and any error.  If the
// value did not change, then the returned etag will be "", but the error will
// be nil.
func (c *Client) poll(url string, etag string, wait bool, v interface{}) (string, error) {

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
		if wait {
			// We use the poll client (pclient), which has a longer
			// (10m) timeout
			client = c.pclient
			req.Header.Set(PollEtagHeader, etag)
			req.Header.Set(PollTimeHeader, "300") // 5 minutes
		}
	}

	res, e := client.Do(req)
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

func (c *Client) get(url string, v interface{}) error {
	req, e := http.NewRequest("GET", url, nil)
	if e != nil {
		return e
	}
	if c.auth {
		req.SetBasicAuth(c.user, c.pass)
	}
	res, e := c.client.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return &Error{Code: res.StatusCode, Message: res.Status}
	}
	body, e := ioutil.ReadAll(res.Body)
	if e != nil {
		return e
	}
	if e := json.Unmarshal(body, v); e != nil {
		return e
	}
	return nil
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

func (c *Client) GetServiceLog(name string) ([]string, error) {
	v := []string{}
	if e := c.get(c.url(name)+"/log", &v); e != nil {
		return nil, e
	}
	return v, nil
}

// NewClient returns a Client handle.  The transport maybe nil to use
// a default transport, but it may also be adjusted to support additional
// options such as TLS.  baseURI is the base URL to use.
func NewClient(t *http.Transport, baseURI string) *Client {
	c := &Client{
		base: baseURI,
	}
	// No reason for us to ever have to wait more than a second for
	// a transfer.
	c.client = &http.Client{
		Timeout: time.Second * 2,
	}
	// Long polling is supposed to return in under 5 minutes
	c.pclient = &http.Client{
		Timeout: time.Minute * 10,
	}
	if t != nil {
		c.client.Transport = t
		c.pclient.Transport = t
	}
	return c
}
