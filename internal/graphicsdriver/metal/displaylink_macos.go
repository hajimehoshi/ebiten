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

import (
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
)

func (v *view) initDisplayLink() error {
	if isCAMetalDisplayLinkAvailable() {
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

var (
	selProcessInfo            = objc.RegisterName("processInfo")
	selOperatingSystemVersion = objc.RegisterName("operatingSystemVersion")

	classNSProcessInfo = objc.GetClass("NSProcessInfo")
)

type nsOperatingSystemVersion struct {
	majorVersion int
	minorVersion int
	patchVersion int
}

var nsClassFromString func(str cocoa.NSString) objc.Class

var (
	cvDisplayLinkCreateWithActiveCGDisplays func(displayLinkOut *uintptr) int32
	cvDisplayLinkSetOutputCallback          func(displayLink uintptr, callback uintptr, userInfo uintptr) int32
	cvDisplayLinkStart                      func(displayLink uintptr) int32
)

func init() {
	foundation, err := purego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&nsClassFromString, foundation, "NSClassFromString")

	coreVideo, err := purego.Dlopen("/System/Library/Frameworks/CoreVideo.framework/CoreVideo", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&cvDisplayLinkCreateWithActiveCGDisplays, coreVideo, "CVDisplayLinkCreateWithActiveCGDisplays")
	purego.RegisterLibFunc(&cvDisplayLinkSetOutputCallback, coreVideo, "CVDisplayLinkSetOutputCallback")
	purego.RegisterLibFunc(&cvDisplayLinkStart, coreVideo, "CVDisplayLinkStart")
}

func isCAMetalDisplayLinkAvailable() bool {
	version := objc.Send[nsOperatingSystemVersion](objc.ID(classNSProcessInfo).Send(selProcessInfo), selOperatingSystemVersion)
	if version.majorVersion >= 14 {
		return nsClassFromString(cocoa.NSString_alloc().InitWithUTF8String("CAMetalDisplayLink")) != 0
	}
	return false
}

var classEbitengineCAMetalDisplayLinkDelegate objc.Class

func (v *view) initCAMetalDisplayLink() error {
	v.drawableCh = make(chan ca.MetalDrawable)
	v.drawableDoneCh = make(chan struct{})
	v.metalDisplayLinkRunLoop = createThreadWithRunLoop()

	c, err := objc.RegisterClass(
		"EbitengineCAMetalDisplayLinkDelegate",
		objc.GetClass("NSObject"),
		[]*objc.Protocol{objc.GetProtocol("CAMetalDisplayLinkDelegate")},
		nil,
		[]objc.MethodDef{
			{
				Cmd: objc.RegisterName("metalDisplayLink:needsUpdate:"),
				Fn: func(id objc.ID, cmd objc.SEL, metalDisplayLink objc.ID, needsUpdate objc.ID) {
					// There is a case where this callback is invoked from the main run loop (#3353).
					// This is very mysterious, but this causes a deadlock.
					// As a workaround, return this immediately when the current run loop is the main run loop.
					if cocoa.NSRunLoop_currentRunLoop() == cocoa.NSRunLoop_mainRunLoop() {
						slog.Debug("metal: metalDisplayLink:needsUpdate: is unexpectedly called from the main run loop")
						return
					}
					drawable := ca.MetalDisplayLinkUpdate{ID: needsUpdate}.Drawable()
					if drawable == (ca.MetalDrawable{}) {
						return
					}
					v.drawableCh <- drawable
					<-v.drawableDoneCh
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("metal: objc.RegisterClass for EbitengineCAMetalDisplayLinkDelegate failed: %w", err)
	}
	classEbitengineCAMetalDisplayLinkDelegate = c

	v.createCAMetalDisplayLink()

	return nil
}

func (v *view) createCAMetalDisplayLink() {
	ch := make(chan uintptr)
	v.metalDisplayLinkRunLoop.PerformBlock(objc.NewBlock(func(block objc.Block) {
		dl := ca.NewMetalDisplayLink(v.ml)
		dl.SetDelegate(objc.ID(classEbitengineCAMetalDisplayLinkDelegate).Send(objc.RegisterName("new")))
		dl.AddToRunLoop(v.metalDisplayLinkRunLoop, cocoa.NSDefaultRunLoopMode)
		dl.SetPaused(false)
		ch <- uintptr(dl.ID)
		close(ch)
	}))
	v.metalDisplayLink = <-ch
}

func createThreadWithRunLoop() cocoa.NSRunLoop {
	ch := make(chan cocoa.NSRunLoop)
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		runLoop := cocoa.NSRunLoop_currentRunLoop()
		ch <- runLoop
		close(ch)

		// Add a dummy mach port to keep alive.
		port := cocoa.NSMachPort_port()
		runLoop.AddPort(port, cocoa.NSRunLoopCommonModes)

		runLoop.Run()
	}()

	runLoop := <-ch
	if runLoop.ID == 0 {
		panic("metal: runLoop must be initialized")
	}
	return runLoop
}

var displayLinkOutputCallbackPtr = purego.NewCallback(displayLinkOutputCallback)

func (v *view) initCADisplayLink() error {
	v.fence = newFence()

	// TODO: CVDisplayLink APIs are deprecated in macOS 10.15 and later.
	// Use new APIs like NSView.displayLink(target:selector:).
	var displayLinkRef uintptr
	if ret := cvDisplayLinkCreateWithActiveCGDisplays(&displayLinkRef); ret != kCVReturnSuccess {
		// Failed to get the display link, so proceed without it.
		return nil
	}
	v.handleToSelf = newViewHandle(v)
	cvDisplayLinkSetOutputCallback(displayLinkRef, displayLinkOutputCallbackPtr, uintptr(v.handleToSelf))
	cvDisplayLinkStart(displayLinkRef)

	v.caDisplayLink = displayLinkRef
	return nil
}

// displayLinkOutputCallback is the callback function for CVDisplayLink.
// The signature matches CVDisplayLinkOutputCallback:
// CVReturn (*CVDisplayLinkOutputCallback)(CVDisplayLinkRef displayLink, const CVTimeStamp *inNow, const CVTimeStamp *inOutputTime, CVOptionFlags flagsIn, CVOptionFlags *flagsOut, void *displayLinkContext)
func displayLinkOutputCallback(displayLink uintptr, inNow, inOutputTime uintptr, flagsIn uint64, flagsOut *uint64, displayLinkContext uintptr) int32 {
	h := viewHandle(displayLinkContext)
	view := h.Value()
	view.fence.advance()
	return 0
}

func (v *view) nextDrawable() ca.MetalDrawable {
	if v.metalDisplayLink != 0 {
		const wait = 100 * time.Millisecond
		if v.drawableTimer == nil {
			v.drawableTimer = time.NewTimer(wait)
		} else {
			v.drawableTimer.Reset(wait)
		}
		defer v.drawableTimer.Stop()
		select {
		case d := <-v.drawableCh:
			return d
		case <-v.drawableTimer.C:
			// This happens when the main thread needs to execute the notification observer callback,
			// or when the appliation goes to full screen (#3354).
			return ca.MetalDrawable{}
		}
	}

	v.waitForDisplayLinkOutputCallback()

	d, err := v.ml.NextDrawable()
	if err != nil {
		// Drawable is nil. This can happen at the initial state. Let's wait and see.
		return ca.MetalDrawable{}
	}
	return d
}

func (v *view) finishDrawableUsage() {
	if v.metalDisplayLink != 0 {
		v.drawableDoneCh <- struct{}{}
		return
	}
}
