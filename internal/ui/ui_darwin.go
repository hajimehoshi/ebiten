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

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/colormode"
	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/opengl"
)

var classEbitengineWindowDelegate objc.Class

func (u *UserInterface) initializePlatform() error {
	pushResizableState := func(id, win objc.ID) {
		window := cocoa.NSWindow{ID: win}
		id.Send(selSetOrigResizable, window.StyleMask()&cocoa.NSWindowStyleMaskResizable != 0)
		if !objc.Send[bool](id, selOrigResizable) {
			window.SetStyleMask(window.StyleMask() | cocoa.NSWindowStyleMaskResizable)
		}
	}
	popResizableState := func(id, win objc.ID) {
		if !objc.Send[bool](id, selOrigResizable) {
			window := cocoa.NSWindow{ID: win}
			window.SetStyleMask(window.StyleMask() & ^uint(cocoa.NSWindowStyleMaskResizable))
		}
		id.Send(selSetOrigResizable, false)
	}
	d, err := objc.RegisterClass(
		"EbitengineWindowDelegate",
		objc.GetClass("NSObject"),
		[]*objc.Protocol{objc.GetProtocol("NSWindowDelegate")},
		[]objc.FieldDef{
			{
				Name:      "origDelegate",
				Type:      reflect.TypeFor[objc.ID](),
				Attribute: objc.ReadWrite,
			},
			{
				Name:      "origResizable",
				Type:      reflect.TypeFor[bool](),
				Attribute: objc.ReadWrite,
			},
		},
		[]objc.MethodDef{
			{
				Cmd: selInitWithOrigDelegate,
				Fn: func(id objc.ID, cmd objc.SEL, origDelegate objc.ID) objc.ID {
					self := id.SendSuper(selInit)
					if self != 0 {
						id.Send(selSetOrigDelegate, origDelegate)
					}
					return self
				},
			},
			// The method set of origDelegate must sync with GLFWWindowDelegate's implementation.
			// See cocoa_window.m in GLFW.
			{
				Cmd: selWindowShouldClose,
				Fn: func(id objc.ID, cmd objc.SEL, sender objc.ID) bool {
					return id.Send(selOrigDelegate).Send(cmd, sender) != 0
				},
			},
			{
				Cmd: selWindowDidResize,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(selOrigDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: selWindowDidMove,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(selOrigDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: selWindowDidMiniaturize,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(selOrigDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: selWindowDidBecomeKey,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(selOrigDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: selWindowDidResignKey,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(selOrigDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: selWindowDidChangeOcclusionState,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					id.Send(selOrigDelegate).Send(cmd, notification)
				},
			},
			{
				Cmd: selWindowWillEnterFullScreen,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					if err := u.setOrigWindowPosWithCurrentPos(); err != nil {
						u.setError(err)
						return
					}
					pushResizableState(id, cocoa.NSNotification{ID: notification}.Object())
				},
			},
			{
				Cmd: selWindowDidEnterFullScreen,
				Fn: func(id objc.ID, cmd objc.SEL, notification objc.ID) {
					popResizableState(id, cocoa.NSNotification{ID: notification}.Object())
				},
			},
			{
				Cmd: selWindowWillExitFullScreen,
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
				Cmd: selWindowDidExitFullScreen,
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
	classEbitengineWindowDelegate = d

	return nil
}

func (u *UserInterface) setApplePressAndHoldEnabled(enabled bool) {
	var val int
	if enabled {
		val = 1
	}
	defaults := objc.ID(classNSMutableDictionary).Send(selAlloc).Send(selInit)
	defaults.Send(selSetObjectForKey,
		objc.ID(classNSNumber).Send(selAlloc).Send(selInitWithBool, val),
		cocoa.NSString_alloc().InitWithUTF8String("ApplePressAndHoldEnabled").ID)
	ud := objc.ID(classNSUserDefaults).Send(selStandardUserDefaults)
	ud.Send(selRegisterDefaults, defaults)
}

type graphicsDriverCreatorImpl struct {
	transparent bool
	colorSpace  color.ColorSpace
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

func (u *UserInterface) adjustWindowPosition(x, y int, monitor *Monitor) (int, int, error) {
	return x, y, nil
}

var (
	classNSAppearance        = objc.GetClass("NSAppearance")
	classNSCursor            = objc.GetClass("NSCursor")
	classNSEvent             = objc.GetClass("NSEvent")
	classNSMutableDictionary = objc.GetClass("NSMutableDictionary")
	classNSNumber            = objc.GetClass("NSNumber")
	classNSUserDefaults      = objc.GetClass("NSUserDefaults")
)

var (
	selAlloc                         = objc.RegisterName("alloc")
	selAppearanceNamed               = objc.RegisterName("appearanceNamed:")
	selCollectionBehavior            = objc.RegisterName("collectionBehavior")
	selDelegate                      = objc.RegisterName("delegate")
	selInit                          = objc.RegisterName("init")
	selInitWithBool                  = objc.RegisterName("initWithBool:")
	selInitWithOrigDelegate          = objc.RegisterName("initWithOrigDelegate:")
	selMouseLocation                 = objc.RegisterName("mouseLocation")
	selOrigDelegate                  = objc.RegisterName("origDelegate")
	selOrigResizable                 = objc.RegisterName("isOrigResizable")
	selRegisterDefaults              = objc.RegisterName("registerDefaults:")
	selSetAppearance                 = objc.RegisterName("setAppearance:")
	selSetCollectionBehavior         = objc.RegisterName("setCollectionBehavior:")
	selSetDelegate                   = objc.RegisterName("setDelegate:")
	selSetDocumentEdited             = objc.RegisterName("setDocumentEdited:")
	selSetObjectForKey               = objc.RegisterName("setObject:forKey:")
	selSetOrigDelegate               = objc.RegisterName("setOrigDelegate:")
	selSetOrigResizable              = objc.RegisterName("setOrigResizable:")
	selStandardUserDefaults          = objc.RegisterName("standardUserDefaults")
	selToggleFullScreen              = objc.RegisterName("toggleFullScreen:")
	selWindowDidBecomeKey            = objc.RegisterName("windowDidBecomeKey:")
	selWindowDidEnterFullScreen      = objc.RegisterName("windowDidEnterFullScreen:")
	selWindowDidExitFullScreen       = objc.RegisterName("windowDidExitFullScreen:")
	selWindowDidMiniaturize          = objc.RegisterName("windowDidMiniaturize:")
	selWindowDidMove                 = objc.RegisterName("windowDidMove:")
	selWindowDidResignKey            = objc.RegisterName("windowDidResignKey:")
	selWindowDidResize               = objc.RegisterName("windowDidResize:")
	selWindowDidChangeOcclusionState = objc.RegisterName("windowDidChangeOcclusionState:")
	selWindowShouldClose             = objc.RegisterName("windowShouldClose:")
	selWindowWillEnterFullScreen     = objc.RegisterName("windowWillEnterFullScreen:")
	selWindowWillExitFullScreen      = objc.RegisterName("windowWillExitFullScreen:")
)

func currentMouseLocation() (x, y int) {
	point := objc.Send[cocoa.NSPoint](objc.ID(classNSEvent), selMouseLocation)

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
	origCollectionBehavior := window.Send(selCollectionBehavior)
	origFullScreen := origCollectionBehavior&cocoa.NSWindowCollectionBehaviorFullScreenPrimary != 0
	if !origFullScreen {
		collectionBehavior := origCollectionBehavior
		collectionBehavior |= cocoa.NSWindowCollectionBehaviorFullScreenPrimary
		collectionBehavior &^= cocoa.NSWindowCollectionBehaviorFullScreenNone
		window.Send(selSetCollectionBehavior, cocoa.NSUInteger(collectionBehavior))
	}
	window.Send(selToggleFullScreen, 0)
	if !origFullScreen {
		window.Send(selSetCollectionBehavior, cocoa.NSUInteger(cocoa.NSUInteger(origCollectionBehavior)))
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
	s := u.windowSizeLimit.Load().(windowSizeRange)
	if s.maxWidthInDIP != glfw.DontCare || s.maxHeightInDIP != glfw.DontCare {
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
	objc.ID(w).Send(selSetCollectionBehavior, collectionBehavior)
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
	delegate := objc.ID(classEbitengineWindowDelegate).Send(selAlloc).Send(selInitWithOrigDelegate, nswindow.Send(selDelegate))
	nswindow.Send(selSetDelegate, delegate)
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
	objc.ID(w).Send(selSetDocumentEdited, edited)
	return nil
}

func (u *UserInterface) afterWindowCreation() error {
	return nil
}

var (
	nsStringAqua     = cocoa.NSString_alloc().InitWithUTF8String("NSAppearanceNameAqua")
	nsStringDarkAqua = cocoa.NSString_alloc().InitWithUTF8String("NSAppearanceNameDarkAqua")
)

// setWindowColorModeImpl must be called from the main thread.
func (u *UserInterface) setWindowColorModeImpl(mode colormode.ColorMode) error {
	w, err := u.window.GetCocoaWindow()
	if err != nil {
		return err
	}

	var appearance objc.ID
	switch mode {
	case colormode.Light:
		appearance = objc.ID(classNSAppearance).Send(selAppearanceNamed, nsStringAqua.ID)
	case colormode.Dark:
		appearance = objc.ID(classNSAppearance).Send(selAppearanceNamed, nsStringDarkAqua.ID)
	case colormode.Unknown:
		appearance = 0
	}

	objc.ID(w).Send(selSetAppearance, appearance)
	return nil
}
