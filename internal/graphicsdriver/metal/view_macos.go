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
	// The layer content is updated by the drawable presentations, not by the view's display
	// mechanism. Keep AppKit from updating the layer content, especially when window resizing
	// starts. Otherwise, the layer content is reset and the window background is shown for a
	// moment (#3478).
	cocoaWindow.ContentView().SetLayerContentsRedrawPolicy(cocoa.NSViewLayerContentsRedrawNever)

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

// setDrawableSize records the new drawable size. The size is applied to the layer at the next
// drawable acquisition (see applyDrawableSizeIfNeeded).
//
// Changing the layer's drawable size drops the current layer content, and the drop can reach the
// screen before the next drawable is presented, showing the window background for a moment.
// Applying the size right before the drawable acquisition minimizes that period (#3478).
func (v *view) setDrawableSize(width, height int) {
	if v.drawableWidth == width && v.drawableHeight == height {
		return
	}
	v.drawableWidth = width
	v.drawableHeight = height
	v.drawableSizeDirty = true
}

// applyDrawableSizeIfNeeded applies the drawable size recorded at setDrawableSize to the layer.
func (v *view) applyDrawableSizeIfNeeded() {
	if !v.drawableSizeDirty {
		return
	}

	v.drawableSizeDirty = false

	// While the window is being resized by the user, update the property on the main thread, so
	// that the visual side effects of the property change are applied in the same Core Animation
	// transaction as the window bounds change and the drawable presentation. A property change
	// on the rendering thread belongs to that thread's implicit transaction, which is committed
	// at an undefined time and can blank the layer for a moment (#3478).
	set := func() {
		v.ml.SetDrawableSize(v.drawableWidth, v.drawableHeight)
	}
	if v.runOnMainThread != nil && v.inLiveResize() {
		v.runOnMainThread(set)
	} else {
		set()
	}
}

// inLiveResize reports whether the window is being resized by the user.
func (v *view) inLiveResize() bool {
	if v.window == 0 {
		return false
	}
	return cocoa.NSWindow{ID: objc.ID(v.window)}.InLiveResize()
}

// isFullscreen reports whether the window is in the fullscreen mode.
func (v *view) isFullscreen() bool {
	if v.window == 0 {
		return false
	}
	return cocoa.NSWindow{ID: objc.ID(v.window)}.StyleMask()&cocoa.NSWindowStyleMaskFullScreen != 0
}

// shouldPresentWithTransaction reports whether a drawable must be presented in sync with
// Core Animation transactions. The state is updated at updatePresentationState.
func (v *view) shouldPresentWithTransaction() bool {
	return v.presentsWithTransaction
}

// presentDrawableWithTransaction presents the drawable in sync with Core Animation transactions.
// This must be called after the command buffer is committed.
func (v *view) presentDrawableWithTransaction(cb mtl.CommandBuffer, d ca.MetalDrawable) {
	v.lastPresentTime = time.Now()

	// While the window is being resized by the user, present on the main thread. The window
	// bounds change belongs to the main thread's implicit transaction, and presenting there puts
	// the new layer content in the same transaction, so the new content and the new bounds are
	// applied atomically. This works since the main thread executes posted functions while it is
	// blocked until the frame rendering for the resizing ends.
	if v.liveResizing.Load() && v.runOnMainThread != nil {
		v.runOnMainThread(func() {
			// Wait until the rendering finishes, not just until the command buffer is scheduled.
			// The transaction can be committed before the GPU finishes rendering to the drawable,
			// and then a not-fully-rendered content would be shown for a moment.
			cb.WaitUntilCompleted()
			d.Present()
			// Commit the transaction immediately so that the presented drawable is on the screen
			// before the next frame starts. Otherwise, presenting a drawable frees the previously
			// displayed drawable for reuse before the transaction is committed, and the next
			// frame can render into a drawable that is still on the screen.
			ca.FlushTransaction()
		})
		return
	}

	// https://developer.apple.com/documentation/quartzcore/cametallayer/1478157-presentswithtransaction
	//
	// While presentsWithTransaction is enabled, wait until the command buffer is scheduled and
	// then present the drawable directly.
	cb.WaitUntilScheduled()
	d.Present()
	// The rendering thread has no run loop, so an implicit transaction on this thread is not
	// committed automatically. Commit it explicitly.
	ca.FlushTransaction()
}

// presentDrawable registers the drawable presentation on the command buffer.
func (v *view) presentDrawable(cb mtl.CommandBuffer, d ca.MetalDrawable) {
	// Track the number of drawables queued for presentation. While vsync is disabled, nextDrawable
	// uses the count to skip a frame instead of blocking until a drawable is available.
	// updatePresentationState uses the count to wait until all the queued presentations
	// finish before switching to transaction-synced presentation.
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
	v.lastPresentTime = time.Now()
	cb.PresentDrawable(d)
}
