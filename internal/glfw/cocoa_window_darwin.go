// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"fmt"
	"image"
	"math"
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

// NSPasteboardType strings.
var (
	nsPasteboardTypeString = cocoa.NSString_alloc().InitWithUTF8String("public.utf8-plain-text")
)

// NSDefaultRunLoopMode for event polling.
var nsDefaultRunLoopMode = cocoa.NSString_alloc().InitWithUTF8String("kCFRunLoopDefaultMode")

// Color space name for custom cursor creation.
var nsCalibratedRGBColorSpace = cocoa.NSString_alloc().InitWithUTF8String("NSCalibratedRGBColorSpace")

// Registered ObjC class references.
var (
	classGLFWWindow         objc.Class
	classGLFWWindowDelegate objc.Class
	classGLFWContentView    objc.Class
)

// translateKey converts a macOS virtual key code to a GLFW key constant.
func translateKey(keyCode uint16) Key {
	if int(keyCode) >= len(_glfw.platformWindow.keycodes) {
		return KeyUnknown
	}
	return _glfw.platformWindow.keycodes[keyCode]
}

// translateFlags converts NSEvent modifier flags to GLFW modifier keys.
func translateFlags(flags uintptr) ModifierKey {
	var mods ModifierKey
	if flags&NSEventModifierFlagShift != 0 {
		mods |= ModShift
	}
	if flags&NSEventModifierFlagControl != 0 {
		mods |= ModControl
	}
	if flags&NSEventModifierFlagOption != 0 {
		mods |= ModAlt
	}
	if flags&NSEventModifierFlagCommand != 0 {
		mods |= ModSuper
	}
	if flags&NSEventModifierFlagCapsLock != 0 {
		mods |= ModCapsLock
	}
	return mods
}

// translateKeyToModifierFlag maps a GLFW key to its NSEvent modifier flag.
func translateKeyToModifierFlag(key Key) uintptr {
	switch key {
	case KeyLeftShift, KeyRightShift:
		return NSEventModifierFlagShift
	case KeyLeftControl, KeyRightControl:
		return NSEventModifierFlagControl
	case KeyLeftAlt, KeyRightAlt:
		return NSEventModifierFlagOption
	case KeyLeftSuper, KeyRightSuper:
		return NSEventModifierFlagCommand
	case KeyCapsLock:
		return NSEventModifierFlagCapsLock
	default:
		return 0
	}
}

// cursorInContentArea checks whether the mouse cursor is within a window's content area.
func cursorInContentArea(window *Window) bool {
	if window.platform.object == 0 {
		return false
	}

	pos := objc.Send[cocoa.NSPoint](window.platform.object, sel_mouseLocationOutsideOfEventStream)
	return objc.Send[bool](window.platform.view, sel_mouse_inRect, pos, objc.Send[cocoa.NSRect](window.platform.view, sel_frame))
}

// hideCursor hides the system cursor and optionally disables mouse/cursor association.
func hideCursor(window *Window) {
	if !_glfw.platformWindow.cursorHidden {
		objc.ID(classNSCursor).Send(objc.RegisterName("hide"))
		_glfw.platformWindow.cursorHidden = true
	}
}

// showCursor shows the system cursor and re-enables mouse/cursor association.
func showCursor(window *Window) {
	if _glfw.platformWindow.cursorHidden {
		objc.ID(classNSCursor).Send(sel_unhide)
		_glfw.platformWindow.cursorHidden = false
	}
}

// updateCursorImage sets the appropriate cursor image based on cursor mode.
func updateCursorImage(window *Window) {
	if window.cursorMode == CursorNormal {
		showCursor(window)
		if window.cursor != nil && window.cursor.platform.object != 0 {
			window.cursor.platform.object.Send(sel_set)
		} else {
			objc.ID(classNSCursor).Send(sel_arrowCursor).Send(sel_set)
		}
	} else {
		hideCursor(window)
	}
}

// updateCursorMode applies cursor mode changes.
func updateCursorMode(window *Window) error {
	if window.cursorMode == CursorDisabled {
		_glfw.platformWindow.disabledCursorWindow = window
		_glfw.platformWindow.restoreCursorPosX, _glfw.platformWindow.restoreCursorPosY, _ = window.platformGetCursorPos()
		if err := window.centerCursorInContentArea(); err != nil {
			return err
		}
		cgAssociateMouseAndMouseCursorPosition(0)
	} else if _glfw.platformWindow.disabledCursorWindow == window {
		_glfw.platformWindow.disabledCursorWindow = nil
		if err := window.platformSetCursorPos(_glfw.platformWindow.restoreCursorPosX, _glfw.platformWindow.restoreCursorPosY); err != nil {
			return err
		}
		// NOTE: The matching CGAssociateMouseAndMouseCursorPosition call is
		//       made in platformSetCursorPos as part of a workaround
	}

	if cursorInContentArea(window) {
		updateCursorImage(window)
	}
	return nil
}

// windowForEvent finds the Window associated with a native NSWindow ID.
func windowForEvent(nsWindow objc.ID) *Window {
	for _, w := range _glfw.windows {
		if w.platform.object == nsWindow {
			return w
		}
	}
	return nil
}

// nsApp returns the shared NSApplication instance.
func nsApp() objc.ID {
	return objc.ID(classNSApplication).Send(sel_sharedApplication)
}

// registerGLFWClasses registers the GLFWWindow, GLFWWindowDelegate, and GLFWContentView
// ObjC classes. Called from platformInit.
func registerGLFWClasses() error {
	// GLFWWindow — NSWindow subclass.
	var err error
	classGLFWWindow, err = objc.RegisterClass(
		"GLFWWindow",
		objc.GetClass("NSWindow"),
		nil,
		nil,
		[]objc.MethodDef{
			{
				Cmd: objc.RegisterName("canBecomeKeyWindow"),
				Fn: func(_ objc.ID, _ objc.SEL) bool {
					return true
				},
			},
			{
				Cmd: objc.RegisterName("canBecomeMainWindow"),
				Fn: func(_ objc.ID, _ objc.SEL) bool {
					return true
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("glfw: failed to register GLFWWindow class: %w", err)
	}

	// GLFWWindowDelegate — NSObject subclass conforming to NSWindowDelegate.
	classGLFWWindowDelegate, err = objc.RegisterClass(
		"GLFWWindowDelegate",
		objc.GetClass("NSObject"),
		[]*objc.Protocol{objc.GetProtocol("NSWindowDelegate")},
		[]objc.FieldDef{
			{
				Name:      "goWindow",
				Type:      reflect.TypeFor[uintptr](),
				Attribute: objc.ReadWrite,
			},
		},
		[]objc.MethodDef{
			{
				Cmd: objc.RegisterName("windowShouldClose:"),
				Fn: func(self objc.ID, _ objc.SEL, sender objc.ID) bool {
					window := getGoWindow(classGLFWWindowDelegate, self)
					if window == nil {
						return false
					}
					window.inputWindowCloseRequest()
					return false
				},
			},
			{
				Cmd: objc.RegisterName("windowDidResize:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(classGLFWWindowDelegate, self)
					if window == nil {
						return
					}

					if window.context.source == NativeContextAPI {
						window.context.platform.object.Send(objc.RegisterName("update"))
					}

					if _glfw.platformWindow.disabledCursorWindow == window {
						_ = window.centerCursorInContentArea()
					}

					maximized := objc.Send[bool](window.platform.object, sel_isZoomed)
					if window.platform.maximized != maximized {
						window.platform.maximized = maximized
						window.inputWindowMaximize(maximized)
					}

					updateWindowSize(window)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidMove:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(classGLFWWindowDelegate, self)
					if window == nil {
						return
					}
					if window.platform.object == 0 {
						return
					}

					if window.context.source == NativeContextAPI {
						window.context.platform.object.Send(objc.RegisterName("update"))
					}

					if _glfw.platformWindow.disabledCursorWindow == window {
						_ = window.centerCursorInContentArea()
					}

					frame := objc.Send[cocoa.NSRect](window.platform.object, sel_frame)
					contentRect := objc.Send[cocoa.NSRect](window.platform.object, sel_contentRectForFrameRect, frame)
					xpos := int(contentRect.Origin.X)
					ypos := int(transformYNS(float32(contentRect.Origin.Y + contentRect.Size.Height - 1)))
					window.inputWindowPos(xpos, ypos)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidMiniaturize:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(classGLFWWindowDelegate, self)
					if window == nil {
						return
					}
					if window.monitor != nil {
						_ = window.releaseMonitor()
					}
					window.inputWindowIconify(true)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidDeminiaturize:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(classGLFWWindowDelegate, self)
					if window == nil {
						return
					}
					if window.monitor != nil {
						_ = window.acquireMonitor()
					}
					window.inputWindowIconify(false)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidBecomeKey:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(classGLFWWindowDelegate, self)
					if window == nil {
						return
					}
					if _glfw.platformWindow.disabledCursorWindow == window {
						_ = window.centerCursorInContentArea()
					}
					window.inputWindowFocus(true)
					_ = updateCursorMode(window)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidResignKey:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(classGLFWWindowDelegate, self)
					if window == nil {
						return
					}
					if window.monitor != nil && window.autoIconify {
						window.platformIconifyWindow()
					}
					window.inputWindowFocus(false)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidChangeOcclusionState:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(classGLFWWindowDelegate, self)
					if window == nil {
						return
					}
					if window.platform.object == 0 {
						return
					}
					state := uintptr(window.platform.object.Send(sel_occlusionState))
					window.platform.occluded = (state & NSWindowOcclusionStateVisible) == 0
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("glfw: failed to register GLFWWindowDelegate class: %w", err)
	}

	// GLFWContentView — NSView subclass.
	classGLFWContentView, err = objc.RegisterClass(
		"GLFWContentView",
		objc.GetClass("NSView"),
		[]*objc.Protocol{objc.GetProtocol("NSTextInputClient")},
		[]objc.FieldDef{
			{
				Name:      "goWindow",
				Type:      reflect.TypeFor[uintptr](),
				Attribute: objc.ReadWrite,
			},
		},
		[]objc.MethodDef{
			// Mouse button events.
			{
				Cmd: objc.RegisterName("mouseDown:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					flags := uintptr(event.Send(sel_modifierFlags))
					window.inputMouseClick(MouseButton1, Press, translateFlags(flags))
				},
			},
			{
				Cmd: objc.RegisterName("mouseUp:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					flags := uintptr(event.Send(sel_modifierFlags))
					window.inputMouseClick(MouseButton1, Release, translateFlags(flags))
				},
			},
			{
				Cmd: objc.RegisterName("rightMouseDown:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					flags := uintptr(event.Send(sel_modifierFlags))
					window.inputMouseClick(MouseButton2, Press, translateFlags(flags))
				},
			},
			{
				Cmd: objc.RegisterName("rightMouseUp:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					flags := uintptr(event.Send(sel_modifierFlags))
					window.inputMouseClick(MouseButton2, Release, translateFlags(flags))
				},
			},
			{
				Cmd: objc.RegisterName("otherMouseDown:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					flags := uintptr(event.Send(sel_modifierFlags))
					button := MouseButton(int(event.Send(sel_buttonNumber)))
					window.inputMouseClick(button, Press, translateFlags(flags))
				},
			},
			{
				Cmd: objc.RegisterName("otherMouseUp:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					flags := uintptr(event.Send(sel_modifierFlags))
					button := MouseButton(int(event.Send(sel_buttonNumber)))
					window.inputMouseClick(button, Release, translateFlags(flags))
				},
			},
			// Mouse move events.
			{
				Cmd: objc.RegisterName("mouseMoved:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					handleMouseMoved(window, event)
				},
			},
			{
				Cmd: objc.RegisterName("mouseDragged:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					handleMouseMoved(window, event)
				},
			},
			{
				Cmd: objc.RegisterName("rightMouseDragged:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					handleMouseMoved(window, event)
				},
			},
			{
				Cmd: objc.RegisterName("otherMouseDragged:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					handleMouseMoved(window, event)
				},
			},
			{
				Cmd: objc.RegisterName("mouseExited:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					if window.cursorMode == CursorHidden {
						showCursor(window)
					}
					window.inputCursorEnter(false)
				},
			},
			{
				Cmd: objc.RegisterName("mouseEntered:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					if window.cursorMode == CursorHidden {
						hideCursor(window)
					}
					window.inputCursorEnter(true)
				},
			},
			// Keyboard events.
			{
				Cmd: objc.RegisterName("keyDown:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					keyCode := uint16(event.Send(sel_keyCode))
					key := translateKey(keyCode)
					flags := uintptr(event.Send(sel_modifierFlags))
					mods := translateFlags(flags)

					window.inputKey(key, int(keyCode), Press, mods)

					// Interpret key events for text input.
					eventArray := objc.ID(classNSArray).Send(sel_arrayWithObject, event)
					self.Send(sel_interpretKeyEvents, eventArray)
				},
			},
			{
				Cmd: objc.RegisterName("keyUp:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					keyCode := uint16(event.Send(sel_keyCode))
					key := translateKey(keyCode)
					flags := uintptr(event.Send(sel_modifierFlags))
					mods := translateFlags(flags)

					window.inputKey(key, int(keyCode), Release, mods)
				},
			},
			{
				Cmd: objc.RegisterName("flagsChanged:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					keyCode := uint16(event.Send(sel_keyCode))
					key := translateKey(keyCode)
					flags := uintptr(event.Send(sel_modifierFlags)) & NSEventModifierFlagDeviceIndependentFlagsMask
					mods := translateFlags(flags)

					modFlag := translateKeyToModifierFlag(key)
					var action Action
					if modFlag != 0 && (flags&modFlag) != 0 {
						if window.keys[key] == Press {
							action = Release
						} else {
							action = Press
						}
					} else {
						action = Release
					}

					window.inputKey(key, int(keyCode), action, mods)
				},
			},
			// Scroll wheel.
			{
				Cmd: objc.RegisterName("scrollWheel:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					deltaX := objc.Send[float64](event, sel_scrollingDeltaX)
					deltaY := objc.Send[float64](event, sel_scrollingDeltaY)

					if objc.Send[bool](event, sel_hasPreciseScrollingDeltas) {
						deltaX *= 0.1
						deltaY *= 0.1
					}

					if deltaX != 0 || deltaY != 0 {
						window.inputScroll(deltaX, deltaY)
					}
				},
			},
			// View lifecycle.
			{
				Cmd: objc.RegisterName("viewDidChangeBackingProperties"),
				Fn: func(self objc.ID, _ objc.SEL) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}

					contentRect := objc.Send[cocoa.NSRect](window.platform.view, sel_frame)
					fbRect := objc.Send[cocoa.NSRect](window.platform.view, sel_convertRectToBacking, contentRect)
					xscale := float32(fbRect.Size.Width / contentRect.Size.Width)
					yscale := float32(fbRect.Size.Height / contentRect.Size.Height)

					if xscale != window.platform.xscale || yscale != window.platform.yscale {
						if window.platform.retina && window.platform.layer != 0 {
							window.platform.layer.Send(objc.RegisterName("setContentsScale:"),
								objc.Send[float64](window.platform.object, sel_backingScaleFactor))
						}

						window.platform.xscale = xscale
						window.platform.yscale = yscale
						window.inputWindowContentScale(xscale, yscale)
					}

					if int(fbRect.Size.Width) != window.platform.fbWidth ||
						int(fbRect.Size.Height) != window.platform.fbHeight {
						window.platform.fbWidth = int(fbRect.Size.Width)
						window.platform.fbHeight = int(fbRect.Size.Height)
						window.inputFramebufferSize(int(fbRect.Size.Width), int(fbRect.Size.Height))
					}
				},
			},
			{
				Cmd: objc.RegisterName("updateTrackingAreas"),
				Fn: func(self objc.ID, _ objc.SEL) {
					// Remove all existing tracking areas.
					areas := self.Send(sel_trackingAreas)
					areaCount := int(areas.Send(sel_count))
					for i := range areaCount {
						area := areas.Send(sel_objectAtIndex, i)
						self.Send(sel_removeTrackingArea, area)
					}

					// Create new tracking area.
					bounds := objc.Send[cocoa.NSRect](self, sel_bounds)
					options := uintptr(NSTrackingMouseEnteredAndExited |
						NSTrackingActiveInKeyWindow |
						NSTrackingEnabledDuringMouseDrag |
						NSTrackingCursorUpdate |
						NSTrackingInVisibleRect |
						NSTrackingAssumeInside)

					trackingArea := objc.ID(classNSTrackingArea).Send(sel_alloc).Send(
						sel_initWithRect_options_owner_userInfo,
						bounds, options, self, 0)
					self.Send(sel_addTrackingArea, trackingArea)
					// Balance the alloc; the view now holds the only reference.
					trackingArea.Send(sel_release)

					// Call super.
					self.SendSuper(objc.RegisterName("updateTrackingAreas"))
				},
			},
			{
				Cmd: objc.RegisterName("dealloc"),
				Fn: func(self objc.ID, _ objc.SEL) {
					window := getGoWindow(classGLFWContentView, self)
					if window != nil && window.platform.markedText != 0 {
						window.platform.markedText.Send(sel_release)
						window.platform.markedText = 0
					}
					self.SendSuper(objc.RegisterName("dealloc"))
				},
			},
			{
				Cmd: objc.RegisterName("canBecomeKeyView"),
				Fn: func(_ objc.ID, _ objc.SEL) bool {
					return true
				},
			},
			{
				Cmd: objc.RegisterName("isOpaque"),
				Fn: func(self objc.ID, _ objc.SEL) bool {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return false
					}
					return objc.Send[bool](window.platform.object, objc.RegisterName("isOpaque"))
				},
			},
			{
				Cmd: objc.RegisterName("acceptsFirstResponder"),
				Fn: func(_ objc.ID, _ objc.SEL) bool {
					return true
				},
			},
			{
				Cmd: objc.RegisterName("wantsUpdateLayer"),
				Fn: func(_ objc.ID, _ objc.SEL) bool {
					return true
				},
			},
			{
				Cmd: objc.RegisterName("updateLayer"),
				Fn: func(self objc.ID, _ objc.SEL) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}

					if window.context.source == NativeContextAPI {
						window.context.platform.object.Send(objc.RegisterName("update"))
					}

					window.inputWindowDamage()
				},
			},
			{
				Cmd: objc.RegisterName("cursorUpdate:"),
				Fn: func(self objc.ID, _ objc.SEL, _ objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					updateCursorImage(window)
				},
			},
			{
				Cmd: objc.RegisterName("acceptsFirstMouse:"),
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool {
					return true
				},
			},
			// NSTextInputClient methods.
			{
				Cmd: sel_hasMarkedText,
				Fn: func(self objc.ID, _ objc.SEL) bool {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return false
					}
					if window.platform.markedText != 0 {
						return objc.Send[uintptr](window.platform.markedText, sel_length) > 0
					}
					return false
				},
			},
			{
				Cmd: sel_markedRange,
				Fn: func(self objc.ID, _ objc.SEL) nsRange {
					window := getGoWindow(classGLFWContentView, self)
					if window != nil && window.platform.markedText != 0 {
						length := objc.Send[uintptr](window.platform.markedText, sel_length)
						if length > 0 {
							return nsRange{Location: 0, Length: length - 1}
						}
					}
					return nsRange{Location: ^uintptr(0), Length: 0} // NSNotFound
				},
			},
			{
				Cmd: sel_selectedRange,
				Fn: func(_ objc.ID, _ objc.SEL) nsRange {
					return nsRange{Location: ^uintptr(0), Length: 0} // NSNotFound
				},
			},
			{
				Cmd: sel_setMarkedText_selectedRange_replacementRange,
				Fn: func(self objc.ID, _ objc.SEL, str objc.ID, _ nsRange, _ nsRange) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					if window.platform.markedText != 0 {
						window.platform.markedText.Send(sel_release)
					}
					if str.Send(sel_isKindOfClass, objc.ID(classNSAttributedString)) != 0 {
						window.platform.markedText = objc.ID(classNSMutableAttributedString).Send(sel_alloc).Send(sel_initWithAttributedString, str)
					} else {
						window.platform.markedText = objc.ID(classNSMutableAttributedString).Send(sel_alloc).Send(sel_initWithString, str)
					}
				},
			},
			{
				Cmd: sel_unmarkText,
				Fn: func(self objc.ID, _ objc.SEL) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					if window.platform.markedText != 0 {
						ms := window.platform.markedText.Send(sel_mutableString)
						emptyStr := cocoa.NSString_alloc().InitWithUTF8String("")
						ms.Send(sel_setString, emptyStr.ID)
						emptyStr.ID.Send(sel_release)
					}
				},
			},
			{
				Cmd: sel_validAttributesForMarkedText,
				Fn: func(_ objc.ID, _ objc.SEL) objc.ID {
					// Return an empty autoreleased NSArray (matching C's [NSArray array]).
					return objc.ID(classNSArray).Send(objc.RegisterName("array"))
				},
			},
			{
				Cmd: sel_attributedSubstringForProposedRange_actualRange,
				Fn: func(_ objc.ID, _ objc.SEL, _ nsRange, _ uintptr) objc.ID {
					return 0 // nil
				},
			},
			{
				Cmd: sel_insertText_replacementRange,
				Fn: func(self objc.ID, _ objc.SEL, text objc.ID, _ nsRange) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					nsApp := objc.ID(classNSApplication).Send(sel_sharedApplication)
					event := nsApp.Send(sel_currentEvent)
					flags := uintptr(objc.Send[uint64](event, sel_modifierFlags))
					mods := translateFlags(flags)
					plain := mods&ModSuper == 0

					// Get the string from the text object.
					// The text parameter can be either NSString or NSAttributedString.
					characters := text
					if text.Send(sel_isKindOfClass, objc.ID(classNSAttributedString)) != 0 {
						characters = text.Send(sel_string)
					}
					str := cocoa.NSString{ID: characters}
					s := str.String()
					for _, ch := range s {
						if ch >= 0xf700 && ch <= 0xf7ff {
							continue
						}
						window.inputChar(ch, mods, plain)
					}
				},
			},
			{
				Cmd: sel_characterIndexForPoint,
				Fn: func(_ objc.ID, _ objc.SEL, _ cocoa.NSPoint) uintptr {
					return 0
				},
			},
			{
				Cmd: sel_firstRectForCharacterRange_actualRange,
				Fn: func(self objc.ID, _ objc.SEL, _ nsRange, _ uintptr) cocoa.NSRect {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return cocoa.NSRect{}
					}
					frame := objc.Send[cocoa.NSRect](window.platform.view, sel_frame)
					return cocoa.NSRect{
						Origin: frame.Origin,
						Size:   cocoa.CGSize{Width: 0, Height: 0},
					}
				},
			},
			{
				Cmd: sel_doCommandBySelector,
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.SEL) {
					// Do nothing.
				},
			},
			// Drawing.
			{
				Cmd: objc.RegisterName("drawRect:"),
				Fn: func(self objc.ID, _ objc.SEL, _ cocoa.NSRect) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					window.inputWindowDamage()
				},
			},
			// Drag and drop.
			{
				Cmd: objc.RegisterName("draggingEntered:"),
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) uintptr {
					return NSDragOperationGeneric
				},
			},
			{
				Cmd: objc.RegisterName("performDragOperation:"),
				Fn: func(self objc.ID, _ objc.SEL, sender objc.ID) bool {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return false
					}

					// Update the cursor position to the drop location.
					contentRect := objc.Send[cocoa.NSRect](window.platform.view, sel_frame)
					pos := objc.Send[cocoa.NSPoint](sender, objc.RegisterName("draggingLocation"))
					window.inputCursorPos(pos.X, contentRect.Size.Height-pos.Y)

					pasteboard := sender.Send(sel_draggingPasteboard)
					urlClass := objc.ID(classNSURL)
					classes := objc.ID(classNSArray).Send(sel_arrayWithObject, urlClass)

					// Filter to file URLs only.
					nsYes := objc.ID(objc.GetClass("NSNumber")).Send(objc.RegisterName("numberWithBool:"), true)
					options := objc.ID(objc.GetClass("NSDictionary")).Send(
						objc.RegisterName("dictionaryWithObject:forKey:"),
						uintptr(nsYes), uintptr(nsPasteboardURLReadingFileURLsOnlyKey))

					urls := pasteboard.Send(sel_readObjectsForClasses_options, classes, uintptr(options))
					urlCount := 0
					if urls != 0 {
						urlCount = int(urls.Send(sel_count))
					}

					if urlCount > 0 {
						paths := make([]string, urlCount)
						for i := range urlCount {
							url := urls.Send(sel_objectAtIndex, i)
							// Use fileSystemRepresentation instead of path to handle
							// HFS+ Unicode normalization correctly.
							fsRep := url.Send(objc.RegisterName("fileSystemRepresentation"))
							if fsRep != 0 {
								paths[i] = goStringFromCString(uintptr(fsRep))
							}
						}

						window.inputDrop(paths)
					}

					return true
				},
			},
		},
	)
	if err != nil {
		return fmt.Errorf("glfw: failed to register GLFWContentView class: %w", err)
	}

	return nil
}

// getGoWindow extracts the Go *Window pointer from an ObjC instance's goWindow field.
func getGoWindow(class objc.Class, id objc.ID) *Window {
	ptr := objc.Send[uintptr](id, objc.RegisterName("goWindow"))
	if ptr == 0 {
		return nil
	}
	return (*Window)(unsafe.Pointer(ptr))
}

// setGoWindow stores the Go *Window pointer in an ObjC instance's goWindow field.
func setGoWindow(id objc.ID, window *Window) {
	id.Send(objc.RegisterName("setGoWindow:"), uintptr(unsafe.Pointer(window)))
}

// handleMouseMoved processes mouse movement events.
func handleMouseMoved(window *Window, event objc.ID) {
	if window.cursorMode == CursorDisabled {
		dx := objc.Send[float64](event, sel_deltaX)
		dy := objc.Send[float64](event, sel_deltaY)

		dx -= window.platform.cursorWarpDeltaX
		dy -= window.platform.cursorWarpDeltaY

		window.inputCursorPos(
			window.virtualCursorPosX+dx,
			window.virtualCursorPosY+dy)
	} else {
		// Get the location in the content view.
		pos := objc.Send[cocoa.NSPoint](event, sel_locationInWindow)

		// Convert from Cocoa coordinates (origin at bottom-left) to GLFW coordinates (origin at top-left).
		contentRect := objc.Send[cocoa.NSRect](window.platform.view, sel_frame)
		pos.Y = contentRect.Size.Height - pos.Y

		window.inputCursorPos(pos.X, pos.Y)
	}

	window.platform.cursorWarpDeltaX = 0
	window.platform.cursorWarpDeltaY = 0
}

// updateWindowSize updates the cached window and framebuffer sizes, invoking callbacks as needed.
func updateWindowSize(window *Window) {
	if window.platform.object == 0 || window.platform.view == 0 {
		return
	}

	contentRect := objc.Send[cocoa.NSRect](window.platform.view, sel_frame)
	fbRect := objc.Send[cocoa.NSRect](window.platform.view, sel_convertRectToBacking, contentRect)

	fbWidth := int(fbRect.Size.Width)
	fbHeight := int(fbRect.Size.Height)

	if fbWidth != window.platform.fbWidth || fbHeight != window.platform.fbHeight {
		window.platform.fbWidth = fbWidth
		window.platform.fbHeight = fbHeight
		window.inputFramebufferSize(fbWidth, fbHeight)
	}

	width := int(contentRect.Size.Width)
	height := int(contentRect.Size.Height)

	if width != window.platform.width || height != window.platform.height {
		window.platform.width = width
		window.platform.height = height
		window.inputWindowSize(width, height)
	}
}

// createNativeWindow creates the actual NSWindow, delegate, and content view.
func createNativeWindow(window *Window, wndconfig *wndconfig, fbconfig_ *fbconfig) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// Create the delegate first (before the window, to avoid leaking the
	// window if delegate creation fails).
	delegateID := objc.ID(classGLFWWindowDelegate).Send(sel_alloc).Send(sel_init)
	if delegateID == 0 {
		return fmt.Errorf("glfw: failed to create window delegate: %w", PlatformError)
	}
	setGoWindow(delegateID, window)
	window.platform.delegate = delegateID

	// Determine the content rect.
	var contentRect cocoa.NSRect
	if window.monitor != nil {
		mode := window.monitor.platformGetVideoMode()
		xpos, ypos, _ := window.monitor.platformGetMonitorPos()
		contentRect = cocoa.NSRect{
			Origin: cocoa.NSPoint{X: float64(xpos), Y: float64(ypos)},
			Size:   cocoa.NSSize{Width: float64(mode.Width), Height: float64(mode.Height)},
		}
	} else {
		contentRect = cocoa.NSRect{
			Origin: cocoa.NSPoint{X: 0, Y: 0},
			Size:   cocoa.NSSize{Width: float64(wndconfig.width), Height: float64(wndconfig.height)},
		}
	}

	// Determine the style mask.
	styleMask := uintptr(NSWindowStyleMaskMiniaturizable)
	if window.monitor != nil || !wndconfig.decorated {
		styleMask |= NSWindowStyleMaskBorderless
	} else {
		styleMask |= NSWindowStyleMaskTitled | NSWindowStyleMaskClosable
		if wndconfig.resizable {
			styleMask |= NSWindowStyleMaskResizable
		}
	}

	// Create the GLFWWindow instance.
	nsWindow := objc.ID(classGLFWWindow).Send(sel_alloc).Send(sel_initWithContentRect_styleMask_backing_defer,
		contentRect, styleMask, uintptr(NSBackingStoreBuffered), false)
	if nsWindow == 0 {
		return fmt.Errorf("glfw: failed to create Cocoa window: %w", PlatformError)
	}

	window.platform.object = nsWindow

	if window.monitor != nil {
		nsWindow.Send(sel_setLevel, uintptr(NSMainMenuWindowLevel+1))
	} else {
		// Center the window on the screen.
		nsWindow.Send(objc.RegisterName("center"))
		cascadeIn := cocoa.NSPoint{
			X: _glfw.platformWindow.cascadePoint[0],
			Y: _glfw.platformWindow.cascadePoint[1],
		}
		cascadeOut := objc.Send[cocoa.NSPoint](nsWindow, objc.RegisterName("cascadeTopLeftFromPoint:"), cascadeIn)
		_glfw.platformWindow.cascadePoint[0] = cascadeOut.X
		_glfw.platformWindow.cascadePoint[1] = cascadeOut.Y

		if wndconfig.resizable {
			nsWindow.Send(sel_setCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenPrimary|_NSWindowCollectionBehaviorManaged))
		} else {
			nsWindow.Send(sel_setCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenNone))
		}

		if wndconfig.floating {
			nsWindow.Send(sel_setLevel, uintptr(NSFloatingWindowLevel))
		}

		if wndconfig.maximized {
			nsWindow.Send(sel_zoom, 0)
		}
	}

	if len(wndconfig.frameName) > 0 {
		name := cocoa.NSString_alloc().InitWithUTF8String(wndconfig.frameName)
		nsWindow.Send(sel_setFrameAutosaveName, name.ID)
		name.ID.Send(sel_release)
	}

	// Create the content view.
	viewID := objc.ID(classGLFWContentView).Send(sel_alloc).Send(sel_init)
	setGoWindow(viewID, window)
	window.platform.view = viewID
	window.platform.retina = wndconfig.retina
	window.platform.markedText = objc.ID(classNSMutableAttributedString).Send(sel_alloc).Send(objc.RegisterName("init"))

	// Handle transparent framebuffer.
	if fbconfig_.transparent {
		nsWindow.Send(sel_setOpaque, false)
		nsWindow.Send(sel_setHasShadow, false)
		nsWindow.Send(sel_setBackgroundColor, objc.ID(classNSColor).Send(sel_clearColor))
	}

	nsWindow.Send(sel_setContentView, viewID)
	nsWindow.Send(sel_makeFirstResponder, viewID)
	titleStr := cocoa.NSString_alloc().InitWithUTF8String(wndconfig.title)
	nsWindow.Send(sel_setTitle, titleStr.ID)
	titleStr.ID.Send(sel_release)
	nsWindow.Send(sel_setDelegate, delegateID)
	nsWindow.Send(objc.RegisterName("setAcceptsMouseMovedEvents:"), true)
	nsWindow.Send(sel_setRestorable, false)

	// Disable window tabbing (macOS 10.12+).
	sel_setTabbingMode := objc.RegisterName("setTabbingMode:")
	if nsWindow.Send(objc.RegisterName("respondsToSelector:"), sel_setTabbingMode) != 0 {
		nsWindow.Send(sel_setTabbingMode, uintptr(2)) // NSWindowTabbingModeDisallowed = 2
	}

	viewID.Send(objc.RegisterName("updateTrackingAreas"))

	// Register for dragged types (URLs).
	typesArray := objc.ID(classNSArray).Send(sel_arrayWithObject, nsPasteboardTypeURL)
	viewID.Send(sel_registerForDraggedTypes, typesArray)

	// Update initial size cache.
	contentViewRect := objc.Send[cocoa.NSRect](viewID, sel_frame)
	window.platform.width = int(contentViewRect.Size.Width)
	window.platform.height = int(contentViewRect.Size.Height)

	fbRect := objc.Send[cocoa.NSRect](viewID, sel_convertRectToBacking, contentViewRect)
	window.platform.fbWidth = int(fbRect.Size.Width)
	window.platform.fbHeight = int(fbRect.Size.Height)

	return nil
}

// --- Platform window functions ---

func (w *Window) platformCreateWindow(wndconfig *wndconfig, ctxconfig *ctxconfig, fbconfig_ *fbconfig) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if err := createNativeWindow(w, wndconfig, fbconfig_); err != nil {
		return err
	}

	if ctxconfig.client != NoAPI {
		if ctxconfig.source == NativeContextAPI {
			if err := initNSGL(); err != nil {
				return err
			}
			if err := w.createContextNSGL(ctxconfig, fbconfig_); err != nil {
				return err
			}
		}
		if err := w.refreshContextAttribs(ctxconfig); err != nil {
			return err
		}
	}

	if wndconfig.mousePassthrough {
		if err := w.platformSetWindowMousePassthrough(true); err != nil {
			return err
		}
	}

	if w.monitor != nil {
		w.platformShowWindow()
		if err := w.platformFocusWindow(); err != nil {
			return err
		}
		if err := w.acquireMonitor(); err != nil {
			return err
		}
		if wndconfig.centerCursor {
			if err := w.centerCursorInContentArea(); err != nil {
				return err
			}
		}
	} else {
		if wndconfig.visible {
			w.platformShowWindow()
			if wndconfig.focused {
				if err := w.platformFocusWindow(); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (w *Window) platformDestroyWindow() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if _glfw.platformWindow.disabledCursorWindow == w {
		_glfw.platformWindow.disabledCursorWindow = nil
	}

	if w.platform.object != 0 {
		w.platform.object.Send(sel_orderOut, 0)
	}

	if w.monitor != nil {
		if err := w.releaseMonitor(); err != nil {
			return err
		}
	}

	if w.context.destroy != nil {
		if err := w.context.destroy(w); err != nil {
			return err
		}
	}

	if w.platform.delegate != 0 {
		w.platform.object.Send(sel_setDelegate, 0)
		w.platform.delegate.Send(sel_release)
		w.platform.delegate = 0
	}

	if w.platform.view != 0 {
		w.platform.view.Send(sel_release)
		w.platform.view = 0
	}

	if w.platform.object != 0 {
		w.platform.object.Send(objc.RegisterName("close"))
		w.platform.object = 0
	}

	// HACK: Allow Cocoa to catch up before returning
	if err := platformPollEvents(); err != nil {
		return err
	}

	return nil
}

func (w *Window) platformSetWindowTitle(title string) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	s := cocoa.NSString_alloc().InitWithUTF8String(title)
	w.platform.object.Send(sel_setTitle, s.ID)
	// HACK: Set the miniwindow title explicitly as setTitle: doesn't update it
	//       if the window lacks NSWindowStyleMaskTitled
	w.platform.object.Send(sel_setMiniwindowTitle, s.ID)
	s.ID.Send(sel_release)
	return nil
}

func (w *Window) platformSetWindowIcon(images []*Image) error {
	// macOS does not support per-window icons. The dock icon is set at the application level.
	return nil
}

func (w *Window) platformGetWindowPos() (xpos, ypos int, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	frame := objc.Send[cocoa.NSRect](w.platform.object, sel_frame)
	contentRect := objc.Send[cocoa.NSRect](w.platform.object, sel_contentRectForFrameRect, frame)

	xpos = int(contentRect.Origin.X)
	ypos = int(transformYNS(float32(contentRect.Origin.Y + contentRect.Size.Height - 1)))
	return xpos, ypos, nil
}

func (w *Window) platformSetWindowPos(xpos, ypos int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	viewFrame := objc.Send[cocoa.NSRect](w.platform.view, sel_frame)
	dummyRect := cocoa.NSRect{
		Origin: cocoa.NSPoint{
			X: float64(xpos),
			Y: float64(transformYNS(float32(float64(ypos) + viewFrame.Size.Height - 1))),
		},
		Size: cocoa.NSSize{Width: 0, Height: 0},
	}

	frameRect := objc.Send[cocoa.NSRect](w.platform.object, sel_frameRectForContentRect, dummyRect)
	w.platform.object.Send(sel_setFrameOrigin, frameRect.Origin)
	return nil
}

func (w *Window) platformGetWindowSize() (width, height int, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	contentRect := objc.Send[cocoa.NSRect](w.platform.view, sel_frame)
	return int(contentRect.Size.Width), int(contentRect.Size.Height), nil
}

func (w *Window) platformSetWindowSize(width, height int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if w.monitor != nil {
		if w.monitor.window == w {
			if err := w.acquireMonitor(); err != nil {
				return err
			}
		}
		return nil
	}

	contentRect := objc.Send[cocoa.NSRect](w.platform.object, sel_contentRectForFrameRect,
		objc.Send[cocoa.NSRect](w.platform.object, sel_frame))
	contentRect.Origin.Y += contentRect.Size.Height - float64(height)
	contentRect.Size = cocoa.NSSize{Width: float64(width), Height: float64(height)}
	frameRect := objc.Send[cocoa.NSRect](w.platform.object, sel_frameRectForContentRect, contentRect)
	w.platform.object.Send(objc.RegisterName("setFrame:display:"), frameRect, true)
	return nil
}

func (w *Window) platformSetWindowSizeLimits(minwidth, minheight, maxwidth, maxheight int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if minwidth == DontCare || minheight == DontCare {
		w.platform.object.Send(sel_setContentMinSize, cocoa.NSSize{Width: 0, Height: 0})
	} else {
		w.platform.object.Send(sel_setContentMinSize, cocoa.NSSize{Width: float64(minwidth), Height: float64(minheight)})
	}

	if maxwidth == DontCare || maxheight == DontCare {
		w.platform.object.Send(sel_setContentMaxSize, cocoa.NSSize{Width: math.MaxFloat64, Height: math.MaxFloat64})
	} else {
		w.platform.object.Send(sel_setContentMaxSize, cocoa.NSSize{Width: float64(maxwidth), Height: float64(maxheight)})
	}

	return nil
}

func (w *Window) platformSetWindowAspectRatio(numer, denom int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if numer == DontCare || denom == DontCare {
		w.platform.object.Send(sel_setResizeIncrements, cocoa.NSSize{Width: 1, Height: 1})
	} else {
		w.platform.object.Send(sel_setContentAspectRatio, cocoa.NSSize{Width: float64(numer), Height: float64(denom)})
	}
	return nil
}

func (w *Window) platformGetFramebufferSize() (width, height int, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	contentRect := objc.Send[cocoa.NSRect](w.platform.view, sel_frame)
	fbRect := objc.Send[cocoa.NSRect](w.platform.view, sel_convertRectToBacking, contentRect)
	return int(fbRect.Size.Width), int(fbRect.Size.Height), nil
}

func (w *Window) platformGetWindowFrameSize() (left, top, right, bottom int, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	contentRect := objc.Send[cocoa.NSRect](w.platform.view, sel_frame)
	frameRect := objc.Send[cocoa.NSRect](w.platform.object, sel_frameRectForContentRect, contentRect)

	left = int(contentRect.Origin.X - frameRect.Origin.X)
	top = int((frameRect.Origin.Y + frameRect.Size.Height) - (contentRect.Origin.Y + contentRect.Size.Height))
	right = int((frameRect.Origin.X + frameRect.Size.Width) - (contentRect.Origin.X + contentRect.Size.Width))
	bottom = int(contentRect.Origin.Y - frameRect.Origin.Y)
	return left, top, right, bottom, nil
}

func (w *Window) platformGetWindowContentScale() (xscale, yscale float32, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	points := objc.Send[cocoa.NSRect](w.platform.view, sel_frame)
	pixels := objc.Send[cocoa.NSRect](w.platform.view, sel_convertRectToBacking, points)

	return float32(pixels.Size.Width / points.Size.Width), float32(pixels.Size.Height / points.Size.Height), nil
}

func (w *Window) platformIconifyWindow() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(sel_miniaturize, 0)
}

func (w *Window) platformRestoreWindow() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if objc.Send[bool](w.platform.object, objc.RegisterName("isMiniaturized")) {
		w.platform.object.Send(sel_deminiaturize, 0)
	} else if objc.Send[bool](w.platform.object, sel_isZoomed) {
		w.platform.object.Send(sel_zoom, 0)
	}
}

func (w *Window) platformMaximizeWindow() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if !objc.Send[bool](w.platform.object, sel_isZoomed) {
		w.platform.object.Send(sel_zoom, 0)
	}
	return nil
}

func (w *Window) platformShowWindow() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(sel_orderFront, 0)
}

func (w *Window) platformHideWindow() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(sel_orderOut, 0)
}

func (w *Window) platformRequestWindowAttention() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// NSInformationalRequest = 10
	nsApp().Send(sel_requestUserAttention, uintptr(10))
}

func (w *Window) platformFocusWindow() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	nsApp().Send(sel_activateIgnoringOtherApps, true)
	w.platform.object.Send(sel_makeKeyAndOrderFront, 0)
	return nil
}

func (w *Window) platformSetWindowMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if w.monitor == monitor {
		if monitor != nil {
			if monitor.window == w {
				if err := w.acquireMonitor(); err != nil {
					return err
				}
			}
		} else {
			contentRect := cocoa.NSRect{
				Origin: cocoa.NSPoint{
					X: float64(xpos),
					Y: float64(transformYNS(float32(ypos + height - 1))),
				},
				Size: cocoa.NSSize{Width: float64(width), Height: float64(height)},
			}
			styleMask := uintptr(objc.Send[uint64](w.platform.object, sel_styleMask))
			frameRect := objc.Send[cocoa.NSRect](w.platform.object, sel_frameRectForContentRect_styleMask, contentRect, styleMask)
			w.platform.object.Send(objc.RegisterName("setFrame:display:"), frameRect, true)
		}
		return nil
	}

	if w.monitor != nil {
		if err := w.releaseMonitor(); err != nil {
			return err
		}
	}

	w.inputWindowMonitor(monitor)

	// HACK: Allow the state cached in Cocoa to catch up to reality
	if err := platformPollEvents(); err != nil {
		return err
	}

	styleMask := uintptr(objc.Send[uint64](w.platform.object, sel_styleMask))

	if w.monitor != nil {
		styleMask &^= NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskResizable
		styleMask |= NSWindowStyleMaskBorderless
	} else {
		if w.decorated {
			styleMask &^= NSWindowStyleMaskBorderless
			styleMask |= NSWindowStyleMaskTitled | NSWindowStyleMaskClosable
		}
		if w.resizable {
			styleMask |= NSWindowStyleMaskResizable
		} else {
			styleMask &^= NSWindowStyleMaskResizable
		}
	}

	w.platform.object.Send(sel_setStyleMask, styleMask)
	// HACK: Changing the style mask can cause the first responder to be cleared
	w.platform.object.Send(sel_makeFirstResponder, w.platform.view)

	if w.monitor != nil {
		w.platform.object.Send(sel_setLevel, uintptr(NSMainMenuWindowLevel+1))
		w.platform.object.Send(sel_setHasShadow, false)

		if err := w.acquireMonitor(); err != nil {
			return err
		}
	} else {
		contentRect := cocoa.NSRect{
			Origin: cocoa.NSPoint{
				X: float64(xpos),
				Y: float64(transformYNS(float32(ypos + height - 1))),
			},
			Size: cocoa.NSSize{Width: float64(width), Height: float64(height)},
		}
		frameRect := objc.Send[cocoa.NSRect](w.platform.object, sel_frameRectForContentRect_styleMask, contentRect, styleMask)
		w.platform.object.Send(objc.RegisterName("setFrame:display:"), frameRect, true)

		if w.numer != DontCare && w.denom != DontCare {
			w.platform.object.Send(sel_setContentAspectRatio, cocoa.NSSize{Width: float64(w.numer), Height: float64(w.denom)})
		}

		if w.minwidth != DontCare && w.minheight != DontCare {
			w.platform.object.Send(sel_setContentMinSize, cocoa.NSSize{Width: float64(w.minwidth), Height: float64(w.minheight)})
		}

		if w.maxwidth != DontCare && w.maxheight != DontCare {
			w.platform.object.Send(sel_setContentMaxSize, cocoa.NSSize{Width: float64(w.maxwidth), Height: float64(w.maxheight)})
		}

		if w.floating {
			w.platform.object.Send(sel_setLevel, uintptr(NSFloatingWindowLevel))
		} else {
			w.platform.object.Send(sel_setLevel, uintptr(NSNormalWindowLevel))
		}

		if w.resizable {
			w.platform.object.Send(sel_setCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenPrimary|_NSWindowCollectionBehaviorManaged))
		} else {
			w.platform.object.Send(sel_setCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenNone))
		}

		w.platform.object.Send(sel_setHasShadow, true)
		// HACK: Clearing NSWindowStyleMaskTitled resets and disables the window
		//       title property but the miniwindow title property is unaffected
		miniTitle := w.platform.object.Send(sel_miniwindowTitle)
		w.platform.object.Send(sel_setTitle, miniTitle)
	}

	return nil
}

func (w *Window) platformWindowFocused() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return objc.Send[bool](w.platform.object, sel_isKeyWindow)
}

func (w *Window) platformWindowIconified() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return objc.Send[bool](w.platform.object, sel_isMiniaturized)
}

func (w *Window) platformWindowVisible() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return objc.Send[bool](w.platform.object, sel_isVisible)
}

func (w *Window) platformWindowMaximized() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if w.resizable {
		return objc.Send[bool](w.platform.object, sel_isZoomed)
	}
	return false
}

func (w *Window) platformWindowHovered() (bool, error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	pos := objc.Send[cocoa.NSPoint](objc.ID(classNSEvent), objc.RegisterName("mouseLocation"))

	// Check if this window is the topmost window at the cursor position.
	topWindowNumber := objc.Send[uintptr](objc.ID(classNSWindow), objc.RegisterName("windowNumberAtPoint:belowWindowWithWindowNumber:"), pos, uintptr(0))
	if topWindowNumber != uintptr(w.platform.object.Send(sel_windowNumber)) {
		return false, nil
	}

	viewFrame := objc.Send[cocoa.NSRect](w.platform.view, sel_frame)
	screenRect := objc.Send[cocoa.NSRect](w.platform.object, sel_convertRectToScreen, viewFrame)
	// Match NSMouseInRect(point, rect, NO) behavior for non-flipped coordinates:
	// x >= origin.x && x < maxX && y > origin.y && y <= maxY
	return pos.X >= screenRect.Origin.X &&
		pos.X < screenRect.Origin.X+screenRect.Size.Width &&
		pos.Y > screenRect.Origin.Y &&
		pos.Y <= screenRect.Origin.Y+screenRect.Size.Height, nil
}

func (w *Window) platformFramebufferTransparent() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return !objc.Send[bool](w.platform.object, objc.RegisterName("isOpaque")) &&
		!objc.Send[bool](w.platform.view, objc.RegisterName("isOpaque"))
}

func (w *Window) platformSetWindowResizable(enabled bool) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	mask := uintptr(w.platform.object.Send(objc.RegisterName("styleMask")))
	if enabled {
		mask |= NSWindowStyleMaskResizable
	} else {
		mask &^= NSWindowStyleMaskResizable
	}
	w.platform.object.Send(objc.RegisterName("setStyleMask:"), mask)
	if enabled {
		w.platform.object.Send(sel_setCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenPrimary|_NSWindowCollectionBehaviorManaged))
	} else {
		w.platform.object.Send(sel_setCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenNone))
	}
	return nil
}

func (w *Window) platformSetWindowDecorated(enabled bool) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	mask := uintptr(w.platform.object.Send(sel_styleMask))
	if enabled {
		mask |= NSWindowStyleMaskTitled | NSWindowStyleMaskClosable
		mask &^= NSWindowStyleMaskBorderless
	} else {
		mask |= NSWindowStyleMaskBorderless
		mask &^= (NSWindowStyleMaskTitled | NSWindowStyleMaskClosable)
	}
	w.platform.object.Send(sel_setStyleMask, mask)
	w.platform.object.Send(sel_makeFirstResponder, w.platform.view)
	return nil
}

func (w *Window) platformSetWindowFloating(enabled bool) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if enabled {
		w.platform.object.Send(sel_setLevel, uintptr(NSFloatingWindowLevel))
	} else {
		w.platform.object.Send(sel_setLevel, uintptr(NSNormalWindowLevel))
	}
	return nil
}

func (w *Window) platformSetWindowMousePassthrough(enabled bool) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(sel_setIgnoresMouseEvents, enabled)
	return nil
}

func (w *Window) platformGetWindowOpacity() (float32, error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return float32(objc.Send[float64](w.platform.object, sel_alphaValue)), nil
}

func (w *Window) platformSetWindowOpacity(opacity float32) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(sel_setAlphaValue, float64(opacity))
	return nil
}

func (w *Window) platformSetRawMouseMotion(enabled bool) error {
	// Raw mouse motion is not supported on macOS.
	return nil
}

// --- Event polling ---

func platformPollEvents() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	distantPast := objc.ID(objc.GetClass("NSDate")).Send(objc.RegisterName("distantPast"))
	for {
		event := objc.Send[objc.ID](nsApp(), sel_nextEventMatchingMask_untilDate_inMode_dequeue,
			uintptr(NSEventMaskAny),
			distantPast,
			nsDefaultRunLoopMode.ID,
			true)
		if event == 0 {
			break
		}
		nsApp().Send(sel_sendEvent, event)
	}
	return nil
}

func platformWaitEvents() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// Wait for an event with no timeout (distantFuture).
	distantFuture := objc.ID(objc.GetClass("NSDate")).Send(objc.RegisterName("distantFuture"))
	event := objc.Send[objc.ID](nsApp(), sel_nextEventMatchingMask_untilDate_inMode_dequeue,
		uintptr(NSEventMaskAny),
		distantFuture,
		nsDefaultRunLoopMode.ID,
		true)
	if event != 0 {
		nsApp().Send(sel_sendEvent, event)
	}

	if err := platformPollEvents(); err != nil {
		return err
	}
	return nil
}

func platformWaitEventsTimeout(timeout float64) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// Create an NSDate for the timeout.
	date := objc.ID(objc.GetClass("NSDate")).Send(objc.RegisterName("dateWithTimeIntervalSinceNow:"), timeout)
	event := objc.Send[objc.ID](nsApp(), sel_nextEventMatchingMask_untilDate_inMode_dequeue,
		uintptr(NSEventMaskAny),
		date,
		nsDefaultRunLoopMode.ID,
		true)
	if event != 0 {
		nsApp().Send(sel_sendEvent, event)
	}

	if err := platformPollEvents(); err != nil {
		return err
	}
	return nil
}

func platformPostEmptyEvent() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	postEmptyEvent()
	return nil
}

// --- Cursor functions ---

func (w *Window) platformGetCursorPos() (xpos, ypos float64, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	contentRect := objc.Send[cocoa.NSRect](w.platform.view, sel_frame)
	// NOTE: The returned location uses base 0,1 not 0,0
	pos := objc.Send[cocoa.NSPoint](w.platform.object, sel_mouseLocationOutsideOfEventStream)

	xpos = pos.X
	ypos = contentRect.Size.Height - pos.Y

	return xpos, ypos, nil
}

func (w *Window) platformSetCursorPos(xpos, ypos float64) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	updateCursorImage(w)

	contentRect := objc.Send[cocoa.NSRect](w.platform.view, sel_frame)
	// NOTE: The returned location uses base 0,1 not 0,0
	pos := objc.Send[cocoa.NSPoint](w.platform.object, sel_mouseLocationOutsideOfEventStream)

	w.platform.cursorWarpDeltaX += xpos - pos.X
	w.platform.cursorWarpDeltaY += ypos - contentRect.Size.Height + pos.Y

	if w.monitor != nil {
		cgDisplayMoveCursorToPoint(w.monitor.platform.displayID, cocoa.CGPoint{X: xpos, Y: ypos})
	} else {
		localRect := cocoa.NSRect{
			Origin: cocoa.NSPoint{X: xpos, Y: contentRect.Size.Height - ypos - 1},
			Size:   cocoa.NSSize{Width: 0, Height: 0},
		}
		globalRect := objc.Send[cocoa.NSRect](w.platform.object, sel_convertRectToScreen, localRect)
		globalPoint := globalRect.Origin

		cgWarpMouseCursorPosition(cocoa.CGPoint{
			X: globalPoint.X,
			Y: float64(transformYNS(float32(globalPoint.Y))),
		})
	}

	// HACK: Calling this right after setting the cursor position prevents macOS
	//       from freezing the cursor for a fraction of a second afterwards.
	if w.cursorMode != CursorDisabled {
		cgAssociateMouseAndMouseCursorPosition(1)
	}

	return nil
}

func (w *Window) platformSetCursorMode(mode int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if w.platformWindowFocused() {
		if err := updateCursorMode(w); err != nil {
			return err
		}
	}

	return nil
}

func (c *Cursor) platformCreateCursor(img *image.NRGBA, xhot, yhot int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	rep := objc.ID(classNSBitmapImageRep).Send(sel_alloc).Send(sel_initWithBitmapDataPlanes_pixelsWide_pixelsHigh_bitsPerSample_samplesPerPixel_hasAlpha_isPlanar_colorSpaceName_bitmapFormat_bytesPerRow_bitsPerPixel,
		uintptr(0),                   // planes (NULL = allocate)
		uintptr(w),                   // pixelsWide
		uintptr(h),                   // pixelsHigh
		uintptr(8),                   // bitsPerSample
		uintptr(4),                   // samplesPerPixel
		true,                         // hasAlpha
		false,                        // isPlanar
		nsCalibratedRGBColorSpace.ID, // colorSpaceName
		uintptr(1<<0),                // bitmapFormat: NSBitmapFormatAlphaNonpremultiplied = 1 << 0
		uintptr(w*4),                 // bytesPerRow
		uintptr(32),                  // bitsPerPixel
	)
	if rep == 0 {
		return fmt.Errorf("glfw: failed to create NSBitmapImageRep: %w", PlatformError)
	}

	// Copy pixel data into the bitmap.
	bitmapData := rep.Send(sel_bitmapData)
	if bitmapData != 0 {
		dst := unsafe.Slice((*byte)(unsafe.Pointer(bitmapData)), w*h*4)
		copy(dst, img.Pix)
	}

	native := objc.ID(classNSImage).Send(sel_alloc).Send(sel_initWithSize,
		cocoa.CGSize{Width: float64(w), Height: float64(h)})
	native.Send(sel_addRepresentation, rep)

	cursor := objc.ID(classNSCursor).Send(sel_alloc).Send(
		sel_initWithImage_hotSpot,
		native,
		cocoa.NSPoint{X: float64(xhot), Y: float64(yhot)})

	native.Send(sel_release)
	rep.Send(sel_release)

	if cursor == 0 {
		return fmt.Errorf("glfw: failed to create custom cursor: %w", PlatformError)
	}

	c.platform.object = cursor
	return nil
}

func (c *Cursor) platformCreateStandardCursor(shape StandardCursor) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// Try private selectors for resize cursors first.
	var cursorSelector objc.SEL
	switch shape {
	case HResizeCursor:
		cursorSelector = objc.RegisterName("_windowResizeEastWestCursor")
	case VResizeCursor:
		cursorSelector = objc.RegisterName("_windowResizeNorthSouthCursor")
	case ResizeNWSECursor:
		cursorSelector = objc.RegisterName("_windowResizeNorthWestSouthEastCursor")
	case ResizeNESWCursor:
		cursorSelector = objc.RegisterName("_windowResizeNorthEastSouthWestCursor")
	}

	var cursor objc.ID
	if cursorSelector != 0 && objc.Send[bool](objc.ID(classNSCursor), sel_respondsToSelector, cursorSelector) {
		id := objc.ID(classNSCursor).Send(sel_performSelector, cursorSelector)
		if id != 0 && objc.Send[bool](id, sel_isKindOfClass, objc.ID(classNSCursor)) {
			cursor = id
		}
	}

	if cursor == 0 {
		switch shape {
		case ArrowCursor:
			cursor = objc.ID(classNSCursor).Send(sel_arrowCursor)
		case IBeamCursor:
			cursor = objc.ID(classNSCursor).Send(sel_IBeamCursor)
		case CrosshairCursor:
			cursor = objc.ID(classNSCursor).Send(sel_crosshairCursor)
		case HandCursor:
			cursor = objc.ID(classNSCursor).Send(sel_pointingHandCursor)
		case ResizeAllCursor:
			// Use the OS's resource: https://stackoverflow.com/a/21786835/5435443
			cursorName := cocoa.NSString_alloc().InitWithUTF8String("move")
			basePath := cocoa.NSString_alloc().InitWithUTF8String("/System/Library/Frameworks/ApplicationServices.framework/Versions/A/Frameworks/HIServices.framework/Versions/A/Resources/cursors")
			cursorPath := cocoa.NSString{ID: basePath.ID.Send(sel_stringByAppendingPathComponent, cursorName.ID)}
			cursorName.ID.Send(sel_release)
			basePath.ID.Send(sel_release)
			cursorPDF := cocoa.NSString_alloc().InitWithUTF8String("cursor.pdf")
			imagePath := cocoa.NSString{ID: cursorPath.ID.Send(sel_stringByAppendingPathComponent, cursorPDF.ID)}
			cursorPDF.ID.Send(sel_release)
			infoPlist := cocoa.NSString_alloc().InitWithUTF8String("info.plist")
			infoPath := cocoa.NSString{ID: cursorPath.ID.Send(sel_stringByAppendingPathComponent, infoPlist.ID)}
			infoPlist.ID.Send(sel_release)
			image := objc.ID(classNSImage).Send(sel_alloc).Send(sel_initByReferencingFile, imagePath.ID)
			info := objc.ID(classNSDictionary).Send(sel_dictionaryWithContentsOfFile, infoPath.ID)
			if image != 0 && info != 0 {
				hotxKey := cocoa.NSString_alloc().InitWithUTF8String("hotx")
				hotx := objc.Send[float64](info.Send(sel_valueForKey, hotxKey.ID), sel_doubleValue)
				hotxKey.ID.Send(sel_release)
				hotyKey := cocoa.NSString_alloc().InitWithUTF8String("hoty")
				hoty := objc.Send[float64](info.Send(sel_valueForKey, hotyKey.ID), sel_doubleValue)
				hotyKey.ID.Send(sel_release)
				cursor = objc.ID(classNSCursor).Send(sel_alloc).Send(sel_initWithImage_hotSpot, image, cocoa.NSPoint{X: hotx, Y: hoty})
			}
			if image != 0 {
				image.Send(sel_release)
			}
		case NotAllowedCursor:
			cursor = objc.ID(classNSCursor).Send(sel_operationNotAllowedCursor)
		}
	}

	if cursor == 0 {
		return fmt.Errorf("glfw: failed to create standard cursor: %w", PlatformError)
	}

	cursor.Send(sel_retain)
	c.platform.object = cursor
	return nil
}

func (c *Cursor) platformDestroyCursor() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if c.platform.object != 0 {
		c.platform.object.Send(sel_release)
		c.platform.object = 0
	}
	return nil
}

func (w *Window) platformSetCursor(cursor *Cursor) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if cursorInContentArea(w) {
		updateCursorImage(w)
	}
	return nil
}

// --- Clipboard ---

func platformSetClipboardString(str string) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	pasteboard := objc.ID(classNSPasteboard).Send(sel_generalPasteboard)
	types := objc.ID(classNSArray).Send(sel_arrayWithObject, nsPasteboardTypeString.ID)
	pasteboard.Send(sel_declareTypes_owner, types, 0)
	clipStr := cocoa.NSString_alloc().InitWithUTF8String(str)
	pasteboard.Send(sel_setString_forType,
		clipStr.ID,
		nsPasteboardTypeString.ID)
	clipStr.ID.Send(sel_release)
	return nil
}

func platformGetClipboardString() (string, error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	pasteboard := objc.ID(classNSPasteboard).Send(sel_generalPasteboard)

	types := pasteboard.Send(sel_types)
	if objc.Send[bool](types, sel_containsObject, nsPasteboardTypeString.ID) == false {
		return "", fmt.Errorf("glfw: failed to retrieve string from pasteboard: %w", FormatUnavailable)
	}

	strID := pasteboard.Send(sel_stringForType, nsPasteboardTypeString.ID)
	if strID == 0 {
		return "", fmt.Errorf("glfw: failed to retrieve object from pasteboard: %w", PlatformError)
	}

	str := cocoa.NSString{ID: strID}
	return str.String(), nil
}

// --- Key functions ---

func platformGetScancodeName(scancode int) (string, error) {
	if scancode < 0 || scancode > 0xff {
		return "", fmt.Errorf("glfw: invalid scancode %d: %w", scancode, InvalidValue)
	}
	key := _glfw.platformWindow.keycodes[scancode]
	if key == KeyUnknown {
		return "", nil
	}

	unicodeData := _glfw.platformWindow.unicodeData
	if unicodeData == 0 {
		return "", nil
	}

	const (
		kUCKeyActionDisplay          = 3
		kUCKeyTranslateNoDeadKeysBit = 0
	)

	var deadKeyState uint32
	var characters [4]uint16
	var characterCount int

	layoutPtr := cfDataGetBytePtr(unicodeData)
	if layoutPtr == 0 {
		return "", nil
	}

	if _glfw.platformWindow.tis.UCKeyTranslate(layoutPtr,
		uint16(scancode),
		kUCKeyActionDisplay,
		0,
		uint32(_glfw.platformWindow.tis.GetKbdType()),
		kUCKeyTranslateNoDeadKeysBit,
		&deadKeyState,
		len(characters),
		&characterCount,
		&characters[0]) != 0 {
		return "", nil
	}

	if characterCount == 0 {
		return "", nil
	}

	str := cfStringCreateWithCharacters(0, &characters[0], characterCount)
	if str == 0 {
		return "", nil
	}
	defer cfRelease(str)

	length := cfStringGetLength(str)
	size := cfStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8)
	buf := make([]byte, size+1)
	cfStringGetCString(str, &buf[0], size+1, kCFStringEncodingUTF8)

	// Find the null terminator.
	name := cStringToGoString(buf)
	_glfw.platformWindow.keynames[key] = name
	return _glfw.platformWindow.keynames[key], nil
}

func platformGetKeyScancode(key Key) int {
	return _glfw.platformWindow.scancodes[key]
}

func platformRawMouseMotionSupported() bool {
	return false
}

// --- Monitor acquire/release ---

func (w *Window) acquireMonitor() error {
	if w.monitor == nil {
		return nil
	}
	vm := &w.videoMode
	if err := w.monitor.setVideoModeNS(vm); err != nil {
		return err
	}
	bounds := cgDisplayBounds(w.monitor.platform.displayID)
	frame := cocoa.NSRect{
		Origin: cocoa.NSPoint{
			X: bounds.X,
			Y: float64(transformYNS(float32(bounds.Y + bounds.Height - 1))),
		},
		Size: cocoa.NSSize{Width: bounds.Width, Height: bounds.Height},
	}
	w.platform.object.Send(objc.RegisterName("setFrame:display:"), frame, true)

	w.monitor.inputMonitorWindow(w)
	return nil
}

func (w *Window) releaseMonitor() error {
	if w.monitor == nil {
		return nil
	}
	if w.monitor.window != w {
		return nil
	}
	w.monitor.inputMonitorWindow(nil)
	w.monitor.restoreVideoModeNS()
	return nil
}

// goStringFromCString converts a null-terminated C string pointer to a Go string.
func goStringFromCString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	p := (*byte)(unsafe.Pointer(ptr))
	var n int
	for {
		if *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(n))) == 0 {
			break
		}
		n++
	}
	return string(unsafe.Slice(p, n))
}

// Ensure reflect is used (needed for objc.RegisterClass field definitions).
var _ = reflect.TypeFor[uintptr]
