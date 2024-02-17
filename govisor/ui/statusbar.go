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
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/gdamore/tcell/v2/views"
)

// StatusBar is like a titlebar, but it changes color based on the
// status of a screen -- e.g. red background to indicate a fault condition.
type StatusBar struct {
	once   sync.Once
	status string
	views.SimpleStyledTextBar
}

var (
	StatusBarStyleNormal = tcell.StyleDefault.
				Foreground(tcell.ColorBlack).
				Background(tcell.ColorSilver)
	StatusBarStyleGood = tcell.StyleDefault.
				Foreground(tcell.ColorWhite).
				Background(tcell.ColorGreen).
				Bold(true)
	StatusBarStyleWarn = tcell.StyleDefault.
				Foreground(tcell.ColorBlack).
				Background(tcell.ColorYellow)
	StatusBarStyleError = tcell.StyleDefault.
				Foreground(tcell.ColorWhite).
				Background(tcell.ColorMaroon).
				Bold(true)
)

func (sb *StatusBar) Init() {
	sb.once.Do(func() {
		sb.SimpleStyledTextBar.Init()
		sb.SetNormal()
	})
}

func (sb *StatusBar) SetStyle(style tcell.Style) {
	sb.SimpleStyledTextBar.SetStyle(style)
	sb.SimpleStyledTextBar.RegisterLeftStyle('N', style)
	sb.SimpleStyledTextBar.SetLeft(sb.status)
}

func (sb *StatusBar) SetGood() {
	sb.SetStyle(StatusBarStyleGood)
}

func (sb *StatusBar) SetNormal() {
	sb.SetStyle(StatusBarStyleNormal)
}

func (sb *StatusBar) SetWarn() {
	sb.SetStyle(StatusBarStyleWarn)
}

func (sb *StatusBar) SetError() {
	sb.SetStyle(StatusBarStyleError)
}

func (sb *StatusBar) SetText(status string) {
	sb.status = status
	sb.SetLeft(status)
}

func NewStatusBar() *StatusBar {
	sb := &StatusBar{}
	sb.Init()
	return sb
}
