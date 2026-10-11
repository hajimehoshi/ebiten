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

import "testing"

type restoreTestBackend struct {
	uiBackend
	window backendWindow
}

func (b *restoreTestBackend) Window() backendWindow {
	return b.window
}

type restoreTestWindow struct {
	backendWindow
	maximized bool
	minimized bool
	restored  bool
}

func (w *restoreTestWindow) IsMaximized() bool {
	return w.maximized
}

func (w *restoreTestWindow) IsMinimized() bool {
	return w.minimized
}

func (w *restoreTestWindow) Restore() {
	w.restored = true
}

func TestRestoreWindowWithMaximumSizeLimit(t *testing.T) {
	tests := []struct {
		name      string
		maximized bool
		minimized bool
		want      bool
	}{
		{name: "minimized", minimized: true, want: true},
		{name: "maximized and minimized", maximized: true, minimized: true, want: true},
		{name: "maximized", maximized: true, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UserInterface{}
			window := &restoreTestWindow{maximized: tt.maximized, minimized: tt.minimized}
			u.setRunningBackend(&restoreTestBackend{window: window})
			defer u.setRunningBackend(nil)

			w := &desktopWindow{ui: u}
			w.windowResizingMode.Store(WindowResizingModeEnabled)
			w.setWindowSizeLimitsInDIP(-1, -1, 1280, 960)
			w.Restore()

			if window.restored != tt.want {
				t.Fatalf("Restore reached the backend = %t, want %t", window.restored, tt.want)
			}
		})
	}
}
