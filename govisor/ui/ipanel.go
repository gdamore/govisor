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

package ui

import (
	"fmt"

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"

	"github.com/gdamore/govisor/govisor/util"
	"github.com/gdamore/govisor/rest"
)

type InfoPanel struct {
	text *views.TextArea
	info *rest.ServiceInfo
	name string // service name
	err  error  // last error retrieving state

	Panel
}

func NewInfoPanel(app *App) *InfoPanel {
	ipanel := &InfoPanel{}
	ipanel.Panel.Init(app)

	ipanel.text = views.NewTextArea()
	ipanel.text.EnableCursor(false)
	ipanel.SetContent(ipanel.text)
	ipanel.text.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorSilver).Background(tcell.ColorBlack))

	// We don't change the keybar, so set it once
	ipanel.SetKeys([]string{"[Q] Quit", "[H] Help"})

	return ipanel
}

func (i *InfoPanel) Draw() {
	i.update()
	i.Panel.Draw()
}

func (i *InfoPanel) HandleEvent(ev tcell.Event) bool {
	info := i.info
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEsc:
			i.App().ShowMain()
			return true
		case tcell.KeyF1:
			i.App().ShowHelp()
			return true
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'Q', 'q':
				i.App().ShowMain()
				return true
			case 'H', 'h':
				i.App().ShowHelp()
				return true
			case 'L', 'l':
				if info != nil {
					i.App().ShowLog(info.Name)
					return true
				}
			case 'R', 'r':
				if info != nil {
					i.App().RestartService(info.Name)
					return true
				}
			case 'E', 'e':
				if info != nil && !info.Enabled {
					i.App().EnableService(info.Name)
					return true
				}
			case 'D', 'd':
				if info != nil && info.Enabled {
					i.App().DisableService(info.Name)
					return true
				}
			case 'C', 'c':
				if info != nil && info.Failed {
					i.App().ClearService(info.Name)
					return true
				}
			}
		}
	}
	return i.Panel.HandleEvent(ev)
}

func (i *InfoPanel) SetName(name string) {
	i.name = name
}

// update must be called with AppLock held.
func (i *InfoPanel) update() {

	s, e := i.App().GetItem(i.name)

	if i.info == s && i.err == e {
		return
	}
	i.info = s
	i.err = e
	words := []string{"[ESC] Main", "[H] Help"}

	i.SetTitle("Details for " + i.name)

	if s == nil {
		if i.err != nil {
			i.SetStatus(fmt.Sprintf("No data: %v", i.err))
			i.SetError()
		} else {
			i.SetStatus("Loading...")
			i.SetNormal()
		}
		i.text.SetLines(nil)
		i.SetKeys(words)
		return
	}

	i.SetStatus("")
	if !s.Enabled {
		i.SetNormal()
	} else if s.Failed {
		i.SetError()
	} else if s.Running {
		i.SetGood()
	} else {
		i.SetWarn()
	}

	lines := make([]string, 0, 8)
	lines = append(lines, fmt.Sprintf("%13s %s", "Name:", s.Name))
	lines = append(lines, fmt.Sprintf("%13s %s", "Description:",
		s.Description))
	lines = append(lines, fmt.Sprintf("%13s %s", "Status:", util.Status(s)))
	lines = append(lines, fmt.Sprintf("%13s %v", "Since:", s.TimeStamp))
	lines = append(lines, fmt.Sprintf("%13s %s", "Detail:", s.Status))

	l := fmt.Sprintf("%13s", "Provides:")
	for _, p := range s.Provides {
		l = l + fmt.Sprintf(" %s", p)
	}
	lines = append(lines, l)

	l = fmt.Sprintf("%13s", "Depends:")
	for _, p := range s.Depends {
		l = l + fmt.Sprintf(" %s", p)
	}
	lines = append(lines, l)

	l = fmt.Sprintf("%13s", "Conflicts:")
	for _, p := range s.Conflicts {
		l = l + fmt.Sprintf(" %s", p)
	}
	lines = append(lines, l)

	i.text.SetLines(lines)

	words = append(words, "[L] Log")
	if !s.Enabled {
		words = append(words, "[E] Enable")
	} else {
		words = append(words, "[D] Disable")
		if s.Failed {
			words = append(words, "[C] Clear")
		}
		words = append(words, "[R] Restart")
	}
	i.SetKeys(words)
}
