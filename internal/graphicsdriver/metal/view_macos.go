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
	"runtime"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
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
	v.ml.SetMaximumDrawableCount(v.maximumDrawableCount())

	if !v.windowChanged {
		return
	}

	// TODO: Should this be called on the main thread?
	cocoaWindow := cocoa.NSWindow{ID: objc.ID(v.window)}
	cocoaWindow.ContentView().SetLayer(uintptr(v.ml.Layer()))
	cocoaWindow.ContentView().SetWantsLayer(true)

	v.windowChanged = false
}

const (
	storageMode         = mtl.StorageModeManaged
	resourceStorageMode = mtl.ResourceStorageModeManaged
)

func (v *view) maximumDrawableCount() int {
	// Note that the architecture might not be the true reason of the issues (#2880, #2883).
	// Hajime tested only MacBook Pro 2020 (Intel) and MacBook Pro 2023 (M3).

	// Use 3 for Intel Mac and iOS. With 2, There are some situations that the FPS becomes half, or the FPS becomes too low (#2880).
	if runtime.GOARCH == "amd64" {
		return 3
	}

	// Use 3 in fullscren.
	// Though this might degrade FPS, this is necessary to avoid mysterious rendering delays.
	if v.isFullscreen() {
		return 3
	}

	// Use 2 for a Wnidow to avoid mysterious blinking (#2883).
	return 2
}

func (v *view) isFullscreen() bool {
	return cocoa.NSWindow{ID: objc.ID(v.window)}.StyleMask()&cocoa.NSWindowStyleMaskFullScreen != 0
}
