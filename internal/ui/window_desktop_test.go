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
	minimized bool
	restored  bool
}

func (w *restoreTestWindow) IsMaximized() bool {
	return false
}

func (w *restoreTestWindow) IsMinimized() bool {
	return w.minimized
}

func (w *restoreTestWindow) Restore() {
	w.restored = true
}

func TestRestoreMinimizedWindowWithMaximumSizeLimit(t *testing.T) {
	u := &UserInterface{}
	window := &restoreTestWindow{minimized: true}
	u.setRunningBackend(&restoreTestBackend{window: window})
	defer u.setRunningBackend(nil)

	w := &desktopWindow{ui: u}
	w.windowResizingMode.Store(WindowResizingModeEnabled)
	w.setWindowSizeLimitsInDIP(-1, -1, 1280, 960)
	w.Restore()

	if !window.restored {
		t.Fatal("Restore did not reach the backend for a minimized window with a maximum size limit")
	}
}
