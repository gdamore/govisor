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
	"github.com/gdamore/topsl"
)

type HelpPanel struct {
	app       *App
	text      *topsl.TextArea
	titlebar  *topsl.TitleBar
	statusbar *topsl.StatusBar
	keybar    *topsl.KeyBar
	panel     *topsl.Panel
}

func (h *HelpPanel) Draw() {
	h.panel.Draw()
}

func (h *HelpPanel) HandleEvent(ev topsl.Event) bool {
	switch ev := ev.(type) {
	case *topsl.KeyEvent:
		switch ev.Ch {
		case 0:
			switch ev.Key {
			case topsl.KeyEsc:
				h.app.ShowMain()
				return true
			}
		case 'Q', 'q':
			h.app.ShowMain()
			return true
		}
	}
	return h.panel.HandleEvent(ev)
}

func (h *HelpPanel) SetView(view topsl.View) {
	h.panel.SetView(view)
}

func (h *HelpPanel) Resize() {
	h.panel.Resize()
}

func NewHelpPanel(app *App) *HelpPanel {
	h := &HelpPanel{
		titlebar:  topsl.NewTitleBar(),
		statusbar: topsl.NewStatusBar(),
		keybar:    topsl.NewKeyBar(),
		text:      topsl.NewTextArea(),
		panel:     topsl.NewPanel(),
		app:       app,
	}

	h.titlebar.SetRight(app.GetAppName())
	h.titlebar.SetCenter("Help")

	h.text.SetLines([]string{
		"Supported keys (not all keys available in all contexts)",
		"",
		"  <ESC>          : return to main screen",
		"  <CTRL-C>       : quit",
		"  <CTRL-L>       : refresh the screeen",
		"  <H>            : show this help",
		"  <UP>, <DOWN>   : navigation",
		"  <E>            : enable selected service",
		"  <D>            : disable selected service",
		"  <I>            : view detailed information for service",
		"  <R>            : restart selected service",
		"  <C>            : clear faults on selected service",
		"  <L>            : view log for selected service",
		"",
		"This program is distributed under the Apache 2.0 License",
		"Copyright 2015 The Govisor Authors",
	})

	h.keybar.SetKeys([]string{"[ESC] Main"})
	h.panel.SetTitle(h.titlebar)
	h.panel.SetStatus(h.statusbar)
	h.panel.SetBottom(h.keybar)
	h.panel.SetContent(h.text)

	return h
}
