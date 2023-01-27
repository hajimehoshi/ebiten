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

package ui

import (
	"fmt"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

var (
	class_EbitengineWindowDelegate objc.Class
)

type windowDelegate struct {
	isa           objc.Class `objc:"EbitengineWindowDelegate : NSObject <NSWindowDelegate>"`
	origDelegate  objc.ID
	origResizable bool
}

func (w *windowDelegate) pushResizableState(win objc.ID) {
	window := cocoa.NSWindow{ID: win}
	w.origResizable = window.StyleMask()&cocoa.NSWindowStyleMaskResizable != 0
	if !w.origResizable {
		window.SetStyleMask(window.StyleMask() | cocoa.NSWindowStyleMaskResizable)
	}
}

func (w *windowDelegate) popResizableState(win objc.ID) {
	if !w.origResizable {
		window := cocoa.NSWindow{ID: win}
		window.SetStyleMask(window.StyleMask() & ^uint(cocoa.NSWindowStyleMaskResizable))
	}
	w.origResizable = false
}

func (w *windowDelegate) InitWithOrigDelegate(cmd objc.SEL, origDelegate objc.ID) objc.ID {
	self := objc.ID(unsafe.Pointer(w)).SendSuper(sel_init)
	if self != 0 {
		w = *(**windowDelegate)(unsafe.Pointer(&self))
		w.origDelegate = origDelegate
	}
	return self
}

// The method set of origDelegate_ must sync with GLFWWindowDelegate's implementation.
// See cocoa_window.m in GLFW.

func (w *windowDelegate) WindowShouldClose(cmd objc.SEL, notification objc.ID) bool {
	return w.origDelegate.Send(cmd, notification) != 0
}

func (w *windowDelegate) WindowDidResize(cmd objc.SEL, notification objc.ID) {
	w.origDelegate.Send(cmd, notification)
}

func (w *windowDelegate) WindowDidMove(cmd objc.SEL, notification objc.ID) {
	w.origDelegate.Send(cmd, notification)
}

func (w *windowDelegate) WindowDidMiniaturize(cmd objc.SEL, notification objc.ID) {
	w.origDelegate.Send(cmd, notification)
}

func (w *windowDelegate) WindowDidDeminiaturize(cmd objc.SEL, notification objc.ID) {
	w.origDelegate.Send(cmd, notification)
}

func (w *windowDelegate) WindowDidBecomeKey(cmd objc.SEL, notification objc.ID) {
	w.origDelegate.Send(cmd, notification)
}

func (w *windowDelegate) WindowDidResignKey(cmd objc.SEL, notification objc.ID) {
	w.origDelegate.Send(cmd, notification)
}

func (w *windowDelegate) WindowDidChangeOcclusionState(cmd objc.SEL, notification objc.ID) {
	w.origDelegate.Send(cmd, notification)
}

func (w *windowDelegate) WindowWillEnterFullScreen(cmd objc.SEL, notification objc.ID) {
	w.pushResizableState(cocoa.NSNotification{ID: notification}.Object())
}

func (w *windowDelegate) WindowDidEnterFullScreen(cmd objc.SEL, notification objc.ID) {
	w.popResizableState(cocoa.NSNotification{ID: notification}.Object())
}

func (w *windowDelegate) WindowWillExitFullScreen(cmd objc.SEL, notification objc.ID) {
	w.pushResizableState(cocoa.NSNotification{ID: notification}.Object())
}

func (w *windowDelegate) WindowDidExitFullScreen(cmd objc.SEL, notification objc.ID) {
	w.popResizableState(cocoa.NSNotification{ID: notification}.Object())
	// Do not call setFrame here (#2295). setFrame here causes unexpected results.
}

func (w *windowDelegate) Selector(cmd string) objc.SEL {
	switch cmd {
	case "InitWithOrigDelegate":
		return sel_initWithOrigDelegate
	case "WindowShouldClose":
		return sel_windowShouldClose
	case "WindowDidResize":
		return sel_windowDidResize
	case "WindowDidMove":
		return sel_windowDidMove
	case "WindowDidMiniaturize":
		return sel_windowDidMiniaturize
	case "WindowDidDeminiaturize":
		return sel_windowDidDeminiaturize
	case "WindowDidBecomeKey":
		return sel_windowDidBecomeKey
	case "WindowDidResignKey":
		return sel_windowDidResignKey
	case "WindowDidChangeOcclusionState":
		return sel_windowDidChangeOcclusionState
	case "WindowWillEnterFullScreen":
		return sel_windowWillEnterFullScreen
	case "WindowDidEnterFullScreen":
		return sel_windowDidEnterFullScreen
	case "WindowWillExitFullScreen":
		return sel_windowWillExitFullScreen
	case "WindowDidExitFullScreen":
		return sel_windowDidExitFullScreen
	default:
		return 0
	}
}

func init() {
	var err error
	class_EbitengineWindowDelegate, err = objc.RegisterClass(&windowDelegate{})
	if err != nil {
		panic(err)
	}
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

var (
	class_NSCursor = objc.GetClass("NSCursor")
	class_NSEvent  = objc.GetClass("NSEvent")
)

var (
	sel_alloc                         = objc.RegisterName("alloc")
	sel_arrowCursor                   = objc.RegisterName("arrowCursor")
	sel_class                         = objc.RegisterName("class")
	sel_collectionBehavior            = objc.RegisterName("collectionBehavior")
	sel_crosshairCursor               = objc.RegisterName("crosshairCursor")
	sel_delegate                      = objc.RegisterName("delegate")
	sel_IBeamCursor                   = objc.RegisterName("IBeamCursor")
	sel_init                          = objc.RegisterName("init")
	sel_initWithOrigDelegate          = objc.RegisterName("initWithOrigDelegate:")
	sel_mouseLocation                 = objc.RegisterName("mouseLocation")
	sel_performSelector               = objc.RegisterName("performSelector:")
	sel_pointingHandCursor            = objc.RegisterName("pointingHandCursor")
	sel_set                           = objc.RegisterName("set")
	sel_setCollectionBehavior         = objc.RegisterName("setCollectionBehavior:")
	sel_setDelegate                   = objc.RegisterName("setDelegate:")
	sel_toggleFullScreen              = objc.RegisterName("toggleFullScreen:")
	sel_windowDidBecomeKey            = objc.RegisterName("windowDidBecomeKey:")
	sel_windowDidDeminiaturize        = objc.RegisterName("windowDidDeminiaturize:")
	sel_windowDidEnterFullScreen      = objc.RegisterName("windowDidEnterFullScreen:")
	sel_windowDidExitFullScreen       = objc.RegisterName("windowDidExitFullScreen:")
	sel_windowDidMiniaturize          = objc.RegisterName("windowDidMiniaturize:")
	sel_windowDidMove                 = objc.RegisterName("windowDidMove:")
	sel_windowDidResignKey            = objc.RegisterName("windowDidResignKey:")
	sel_windowDidResize               = objc.RegisterName("windowDidResize:")
	sel_windowDidChangeOcclusionState = objc.RegisterName("windowDidChangeOcclusionState:")
	sel_windowResizeEastWestCursor    = objc.RegisterName("_windowResizeEastWestCursor")
	sel_windowResizeNorthSouthCursor  = objc.RegisterName("_windowResizeNorthSouthCursor")
	sel_windowShouldClose             = objc.RegisterName("windowShouldClose:")
	sel_windowWillEnterFullScreen     = objc.RegisterName("windowWillEnterFullScreen:")
	sel_windowWillExitFullScreen      = objc.RegisterName("windowWillExitFullScreen:")
)

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
	x, y := currentMouseLocation()

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
	window := cocoa.NSWindow{ID: objc.ID(w.GetCocoaWindow())}
	pool := cocoa.NSAutoreleasePool_new()
	screen := cocoa.NSScreen_mainScreen()
	if window.ID != 0 && window.IsVisible() {
		// When the window is visible, the window is already initialized.
		// [NSScreen mainScreen] sometimes tells a lie when the window is put across monitors (#703).
		screen = window.Screen()
	}
	screenDictionary := screen.DeviceDescription()
	screenID := cocoa.NSNumber{ID: screenDictionary.ObjectForKey(cocoa.NSString_alloc().InitWithUTF8String("NSScreenNumber").ID)}
	aID := uintptr(screenID.UnsignedIntValue()) // CGDirectDisplayID
	pool.Release()
	for _, m := range ensureMonitors() {
		if m.m.GetCocoaMonitor() == aID {
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
	NSCursor := objc.ID(class_NSCursor).Send(sel_class)
	cursor := NSCursor.Send(sel_performSelector, sel_arrowCursor)
	switch shape {
	case 0:
		cursor = NSCursor.Send(sel_performSelector, sel_arrowCursor)
	case 1:
		cursor = NSCursor.Send(sel_performSelector, sel_IBeamCursor)
	case 2:
		cursor = NSCursor.Send(sel_performSelector, sel_crosshairCursor)
	case 3:
		cursor = NSCursor.Send(sel_performSelector, sel_pointingHandCursor)
	case 4:
		cursor = NSCursor.Send(sel_performSelector, sel_windowResizeEastWestCursor)
	case 5:
		cursor = NSCursor.Send(sel_performSelector, sel_windowResizeNorthSouthCursor)
	}
	cursor.Send(sel_set)
}

func (u *userInterfaceImpl) isNativeFullscreenAvailable() bool {
	// TODO: If the window is transparent, we should use GLFW's windowed fullscreen (#1822, #1857).
	// However, if the user clicks the green button, should this window be in native fullscreen mode?
	return true
}

func (u *userInterfaceImpl) setNativeFullscreen(fullscreen bool) {
	// Toggling fullscreen might ignore events like keyUp. Ensure that events are fired.
	glfw.WaitEventsTimeout(0.1)
	window := cocoa.NSWindow{ID: objc.ID(u.window.GetCocoaWindow())}
	if window.StyleMask()&cocoa.NSWindowStyleMaskFullScreen != 0 == fullscreen {
		return
	}
	// Even though EbitengineWindowDelegate is used, this hack is still required.
	// toggleFullscreen doesn't work when the window is not resizable.
	origCollectionBehavior := window.Send(sel_collectionBehavior)
	origFullScreen := origCollectionBehavior&cocoa.NSWindowCollectionBehaviorFullScreenPrimary != 0
	if !origFullScreen {
		collectionBehavior := origCollectionBehavior
		collectionBehavior |= cocoa.NSWindowCollectionBehaviorFullScreenPrimary
		collectionBehavior &^= cocoa.NSWindowCollectionBehaviorFullScreenNone
		window.Send(sel_setCollectionBehavior, cocoa.NSUInteger(collectionBehavior))
	}
	window.Send(sel_toggleFullScreen, 0)
	if !origFullScreen {
		window.Send(sel_setCollectionBehavior, cocoa.NSUInteger(cocoa.NSUInteger(origCollectionBehavior)))
	}
}

func (u *userInterfaceImpl) adjustViewSizeAfterFullscreen() {
	if u.graphicsDriver.IsGL() {
		return
	}

	window := cocoa.NSWindow{ID: objc.ID(u.window.GetCocoaWindow())}
	if window.StyleMask()&cocoa.NSWindowStyleMaskFullScreen == 0 {
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
	var collectionBehavior uint
	if allowFullscreen {
		collectionBehavior |= cocoa.NSWindowCollectionBehaviorManaged
		collectionBehavior |= cocoa.NSWindowCollectionBehaviorFullScreenPrimary
	} else {
		collectionBehavior |= cocoa.NSWindowCollectionBehaviorFullScreenNone
	}
	objc.ID(u.window.GetCocoaWindow()).Send(sel_setCollectionBehavior, collectionBehavior)
}

func initializeWindowAfterCreation(w *glfw.Window) {
	// TODO: Register NSWindowWillEnterFullScreenNotification and so on.
	// Enable resizing temporary before making the window fullscreen.
	nswindow := objc.ID(w.GetCocoaWindow())
	delegate := objc.ID(class_EbitengineWindowDelegate).Send(sel_alloc).Send(sel_initWithOrigDelegate, nswindow.Send(sel_delegate))
	nswindow.Send(sel_setDelegate, delegate)
}

func (u *userInterfaceImpl) skipTaskbar() error {
	return nil
}
