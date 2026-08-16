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

package ui

import (
	"image"
	"sync"
)

// virtualKeyboard holds what the screen transform needs in order to keep a text-input caret
// out of the region a virtual keyboard covers. The text-input package writes it; the values
// it reports come from the platform, which is why they are not measured here.
type virtualKeyboard struct {
	caretBounds image.Rectangle
	caretKnown  bool

	visibleRegion      image.Rectangle
	visibleRegionKnown bool

	mu sync.Mutex
}

var theVirtualKeyboard virtualKeyboard

// SetTextInputCaretBounds records the caret of the live text-input session in logical units.
// The rendering shifts to keep the caret out of the region a virtual keyboard covers.
func (u *UserInterface) SetTextInputCaretBounds(bounds image.Rectangle) {
	v := &theVirtualKeyboard
	v.mu.Lock()
	defer v.mu.Unlock()
	v.caretBounds = bounds
	v.caretKnown = true
}

// ClearTextInputCaretBounds drops the caret recorded by
// [UserInterface.SetTextInputCaretBounds], ending the shift.
func (u *UserInterface) ClearTextInputCaretBounds() {
	v := &theVirtualKeyboard
	v.mu.Lock()
	defer v.mu.Unlock()
	v.caretBounds = image.Rectangle{}
	v.caretKnown = false
}

// SetVirtualKeyboardVisibleRegion records the client-area region a virtual keyboard leaves
// visible, in native pixels. known is false when the platform cannot report it, in which case
// no shift happens.
func (u *UserInterface) SetVirtualKeyboardVisibleRegion(region image.Rectangle, known bool) {
	v := &theVirtualKeyboard
	v.mu.Lock()
	defer v.mu.Unlock()
	v.visibleRegion = region
	v.visibleRegionKnown = known
}

// state returns the caret bounds in logical units and the visible client region in native
// pixels. ok is false unless both are known.
func (v *virtualKeyboard) state() (caretBounds, visibleRegion image.Rectangle, ok bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !v.caretKnown || !v.visibleRegionKnown {
		return image.Rectangle{}, image.Rectangle{}, false
	}
	return v.caretBounds, v.visibleRegion, true
}
