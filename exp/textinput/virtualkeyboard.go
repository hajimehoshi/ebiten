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

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// reportVirtualKeyboardToUI hands the caret and the platform's virtual keyboard geometry to the
// UI, which shifts the rendering so that the keyboard does not cover the caret. Call it every
// tick a session is live, and [clearVirtualKeyboardFromUI] once it ends.
func reportVirtualKeyboardToUI(caretBounds image.Rectangle) {
	_, region, ok := readVirtualKeyboard()
	ui.Get().SetVirtualKeyboardVisibleRegion(region, ok)
	ui.Get().SetTextInputCaretBounds(caretBounds)
}

// clearVirtualKeyboardFromUI ends the shift set up by [reportVirtualKeyboardToUI].
func clearVirtualKeyboardFromUI() {
	ui.Get().ClearTextInputCaretBounds()
}
