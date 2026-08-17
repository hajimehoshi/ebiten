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

// TextInputDriver is the platform side of text inputting: the hidden EditText
// serving the IME, implemented by EbitenView.
type TextInputDriver interface {
	// StartTextInput seeds the platform text buffer with text and the
	// selection, in UTF-16 units, and shows the virtual keyboard. The caret
	// bounds, in the view's pixels, anchor the platform text editor. States
	// reported for the buffer carry generation back.
	StartTextInput(text string, selectionStartInUTF16, selectionEndInUTF16, caretX, caretY, caretWidth, caretHeight, generation int)

	// DismissTextInput hides the virtual keyboard and drops the platform text
	// buffer's content.
	DismissTextInput()
}

// androidTextInput is the text-input plumbing between the platform view and
// exp/textinput.
type androidTextInput struct {
	driver      TextInputDriver
	onState     func(text string, selectionStartInUTF16, selectionEndInUTF16, composingStartInUTF16, composingEndInUTF16 int, passthroughKey bool, generation int)
	onEndByUser func()
	vkShown     bool
	vkRegion    image.Rectangle
	vkKnown     bool

	m sync.Mutex
}

var theAndroidTextInput androidTextInput

// SetTextInputDriver sets the platform text-input driver.
func (u *UserInterface) SetTextInputDriver(driver TextInputDriver) {
	t := &theAndroidTextInput
	t.m.Lock()
	defer t.m.Unlock()
	t.driver = driver
}

// SetTextInputStateCallback sets the callback receiving the platform text
// buffer's states reported through [UserInterface.UpdateTextInputState].
func (u *UserInterface) SetTextInputStateCallback(onState func(text string, selectionStartInUTF16, selectionEndInUTF16, composingStartInUTF16, composingEndInUTF16 int, passthroughKey bool, generation int)) {
	t := &theAndroidTextInput
	t.m.Lock()
	defer t.m.Unlock()
	t.onState = onState
}

// SetTextInputEndByUserCallback sets the callback receiving the user's
// ending reported through [UserInterface.DispatchTextInputEndByUser].
func (u *UserInterface) SetTextInputEndByUserCallback(onEndByUser func()) {
	t := &theAndroidTextInput
	t.m.Lock()
	defer t.m.Unlock()
	t.onEndByUser = onEndByUser
}

// textInputDriver returns the registered driver, or nil.
func (u *UserInterface) textInputDriver() TextInputDriver {
	t := &theAndroidTextInput
	t.m.Lock()
	defer t.m.Unlock()
	return t.driver
}

// StartPlatformTextInput starts text inputting with the platform driver and
// reports whether a driver is registered. caretBounds is in the client area's
// native pixels.
func (u *UserInterface) StartPlatformTextInput(text string, selectionStartInUTF16, selectionEndInUTF16 int, caretBounds image.Rectangle, generation int) bool {
	d := u.textInputDriver()
	if d == nil {
		return false
	}
	d.StartTextInput(text, selectionStartInUTF16, selectionEndInUTF16, caretBounds.Min.X, caretBounds.Min.Y, caretBounds.Dx(), caretBounds.Dy(), generation)
	return true
}

// DismissPlatformTextInput dismisses text inputting with the platform driver.
func (u *UserInterface) DismissPlatformTextInput() {
	d := u.textInputDriver()
	if d == nil {
		return
	}
	d.DismissTextInput()
}

// UpdateTextInputState reports the platform text buffer's state to the
// callback. The composing range is negative when no composition is in
// progress. passthroughKey reports whether the state comes with a key press
// the game receives too.
func (u *UserInterface) UpdateTextInputState(text string, selectionStartInUTF16, selectionEndInUTF16, composingStartInUTF16, composingEndInUTF16 int, passthroughKey bool, generation int) {
	t := &theAndroidTextInput
	t.m.Lock()
	onState := t.onState
	t.m.Unlock()
	if onState == nil {
		return
	}
	onState(text, selectionStartInUTF16, selectionEndInUTF16, composingStartInUTF16, composingEndInUTF16, passthroughKey, generation)
}

// DispatchTextInputEndByUser reports that the user ended text inputting,
// e.g. by dismissing the virtual keyboard, to the callback.
func (u *UserInterface) DispatchTextInputEndByUser() {
	t := &theAndroidTextInput
	t.m.Lock()
	onEndByUser := t.onEndByUser
	t.m.Unlock()
	if onEndByUser == nil {
		return
	}
	onEndByUser()
}

// UpdateVirtualKeyboardState records the virtual keyboard state: whether it is
// shown, and the client-area region it leaves visible in native pixels.
func (u *UserInterface) UpdateVirtualKeyboardState(shown bool, visibleRegion image.Rectangle) {
	t := &theAndroidTextInput
	t.m.Lock()
	defer t.m.Unlock()
	t.vkShown = shown
	t.vkRegion = visibleRegion
	t.vkKnown = true
}

// VirtualKeyboardState returns the last recorded virtual keyboard state. ok is
// false until a state is recorded.
func (u *UserInterface) VirtualKeyboardState() (shown bool, visibleRegion image.Rectangle, ok bool) {
	t := &theAndroidTextInput
	t.m.Lock()
	defer t.m.Unlock()
	return t.vkShown, t.vkRegion, t.vkKnown
}
