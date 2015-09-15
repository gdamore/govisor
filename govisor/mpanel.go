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

package main

import (
	"fmt"
	"time"

	"github.com/gdamore/govisor/rest"
	"github.com/gdamore/topsl"
)

// MainPanel implements a topsl.Widget as a topsl.Panel, but provides the data
// model and handling for the content area, using data loaded from a Govisor
// REST API service.
type MainPanel struct {
	info      *rest.ServiceInfo
	name      string // service name
	err       error  // last error retrieving state
	statusbar *topsl.StatusBar
	keybar    *topsl.KeyBar
	titlebar  *topsl.TitleBar
	content   *topsl.CellView
	panel     *topsl.Panel
	app       *App
	selected  *rest.ServiceInfo
	ndisabled int
	nfailed   int
	nrunning  int
	nstopped  int
	width     int
	height    int
	curx      int
	cury      int
	lines     []string
	styles    []topsl.Style
	items     []*rest.ServiceInfo
}

// mainModel provides the model for a CellArea.
type mainModel struct {
	m *MainPanel
}

func NewMainPanel(app *App, server string) *MainPanel {
	m := &MainPanel{
		content:   topsl.NewCellView(),
		info:      nil,
		titlebar:  topsl.NewTitleBar(),
		keybar:    topsl.NewKeyBar(),
		statusbar: topsl.NewStatusBar(),
		panel:     topsl.NewPanel(),
		app:       app,
	}

	m.panel.SetBottom(m.keybar)
	m.panel.SetTitle(m.titlebar)
	m.panel.SetStatus(m.statusbar)
	m.panel.SetContent(m.content)

	m.content.SetModel(&mainModel{m})
	m.titlebar.SetRight(app.GetAppName())
	m.titlebar.SetCenter(server)

	// We don't change the keybar, so set it once
	m.keybar.SetKeys([]string{"_Quit"})

	return m
}

func (m *MainPanel) Draw() {
	m.update()
	m.panel.Draw()
}

func (m *MainPanel) HandleEvent(ev topsl.Event) bool {
	switch ev := ev.(type) {
	case *topsl.KeyEvent:
		switch ev.Ch {
		case 0:
			switch ev.Key {
			case topsl.KeyEsc:
				m.unselect()
				return true
			case topsl.KeyF1:
				m.app.ShowHelp()
				return true
			case topsl.KeyEnter:
				if m.selected != nil {
					m.app.ShowInfo(m.selected.Name)
					return true
				}
			}
		case 'Q', 'q':
			m.app.Quit()
			return true
		case 'H', 'h':
			m.app.ShowHelp()
			return true
		case 'I', 'i':
			if m.selected != nil {
				m.app.ShowInfo(m.selected.Name)
				return true
			}
		case 'L', 'l':
			if m.selected != nil {
				m.app.ShowLog(m.selected.Name)
				return true
			}
		case 'E', 'e':
			if m.selected != nil && !m.selected.Enabled {
				m.app.EnableService(m.selected.Name)
				return true
			}
		case 'D', 'd':
			if m.selected != nil && m.selected.Enabled {
				m.app.DisableService(m.selected.Name)
				return true
			}
		case 'C', 'c':
			if m.selected != nil && m.selected.Failed {
				m.app.ClearService(m.selected.Name)
				return true
			}
		case 'R', 'r':
			if m.selected != nil {
				m.app.RestartService(m.selected.Name)
				return true
			}
		}
	}
	return m.panel.HandleEvent(ev)
}

func (m *MainPanel) SetView(view topsl.View) {
	m.panel.SetView(view)
}

func (m *MainPanel) Resize() {
	m.panel.Resize()
}

// Model items

func (model *mainModel) GetCell(x, y int) (rune, topsl.Style) {
	var ch rune
	var style topsl.Style

	m := model.m

	if y < 0 || y >= len(m.lines) {
		return ch, style
	}

	if x >= 0 && x < len(m.lines[y]) {
		ch = rune(m.lines[y][x])
	} else {
		ch = ' '
	}
	style = m.styles[y]
	if m.items[y] == m.selected {
		style = style.Reverse()
	}
	return ch, style
}

func (model *mainModel) GetBounds() (int, int) {
	// This assumes that all content is displayable runes of width 1.
	m := model.m
	y := len(m.lines)
	x := 0
	for _, l := range m.lines {
		if x < len(l) {
			x = len(l)
		}
	}
	return x, y
}

func (model *mainModel) GetCursor() (int, int, bool, bool) {
	m := model.m
	return m.curx, m.cury, true, false
}

func (model *mainModel) MoveCursor(offx, offy int) {

	m := model.m
	m.curx += offx
	m.cury += offy
	m.updateCursor(true)
}

func (model *mainModel) SetCursor(x, y int) {
	m := model.m
	m.curx = x
	m.cury = y
	m.updateCursor(true)
}

func (m *MainPanel) unselect() {
	m.cury = 0
	m.curx = 0
	m.updateCursor(false)
}

func (m *MainPanel) updateCursor(selected bool) {
	if m.curx > m.width-1 {
		m.curx = m.width - 1
	}
	if m.cury > m.height-1 {
		m.cury = m.height - 1
	}
	if m.curx < 0 {
		m.curx = 0
	}
	if m.cury < 0 {
		m.cury = 0
	}
	if selected && m.height > 0 {
		if m.selected == nil {
			m.curx = 0
			m.cury = 0
		}
		m.selected = m.items[m.cury]
	} else {
		m.selected = nil
	}
}

// update is called to update content, e.g. in response to Draw() or
// as part of another update.  It is called with the AppLock held.
func (m *MainPanel) update() {

	items, err := m.app.GetItems()
	m.items = items

	// preserve selected item
	if sel := m.selected; sel != nil {
		m.selected = nil
		for _, item := range m.items {
			if item.Name == sel.Name {
				m.selected = item
			}
		}
	}
	if err != nil {
		m.statusbar.SetFail()
		m.statusbar.SetStatus(fmt.Sprintf("Cannot load items: %v", err))
		m.lines = []string{}
		m.styles = []topsl.Style{}
		return
	}

	lines := make([]string, 0, len(m.items))
	styles := make([]topsl.Style, 0, len(m.items))

	m.ndisabled = 0
	m.nfailed = 0
	m.nstopped = 0
	m.nrunning = 0

	m.height = 0
	m.width = 0

	for _, info := range items {
		d := time.Since(info.TimeStamp)
		d -= d % time.Second
		line := fmt.Sprintf("%-20s %-10s %10s   %10s",
			info.Name, status(info), formatDuration(d), info.Status)

		if len(line) > m.width {
			m.width = len(line)
		}
		m.height++

		lines = append(lines, line)
		var style topsl.Style
		if !info.Enabled {
			style = topsl.StyleText
			m.ndisabled++
		} else if info.Failed {
			style = topsl.StyleError
			m.nfailed++
		} else if !info.Running {
			style = topsl.StyleWarn
			m.nstopped++
		} else {
			style = topsl.StyleGood
			m.nrunning++
		}
		styles = append(styles, style)
	}

	m.lines = lines
	m.styles = styles

	m.statusbar.SetStatus(fmt.Sprintf(
		"%6d Services %6d Faulted %6d Running %6d Standby %6d Disabled",
		len(m.items),
		m.nfailed, m.nrunning, m.nstopped, m.ndisabled))

	if m.nfailed > 0 {
		m.statusbar.SetFail()
	} else if m.nstopped > 0 {
		m.statusbar.SetWarn()
	} else if m.nrunning > 0 {
		m.statusbar.SetGood()
	} else {
		m.statusbar.SetNormal()
	}

	words := []string{"[Q] Quit", "[H] Help"}

	if item := m.selected; item != nil {
		words = append(words, "[I] Info")
		words = append(words, "[L] Log")
		if !item.Enabled {
			words = append(words, "[E] Enable")
		} else {
			words = append(words, "[D] Disable")
			if item.Failed {
				words = append(words, "[C] Clear")
			}
			words = append(words, "[R] Restart")
		}
	}
	m.keybar.SetKeys(words)
}
