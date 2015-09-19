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

package govisor

import (
	"strings"
	"sync"
	"time"
)

const (
	MaxLogRecords = 1000
)

type LogRecord struct {
	Id   int64     `json:"id,string"`
	Time time.Time `json:"time"`
	Text string    `json:"text"`
}

type Log struct {
	records    []LogRecord
	numRecords int
	maxRecords int
	id         int64
	cvs        map[*sync.Cond]bool
	mx         sync.Mutex
}

func (log *Log) lock() {
	log.mx.Lock()
}

func (log *Log) unlock() {
	log.mx.Unlock()
}

// Write implements the Writer interface consumed by Logger.
func (log *Log) Write(b []byte) (int, error) {
	if log.maxRecords == 0 {
		log.maxRecords = MaxLogRecords
	}
	if log.records == nil {
		log.records = make([]LogRecord, log.maxRecords)
		log.numRecords = 0
	}
	str := strings.Trim(string(b), "\n")
	log.lock()
	for _, line := range strings.Split(str, "\n") {
		idx := log.numRecords % log.maxRecords
		log.id++
		log.records[idx].Text = line
		log.records[idx].Id = log.id
		log.records[idx].Time = time.Now()
		// NB: numRecords may actually be more than maxRecords.
		// In that case, we've looped, but we use this really to
		// track the next index.
		log.numRecords++
	}
	for cv := range log.cvs {
		cv.Broadcast()
	}
	log.unlock()
	return len(b), nil
}

func (log *Log) Clear() {
	log.lock()
	log.numRecords = 0
	// We presume that we cannot add new records more quickly than
	// once every nanosecond.
	log.id = time.Now().UnixNano()
	log.unlock()
}

// GetRecords returns the records that are stored, as well as an ID
// suitable for use as an Etag.  The last parameter can be the last ID
// that was checked, in which case this function will return nil immediately
// if the log has not changed since that ID was returned, without duplicating
// any records.  These IDs are suitable for use as an Etag in REST APIs.
// Note that IDs are not unique across different Log instances.
func (log *Log) GetRecords(last int64) ([]LogRecord, int64) {
	log.lock()
	if log.id == last {
		log.unlock()
		return nil, last
	}
	var recs []LogRecord
	cnt := log.numRecords
	cur := log.numRecords
	if log.numRecords > log.maxRecords {
		recs = make([]LogRecord, 0, log.maxRecords)
		cnt = log.maxRecords
	} else {
		recs = make([]LogRecord, 0, log.numRecords)
	}
	if cnt > cur {
		cnt = cur
	}
	index := cur - cnt
	for j := 0; j < cnt; j++ {
		recs = append(recs, log.records[index%log.maxRecords])
		index++
	}
	id := log.id
	log.unlock()
	return recs, id
}

func (log *Log) Watch(last int64, expire time.Duration) int64 {
	expired := false
	var timer *time.Timer
	cv := sync.NewCond(&log.mx)
	if expire > 0 {
		timer = time.AfterFunc(expire, func() {
			log.lock()
			expired = true
			cv.Broadcast()
			log.unlock()
		})
	} else {
		expired = true
	}

	log.lock()
	log.cvs[cv] = true
	for {
		if log.id != last || expired {
			break
		}
		cv.Wait()
	}
	delete(log.cvs, cv)
	if log.id != last {
		last = log.id
	}
	log.unlock()
	if timer != nil {
		timer.Stop()
	}
	return last
}

// NewLog returns a Log instance.
func NewLog() *Log {
	log := &Log{
		maxRecords: MaxLogRecords,
		id:         time.Now().UnixNano(),
		cvs:        make(map[*sync.Cond]bool),
	}
	return log
}
