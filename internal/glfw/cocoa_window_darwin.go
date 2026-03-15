// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"fmt"
	"math"
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

// NSPasteboardType strings.
var (
	nsPasteboardTypeString  = cocoa.NSString_alloc().InitWithUTF8String("public.utf8-plain-text")
	nsPasteboardTypeFileURL = cocoa.NSString_alloc().InitWithUTF8String("public.file-url")
)

// NSDefaultRunLoopMode for event polling.
var nsDefaultRunLoopMode = cocoa.NSString_alloc().InitWithUTF8String("kCFRunLoopDefaultMode")

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

	pos := objc.Send[cocoa.NSPoint](window.platform.object, selMouseLocationOutsideOfEventStream)
	return objc.Send[bool](window.platform.view, selMouseInRect, pos, objc.Send[cocoa.NSRect](window.platform.view, selFrame))
}

// hideCursor hides the system cursor and optionally disables mouse/cursor association.
func hideCursor(window *Window) {
	if !_glfw.platformWindow.cursorHidden {
		objc.ID(classNSCursor).Send(selHideCursor)
		_glfw.platformWindow.cursorHidden = true
	}
}

// showCursor shows the system cursor and re-enables mouse/cursor association.
func showCursor(window *Window) {
	if _glfw.platformWindow.cursorHidden {
		objc.ID(classNSCursor).Send(selUnhideCursor)
		_glfw.platformWindow.cursorHidden = false
	}
}

// updateCursorImage sets the appropriate cursor image based on cursor mode.
func updateCursorImage(window *Window) {
	if window.cursorMode == CursorNormal {
		showCursor(window)
		if window.cursor != nil && window.cursor.platform.object != 0 {
			window.cursor.platform.object.Send(selSetCursor)
		} else {
			objc.ID(classNSCursor).Send(selArrowCursor).Send(selSetCursor)
		}
	} else if window.cursorMode == CursorHidden {
		hideCursor(window)
	} else if window.cursorMode == CursorDisabled {
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
	return objc.ID(classNSApplication).Send(selSharedApplication)
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

					if window.context.client != NoAPI {
						window.context.platform.object.Send(objc.RegisterName("update"))
					}

					if _glfw.platformWindow.disabledCursorWindow == window {
						_ = window.centerCursorInContentArea()
					}

					maximized := objc.Send[bool](window.platform.object, selIsZoomed)
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

					if window.context.client != NoAPI {
						window.context.platform.object.Send(objc.RegisterName("update"))
					}

					if _glfw.platformWindow.disabledCursorWindow == window {
						_ = window.centerCursorInContentArea()
					}

					frame := objc.Send[cocoa.NSRect](window.platform.object, selFrame)
					contentRect := objc.Send[cocoa.NSRect](window.platform.object, selContentRectForFrameRect, frame)
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
					state := uintptr(window.platform.object.Send(selOcclusionState))
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
					flags := uintptr(event.Send(selModifierFlags))
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
					flags := uintptr(event.Send(selModifierFlags))
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
					flags := uintptr(event.Send(selModifierFlags))
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
					flags := uintptr(event.Send(selModifierFlags))
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
					flags := uintptr(event.Send(selModifierFlags))
					button := MouseButton(int(event.Send(selButtonNumber)))
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
					flags := uintptr(event.Send(selModifierFlags))
					button := MouseButton(int(event.Send(selButtonNumber)))
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
					keyCode := uint16(event.Send(selKeyCode))
					key := translateKey(keyCode)
					flags := uintptr(event.Send(selModifierFlags))
					mods := translateFlags(flags)

					window.inputKey(key, int(keyCode), Press, mods)

					// Interpret key events for text input.
					eventArray := objc.ID(classNSArray).Send(selArrayWithObject, event)
					self.Send(selInterpretKeyEvents, eventArray)
				},
			},
			{
				Cmd: objc.RegisterName("keyUp:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					keyCode := uint16(event.Send(selKeyCode))
					key := translateKey(keyCode)
					flags := uintptr(event.Send(selModifierFlags))
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
					keyCode := uint16(event.Send(selKeyCode))
					key := translateKey(keyCode)
					if key == KeyUnknown {
						return
					}
					flags := uintptr(event.Send(selModifierFlags)) & NSEventModifierFlagDeviceIndependentFlagsMask
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
					deltaX := objc.Send[float64](event, selScrollingDeltaX)
					deltaY := objc.Send[float64](event, selScrollingDeltaY)

					if objc.Send[bool](event, selHasPreciseScrollingDeltas) {
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

					if window.platform.retina && window.platform.layer != 0 {
						scale := objc.Send[float64](window.platform.object.Send(selScreen), selBackingScaleFactor)
						window.platform.layer.Send(objc.RegisterName("setContentsScale:"), scale)
					}

					updateWindowSize(window)

					screen := window.platform.object.Send(selScreen)
					if screen != 0 {
						scale := objc.Send[float64](screen, selBackingScaleFactor)
						xscale := float32(scale)
						yscale := float32(scale)
						if xscale != window.platform.xscale || yscale != window.platform.yscale {
							window.platform.xscale = xscale
							window.platform.yscale = yscale
							window.inputWindowContentScale(xscale, yscale)
						}
					}
				},
			},
			{
				Cmd: objc.RegisterName("updateTrackingAreas"),
				Fn: func(self objc.ID, _ objc.SEL) {
					// Remove all existing tracking areas.
					areas := self.Send(selTrackingAreas)
					areaCount := int(areas.Send(selCount))
					for i := range areaCount {
						area := areas.Send(selObjectAtIndex, i)
						self.Send(selRemoveTrackingArea, area)
					}

					// Create new tracking area.
					bounds := objc.Send[cocoa.NSRect](self, selBounds)
					options := uintptr(NSTrackingMouseEnteredAndExited |
						NSTrackingActiveInKeyWindow |
						NSTrackingEnabledDuringMouseDrag |
						NSTrackingCursorUpdate |
						NSTrackingInVisibleRect |
						NSTrackingAssumeInside)

					trackingArea := objc.ID(classNSTrackingArea).Send(selAlloc).Send(
						selInitWithRectOptionsOwnerUserInfo,
						bounds, options, self, 0)
					self.Send(selAddTrackingArea, trackingArea)

					// Call super.
					self.SendSuper(objc.RegisterName("updateTrackingAreas"))
				},
			},
			{
				Cmd: objc.RegisterName("canBecomeKeyView"),
				Fn: func(_ objc.ID, _ objc.SEL) bool {
					return true
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
				Cmd: selHasMarkedText,
				Fn: func(_ objc.ID, _ objc.SEL) bool {
					return false
				},
			},
			{
				Cmd: selMarkedRange,
				Fn: func(_ objc.ID, _ objc.SEL) nsRange {
					return nsRange{Location: ^uintptr(0), Length: 0} // NSNotFound
				},
			},
			{
				Cmd: selSelectedRange,
				Fn: func(_ objc.ID, _ objc.SEL) nsRange {
					return nsRange{Location: ^uintptr(0), Length: 0} // NSNotFound
				},
			},
			{
				Cmd: selSetMarkedText,
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID, _ nsRange, _ nsRange) {
				},
			},
			{
				Cmd: selUnmarkText,
				Fn: func(_ objc.ID, _ objc.SEL) {
				},
			},
			{
				Cmd: selValidAttributesForMarkedText,
				Fn: func(_ objc.ID, _ objc.SEL) objc.ID {
					// Return an empty NSArray.
					return objc.ID(classNSArray).Send(selAlloc).Send(selInit)
				},
			},
			{
				Cmd: selAttributedSubstringForProposedRange,
				Fn: func(_ objc.ID, _ objc.SEL, _ nsRange, _ uintptr) objc.ID {
					return 0 // nil
				},
			},
			{
				Cmd: selInsertText,
				Fn: func(self objc.ID, _ objc.SEL, text objc.ID, _ nsRange) {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return
					}
					nsApp := objc.ID(classNSApplication).Send(selNSApp)
					event := nsApp.Send(selCurrentEvent)
					flags := uintptr(objc.Send[uint64](event, selModifierFlags))
					mods := translateFlags(flags)
					plain := mods&ModSuper == 0

					// Get the string from the text object.
					str := cocoa.NSString{ID: text}
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
				Cmd: selCharacterIndexForPoint,
				Fn: func(_ objc.ID, _ objc.SEL, _ cocoa.NSPoint) uintptr {
					return 0
				},
			},
			{
				Cmd: selFirstRectForCharacterRange,
				Fn: func(self objc.ID, _ objc.SEL, _ nsRange, _ uintptr) cocoa.NSRect {
					window := getGoWindow(classGLFWContentView, self)
					if window == nil {
						return cocoa.NSRect{}
					}
					frame := objc.Send[cocoa.NSRect](window.platform.view, selFrame)
					return cocoa.NSRect{
						Origin: frame.Origin,
						Size:   cocoa.CGSize{Width: 0, Height: 0},
					}
				},
			},
			{
				Cmd: selDoCommandBySelector,
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
					contentRect := objc.Send[cocoa.NSRect](window.platform.view, selFrame)
					pos := objc.Send[cocoa.NSPoint](sender, objc.RegisterName("draggingLocation"))
					window.inputCursorPos(pos.X, contentRect.Size.Height-pos.Y)

					pasteboard := sender.Send(selDraggingPasteboard)
					urlClass := objc.ID(classNSURL)
					classes := objc.ID(classNSArray).Send(selArrayWithObject, urlClass)

					// Filter to file URLs only.
					fileURLsOnlyKey := cocoa.NSString_alloc().InitWithUTF8String("NSPasteboardURLReadingFileURLsOnlyKey")
					defer fileURLsOnlyKey.ID.Send(selRelease)
					nsYes := objc.ID(objc.GetClass("NSNumber")).Send(objc.RegisterName("numberWithBool:"), true)
					options := objc.ID(objc.GetClass("NSDictionary")).Send(
						objc.RegisterName("dictionaryWithObject:forKey:"),
						uintptr(nsYes), uintptr(fileURLsOnlyKey.ID))

					urls := pasteboard.Send(selReadObjectsForClasses, classes, uintptr(options))
					if urls == 0 {
						return false
					}

					urlCount := int(urls.Send(selCount))
					if urlCount == 0 {
						return false
					}

					paths := make([]string, urlCount)
					for i := range urlCount {
						url := urls.Send(selObjectAtIndex, i)
						pathStr := cocoa.NSString{ID: url.Send(selPath)}
						paths[i] = pathStr.String()
					}

					window.inputDrop(paths)
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
		dx := objc.Send[float64](event, selDeltaX)
		dy := objc.Send[float64](event, selDeltaY)

		dx -= window.platform.cursorWarpDeltaX
		dy -= window.platform.cursorWarpDeltaY

		window.inputCursorPos(
			window.virtualCursorPosX+dx,
			window.virtualCursorPosY+dy)
	} else {
		// Get the location in the content view.
		pos := objc.Send[cocoa.NSPoint](event, selLocationInWindow)

		// Convert from Cocoa coordinates (origin at bottom-left) to GLFW coordinates (origin at top-left).
		contentRect := objc.Send[cocoa.NSRect](window.platform.view, selFrame)
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

	contentRect := objc.Send[cocoa.NSRect](window.platform.view, selFrame)
	fbRect := objc.Send[cocoa.NSRect](window.platform.view, selConvertRectToBacking, contentRect)

	width := int(contentRect.Size.Width)
	height := int(contentRect.Size.Height)

	if width != window.platform.width || height != window.platform.height {
		window.platform.width = width
		window.platform.height = height
		window.inputWindowSize(width, height)
	}

	fbWidth := int(fbRect.Size.Width)
	fbHeight := int(fbRect.Size.Height)

	if fbWidth != window.platform.fbWidth || fbHeight != window.platform.fbHeight {
		window.platform.fbWidth = fbWidth
		window.platform.fbHeight = fbHeight
		window.inputFramebufferSize(fbWidth, fbHeight)
	}
}

// createNativeWindow creates the actual NSWindow, delegate, and content view.
func createNativeWindow(window *Window, wndconfig *wndconfig, fbconfig_ *fbconfig) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

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
	nsWindow := objc.ID(classGLFWWindow).Send(selAlloc).Send(selInitWithContentRect,
		contentRect, styleMask, uintptr(NSBackingStoreBuffered), false)
	if nsWindow == 0 {
		return fmt.Errorf("glfw: failed to create Cocoa window: %w", PlatformError)
	}

	window.platform.object = nsWindow

	if window.monitor != nil {
		nsWindow.Send(selSetLevel, uintptr(NSMainMenuWindowLevel+1))
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
			nsWindow.Send(selSetCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenPrimary|_NSWindowCollectionBehaviorManaged))
		} else {
			nsWindow.Send(selSetCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenNone))
		}

		if wndconfig.floating {
			nsWindow.Send(selSetLevel, uintptr(NSFloatingWindowLevel))
		}

		if wndconfig.maximized {
			nsWindow.Send(selZoom, 0)
		}
	}

	nsWindow.Send(selSetTitle, cocoa.NSString_alloc().InitWithUTF8String(wndconfig.title).ID)
	nsWindow.Send(selSetRestorable, false)

	// Disable window tabbing (macOS 10.12+).
	selSetTabbingMode := objc.RegisterName("setTabbingMode:")
	if nsWindow.Send(objc.RegisterName("respondsToSelector:"), selSetTabbingMode) != 0 {
		nsWindow.Send(selSetTabbingMode, uintptr(2)) // NSWindowTabbingModeDisallowed = 2
	}

	// Create the delegate.
	delegateID := objc.ID(classGLFWWindowDelegate).Send(selAlloc).Send(selInit)
	setGoWindow(delegateID, window)
	window.platform.delegate = delegateID
	nsWindow.Send(selSetDelegate, delegateID)

	// Create the content view.
	viewID := objc.ID(classGLFWContentView).Send(selAlloc).Send(selInit)
	setGoWindow(viewID, window)
	window.platform.view = viewID
	nsWindow.Send(selSetContentView, viewID)
	nsWindow.Send(selMakeFirstResponder, viewID)

	// Register for dragged types (file URLs).
	fileURLType := cocoa.NSString_alloc().InitWithUTF8String("public.file-url")
	typesArray := objc.ID(classNSArray).Send(selArrayWithObject, fileURLType.ID)
	viewID.Send(selRegisterForDraggedTypes, typesArray)

	// Set up retina/HiDPI support.
	screen := nsWindow.Send(selScreen)
	window.platform.retina = wndconfig.retina
	if screen != 0 {
		scale := objc.Send[float64](screen, selBackingScaleFactor)
		window.platform.xscale = float32(scale)
		window.platform.yscale = float32(scale)
	} else {
		window.platform.xscale = 1.0
		window.platform.yscale = 1.0
	}

	// Handle transparent framebuffer.
	if fbconfig_.transparent {
		nsWindow.Send(selSetOpaque, false)
		nsWindow.Send(selSetHasShadow, false)
		nsWindow.Send(selSetBackgroundColor, objc.ID(classNSColor).Send(selClearColor))
	}

	// Update initial size cache.
	contentViewRect := objc.Send[cocoa.NSRect](viewID, selFrame)
	window.platform.width = int(contentViewRect.Size.Width)
	window.platform.height = int(contentViewRect.Size.Height)

	fbRect := objc.Send[cocoa.NSRect](viewID, selConvertRectToBacking, contentViewRect)
	window.platform.fbWidth = int(fbRect.Size.Width)
	window.platform.fbHeight = int(fbRect.Size.Height)

	// Accept mouse-moved events.
	nsWindow.Send(objc.RegisterName("setAcceptsMouseMovedEvents:"), true)

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
		w.platform.object.Send(selOrderOut, 0)
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
		w.platform.object.Send(selSetDelegate, 0)
		w.platform.delegate.Send(selRelease)
		w.platform.delegate = 0
	}

	if w.platform.view != 0 {
		w.platform.view.Send(selRelease)
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
	w.platform.object.Send(selSetTitle, s.ID)
	// HACK: Set the miniwindow title explicitly as setTitle: doesn't update it
	//       if the window lacks NSWindowStyleMaskTitled
	w.platform.object.Send(objc.RegisterName("setMiniwindowTitle:"), s.ID)
	return nil
}

func (w *Window) platformSetWindowIcon(images []*Image) error {
	// macOS does not support per-window icons. The dock icon is set at the application level.
	return nil
}

func (w *Window) platformGetWindowPos() (xpos, ypos int, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	frame := objc.Send[cocoa.NSRect](w.platform.object, selFrame)
	contentRect := objc.Send[cocoa.NSRect](w.platform.object, selContentRectForFrameRect, frame)

	xpos = int(contentRect.Origin.X)
	ypos = int(transformYNS(float32(contentRect.Origin.Y + contentRect.Size.Height - 1)))
	return xpos, ypos, nil
}

func (w *Window) platformSetWindowPos(xpos, ypos int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	viewFrame := objc.Send[cocoa.NSRect](w.platform.view, selFrame)
	dummyRect := cocoa.NSRect{
		Origin: cocoa.NSPoint{
			X: float64(xpos),
			Y: float64(transformYNS(float32(float64(ypos) + viewFrame.Size.Height - 1))),
		},
		Size: cocoa.NSSize{Width: 0, Height: 0},
	}

	frameRect := objc.Send[cocoa.NSRect](w.platform.object, selFrameRectForContentRect, dummyRect)
	w.platform.object.Send(selSetFrameOrigin, frameRect.Origin)
	return nil
}

func (w *Window) platformGetWindowSize() (width, height int, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	contentRect := objc.Send[cocoa.NSRect](w.platform.view, selFrame)
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

	contentRect := objc.Send[cocoa.NSRect](w.platform.object, selContentRectForFrameRect,
		objc.Send[cocoa.NSRect](w.platform.object, selFrame))
	contentRect.Origin.Y += contentRect.Size.Height - float64(height)
	contentRect.Size = cocoa.NSSize{Width: float64(width), Height: float64(height)}
	frameRect := objc.Send[cocoa.NSRect](w.platform.object, selFrameRectForContentRect, contentRect)
	w.platform.object.Send(objc.RegisterName("setFrame:display:"), frameRect, true)
	return nil
}

func (w *Window) platformSetWindowSizeLimits(minwidth, minheight, maxwidth, maxheight int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if minwidth == DontCare || minheight == DontCare {
		w.platform.object.Send(selSetContentMinSize, cocoa.NSSize{Width: 0, Height: 0})
	} else {
		w.platform.object.Send(selSetContentMinSize, cocoa.NSSize{Width: float64(minwidth), Height: float64(minheight)})
	}

	if maxwidth == DontCare || maxheight == DontCare {
		w.platform.object.Send(selSetContentMaxSize, cocoa.NSSize{Width: math.MaxFloat64, Height: math.MaxFloat64})
	} else {
		w.platform.object.Send(selSetContentMaxSize, cocoa.NSSize{Width: float64(maxwidth), Height: float64(maxheight)})
	}

	return nil
}

func (w *Window) platformSetWindowAspectRatio(numer, denom int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if numer == DontCare || denom == DontCare {
		w.platform.object.Send(selSetResizeIncrements, cocoa.NSSize{Width: 1, Height: 1})
	} else {
		w.platform.object.Send(selSetContentAspectRatio, cocoa.NSSize{Width: float64(numer), Height: float64(denom)})
	}
	return nil
}

func (w *Window) platformGetFramebufferSize() (width, height int, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	contentRect := objc.Send[cocoa.NSRect](w.platform.view, selFrame)
	fbRect := objc.Send[cocoa.NSRect](w.platform.view, selConvertRectToBacking, contentRect)
	return int(fbRect.Size.Width), int(fbRect.Size.Height), nil
}

func (w *Window) platformGetWindowFrameSize() (left, top, right, bottom int, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	contentRect := objc.Send[cocoa.NSRect](w.platform.view, selFrame)
	frameRect := objc.Send[cocoa.NSRect](w.platform.object, selFrame)

	contentRectInWindow := objc.Send[cocoa.NSRect](w.platform.object, selContentRectForFrameRect, frameRect)

	left = int(contentRectInWindow.Origin.X - frameRect.Origin.X)
	top = int((frameRect.Origin.Y + frameRect.Size.Height) - (contentRectInWindow.Origin.Y + contentRectInWindow.Size.Height))
	right = int((frameRect.Origin.X + frameRect.Size.Width) - (contentRectInWindow.Origin.X + contentRect.Size.Width))
	bottom = int(contentRectInWindow.Origin.Y - frameRect.Origin.Y)
	return left, top, right, bottom, nil
}

func (w *Window) platformGetWindowContentScale() (xscale, yscale float32, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	screen := w.platform.object.Send(selScreen)
	if screen == 0 {
		screen = objc.ID(classNSScreen).Send(selMainScreen)
	}

	scale := objc.Send[float64](screen, selBackingScaleFactor)
	return float32(scale), float32(scale), nil
}

func (w *Window) platformIconifyWindow() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(selMiniaturize, 0)
}

func (w *Window) platformRestoreWindow() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if objc.Send[bool](w.platform.object, objc.RegisterName("isMiniaturized")) {
		w.platform.object.Send(selDeminiaturize, 0)
	} else if objc.Send[bool](w.platform.object, selIsZoomed) {
		w.platform.object.Send(selZoom, 0)
	}
}

func (w *Window) platformMaximizeWindow() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if !objc.Send[bool](w.platform.object, selIsZoomed) {
		w.platform.object.Send(selZoom, 0)
	}
	return nil
}

func (w *Window) platformShowWindow() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(selOrderFront, 0)
}

func (w *Window) platformHideWindow() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(selOrderOut, 0)
}

func (w *Window) platformRequestWindowAttention() {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// NSInformationalRequest = 10
	nsApp().Send(selRequestUserAttention, uintptr(10))
}

func (w *Window) platformFocusWindow() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	nsApp().Send(selActivateIgnoringOtherApps, true)
	w.platform.object.Send(selMakeKeyAndOrderFront, 0)
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
			frameRect := objc.Send[cocoa.NSRect](w.platform.object, selFrameRectForContentRect, contentRect)
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

	styleMask := uintptr(objc.Send[uint64](w.platform.object, selStyleMask))

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

	w.platform.object.Send(selSetStyleMask, styleMask)
	// HACK: Changing the style mask can cause the first responder to be cleared
	w.platform.object.Send(selMakeFirstResponder, w.platform.view)

	if w.monitor != nil {
		w.platform.object.Send(selSetLevel, uintptr(NSMainMenuWindowLevel+1))
		w.platform.object.Send(selSetHasShadow, false)

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
		frameRect := objc.Send[cocoa.NSRect](w.platform.object, selFrameRectForContentRect, contentRect)
		w.platform.object.Send(objc.RegisterName("setFrame:display:"), frameRect, true)

		if w.numer != DontCare && w.denom != DontCare {
			w.platform.object.Send(selSetContentAspectRatio, cocoa.NSSize{Width: float64(w.numer), Height: float64(w.denom)})
		}

		if w.minwidth != DontCare && w.minheight != DontCare {
			w.platform.object.Send(selSetContentMinSize, cocoa.NSSize{Width: float64(w.minwidth), Height: float64(w.minheight)})
		}

		if w.maxwidth != DontCare && w.maxheight != DontCare {
			w.platform.object.Send(selSetContentMaxSize, cocoa.NSSize{Width: float64(w.maxwidth), Height: float64(w.maxheight)})
		}

		if w.floating {
			w.platform.object.Send(selSetLevel, uintptr(NSFloatingWindowLevel))
		} else {
			w.platform.object.Send(selSetLevel, uintptr(NSNormalWindowLevel))
		}

		if w.resizable {
			w.platform.object.Send(selSetCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenPrimary|_NSWindowCollectionBehaviorManaged))
		} else {
			w.platform.object.Send(selSetCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenNone))
		}

		w.platform.object.Send(selSetHasShadow, true)
		// HACK: Clearing NSWindowStyleMaskTitled resets and disables the window
		//       title property but the miniwindow title property is unaffected
		miniTitle := w.platform.object.Send(selMiniwindowTitle)
		w.platform.object.Send(selSetTitle, miniTitle)
	}

	return nil
}

func (w *Window) platformWindowFocused() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return objc.Send[bool](w.platform.object, selIsKeyWindow)
}

func (w *Window) platformWindowIconified() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return objc.Send[bool](w.platform.object, selIsMiniaturized)
}

func (w *Window) platformWindowVisible() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return objc.Send[bool](w.platform.object, selIsVisible)
}

func (w *Window) platformWindowMaximized() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if w.resizable {
		return objc.Send[bool](w.platform.object, selIsZoomed)
	}
	return false
}

func (w *Window) platformWindowHovered() (bool, error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	pos := objc.Send[cocoa.NSPoint](objc.ID(classNSEvent), objc.RegisterName("mouseLocation"))

	// Check if this window is the topmost window at the cursor position.
	topWindowNumber := objc.Send[uintptr](objc.ID(classNSWindow), objc.RegisterName("windowNumberAtPoint:belowWindowWithWindowNumber:"), pos, uintptr(0))
	if topWindowNumber != uintptr(w.platform.object.Send(selWindowNumber)) {
		return false, nil
	}

	viewFrame := objc.Send[cocoa.NSRect](w.platform.view, selFrame)
	screenRect := objc.Send[cocoa.NSRect](w.platform.object, selConvertRectToScreen, viewFrame)
	return pos.X >= screenRect.Origin.X &&
		pos.X < screenRect.Origin.X+screenRect.Size.Width &&
		pos.Y >= screenRect.Origin.Y &&
		pos.Y < screenRect.Origin.Y+screenRect.Size.Height, nil
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
		w.platform.object.Send(selSetCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenPrimary|_NSWindowCollectionBehaviorManaged))
	} else {
		mask &^= NSWindowStyleMaskResizable
		w.platform.object.Send(selSetCollectionBehavior, uintptr(_NSWindowCollectionBehaviorFullScreenNone))
	}
	w.platform.object.Send(objc.RegisterName("setStyleMask:"), mask)
	return nil
}

func (w *Window) platformSetWindowDecorated(enabled bool) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	mask := uintptr(w.platform.object.Send(selStyleMask))
	if enabled {
		mask |= NSWindowStyleMaskTitled | NSWindowStyleMaskClosable
		mask &^= NSWindowStyleMaskBorderless
	} else {
		mask |= NSWindowStyleMaskBorderless
		mask &^= (NSWindowStyleMaskTitled | NSWindowStyleMaskClosable)
	}
	w.platform.object.Send(selSetStyleMask, mask)
	w.platform.object.Send(selMakeFirstResponder, w.platform.view)
	return nil
}

func (w *Window) platformSetWindowFloating(enabled bool) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if enabled {
		w.platform.object.Send(selSetLevel, uintptr(NSFloatingWindowLevel))
	} else {
		w.platform.object.Send(selSetLevel, uintptr(NSNormalWindowLevel))
	}
	return nil
}

func (w *Window) platformSetWindowMousePassthrough(enabled bool) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(selSetIgnoresMouseEvents, enabled)
	return nil
}

func (w *Window) platformGetWindowOpacity() (float32, error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return objc.Send[float32](w.platform.object, selAlphaValue), nil
}

func (w *Window) platformSetWindowOpacity(opacity float32) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(selSetAlphaValue, float64(opacity))
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

	for {
		event := objc.Send[objc.ID](nsApp(), selNextEventMatchingMask,
			uintptr(NSEventMaskAny),
			uintptr(0), // distantPast (nil = no wait)
			nsDefaultRunLoopMode.ID,
			true)
		if event == 0 {
			break
		}
		nsApp().Send(selSendEvent, event)
	}
	return nil
}

func platformWaitEvents() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// Wait for an event with no timeout (distantFuture).
	distantFuture := objc.ID(objc.GetClass("NSDate")).Send(objc.RegisterName("distantFuture"))
	event := objc.Send[objc.ID](nsApp(), selNextEventMatchingMask,
		uintptr(NSEventMaskAny),
		distantFuture,
		nsDefaultRunLoopMode.ID,
		true)
	if event != 0 {
		nsApp().Send(selSendEvent, event)
	}

	// Process remaining events.
	for {
		event := objc.Send[objc.ID](nsApp(), selNextEventMatchingMask,
			uintptr(NSEventMaskAny),
			uintptr(0),
			nsDefaultRunLoopMode.ID,
			true)
		if event == 0 {
			break
		}
		nsApp().Send(selSendEvent, event)
	}
	return nil
}

func platformWaitEventsTimeout(timeout float64) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// Create an NSDate for the timeout.
	date := objc.ID(objc.GetClass("NSDate")).Send(objc.RegisterName("dateWithTimeIntervalSinceNow:"), timeout)
	event := objc.Send[objc.ID](nsApp(), selNextEventMatchingMask,
		uintptr(NSEventMaskAny),
		date,
		nsDefaultRunLoopMode.ID,
		true)
	if event != 0 {
		nsApp().Send(selSendEvent, event)
	}

	// Process remaining events.
	for {
		event := objc.Send[objc.ID](nsApp(), selNextEventMatchingMask,
			uintptr(NSEventMaskAny),
			uintptr(0),
			nsDefaultRunLoopMode.ID,
			true)
		if event == 0 {
			break
		}
		nsApp().Send(selSendEvent, event)
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

	contentRect := objc.Send[cocoa.NSRect](w.platform.view, selFrame)
	// NOTE: The returned location uses base 0,1 not 0,0
	pos := objc.Send[cocoa.NSPoint](w.platform.object, selMouseLocationOutsideOfEventStream)

	xpos = pos.X
	ypos = contentRect.Size.Height - pos.Y

	return xpos, ypos, nil
}

func (w *Window) platformSetCursorPos(xpos, ypos float64) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	updateCursorImage(w)

	contentRect := objc.Send[cocoa.NSRect](w.platform.view, selFrame)
	// NOTE: The returned location uses base 0,1 not 0,0
	pos := objc.Send[cocoa.NSPoint](w.platform.object, selMouseLocationOutsideOfEventStream)

	w.platform.cursorWarpDeltaX += xpos - pos.X
	w.platform.cursorWarpDeltaY += ypos - contentRect.Size.Height + pos.Y

	if w.monitor != nil {
		cgDisplayMoveCursorToPoint(w.monitor.platform.displayID, cocoa.CGPoint{X: xpos, Y: ypos})
	} else {
		localRect := cocoa.NSRect{
			Origin: cocoa.NSPoint{X: xpos, Y: contentRect.Size.Height - ypos - 1},
			Size:   cocoa.NSSize{Width: 0, Height: 0},
		}
		globalRect := objc.Send[cocoa.NSRect](w.platform.object, selConvertRectToScreen, localRect)
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

func (c *Cursor) platformCreateStandardCursor(shape StandardCursor) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	var cursor objc.ID
	switch shape {
	case ArrowCursor:
		cursor = objc.ID(classNSCursor).Send(selArrowCursor)
	case IBeamCursor:
		cursor = objc.ID(classNSCursor).Send(selIBeamCursor)
	case CrosshairCursor:
		cursor = objc.ID(classNSCursor).Send(selCrosshairCursor)
	case HandCursor:
		cursor = objc.ID(classNSCursor).Send(selPointingHandCursor)
	case HResizeCursor:
		cursor = objc.ID(classNSCursor).Send(selResizeLeftRightCursor)
	case VResizeCursor:
		cursor = objc.ID(classNSCursor).Send(selResizeUpDownCursor)
	case ResizeNWSECursor:
		// macOS doesn't have a specific NWSE cursor; use closed hand.
		cursor = objc.ID(classNSCursor).Send(selClosedHandCursor)
	case ResizeNESWCursor:
		// macOS doesn't have a specific NESW cursor; use closed hand.
		cursor = objc.ID(classNSCursor).Send(selClosedHandCursor)
	case ResizeAllCursor:
		cursor = objc.ID(classNSCursor).Send(selOpenHandCursor)
	case NotAllowedCursor:
		cursor = objc.ID(classNSCursor).Send(selOperationNotAllowedCursor)
	default:
		return fmt.Errorf("glfw: invalid standard cursor 0x%08X: %w", shape, InvalidEnum)
	}

	if cursor == 0 {
		return fmt.Errorf("glfw: failed to create standard cursor: %w", PlatformError)
	}

	cursor.Send(selRetain)
	c.platform.object = cursor
	return nil
}

func (c *Cursor) platformDestroyCursor() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if c.platform.object != 0 {
		c.platform.object.Send(selRelease)
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

	pasteboard := objc.ID(classNSPasteboard).Send(selGeneralPasteboard)
	types := objc.ID(classNSArray).Send(selArrayWithObject, nsPasteboardTypeString.ID)
	pasteboard.Send(selDeclareTypes, types, 0)
	pasteboard.Send(selSetStringForType,
		cocoa.NSString_alloc().InitWithUTF8String(str).ID,
		nsPasteboardTypeString.ID)
	return nil
}

func platformGetClipboardString() (string, error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	pasteboard := objc.ID(classNSPasteboard).Send(selGeneralPasteboard)
	strID := pasteboard.Send(selStringForType, nsPasteboardTypeString.ID)
	if strID == 0 {
		return "", fmt.Errorf("glfw: pasteboard doesn't contain a string: %w", FormatUnavailable)
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

// Ensure reflect is used (needed for objc.RegisterClass field definitions).
var _ = reflect.TypeFor[uintptr]
