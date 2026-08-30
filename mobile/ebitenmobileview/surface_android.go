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

// OnSurfaceChanged is called from EbitenSurfaceView's render thread with the surface's size in
// pixels. Without it, the size is derived from the view's size in device-independent pixels.
//
// OnSurfaceChanged is concurrent safe.
func OnSurfaceChanged(width, height int) {
	ui.Get().SetSurfaceSize(width, height)
}
