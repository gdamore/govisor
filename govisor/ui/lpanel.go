// Copyright 2024 The Govisor Authors
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
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"

	"github.com/gdamore/govisor/rest"
)

type LogPanel struct {
	text *views.TextArea
	info *rest.ServiceInfo
	name string // service name
	err  error  // last error retrieving state

	Panel
}

func NewLogPanel(app *App) *LogPanel {
	p := &LogPanel{}

	p.Panel.Init(app)

	// We don't change the keybar, so set it once
	p.SetKeys([]string{"[Q] Quit", "[H] Help"})

	p.text = views.NewTextArea()
	p.text.EnableCursor(false)
	p.text.SetStyle(tcell.StyleDefault.
		Foreground(tcell.ColorSilver).Background(tcell.ColorBlack))
	p.SetContent(p.text)
	p.update()

	return p
}

func (p *LogPanel) Draw() {
	p.update()
	p.Panel.Draw()
}

func (p *LogPanel) HandleEvent(ev tcell.Event) bool {
	info := p.info
	app := p.app
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEsc:
			app.ShowMain()
			return true
		case tcell.KeyF1:
			app.ShowHelp()
			return true
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'Q', 'q':
				app.ShowMain()
				return true
			case 'H', 'h':
				app.ShowHelp()
				return true
			case 'I', 'i':
				if info != nil {
					app.ShowInfo(info.Name)
					return true
				}
			case 'R', 'r':
				if info != nil {
					app.RestartService(info.Name)
					return true
				}
			case 'E', 'e':
				if info != nil && !info.Enabled {
					app.EnableService(info.Name)
					return true
				}
			case 'D', 'd':
				if info != nil && info.Enabled {
					app.DisableService(info.Name)
					return true
				}
			case 'C', 'c':
				if info != nil && info.Failed {
					app.ClearService(info.Name)
					return true
				}
			}
		}
	}
	return p.Panel.HandleEvent(ev)
}

func (p *LogPanel) SetName(name string) {
	p.SetTitle("Loading")
	p.text.SetLines(nil)
	p.name = name
}

// update must be called with AppLock held.
func (p *LogPanel) update() {

	svcinfo, e1 := p.app.GetItem(p.name)
	loginfo, e2 := p.app.GetLog(p.name)
	p.info = svcinfo

	words := []string{"[ESC] Main", "[H] Help"}

	if p.name == "" {
		p.SetTitle("Consolidated Log")
	} else {
		p.SetTitle("Log for " + p.name)
	}

	if (svcinfo == nil && p.name != "") || loginfo == nil {
		e := e2
		if e == nil {
			e = e1
		}
		if p.err != nil {
			p.SetStatus(fmt.Sprintf("No data: %v", e))
			p.SetError()
		} else {
			p.SetStatus("Loading ...")
			p.SetNormal()
		}
		p.text.SetLines([]string{""})
		p.SetKeys(words)
		return
	}

	p.SetStatus("")
	if svcinfo != nil {
		if !svcinfo.Enabled {
			p.SetNormal()
		} else if svcinfo.Failed {
			p.SetError()
		} else if svcinfo.Running {
			p.SetGood()
		} else {
			p.SetWarn()
		}
	}

	lines := make([]string, 0, len(loginfo.Records))
	for _, r := range loginfo.Records {
		line := fmt.Sprintf("%s %s",
			r.Time.Format(time.StampMilli), r.Text)
		lines = append(lines, line)
	}
	p.text.SetLines(lines)

	if svcinfo != nil {
		words = append(words, "[I] Info")
		if !svcinfo.Enabled {
			words = append(words, "[E] Enable")
		} else {
			words = append(words, "[D] Disable")
			if svcinfo.Failed {
				words = append(words, "[C] Clear")
			}
			words = append(words, "[R] Restart")
		}
	}
	p.SetKeys(words)
}
