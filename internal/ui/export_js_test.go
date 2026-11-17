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
	"syscall/js"
	"time"
)

func IsShortcutChordForTest(ctrl, alt, meta, altGraph, isApple bool) bool {
	return isShortcutChord(ctrl, alt, meta, altGraph, isApple)
}

func IsKeyStringForTest(str string) bool {
	return isKeyString(str)
}

func (u *UserInterface) UpdateInputFromEventForTest(e js.Value) error {
	return u.updateInputFromEvent(e)
}

func (u *UserInterface) ReleaseAllInputsForTest() {
	u.releaseAllInputs()
}

func (u *UserInterface) InputStateForTest() InputState {
	return u.inputState
}

// NewKeyboardLayoutUIForTest replaces the browser API without requiring Keyboard Map support.
func NewKeyboardLayoutUIForTest(getLayoutMap js.Value) (*UserInterface, func()) {
	oldAvailable, oldGet := jsKeyboardLayoutAvailable, jsKeyboardGetLayoutMap
	oldCh, oldThen, oldCatch := jsKeyboardGetLayoutMapCh, jsKeyboardGetLayoutMapThenCallback, jsKeyboardGetLayoutMapCatchCallback
	jsKeyboardLayoutAvailable, jsKeyboardGetLayoutMap = true, getLayoutMap
	ch := make(chan js.Value, 1)
	jsKeyboardGetLayoutMapCh = ch
	then := js.FuncOf(func(this js.Value, args []js.Value) any { ch <- args[0]; return nil })
	catch := js.FuncOf(func(this js.Value, args []js.Value) any { ch <- js.Undefined(); return nil })
	jsKeyboardGetLayoutMapThenCallback, jsKeyboardGetLayoutMapCatchCallback = then, catch
	u := &UserInterface{}
	u.running.Store(true)
	return u, func() {
		jsKeyboardLayoutAvailable, jsKeyboardGetLayoutMap = oldAvailable, oldGet
		jsKeyboardGetLayoutMapCh, jsKeyboardGetLayoutMapThenCallback, jsKeyboardGetLayoutMapCatchCallback = oldCh, oldThen, oldCatch
		then.Release()
		catch.Release()
	}
}

func (u *UserInterface) ReadInputStateForTest() { var state InputState; u.readInputState(&state) }
func (u *UserInterface) ExpireKeyboardLayoutForTest() {
	u.keyboardLayoutMapTime = time.Now().Add(-2 * time.Second)
}
