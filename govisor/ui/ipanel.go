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

package ui

import (
	"fmt"

	"github.com/gdamore/topsl"

	"github.com/gdamore/govisor/govisor/util"
	"github.com/gdamore/govisor/rest"
)

type InfoPanel struct {
	text      *topsl.TextArea
	info      *rest.ServiceInfo
	name      string // service name
	err       error  // last error retrieving state
	statusbar *topsl.StatusBar
	keybar    *topsl.KeyBar
	titlebar  *topsl.TitleBar
	panel     *topsl.Panel
	app       *App
}

func NewInfoPanel(app *App) *InfoPanel {
	ipanel := &InfoPanel{
		text:      topsl.NewTextArea(),
		info:      nil,
		titlebar:  topsl.NewTitleBar(),
		keybar:    topsl.NewKeyBar(),
		statusbar: topsl.NewStatusBar(),
		panel:     topsl.NewPanel(),
		app:       app,
	}

	ipanel.panel.SetBottom(ipanel.keybar)
	ipanel.panel.SetTitle(ipanel.titlebar)
	ipanel.panel.SetStatus(ipanel.statusbar)
	ipanel.panel.SetContent(ipanel.text)

	ipanel.titlebar.SetRight(app.GetAppName())

	// We don't change the keybar, so set it once
	ipanel.keybar.SetKeys([]string{"[Q] Quit", "[H] Help"})

	// Cursor disabled
	ipanel.text.EnableCursor(false)

	return ipanel
}

func (i *InfoPanel) Draw() {
	i.update()
	i.panel.Draw()
}

func (i *InfoPanel) HandleEvent(ev topsl.Event) bool {
	info := i.info
	switch ev := ev.(type) {
	case *topsl.KeyEvent:
		switch ev.Ch {
		case 0:
			switch ev.Key {
			case topsl.KeyEsc:
				i.app.ShowMain()
				return true
			case topsl.KeyF1:
				i.app.ShowHelp()
				return true
			}
		case 'Q', 'q':
			i.app.ShowMain()
			return true
		case 'H', 'h':
			i.app.ShowHelp()
			return true
		case 'L', 'l':
			if info != nil {
				i.app.ShowLog(info.Name)
				return true
			}
		case 'R', 'r':
			if info != nil {
				i.app.RestartService(info.Name)
				return true
			}
		case 'E', 'e':
			if info != nil && !info.Enabled {
				i.app.EnableService(info.Name)
				return true
			}
		case 'D', 'd':
			if info != nil && info.Enabled {
				i.app.DisableService(info.Name)
				return true
			}
		case 'C', 'c':
			if info != nil && info.Failed {
				i.app.ClearService(info.Name)
				return true
			}
		}
	}
	return i.panel.HandleEvent(ev)
}

func (i *InfoPanel) SetView(view topsl.View) {
	i.panel.SetView(view)
}

func (i *InfoPanel) Resize() {
	i.panel.Resize()
}

func (i *InfoPanel) SetName(name string) {
	i.name = name
}

// update must be called with AppLock held.
func (i *InfoPanel) update() {

	s, e := i.app.GetItem(i.name)

	if i.info == s && i.err == e {
		return
	}
	i.info = s
	i.err = e
	words := []string{"[ESC] Main", "[H] Help"}

	i.titlebar.SetCenter("Details for " + i.name)

	if s == nil {
		if i.err != nil {
			i.statusbar.SetStatus(fmt.Sprintf(
				"No data: %v", i.err))
			i.statusbar.SetFail()
		} else {
			i.statusbar.SetStatus("Loading...")
			i.statusbar.SetNormal()
		}
		i.text.SetLines(nil)
		i.keybar.SetKeys(words)
		return
	}

	i.statusbar.SetStatus("")
	if !s.Enabled {
		i.statusbar.SetNormal()
	} else if s.Failed {
		i.statusbar.SetFail()
	} else if s.Running {
		i.statusbar.SetGood()
	} else {
		i.statusbar.SetWarn()
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
	i.keybar.SetKeys(words)
}
