// Copyright 2025 The Ebitengine Authors
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

// #cgo CFLAGS: -x objective-c
//
// #include <Foundation/Foundation.h>
// #include <CoreVideo/CVDisplayLink.h>
// #if __has_include(<QuartzCore/CAMetalLayer.h>)
//   #include <QuartzCore/CAMetalLayer.h>
// #endif
//
// static bool isCAMetalDisplayLinkAvailable() {
//   // TODO: Use PureGo if returning a struct is supported (ebitengine/purego#225).
//   // As operatingSystemVersion returns a struct, this cannot be written with PureGo.
//   NSOperatingSystemVersion version = [[NSProcessInfo processInfo] operatingSystemVersion];
//   if (version.majorVersion >= 14) {
//     // Also check if the CAMetalDisplayLink class exists
//     return NSClassFromString(@"CAMetalDisplayLink") != nil;
//   }
//   return false;
// }
//
// int ebitengine_DisplayLinkOutputCallback(CVDisplayLinkRef displayLinkRef, CVTimeStamp* inNow, CVTimeStamp* inOutputTime, uint64_t flagsIn, uint64_t* flagsOut, void* displayLinkContext);
import "C"
import (
	"runtime/cgo"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
)

func (v *view) initDisplayLink() error {
	if C.isCAMetalDisplayLinkAvailable() {
		if err := v.initCAMetalDisplayLink(); err != nil {
			return err
		}
		return nil
	}
	if err := v.initCADisplayLink(); err != nil {
		return err
	}
	return nil
}

var class_EbitengineCAMetalDisplayLinkDelegate objc.Class

func (v *view) initCAMetalDisplayLink() error {
	v.drawableCh = make(chan ca.MetalDrawable)

	displayLink := ca.NewMetalDisplayLink(v.ml)

	c, err := objc.RegisterClass(
		"EbitengineCAMetalDisplayLinkDelegate",
		objc.GetClass("NSObject"),
		[]*objc.Protocol{objc.GetProtocol("CAMetalDisplayLinkDelegate")},
		nil,
		[]objc.MethodDef{
			{
				Cmd: objc.RegisterName("metalDisplayLink:needsUpdate:"),
				Fn: func(id objc.ID, cmd objc.SEL, metalDisplayLink objc.ID, needsUpdate objc.ID) {
					println("delegate is invoked")
					v.drawableCh <- ca.MetalDisplayLinkUpdate{needsUpdate}.Drawable()
				},
			},
		},
	)
	if err != nil {
		return err
	}
	class_EbitengineCAMetalDisplayLinkDelegate = c

	displayLink.SetDelegate(objc.ID(class_EbitengineCAMetalDisplayLinkDelegate).Send(objc.RegisterName("new")))
	displayLink.AddToRunLoop(cocoa.NSRunLoop_mainRunLoop(), cocoa.NSRunLoopCommonModes)

	v.metalDisplayLink = uintptr(displayLink.ID)

	return nil
}

func (v *view) initCADisplayLink() error {
	v.fence = newFence()

	// TODO: CVDisplayLink APIs are deprecated in macOS 10.15 and later.
	// Use new APIs like NSView.displayLink(target:selector:).
	var displayLinkRef C.CVDisplayLinkRef
	if ret := C.CVDisplayLinkCreateWithActiveCGDisplays(&displayLinkRef); ret != kCVReturnSuccess {
		// Failed to get the display link, so proceed without it.
		return nil
	}
	v.handleToSelf = cgo.NewHandle(v)
	C.CVDisplayLinkSetOutputCallback(displayLinkRef, C.CVDisplayLinkOutputCallback(C.ebitengine_DisplayLinkOutputCallback), unsafe.Pointer(&v.handleToSelf))
	C.CVDisplayLinkStart(displayLinkRef)

	v.caDisplayLink = uintptr(displayLinkRef)
	return nil
}

//export ebitengine_DisplayLinkOutputCallback
func ebitengine_DisplayLinkOutputCallback(displayLinkRef C.CVDisplayLinkRef, inNow, inOutputTime *C.CVTimeStamp, flagsIn C.uint64_t, flagsOut *C.uint64_t, displayLinkContext unsafe.Pointer) C.int {
	cgoHandle := (*cgo.Handle)(displayLinkContext)
	view := cgoHandle.Value().(*view)
	view.fence.advance()
	return 0
}

func (v *view) nextDrawable() ca.MetalDrawable {
	if v.metalDisplayLink != 0 {
		return <-v.drawableCh
	}

	v.waitForDisplayLinkOutputCallback()

	d, err := v.ml.NextDrawable()
	if err != nil {
		// Drawable is nil. This can happen at the initial state. Let's wait and see.
		return ca.MetalDrawable{}
	}
	return d
}
