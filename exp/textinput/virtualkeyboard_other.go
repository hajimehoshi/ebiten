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

//go:build !ios && !js

package textinput

import (
	"image"
)

// readVirtualKeyboard reports whether a virtual keyboard is shown and the client-area region it
// leaves visible, in native pixels. ok is false when the region is unknown.
//
// No platform here reports a virtual keyboard yet.
func readVirtualKeyboard() (visible bool, visibleClientRegion image.Rectangle, ok bool) {
	// TODO: Implement this for Android (#2831).
	return false, image.Rectangle{}, false
}
