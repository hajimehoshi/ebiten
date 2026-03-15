// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"fmt"
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
	class_GLFWWindow         objc.Class
	class_GLFWWindowDelegate objc.Class
	class_GLFWContentView    objc.Class
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
	if flags&NSEventModifierFlagNumericPad != 0 {
		mods |= ModNumLock
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

	// Get cursor location in screen coordinates.
	pos := objc.Send[cocoa.NSPoint](objc.ID(class_NSEvent), objc.RegisterName("mouseLocation"))

	// Convert to window coordinates.
	windowFrame := objc.Send[cocoa.NSRect](window.platform.object, sel_frame)
	contentRect := objc.Send[cocoa.NSRect](window.platform.object, sel_contentRectForFrameRect, windowFrame)

	return pos.X >= contentRect.Origin.X &&
		pos.X < contentRect.Origin.X+contentRect.Size.Width &&
		pos.Y >= contentRect.Origin.Y &&
		pos.Y < contentRect.Origin.Y+contentRect.Size.Height
}

// hideCursor hides the system cursor and optionally disables mouse/cursor association.
func hideCursor(window *Window) {
	if !_glfw.platformWindow.cursorHidden {
		objc.ID(class_NSCursor).Send(sel_hide_cursor)
		_glfw.platformWindow.cursorHidden = true
	}
}

// showCursor shows the system cursor and re-enables mouse/cursor association.
func showCursor(window *Window) {
	if _glfw.platformWindow.cursorHidden {
		objc.ID(class_NSCursor).Send(sel_unhide_cursor)
		_glfw.platformWindow.cursorHidden = false
	}
}

// updateCursorImage sets the appropriate cursor image based on cursor mode.
func updateCursorImage(window *Window) {
	if window.cursorMode == CursorNormal {
		showCursor(window)
		if window.cursor != nil && window.cursor.platform.object != 0 {
			window.cursor.platform.object.Send(sel_set_cursor)
		} else {
			objc.ID(class_NSCursor).Send(sel_arrowCursor).Send(sel_set_cursor)
		}
	} else if window.cursorMode == CursorHidden {
		hideCursor(window)
	} else if window.cursorMode == CursorDisabled {
		hideCursor(window)
	}
}

// updateCursorMode applies cursor mode changes.
func updateCursorMode(window *Window) {
	if window.cursorMode == CursorDisabled {
		_glfw.platformWindow.disabledCursorWindow = window
		cgAssociateMouseAndMouseCursorPosition(0)
		hideCursor(window)
	} else if _glfw.platformWindow.disabledCursorWindow == window {
		_glfw.platformWindow.disabledCursorWindow = nil
		cgAssociateMouseAndMouseCursorPosition(1)
		updateCursorImage(window)
	} else {
		updateCursorImage(window)
	}
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
	return objc.ID(class_NSApplication).Send(sel_sharedApplication)
}

// registerGLFWClasses registers the GLFWWindow, GLFWWindowDelegate, and GLFWContentView
// ObjC classes. Called from platformInit.
func registerGLFWClasses() error {
	// GLFWWindow — NSWindow subclass.
	var err error
	class_GLFWWindow, err = objc.RegisterClass(
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
	class_GLFWWindowDelegate, err = objc.RegisterClass(
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
					window := getGoWindow(class_GLFWWindowDelegate, self)
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
					window := getGoWindow(class_GLFWWindowDelegate, self)
					if window == nil {
						return
					}
					updateWindowSize(window)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidMove:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(class_GLFWWindowDelegate, self)
					if window == nil {
						return
					}
					if window.platform.object == 0 {
						return
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
					window := getGoWindow(class_GLFWWindowDelegate, self)
					if window == nil {
						return
					}
					window.inputWindowIconify(true)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidDeminiaturize:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(class_GLFWWindowDelegate, self)
					if window == nil {
						return
					}
					window.inputWindowIconify(false)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidBecomeKey:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(class_GLFWWindowDelegate, self)
					if window == nil {
						return
					}
					window.inputWindowFocus(true)
					updateCursorMode(window)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidResignKey:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(class_GLFWWindowDelegate, self)
					if window == nil {
						return
					}
					window.inputWindowFocus(false)
					showCursor(window)
				},
			},
			{
				Cmd: objc.RegisterName("windowDidChangeOcclusionState:"),
				Fn: func(self objc.ID, _ objc.SEL, notification objc.ID) {
					window := getGoWindow(class_GLFWWindowDelegate, self)
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
	class_GLFWContentView, err = objc.RegisterClass(
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
					window := getGoWindow(class_GLFWContentView, self)
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
					window := getGoWindow(class_GLFWContentView, self)
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
					window := getGoWindow(class_GLFWContentView, self)
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
					window := getGoWindow(class_GLFWContentView, self)
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
					window := getGoWindow(class_GLFWContentView, self)
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
					window := getGoWindow(class_GLFWContentView, self)
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
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return
					}
					handleMouseMoved(window, event)
				},
			},
			{
				Cmd: objc.RegisterName("mouseDragged:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return
					}
					handleMouseMoved(window, event)
				},
			},
			{
				Cmd: objc.RegisterName("rightMouseDragged:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return
					}
					handleMouseMoved(window, event)
				},
			},
			{
				Cmd: objc.RegisterName("otherMouseDragged:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return
					}
					handleMouseMoved(window, event)
				},
			},
			{
				Cmd: objc.RegisterName("mouseExited:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return
					}
					window.inputCursorEnter(false)
				},
			},
			{
				Cmd: objc.RegisterName("mouseEntered:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return
					}
					window.inputCursorEnter(true)
				},
			},
			// Keyboard events.
			{
				Cmd: objc.RegisterName("keyDown:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return
					}
					keyCode := uint16(event.Send(sel_keyCode))
					key := translateKey(keyCode)
					flags := uintptr(event.Send(sel_modifierFlags))
					mods := translateFlags(flags)

					window.inputKey(key, int(keyCode), Press, mods)

					// Interpret key events for text input.
					eventArray := objc.ID(class_NSArray).Send(sel_arrayWithObject, event)
					self.Send(sel_interpretKeyEvents, eventArray)
				},
			},
			{
				Cmd: objc.RegisterName("keyUp:"),
				Fn: func(self objc.ID, _ objc.SEL, event objc.ID) {
					window := getGoWindow(class_GLFWContentView, self)
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
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return
					}
					keyCode := uint16(event.Send(sel_keyCode))
					key := translateKey(keyCode)
					if key == KeyUnknown {
						return
					}
					flags := uintptr(event.Send(sel_modifierFlags))
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
					window := getGoWindow(class_GLFWContentView, self)
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
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return
					}
					updateWindowSize(window)

					screen := window.platform.object.Send(sel_screen)
					if screen != 0 {
						scale := objc.Send[float64](screen, sel_backingScaleFactor)
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
						NSTrackingCursorUpdate |
						NSTrackingInVisibleRect |
						NSTrackingAssumeInside)

					trackingArea := objc.ID(class_NSTrackingArea).Send(sel_alloc).Send(
						sel_initWithRect_options_owner_userInfo,
						bounds, options, self, 0)
					self.Send(sel_addTrackingArea, trackingArea)

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
				Cmd: objc.RegisterName("acceptsFirstMouse:"),
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID) bool {
					return true
				},
			},
			// NSTextInputClient methods.
			{
				Cmd: sel_hasMarkedText,
				Fn: func(_ objc.ID, _ objc.SEL) bool {
					return false
				},
			},
			{
				Cmd: sel_markedRange,
				Fn: func(_ objc.ID, _ objc.SEL) nsRange {
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
				Cmd: sel_setMarkedText,
				Fn: func(_ objc.ID, _ objc.SEL, _ objc.ID, _ nsRange, _ nsRange) {
				},
			},
			{
				Cmd: sel_unmarkText,
				Fn: func(_ objc.ID, _ objc.SEL) {
				},
			},
			{
				Cmd: sel_validAttributesForMarkedText,
				Fn: func(_ objc.ID, _ objc.SEL) objc.ID {
					// Return an empty NSArray.
					return objc.ID(class_NSArray).Send(sel_alloc).Send(sel_init)
				},
			},
			{
				Cmd: sel_attributedSubstringForProposedRange,
				Fn: func(_ objc.ID, _ objc.SEL, _ nsRange, _ uintptr) objc.ID {
					return 0 // nil
				},
			},
			{
				Cmd: sel_insertText,
				Fn: func(self objc.ID, _ objc.SEL, text objc.ID, _ nsRange) {
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return
					}
					// Get the string from the text object.
					str := cocoa.NSString{ID: text}
					s := str.String()
					for _, ch := range s {
						flags := uintptr(objc.ID(class_NSEvent).Send(objc.RegisterName("modifierFlags")))
						mods := translateFlags(flags)
						window.inputChar(ch, mods, true)
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
				Cmd: sel_firstRectForCharacterRange,
				Fn: func(self objc.ID, _ objc.SEL, _ nsRange, _ uintptr) cocoa.NSRect {
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return cocoa.NSRect{}
					}
					frame := objc.Send[cocoa.NSRect](window.platform.object, sel_frame)
					return frame
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
					window := getGoWindow(class_GLFWContentView, self)
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
					window := getGoWindow(class_GLFWContentView, self)
					if window == nil {
						return false
					}

					pasteboard := sender.Send(sel_draggingPasteboard)
					urlClass := objc.ID(class_NSURL)
					classes := objc.ID(class_NSArray).Send(sel_arrayWithObject, urlClass)
					urls := pasteboard.Send(sel_readObjectsForClasses, classes, 0)
					if urls == 0 {
						return false
					}

					urlCount := int(urls.Send(sel_count))
					if urlCount == 0 {
						return false
					}

					paths := make([]string, urlCount)
					for i := range urlCount {
						url := urls.Send(sel_objectAtIndex, i)
						pathStr := cocoa.NSString{ID: url.Send(sel_path)}
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
		dx := objc.Send[float64](event, sel_deltaX)
		dy := objc.Send[float64](event, sel_deltaY)

		dx -= window.platform.cursorWarpDeltaX
		dy -= window.platform.cursorWarpDeltaY
		window.platform.cursorWarpDeltaX = 0
		window.platform.cursorWarpDeltaY = 0

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
}

// updateWindowSize updates the cached window and framebuffer sizes, invoking callbacks as needed.
func updateWindowSize(window *Window) {
	if window.platform.object == 0 || window.platform.view == 0 {
		return
	}

	contentRect := objc.Send[cocoa.NSRect](window.platform.view, sel_frame)
	fbRect := objc.Send[cocoa.NSRect](window.platform.view, sel_convertRectToBacking, contentRect)

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

	// Determine the style mask.
	var styleMask uintptr
	if wndconfig.decorated {
		styleMask = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable
		if wndconfig.resizable {
			styleMask |= NSWindowStyleMaskResizable
		}
	} else {
		styleMask = NSWindowStyleMaskBorderless
	}

	// Create the window content rect.
	contentRect := cocoa.NSRect{
		Origin: cocoa.NSPoint{X: 0, Y: 0},
		Size:   cocoa.NSSize{Width: float64(wndconfig.width), Height: float64(wndconfig.height)},
	}

	// Create the GLFWWindow instance.
	nsWindow := objc.ID(class_GLFWWindow).Send(sel_alloc).Send(sel_initWithContentRect,
		contentRect, styleMask, uintptr(NSBackingStoreBuffered), false)
	if nsWindow == 0 {
		return fmt.Errorf("glfw: failed to create Cocoa window: %w", PlatformError)
	}

	window.platform.object = nsWindow

	// Center the window on the screen.
	nsWindow.Send(objc.RegisterName("center"))

	// Set the window properties.
	if wndconfig.floating {
		nsWindow.Send(sel_setLevel, uintptr(NSFloatingWindowLevel))
	}

	nsWindow.Send(sel_setTitle, cocoa.NSString_alloc().InitWithUTF8String(wndconfig.title).ID)
	nsWindow.Send(sel_setRestorable, false)

	// Set collection behavior.
	var collectionBehavior uintptr
	if wndconfig.resizable {
		collectionBehavior |= _NSWindowCollectionBehaviorFullScreenPrimary
		collectionBehavior |= _NSWindowCollectionBehaviorManaged
	}
	nsWindow.Send(sel_setCollectionBehavior, collectionBehavior)

	// Create the delegate.
	delegateID := objc.ID(class_GLFWWindowDelegate).Send(sel_alloc).Send(sel_init)
	setGoWindow(delegateID, window)
	window.platform.delegate = delegateID
	nsWindow.Send(sel_setDelegate, delegateID)

	// Create the content view.
	viewID := objc.ID(class_GLFWContentView).Send(sel_alloc).Send(sel_init)
	setGoWindow(viewID, window)
	window.platform.view = viewID
	nsWindow.Send(sel_setContentView, viewID)

	// Register for dragged types (file URLs).
	fileURLType := cocoa.NSString_alloc().InitWithUTF8String("public.file-url")
	typesArray := objc.ID(class_NSArray).Send(sel_arrayWithObject, fileURLType.ID)
	viewID.Send(sel_registerForDraggedTypes, typesArray)

	// Set up retina/HiDPI support.
	screen := nsWindow.Send(sel_screen)
	if screen != 0 {
		scale := objc.Send[float64](screen, sel_backingScaleFactor)
		window.platform.xscale = float32(scale)
		window.platform.yscale = float32(scale)
		window.platform.retina = scale != 1.0
	} else {
		window.platform.xscale = 1.0
		window.platform.yscale = 1.0
	}

	// Handle transparent framebuffer.
	if fbconfig_.transparent {
		nsWindow.Send(sel_setOpaque, false)
		nsWindow.Send(sel_setHasShadow, false)
		nsWindow.Send(sel_setBackgroundColor, objc.ID(class_NSColor).Send(sel_clearColor))
	}

	// Update initial size cache.
	contentViewRect := objc.Send[cocoa.NSRect](viewID, sel_frame)
	window.platform.width = int(contentViewRect.Size.Width)
	window.platform.height = int(contentViewRect.Size.Height)

	fbRect := objc.Send[cocoa.NSRect](viewID, sel_convertRectToBacking, contentViewRect)
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

	if _glfw.platformWindow.disabledCursorWindow == w {
		_glfw.platformWindow.disabledCursorWindow = nil
		cgAssociateMouseAndMouseCursorPosition(1)
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

	return nil
}

func (w *Window) platformSetWindowTitle(title string) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	w.platform.object.Send(sel_setTitle, cocoa.NSString_alloc().InitWithUTF8String(title).ID)
	// NSWindow.miniwindowTitle is auto-derived from the title.
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

	contentRect := cocoa.NSRect{
		Origin: cocoa.NSPoint{
			X: float64(xpos),
			Y: float64(transformYNS(float32(ypos))),
		},
		Size: cocoa.NSSize{Width: 0, Height: 0},
	}

	frameRect := objc.Send[cocoa.NSRect](w.platform.object, sel_frameRectForContentRect, contentRect)
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

	w.platform.object.Send(sel_setContentSize, cocoa.NSSize{Width: float64(width), Height: float64(height)})
	return nil
}

func (w *Window) platformSetWindowSizeLimits(minwidth, minheight, maxwidth, maxheight int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if minwidth == DontCare || minheight == DontCare {
		w.platform.object.Send(sel_setMinSize, cocoa.NSSize{Width: 0, Height: 0})
	} else {
		w.platform.object.Send(sel_setMinSize, cocoa.NSSize{Width: float64(minwidth), Height: float64(minheight)})
	}

	if maxwidth == DontCare || maxheight == DontCare {
		w.platform.object.Send(sel_setMaxSize, cocoa.NSSize{Width: 0, Height: 0})
	} else {
		w.platform.object.Send(sel_setMaxSize, cocoa.NSSize{Width: float64(maxwidth), Height: float64(maxheight)})
	}

	return nil
}

func (w *Window) platformSetWindowAspectRatio(numer, denom int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if numer == DontCare || denom == DontCare {
		w.platform.object.Send(sel_setContentAspectRatio, cocoa.NSSize{Width: 0, Height: 0})
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
	frameRect := objc.Send[cocoa.NSRect](w.platform.object, sel_frame)

	contentRectInWindow := objc.Send[cocoa.NSRect](w.platform.object, sel_contentRectForFrameRect, frameRect)

	left = int(contentRectInWindow.Origin.X - frameRect.Origin.X)
	top = int((frameRect.Origin.Y + frameRect.Size.Height) - (contentRectInWindow.Origin.Y + contentRectInWindow.Size.Height))
	right = int((frameRect.Origin.X + frameRect.Size.Width) - (contentRectInWindow.Origin.X + contentRect.Size.Width))
	bottom = int(contentRectInWindow.Origin.Y - frameRect.Origin.Y)
	return left, top, right, bottom, nil
}

func (w *Window) platformGetWindowContentScale() (xscale, yscale float32, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	screen := w.platform.object.Send(sel_screen)
	if screen == 0 {
		screen = objc.ID(class_NSScreen).Send(sel_mainScreen)
	}

	scale := objc.Send[float64](screen, sel_backingScaleFactor)
	return float32(scale), float32(scale), nil
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

	w.platform.object.Send(sel_makeKeyAndOrderFront, 0)
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
			frameRect := objc.Send[cocoa.NSRect](w.platform.object, sel_frameRectForContentRect, contentRect)
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

	if w.monitor != nil {
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
		frameRect := objc.Send[cocoa.NSRect](w.platform.object, sel_frameRectForContentRect, contentRect)
		w.platform.object.Send(objc.RegisterName("setFrame:display:"), frameRect, true)
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

	return objc.Send[bool](w.platform.object, sel_isZoomed)
}

func (w *Window) platformWindowHovered() (bool, error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return cursorInContentArea(w), nil
}

func (w *Window) platformFramebufferTransparent() bool {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	return !objc.Send[bool](w.platform.object, objc.RegisterName("isOpaque"))
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
	return nil
}

func (w *Window) platformSetWindowDecorated(enabled bool) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	var mask uintptr
	if enabled {
		mask = NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable
		if w.resizable {
			mask |= NSWindowStyleMaskResizable
		}
	} else {
		mask = NSWindowStyleMaskBorderless
	}
	w.platform.object.Send(objc.RegisterName("setStyleMask:"), mask)
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

	return objc.Send[float32](w.platform.object, sel_alphaValue), nil
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

	for {
		event := objc.Send[objc.ID](nsApp(), sel_nextEventMatchingMask,
			uintptr(NSEventMaskAny),
			uintptr(0), // distantPast (nil = no wait)
			nsDefaultRunLoopMode.ID,
			true)
		if event == 0 {
			break
		}
		nsApp().Send(sel_sendEvent, event)
	}
	nsApp().Send(sel_updateWindows)
	return nil
}

func platformWaitEvents() error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// Wait for an event with no timeout (distantFuture).
	distantFuture := objc.ID(objc.GetClass("NSDate")).Send(objc.RegisterName("distantFuture"))
	event := objc.Send[objc.ID](nsApp(), sel_nextEventMatchingMask,
		uintptr(NSEventMaskAny),
		distantFuture,
		nsDefaultRunLoopMode.ID,
		true)
	if event != 0 {
		nsApp().Send(sel_sendEvent, event)
	}

	// Process remaining events.
	for {
		event := objc.Send[objc.ID](nsApp(), sel_nextEventMatchingMask,
			uintptr(NSEventMaskAny),
			uintptr(0),
			nsDefaultRunLoopMode.ID,
			true)
		if event == 0 {
			break
		}
		nsApp().Send(sel_sendEvent, event)
	}
	nsApp().Send(sel_updateWindows)
	return nil
}

func platformWaitEventsTimeout(timeout float64) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// Create an NSDate for the timeout.
	date := objc.ID(objc.GetClass("NSDate")).Send(objc.RegisterName("dateWithTimeIntervalSinceNow:"), timeout)
	event := objc.Send[objc.ID](nsApp(), sel_nextEventMatchingMask,
		uintptr(NSEventMaskAny),
		date,
		nsDefaultRunLoopMode.ID,
		true)
	if event != 0 {
		nsApp().Send(sel_sendEvent, event)
	}

	// Process remaining events.
	for {
		event := objc.Send[objc.ID](nsApp(), sel_nextEventMatchingMask,
			uintptr(NSEventMaskAny),
			uintptr(0),
			nsDefaultRunLoopMode.ID,
			true)
		if event == 0 {
			break
		}
		nsApp().Send(sel_sendEvent, event)
	}
	nsApp().Send(sel_updateWindows)
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

	pos := objc.Send[cocoa.NSPoint](objc.ID(class_NSEvent), objc.RegisterName("mouseLocation"))

	// Convert from screen coordinates to window content coordinates.
	windowFrame := objc.Send[cocoa.NSRect](w.platform.object, sel_frame)
	contentRect := objc.Send[cocoa.NSRect](w.platform.object, sel_contentRectForFrameRect, windowFrame)

	xpos = pos.X - contentRect.Origin.X
	ypos = (contentRect.Origin.Y + contentRect.Size.Height) - pos.Y

	return xpos, ypos, nil
}

func (w *Window) platformSetCursorPos(xpos, ypos float64) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	windowFrame := objc.Send[cocoa.NSRect](w.platform.object, sel_frame)
	contentRect := objc.Send[cocoa.NSRect](w.platform.object, sel_contentRectForFrameRect, windowFrame)

	// Convert from content coordinates to screen coordinates (Cocoa convention).
	screenX := contentRect.Origin.X + xpos
	screenY := contentRect.Origin.Y + contentRect.Size.Height - ypos

	// Convert from Cocoa screen coordinates to CoreGraphics coordinates (top-left origin).
	primaryBounds := cgDisplayBounds(cgMainDisplayID())
	cgPoint := cocoa.CGPoint{
		X: screenX,
		Y: primaryBounds.Height - screenY,
	}

	cgWarpMouseCursorPosition(cgPoint)

	// Note the warp delta so we can subtract it from the next mouse moved event.
	w.platform.cursorWarpDeltaX += xpos - float64(w.platform.width)/2
	w.platform.cursorWarpDeltaY += ypos - float64(w.platform.height)/2

	return nil
}

func (w *Window) platformSetCursorMode(mode int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if w.platformWindowFocused() {
		updateCursorMode(w)
	}

	if mode == CursorDisabled {
		_glfw.platformWindow.restoreCursorPosX, _glfw.platformWindow.restoreCursorPosY, _ = w.platformGetCursorPos()
	} else if _glfw.platformWindow.disabledCursorWindow == w {
		// This is handled by updateCursorMode above.
	}

	return nil
}

func (c *Cursor) platformCreateStandardCursor(shape StandardCursor) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	var cursor objc.ID
	switch shape {
	case ArrowCursor:
		cursor = objc.ID(class_NSCursor).Send(sel_arrowCursor)
	case IBeamCursor:
		cursor = objc.ID(class_NSCursor).Send(sel_IBeamCursor)
	case CrosshairCursor:
		cursor = objc.ID(class_NSCursor).Send(sel_crosshairCursor)
	case HandCursor:
		cursor = objc.ID(class_NSCursor).Send(sel_pointingHandCursor)
	case HResizeCursor:
		cursor = objc.ID(class_NSCursor).Send(sel_resizeLeftRightCursor)
	case VResizeCursor:
		cursor = objc.ID(class_NSCursor).Send(sel_resizeUpDownCursor)
	case ResizeNWSECursor:
		// macOS doesn't have a specific NWSE cursor; use closed hand.
		cursor = objc.ID(class_NSCursor).Send(sel_closedHandCursor)
	case ResizeNESWCursor:
		// macOS doesn't have a specific NESW cursor; use closed hand.
		cursor = objc.ID(class_NSCursor).Send(sel_closedHandCursor)
	case ResizeAllCursor:
		cursor = objc.ID(class_NSCursor).Send(sel_openHandCursor)
	case NotAllowedCursor:
		cursor = objc.ID(class_NSCursor).Send(sel_operationNotAllowedCursor)
	default:
		return fmt.Errorf("glfw: invalid standard cursor 0x%08X: %w", shape, InvalidEnum)
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

	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	types := objc.ID(class_NSArray).Send(sel_arrayWithObject, nsPasteboardTypeString.ID)
	pasteboard.Send(sel_declareTypes, types, 0)
	pasteboard.Send(sel_setStringForType,
		cocoa.NSString_alloc().InitWithUTF8String(str).ID,
		nsPasteboardTypeString.ID)
	return nil
}

func platformGetClipboardString() (string, error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	pasteboard := objc.ID(class_NSPasteboard).Send(sel_generalPasteboard)
	strID := pasteboard.Send(sel_stringForType, nsPasteboardTypeString.ID)
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
	w.monitor.window = w
	return nil
}

func (w *Window) releaseMonitor() error {
	if w.monitor == nil {
		return nil
	}
	w.monitor.restoreVideoModeNS()
	w.monitor.window = nil
	return nil
}

// Ensure reflect is used (needed for objc.RegisterClass field definitions).
var _ = reflect.TypeFor[uintptr]
