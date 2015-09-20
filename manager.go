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
	"runtime"
	"sync"
	"time"
)

type Manager struct {
	services   map[*Service]bool
	name       string
	baseDir    string
	logger     *log.Logger
	log        *Log
	mlog       *MultiLogger
	writer     io.Writer
	cleanup    bool
	monitoring bool
	serial     int64
	listSerial int64
	listStamp  time.Time
	createTime time.Time
	updateTime time.Time
	mx         sync.Mutex
	cvs        map[*sync.Cond]bool
}

type ManagerInfo struct {
	Name       string
	Serial     int64
	UpdateTime time.Time
	CreateTime time.Time
}

func (m *Manager) lock() {
	m.mx.Lock()
}

func (m *Manager) unlock() {
	m.mx.Unlock()
}

func (m *Manager) wakeUp() {
	// NB: If the lock is not held here, then there is a risk
	// that the woken goroutines won't get see the updated
	// serial number!!
	for cv := range m.cvs {
		cv.Broadcast()
	}
}

// bumpSerial increments the serial and notifies watchers.  It returns
// the new serial number, so that it can be stored in services.
// Call with lock held.
func (m *Manager) bumpSerial() int64 {
	m.updateTime = time.Now()
	m.serial++
	rv := m.serial
	m.wakeUp()
	return rv
}

// watchSerial monitors for a change in a specific serial number.  It returns
// the new serial number when it changes.  If the serial number has not
// changed in the given duration then the old value is returned.  A poll
// can be done by supplying 0 for the expiration.
func (m *Manager) watchSerial(old int64, src *int64, expire time.Duration) int64 {
	expired := false
	cv := sync.NewCond(&m.mx)
	var timer *time.Timer
	var rv int64

	// Schedule timeout
	if expire > 0 {
		timer = time.AfterFunc(expire, func() {
			m.lock()
			expired = true
			cv.Broadcast()
			m.unlock()
		})
	} else {
		expired = true
	}

	m.lock()
	m.cvs[cv] = true
	for {
		rv = *src
		if rv != old || expired {
			break
		}
		cv.Wait()
	}
	delete(m.cvs, cv)
	m.unlock()
	if timer != nil {
		timer.Stop()
	}
	return rv
}

// WatchSerial monitors for a change in the global serial number.
func (m *Manager) WatchSerial(old int64, expire time.Duration) int64 {
	return m.watchSerial(old, &m.serial, expire)
}

// WatchServices monitors for a change in the list of services.
func (m *Manager) WatchServices(old int64, expire time.Duration) int64 {
	return m.watchSerial(old, &m.listSerial, expire)
}

// Serial returns the global serial number.  This is incremented
// anytime a service has a state change.
func (m *Manager) Serial() int64 {
	m.lock()
	rv := m.serial
	m.unlock()
	return rv
}

// Name returns the name the manager was allocated with.  This makes it
// possible to distinguish between separate manager instances.  This name
// can influence logged messages and file/directory names.
func (m *Manager) Name() string {
	return m.name
}

// GetInfo returns top-level information about the Manager.  This is done
// in a manner that ensures that the info is consistent.
func (m *Manager) GetInfo() *ManagerInfo {
	m.lock()
	i := &ManagerInfo{
		Name:       m.name,
		Serial:     m.serial,
		CreateTime: m.createTime,
		UpdateTime: m.updateTime,
	}
	m.unlock()
	return i
}

// AddService adds a service, registering it, to the manager.
func (m *Manager) AddService(s *Service) {
	m.lock()
	s.setManager(m)
	m.listSerial = m.bumpSerial()
	s.serial = m.bumpSerial()
	m.listStamp = time.Now()
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
	m.listSerial = m.bumpSerial()
	s.serial = m.bumpSerial()
	m.listStamp = time.Now()
	m.unlock()
	return nil
}

// Services returns all of our services.  Note that the order is
// arbitrary.  (At present it happens to be done based on order of
// addition.)
func (m *Manager) Services() ([]*Service, int64, time.Time) {
	m.lock()
	rv := make([]*Service, 0, len(m.services))
	for s := range m.services {
		rv = append(rv, s)
	}
	ts := m.listStamp
	sn := m.listSerial
	m.unlock()
	return rv, sn, ts
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

// SetLogger is used to establish a logger.  It overrides the default, so it
// shouldn't be used unless you want to control all logging.
func (m *Manager) SetLogger(l *log.Logger) {
	if m.logger != nil {
		m.mlog.DelLogger(m.logger)
	}
	m.logger = l
	m.mlog.AddLogger(l)
}

func (m *Manager) getLogger(s *Service) *log.Logger {

	return log.New(m.mlog, "", 0)
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

func (m *Manager) logf(format string, v ...interface{}) {
	if m.logger != nil {
		m.logger.Printf(format, v...)
	} else {
		log.Printf(format, v...)
	}
}

func (m *Manager) StopMonitoring() {
	m.lock()
	m.monitoring = false
	m.unlock()
	m.logf("*** Govisor stopping monitoring: %s ***", m.name)
}

func (m *Manager) StartMonitoring() {
	m.logf("*** Govisor starting monitoring: %s ***", m.name)
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
		s.stopRecurse("Shutting down")
		s.delManager()
	}
	m.unlock()
	m.logf("*** Govisor shut down: %s ***", m.name)
}

func (m *Manager) GetLog(lastid int64) ([]LogRecord, int64) {
	m.lock()
	defer m.unlock()
	return m.log.GetRecords(lastid)
}

func (m *Manager) WatchLog(old int64, expire time.Duration) int64 {
	return m.log.Watch(old, expire)
}

func NewManager(name string) *Manager {
	if name == "" {
		name = "govisor"
	}
	// We set the origin serial number to the current timestamp in nsec.
	// The assumption here is that we won't have changes to serial number
	// occur at frequency > 1GHz.  Hence, it should be safe for us to use
	// these as unique values, and this may help clients that cache force
	// an invalidation if the server for some reason restarts.
	m := &Manager{name: name, serial: time.Now().UnixNano()}
	m.services = make(map[*Service]bool)
	m.cvs = make(map[*sync.Cond]bool)
	m.createTime = time.Now()
	m.updateTime = m.createTime
	m.mlog = NewMultiLogger()
	m.log = NewLog()
	m.mlog.AddLogger(log.New(m.log, "", 0))
	m.logger = log.New(os.Stderr, "", 0)
	m.setBaseDir()
	go m.monitor()
	return m
}
