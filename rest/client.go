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
	"time"
)

type Client struct {
	user   string // HTTP Basic-Auth
	pass   string
	base   string // URI to root of tree on server
	auth   bool
	client *http.Client
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

// Services returns a list of service names known to the implementation
func (c *Client) Services() ([]string, error) {
	v := []string{}
	if e := c.get(c.url(""), &v); e != nil {
		return nil, e
	}
	return v, nil
}

func (c *Client) GetService(name string) (*ServiceInfo, error) {
	v := &ServiceInfo{}
	if e := c.get(c.url(name), v); e != nil {
		return nil, e
	}
	return v, nil
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
	if t != nil {
		c.client.Transport = t
	}
	return c
}
