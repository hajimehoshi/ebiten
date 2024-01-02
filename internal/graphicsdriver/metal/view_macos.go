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

package metal

import (
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

type viewPlatform struct {
	window        cocoa.NSWindow
	windowChanged bool
	fullscreen    bool
}

func (v *view) setWindow(window uintptr) {
	// NSView can be updated e.g., fullscreen-state is switched.
	v.window = cocoa.NSWindow{ID: objc.ID(window)}
	v.windowChanged = true
}

func (v *view) setUIView(uiview uintptr) {
	panic("metal: setUIView is not available on macOS")
}

func (v *view) update() {
	if v.windowChanged {
		// TODO: Should this be called on the main thread?
		v.window.ContentView().SetLayer(uintptr(v.ml.Layer()))
		v.window.ContentView().SetWantsLayer(true)
		v.windowChanged = false
	}

	fullscreen := v.window.StyleMask()&cocoa.NSWindowStyleMaskFullScreen != 0
	if v.fullscreen != fullscreen {
		v.fullscreen = fullscreen
		v.updateMaximumDrawableCount()
	}
}

func (v *view) isFullscreen() bool {
	return v.fullscreen
}

const (
	storageMode         = mtl.StorageModeManaged
	resourceStorageMode = mtl.ResourceStorageModeManaged
)
