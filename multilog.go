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
	"log"
	"strings"
	"sync"
)

// MultiLogger implements a wrapper around log.Logger, that permits a single
// logger interface to be used to fan out multiple logs.  The idea is that it
// implements an io.Writer, which breaks up the lines and delivers them
// each to the various contained loggers.  The contained loggers may have
// their own Prefix and Flags, and those shall not interfere with the parent.
type MultiLogger struct {
	log     *log.Logger
	loggers []*log.Logger
	lock    sync.Mutex
}

// Write implements the io.Writer, suitable for use with Logger.  It is
// expected that the input is text, delimited by newlines, and delivered
// an entire line at a time.  This isn't exactly io.Writer, but it is the
// semantic to which the log.Logger interface conforms.
func (l *MultiLogger) Write(b []byte) (int, error) {
	lines := strings.Split(strings.Trim(string(b), "\n"), "\n")
	l.lock.Lock()
	for _, line := range lines {
		for _, logger := range l.loggers {
			logger.Println(line)
		}
	}
	l.lock.Unlock()
	return len(b), nil
}

// AddLogger adds a logger to the MultiLogger.  Once called, all new log entries
// will be fanned out to this logger, as well as any others that may have been
// registered earlier.  A logger can only be added once.
func (l *MultiLogger) AddLogger(logger *log.Logger) {
	l.lock.Lock()
	defer l.lock.Unlock()
	for _, x := range l.loggers {
		if x == logger {
			return
		}
	}
	l.loggers = append(l.loggers, logger)
}

// DeleteLogger is removes a logger from the list of destinations that logged
// events are fanned out to.
func (l *MultiLogger) DelLogger(logger *log.Logger) {
	l.lock.Lock()
	defer l.lock.Unlock()

	for i, x := range l.loggers {
		if x == logger {
			l.loggers = append(l.loggers[:i], l.loggers[i+1:]...)
			break
		}
	}
}

// SetPrefix applies the prefix to every registered logger.
func (l *MultiLogger) SetPrefix(prefix string) {
	l.lock.Lock()
	for _, x := range l.loggers {
		x.SetPrefix(prefix)
	}
	l.lock.Unlock()
}

// SetFlags applies the flags to every registered logger.
func (l *MultiLogger) SetFlags(flags int) {
	l.lock.Lock()
	for _, x := range l.loggers {
		x.SetFlags(flags)
	}
	l.lock.Unlock()
}

func (l *MultiLogger) Logger() *log.Logger {
	return l.log
}

func NewMultiLogger() *MultiLogger {
	m := &MultiLogger{}
	m.log = log.New(m, "", 0)
	return m
}
