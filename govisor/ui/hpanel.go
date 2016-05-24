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
	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

type HelpPanel struct {
	text *views.TextArea
	Panel
}

func (h *HelpPanel) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEsc:
			h.App().ShowMain()
			return true
		case tcell.KeyRune:
			switch ev.Rune() {
			case 'Q', 'q':
				h.app.ShowMain()
				return true
			}
		}
	}
	return h.Panel.HandleEvent(ev)
}

func (h *HelpPanel) Draw() {
	h.SetKeys([]string{"[ESC] Main"})
	h.SetTitle("Help")
	h.Panel.Draw()
}

func (h *HelpPanel) Init(app *App) {

	h.Panel.Init(app)

	// No, we don't have context-sensitive help.
	h.text = views.NewTextArea()
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
		"Copyright 2016 The Govisor Authors",
	})
	h.SetContent(h.text)
}

func NewHelpPanel(app *App) *HelpPanel {

	h := &HelpPanel{}

	h.Init(app)
	return h
}
