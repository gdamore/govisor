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
	"errors"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

type testLog struct {
	t *testing.T
}

func (tl *testLog) Write(p []byte) (n int, err error) {
	s := string(p)
	s = strings.Trim(s, "\n")
	tl.t.Log(s)
	return len(p), nil
}

type testS struct {
	name      string
	failed    bool
	started   bool
	provides  []string
	depends   []string
	conflicts []string
	logger    *log.Logger
	notify    func()
	sync.Mutex
}

func (s *testS) Name() string {
	return s.name
}

func (s *testS) Description() string {
	return "Test Service"
}

func (s *testS) Start() error {
	s.Lock()
	defer s.Unlock()
	if s.failed {
		return errors.New("Injected failure")
	}
	s.started = true
	return nil
}

func (s *testS) Stop() {
	s.Lock()
	s.started = false
	s.Unlock()
}

func (s *testS) Check() error {
	s.Lock()
	defer s.Unlock()
	if s.failed {
		return errors.New("Test service failure")
	}
	return nil
}

func (s *testS) Provides() []string {
	return s.provides
}

func (s *testS) Depends() []string {
	return s.depends
}

func (s *testS) Conflicts() []string {
	return s.conflicts
}

func (s *testS) SetProperty(n PropertyName, v interface{}) error {
	switch n {
	case PropLogger:
		if v, ok := v.(*log.Logger); ok {
			s.logger = v
			return nil
		}
		return ErrBadPropType
	case PropNotify:
		if v, ok := v.(func()); ok {
			s.notify = v
			return nil
		}
		return ErrBadPropType
	default:
		return ErrBadPropName
	}
}

func (s *testS) Property(n PropertyName) (interface{}, error) {
	switch n {
	case PropLogger:
		return s.logger, nil
	default:
		return nil, ErrBadPropName
	}
}

func (s *testS) inject() {
	s.Lock()
	s.logger.Printf("Injecting failure on %s", s.name)
	s.failed = true
	if s.notify != nil {
		s.logger.Printf("Sending fail notify")
		s.notify()
	}
	s.Unlock()
}

func (s *testS) clear() {
	s.Lock()
	s.logger.Printf("Clearing failure on %s", s.name)
	s.failed = false
	if s.notify != nil {
		s.logger.Printf("Sending clear notify")
		s.notify()
	}
	s.Unlock()
}

var testS1 = &testS{
	name:      "test:S1",
	provides:  []string{"alias:S1", "dep:S2"},
	conflicts: []string{"conflict:S1"},
	depends:   []string{},
}

var testS2 = &testS{
	name:      "test:S2",
	provides:  []string{},
	conflicts: []string{},
	depends:   []string{"dep:S2"},
}

func WithManager(t *testing.T, name string, fn func(m *Manager)) func() {
	return func() {
		m := NewManager(name)
		So(m, ShouldNotBeNil)
		m.SetLogWriter(&testLog{t: t})
		Reset(func() {
			m.Shutdown()
		})
		fn(m)
	}
}

func TestBadPropertyName(t *testing.T) {
	Convey("Bogus property name", t,
		WithManager(t, "BadPropName", func(m *Manager) {
			s1 := NewService(&testS{name: "test:BadName"})
			So(s1, ShouldNotBeNil)
			m.AddService(s1)
			e := s1.SetProperty(PropertyName("Nosuch"), true)
			So(e, ShouldNotBeNil)
		}))
}

func TestBadPropertyType(t *testing.T) {
	Convey("Bad property type", t,
		WithManager(t, "BadPropType", func(m *Manager) {
			s1 := NewService(&testS{name: "test:BadType"})
			So(s1, ShouldNotBeNil)
			m.AddService(s1)
			e := s1.SetProperty(PropName, 42)
			So(e, ShouldNotBeNil)
		}))
}

func TestSetPropOK(t *testing.T) {
	Convey("Set Properties", t,
		WithManager(t, "SetProp", func(m *Manager) {
			s1 := NewService(&testS{name: "test:Name"})
			So(s1, ShouldNotBeNil)
			e := s1.SetProperty(PropName, "test:NewName")
			So(e, ShouldBeNil)
			n, e := s1.GetProperty(PropName)
			So(e, ShouldBeNil)
			ns, ok := n.(string)
			So(ok, ShouldBeTrue)
			So(ns, ShouldEqual, "test:NewName")

			e = s1.SetProperty(PropDepends, []string{"s1:dep"})
			So(e, ShouldBeNil)

			e = s1.SetProperty(PropConflicts, []string{"conf"})
			So(e, ShouldBeNil)

			e = s1.SetProperty(PropProvides, []string{"abc:123"})
			So(e, ShouldBeNil)
		}))
}

func TestReadOnlyProps(t *testing.T) {
	Convey("Read only properties", t,
		WithManager(t, "ReadOnly", func(m *Manager) {
			s1 := NewService(&testS{name: "test:ro"})
			m.AddService(s1)
			e := s1.SetProperty(PropName, "test:shouldfail")
			So(e, ShouldNotBeNil)
		}))
}

func TestDependencies(t *testing.T) {
	Convey("Dependencies", t,
		WithManager(t, "Deps", func(m *Manager) {
			s1 := NewService(&testS{name: "test:s1"})
			So(s1, ShouldNotBeNil)
			s2 := NewService(&testS{name: "test:s2"})
			So(s2, ShouldNotBeNil)
			e := s2.SetProperty(PropDepends, []string{"test:s1"})
			So(e, ShouldBeNil)

			Convey("Both start disabled", func() {
				So(s1.Enabled(), ShouldBeFalse)
				So(s2.Enabled(), ShouldBeFalse)
			})

			Convey("Enabling S2 works", func() {
				m.AddService(s1)
				m.AddService(s2)
				So(s1.Enabled(), ShouldBeFalse)
				So(s2.Enabled(), ShouldBeFalse)
				e = s2.Enable()
				So(e, ShouldBeNil)

				Convey("But S2 isn't running yet", func() {
					So(s2.Running(), ShouldBeFalse)
				})

				Convey("Enabling S1 starts S2", func() {
					e = s1.Enable()
					So(e, ShouldBeNil)
					So(s1.Enabled(), ShouldBeTrue)
					So(s2.Enabled(), ShouldBeTrue)
					So(s1.Running(), ShouldBeTrue)
					So(s2.Running(), ShouldBeTrue)

					Convey("Disabling S1 stops both", func() {
						e = s1.Disable()
						So(e, ShouldBeNil)
						So(s1.Enabled(), ShouldBeFalse)
						So(s2.Enabled(), ShouldBeTrue)
						So(s1.Running(), ShouldBeFalse)
						So(s2.Running(), ShouldBeFalse)
					})
				})
			})
		}))
}

// XXX: This test function needs to be refactored
func TestGovisor(t *testing.T) {
	t1 := *testS1
	t2 := *testS2

	Convey("Given a new govisor", t, func() {
		m := NewManager("TestGoVisor")
		So(m, ShouldNotBeNil)
		m.SetLogWriter(&testLog{t: t})
		Convey("And new services S1 and S2", func() {
			s1 := NewService(&t1)
			So(s1, ShouldNotBeNil)
			m.AddService(s1)
			So(s1.Enabled(), ShouldBeFalse)
			So(s1.Running(), ShouldBeFalse)
			So(s1.Failed(), ShouldBeFalse)

			s2 := NewService(&t2)
			So(s2, ShouldNotBeNil)
			m.AddService(s2)
			So(s2.Enabled(), ShouldBeFalse)
			So(s2.Running(), ShouldBeFalse)
			So(s2.Failed(), ShouldBeFalse)

			Convey("We can enable S2 (depends on S1)", func() {
				e := s2.Enable()
				So(e, ShouldBeNil)
				So(s2.Enabled(), ShouldBeTrue)
				Convey("But it isn't running yet", func() {
					So(s2.Running(), ShouldBeFalse)
				})
				Convey("We can enable S1", func() {
					e = s1.Enable()
					So(e, ShouldBeNil)
					So(s1.Enabled(), ShouldBeTrue)
					So(s1.Running(), ShouldBeTrue)
					So(s2.Running(), ShouldBeTrue)
					Convey("We can restart all of them", func() {
						e := s2.Restart()
						So(e, ShouldBeNil)
						So(s1.Failed(), ShouldBeFalse)
						So(s2.Failed(), ShouldBeFalse)
						So(s1.Running(), ShouldBeTrue)
						So(s2.Running(), ShouldBeTrue)
					})
					Convey("Failure injection", func() {
						m.StopMonitoring()
						t1.inject()
						e := s1.Check()
						So(e, ShouldNotBeNil)
						So(s1.Failed(), ShouldBeTrue)
						So(s1.Running(), ShouldBeFalse)
						So(s2.Running(), ShouldBeFalse)
						t1.clear()
						s1.Clear()
						m.StartMonitoring()

						t1.inject()
						// wait for callbacks
						time.Sleep(time.Millisecond)
						So(s1.Failed(), ShouldBeTrue)
						So(s1.Running(), ShouldBeFalse)
						So(s2.Running(), ShouldBeFalse)
						t1.clear()
						s1.Clear()

						t.Logf("Test without healing")
						t1.inject()
						time.Sleep(time.Millisecond)
						So(s1.Failed(), ShouldBeTrue)
						t1.clear()
						time.Sleep(time.Millisecond)
						So(s1.Failed(), ShouldBeTrue)
						So(s1.Running(), ShouldBeFalse)
						s1.Clear()
						So(s1.Failed(), ShouldBeFalse)
						So(s1.Running(), ShouldBeTrue)

						t.Logf("Check self healing")
						e = s1.SetProperty(PropRestart, true)
						So(e, ShouldBeNil)
						t1.inject()
						time.Sleep(time.Millisecond)
						So(s1.Failed(), ShouldBeTrue)
						So(s1.Running(), ShouldBeFalse)
						t1.clear()
						time.Sleep(time.Millisecond)
						So(s1.Failed(), ShouldBeFalse)
						So(s1.Running(), ShouldBeTrue)
					})
				})
			})
		})
	})
}
