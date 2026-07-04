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
	sel_processInfo            = objc.RegisterName("processInfo")
	sel_operatingSystemVersion = objc.RegisterName("operatingSystemVersion")

	class_NSProcessInfo = objc.GetClass("NSProcessInfo")
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
	version := objc.Send[nsOperatingSystemVersion](objc.ID(class_NSProcessInfo).Send(sel_processInfo), sel_operatingSystemVersion)
	if version.majorVersion >= 14 {
		return nsClassFromString(cocoa.NSString_alloc().InitWithUTF8String("CAMetalDisplayLink")) != 0
	}
	return false
}

var class_EbitengineCAMetalDisplayLinkDelegate objc.Class

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
					// vsyncDisabled or liveResizing becomes true before the display link is invalidated
					// (see updateMetalDisplayLink).
					// Return without sending a drawable so that the run loop can execute the invalidation block.
					if v.vsyncDisabled.Load() || v.liveResizing.Load() {
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
	class_EbitengineCAMetalDisplayLinkDelegate = c

	v.updateMetalDisplayLink()

	return nil
}

// updateMetalDisplayLink creates or destroys CAMetalDisplayLink for the current vsync and
// live-resize states.
func (v *view) updateMetalDisplayLink() {
	// A zero run loop means CAMetalDisplayLink is not used: either the OS doesn't support it,
	// or the display link is not initialized yet. In the latter case, initCAMetalDisplayLink
	// calls this function after creating the run loop.
	if v.metalDisplayLinkRunLoop.ID == 0 {
		return
	}

	// Destroy the display link while vsync is disabled or the window is being resized.
	// CAMetalLayer's nextDrawable is not available while CAMetalDisplayLink exists for the
	// layer, and drawables must be obtained directly from the Metal layer in these cases
	// (see nextDrawable):
	//   - While vsync is disabled, getting a drawable must not wait for the display refresh.
	//   - While the window is being resized, the display link creates a drawable before
	//     invoking the delegate callback, so the drawable often has a stale size while the
	//     drawable size keeps changing, and waiting for a drawable with the correct size
	//     wastes the tight drawable pool. A drawable obtained directly from the Metal layer
	//     always has the current drawable size (#3478).
	if v.vsyncDisabled.Load() || v.liveResizing.Load() {
		if v.metalDisplayLink == 0 {
			return
		}
		dl := ca.MetalDisplayLink{ID: objc.ID(v.metalDisplayLink)}
		v.metalDisplayLink = 0

		// If a drawable from the display link is still in use, the delegate callback that delivered it is
		// blocked until the drawable usage finishes. Unblock the callback. The drawable remains usable and
		// presentable.
		if v.drawableFromDisplayLink {
			v.drawableFromDisplayLink = false
			v.drawableDoneCh <- struct{}{}
		}

		done := make(chan struct{})
		v.metalDisplayLinkRunLoop.PerformBlock(objc.NewBlock(func(block objc.Block) {
			dl.Invalidate()
			dl.Release()
			close(done)
		}))

		// A delegate callback might be blocked to send a drawable, preventing the run loop from executing
		// the block above. Receive drawables until the display link is invalidated.
		// New delegate callbacks return without sending a drawable as vsyncDisabled or liveResizing is
		// already true, so this loop always terminates.
	loop:
		for {
			select {
			case <-v.drawableCh:
				v.drawableDoneCh <- struct{}{}
			case <-done:
				break loop
			}
		}
		return
	}

	// Create the display link while vsync is enabled and the window is not being resized.
	if v.metalDisplayLink != 0 {
		return
	}

	if v.metalDisplayLinkDelegate == 0 {
		v.metalDisplayLinkDelegate = objc.ID(class_EbitengineCAMetalDisplayLinkDelegate).Send(objc.RegisterName("new"))
	}

	ch := make(chan uintptr)
	v.metalDisplayLinkRunLoop.PerformBlock(objc.NewBlock(func(block objc.Block) {
		dl := ca.NewMetalDisplayLink(v.ml)
		dl.SetDelegate(v.metalDisplayLinkDelegate)
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

// updatePresentationState synchronizes the presentation-related states with the window.
// This must be called at the beginning of a frame.
func (v *view) updatePresentationState() {
	inLiveResize := v.inLiveResize()
	if v.liveResizing.Load() != inLiveResize {
		v.liveResizing.Store(inLiveResize)
		v.updateMetalDisplayLink()
	}

	// presentsWithTransaction is enabled unless vsync is disabled (#1196) or the window is in
	// the fullscreen mode (#1745, #1974). Toggling presentsWithTransaction drops the current
	// layer content immediately and irrecoverably, showing the window background for a moment
	// (#3478). The property must not be toggled at the beginning of window resizing, so it is
	// kept enabled during the entire windowed lifetime with vsync on instead.
	//
	// While vsync is disabled, presentsWithTransaction is enabled only during window resizing:
	// frames are event-paced then, so the unlimited presentation rate does not matter, and the
	// synced presentations keep the content from being distorted. The toggle at the beginning
	// of resizing can show the window background for a moment, which is less noticeable than
	// the content being distorted during the entire resizing.
	presentsWithTransaction := !v.isFullscreen() && (!v.vsyncDisabled.Load() || inLiveResize)
	if v.presentsWithTransaction != presentsWithTransaction {
		if presentsWithTransaction {
			// Wait until all the drawables queued for asynchronous presentation are presented.
			// Otherwise, a pending presentation can replace the layer content after the first
			// transaction-synced presentation, and a stale frame is shown for a moment.
			for start := time.Now(); v.queuedPresents.Load() > 0 && time.Since(start) < 100*time.Millisecond; {
				time.Sleep(time.Millisecond)
			}
		}
		set := func() {
			v.ml.SetPresentsWithTransaction(presentsWithTransaction)
		}
		if v.runOnMainThread != nil {
			v.runOnMainThread(set)
		} else {
			set()
		}
		v.presentsWithTransaction = presentsWithTransaction
	}
}

func (v *view) nextDrawable() ca.MetalDrawable {
	v.applyDrawableSizeIfNeeded()

	if v.metalDisplayLink != 0 {
		const wait = 100 * time.Millisecond
		if v.drawableTimer == nil {
			v.drawableTimer = time.NewTimer(wait)
		} else {
			v.drawableTimer.Reset(wait)
		}
		defer v.drawableTimer.Stop()
		for {
			select {
			case d := <-v.drawableCh:
				v.drawableFromDisplayLink = true
				// The display link creates a drawable before invoking the delegate callback, so the
				// drawable might have a stale size when the drawable size has just been changed e.g.
				// by window resizing. Skip such a drawable and wait for one with the new size (#3478).
				if v.drawableWidth != 0 && v.drawableHeight != 0 {
					if t := d.Texture(); t.Width() != v.drawableWidth || t.Height() != v.drawableHeight {
						v.finishDrawableUsage()
						continue
					}
				}
				return d
			case <-v.drawableTimer.C:
				// This happens when the main thread needs to execute the notification observer callback,
				// or when the appliation goes to full screen (#3354).
				return ca.MetalDrawable{}
			}
		}
	}

	// While vsync is disabled, getting a drawable must not block until one is available.
	// When all the drawables but the one on the display are queued for presentation,
	// skip the frame instead of blocking. The time condition is a fallback to keep presenting
	// even when the tracking is stuck e.g. by a presented handler not being called.
	if v.vsyncDisabled.Load() &&
		v.queuedPresents.Load() >= maximumDrawableCount-1 &&
		time.Since(v.lastPresentTime) < time.Second/4 {
		return ca.MetalDrawable{}
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
	if !v.drawableFromDisplayLink {
		return
	}
	v.drawableFromDisplayLink = false
	v.drawableDoneCh <- struct{}{}
}
