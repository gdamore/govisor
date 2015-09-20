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
	"time"

	"github.com/gdamore/topsl"

	"github.com/gdamore/govisor/rest"
)

type LogPanel struct {
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

func NewLogPanel(app *App) *LogPanel {
	p := &LogPanel{
		text:      topsl.NewTextArea(),
		info:      nil,
		titlebar:  topsl.NewTitleBar(),
		keybar:    topsl.NewKeyBar(),
		statusbar: topsl.NewStatusBar(),
		panel:     topsl.NewPanel(),
		app:       app,
	}

	p.panel.SetBottom(p.keybar)
	p.panel.SetTitle(p.titlebar)
	p.panel.SetStatus(p.statusbar)
	p.panel.SetContent(p.text)

	p.titlebar.SetRight(app.GetAppName())

	// We don't change the keybar, so set it once
	p.keybar.SetKeys([]string{"[Q] Quit", "[H] Help"})

	// Cursor disabled
	p.text.EnableCursor(false)

	return p
}

func (p *LogPanel) Draw() {
	p.update()
	p.panel.Draw()
}

func (p *LogPanel) HandleEvent(ev topsl.Event) bool {
	info := p.info
	app := p.app
	switch ev := ev.(type) {
	case *topsl.KeyEvent:
		switch ev.Ch {
		case 0:
			switch ev.Key {
			case topsl.KeyEsc:
				app.ShowMain()
				return true
			case topsl.KeyF1:
				app.ShowHelp()
				return true
			}
		case 'Q', 'q':
			app.ShowMain()
			return true
		case 'H', 'h':
			app.ShowHelp()
			return true
		case 'I', 'i':
			app.ShowInfo(info.Name)
			return true
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
	return p.panel.HandleEvent(ev)
}

func (p *LogPanel) SetView(view topsl.View) {
	p.panel.SetView(view)
}

func (p *LogPanel) Resize() {
	p.panel.Resize()
}

func (p *LogPanel) SetName(name string) {
	p.titlebar.SetCenter("Loading")
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
		p.titlebar.SetCenter("Consoliated Log")
	} else {
		p.titlebar.SetCenter("Log for " + p.name)
	}

	if svcinfo == nil || loginfo == nil {
		e := e2
		if e == nil {
			e = e1
		}
		p.titlebar.SetCenter("")
		if p.err != nil {
			p.statusbar.SetStatus(fmt.Sprintf("No data: %v", e))
			p.statusbar.SetFail()
		} else {
			p.statusbar.SetStatus("Loading ...")
			p.statusbar.SetNormal()
		}
		p.text.SetLines(nil)
		p.keybar.SetKeys(words)
		return
	}

	p.statusbar.SetStatus("")
	if !svcinfo.Enabled {
		p.statusbar.SetNormal()
	} else if svcinfo.Failed {
		p.statusbar.SetFail()
	} else if svcinfo.Running {
		p.statusbar.SetGood()
	} else {
		p.statusbar.SetWarn()
	}

	lines := make([]string, 0, len(loginfo.Records))
	for _, r := range loginfo.Records {
		line := fmt.Sprintf("%s %s",
			r.Time.Format(time.StampMilli), r.Text)
		lines = append(lines, line)
	}
	p.text.SetLines(lines)

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
	p.keybar.SetKeys(words)
}
