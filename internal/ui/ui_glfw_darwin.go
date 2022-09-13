// Copyright 2016 Hajime Hoshi
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

//go:build !ios && !nintendosdk
// +build !ios,!nintendosdk

package ui

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework AppKit
//
// #import <AppKit/AppKit.h>
//
// @interface EbitengineWindowDelegate : NSObject <NSWindowDelegate>
// @end
//
// @implementation EbitengineWindowDelegate {
//   id<NSWindowDelegate> origDelegate_;
//   bool origResizable_;
// }
//
// - (instancetype)initWithOrigDelegate:(id<NSWindowDelegate>)origDelegate {
//   self = [super init];
//   if (self != nil) {
//     origDelegate_ = origDelegate;
//   }
//   return self;
// }
//
// // The method set of origDelegate_ must sync with GLFWWindowDelegate's implementation.
// // See cocoa_window.m in GLFW.
// - (BOOL)windowShouldClose:(id)sender {
//   return [origDelegate_ windowShouldClose:sender];
// }
// - (void)windowDidResize:(NSNotification *)notification {
//   [origDelegate_ windowDidResize:notification];
// }
// - (void)windowDidMove:(NSNotification *)notification {
//   [origDelegate_ windowDidMove:notification];
// }
// - (void)windowDidMiniaturize:(NSNotification *)notification {
//   [origDelegate_ windowDidMiniaturize:notification];
// }
// - (void)windowDidDeminiaturize:(NSNotification *)notification {
//   [origDelegate_ windowDidDeminiaturize:notification];
// }
// - (void)windowDidBecomeKey:(NSNotification *)notification {
//   [origDelegate_ windowDidBecomeKey:notification];
// }
// - (void)windowDidResignKey:(NSNotification *)notification {
//   [origDelegate_ windowDidResignKey:notification];
// }
// - (void)windowDidChangeOcclusionState:(NSNotification* )notification {
//   [origDelegate_ windowDidChangeOcclusionState:notification];
// }
//
// - (void)pushResizableState:(NSWindow*)window {
//   origResizable_ = window.styleMask & NSWindowStyleMaskResizable;
//   if (!origResizable_) {
//     window.styleMask |= NSWindowStyleMaskResizable;
//   }
// }
//
// - (void)popResizableState:(NSWindow*)window {
//   if (!origResizable_) {
//     window.styleMask &= ~NSWindowStyleMaskResizable;
//   }
//   origResizable_ = false;
// }
//
// - (void)windowWillEnterFullScreen:(NSNotification *)notification {
//   NSWindow* window = (NSWindow*)[notification object];
//   [self pushResizableState:window];
// }
//
// - (void)windowDidEnterFullScreen:(NSNotification *)notification {
//   NSWindow* window = (NSWindow*)[notification object];
//   [self popResizableState:window];
// }
//
// - (void)windowWillExitFullScreen:(NSNotification *)notification {
//   NSWindow* window = (NSWindow*)[notification object];
//   [self pushResizableState:window];
// }
//
// - (void)windowDidExitFullScreen:(NSNotification *)notification {
//   NSWindow* window = (NSWindow*)[notification object];
//   [self popResizableState:window];
//   // Do not call setFrame here (#2295). setFrame here causes unexpected results.
// }
//
// @end
//
import "C"

import (
	"fmt"
	"github.com/ebitengine/purego/objc"
	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

var EbitengineWindowDelegate objc.Class

func init() {
	//EbitengineWindowDelegate = objc.AllocateClassPair(objc.GetClass("NSObject"), "EbitengineWindowDelegate", 0)
	//EbitengineWindowDelegate.Register()

}

type graphicsDriverCreatorImpl struct {
	transparent bool
}

func (g *graphicsDriverCreatorImpl) newAuto() (graphicsdriver.Graphics, GraphicsLibrary, error) {
	m, err1 := g.newMetal()
	if err1 == nil {
		return m, GraphicsLibraryMetal, nil
	}
	o, err2 := g.newOpenGL()
	if err2 == nil {
		return o, GraphicsLibraryOpenGL, nil
	}
	return nil, GraphicsLibraryUnknown, fmt.Errorf("ui: failed to choose graphics drivers: Metal: %v, OpenGL: %v", err1, err2)
}

func (*graphicsDriverCreatorImpl) newOpenGL() (graphicsdriver.Graphics, error) {
	return opengl.NewGraphics()
}

func (*graphicsDriverCreatorImpl) newDirectX() (graphicsdriver.Graphics, error) {
	return nil, nil
}

func (*graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return metal.NewGraphics()
}

// clearVideoModeScaleCache must be called from the main thread.
func clearVideoModeScaleCache() {}

// dipFromGLFWMonitorPixel must be called from the main thread.
func (u *userInterfaceImpl) dipFromGLFWMonitorPixel(x float64, monitor *glfw.Monitor) float64 {
	return x
}

// dipFromGLFWPixel must be called from the main thread.
func (u *userInterfaceImpl) dipFromGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	// NOTE: On macOS, GLFW exposes the device independent coordinate system.
	// Thus, the conversion functions are unnecessary,
	// however we still need the deviceScaleFactor internally
	// so we can create and maintain a HiDPI frame buffer.
	return x
}

// dipToGLFWPixel must be called from the main thread.
func (u *userInterfaceImpl) dipToGLFWPixel(x float64, monitor *glfw.Monitor) float64 {
	return x
}

func (u *userInterfaceImpl) adjustWindowPosition(x, y int, monitor *glfw.Monitor) (int, int) {
	return x, y
}

func flipY(y int) int {
	for _, m := range ensureMonitors() {
		if m.x == 0 && m.y == 0 {
			y = -y
			y += m.vm.Height
			break
		}
	}
	return y
}

var class_NSEvent = objc.GetClass("NSEvent")
var sel_mouseLocation = objc.RegisterName("mouseLocation")

func currentMouseLocation() (x, y int) {
	sig := cocoa.NSMethodSignature_signatureWithObjCTypes("{NSPoint=dd}@:")
	inv := cocoa.NSInvocation_invocationWithMethodSignature(sig)
	inv.SetTarget(objc.ID(class_NSEvent))
	inv.SetSelector(sel_mouseLocation)
	inv.Invoke()
	var point cocoa.NSPoint
	inv.GetReturnValue(unsafe.Pointer(&point))
	return int(point.X), int(point.Y)
}

func initialMonitorByOS() (*glfw.Monitor, error) {
	var cx, cy = currentMouseLocation()
	x, y := int(cx), flipY(int(cy))

	// Find the monitor including the cursor.
	for _, m := range ensureMonitors() {
		w, h := m.vm.Width, m.vm.Height
		if x >= m.x && x < m.x+w && y >= m.y && y < m.y+h {
			return m.m, nil
		}
	}

	return nil, nil
}

func monitorFromWindowByOS(w *glfw.Window) *glfw.Monitor {
	var x, y C.int
	window := cocoa.NSWindow{ID: objc.ID(w.GetCocoaWindow())}
	pool := cocoa.NSAutoreleasePool_new()
	screen := cocoa.NSScreen_mainScreen()
	if window.ID != 0 && window.IsVisibile() {
		// When the window is visible, the window is already initialized.
		// [NSScreen mainScreen] sometimes tells a lie when the window is put across monitors (#703).
		screen = window.Screen()
	}
	screenDictionary := screen.DeviceDescription()
	screenID := cocoa.NSNumber{ID: screenDictionary.ObjectForKey(cocoa.NSString_alloc().InitWithUTF8String("NSScreenNumber").ID)}
	aID := screenID.UnsignedIntValue() //CGDirectDisplayID
	var bounds C.CGRect = C.CGDisplayBounds(C.CGDirectDisplayID(aID))
	x = C.int(bounds.origin.x)
	y = C.int(bounds.origin.y)
	pool.Release()
	for _, m := range ensureMonitors() {
		if int(x) == m.x && int(y) == m.y {
			return m.m
		}
	}
	return nil
}

func (u *userInterfaceImpl) nativeWindow() uintptr {
	return u.window.GetCocoaWindow()
}

func (u *userInterfaceImpl) isNativeFullscreen() bool {
	return cocoa.NSWindow{ID: objc.ID(u.window.GetCocoaWindow())}.StyleMask()&cocoa.NSWindowStyleMaskFullScreen != 0
}

func (u *userInterfaceImpl) setNativeCursor(shape CursorShape) {
	class_NSCursor := objc.GetClass("NSCursor")
	NSCursor := objc.ID(class_NSCursor).Send(objc.RegisterName("class"))
	var sel_performSelector = objc.RegisterName("performSelector:")
	cursor := NSCursor.Send(sel_performSelector, objc.RegisterName("arrowCursor"))
	switch shape {
	case 0:
		cursor = NSCursor.Send(sel_performSelector, objc.RegisterName("arrowCursor"))
	case 1:
		cursor = NSCursor.Send(sel_performSelector, objc.RegisterName("IBeamCursor"))
	case 2:
		cursor = NSCursor.Send(sel_performSelector, objc.RegisterName("crosshairCursor"))
	case 3:
		cursor = NSCursor.Send(sel_performSelector, objc.RegisterName("pointHandCursor"))
	case 4:
		cursor = NSCursor.Send(sel_performSelector, objc.RegisterName("_windowResizeEastWestCursor"))
	case 5:
		cursor = NSCursor.Send(sel_performSelector, objc.RegisterName("_windowResizeNorthSouthCursor"))
	}
	cursor.Send(objc.RegisterName("push"))
}

func (u *userInterfaceImpl) isNativeFullscreenAvailable() bool {
	// TODO: If the window is transparent, we should use GLFW's windowed fullscreen (#1822, #1857).
	// However, if the user clicks the green button, should this window be in native fullscreen mode?
	return true
}

var sel_collectionBehavior = objc.RegisterName("collectionBehavior")
var sel_setCollectionBehavior = objc.RegisterName("setCollectionBehavior:")

func (u *userInterfaceImpl) setNativeFullscreen(fullscreen bool) {
	// Toggling fullscreen might ignore events like keyUp. Ensure that events are fired.
	glfw.WaitEventsTimeout(0.1)
	var window = cocoa.NSWindow{ID: objc.ID(u.window.GetCocoaWindow())}
	if window.StyleMask()&cocoa.NSWindowStyleMaskFullScreen != 0 == fullscreen {
		return
	}
	// Even though EbitengineWindowDelegate is used, this hack is still required.
	// toggleFullscreen doesn't work when the window is not resizable.
	var origFullScreen = window.Send(sel_collectionBehavior)&cocoa.NSWindowCollectionBehaviorFullScreenPrimary != 0
	if !origFullScreen {
		window.Send(sel_setCollectionBehavior, window.Send(sel_collectionBehavior)|cocoa.NSWindowCollectionBehaviorFullScreenPrimary)
	}
	window.Send(objc.RegisterName("toggleFullScreen:"), 0)
	if !origFullScreen {
		window.Send(sel_setCollectionBehavior, window.Send(sel_collectionBehavior)&cocoa.NSWindowCollectionBehaviorFullScreenPrimary)
	}
}

func (u *userInterfaceImpl) adjustViewSize() {
	if u.graphicsDriver.IsGL() {
		return
	}
	window := cocoa.NSWindow{ID: objc.ID(u.window.GetCocoaWindow())}
	if window.StyleMask()&cocoa.NSWindowStyleMaskFullScreen == 0 {
		return
	}
	// Apparently, adjusting the view size is not needed as of macOS 12 (#1745).
	if cocoa.NSProcessInfo_processInfo().IsOperatingSystemAtLeastVersion(cocoa.NSOperatingSystemVersion{
		Major: 12,
	}) {
		return
	}

	// Reduce the view height (#1745).
	// https://stackoverflow.com/questions/27758027/sprite-kit-serious-fps-issue-in-full-screen-mode-on-os-x
	windowSize := window.Frame().Size
	view := window.ContentView()
	viewSize := view.Frame().Size
	if windowSize.Width != viewSize.Width || windowSize.Height != viewSize.Height {
		return
	}
	viewSize.Width--
	view.SetFrameSize(viewSize)
	// NSColor.blackColor (0, 0, 0, 1) didn't work.
	// Use the transparent color instead.
	window.SetBackgroundColor(cocoa.NSColor_colorWithSRGBRedGreenBlueAlpha(0, 0, 0, 0))
}

func (u *userInterfaceImpl) setWindowResizingModeForOS(mode WindowResizingMode) {
	allowFullscreen := mode == WindowResizingModeOnlyFullscreenEnabled ||
		mode == WindowResizingModeEnabled
	var collectionBehavior = int(objc.ID(u.window.GetCocoaWindow()).Send(sel_collectionBehavior))
	if allowFullscreen {
		collectionBehavior |= cocoa.NSWindowCollectionBehaviorFullScreenPrimary
	} else {
		collectionBehavior &= ^cocoa.NSWindowCollectionBehaviorFullScreenPrimary
	}
	objc.ID(u.window.GetCocoaWindow()).Send(objc.RegisterName("setCollectionBehavior:"), collectionBehavior)
}

//	static void initializeWindow(uintptr_t windowPtr) {
//	  NSWindow* window = (NSWindow*)windowPtr;
//	  // This delegate is never released. This assumes that the window lives until the process lives.
//	  window.delegate = [[EbitengineWindowDelegate alloc] initWithOrigDelegate:window.delegate];
//	}
func initializeWindowAfterCreation(w *glfw.Window) {
	// TODO: Register NSWindowWillEnterFullScreenNotification and so on.
	// Enable resizing temporary before making the window fullscreen.
	var nswindow = objc.ID(w.GetCocoaWindow())
	var class_EbitengineWindowDelegate = objc.GetClass("EbitengineWindowDelegate")
	var sel_delegate = objc.RegisterName("delegate")
	var delegate = objc.ID(class_EbitengineWindowDelegate).Send(objc.RegisterName("alloc")).Send(objc.RegisterName("initWithOrigDelegate:"), nswindow.Send(sel_delegate))
	nswindow.Send(objc.RegisterName("setDelegate:"), delegate)
}
