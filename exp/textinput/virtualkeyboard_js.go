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

// readVirtualKeyboard reports whether a virtual keyboard is shown and the client-area region it
// leaves visible, in native pixels. ok is false when the region is unknown.
//
// A browser exposes neither the virtual keyboard state nor its geometry directly. Whether the
// keyboard is shown is inferred from the device's pointer capability while text inputting holds
// the DOM focus, and the region from the visual viewport, which the keyboard shrinks. The region
// also reflects any part of the canvas scrolled or panned out of the viewport.
func readVirtualKeyboard() (visible bool, visibleClientRegion image.Rectangle, ok bool) {
	visible = theTextInputImpl.textInputElementFocused() && isVirtualKeyboard()
	visibleClientRegion, ok = ui.Get().VisibleClientRegionInNativePixels()
	return visible, visibleClientRegion, ok
}
