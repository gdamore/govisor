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
	"io"
	"log"
	"os"
	"path"
	"runtime"
	"sync"
	"time"
)

type Manager struct {
	services   map[*Service]bool
	name       string
	baseDir    string
	logger     *log.Logger
	writer     io.Writer
	cleanup    bool
	monitoring bool
	mx         sync.Mutex
}

func (m *Manager) lock() {
	m.mx.Lock()
}

func (m *Manager) unlock() {
	m.mx.Unlock()
}

// Name returns the name the manager was allocated with.  This makes it
// possible to distinguish between separate manager instances.  This name
// can influence logged messages and file/directory names.
func (m *Manager) Name() string {
	return m.name
}

// AddService adds a service, registering it, to the manager.
func (m *Manager) AddService(s *Service) {
	m.lock()
	s.setManager(m)
	m.unlock()
}

// DeleteService deletes a service from the manager.
func (m *Manager) DeleteService(s *Service) error {
	m.lock()
	if s.enabled {
		m.unlock()
		return ErrIsEnabled
	}
	s.delManager()
	m.unlock()
	return nil
}

// Services returns all of our services.  Note that the order is
// arbitrary.  (At present it happens to be done based on order of
// addition.
func (m *Manager) Services() []*Service {
	m.lock()
	rv := make([]*Service, 0, len(m.services))
	for s := range m.services {
		rv = append(rv, s)
	}
	m.unlock()
	return rv
}

// FindServices finds the list of services that have either a matching
// Name, or Provides.  That is, they find all of our services, where the
// service.Match() would return true for the string match.
func (m *Manager) FindServices(match string) []*Service {
	rv := []*Service{}
	m.lock()
	for s := range m.services {
		if s.Matches(match) {
			rv = append(rv, s)
		}
	}
	m.unlock()
	return rv
}

func (m *Manager) setBaseDir() {
	m.baseDir = os.Getenv("GOVISORDIR")
	switch runtime.GOOS {
	case "nacl", "plan9":
		m.baseDir = ""
	case "windows":
		if len(m.baseDir) == 0 {
			m.baseDir = os.Getenv("HOME")
		}
		if len(m.baseDir) == 0 {
			m.baseDir = "C:\\"
		}
	default:
		if len(m.baseDir) == 0 {
			if os.Geteuid() == 0 {
				m.baseDir = "/var"
			} else {
				m.baseDir = os.Getenv("HOME")
			}
		}
		if len(m.baseDir) == 0 {
			m.baseDir = "."
		}
	}
}

// SetLogger sets the logger to use.  This allows a framework to use a single
// logger for everything.  Note that this must be called before services are
// added in order to have any effect.
func (m *Manager) SetLogger(l *log.Logger) {
	m.logger = l
}

// SetLogWriter works like SetLogger, except that it only sets an output
// writer.  This is probably more convenient for most loggers.
func (m *Manager) SetLogWriter(w io.Writer) {
	m.logger = log.New(w, "["+m.Name()+"] ", log.LstdFlags)
	m.writer = w
}

func (m *Manager) getLogger(s *Service) *log.Logger {

	flags := log.LstdFlags
	if m.logger != nil {
		flags = m.logger.Flags()
	}
	if m.writer != nil {
		prefix := "[" + s.Name() + "] "
		return log.New(m.writer, prefix, flags)
	} else if m.logger != nil {
		return m.logger
	}

	// Default logger
	prefix := "[" + s.Name() + "] "
	if len(m.baseDir) == 0 {
		return log.New(os.Stderr, prefix, flags)
	}

	if runtime.GOOS == "windows" {
		// XXX: this needs to generate a proper Windows service log
		return log.New(os.Stderr, prefix, log.LstdFlags)
	}

	// XXX: service specific file names?
	f := path.Join(m.baseDir, m.Name(), s.Name()+".log")

	w, e := os.OpenFile(f, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if e != nil {
		return log.New(os.Stderr, prefix, log.LstdFlags)
	}
	return log.New(w, prefix, log.LstdFlags)
}

func (m *Manager) monitor() {
	finish := false
	for !finish {
		m.lock()
		if m.monitoring {
			for s := range m.services {
				if s.enabled {
					if e := s.checkService(); e != nil {
						s.selfHeal()
					}
				}
			}
		}
		if m.cleanup {
			m.monitoring = false
			finish = true
		}
		m.unlock()

		// a "prime" number of milliseconds, to ensure a more
		// or less even distribution of clock events
		time.Sleep(time.Millisecond * 587)
	}
}

// notify is called asynchronously by services, when they detect a failure.
// It MUST NOT be called by the service as part of a synchronous call to
// the check routine.  We do add a check to prevent infinite recursion, but
// again, the caller should be careful not to do this.  Notification should
// only be done when a service transitions from succeeding to failing, or vice
// versa.
func (m *Manager) notify(s *Service) {
	if s.checking {
		// No need for notification, and lets not recurse!
		return
	}
	if s.enabled {
		if e := s.checkService(); e != nil {
			s.selfHeal()
		}
	}
}

func (m *Manager) StopMonitoring() {
	m.lock()
	m.monitoring = false
	m.unlock()
}

func (m *Manager) StartMonitoring() {
	m.lock()
	m.monitoring = true
	m.unlock()
}

// Shutdown stops all services, and stops monitoring too.  Finally, it removes
// them all from the manager.  Think of this as effectively tearing down the
// entire thing.
func (m *Manager) Shutdown() {
	m.lock()
	m.monitoring = false
	for s := range m.services {
		s.enabled = false
		s.stopRecurse()
		s.delManager()
	}
	m.unlock()
}

func NewManager(name string) *Manager {
	if name == "" {
		name = "govisor"
	}
	m := &Manager{name: name}
	m.services = make(map[*Service]bool)
	m.setBaseDir()
	go m.monitor()
	return m
}
