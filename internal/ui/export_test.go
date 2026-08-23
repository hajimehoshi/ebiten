// Copyright 2025 The Ebitengine Authors
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
)

func (i *InputState) SetKeyPressed(key Key, t InputTime) {
	i.setKeyPressed(key, t)
}

func (i *InputState) SetKeyReleased(key Key, t InputTime) {
	i.setKeyReleased(key, t)
}

func (i *InputState) SetMouseButtonPressed(button MouseButton, t InputTime) {
	i.setMouseButtonPressed(button, t)
}

func (i *InputState) SetMouseButtonReleased(button MouseButton, t InputTime) {
	i.setMouseButtonReleased(button, t)
}

func IsConnectionReset(err error) bool {
	return isConnectionReset(err)
}

func SetVirtualKeyboardStateForTest(caretBounds image.Rectangle, caretKnown bool, visibleRegion image.Rectangle, visibleRegionKnown bool) {
	v := &theVirtualKeyboard
	v.mu.Lock()
	defer v.mu.Unlock()
	v.caretBounds = caretBounds
	v.caretKnown = caretKnown
	v.visibleRegion = visibleRegion
	v.visibleRegionKnown = visibleRegionKnown
}

// ScreenScaleAndOffsetsForTest returns the screen transform for the given sizes, with the
// virtual keyboard state set by SetVirtualKeyboardStateForTest applied.
func ScreenScaleAndOffsetsForTest(screenWidth, screenHeight int, offscreenWidth, offscreenHeight float64) (scale, offsetX, offsetY float64) {
	c := &context{
		screenWidth:     screenWidth,
		screenHeight:    screenHeight,
		offscreenWidth:  offscreenWidth,
		offscreenHeight: offscreenHeight,
	}
	c.updateVirtualKeyboardOffsetY()
	return c.screenScaleAndOffsets()
}
