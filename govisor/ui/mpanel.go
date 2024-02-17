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

	"github.com/gdamore/govisor/govisor/util"
	"github.com/gdamore/govisor/rest"
)

type sorted []*rest.ServiceInfo

var (
	StyleNormal = tcell.StyleDefault.
			Foreground(tcell.ColorSilver).
			Background(tcell.ColorBlack)
	StyleGood = tcell.StyleDefault.
			Foreground(tcell.ColorGreen).
			Background(tcell.ColorBlack)
	StyleWarn = tcell.StyleDefault.
			Foreground(tcell.ColorYellow).
			Background(tcell.ColorBlack)
	StyleError = tcell.StyleDefault.
			Foreground(tcell.ColorMaroon).
			Background(tcell.ColorBlack)
)

func (s sorted) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s sorted) Len() int {
	return len(s)
}

func (s sorted) Less(i, j int) bool {
	a := s[i]
	b := s[j]

	if a.Failed != b.Failed {
		// put failed items at front
		return a.Failed
	}
	if a.Enabled != b.Enabled {
		// enabled in front of non-enabled items
		return a.Enabled
	}
	// We don't worry about suspended items vs. running -- no clear order
	// there.  We just sort based on name
	return a.Name < b.Name
}

// MainPanel implements a Widget as a Panel, but provides the data
// model and handling for the content area, using data loaded from a Govisor
// REST API service.
type MainPanel struct {
	info      *rest.ServiceInfo
	name      string // service name
	err       error  // last error retrieving state
	content   *views.CellView
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
	styles    []tcell.Style
	items     []*rest.ServiceInfo

	Panel
}

// mainModel provides the model for a CellArea.
type mainModel struct {
	m *MainPanel
}

func NewMainPanel(app *App, server string) *MainPanel {
	m := &MainPanel{}

	m.Panel.Init(app)
	m.content = views.NewCellView()
	m.SetContent(m.content)

	m.content.SetModel(&mainModel{m})
	m.content.SetStyle(StyleNormal)

	m.SetTitle(server)
	m.SetKeys([]string{"[Q] Quit"})

	return m
}

func (m *MainPanel) Draw() {
	m.update()
	m.Panel.Draw()
}

func (m *MainPanel) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEsc:
			m.unselect()
			return true
		case tcell.KeyF1:
			m.App().ShowHelp()
			return true
		case tcell.KeyEnter:
			if m.selected != nil {
				m.App().ShowInfo(m.selected.Name)
				return true
			}
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'Q', 'q':
				m.App().Quit()
				return true
			case 'H', 'h':
				m.App().ShowHelp()
				return true
			case 'I', 'i':
				if m.selected != nil {
					m.App().ShowInfo(m.selected.Name)
					return true
				}
			case 'L', 'l':
				if m.selected != nil {
					m.App().ShowLog(m.selected.Name)
					return true
				} else {
					m.App().ShowLog("")
					return true
				}
			case 'E', 'e':
				if m.selected != nil && !m.selected.Enabled {
					m.App().EnableService(m.selected.Name)
					return true
				}
			case 'D', 'd':
				if m.selected != nil && m.selected.Enabled {
					m.App().DisableService(m.selected.Name)
					return true
				}
			case 'C', 'c':
				if m.selected != nil && m.selected.Failed {
					m.App().ClearService(m.selected.Name)
					return true
				}
			case 'R', 'r':
				if m.selected != nil {
					m.App().RestartService(m.selected.Name)
					return true
				}
			}
		}
	}
	return m.Panel.HandleEvent(ev)
}

// Model items
func (model *mainModel) GetCell(x, y int) (rune, tcell.Style, []rune, int) {
	var ch rune
	var style tcell.Style

	m := model.m

	if y < 0 || y >= len(m.lines) {
		return ch, StyleNormal, nil, 1
	}

	if x >= 0 && x < len(m.lines[y]) {
		ch = rune(m.lines[y][x])
	} else {
		ch = ' '
	}
	style = m.styles[y]
	if m.items[y] == m.selected {
		style = style.Reverse(true)
	}
	return ch, style, nil, 1
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

	items, err := m.App().GetItems()
	m.items = items

	// preserve selected item
	if sel := m.selected; sel != nil {
		m.selected = nil
		cury := 0
		for _, item := range m.items {
			if item.Name == sel.Name {
				m.selected = item
				m.cury = cury
			}
			cury++
		}
	}
	if err != nil {
		switch e := err.(type) {
		case *rest.Error:
			if e.Code == 401 {
				m.App().ShowAuth()
				return
			}
		}
		m.SetError()
		m.SetStatus(fmt.Sprintf("Cannot load items: %v", err))
		m.lines = []string{}
		m.styles = []tcell.Style{}
		return
	}

	lines := make([]string, 0, len(m.items))
	styles := make([]tcell.Style, 0, len(m.items))

	m.ndisabled = 0
	m.nfailed = 0
	m.nstopped = 0
	m.nrunning = 0

	m.height = 0
	m.width = 0

	for _, info := range items {
		d := time.Since(info.TimeStamp)
		d -= d % time.Second
		line := fmt.Sprintf("%-20s %-10s %10s   %-10s",
			info.Name, util.Status(info), util.FormatDuration(d),
			info.Status)

		if len(line) > m.width {
			m.width = len(line)
		}
		m.height++

		lines = append(lines, line)
		var style tcell.Style
		if !info.Enabled {
			style = StyleNormal
			m.ndisabled++
		} else if info.Failed {
			style = StyleError
			m.nfailed++
		} else if !info.Running {
			style = StyleWarn
			m.nstopped++
		} else {
			style = StyleGood
			m.nrunning++
		}
		styles = append(styles, style)
	}

	m.lines = lines
	m.styles = styles

	m.SetStatus(fmt.Sprintf(
		"%6d Services %6d Faulted %6d Running %6d Standby %6d Disabled",
		len(m.items),
		m.nfailed, m.nrunning, m.nstopped, m.ndisabled))

	if m.nfailed > 0 {
		m.SetError()
	} else if m.nstopped > 0 {
		m.SetWarn()
	} else if m.nrunning > 0 {
		m.SetGood()
	} else {
		m.SetNormal()
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
	} else {
		words = append(words, "[L] Log")
	}
	m.SetKeys(words)
}
