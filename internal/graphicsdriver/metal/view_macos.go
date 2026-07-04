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
	"time"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

const kCVReturnSuccess = 0

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

func (v *view) initializeOS() error {
	if err := v.initDisplayLink(); err != nil {
		return err
	}
	return nil
}

func (v *view) waitForDisplayLinkOutputCallback() {
	if v.caDisplayLink == 0 {
		return
	}
	if v.vsyncDisabled.Load() {
		return
	}
	v.fence.wait()
}

// presentDrawable registers the drawable presentation on the command buffer.
func (v *view) presentDrawable(cb mtl.CommandBuffer, d ca.MetalDrawable) {
	// While vsync is disabled, the number of drawables queued for presentation is tracked
	// so that nextDrawable can skip a frame instead of blocking until a drawable is available.
	if v.vsyncDisabled.Load() {
		v.presentedHandlerOnce.Do(func() {
			// addPresentedHandler is available as of macOS 10.15.4.
			if !d.CanAddPresentedHandler() {
				return
			}
			v.presentedHandler = objc.NewBlock(func(block objc.Block, drawable objc.ID) {
				v.queuedPresents.Add(-1)
			})
		})
		if v.presentedHandler != 0 {
			v.queuedPresents.Add(1)
			d.AddPresentedHandler(v.presentedHandler)
		}
	}
	v.lastPresentTime = time.Now()
	cb.PresentDrawable(d)
}
