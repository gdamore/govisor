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
	"sync"

	"github.com/gdamore/tcell/views"
)

// Panel is just a wrapper around the views.Panel, but it changes
// the names of elements to match our usage, making it easier (hopefully)
// to grok what is going on.

type Panel struct {
	tb   *TitleBar
	sb   *StatusBar
	kb   *KeyBar
	once sync.Once
	app  *App

	views.Panel
}

func (p *Panel) SetTitle(title string) {
	p.tb.SetCenter(title)
}

func (p *Panel) SetKeys(words []string) {
	p.kb.SetKeys(words)
}

func (p *Panel) SetStatus(status string) {
	p.sb.SetText(status)
}

func (p *Panel) SetGood() {
	p.sb.SetGood()
}

func (p *Panel) SetNormal() {
	p.sb.SetNormal()
}

func (p *Panel) SetWarn() {
	p.sb.SetWarn()
}

func (p *Panel) SetError() {
	p.sb.SetError()
}

func (p *Panel) Init(app *App) {
	p.once.Do(func() {
		p.app = app

		p.tb = NewTitleBar()
		p.tb.SetRight(app.GetAppName())
		p.tb.SetCenter(" ")

		p.kb = NewKeyBar()

		p.sb = NewStatusBar()

		p.Panel.SetTitle(p.tb)
		p.Panel.SetMenu(p.sb)
		p.Panel.SetStatus(p.kb)
	})
}

func (p *Panel) App() *App {
	return p.app
}

func NewPanel(app *App) *Panel {
	p := &Panel{}
	p.Init(app)
	return p
}
