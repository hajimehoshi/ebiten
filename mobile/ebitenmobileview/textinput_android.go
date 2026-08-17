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
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// TextInputDriver is the view side of text inputting: the hidden EditText
// serving the IME, implemented by EbitenView.
type TextInputDriver interface {
	// StartTextInput seeds the text buffer with text and the selection, in
	// UTF-16 units, and shows the virtual keyboard. The caret bounds, in the
	// view's pixels, anchor the platform text editor. States reported for the
	// buffer carry generation back.
	StartTextInput(text string, selectionStartInUTF16, selectionEndInUTF16, caretX, caretY, caretWidth, caretHeight, generation int)

	// DismissTextInput hides the virtual keyboard and drops the text buffer's
	// content.
	DismissTextInput()
}

// SetTextInputDriver sets the text-input driver.
func SetTextInputDriver(driver TextInputDriver) {
	ui.Get().SetTextInputDriver(driver)
}

// OnTextInputStateChanged reports the text buffer's state: its content, the
// selection and the composing range in UTF-16 units, and the generation of the
// StartTextInput the state belongs to. The composing range is negative when no
// composition is in progress. passthroughKey reports whether the state comes
// with a key press the game receives too.
func OnTextInputStateChanged(text string, selectionStartInUTF16, selectionEndInUTF16, composingStartInUTF16, composingEndInUTF16 int, passthroughKey bool, generation int) {
	ui.Get().UpdateTextInputState(text, selectionStartInUTF16, selectionEndInUTF16, composingStartInUTF16, composingEndInUTF16, passthroughKey, generation)
}

// OnVirtualKeyboardChanged reports whether the virtual keyboard is shown and
// the view region it leaves visible, in the view's pixels.
func OnVirtualKeyboardChanged(shown bool, x, y, width, height int) {
	ui.Get().UpdateVirtualKeyboardState(shown, image.Rect(x, y, x+width, y+height))
}

// OnTextInputEndedByUser reports that the user ended text inputting, e.g.
// with the Back key hiding the virtual keyboard.
func OnTextInputEndedByUser() {
	ui.Get().DispatchTextInputEndByUser()
}
