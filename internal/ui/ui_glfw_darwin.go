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
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

var class_EbitengineWindowDelegate objc.Class

func init() {
	var err error
	pushResizableState := func(id, win objc.ID) {
		window := cocoa.NSWindow{ID: win}
		id.Send(sel_setOrigResizable, window.StyleMask()&cocoa.NSWindowStyleMaskResizable != 0)
		if !objc.Send[bool](id, sel_origResizable) {
			window.SetStyleMask(window.StyleMask() | cocoa.NSWindowStyleMaskResizable)
		}
	}
	popResizableState := func(id, win objc.ID) {
		if !objc.Send[bool](id, sel_origResizable) {
			window := cocoa.NSWindow{ID: win}
			window.SetStyleMask(window.StyleMask() & ^uint(cocoa.NSWindowStyleMaskResizable))
		}
		id.Send(sel_setOrigResizable, false)
	}
	class_EbitengineWindowDelegate, err = objc.RegisterClass(
		"EbitengineWindowDelegate",
		objc.GetClass("NSObject"),
		[]*objc.Protocol{objc.GetProtocol("NSWindowDelegate")},
		[]objc.FieldDef{
			{
				Name:      "origDelegate",
				Type:      reflect.TypeOf(objc.ID(0)),
				Attribute: objc.ReadWrite,
			},
			{
				Name:      "origResizable",
				Type:      reflect.TypeOf(true),
				Attribute: objc.ReadWrite,
			},
		},
		[]objc.MethodDef{
			{
				Cmd: sel_initWithOrigDelegate,
				Fn: func(id objc.ID, cmd objc.SEL, origDelegate objc.ID) objc.ID {
					self := id.SendSuper(sel_init)
					if self != 0 {
						id.Send(sel_setOrigDelegate, origDelegate)
					}
					return self
				},
			},
			// The method set of origDelegate must sync with GLFWWindowDelegate's implementation.
			// See cocoa_window.m in GLFW.
			{
				Cmd: sel_windowShouldClose,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) bool {
					return id.Send(sel_origDelegate).Send(cmd, notification) != 0
				},
			},
			{
				Cmd: sel_windowDidResize,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(sel_origDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: sel_windowDidMove,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(sel_origDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: sel_windowDidMiniaturize,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(sel_origDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: sel_windowDidBecomeKey,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(sel_origDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: sel_windowDidResignKey,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(sel_origDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: sel_windowDidChangeOcclusionState,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(sel_origDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: sel_windowWillEnterFullScreen,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					theUI.setOrigWindowPosWithCurrentPos()
					pushResizableState(id, cocoa.NSNotification{ID: notification}.Object())
				},
			},
			{
				Cmd: sel_windowDidEnterFullScreen,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					popResizableState(id, cocoa.NSNotification{ID: notification}.Object())
				},
			},
			{
				Cmd: sel_windowWillExitFullScreen,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					pushResizableState(id, cocoa.NSNotification{ID: notification}.Object())
					// Even a window has a size limitation, a window can be fullscreen by calling SetFullscreen(true).
					// In this case, the window size limitation is disabled temporarily.
					// When exiting from fullscreen, reset the window size limitation.
					theUI.updateWindowSizeLimits()
				},
			},
			{
				Cmd: sel_windowDidExitFullScreen,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					popResizableState(id, cocoa.NSNotification{ID: notification}.Object())
					// Do not call setFrame here (#2295). setFrame here causes unexpected results.
				},
			},
		},
	)
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
	sel_collectionBehavior            = objc.RegisterName("collectionBehavior")
	sel_delegate                      = objc.RegisterName("delegate")
	sel_init                          = objc.RegisterName("init")
	sel_initWithOrigDelegate          = objc.RegisterName("initWithOrigDelegate:")
	sel_mouseLocation                 = objc.RegisterName("mouseLocation")
	sel_origDelegate                  = objc.RegisterName("origDelegate")
	sel_origResizable                 = objc.RegisterName("isOrigResizable")
	sel_setCollectionBehavior         = objc.RegisterName("setCollectionBehavior:")
	sel_setDelegate                   = objc.RegisterName("setDelegate:")
	sel_setOrigDelegate               = objc.RegisterName("setOrigDelegate:")
	sel_setOrigResizable              = objc.RegisterName("setOrigResizable:")
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
	sel_windowShouldClose             = objc.RegisterName("windowShouldClose:")
	sel_windowWillEnterFullScreen     = objc.RegisterName("windowWillEnterFullScreen:")
	sel_windowWillExitFullScreen      = objc.RegisterName("windowWillExitFullScreen:")
)

func currentMouseLocationInDIP() (x, y int) {
	sig := cocoa.NSMethodSignature_signatureWithObjCTypes("{NSPoint=dd}@:")
	inv := cocoa.NSInvocation_invocationWithMethodSignature(sig)
	inv.SetTarget(objc.ID(class_NSEvent))
	inv.SetSelector(sel_mouseLocation)
	inv.Invoke()
	var point cocoa.NSPoint
	inv.GetReturnValue(unsafe.Pointer(&point))

	// On macOS, the unit of GLFW (OS-native) pixels' scale and device-independent pixels's scale are the same.
	// The monitor sizes' scales are also the same.
	x, y = int(point.X), int(point.Y)

	// On macOS, the Y axis is upward. Adjust the Y position (#807, #2794).
	y = -y
	m := theMonitors.primaryMonitor()
	y += m.vm.Height
	return x, y
}

func initialMonitorByOS() (*glfw.Monitor, error) {
	x, y := currentMouseLocationInDIP()

	// Find the monitor including the cursor.
	for _, m := range theMonitors.append(nil) {
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
	for _, m := range theMonitors.append(nil) {
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

func (u *userInterfaceImpl) isFullscreenAllowedFromUI(mode WindowResizingMode) bool {
	if u.maxWindowWidthInDIP != glfw.DontCare || u.maxWindowHeightInDIP != glfw.DontCare {
		return false
	}
	if mode == WindowResizingModeOnlyFullscreenEnabled {
		return true
	}
	if mode == WindowResizingModeEnabled {
		return true
	}
	return false
}

func (u *userInterfaceImpl) setWindowResizingModeForOS(mode WindowResizingMode) {
	var collectionBehavior uint
	if u.isFullscreenAllowedFromUI(mode) {
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
