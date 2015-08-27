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
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	PropProcessFailOnExit PropertyName = "_ProcFailOnExit"
	PropProcessStopCmd                 = "_ProcStopCmd"
	PropProcessStopTime                = "_ProcStopTime"
	PropProcessCheckCmd                = "_ProcCheckCmd"
)

//
// Process represents an actual operating system level process.  This implements
// the Provider interface, and hence Process objects can used as such.
//
// XXX: is there any reason for this to be public?
// XXX: Should we support Setsid and other SysProcAttr settings?
//
type Process struct {
	name      string      // This is the Govisor name, must be set
	desc      string      // Description
	provides  []string    // Usually empty, but a service can offer more
	depends   []string    // Govisor services we depend upon
	conflicts []string    // Govisor services that conflict with us
	logger    *log.Logger // Log for messages, stdout, and stderr.
	reason    error       // Why we failed
	failed    bool        // True if we are in failure state
	stopped   bool        // True if we were stopped

	stopTime   time.Duration // Time to wait for clean shutdown, 0 = forever
	failOnExit bool          // If true, mark failed if the process exits.
	stopCmd    *exec.Cmd
	checkCmd   *exec.Cmd
	startCmd   exec.Cmd

	lock   sync.Mutex
	waiter sync.WaitGroup
}

func (p *Process) doLog(r io.ReadCloser, prefix string) {
	// Gather stdin/stdout in chunks of lines
	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if len(line) != 0 {
			p.logger.Print(prefix, strings.Trim(line, "\n"))
		}
		if err != nil {
			return
		}
	}
}

func (p *Process) Name() string {
	return p.name
}

func (p *Process) Description() string {
	return p.desc
}

func copyArray(src []string) []string {
	rv := make([]string, 0, len(src))
	rv = append(rv, src...)
	return rv
}

func (p *Process) Provides() []string {
	return copyArray(p.provides)
}

func (p *Process) Conflicts() []string {
	return copyArray(p.conflicts)
}

func (p *Process) Depends() []string {
	return copyArray(p.depends)
}

func (p *Process) doWait() {

	e := p.startCmd.Wait()
	p.lock.Lock()
	if !p.stopped {
		if e != nil {
			p.failed = true
			p.reason = e
			p.logger.Printf("Failed: %v", e)
		} else if p.failOnExit {
			e = errors.New("Unexpected termination")
			p.reason = e
			p.failed = true
			p.logger.Printf("Failed: %v", e)
		}
	}
	p.lock.Unlock()
	p.waiter.Done()
}

func (p *Process) Start() error {

	p.lock.Lock()
	defer p.lock.Unlock()

	p.stopped = false
	p.failed = false
	p.reason = nil

	if p.startCmd.Stdout == nil {
		stdout, e := p.startCmd.StdoutPipe()
		if e != nil {
			p.logger.Printf("Failed to capture stdout: %v", e)
		} else {
			go p.doLog(stdout, "stdout> ")
		}
	}
	if p.startCmd.Stderr == nil {
		stderr, e := p.startCmd.StderrPipe()
		if e != nil {
			p.logger.Printf("Failed to capture stderr: %v", e)
		} else {
			go p.doLog(stderr, "stderr> ")
		}
	}

	if e := p.startCmd.Start(); e != nil {
		p.failed = true
		p.reason = e
		return e
	}
	p.waiter.Add(1)

	go p.doWait()

	return nil
}

func (p *Process) runCmdWithTimeout(pfx string, c *exec.Cmd, d time.Duration) error {
	newc := &exec.Cmd{}
	*newc = *c
	if proc := p.startCmd.Process; proc != nil {
		if c.Env == nil {
			newc.Env = os.Environ()
		}
		newc.Env = append(make([]string, 0, len(newc.Env)+1), newc.Env...)
		newc.Env = append(newc.Env, fmt.Sprintf("PID=%d", proc.Pid))
	}

	newc.Process = nil
	newc.ProcessState = nil

	if d == 0 {
		d = time.Second * 10
	}
	if stderr, e := newc.StderrPipe(); e != nil {
		p.logger.Printf("Failed to capture stderr: %v", e)
	} else {
		go p.doLog(stderr, pfx+"stderr> ")
	}
	if stdout, e := newc.StdoutPipe(); e != nil {
		p.logger.Printf("Failed to capture stdout: %v", e)
	} else {
		go p.doLog(stdout, pfx+"stdout> ")
	}

	if e := newc.Start(); e != nil {
		return e
	}
	proc := newc.Process
	timer := time.AfterFunc(d, func() {
		p.logger.Printf("Timeout waiting for %s command", pfx)
		proc.Kill()
	})
	e := newc.Wait()
	timer.Stop()
	return e
}

func (p *Process) shutdown() {
	if proc := p.startCmd.Process; proc != nil && proc.Pid != -1 &&
		p.startCmd.ProcessState == nil {
		if p.stopCmd == nil {
			e := proc.Signal(syscall.SIGTERM)
			if e != nil {
				p.logger.Printf("Failed sending SIGTERM: %v", e)
			}
		} else {
			// Put the Pid into the environment as $PID
			e := p.runCmdWithTimeout("stop", p.stopCmd, p.stopTime)
			if e != nil {
				p.logger.Printf("Failed stop cmd: %v", e)
			}
		}
	}
}

func (p *Process) kill() {
	if proc := p.startCmd.Process; proc != nil {
		e := proc.Kill()
		if e != nil {
			p.logger.Printf("Failed killing: %v", e)
		}
	}
}

func (p *Process) Stop() {

	p.lock.Lock()
	p.stopped = true
	if proc := p.startCmd.Process; proc != nil {
		var timer *time.Timer
		p.shutdown()
		if p.stopTime > 0 {
			timer = time.AfterFunc(p.stopTime, func() {
				p.logger.Printf("Graceful shutdown timed out")
				p.lock.Lock()
				p.kill()
				p.lock.Unlock()
			})
		}
		p.lock.Unlock()
		p.waiter.Wait()
		p.lock.Lock()
		if timer != nil {
			timer.Stop()
		}
	}
	p.startCmd.Process = nil
	p.lock.Unlock()
}

func (p *Process) Check() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.failed {
		return p.reason
	}
	return nil
}

func (p *Process) SetProperty(n PropertyName, v interface{}) error {
	switch n {
	case PropLogger:
		if v, ok := v.(*log.Logger); ok {
			p.logger = v
			return nil
		}
		return ErrBadPropType
	case PropProcessFailOnExit:
		if v, ok := v.(bool); ok {
			p.failOnExit = v
			return nil
		}
		return ErrBadPropType
	case PropProcessStopTime:
		if v, ok := v.(time.Duration); ok {
			p.stopTime = v
			return nil
		}
		return ErrBadPropType
	case PropProcessStopCmd:
		if v, ok := v.(*exec.Cmd); ok {
			p.stopCmd = new(exec.Cmd)
			*p.stopCmd = *v
			return nil
		}
		return ErrBadPropType
	}
	return ErrBadPropName
}

func (p *Process) Property(n PropertyName) (interface{}, error) {
	switch n {
	case PropLogger:
		return p.logger, nil
	case PropProcessFailOnExit:
		return p.failOnExit, nil
	case PropProcessStopTime:
		return p.stopTime, nil
	case PropProcessStopCmd:
		return p.stopCmd, nil
	}
	return nil, ErrBadPropName
}

type ProcessManifest struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Command     []string      `json:"command"`
	Env         []string      `json:"env"`
	StopCmd     []string      `json:"stopCommand"`
	StopTime    time.Duration `json:"stopTime"`
	FailOnExit  bool          `json:"failOnExit"`
	CheckCmd    []string      `json:"check"`
	Restart     bool          `json:"restart"`
	Provides    []string      `json:"provides"`
	Depends     []string      `json:"depends"`
	Conflicts   []string      `json:"conflicts"`
}

func NewProcessFromManifest(m ProcessManifest) *Service {
	p := &Process{}
	p.name = m.Name
	p.desc = m.Description
	if len(m.Command) != 0 {
		p.startCmd.Path = m.Command[0]
		p.startCmd.Args = m.Command
	}
	if len(m.StopCmd) != 0 {
		p.stopCmd = exec.Command(m.StopCmd[0], m.StopCmd[1:]...)
	}
	if len(m.CheckCmd) != 0 {
		p.checkCmd = exec.Command(m.CheckCmd[0], m.CheckCmd[1:]...)
	}
	p.stopTime = m.StopTime
	p.depends = m.Depends
	p.conflicts = m.Conflicts
	p.provides = m.Provides
	p.failOnExit = m.FailOnExit

	s := NewService(p)
	s.SetProperty(PropRestart, m.Restart)
	return s
}

func NewProcessFromJson(r io.Reader) (*Service, error) {
	dec := json.NewDecoder(r)
	var m ProcessManifest
	if e := dec.Decode(&m); e != nil {
		return nil, e
	}
	return NewProcessFromManifest(m), nil
}

func NewProcess(name string, cmd *exec.Cmd) *Service {
	p := &Process{}
	p.logger = log.New(os.Stderr, "", log.LstdFlags)
	p.stopTime = time.Second * 10
	p.startCmd = *cmd
	p.name = name
	p.desc = name + " process: " + cmd.Path
	return NewService(p)
}
