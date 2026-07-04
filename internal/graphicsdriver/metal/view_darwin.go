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

package metal

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

// maximumDrawableCount is the maximum number of drawable objects.
//
// Always use 3 for macOS (#2880, #2883, #3278).
// At least, this should work with MacBook Pro 2020 (Intel) and MacBook Pro 2023 (M3).
const maximumDrawableCount = 3

type view struct {
	window uintptr
	uiview uintptr

	windowChanged bool

	// vsyncDisabled reports whether vsync is disabled.
	// This can be read from the thread for CAMetalDisplayLink, while the other members are used only on the rendering thread.
	vsyncDisabled atomic.Bool

	// liveResizing reports whether the window is being resized by the user on macOS.
	// This is written on the rendering thread, and can be read from the thread for CAMetalDisplayLink.
	// This is always false on iOS.
	liveResizing atomic.Bool

	device mtl.Device
	ml     ca.MetalLayer

	once sync.Once

	caDisplayLink    uintptr
	metalDisplayLink uintptr

	// queuedPresents is the number of drawables that are queued for presentation and not presented yet.
	// While vsync is disabled on macOS, this is used to skip a frame instead of blocking until a
	// drawable is available (see nextDrawable). This is also used to wait for all the queued
	// presentations before switching to transaction-synced presentation (see updatePresentsWithTransaction).
	// queuedPresents is incremented on the rendering thread and decremented on Metal's presentation thread.
	queuedPresents atomic.Int32

	// presentedHandler is the handler called when a drawable is presented.
	// presentedHandler is created at most once and reused for all the drawables.
	// A zero presentedHandler means the handler is not created yet or not available.
	presentedHandler objc.Block

	// presentedHandlerOnce guards the creation of presentedHandler.
	presentedHandlerOnce sync.Once

	// lastPresentTime is the last time when a drawable presentation was registered on a command buffer.
	lastPresentTime time.Time

	// presentsWithTransaction is the last value set for the layer's presentsWithTransaction property.
	// This is true only while the window is being resized by the user on macOS, and always false on iOS.
	presentsWithTransaction bool

	// drawableWidth and drawableHeight are the last drawable size set at setDrawableSize.
	// These are used only on the rendering thread, and only on macOS.
	drawableWidth  int
	drawableHeight int

	// drawableSizeDirty reports whether the drawable size recorded at setDrawableSize is not
	// applied to the layer yet. This is used only on the rendering thread, and only on macOS.
	drawableSizeDirty bool

	// runOnMainThread runs a function on the main thread synchronously.
	// This is set at most once at the initialization on macOS, and nil on iOS.
	runOnMainThread func(f func())

	// The following members are used only with CAMetalDisplayLink.
	drawableCh               chan ca.MetalDrawable
	drawableDoneCh           chan struct{}
	drawableTimer            *time.Timer
	drawableFromDisplayLink  bool
	metalDisplayLinkRunLoop  cocoa.NSRunLoop
	metalDisplayLinkDelegate objc.ID

	// The following members are used only with CADisplayLink.
	handleToSelf viewHandle
	fence        *fence
}

func (v *view) getMTLDevice() mtl.Device {
	return v.device
}

func (v *view) setDisplaySyncEnabled(enabled bool) {
	if !v.vsyncDisabled.Load() == enabled {
		return
	}
	v.forceSetDisplaySyncEnabled(enabled)
}

func (v *view) forceSetDisplaySyncEnabled(enabled bool) {
	v.ml.SetDisplaySyncEnabled(enabled)
	v.vsyncDisabled.Store(!enabled)
	v.updateMetalDisplayLink()
}

func (v *view) colorPixelFormat() mtl.PixelFormat {
	return v.ml.PixelFormat()
}

func (v *view) initialize(device mtl.Device, colorSpace color.ColorSpace) error {
	v.device = device

	ml, err := ca.NewMetalLayer(colorSpace)
	if err != nil {
		return fmt.Errorf("metal: ca.NewMetalLayer failed: %w", err)
	}
	v.ml = ml
	v.ml.SetDevice(v.device)
	// https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat
	//
	// The pixel format for a Metal layer must be MTLPixelFormatBGRA8Unorm,
	// MTLPixelFormatBGRA8Unorm_sRGB, MTLPixelFormatRGBA16Float, MTLPixelFormatBGRA10_XR, or
	// MTLPixelFormatBGRA10_XR_sRGB.
	v.ml.SetPixelFormat(mtl.PixelFormatBGRA8UNorm)

	// The vsync state might be reset. Set the state again (#1364).
	v.forceSetDisplaySyncEnabled(!v.vsyncDisabled.Load())
	v.ml.SetFramebufferOnly(true)

	// presentsWithTransaction is disabled at first, and updated at every frame on macOS
	// (see updatePresentationState). This is always disabled on iOS.
	v.ml.SetPresentsWithTransaction(false)

	v.ml.SetMaximumDrawableCount(maximumDrawableCount)

	if err := v.initializeOS(); err != nil {
		return err
	}

	return nil
}

// viewHandle is a cgo-free replacement for cgo.Handle to pass *view through C callbacks.
type viewHandle uintptr

var (
	viewHandleMu      sync.Mutex
	viewHandleMap     = map[viewHandle]*view{}
	viewHandleCounter viewHandle
)

func newViewHandle(v *view) viewHandle {
	viewHandleMu.Lock()
	defer viewHandleMu.Unlock()
	viewHandleCounter++
	h := viewHandleCounter
	viewHandleMap[h] = v
	return h
}

func (h viewHandle) Value() *view {
	viewHandleMu.Lock()
	defer viewHandleMu.Unlock()
	return viewHandleMap[h]
}

type fence struct {
	value     uint64
	lastValue uint64
	cond      *sync.Cond
}

func newFence() *fence {
	return &fence{
		cond: sync.NewCond(&sync.Mutex{}),
	}
}

func (f *fence) wait() {
	f.cond.L.Lock()
	defer f.cond.L.Unlock()
	for f.lastValue >= f.value {
		f.cond.Wait()
	}
	f.lastValue = f.value
}

func (f *fence) advance() {
	f.cond.L.Lock()
	defer f.cond.L.Unlock()
	f.value++
	f.cond.Broadcast()
}
