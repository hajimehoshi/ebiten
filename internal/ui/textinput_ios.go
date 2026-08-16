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
	"sync"
)

// iosTextInput is the text-input plumbing between the platform view and
// exp/textinput.
type iosTextInput struct {
	dispatchKeyPress func(key Key)
	dispatchKeyDown  func(key Key)
	dispatchKeyUp    func(key Key)

	m sync.Mutex
}

var theIOSTextInput iosTextInput

// SetKeyPressDispatcher sets the function delivering the key presses reported
// by a platform text editor rather than by the game's view.
func (u *UserInterface) SetKeyPressDispatcher(dispatch func(key Key)) {
	t := &theIOSTextInput
	t.m.Lock()
	defer t.m.Unlock()
	t.dispatchKeyPress = dispatch
}

// SetKeyEventDispatchers sets the functions delivering a key press and a key
// release reported by a platform text editor rather than by the game's view.
func (u *UserInterface) SetKeyEventDispatchers(down, up func(key Key)) {
	t := &theIOSTextInput
	t.m.Lock()
	defer t.m.Unlock()
	t.dispatchKeyDown = down
	t.dispatchKeyUp = up
}

// DispatchKeyDown delivers a key press to the game's key input. It does
// nothing until a dispatcher is registered.
func (u *UserInterface) DispatchKeyDown(key Key) {
	t := &theIOSTextInput
	t.m.Lock()
	dispatch := t.dispatchKeyDown
	t.m.Unlock()
	if dispatch == nil {
		return
	}
	dispatch(key)
}

// DispatchKeyUp delivers a key release to the game's key input. It does
// nothing until a dispatcher is registered.
func (u *UserInterface) DispatchKeyUp(key Key) {
	t := &theIOSTextInput
	t.m.Lock()
	dispatch := t.dispatchKeyUp
	t.m.Unlock()
	if dispatch == nil {
		return
	}
	dispatch(key)
}

// DispatchKeyPress delivers a key press, and its release, to the game's key
// input. It does nothing until a dispatcher is registered.
func (u *UserInterface) DispatchKeyPress(key Key) {
	t := &theIOSTextInput
	t.m.Lock()
	dispatch := t.dispatchKeyPress
	t.m.Unlock()
	if dispatch == nil {
		return
	}
	dispatch(key)
}

// IsKeyDown reports whether key is currently reported down by the platform's
// key events.
func (u *UserInterface) IsKeyDown(key Key) bool {
	if key < 0 || KeyMax < key {
		return false
	}
	u.m.Lock()
	defer u.m.Unlock()
	return u.inputState.KeyPressedTimes[key] > u.inputState.KeyReleasedTimes[key]
}
