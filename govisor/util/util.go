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

// Package util is used for internal implementation bits in the CLI/UI.
package util

import (
	"fmt"
	"sort"
	"time"

	"github.com/gdamore/govisor/rest"
)

func Status(s *rest.ServiceInfo) string {
	if !s.Enabled {
		return "disabled"
	}
	if s.Failed {
		return "failed"
	}
	if s.Running {
		return "running"
	}
	return "standby"
}

func FormatDuration(d time.Duration) string {

	sec := int((d % time.Minute) / time.Second)
	min := int((d % time.Hour) / time.Minute)
	hour := int(d / time.Hour)

	return fmt.Sprintf("%d:%02d:%02d", hour, min, sec)
}

type sorted []*rest.ServiceInfo

func (s sorted) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sorted) Len() int {
	return len(s)
}

func (s sorted) Less(i, j int) bool {
	a := s[i]
	b := s[j]

	if a.Failed != b.Failed {
		// put failed items at front
		return a.Failed
	}
	if a.Enabled != b.Enabled {
		// enabled in front of non-enabled items
		return a.Enabled
	}
	// We don't worry about suspended items vs. running -- no clear order
	// there.  We just sort based on name
	return a.Name < b.Name
}

func SortServices(items []*rest.ServiceInfo) {
	sort.Sort(sorted(items))
}
