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

package ui_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// TestRestoreWindowWithMaximumSizeLimit tests that Restore reaches the backend even when a
// maximum window size is set (#3651). The maximizable check exists to keep Maximize from
// exceeding the configured maximum size; it has no bearing on restoring a window that is already
// maximized or minimized, and must not silently turn Restore into a no-op.
func TestRestoreWindowWithMaximumSizeLimit(t *testing.T) {
	w := ui.NewDesktopWindowForTest()
	w.SetResizingMode(ui.WindowResizingModeEnabled)
	w.SetSizeLimits(glfw.DontCare, glfw.DontCare, 1280, 960)

	fw := w.InstallFakeBackendForTest(true)

	w.Restore()

	if !fw.Restored {
		t.Errorf("Restore did not reach the backend when a maximum window size was set")
	}
}

// TestRestoreWindowWithFixedSize tests that Restore still does nothing for a fixed-size window,
// whose minimum and maximum sizes are both specified and equal. Restoring such a window can
// reposition it on macOS (#2259), so this case must stay blocked even though #3651 lifted the
// block for windows that merely have a maximum size.
func TestRestoreWindowWithFixedSize(t *testing.T) {
	w := ui.NewDesktopWindowForTest()
	w.SetResizingMode(ui.WindowResizingModeEnabled)
	w.SetSizeLimits(500, 500, 500, 500)

	fw := w.InstallFakeBackendForTest(true)

	w.Restore()

	if fw.Restored {
		t.Errorf("Restore reached the backend for a fixed-size window; this must stay blocked (#2259)")
	}
}
