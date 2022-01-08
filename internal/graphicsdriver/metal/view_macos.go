// Copyright 2019 The Ebiten Authors
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

//go:build darwin && !ios
// +build darwin,!ios

package metal

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ns"
)

func (v *view) setWindow(window uintptr) {
	// NSView can be updated e.g., fullscreen-state is switched.
	v.window = window
	v.windowChanged = true
}

func (v *view) setUIView(uiview uintptr) {
	panic("metal: setUIView is not available on macOS")
}

func (v *view) update() {
	if !v.windowChanged {
		return
	}

	cocoaWindow := ns.NewWindow(v.window)
	cocoaWindow.ContentView().SetLayer(v.ml)
	cocoaWindow.ContentView().SetWantsLayer(true)
	v.windowChanged = false
}

func (v *view) usePresentsWithTransaction() bool {
	// Disable presentsWithTransaction on the fullscreen mode (#1745).
	return !v.vsyncDisabled
}

func (v *view) maximumDrawableCount() int {
	// When presentsWithTransaction is YES and triple buffering is enabled, nextDrawing returns immediately once every two times.
	// This makes FPS doubled. To avoid this, disable the triple buffering.
	if v.usePresentsWithTransaction() {
		return 2
	}
	return 3
}

const (
	storageMode         = mtl.StorageModeManaged
	resourceStorageMode = mtl.ResourceStorageModeManaged
)
