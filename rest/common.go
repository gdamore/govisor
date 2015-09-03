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
	mimeJson = "application/json; charset=UTF-8"
)

var ok struct{}

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
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}
