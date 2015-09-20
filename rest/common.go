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
	"time"
)

const (
	MimeJson = "application/json; charset=UTF-8"
)

const (
	// PollHeader should be set to the last Etag on an incoming request.
	// If set we will wait until the resource has an ETag that is different
	// from the supplied value, yielding a form of long polling.
	// This is only valid with GET requests.  Note that the service may
	// return early without actually waiting, even if the ETag has not
	// changed.  Typically there is a default timeout of around a minute
	// to make sure that the client is alive and well.
	PollEtagHeader = "X-Govisor-Poll-Etag"
	PollTimeHeader = "X-Govisor-Poll-Time"
)

var ok struct{}

type ManagerInfo struct {
	Name       string    `json:"name"`
	Serial     string    `json:"serial"`
	CreateTime time.Time `json:"created"`
	UpdateTime time.Time `json:"updated"`
	etag       string
}

type ServiceInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	Running     bool      `json:"running"`
	Failed      bool      `json:"failed"`
	Provides    []string  `json:"provides"`
	Depends     []string  `json:"depends"`
	Conflicts   []string  `json:"conflicts"`
	Status      string    `json:"status"`
	TimeStamp   time.Time `json:"tstamp"`
	Serial      string    `json:"serial"`
	etag        string
}

type LogRecord struct {
	Id   string    `json:"id"`
	Time time.Time `json:"time"`
	Text string    `json:"text"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}
