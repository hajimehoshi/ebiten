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

//go:build !ios

package ui

import (
	"errors"
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

func (u *UserInterface) initializePlatform() error {
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
	d, err := objc.RegisterClass(
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
					if err := u.setOrigWindowPosWithCurrentPos(); err != nil {
						u.setError(err)
						return
					}
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
					if err := u.updateWindowSizeLimits(); err != nil {
						u.setError(err)
						return
					}
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
		return err
	}
	class_EbitengineWindowDelegate = d

	return nil
}

type graphicsDriverCreatorImpl struct {
	transparent bool
	colorSpace  graphicsdriver.ColorSpace
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
	return nil, errors.New("ui: DirectX is not supported in this environment")
}

func (g *graphicsDriverCreatorImpl) newMetal() (graphicsdriver.Graphics, error) {
	return metal.NewGraphics(g.colorSpace)
}

func (*graphicsDriverCreatorImpl) newPlayStation5() (graphicsdriver.Graphics, error) {
	return nil, errors.New("ui: PlayStation 5 is not supported in this environment")
}

// glfwMonitorSizeInGLFWPixels must be called from the main thread.
func glfwMonitorSizeInGLFWPixels(m *glfw.Monitor) (int, int, error) {
	vm, err := m.GetVideoMode()
	if err != nil {
		return 0, 0, err
	}
	return vm.Width, vm.Height, nil
}

func dipFromGLFWPixel(x float64, scale float64) float64 {
	// NOTE: On macOS, GLFW exposes the device independent coordinate system.
	// Thus, the conversion functions are unnecessary,
	// however we still need the deviceScaleFactor internally
	// so we can create and maintain a HiDPI frame buffer.
	return x
}

func dipToGLFWPixel(x float64, scale float64) float64 {
	return x
}

func (u *UserInterface) adjustWindowPosition(x, y int, monitor *Monitor) (int, int) {
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
	sel_setDocumentEdited             = objc.RegisterName("setDocumentEdited:")
	sel_setOrigDelegate               = objc.RegisterName("setOrigDelegate:")
	sel_setOrigResizable              = objc.RegisterName("setOrigResizable:")
	sel_toggleFullScreen              = objc.RegisterName("toggleFullScreen:")
	sel_windowDidBecomeKey            = objc.RegisterName("windowDidBecomeKey:")
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

func currentMouseLocation() (x, y int) {
	sig := cocoa.NSMethodSignature_signatureWithObjCTypes("{NSPoint=dd}@:")
	inv := cocoa.NSInvocation_invocationWithMethodSignature(sig)
	inv.SetTarget(objc.ID(class_NSEvent))
	inv.SetSelector(sel_mouseLocation)
	inv.Invoke()
	var point cocoa.NSPoint
	inv.GetReturnValue(unsafe.Pointer(&point))

	x, y = int(point.X), int(point.Y)

	// On macOS, the Y axis is upward. Adjust the Y position (#807, #2794).
	y = -y
	m := theMonitors.primaryMonitor()
	y += m.videoMode.Height
	return x, y
}

func initialMonitorByOS() (*Monitor, error) {
	x, y := currentMouseLocation()

	// Find the monitor including the cursor.
	return theMonitors.monitorFromPosition(x, y), nil
}

func monitorFromWindowByOS(w *glfw.Window) (*Monitor, error) {
	cocoaWindow, err := w.GetCocoaWindow()
	if err != nil {
		return nil, err
	}
	window := cocoa.NSWindow{ID: objc.ID(cocoaWindow)}
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
		cocoaMonitor, err := m.m.GetCocoaMonitor()
		if err != nil {
			return nil, err
		}
		if cocoaMonitor == aID {
			return m, nil
		}
	}
	return nil, nil
}

func (u *UserInterface) nativeWindow() (uintptr, error) {
	return u.window.GetCocoaWindow()
}

func (u *UserInterface) isNativeFullscreen() (bool, error) {
	w, err := u.window.GetCocoaWindow()
	if err != nil {
		return false, err
	}
	return cocoa.NSWindow{ID: objc.ID(w)}.StyleMask()&cocoa.NSWindowStyleMaskFullScreen != 0, nil
}

func (u *UserInterface) isNativeFullscreenAvailable() bool {
	// TODO: If the window is transparent, we should use GLFW's windowed fullscreen (#1822, #1857).
	// However, if the user clicks the green button, should this window be in native fullscreen mode?
	return true
}

func (u *UserInterface) setNativeFullscreen(fullscreen bool) error {
	// Toggling fullscreen might ignore events like keyUp. Ensure that events are fired.
	if err := glfw.WaitEventsTimeout(0.1); err != nil {
		return err
	}
	w, err := u.window.GetCocoaWindow()
	if err != nil {
		return err
	}
	window := cocoa.NSWindow{ID: objc.ID(w)}
	if window.StyleMask()&cocoa.NSWindowStyleMaskFullScreen != 0 == fullscreen {
		return nil
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

	return nil
}

func (u *UserInterface) adjustViewSizeAfterFullscreen() error {
	if u.GraphicsLibrary() == GraphicsLibraryOpenGL {
		return nil
	}

	w, err := u.window.GetCocoaWindow()
	if err != nil {
		return err
	}
	window := cocoa.NSWindow{ID: objc.ID(w)}
	if window.StyleMask()&cocoa.NSWindowStyleMaskFullScreen == 0 {
		return nil
	}

	// Reduce the view height (#1745).
	// https://stackoverflow.com/questions/27758027/sprite-kit-serious-fps-issue-in-full-screen-mode-on-os-x
	windowSize := window.Frame().Size
	view := window.ContentView()
	viewSize := view.Frame().Size
	if windowSize.Width != viewSize.Width || windowSize.Height != viewSize.Height {
		return nil
	}
	viewSize.Width--
	view.SetFrameSize(viewSize)

	// NSColor.blackColor (0, 0, 0, 1) didn't work.
	// Use the transparent color instead.
	window.SetBackgroundColor(cocoa.NSColor_colorWithSRGBRedGreenBlueAlpha(0, 0, 0, 0))
	return nil
}

func (u *UserInterface) isFullscreenAllowedFromUI(mode WindowResizingMode) bool {
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

func (u *UserInterface) setWindowResizingModeForOS(mode WindowResizingMode) error {
	var collectionBehavior uint
	if u.isFullscreenAllowedFromUI(mode) {
		collectionBehavior |= cocoa.NSWindowCollectionBehaviorManaged
		collectionBehavior |= cocoa.NSWindowCollectionBehaviorFullScreenPrimary
	} else {
		collectionBehavior |= cocoa.NSWindowCollectionBehaviorFullScreenNone
	}
	w, err := u.window.GetCocoaWindow()
	if err != nil {
		return err
	}
	objc.ID(w).Send(sel_setCollectionBehavior, collectionBehavior)
	return nil
}

func initializeWindowAfterCreation(w *glfw.Window) error {
	// TODO: Register NSWindowWillEnterFullScreenNotification and so on.
	// Enable resizing temporary before making the window fullscreen.
	cocoaWindow, err := w.GetCocoaWindow()
	if err != nil {
		return err
	}
	nswindow := objc.ID(cocoaWindow)
	delegate := objc.ID(class_EbitengineWindowDelegate).Send(sel_alloc).Send(sel_initWithOrigDelegate, nswindow.Send(sel_delegate))
	nswindow.Send(sel_setDelegate, delegate)
	return nil
}

func (u *UserInterface) skipTaskbar() error {
	return nil
}

// setDocumentEdited must be called from the main thread.
func (u *UserInterface) setDocumentEdited(edited bool) error {
	w, err := u.window.GetCocoaWindow()
	if err != nil {
		return err
	}
	objc.ID(w).Send(sel_setDocumentEdited, edited)
	return nil
}

func (u *UserInterface) afterWindowCreation() error {
	return nil
}
