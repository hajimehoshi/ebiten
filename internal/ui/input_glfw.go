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
	"io/fs"
	"math"
	"runtime"
	"sync"
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

// glfwInput owns the input state that the GLFW callbacks write on the main thread and the game
// goroutine drains once per tick.
type glfwInput struct {
	state InputState

	// savedCursorX and savedCursorY are the cursor position kept across a fullscreen transition
	// with a disabled cursor, in logical units. They are NaN when no position is saved.
	savedCursorX float64
	savedCursorY float64

	// rawCursorX and rawCursorY are the cursor position fetched for the current frame,
	// in GLFW pixels.
	rawCursorX float64
	rawCursorY float64

	lastWheelOffsetX float64
	lastWheelOffsetY float64
	lastWheelTime    time.Time

	mu sync.Mutex
}

// handleKey records a key action reported by GLFW.
func (i *glfwInput) handleKey(key Key, action glfw.Action, mods glfw.ModifierKey, t InputTime) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if action != glfw.Press {
		i.state.setKeyReleased(key, t)
		return
	}

	i.state.setKeyPressed(key, t)
	// See the comment on syncModKeysByMods for why this is needed and why it is macOS-only.
	// TODO: This may be redundant with the per-tick syncModKeysFromOS poll. Confirm
	// within-tick IsKeyJustPressed/IsKeyJustReleased semantics for Cmd+A still hold
	// without it, then remove.
	if runtime.GOOS == "darwin" {
		i.state.syncModKeysByMods(mods, t)
	}
}

// handleMouseButton records a mouse button action reported by GLFW.
func (i *glfwInput) handleMouseButton(button MouseButton, action glfw.Action, t InputTime) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if action == glfw.Press {
		i.state.setMouseButtonPressed(button, t)
		return
	}
	i.state.setMouseButtonReleased(button, t)
}

// appendRune records a character reported by GLFW.
func (i *glfwInput) appendRune(r rune) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.state.appendRune(r)
}

// handleScroll records a wheel offset reported by GLFW, dropping an anomalous value.
func (i *glfwInput) handleScroll(xoff, yoff float64) {
	i.mu.Lock()
	defer i.mu.Unlock()

	now := time.Now()

	// Sometimes the wheel event accepts anomalous values like sudden spikes and rapid reversals (#3390).
	// Such values should be ignored.
	if now.Sub(i.lastWheelTime) < 100*time.Millisecond {
		// Thresholds are determined in a heuristic way.
		const (
			rapidReversalThreshold = 0.75
			spikeThreshold         = 50
		)
		if math.Abs(xoff) >= 1 && i.lastWheelOffsetX != 0 {
			rate := math.Abs(xoff) / math.Abs(i.lastWheelOffsetX)
			sb := i.lastWheelOffsetX*xoff > 0
			if rate >= spikeThreshold && sb {
				xoff = 0
			}
			if rate >= rapidReversalThreshold && !sb {
				xoff = 0
			}
		}
		if math.Abs(yoff) >= 1 && i.lastWheelOffsetY != 0 {
			rate := math.Abs(yoff) / math.Abs(i.lastWheelOffsetY)
			sb := i.lastWheelOffsetY*yoff > 0
			if rate >= spikeThreshold && sb {
				yoff = 0
			}
			if rate >= rapidReversalThreshold && !sb {
				yoff = 0
			}
		}
	}

	i.lastWheelOffsetX = xoff
	i.lastWheelOffsetY = yoff
	i.lastWheelTime = now

	i.state.WheelX += xoff
	i.state.WheelY += yoff
}

// syncModKeys reconciles the modifier key state against mods.
func (i *glfwInput) syncModKeys(mods glfw.ModifierKey, t InputTime) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.state.syncModKeysByMods(mods, t)
}

// setLockKeys records the state of the lock keys.
func (i *glfwInput) setLockKeys(capsLock, numLock LockKeyState) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.state.CapsLock = capsLock
	i.state.NumLock = numLock
}

// setWindowBeingClosed records that the window is being closed.
func (i *glfwInput) setWindowBeingClosed() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.state.WindowBeingClosed = true
}

// setDroppedFiles records the files dropped onto the window.
func (i *glfwInput) setDroppedFiles(files fs.FS) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.state.DroppedFiles = files
}

// read copies the accumulated input state into dst and resets it.
func (i *glfwInput) read(dst *InputState) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.state.copyAndReset(dst)
}

// clearSavedCursorPos drops the cursor position saved for a fullscreen transition.
func (i *glfwInput) clearSavedCursorPos() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.savedCursorX = math.NaN()
	i.savedCursorY = math.NaN()
}

// saveCursorPos records the current cursor position so that it survives a fullscreen transition.
func (i *glfwInput) saveCursorPos() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.savedCursorX = i.state.CursorX
	i.savedCursorY = i.state.CursorY
}

// takeSavedCursorPos returns the saved cursor position in logical units and clears it.
// ok is false when no position is saved.
func (i *glfwInput) takeSavedCursorPos() (x, y float64, ok bool) {
	i.mu.Lock()
	defer i.mu.Unlock()

	x, y = i.savedCursorX, i.savedCursorY
	i.savedCursorX = math.NaN()
	i.savedCursorY = math.NaN()
	return x, y, !math.IsNaN(x) && !math.IsNaN(y)
}

// setRawCursorPos records the cursor position fetched for the current frame, in GLFW pixels.
func (i *glfwInput) setRawCursorPos(x, y float64) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.rawCursorX, i.rawCursorY = x, y
}

// rawCursorPos returns the cursor position fetched for the current frame, in GLFW pixels.
func (i *glfwInput) rawCursorPos() (x, y float64) {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.rawCursorX, i.rawCursorY
}

// setCursorPos records the cursor position for the current frame, in logical units.
func (i *glfwInput) setCursorPos(x, y float64) {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.state.CursorX, i.state.CursorY = x, y
}

func (u *glfwBackend) registerInputCallbacks() error {
	if _, err := u.window.SetKeyCallback(func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		// Ignore key repeats for now.
		if action == glfw.Repeat {
			return
		}

		uk, ok := glfwKeyToUIKey[key]
		if !ok {
			return
		}
		u.input.handleKey(uk, action, mods, u.InputTime())
	}); err != nil {
		return err
	}

	if _, err := u.window.SetMouseButtonCallback(func(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
		// Ignore key repeats for now.
		if action == glfw.Repeat {
			return
		}

		ub, ok := glfwMouseButtonToMouseButton[button]
		if !ok {
			return
		}
		u.input.handleMouseButton(ub, action, u.InputTime())
	}); err != nil {
		return err
	}

	// The character callback skips the characters that are produced with the modifier combinations
	// the platform treats as shortcuts, like Ctrl+= on X11 (#3502).
	if _, err := u.window.SetCharCallback(func(w *glfw.Window, char rune) {
		u.input.appendRune(char)
	}); err != nil {
		return err
	}

	if _, err := u.window.SetScrollCallback(func(w *glfw.Window, xoff float64, yoff float64) {
		u.input.handleScroll(xoff, yoff)
	}); err != nil {
		return err
	}

	return nil
}

// updateInputStateForFrame updates the input state using pre-fetched cursor position
// and device scale factor. GetCursorPos and gamepad.Update are already called in
// the mainThread.Call block of updateGame, so this avoids an extra round-trip.
func (u *glfwBackend) updateInputStateForFrame(deviceScaleFactor float64) error {
	s := deviceScaleFactor

	cx, cy, ok := u.input.takeSavedCursorPos()
	if ok {
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
		rx, ry := u.input.rawCursorPos()
		cx2 := dipFromGLFWPixel(rx, s)
		cy2 := dipFromGLFWPixel(ry, s)
		cx, cy = u.context.clientPositionToLogicalPosition(cx2, cy2, s)
	}

	// The adjusted position can be NaN at the initialization.
	if !math.IsNaN(cx) && !math.IsNaN(cy) {
		u.input.setCursorPos(cx, cy)
	}

	// gamepad.Update is already called in updateGame's mainThread.Call block.
	return nil
}

func (u *glfwBackend) KeyName(key Key) string {
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

// syncModKeysByMods reconciles per-key modifier state with a mods bitmask.
// Needed on macOS to recover from text-input intercepts (e.g. Cmd+A) and
// system hotkey absorption (e.g. Cmd+Shift+4). macOS-only: on Windows,
// AltGr would make Ctrl appear stuck (#3453).
//
// TODO: Revisit how modifier keys are tracked once modifiers get their own type
// (#3498). This reconciliation exists because modifier state is derived from the
// per-key press and release times, which the OS does not always deliver. If
// IsModifierPressed reads the mods bitmask directly instead, this may become
// unnecessary.
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
		if mods&m.mod != 0 {
			// The mod flag is set, so at least one of left/right should be pressed.
			// If one already is, the state is in sync: leave its press time intact,
			// otherwise the key would keep registering as just-pressed every tick.
			leftPressed := i.KeyPressedTimes[m.left] > i.KeyReleasedTimes[m.left]
			rightPressed := i.KeyPressedTimes[m.right] > i.KeyReleasedTimes[m.right]
			if leftPressed || rightPressed {
				continue
			}
			// Neither is pressed: press whichever was most recently pressed,
			// defaulting to the left variant if neither was ever pressed.
			if i.KeyPressedTimes[m.left] >= i.KeyPressedTimes[m.right] {
				i.setKeyPressed(m.left, t)
			} else {
				i.setKeyPressed(m.right, t)
			}
			continue
		}
		// The mod flag is clear: release any variant currently in the pressed state.
		if i.KeyPressedTimes[m.left] > i.KeyReleasedTimes[m.left] {
			i.setKeyReleased(m.left, t)
		}
		if i.KeyPressedTimes[m.right] > i.KeyReleasedTimes[m.right] {
			i.setKeyReleased(m.right, t)
		}
	}
}
