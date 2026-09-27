// Copyright 2026 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ui_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestIsShortcutChord(t *testing.T) {
	for _, tt := range []struct {
		name     string
		ctrl     bool
		alt      bool
		meta     bool
		altGraph bool
		isApple  bool
		want     bool
	}{
		{
			name: "no modifiers",
			want: false,
		},
		{
			name:    "no modifiers on macOS",
			isApple: true,
			want:    false,
		},
		{
			name: "Ctrl+C",
			ctrl: true,
			want: true,
		},
		{
			name:    "Ctrl+C on macOS",
			ctrl:    true,
			isApple: true,
			want:    true,
		},
		{
			name:    "Cmd+S",
			meta:    true,
			isApple: true,
			want:    true,
		},
		{
			name:    "Cmd+Option+A",
			alt:     true,
			meta:    true,
			isApple: true,
			want:    true,
		},
		{
			name: "Alt+F on Windows or Linux",
			alt:  true,
			want: true,
		},
		{
			name:    "Option+A on macOS",
			alt:     true,
			isApple: true,
			want:    false,
		},
		{
			name:     "AltGr+Q on Windows",
			ctrl:     true,
			alt:      true,
			altGraph: true,
			want:     false,
		},
		{
			name:     "AltGr+E on Linux",
			altGraph: true,
			want:     false,
		},
		{
			name:     "AltGr with Meta",
			meta:     true,
			altGraph: true,
			want:     true,
		},
		{
			name: "Ctrl+Alt without AltGraph",
			ctrl: true,
			alt:  true,
			want: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ui.IsShortcutChordForTest(tt.ctrl, tt.alt, tt.meta, tt.altGraph, tt.isApple); got != tt.want {
				t.Errorf("IsShortcutChordForTest(ctrl: %t, alt: %t, meta: %t, altGraph: %t, isApple: %t): got: %t, want: %t", tt.ctrl, tt.alt, tt.meta, tt.altGraph, tt.isApple, got, tt.want)
			}
		})
	}
}
