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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui

import (
	"image"
	"testing"
)

type windowSetterTestBackend struct {
	uiBackend
	window backendWindow
}

func (b *windowSetterTestBackend) Window() backendWindow {
	return b.window
}

type windowSetterTestWindow struct {
	nullWindow

	decorated        bool
	floating         bool
	size             image.Point
	monitor          *Monitor
	mousePassthrough bool
}

func (w *windowSetterTestWindow) SetDecorated(decorated bool) {
	w.decorated = decorated
}

func (w *windowSetterTestWindow) SetFloating(floating bool) {
	w.floating = floating
}

func (w *windowSetterTestWindow) SetMonitor(monitor *Monitor) {
	w.monitor = monitor
}

func (w *windowSetterTestWindow) SetSize(width, height int) {
	w.size = image.Pt(width, height)
}

func (w *windowSetterTestWindow) SetMousePassthrough(enabled bool) {
	w.mousePassthrough = enabled
}

func TestDesktopWindowSettersKeepInitStateAfterBackendPublishes(t *testing.T) {
	monitor := &Monitor{}
	tests := []struct {
		name  string
		set   func(*desktopWindow)
		check func(*testing.T, *UserInterface, *windowSetterTestWindow)
	}{
		{
			name: "decorated",
			set: func(w *desktopWindow) {
				w.SetDecorated(false)
			},
			check: func(t *testing.T, u *UserInterface, w *windowSetterTestWindow) {
				t.Helper()
				if u.desktopWindow.isInitWindowDecorated() || w.decorated {
					t.Errorf("SetDecorated(false) left init=%t window=%t; want false, false", u.desktopWindow.isInitWindowDecorated(), w.decorated)
				}
			},
		},
		{
			name: "floating",
			set: func(w *desktopWindow) {
				w.SetFloating(true)
			},
			check: func(t *testing.T, u *UserInterface, w *windowSetterTestWindow) {
				t.Helper()
				if !u.desktopWindow.isInitWindowFloating() || !w.floating {
					t.Errorf("SetFloating(true) left init=%t window=%t; want true, true", u.desktopWindow.isInitWindowFloating(), w.floating)
				}
			},
		},
		{
			name: "monitor",
			set: func(w *desktopWindow) {
				w.SetMonitor(monitor)
			},
			check: func(t *testing.T, u *UserInterface, w *windowSetterTestWindow) {
				t.Helper()
				if u.getInitMonitor() != monitor || w.monitor != monitor {
					t.Errorf("SetMonitor left init=%p window=%p; want %p", u.getInitMonitor(), w.monitor, monitor)
				}
			},
		},
		{
			name: "size",
			set: func(w *desktopWindow) {
				w.SetSize(320, 240)
			},
			check: func(t *testing.T, u *UserInterface, w *windowSetterTestWindow) {
				t.Helper()
				width, height := u.desktopWindow.getInitWindowSizeInDIP()
				if width != 320 || height != 240 || w.size != image.Pt(320, 240) {
					t.Errorf("SetSize(320, 240) left init=(%d, %d) window=%v; want (320, 240), (320, 240)", width, height, w.size)
				}
			},
		},
		{
			name: "mouse passthrough",
			set: func(w *desktopWindow) {
				w.SetMousePassthrough(true)
			},
			check: func(t *testing.T, u *UserInterface, w *windowSetterTestWindow) {
				t.Helper()
				if !u.desktopWindow.isInitWindowMousePassthrough() || !w.mousePassthrough {
					t.Errorf("SetMousePassthrough(true) left init=%t window=%t; want true, true", u.desktopWindow.isInitWindowMousePassthrough(), w.mousePassthrough)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			u := &UserInterface{}
			if err := u.init(); err != nil {
				t.Fatal(err)
			}
			window := &windowSetterTestWindow{}
			u.setRunningBackend(&windowSetterTestBackend{window: window})

			test.set(&u.desktopWindow)
			test.check(t, u, window)
		})
	}
}
