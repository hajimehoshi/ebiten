// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import "github.com/ebitengine/purego/objc"

type platformWindowState struct {
	object   objc.ID // NSWindow
	delegate objc.ID // GLFWWindowDelegate
	view     objc.ID // GLFWContentView
	layer    objc.ID // CALayer

	maximized bool
	occluded  bool
	retina    bool

	width, height     int
	fbWidth, fbHeight int
	xscale, yscale    float32

	cursorWarpDeltaX, cursorWarpDeltaY float64

	markedText objc.ID // NSMutableAttributedString for IME composition
}

type platformMonitorState struct {
	displayID           uint32  // CGDirectDisplayID
	previousMode        uintptr // CGDisplayModeRef
	unitNumber          uint32
	screen              objc.ID // NSScreen
	fallbackRefreshRate float64
}

type platformCursorState struct {
	object objc.ID // NSCursor
}

type platformTLSState struct {
	allocated bool
	value     uintptr
}

type platformLibraryWindowState struct {
	eventSource  uintptr // CGEventSourceRef
	delegate     objc.ID
	cursorHidden bool
	inputSource  uintptr // TISInputSourceRef
	unicodeData  uintptr // CFDataRef
	helper       objc.ID
	keyUpMonitor objc.ID
	nibObjects   objc.ID

	keynames  [KeyLast + 1]string
	keycodes  [256]Key
	scancodes [KeyLast + 1]int

	clipboardString      string
	cascadePoint         [2]float64
	restoreCursorPosX    float64
	restoreCursorPosY    float64
	disabledCursorWindow *Window

	tis tisState
}

type tisState struct {
	kPropertyUnicodeKeyLayoutData        uintptr // CFStringRef
	CopyCurrentKeyboardLayoutInputSource func() uintptr
	GetInputSourceProperty               func(inputSource uintptr, propertyKey uintptr) uintptr
	GetKbdType                           func() uint8
	UCKeyTranslate                       func(keyLayoutPtr uintptr, virtualKeyCode uint16, keyAction uint16, modifierKeyState uint32, keyboardType uint32, keyTranslateOptions uint32, deadKeyState *uint32, maxStringLength int, actualStringLength *int, unicodeString *uint16) int32
}

type platformContextState struct {
	object      objc.ID // NSOpenGLContext
	pixelFormat objc.ID // NSOpenGLPixelFormat
}

type platformLibraryContextState struct{}
