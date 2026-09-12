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

package ebitenmobileview

import (
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

type position struct {
	x float64
	y float64
}

// Android MotionEvent action values.
// See https://developer.android.com/reference/android/view/MotionEvent
const (
	actionDown        = 0x00
	actionUp          = 0x01
	actionMove        = 0x02
	actionCancel      = 0x03
	actionPointerDown = 0x05
	actionPointerUp   = 0x06
)

// updateTouchesAndroid applies one Android MotionEvent action for a single pointer to touches.
// It reports whether the touch state was modified, so that the caller can update the input accordingly.
//
// An ACTION_CANCEL cancels the whole gesture: Android reports that every pointer that was down
// must then be considered lifted, not only the action-index pointer.
// See https://developer.android.com/reference/android/view/MotionEvent.html#ACTION_CANCEL
func updateTouchesAndroid(touches map[ui.TouchID]position, action, id int, x, y float64) bool {
	switch action {
	case actionDown, actionPointerDown, actionMove:
		touches[ui.TouchID(id)] = position{x, y}
		return true
	case actionUp, actionPointerUp:
		delete(touches, ui.TouchID(id))
		return true
	case actionCancel:
		clear(touches)
		return true
	}
	return false
}
