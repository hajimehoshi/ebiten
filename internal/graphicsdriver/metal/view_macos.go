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

// #include <CoreVideo/CVDisplayLink.h>
//
// int ebitengine_DisplayLinkOutputCallback(CVDisplayLinkRef displayLinkRef, CVTimeStamp inNow, CVTimeStamp inOutputTime, uint64_t flagsIn, uint64_t* flagsOut, void* displayLinkContext);
import "C"

import (
	"runtime/cgo"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
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
	// Always use 3 for macOS (#2880, #2883, #3278).
	// At least, this should work with MacBook Pro 2020 (Intel) and MacBook Pro 2023 (M3).
	return 3
}

func (v *view) initializeDisplayLink() {
	v.fenceCond = sync.NewCond(&sync.Mutex{})

	// TODO: CVDisplayLink APIs are deprecated in macOS 10.15 and later.
	// Use new APIs like NSView.displayLink(target:selector:).
	var displayLinkRef C.CVDisplayLinkRef
	if ret := C.CVDisplayLinkCreateWithActiveCGDisplays(&displayLinkRef); ret != kCVReturnSuccess {
		// Failed to get the display link, so proceed without it.
		return
	}
	v.handleToSelf = cgo.NewHandle(v)
	C.CVDisplayLinkSetOutputCallback(displayLinkRef, C.CVDisplayLinkOutputCallback(C.ebitengine_DisplayLinkOutputCallback), unsafe.Pointer(&v.handleToSelf))
	C.CVDisplayLinkStart(displayLinkRef)

	v.displayLink = uintptr(displayLinkRef)
}

func (v *view) waitForDisplayLinkOutputCallback() {
	if v.displayLink == 0 {
		return
	}
	if v.vsyncDisabled {
		// TODO: nextDrawable still waits for the next drawable available, so this should be fixed not to wait.
		return
	}

	v.fenceCond.L.Lock()
	for {
		if f := v.fence.Load(); v.lastFence < f {
			v.lastFence = f
			break
		}
		v.fenceCond.Wait()
	}
	v.fenceCond.L.Unlock()
}

//export ebitengine_DisplayLinkOutputCallback
func ebitengine_DisplayLinkOutputCallback(displayLinkRef C.CVDisplayLinkRef, inNow, inOutputTime C.CVTimeStamp, flagsIn C.uint64_t, flagsOut *C.uint64_t, displayLinkContext unsafe.Pointer) C.int {
	cgoHandle := (*cgo.Handle)(displayLinkContext)
	cgoHandle.Value().(*view).advanceFence()
	return 0
}

func (v *view) advanceFence() {
	v.fence.Add(1)
	v.fenceCond.Signal()
}
