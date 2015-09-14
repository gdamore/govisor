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
	"log"
	"os"
	"sync"
	"time"

	"github.com/gdamore/govisor/rest"
	"github.com/gdamore/topsl"
)

var dlog *log.Logger

func init() {
	f, e := os.Create("/tmp/tlog")
	if e == nil {
		dlog = log.New(f, "DEBUG:", log.LstdFlags)
		log.SetOutput(f)
	}
}

type App struct {
	entries   *Entries
	log       *topsl.TextArea
	info      *InfoPanel
	help      *topsl.TextArea
	about     *topsl.TextArea
	titlebar  *topsl.TitleBar
	statusbar *topsl.StatusBar
	content   topsl.Widget
	keybar    *topsl.KeyBar
	panel     *topsl.Panel
	server    string
	client    *rest.Client
	closeq    chan struct{}
	refresh   chan bool
}

// Entries represents both the *model* and the *view* wrapped in one.
type Entries struct {
	selected  int
	items     []*rest.ServiceInfo
	lines     []string
	styles    []topsl.Style
	width     int
	height    int
	curx      int
	cury      int
	nfailed   int
	nstopped  int
	ndisabled int
	nrunning  int
	status    string
	keys      []string
	view      *topsl.CellView
	statusbar *topsl.StatusBar
	keybar    *topsl.KeyBar
	titlebar  *topsl.TitleBar
	sync.Mutex
}

func (m *Entries) GetCursor() (int, int, bool, bool) {
	return m.curx, m.cury, true, false
}

func (m *Entries) MoveCursor(offx, offy int) {
	m.curx += offx
	m.cury += offy
	m.updateCursor()
}

func (m *Entries) SetCursor(x, y int) {
	m.curx = x
	m.cury = y
	m.updateCursor()
}

func (m *Entries) unselect() {
	m.selected = -1
	m.cury = 0
	m.curx = 0
}

func (m *Entries) updateCursor() {
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
	if m.height > 0 {
		if m.selected < 0 {
			m.curx = 0
			m.cury = 0
		}
		m.selected = m.cury
	} else {
		m.selected = -1
	}

	words := []string{"_Quit", "_Help"}
	if m.selected >= 0 && m.selected < len(m.items) {
		info := m.items[m.selected]

		words = append(words, "_Info")
		words = append(words, "_Log")
		if !info.Enabled {
			words = append(words, "_Enable")
		} else {
			words = append(words, "_Disable")
			if info.Failed {
				words = append(words, "_Clear fault")
			}
			words = append(words, "_Restart")
		}
	}
	m.keys = words
	if m.keybar != nil {
		m.keybar.SetKeys(words)
	}
}

func (m *Entries) SetInfos(items []*rest.ServiceInfo) {
	sortInfos(items)
	m.items = items
	m.update()
}

func (m *Entries) update() {
	lines := make([]string, 0, len(m.items))
	styles := make([]topsl.Style, 0, len(m.items))

	m.ndisabled = 0
	m.nfailed = 0
	m.nstopped = 0
	m.nrunning = 0

	m.height = 0
	m.width = 0

	for _, info := range m.items {
		d := time.Since(info.TimeStamp)
		d -= d % time.Second
		line := fmt.Sprintf("%-20s %-10s %10s   %10s",
			info.Name, status(info), d.String(), info.Status)

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

	m.status = fmt.Sprintf(
		"%6d Services %6d Faulted %6d Running %6d Standby %6d Disabled",
		len(m.items),
		m.nfailed, m.nrunning, m.nstopped, m.ndisabled)

	m.Draw()
}

func (m *Entries) GetCell(x, y int) (rune, topsl.Style) {
	var ch rune
	var style topsl.Style

	if y < 0 || y >= len(m.lines) {
		return ch, style
	}

	if x >= 0 && x < len(m.lines[y]) {
		ch = rune(m.lines[y][x])
	} else {
		ch = ' '
	}
	style = m.styles[y]
	if y == m.selected {
		style = style.Reverse()
	}
	return ch, style
}

func (m *Entries) GetBounds() (int, int) {
	// This assumes that all content is displayable runes of width 1.
	y := len(m.lines)
	x := 0
	for _, l := range m.lines {
		if x < len(l) {
			x = len(l)
		}
	}
	return x, y
}

func (m *Entries) getSelected() *rest.ServiceInfo {
	if m.selected < 0 || m.selected >= len(m.items) {
		return nil
	}
	return m.items[m.selected]
}

func (e *Entries) Draw() {
	if e.nfailed > 0 {
		e.statusbar.SetFail()
	} else if e.nstopped > 0 {
		e.statusbar.SetWarn()
	} else if e.nrunning > 0 {
		e.statusbar.SetGood()
	} else {
		e.statusbar.SetNormal()
	}
	e.statusbar.SetStatus(e.status)

	e.keybar.SetKeys(e.keys)
	e.view.Draw()
}

func (e *Entries) Resize() {
	e.view.Resize()
}

func (e *Entries) SetView(view topsl.View) {
	e.view.SetView(view)
}

func (e *Entries) HandleEvent(ev topsl.Event) bool {
	switch ev := ev.(type) {
	case *topsl.KeyEvent:
		switch ev.Ch {
		case 0:
			switch ev.Key {
			case topsl.KeyEsc:
				e.SetCursor(0, 0)
				e.selected = -1
				return true
			}
		}
	}
	return e.view.HandleEvent(ev)
}

func NewEntries(a *App) *Entries {
	e := &Entries{}
	e.view = topsl.NewCellView()
	e.view.SetModel(e)
	e.keybar = a.keybar
	e.titlebar = a.titlebar
	e.statusbar = a.statusbar
	return e
}

type InfoPanel struct {
	text      *topsl.TextArea
	info      *rest.ServiceInfo
	statusbar *topsl.StatusBar
	keybar    *topsl.KeyBar
	titlebar  *topsl.TitleBar
}

func NewInfoPanel(a *App) *InfoPanel {
	ipanel := &InfoPanel{
		text:      topsl.NewTextArea(),
		info:      nil,
		titlebar:  a.titlebar,
		keybar:    a.keybar,
		statusbar: a.statusbar,
	}

	return ipanel
}

func (i *InfoPanel) Draw() {

	s := i.info
	if s == nil {
		return
	}
	lines := make([]string, 0, 8)
	d := time.Now().Sub(s.TimeStamp)
	d -= d % time.Second
	lines = append(lines, fmt.Sprintf("%13s %s", "Name:", s.Name))
	lines = append(lines, fmt.Sprintf("%13s %s", "Description:",
		s.Description))
	lines = append(lines, fmt.Sprintf("%13s %s", "Status:", status(s)))
	lines = append(lines, fmt.Sprintf("%13s %v (%v)", "Since:", d, s.TimeStamp))
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
		l = l + fmt.Sprintf("   %s", p)
	}
	lines = append(lines, l)

	i.titlebar.SetCenter("Details for "+s.Name, topsl.StyleTitle)
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

	i.text.SetLines(lines)
	i.keybar.SetKeys([]string{"_Quit"})

	i.text.Draw()
	i.text.EnableCursor(false)
}

func (i *InfoPanel) HandleEvent(e topsl.Event) bool {
	return i.text.HandleEvent(e)
}

func (i *InfoPanel) SetView(view topsl.View) {
	i.text.SetView(view)
}

func (i *InfoPanel) Resize() {
	i.text.Resize()
}

func (i *InfoPanel) setInfo(s *rest.ServiceInfo) {
	i.info = s
	i.Draw()
	return
	lines := make([]string, 0, 8)
	d := time.Now().Sub(s.TimeStamp)
	d -= d % time.Second
	lines = append(lines, fmt.Sprintf("%13s %s", "Name:", s.Name))
	lines = append(lines, fmt.Sprintf("%13s %s", "Description:",
		s.Description))
	lines = append(lines, fmt.Sprintf("%13s %s", "Status:", status(s)))
	lines = append(lines, fmt.Sprintf("%13s %v (%v)", "Since:", d, s.TimeStamp))
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
		l = l + fmt.Sprintf("   %s", p)
	}
	lines = append(lines, l)

	i.titlebar.SetCenter("Details for "+s.Name, topsl.StyleTitle)
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

	i.text.SetLines(lines)
	i.keybar.SetKeys([]string{"_Quit"})
}

func (a *App) setContent(w topsl.Widget) {
	a.content = w
	a.panel.SetContent(w)
}

func (a *App) doHelp() {
	a.help.SetLines([]string{
		"Insert help text here.",
		"",
		"This program is distributed under the Apache 2.0 License",
		"Copyright 2015 The Govisor Authors",
	})
	a.titlebar.SetCenter("Help", topsl.StyleTitle)
	a.keybar.SetKeys([]string{"_Quit"})
	a.statusbar.SetNormal()
	a.statusbar.SetStatus("")
	a.setContent(a.help)
}

func (a *App) doEntries() {
	a.titlebar.SetCenter(a.server, topsl.StyleTitle)
	a.setContent(a.entries)
	a.entries.update()
}

func (a *App) doInfo() {
	if a.content != a.entries {
		return
	}
	s := a.entries.getSelected()
	if s == nil {
		return
	}

	a.info.setInfo(s)
	a.setContent(a.info)
}

func (a *App) doLog() {
	if a.content != a.entries {
		return
	}
	s := a.entries.getSelected()
	if s == nil {
		return
	}
	lines, e := a.client.GetServiceLog(s.Name)
	if e != nil {
		a.Die(e.Error())
	}
	a.log.SetLines(lines)
	a.titlebar.SetCenter("Log for "+s.Name, topsl.StyleTitle)
	a.statusbar.SetNormal()
	a.statusbar.SetStatus("")
	a.keybar.SetKeys([]string{"_Quit"})
	a.setContent(a.log)
}

func (a *App) doDisable() {
	s := a.entries.getSelected()
	if s == nil {
		return
	}
	a.entries.unselect()
	a.client.DisableService(s.Name)
	a.kick()
}

func (a *App) doEnable() {
	s := a.entries.getSelected()
	if s == nil {
		return
	}
	a.entries.unselect()
	a.client.EnableService(s.Name)
	a.kick()
}

func (a *App) doClear() {
	s := a.entries.getSelected()
	if s == nil {
		return
	}
	a.entries.unselect()
	a.client.ClearService(s.Name)
	a.kick()
}

func (a *App) doRestart() {
	s := a.entries.getSelected()
	if s == nil {
		return
	}
	a.entries.unselect()
	a.client.RestartService(s.Name)
	a.kick()
}

func (a *App) Die(s string) {
	topsl.AppFini()
	fmt.Printf("Failure: %s", s)
	os.Exit(1)
}

// doLog

func (a *App) HandleEvent(ev topsl.Event) bool {
	switch ev := ev.(type) {
	case *topsl.KeyEvent:
		switch ev.Ch {
		case 0:
			switch ev.Key {
			case topsl.KeyCtrlL:
				// XXX: redraw screen
				topsl.AppRedraw()
			}
		case 'Q', 'q':
			if a.content == a.info || a.content == a.log ||
				a.content == a.help {
				a.doEntries()
			} else {
				topsl.AppFini()
			}
			return true

		case 'H', 'h':
			a.doHelp()
			return true

		case 'L', 'l':
			a.doLog()
			return true

		case 'I', 'i':
			a.doInfo()
			return true

		case 'R', 'r':
			a.doRestart()
			return true

		case 'E', 'e':
			a.doEnable()
			return true

		case 'D', 'd':
			a.doDisable()
			return true

		case 'C', 'c':
			a.doClear()
			return true
		}
	}
	return a.panel.HandleEvent(ev)
}

func (a *App) Draw() {

	a.panel.Draw()
}

func (a *App) Resize() {
	a.panel.Resize()
}

func (a *App) SetView(view topsl.View) {
	if a.panel != nil {
		a.panel.SetView(view)
	}
}

func (a *App) kick() {
	select {
	case a.refresh <- true:
	default:
	}
}

func (a *App) refreshLoop() {
	for {
		a.client.Watch()
		a.refreshInfos()
		select {
		default:
		case <-a.closeq:
			return
		}
	}
}

func (a *App) refreshInfos() {
	names, e := a.client.Services()
	if e != nil {
		// XXX: signal an alert
		topsl.AppFini()
		fmt.Errorf("Failed to query services: %v", e)
		panic("failed: " + e.Error())
	}

	infos := []*rest.ServiceInfo{}
	for _, n := range names {
		info, e := a.client.GetService(n)
		if e == nil {
			infos = append(infos, info)
		}
	}

	topsl.AppLock()
	a.entries.SetInfos(infos)
	a.info.setInfo(a.entries.getSelected())
	topsl.AppUnlock()
	topsl.AppDraw()
}

func (a *App) redrawLoop() {
	for {
		select {
		case <-a.closeq:
			return
		case <-time.After(time.Second):
		}
		topsl.AppDraw()
	}
}

func NewApp(client *rest.Client, url string) *App {

	app := &App{}
	app.client = client
	app.server = url
	app.keybar = topsl.NewKeyBar(nil)
	app.statusbar = topsl.NewStatusBar("")
	app.titlebar = topsl.NewTitleBar(url)
	app.info = NewInfoPanel(app)
	app.help = topsl.NewTextArea()
	app.log = topsl.NewTextArea()
	app.entries = NewEntries(app)

	app.titlebar.SetRight("Govisor 1.0", topsl.StyleStatus)
	app.titlebar.SetCenter(url, topsl.StyleTitle)

	app.panel = topsl.NewPanel()
	app.panel.SetTitle(app.titlebar)
	app.panel.SetBottom(app.keybar)
	app.panel.SetStatus(app.statusbar)
	app.panel.SetContent(app.entries)
	app.refresh = make(chan bool)
	app.closeq = make(chan struct{})
	return app
}

func doUI(client *rest.Client, url string) {
	app := NewApp(client, url)

	topsl.AppInit()
	app.doEntries()
	go app.refreshLoop()
	go app.redrawLoop()
	topsl.SetApplication(app)
	topsl.RunApplication()
}

/*
   Our screen has the following appearance:

    Server: http://localhost:8321/
    xxx Services  xxx Running  yyy Faulted  zzz Standby            Govisor v1.0
   ____________________________________________________________________________
   ...
   testservice:name        faulted      4d10m32s    Failed: Terminated
   ...
   testservice:ok          running            5s    Service started
   ...
   dontrunme:ever          disabled    132d10m5s    Service disabled
   ...
   ____________________________________________________________________________
   [Q]uit [I]Info [E]nable [D]isable [R]estart [C]lear [L]og  [H]elp
*/
