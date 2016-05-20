// Copyright 2016 The Govisor Authors
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

// +build darwin dragonfly freebsd linux netbsd openbsd solaris

// The test suite relies pretty heavily on a process_test.sh script that is
// bundled, but is pretty specific to POSIX systems.  Implementing a suitable
// test script for other systems is left as an exercise for the reader.

package govisor

import (
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestProcessStartStop(t *testing.T) {
	Convey("Test start/stop of a new process", t, func() {
		m := NewManager("TestProcessStartStop")
		SetTestLogger(t, m)
		s1 := NewProcess("ProcessStartStop:S1", &exec.Cmd{
			Path: "process_test.sh",
			Args: []string{"process_test.sh", "3600"},
		})
		So(s1, ShouldNotBeNil)

		m.AddService(s1)
		So(s1.Enabled(), ShouldBeFalse)
		So(s1.Running(), ShouldBeFalse)
		e := s1.Enable()
		So(e, ShouldBeNil)
		So(s1.Enabled(), ShouldBeTrue)
		So(s1.Running(), ShouldBeTrue)

		time.Sleep(time.Millisecond * 10)

		e = s1.Disable()
		So(e, ShouldBeNil)
		So(s1.Enabled(), ShouldBeFalse)
		So(s1.Running(), ShouldBeFalse)

		time.Sleep(time.Millisecond * 10)

		m.Shutdown()
	})
}

func TestProcessFail(t *testing.T) {
	Convey("Test a failing process", t, func() {
		m := NewManager("TestProcessFail")
		SetTestLogger(t, m)
		s1 := NewProcess("ProcessFail:S1", &exec.Cmd{
			Path: "process_test.sh",
			Args: []string{"process_test.sh", "fail"},
		})
		So(s1, ShouldNotBeNil)
		m.AddService(s1)
		m.StopMonitoring()
		e := s1.Enable()
		So(e, ShouldBeNil)
		So(s1.Enabled(), ShouldBeTrue)
		time.Sleep(time.Millisecond * 10)
		e = s1.Check()
		So(e, ShouldNotBeNil)
		So(s1.Enabled(), ShouldBeTrue)
		So(s1.Failed(), ShouldBeTrue)
		So(s1.Running(), ShouldBeFalse)
	})
}

func TestProcessFromManifest(t *testing.T) {
	Convey("Test process from a manifest", t, func() {
		mydir, _ := os.Getwd()
		exname := mydir + "/" + "process_test.sh"
		manifest := ProcessManifest{
			Name: "SampleProcessManifest",
			Description: "A sample description",
			Directory: "/usr",
			Command: []string{ exname, "checkwd", "/usr"},
			FailOnExit:	false,
			Provides: []string{"testmanifest"},
		}

		m := NewManager("TestProcessFromManifest")
		SetTestLogger(t, m)
		s1 := NewProcessFromManifest(manifest);
		So(s1, ShouldNotBeNil)

		m.AddService(s1)
		So(s1.Enabled(), ShouldBeFalse)
		So(s1.Running(), ShouldBeFalse)
		e := s1.Enable()
		So(e, ShouldBeNil)
		So(s1.Enabled(), ShouldBeTrue)

		time.Sleep(time.Millisecond * 100)
		So(s1.Failed(), ShouldBeFalse)

		e = s1.Disable()
		So(e, ShouldBeNil)
		So(s1.Enabled(), ShouldBeFalse)
		So(s1.Running(), ShouldBeFalse)

		time.Sleep(time.Millisecond * 10)

		m.Shutdown()
	})
}
