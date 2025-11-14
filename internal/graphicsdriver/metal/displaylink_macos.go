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
// #if __has_include(<QuartzCore/CAMetalLayer.h>)
//   #include <QuartzCore/CAMetalLayer.h>
// #endif
//
// #cgo noescape isCAMetalDisplayLinkAvailable
// #cgo nocallback isCAMetalDisplayLinkAvailable
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
import "C"
import (
	"runtime"
	"sync"
	"time"

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

var (
	class_EbitengineCAMetalDisplayLinkDelegate objc.Class
	class_EbitengineCADisplayLinkDelegate      objc.Class
	caDisplayLinkDelegateToView                sync.Map // map[objc.ID]*view
)

func (v *view) initCAMetalDisplayLink() error {
	v.drawableCh = make(chan ca.MetalDrawable)
	v.drawableDoneCh = make(chan struct{})
	v.metalDisplayLinkRunLoop = createThreadWithRunLoop()
	v.prevMetalDisplayLink = make(chan uintptr, 1)

	c, err := objc.RegisterClass(
		"EbitengineCAMetalDisplayLinkDelegate",
		objc.GetClass("NSObject"),
		[]*objc.Protocol{objc.GetProtocol("CAMetalDisplayLinkDelegate")},
		nil,
		[]objc.MethodDef{
			{
				Cmd: objc.RegisterName("metalDisplayLink:needsUpdate:"),
				Fn: func(id objc.ID, cmd objc.SEL, metalDisplayLink objc.ID, needsUpdate objc.ID) {
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
		return err
	}
	class_EbitengineCAMetalDisplayLinkDelegate = c

	v.createCAMetalDisplayLink()

	// Recreate the display link when the app is recovered from sleep.
	// TODO: Recreation might be needed when the display is changed.
	nc := cocoa.NSWorkspace_sharedWorkspace().NotificationCenter()
	if v.meltaDisplayLinkRecreateBlock == 0 {
		v.meltaDisplayLinkRecreateBlock = objc.NewBlock(func(block objc.Block) {
			v.createCAMetalDisplayLink()
		})
	}
	mainQueue := cocoa.NSOperationQueue_mainQueue()
	v.notificatioObserver = nc.AddObserverForName(cocoa.NSWorkspaceDidWakeNotification, 0, mainQueue, v.meltaDisplayLinkRecreateBlock)
	cocoa.NSObject{ID: v.notificatioObserver}.Retain()

	return nil
}

func (v *view) createCAMetalDisplayLink() {
	// Release the previous display link if any.
	// This is done in the thread for the display link, so that the callback is not called during releasing.
	if v.metalDisplayLink != 0 {
		// Unfortunately, there is no blocking 'performBlock' for NSRunLoop, so use a channel to wait.
		if v.metalDisplayLinkReleaseBlock == 0 {
			v.metalDisplayLinkReleaseBlock = objc.NewBlock(func(block objc.Block) {
				dl := ca.MetalDisplayLink{ID: objc.ID(<-v.prevMetalDisplayLink)}
				dl.RemoveFromRunLoop(v.metalDisplayLinkRunLoop, cocoa.NSDefaultRunLoopMode)
				dl.Release()
			})
		}
		v.prevMetalDisplayLink <- v.metalDisplayLink
		v.metalDisplayLinkRunLoop.PerformBlock(v.metalDisplayLinkReleaseBlock)
	}

	dl := ca.NewMetalDisplayLink(v.ml)
	dl.SetDelegate(objc.ID(class_EbitengineCAMetalDisplayLinkDelegate).Send(objc.RegisterName("new")))
	dl.AddToRunLoop(v.metalDisplayLinkRunLoop, cocoa.NSDefaultRunLoopMode)
	dl.SetPaused(false)
	v.metalDisplayLink = uintptr(dl.ID)
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

func (v *view) initCADisplayLink() error {
	v.fence = newFence()

	// Register the delegate class for CADisplayLink callback.
	// The window might not be available yet, so we'll create the display link lazily.
	if class_EbitengineCADisplayLinkDelegate == 0 {
		c, err := objc.RegisterClass(
			"EbitengineCADisplayLinkDelegate",
			objc.GetClass("NSObject"),
			nil,
			nil,
			[]objc.MethodDef{
				{
					Cmd: objc.RegisterName("displayLinkCallback:"),
					Fn: func(id objc.ID, cmd objc.SEL, displayLink objc.ID) {
						// Get the view from the map
						viewPtr, ok := caDisplayLinkDelegateToView.Load(id)
						if !ok {
							return
						}
						view := viewPtr.(*view)
						view.fence.advance()
					},
				},
			},
		)
		if err != nil {
			return err
		}
		class_EbitengineCADisplayLinkDelegate = c
	}

	// Try to create the display link if the window is available.
	// If not, it will be created when the window is set.
	v.createCADisplayLinkIfNeeded()

	return nil
}

func (v *view) createCADisplayLinkIfNeeded() {
	if v.window == 0 {
		// Window not available yet, will be created when window is set.
		return
	}
	if v.caDisplayLink != 0 {
		// Already created.
		return
	}

	cocoaWindow := cocoa.NSWindow{ID: objc.ID(v.window)}
	contentView := cocoaWindow.ContentView()
	if contentView.ID == 0 {
		return
	}

	// Create delegate instance
	delegate := objc.ID(class_EbitengineCADisplayLinkDelegate).Send(objc.RegisterName("new"))
	// Store the view pointer in the map so we can access it in the callback
	caDisplayLinkDelegateToView.Store(delegate, v)

	// Create display link using NSView.displayLink(target:selector:)
	displayLink := contentView.DisplayLink(delegate, objc.RegisterName("displayLinkCallback:"))
	if displayLink.ID == 0 {
		return
	}

	// Add to main run loop
	mainRunLoop := cocoa.NSRunLoop_mainRunLoop()
	displayLink.AddToRunLoop(mainRunLoop, cocoa.NSDefaultRunLoopMode)
	displayLink.SetPaused(false)

	v.caDisplayLink = uintptr(displayLink.ID)
}

func (v *view) nextDrawable() ca.MetalDrawable {
	if v.metalDisplayLink != 0 {
		if v.drawableTimer == nil {
			v.drawableTimer = time.NewTimer(time.Second)
		} else {
			v.drawableTimer.Reset(time.Second)
		}
		defer v.drawableTimer.Stop()
		select {
		case d := <-v.drawableCh:
			return d
		case <-v.drawableTimer.C:
			// This happens when the main thread needs to execute the notification observer callback.
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
