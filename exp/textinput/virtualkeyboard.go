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

package textinput

import (
	"image"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// IsVirtualKeyboardVisible reports whether a virtual keyboard, a software keyboard on the
// screen, is shown. On Web browsers the report is best-effort.
//
// IsVirtualKeyboardVisible is concurrent-safe. The value is fixed within a tick.
func IsVirtualKeyboardVisible() bool {
	visible, _ := theVirtualKeyboardState.get()
	return visible
}

// VirtualKeyboardVisibleBounds returns the bounds of the screen region the user can see, not
// covered by a virtual keyboard, in logical units, the same coordinate space as the game screen.
// VirtualKeyboardVisibleBounds returns the whole screen bounds when no virtual keyboard is shown
// or the covered region is unknown.
//
// VirtualKeyboardVisibleBounds is useful in environments where the rendering result is not
// shifted automatically when a virtual keyboard appears, like Android and iOS without settings
// to shift the view. In such environments, adjust the rendering with the returned bounds so
// that the editing text is visible.
//
// VirtualKeyboardVisibleBounds is concurrent-safe. The value is fixed within a tick.
func VirtualKeyboardVisibleBounds() image.Rectangle {
	_, bounds := theVirtualKeyboardState.get()
	return bounds
}

// virtualKeyboardState caches the virtual keyboard state, refreshed at most once per tick.
type virtualKeyboardState struct {
	lastTick int64
	hasState bool
	visible  bool
	bounds   image.Rectangle

	// read and tick override the state and tick sources in tests. Nil means the platform
	// state and [ebiten.Tick].
	read func() (visible bool, visibleClientRegion image.Rectangle, ok bool)
	tick func() int64

	m sync.Mutex
}

var theVirtualKeyboardState virtualKeyboardState

func (v *virtualKeyboardState) currentTick() int64 {
	if v.tick != nil {
		return v.tick()
	}
	return ebiten.Tick()
}

func (v *virtualKeyboardState) get() (visible bool, bounds image.Rectangle) {
	v.m.Lock()
	defer v.m.Unlock()

	if t := v.currentTick(); !v.hasState || v.lastTick != t {
		v.visible, v.bounds = v.refresh()
		v.lastTick = t
		v.hasState = true
	}
	return v.visible, v.bounds
}

// refresh reads the platform state and converts the visible client region to logical units,
// clipped to the screen bounds.
func (v *virtualKeyboardState) refresh() (visible bool, bounds image.Rectangle) {
	read := v.read
	if read == nil {
		read = readVirtualKeyboard
	}
	visible, region, ok := read()

	sw, sh := ui.Get().LogicalScreenSize()
	full := image.Rect(0, 0, int(sw), int(sh))
	if !ok {
		return visible, full
	}

	x0, y0 := ui.Get().ClientPositionInNativePixelsToLogicalPosition(float64(region.Min.X), float64(region.Min.Y))
	x1, y1 := ui.Get().ClientPositionInNativePixelsToLogicalPosition(float64(region.Max.X), float64(region.Max.Y))
	if math.IsNaN(x0) || math.IsNaN(x1) {
		return visible, full
	}
	return visible, image.Rect(int(math.Ceil(x0)), int(math.Ceil(y0)), int(math.Floor(x1)), int(math.Floor(y1))).Intersect(full)
}
