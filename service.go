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
	"time"
)

// Service describes a generic system service -- such as a process, or
// group of processes.  Applications are expected to use the Service
// structure to interact with all managed services.
//
// Implementors can provide custom services (which may be any kind of entity)
// by implementing the Provider interface.
//
// Service methods are not thread safe, until the service is added to a
// Manager.  Once the service is added to a Manager, the Manager's lock
// will protect concurrent accesses.
//
// Services go through a number of possible states as illustrated in the
// following state diagram.  Note that these states are logical, as there is
// no formal state machine in the code.  This diagram is for illustration
// purposes only.
//
//                   +------------+
//                   |            |
//         +--------->  Disabled  <-------+
//         |         |            |       |
//         |         +----+--A----+       |
//         |              |  |            |
//   +-----+----+    +----V--+---+        |
//   |          |    |           |        |
//   |  Failed  +---->  DepWait  <----+   |
//   |          |    |           |    |   |
//   +-----A--A-+    +----+------+    |   |
//         |  |           |           |   |
//         |  |      +----v-------+   |   |
//         |  |      |            |   |   |
//         |  +------+  Starting  |   |   |
//         |         |            |   |   |
//         |         +----+-------+   |   |
//         |              |           |   |
//         |          +---V---+       |   |
//         |          |       |       |   |
//         +----------+  Run  +-------+---+
//                    |       |
//                    +-------+
//
type Service struct {
	prov       Provider
	mgr        *Manager
	name       string
	desc       string
	depends    []string
	conflicts  []string
	provides   []string
	enabled    bool
	running    bool
	stopping   bool
	failed     bool
	restart    bool
	checking   bool
	err        error
	parents    map[string]map[*Service]bool
	children   map[*Service]bool
	incompat   map[*Service]bool
	logger     *log.Logger
	stamp      time.Time
	reason     string
	starts     int
	rateLog    bool
	rateLimit  int
	ratePeriod time.Duration
	startTimes []time.Time
	notify     func()
	slog       *ServiceLog
	mlog       *MultiLogger
}

const maxLogRecords = 1000

type ServiceLog struct {
	Records    []string
	NumRecords int
}

func (s *ServiceLog) Write(b []byte) (int, error) {
	if s.Records == nil {
		s.Records = make([]string, maxLogRecords)
	}
	str := strings.Trim(string(b), "\n")
	for _, line := range strings.Split(str, "\n") {
		s.Records[s.NumRecords%len(s.Records)] = line
		s.NumRecords++
	}
	return len(b), nil
}

// The service name.  This takes either the form <base> or <base>:<variant>.
// Except for the colon used to separate the <base> from <variant>, no
// punctuation characters other than underscores are permitted.  When
// attempting to resolve dependencies, a dependency can list either the
// full <base>:<variant> name or just <base>.  In the former case, the
// full service name must match.  In the latter case, any service with
// the same <base> component matches.
func (s *Service) Name() string {
	return s.name
}

// Description returns a descriptive name for the service.  If possible,
// user interfaces should try to allocate at least 32 characters of horizontal
// space when displaying descriptions.
func (s *Service) Description() string {
	return s.desc
}

// Provides is a way to indicate other service names that this service
// offers. This permits a service instance to support multiple real
// capabilities, or to provide multiple aliases.  For example, a daemon
// might offer "http" and "ftp" both.
func (s *Service) Provides() []string {
	return s.provides
}

// Depends returns a list of service names.  See the Name description
// for how these are used.
func (s *Service) Depends() []string {
	return s.depends
}

// Status returns the most reason status message, and the time when the
// status was recorded.
func (s *Service) Status() (string, time.Time) {
	if m := s.mgr; m != nil {
		m.lock()
		defer m.unlock()
	}
	return s.reason, s.stamp
}

// Conflicts returns a list of strings or service names that
// cannot be enabled with this one.  The system will make sure that
// attempts to enable the service are rejected.  Note that the scope
// of conflict is limited to a single Manager; that is the check will
// not prevent two conflicting services running under the control of
// different Managers.
func (s *Service) Conflicts() []string {
	return s.conflicts
}

// Enabled checks if a service is enabled.
func (s *Service) Enabled() bool {
	if m := s.mgr; m == nil {
		return false
	} else {
		m.lock()
		rv := s.enabled
		m.unlock()
		return rv
	}
}

// Running checks if a service is running.  This will be false if the
// service has failed for any reason, or is unable to run due to a missing
// dependency.
func (s *Service) Running() bool {
	if m := s.mgr; m == nil {
		return false
	} else {
		m.lock()
		rv := s.running && !s.stopping
		m.unlock()
		return rv
	}
}

// Failed returns true if the service is in a failure state.
func (s *Service) Failed() bool {
	if m := s.mgr; m == nil {
		return false
	} else {
		m.lock()
		rv := s.failed
		m.unlock()
		return rv
	}
}

// Enable enables the service.  This will also start any services that may
// have not been running due to unsatisfied dependencies, but which now
// are able to (and were otherwise enabled.)
func (s *Service) Enable() error {
	if s.mgr == nil {
		return ErrNoManager
	}
	s.mgr.lock()
	defer s.mgr.unlock()

	if s.enabled {
		return nil
	}

	for c := range s.incompat {
		if c.enabled {
			s.logf("Cannot enable %s: conflicts with %s",
				s.Name(), c.Name())
			return ErrConflict
		}
	}
	s.reason = "Waiting to start"
	s.stamp = time.Now()
	s.logf("Enabling service %s", s.Name())
	s.enabled = true
	s.starts = 0
	s.startRecurse("Enabled service")
	return nil
}

// Disable disables the service, stopping it.  It also will stop any services
// which will no longer have satisfied dependencies as a result of this
// service being disabled.  It also clears the error state.
func (s *Service) Disable() error {
	if s.mgr == nil {
		return ErrNoManager
	}
	s.mgr.lock()
	defer s.mgr.unlock()

	if !s.enabled {
		return nil
	}

	s.logf("Disabling service %s", s.Name())
	s.stamp = time.Now()
	s.reason = "Disabled service"
	s.enabled = false
	s.failed = false
	s.err = nil
	s.stopRecurse("Disabled service")
	return nil
}

// Restart restarts a service.  It also clears any failure condition
// that may have occurred.
func (s *Service) Restart() error {
	if s.mgr == nil {
		return ErrNoManager
	}

	s.mgr.lock()
	defer s.mgr.unlock()

	if !s.enabled {
		return nil
	}

	s.logf("Restarting service %s", s.Name())
	s.enabled = false
	s.stopRecurse("Restarted service")

	s.stamp = time.Now()
	s.reason = "Restarted service"
	s.starts = 0
	s.failed = false
	s.err = nil
	s.enabled = true
	s.startRecurse("Restarted service")
	return nil
}

// Clear clears any error condition in the service, without actually
// enabling it.  It will attempt to start the service if it isn't
// already running, and is enabled.
func (s *Service) Clear() {
	if s.mgr == nil {
		return
	}
	s.mgr.lock()
	defer s.mgr.unlock()

	if s.failed {
		s.reason = "Cleared fault"
		s.stamp = time.Now()
		s.logf("Clearing fault on %s", s.Name())
	}
	s.starts = 0
	s.failed = false
	s.err = nil
	s.startRecurse("Cleared fault")
}

// Check checks if a service is running, and performs any appropriate health
// checks.  It returns nil if the service is running and healthy, or false
// otherwise.  If it returns false, it will stop the service, as well as
// dependent services, and put the service into failed state.
func (s *Service) Check() error {
	if s.mgr == nil {
		return ErrNoManager
	}
	return s.checkService()
}

// matchServiceNames matches if the first (concrete) name matches
// the second.  This is true if either the variant of s1 is empty,
// or the two variants collide.
func serviceMatches(s1, s2 string) bool {
	a1 := strings.SplitN(s1, ":", 2)
	a2 := strings.SplitN(s2, ":", 2)

	if a1[0] != a2[0] {
		return false
	}
	if len(a1) == 1 {
		return true
	}
	if len(a2) == 1 {
		return false
	}
	return a1[1] == a2[1]
}

// Matches returns true if the service name matches the check.  This is
// true if either the check is a complete match, or if the first part of
// our name (or Provides) is identical to the check.  For example, if our
// name is "x:y", then this would return true for a check of "x", or "x:y",
// but not for "x:z", nor "z:y".
func (s *Service) Matches(check string) bool {
	if serviceMatches(check, s.Name()) {
		return true
	}
	for _, p := range s.Provides() {
		if serviceMatches(check, p) {
			return true
		}
	}
	return false
}

// SetProperty sets a property on the service.
func (s *Service) SetProperty(n PropertyName, v interface{}) error {
	if m := s.mgr; m != nil {
		m.lock()
		defer m.unlock()
	}
	if e := s.setProp(n, v); e != nil {
		s.logf("Failed to set property %s: %v", s.Name(), e)
		return e
	}
	return nil
}

func (s *Service) setProp(n PropertyName, v interface{}) error {
	// Lock it if we are already added.  Some properties cannot be
	// set once a the service is added.
	if m := s.mgr; m != nil {
		switch n {
		case PropName,
			PropDescription,
			PropConflicts,
			PropDepends,
			PropProvides:
			// These properties cannot be altered once they are
			// added to a service.
			return ErrPropReadOnly
		}
	}
	switch n {
	case PropLogger:
		if v, ok := v.(*log.Logger); ok {
			if s.enabled {
				// Cannot change logger while service enabled.
				return ErrPropReadOnly
			}
			if s.logger != nil {
				s.mlog.DelLogger(s.logger)
			}
			s.logger = v
			s.mlog.AddLogger(s.logger)
		} else {
			return ErrBadPropType
		}
	case PropRestart:
		if v, ok := v.(bool); ok {
			s.restart = v
		} else {
			return ErrBadPropType
		}
	case PropRateLimit:
		if v, ok := v.(int); ok {
			s.starts = 0
			if v > 0 {
				s.startTimes = make([]time.Time, v)
			} else {
				s.startTimes = nil
			}
			s.rateLimit = v
		} else {
			return ErrBadPropType
		}
	case PropRatePeriod:
		if v, ok := v.(time.Duration); ok {
			s.starts = 0
			s.ratePeriod = v
		} else {
			return ErrBadPropType
		}
	case PropName:
		if v, ok := v.(string); ok {
			s.name = v
		} else {
			return ErrBadPropType
		}
	case PropDescription:
		if v, ok := v.(string); ok {
			s.desc = v
		} else {
			return ErrBadPropType
		}
	case PropConflicts:
		if v, ok := v.([]string); ok {
			s.conflicts = append([]string{}, v...)
		} else {
			return ErrBadPropType
		}
	case PropDepends:
		if v, ok := v.([]string); ok {
			s.depends = append([]string{}, v...)
		} else {
			return ErrBadPropType
		}
	case PropProvides:
		if v, ok := v.([]string); ok {
			s.provides = append([]string{}, v...)
		} else {
			return ErrBadPropType
		}
	case PropNotify:
		if v, ok := v.(func()); ok {
			s.notify = v
			// We don't want to pass this one down, as we've
			// registered ourselves there.
			return nil
		} else {
			return ErrBadPropType
		}
	default:
		return s.prov.SetProperty(n, v)
	}

	// Pass the new property to the provider.  The provider doesn't get a
	// a chance to veto properties we've already dealt with though.
	s.prov.SetProperty(n, v)
	return nil
}

func (s *Service) GetProperty(n PropertyName) (interface{}, error) {
	if m := s.mgr; m != nil {
		m.lock()
		defer m.unlock()
	}

	switch n {
	case PropLogger:
		return s.logger, nil
	case PropRestart:
		return s.restart, nil
	case PropRateLimit:
		return s.rateLimit, nil
	case PropRatePeriod:
		return s.ratePeriod, nil
	case PropName:
		return s.name, nil
	case PropDescription:
		return s.desc, nil
	case PropConflicts:
		return append([]string{}, s.conflicts...), nil
	case PropDepends:
		return append([]string{}, s.depends...), nil
	case PropProvides:
		return append([]string{}, s.provides...), nil
	case PropNotify:
		return s.notify, nil
	}
	return s.prov.Property(n)
}

func (s *Service) GetLog() []string {
	if m := s.mgr; m != nil {
		m.lock()
		defer m.unlock()
	}
	recs := make([]string, 0, s.slog.NumRecords%len(s.slog.Records))
	cur := s.slog.NumRecords
	cnt := cur % len(s.slog.Records)
	if cnt > cur {
		cnt = cur
	}
	for i := cur - cnt; i < cnt; i++ {
		recs = append(recs, s.slog.Records[i%len(s.slog.Records)])

	}
	return recs
}

// setManager is called by the framework when the service is added to
// the manager.  This calculates the various dependency graphs, updating
// links to other services in the manager.
func (s *Service) setManager(mgr *Manager) {
	if s.mgr != nil {
		// This is a serious programmer mistake
		panic("Already added to a manager")
	}
	s.mlog.AddLogger(mgr.getLogger(s))
	s.mgr = mgr

	s.incompat = make(map[*Service]bool)
	s.children = make(map[*Service]bool)
	s.parents = make(map[string]map[*Service]bool)
	for _, d := range s.Depends() {
		s.parents[d] = make(map[*Service]bool)
	}
	for t := range mgr.services {

		// do we satisfy a dependency of t?
		for _, d := range t.Depends() {
			if s.Matches(d) {
				t.parents[d][s] = true
				s.children[t] = true
				break
			}
		}

		// does t satisfy a dependency of s?
		for _, d := range s.Depends() {
			if t.Matches(d) {
				s.parents[d][t] = true
				t.children[s] = true
				break
			}
		}

		// do we conflict with t?
		for _, c := range t.Conflicts() {
			if s.Matches(c) {
				s.incompat[t] = true
				t.incompat[s] = true
			}
		}
		for _, c := range s.Conflicts() {
			if t.Matches(c) {
				s.incompat[t] = true
				t.incompat[s] = true
			}
		}
	}
	s.stamp = time.Now()
	s.reason = "Added service"
	s.logf("Added service %s to %s: %s", s.Name(), mgr.Name(),
		s.Description())
	mgr.services[s] = true
}

func (s *Service) delManager() {
	if s.mgr == nil {
		return
	}

	// remove the item
	delete(s.mgr.services, s)

	// remove from each of our conflicts
	for c := range s.incompat {
		delete(c.incompat, s)
		delete(s.incompat, c)
	}

	// our children (things that may depend upon us)
	for c := range s.children {
		for p := range c.parents {
			delete(c.parents[p], s)
		}
		delete(s.children, c)
	}

	// our parents (this we depend upon)
	for d, p := range s.parents {
		for t := range p {
			delete(p, t)
			delete(t.children, s)
		}
		delete(s.parents, d)
	}

	s.reason = "Removed service"
	s.stamp = time.Now()
	s.mgr = nil
}

func (s *Service) logf(fmt string, v ...interface{}) {
	s.mlog.Logger().Printf(fmt, v...)
}

func (s *Service) startRecurse(detail string) {
	if s.running {
		return
	}
	if !s.canRun() {
		return
	}
	if e := s.tooQuickly(); e != nil {
		return
	}
	if s.rateLimit > 0 {
		s.startTimes[s.starts%s.rateLimit] = time.Now()
	}
	s.starts++
	if e := s.prov.Start(); e != nil {
		s.logf("Failed to start %s: %v", s.Name(), e)
		s.reason = "Failed to start:" + e.Error()
		s.stamp = time.Now()
		s.err = e
		s.failed = true
		return
	}
	s.reason = "Started: " + detail
	s.stamp = time.Now()
	s.logf("Started %s: %s", s.Name(), detail)
	s.running = true
	s.failed = false
	for child := range s.children {
		child.startRecurse("Dependency running")
	}
}

func (s *Service) stopRecurse(detail string) {
	if !s.running || s.stopping {
		return
	}
	s.stopping = true
	for child := range s.children {
		if child.canRun() {
			continue
		}
		child.stopRecurse("Dependency stopped")
	}
	s.prov.Stop()
	s.reason = "Stopped: " + detail
	s.stamp = time.Now()
	s.logf("Stopped %s: %s", s.Name(), detail)

	s.running = false
	s.stopping = false
}

func (s *Service) canRun() bool {
	if s.stopping || !s.enabled {
		return false
	}
	for _, deps := range s.parents {
		sat := false
		for d := range deps {
			if d.enabled && d.running && !d.stopping && !d.failed {
				sat = true
				break
			}
		}
		if !sat {
			return false
		}
	}

	for c := range s.incompat {
		if c.enabled {
			return false
		}
	}
	return true
}

func (s *Service) checkService() error {
	if s.failed {
		return s.err
	}
	if !s.running {
		return ErrNotRunning
	}
	s.checking = true
	if e := s.prov.Check(); e != nil {
		s.logf("Service %s faulted: %v", s.Name(), e)
		s.failed = true
		s.stopRecurse("Faulted: " + e.Error())
		s.err = e
		s.checking = false
		return e
	}
	s.checking = false
	return nil
}

// A service is restarting too quickly if it restarts more than a specified
// number of times in an interval.  Once we hit that threshold, we wait for
// a full interval count before we will restart.  Effectively, this means
// that if we hit the threshold, we actually won't restart for *another*
// interval, reducing our rate to 1/2 the configured rate, punishing us for
// bad behavior.
func (s *Service) tooQuickly() error {
	if s.rateLimit == 0 {
		return nil
	}
	if s.starts < s.rateLimit {
		return nil
	}

	// If we've restarted more than n times in the last period,
	// then rate limit us.
	idx := (s.starts - 1) % s.rateLimit
	end := s.startTimes[idx]
	if time.Now().Before(end.Add(s.ratePeriod)) {

		// Log it if not already done.
		if !s.rateLog {
			s.logf("Service %s restarting too quickly", s.Name())
		}
		// And we uncoditionally mark this to note cool down.
		s.rateLog = true
		return ErrRateLimited
	}

	// If we haven't restarted recently too quickly, we're done.
	if !s.rateLog {
		// Not in cool down mode.
		return nil
	}

	// Check to see if cool down from prior rate limit is expired.
	idx = (s.starts - 2) % s.rateLimit
	end = s.startTimes[idx]
	if time.Now().Before(end.Add(s.ratePeriod)) {
		return ErrRateLimited
	}

	// All cool downs expired.
	s.rateLog = false
	return nil
}

func (s *Service) selfHeal() {
	if s.failed && s.restart {
		s.logf("Attempting self-healing")
		s.startRecurse("Self-healing attempt")
	}
}

func (s *Service) doNotify() {
	go func() {
		var cb func()
		if m := s.mgr; m != nil {
			m.lock()
			m.notify(s)
			cb = s.notify
			m.unlock()
		} else {
			cb = s.notify
		}
		if cb != nil {
			go cb()
		}
	}()
}

// NewService allocates a service instance from a Provider.  The intention
// is that Providers use this in their own constructors to present only a
// Service interface to applications.
func NewService(p Provider) *Service {
	s := &Service{prov: p}
	s.ratePeriod = time.Minute
	s.rateLimit = 10
	s.startTimes = make([]time.Time, s.rateLimit)

	s.name = p.Name()
	s.desc = p.Description()
	s.conflicts = append([]string{}, p.Conflicts()...)
	s.depends = append([]string{}, p.Depends()...)
	s.provides = append([]string{}, p.Provides()...)
	s.mlog = NewMultiLogger()
	s.mlog.Logger().SetPrefix("[" + s.Name() + "] ")
	s.prov.SetProperty(PropLogger, s.mlog.Logger())
	s.slog = &ServiceLog{}
	s.mlog.AddLogger(log.New(s.slog, "", log.LstdFlags))
	p.SetProperty(PropNotify, s.doNotify)
	return s
}
