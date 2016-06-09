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

	"github.com/gdamore/govisor/rest"
)

type AuthPanel struct {
	hlayout    *views.BoxLayout
	left       *views.BoxLayout
	right      *views.BoxLayout
	uprompt    *views.Text
	pprompt    *views.Text
	ufield     *views.Text
	pfield     *views.Text
	passactive bool
	username   []rune
	password   []rune
	info       *rest.ServiceInfo
	err        error // last error retrieving state

	Panel
}

func NewAuthPanel(app *App, server string) *AuthPanel {
	apanel := &AuthPanel{}
	apanel.Panel.Init(app)

	st := tcell.StyleDefault.
		Foreground(tcell.ColorSilver).
		Background(tcell.ColorBlack)

	apanel.username = make([]rune, 0, 128)
	apanel.password = make([]rune, 0, 128)

	apanel.hlayout = views.NewBoxLayout(views.Horizontal)
	apanel.left = views.NewBoxLayout(views.Vertical)
	apanel.right = views.NewBoxLayout(views.Vertical)
	apanel.uprompt = views.NewText()
	apanel.pprompt = views.NewText()
	apanel.ufield = views.NewText()
	apanel.pfield = views.NewText()
	apanel.ufield.SetText("                ")
	apanel.pfield.SetText("                ")
	apanel.uprompt.SetText("Username: ")
	apanel.pprompt.SetText("Password: ")

	apanel.uprompt.SetStyle(st)
	apanel.pprompt.SetStyle(st)

	apanel.ufield.SetStyle(st)
	apanel.pfield.SetStyle(st)

	apanel.hlayout.SetStyle(st)
	apanel.left.SetStyle(st)
	apanel.right.SetStyle(st)

	apanel.left.AddWidget(views.NewSpacer(), 1.0)
	apanel.left.AddWidget(apanel.uprompt, 0.0)
	apanel.left.AddWidget(apanel.pprompt, 0.0)
	apanel.left.AddWidget(views.NewSpacer(), 1.0)

	apanel.right.AddWidget(views.NewSpacer(), 1.0)
	apanel.right.AddWidget(apanel.ufield, 0.0)
	apanel.right.AddWidget(apanel.pfield, 0.0)
	apanel.right.AddWidget(views.NewSpacer(), 1.0)

	apanel.hlayout.AddWidget(views.NewSpacer(), 1.0)
	apanel.hlayout.AddWidget(apanel.left, 0.0)
	apanel.hlayout.AddWidget(apanel.right, 0.0)
	apanel.hlayout.AddWidget(views.NewSpacer(), 1.0)

	apanel.SetTitle(server)
	apanel.SetStatus("Authentication Required")
	apanel.SetKeys([]string{"[ESC] Quit"})
	apanel.SetContent(apanel.hlayout)

	return apanel
}

func (a *AuthPanel) ResetFields() {
	a.passactive = false
	a.username = a.username[0:0]
	a.password = a.password[0:0]
}

func (a *AuthPanel) Draw() {
	a.update()
	a.Panel.Draw()
}

func (a *AuthPanel) HandleEvent(ev tcell.Event) bool {
	switch ev := ev.(type) {
	case *tcell.EventKey:
		switch ev.Key() {
		case tcell.KeyEsc:
			a.App().Quit()
			return true
		case tcell.KeyTab, tcell.KeyEnter:
			if a.passactive {
				a.App().SetUserPassword(string(a.username),
					string(a.password))
				a.App().ShowMain()
			} else {
				a.passactive = true
			}
		case tcell.KeyBacktab:
			if a.passactive {
				a.passactive = false
			}
		case tcell.KeyCtrlU, tcell.KeyCtrlW:
			if a.passactive {
				a.password = a.password[:0]
			} else {
				a.username = a.username[:0]
			}
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if a.passactive {
				if len(a.password) > 0 {
					a.password =
						a.password[:len(a.password)-1]
				}
			} else {
				if len(a.username) > 0 {
					a.username =
						a.username[:len(a.username)-1]
				}
			}
		case tcell.KeyRune:
			r := ev.Rune()
			if a.passactive {
				if len(a.password) < 256 {
					a.password = append(a.password, r)
				}
			} else {
				if len(a.username) < 256 {
					a.username = append(a.username, r)
				}
			}
		default:
			return false
		}
		return true
	}
	return a.Panel.HandleEvent(ev)
}

// update must be called with AppLock held.
func (a *AuthPanel) update() {

	maxlen := 16

	a.Panel.SetError()

	var passprompt []rune
	userprompt := append([]rune{}, a.username...)

	for range a.password {
		passprompt = append(passprompt, '*')
	}
	if !a.passactive {
		userprompt = append(userprompt, '_')
	} else {
		passprompt = append(passprompt, '_')
	}

	if len(userprompt) > maxlen {
		userprompt = userprompt[len(userprompt)-maxlen:]
		userprompt[0] = '<'
	}
	for len(userprompt) < maxlen {
		userprompt = append(userprompt, ' ')
	}
	if len(passprompt) > maxlen {
		passprompt = passprompt[len(passprompt)-maxlen:]
		passprompt[0] = '<'
	}
	for len(passprompt) < maxlen {
		passprompt = append(passprompt, ' ')
	}
	a.ufield.SetText(string(userprompt))
	a.pfield.SetText(string(passprompt))

	focus := tcell.StyleDefault.
		Foreground(tcell.ColorWhite).Background(tcell.ColorNavy)
	idle := tcell.StyleDefault.
		Foreground(tcell.ColorSilver).Background(tcell.ColorBlack)

	if a.passactive {
		a.pfield.SetStyle(focus)
		a.ufield.SetStyle(idle)
	} else {
		a.ufield.SetStyle(focus)
		a.pfield.SetStyle(idle)
	}
}
