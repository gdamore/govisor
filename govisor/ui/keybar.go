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

	"github.com/gdamore/tcell"
	"github.com/gdamore/tcell/views"
)

type KeyBar struct {
	once sync.Once
	views.SimpleStyledTextBar
}

func (k *KeyBar) Init() {
	k.once.Do(func() {
		normal := tcell.StyleDefault.
			Foreground(tcell.ColorBlack).
			Background(tcell.ColorSilver)
		alternate := tcell.StyleDefault.
			Foreground(tcell.ColorBlue).
			Background(tcell.ColorSilver).Bold(true)

		k.SimpleStyledTextBar.Init()
		k.SimpleStyledTextBar.SetStyle(normal)
		k.RegisterLeftStyle('N', normal)
		k.RegisterLeftStyle('A', alternate)
	})
}

func (k *KeyBar) SetKeys(words []string) {
	b := make([]rune, 0, 80)
	for i, w := range words {
		esc := false
		if i != 0 && len(w) != 0 {
			b = append(b, ' ')
		}
		for _, r := range w {
			if esc {
				if r == ']' {
					b = append(b, '%', 'N')
					esc = false
				} else if r == '%' {
					b = append(b, '%')
				}
				b = append(b, r)

			} else {
				b = append(b, r)
				if r == '[' {
					esc = true
					b = append(b, '%', 'A')
				} else if r == '%' {
					b = append(b, '%')
				}
			}
		}
	}
	k.SetLeft(string(b))
}

func NewKeyBar() *KeyBar {
	kb := &KeyBar{}
	kb.Init()
	return kb
}
