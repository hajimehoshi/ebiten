// Copyright 2015 Hajime Hoshi
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
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

var glfwMouseButtonToMouseButton = map[glfw.MouseButton]MouseButton{
	glfw.MouseButtonLeft:   MouseButton0,
	glfw.MouseButtonMiddle: MouseButton1,
	glfw.MouseButtonRight:  MouseButton2,
	glfw.MouseButton4:      MouseButton3,
	glfw.MouseButton5:      MouseButton4,
}

func (u *UserInterface) registerInputCallbacks() error {
	if _, err := u.window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		// Ignore key repeats for now.
		if action == glfw.Repeat {
			return
		}

		// As this function is called from GLFW callbacks, the current thread is main.
		u.m.Lock()
		defer u.m.Unlock()

		uk, ok := glfwKeyToUIKey[key]
		if !ok {
			return
		}
		t := u.InputTime()
		if action == glfw.Press {
			u.inputState.setKeyPressed(uk, t)
			// On macOS, modifier keys can appear released prematurely when the text input system
			// intercepts certain key combinations (e.g. Ctrl+A). The mods parameter on the key event
			// still correctly reflects which modifiers are physically held. Use it to re-assert
			// modifier key states that may have been incorrectly released.
			u.inputState.syncModKeysByMods(mods, t)
		} else {
			u.inputState.setKeyReleased(uk, t)
		}
	}); err != nil {
		return err
	}

	if _, err := u.window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		// Ignore key repeats for now.
		if action == glfw.Repeat {
			return
		}

		// As this function is called from GLFW callbacks, the current thread is main.
		u.m.Lock()
		defer u.m.Unlock()

		ub, ok := glfwMouseButtonToMouseButton[button]
		if !ok {
			return
		}
		if action == glfw.Press {
			u.inputState.setMouseButtonPressed(ub, u.InputTime())
		} else {
			u.inputState.setMouseButtonReleased(ub, u.InputTime())
		}
	}); err != nil {
		return err
	}

	if _, err := u.window.SetCharModsCallback(func(w *glfw.Window, char rune, mods glfw.ModifierKey) {
		// As this function is called from GLFW callbacks, the current thread is main.
		u.m.Lock()
		defer u.m.Unlock()
		u.inputState.appendRune(char)
	}); err != nil {
		return err
	}

	if _, err := u.window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		// As this function is called from GLFW callbacks, the current thread is main.
		u.m.Lock()
		defer u.m.Unlock()

		now := time.Now()

		// Sometimes the wheel event accepts anomalous values like sudden spikes and rapid reversals (#3390).
		// Such values should be ignored.
		if now.Sub(u.lastWheelTime) < 100*time.Millisecond {
			// Thresholds are determined in a heuristic way.
			const (
				rapidReversalThreshold = 0.75
				spikeThreshold         = 50
			)
			if math.Abs(xoff) >= 1 && u.lastWheelOffsetX != 0 {
				rate := math.Abs(xoff) / math.Abs(u.lastWheelOffsetX)
				sb := u.lastWheelOffsetX*xoff > 0
				if rate >= spikeThreshold && sb {
					xoff = 0
				}
				if rate >= rapidReversalThreshold && !sb {
					xoff = 0
				}
			}
			if math.Abs(yoff) >= 1 && u.lastWheelOffsetY != 0 {
				rate := math.Abs(yoff) / math.Abs(u.lastWheelOffsetY)
				sb := u.lastWheelOffsetY*yoff > 0
				if rate >= spikeThreshold && sb {
					yoff = 0
				}
				if rate >= rapidReversalThreshold && !sb {
					yoff = 0
				}
			}
		}

		u.lastWheelOffsetX = xoff
		u.lastWheelOffsetY = yoff
		u.lastWheelTime = now

		u.inputState.WheelX += xoff
		u.inputState.WheelY += yoff
	}); err != nil {
		return err
	}

	return nil
}

// updateInputStateForFrame updates the input state using pre-fetched cursor position
// and device scale factor. GetCursorPos and gamepad.Update are already called in
// the mainThread.Call block of updateGame, so this avoids an extra round-trip.
func (u *UserInterface) updateInputStateForFrame(deviceScaleFactor float64) error {
	u.m.Lock()
	defer u.m.Unlock()

	s := deviceScaleFactor

	cx, cy := u.savedCursorX, u.savedCursorY
	u.savedCursorX = math.NaN()
	u.savedCursorY = math.NaN()

	if !math.IsNaN(cx) && !math.IsNaN(cy) {
		// Rare path: cursor position was saved (e.g. fullscreen transition with disabled cursor).
		// SetCursorPos requires the main thread.
		cx2, cy2 := u.context.logicalPositionToClientPosition(cx, cy, s)
		cx2 = dipToGLFWPixel(cx2, s)
		cy2 = dipToGLFWPixel(cy2, s)
		var err error
		u.mainThread.Call(func() {
			err = u.window.SetCursorPos(cx2, cy2)
		})
		if err != nil {
			return err
		}
	} else {
		// Common path: use the pre-fetched raw cursor position.
		cx2 := dipFromGLFWPixel(u.rawCursorX, s)
		cy2 := dipFromGLFWPixel(u.rawCursorY, s)
		cx, cy = u.context.clientPositionToLogicalPosition(cx2, cy2, s)
	}

	// AdjustPosition can return NaN at the initialization.
	if !math.IsNaN(cx) && !math.IsNaN(cy) {
		u.inputState.CursorX, u.inputState.CursorY = cx, cy
	}

	// gamepad.Update is already called in updateGame's mainThread.Call block.
	return nil
}

func (u *UserInterface) KeyName(key Key) string {
	if !u.isRunning() {
		return ""
	}

	gk, ok := uiKeyToGLFWKey[key]
	if !ok {
		return ""
	}

	var name string
	u.mainThread.Call(func() {
		if u.isTerminated() {
			return
		}
		n, err := glfw.GetKeyName(gk, 0)
		if err != nil {
			u.setError(err)
			return
		}
		name = n
	})
	return name
}

// syncModKeysByMods re-asserts modifier key states based on the mods bitmask
// from a key event. On macOS, the text input system can intercept modifier+key
// combinations (e.g. Ctrl+A) and prematurely release the modifier key via
// flagsChanged. The mods parameter on the key event still correctly reflects
// which modifiers are physically held, so we use it to restore the state.
func (i *InputState) syncModKeysByMods(mods glfw.ModifierKey, t InputTime) {
	type modMapping struct {
		mod   glfw.ModifierKey
		left  Key
		right Key
	}
	mappings := [...]modMapping{
		{glfw.ModControl, KeyControlLeft, KeyControlRight},
		{glfw.ModShift, KeyShiftLeft, KeyShiftRight},
		{glfw.ModAlt, KeyAltLeft, KeyAltRight},
		{glfw.ModSuper, KeyMetaLeft, KeyMetaRight},
	}
	for _, m := range mappings {
		if mods&m.mod == 0 {
			continue
		}
		// The mod flag is set, so at least one of left/right should be pressed.
		// Re-press whichever was most recently pressed.
		// If neither was ever pressed, default to the left variant.
		lp := i.KeyPressedTimes[m.left]
		rp := i.KeyPressedTimes[m.right]
		if lp >= rp {
			i.setKeyPressed(m.left, t)
		} else {
			i.setKeyPressed(m.right, t)
		}
	}
}
