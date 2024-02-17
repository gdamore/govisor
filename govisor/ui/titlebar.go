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

type TitleBar struct {
	once sync.Once
	views.SimpleStyledTextBar
}

func (tb *TitleBar) Init() {
	tb.once.Do(func() {
		normal := tcell.StyleDefault.
			Foreground(tcell.ColorBlack).
			Background(tcell.ColorSilver)
		alternate := tcell.StyleDefault.
			Foreground(tcell.ColorBlue).
			Background(tcell.ColorSilver)

		tb.SimpleStyledTextBar.Init()
		tb.SimpleStyledTextBar.SetStyle(normal)
		tb.RegisterLeftStyle('N', normal)
		tb.RegisterLeftStyle('A', alternate)
		tb.RegisterCenterStyle('N', normal)
		tb.RegisterCenterStyle('A', alternate)
		tb.RegisterRightStyle('N', normal)
		tb.RegisterRightStyle('A', alternate)
	})
}

func NewTitleBar() *TitleBar {
	tb := &TitleBar{}
	tb.Init()
	return tb
}
