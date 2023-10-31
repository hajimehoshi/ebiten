// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd || openbsd

package glfw

// #include <stdlib.h>
// #define GLFW_INCLUDE_NONE
// #include "glfw3_unix.h"
//
// void goKeyCB(void* window, int key, int  scancode, int action, int mods);
// void goCharCB(void* window, unsigned int character);
// void goCharModsCB(void* window, unsigned int character, int mods);
// void goMouseButtonCB(void* window, int button, int action, int mods);
// void goCursorPosCB(void* window, double xpos, double ypos);
// void goCursorEnterCB(void* window, int entered);
// void goScrollCB(void* window, double xoff, double yoff);
// void goDropCB(void* window, int count, char** names);
//
// static void glfwSetKeyCallbackCB(GLFWwindow *window) {
//   glfwSetKeyCallback(window, (GLFWkeyfun)goKeyCB);
// }
//
// static void glfwSetCharCallbackCB(GLFWwindow *window) {
//    glfwSetCharCallback(window, (GLFWcharfun)goCharCB);
// }
//
// static void glfwSetCharModsCallbackCB(GLFWwindow *window) {
//   glfwSetCharModsCallback(window, (GLFWcharmodsfun)goCharModsCB);
// }
//
// static void glfwSetMouseButtonCallbackCB(GLFWwindow *window) {
//   glfwSetMouseButtonCallback(window, (GLFWmousebuttonfun)goMouseButtonCB);
// }
//
// static void glfwSetCursorPosCallbackCB(GLFWwindow *window) {
//   glfwSetCursorPosCallback(window, (GLFWcursorposfun)goCursorPosCB);
// }
//
// static void glfwSetCursorEnterCallbackCB(GLFWwindow *window) {
//   glfwSetCursorEnterCallback(window, (GLFWcursorenterfun)goCursorEnterCB);
// }
//
// static void glfwSetScrollCallbackCB(GLFWwindow *window) {
//   glfwSetScrollCallback(window, (GLFWscrollfun)goScrollCB);
// }
//
// static void glfwSetDropCallbackCB(GLFWwindow *window) {
//   glfwSetDropCallback(window, (GLFWdropfun)goDropCB);
// }
import "C"

import (
	"image"
	"unsafe"
)

// Cursor represents a cursor.
type Cursor struct {
	data *C.GLFWcursor
}

//export goMouseButtonCB
func goMouseButtonCB(window unsafe.Pointer, button, action, mods C.int) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fMouseButtonHolder(w, MouseButton(button), Action(action), ModifierKey(mods))
}

//export goCursorPosCB
func goCursorPosCB(window unsafe.Pointer, xpos, ypos C.double) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fCursorPosHolder(w, float64(xpos), float64(ypos))
}

//export goCursorEnterCB
func goCursorEnterCB(window unsafe.Pointer, entered C.int) {
	w := windows.get((*C.GLFWwindow)(window))
	hasEntered := entered != 0
	w.fCursorEnterHolder(w, hasEntered)
}

//export goScrollCB
func goScrollCB(window unsafe.Pointer, xoff, yoff C.double) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fScrollHolder(w, float64(xoff), float64(yoff))
}

//export goKeyCB
func goKeyCB(window unsafe.Pointer, key, scancode, action, mods C.int) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fKeyHolder(w, Key(key), int(scancode), Action(action), ModifierKey(mods))
}

//export goCharCB
func goCharCB(window unsafe.Pointer, character C.uint) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fCharHolder(w, rune(character))
}

//export goCharModsCB
func goCharModsCB(window unsafe.Pointer, character C.uint, mods C.int) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fCharModsHolder(w, rune(character), ModifierKey(mods))
}

//export goDropCB
func goDropCB(window unsafe.Pointer, count C.int, names **C.char) { // TODO: The types of name can be `**C.char` or `unsafe.Pointer`, use whichever is better.
	w := windows.get((*C.GLFWwindow)(window))
	namesSlice := make([]string, int(count)) // TODO: Make this better. This part is unfinished, hacky, probably not correct, and not idiomatic.
	for i := 0; i < int(count); i++ {        // TODO: Make this better. It should be cleaned up and vetted.
		var x *C.char                                                                                 // TODO: Make this better.
		p := (**C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(names)) + uintptr(i)*unsafe.Sizeof(x))) // TODO: Make this better.
		namesSlice[i] = C.GoString(*p)                                                                // TODO: Make this better.
	}
	w.fDropHolder(w, namesSlice)
}

// GetInputMode returns the value of an input option of the window.
func (w *Window) GetInputMode(mode InputMode) (int, error) {
	ret := int(C.glfwGetInputMode(w.data, C.int(mode)))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, err
	}
	return ret, nil
}

// SetInputMode sets an input option for the window.
func (w *Window) SetInputMode(mode InputMode, value int) error {
	C.glfwSetInputMode(w.data, C.int(mode), C.int(value))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// RawMouseMotionSupported returns whether raw mouse motion is supported on the
// current system. This status does not change after GLFW has been initialized
// so you only need to check this once. If you attempt to enable raw motion on
// a system that does not support it, PlatformError will be emitted.
//
// Raw mouse motion is closer to the actual motion of the mouse across a
// surface. It is not affected by the scaling and acceleration applied to the
// motion of the desktop cursor. That processing is suitable for a cursor while
// raw motion is better for controlling for example a 3D camera. Because of
// this, raw mouse motion is only provided when the cursor is disabled.
//
// This function must only be called from the main thread.
func RawMouseMotionSupported() bool {
	return int(C.glfwRawMouseMotionSupported()) == True
}

// GetKeyScancode function returns the platform-specific scancode of the
// specified key.
//
// If the key is KeyUnknown or does not exist on the keyboard this method will
// return -1.
func GetKeyScancode(key Key) int {
	return int(C.glfwGetKeyScancode(C.int(key)))
}

// GetKey returns the last reported state of a keyboard key. The returned state
// is one of Press or Release. The higher-level state Repeat is only reported to
// the key callback.
//
// If the StickyKeys input mode is enabled, this function returns Press the first
// time you call this function after a key has been pressed, even if the key has
// already been released.
//
// The key functions deal with physical keys, with key tokens named after their
// use on the standard US keyboard layout. If you want to input text, use the
// Unicode character callback instead.
func (w *Window) GetKey(key Key) (Action, error) {
	ret := Action(C.glfwGetKey(w.data, C.int(key)))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, err
	}
	return ret, nil
}

// GetKeyName returns the localized name of the specified printable key.
//
// If the key is glfw.KeyUnknown, the scancode is used, otherwise the scancode is ignored.
func GetKeyName(key Key, scancode int) (string, error) {
	ret := C.glfwGetKeyName(C.int(key), C.int(scancode))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return "", err
	}
	return C.GoString(ret), nil
}

// GetMouseButton returns the last state reported for the specified mouse button.
//
// If the StickyMouseButtons input mode is enabled, this function returns Press
// the first time you call this function after a mouse button has been pressed,
// even if the mouse button has already been released.
func (w *Window) GetMouseButton(button MouseButton) (Action, error) {
	ret := Action(C.glfwGetMouseButton(w.data, C.int(button)))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, err
	}
	return ret, nil
}

// GetCursorPos returns the last reported position of the cursor.
//
// If the cursor is disabled (with CursorDisabled) then the cursor position is
// unbounded and limited only by the minimum and maximum values of a double.
//
// The coordinate can be converted to their integer equivalents with the floor
// function. Casting directly to an integer type works for positive coordinates,
// but fails for negative ones.
func (w *Window) GetCursorPos() (x, y float64, err error) {
	var xpos, ypos C.double
	C.glfwGetCursorPos(w.data, &xpos, &ypos)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, 0, err
	}
	return float64(xpos), float64(ypos), nil
}

// SetCursorPos sets the position of the cursor. The specified window must
// be focused. If the window does not have focus when this function is called,
// it fails silently.
//
// If the cursor is disabled (with CursorDisabled) then the cursor position is
// unbounded and limited only by the minimum and maximum values of a double.
func (w *Window) SetCursorPos(xpos, ypos float64) error {
	C.glfwSetCursorPos(w.data, C.double(xpos), C.double(ypos))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// CreateCursor creates a new custom cursor image that can be set for a window with SetCursor.
// The cursor can be destroyed with Destroy. Any remaining cursors are destroyed by Terminate.
//
// The image is ideally provided in the form of *image.NRGBA.
// The pixels are 32-bit, little-endian, non-premultiplied RGBA, i.e. eight
// bits per channel with the red channel first. They are arranged canonically
// as packed sequential rows, starting from the top-left corner. If the image
// type is not *image.NRGBA, it will be converted to it.
//
// The cursor hotspot is specified in pixels, relative to the upper-left corner of the cursor image.
// Like all other coordinate systems in GLFW, the X-axis points to the right and the Y-axis points down.
func CreateCursor(img image.Image, xhot, yhot int) (*Cursor, error) {
	glfwImg, free := imageToGLFWImage(img)
	defer free()

	c := C.glfwCreateCursor(&glfwImg, C.int(xhot), C.int(yhot))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}

	return &Cursor{c}, nil
}

// CreateStandardCursor returns a cursor with a standard shape,
// that can be set for a window with SetCursor.
func CreateStandardCursor(shape StandardCursor) (*Cursor, error) {
	c := C.glfwCreateStandardCursor(C.int(shape))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return &Cursor{c}, nil
}

// Destroy destroys a cursor previously created with CreateCursor.
// Any remaining cursors will be destroyed by Terminate.
func (c *Cursor) Destroy() error {
	C.glfwDestroyCursor(c.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// SetCursor sets the cursor image to be used when the cursor is over the client area
// of the specified window. The set cursor will only be visible when the cursor mode of the
// window is CursorNormal.
//
// On some platforms, the set cursor may not be visible unless the window also has input focus.
func (w *Window) SetCursor(c *Cursor) error {
	if c == nil {
		C.glfwSetCursor(w.data, nil)
	} else {
		C.glfwSetCursor(w.data, c.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// KeyCallback is the key callback.
type KeyCallback func(w *Window, key Key, scancode int, action Action, mods ModifierKey)

// SetKeyCallback sets the key callback which is called when a key is pressed,
// repeated or released.
//
// The key functions deal with physical keys, with layout independent key tokens
// named after their values in the standard US keyboard layout. If you want to
// input text, use the SetCharCallback instead.
//
// When a window loses focus, it will generate synthetic key release events for
// all pressed keys. You can tell these events from user-generated events by the
// fact that the synthetic ones are generated after the window has lost focus,
// i.e. Focused will be false and the focus callback will have already been
// called.
func (w *Window) SetKeyCallback(cbfun KeyCallback) (previous KeyCallback, err error) {
	previous = w.fKeyHolder
	w.fKeyHolder = cbfun
	if cbfun == nil {
		C.glfwSetKeyCallback(w.data, nil)
	} else {
		C.glfwSetKeyCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// CharCallback is the character callback.
type CharCallback func(w *Window, char rune)

// SetCharCallback sets the character callback which is called when a
// Unicode character is input.
//
// The character callback is intended for Unicode text input. As it deals with
// characters, it is keyboard layout dependent, whereas the
// key callback is not. Characters do not map 1:1
// to physical keys, as a key may produce zero, one or more characters. If you
// want to know whether a specific physical key was pressed or released, see
// the key callback instead.
//
// The character callback behaves as system text input normally does and will
// not be called if modifier keys are held down that would prevent normal text
// input on that platform, for example a Super (Command) key on OS X or Alt key
// on Windows. There is a character with modifiers callback that receives these events.
func (w *Window) SetCharCallback(cbfun CharCallback) (previous CharCallback, err error) {
	previous = w.fCharHolder
	w.fCharHolder = cbfun
	if cbfun == nil {
		C.glfwSetCharCallback(w.data, nil)
	} else {
		C.glfwSetCharCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// CharModsCallback is the character with modifiers callback.
type CharModsCallback func(w *Window, char rune, mods ModifierKey)

// SetCharModsCallback sets the character with modifiers callback which is called when a
// Unicode character is input regardless of what modifier keys are used.
//
// Deprecated: Scheduled for removal in version 4.0.
//
// The character with modifiers callback is intended for implementing custom
// Unicode character input. For regular Unicode text input, see the
// character callback. Like the character callback, the character with modifiers callback
// deals with characters and is keyboard layout dependent. Characters do not
// map 1:1 to physical keys, as a key may produce zero, one or more characters.
// If you want to know whether a specific physical key was pressed or released,
// see the key callback instead.
func (w *Window) SetCharModsCallback(cbfun CharModsCallback) (previous CharModsCallback, err error) {
	previous = w.fCharModsHolder
	w.fCharModsHolder = cbfun
	if cbfun == nil {
		C.glfwSetCharModsCallback(w.data, nil)
	} else {
		C.glfwSetCharModsCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// MouseButtonCallback is the mouse button callback.
type MouseButtonCallback func(w *Window, button MouseButton, action Action, mods ModifierKey)

// SetMouseButtonCallback sets the mouse button callback which is called when a
// mouse button is pressed or released.
//
// When a window loses focus, it will generate synthetic mouse button release
// events for all pressed mouse buttons. You can tell these events from
// user-generated events by the fact that the synthetic ones are generated after
// the window has lost focus, i.e. Focused will be false and the focus
// callback will have already been called.
func (w *Window) SetMouseButtonCallback(cbfun MouseButtonCallback) (previous MouseButtonCallback, err error) {
	previous = w.fMouseButtonHolder
	w.fMouseButtonHolder = cbfun
	if cbfun == nil {
		C.glfwSetMouseButtonCallback(w.data, nil)
	} else {
		C.glfwSetMouseButtonCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// CursorPosCallback the cursor position callback.
type CursorPosCallback func(w *Window, xpos float64, ypos float64)

// SetCursorPosCallback sets the cursor position callback which is called
// when the cursor is moved. The callback is provided with the position relative
// to the upper-left corner of the client area of the window.
func (w *Window) SetCursorPosCallback(cbfun CursorPosCallback) (previous CursorPosCallback, err error) {
	previous = w.fCursorPosHolder
	w.fCursorPosHolder = cbfun
	if cbfun == nil {
		C.glfwSetCursorPosCallback(w.data, nil)
	} else {
		C.glfwSetCursorPosCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// CursorEnterCallback is the cursor boundary crossing callback.
type CursorEnterCallback func(w *Window, entered bool)

// SetCursorEnterCallback the cursor boundary crossing callback which is called
// when the cursor enters or leaves the client area of the window.
func (w *Window) SetCursorEnterCallback(cbfun CursorEnterCallback) (previous CursorEnterCallback, err error) {
	previous = w.fCursorEnterHolder
	w.fCursorEnterHolder = cbfun
	if cbfun == nil {
		C.glfwSetCursorEnterCallback(w.data, nil)
	} else {
		C.glfwSetCursorEnterCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// ScrollCallback is the scroll callback.
type ScrollCallback func(w *Window, xoff float64, yoff float64)

// SetScrollCallback sets the scroll callback which is called when a scrolling
// device is used, such as a mouse wheel or scrolling area of a touchpad.
func (w *Window) SetScrollCallback(cbfun ScrollCallback) (previous ScrollCallback, err error) {
	previous = w.fScrollHolder
	w.fScrollHolder = cbfun
	if cbfun == nil {
		C.glfwSetScrollCallback(w.data, nil)
	} else {
		C.glfwSetScrollCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// DropCallback is the drop callback.
type DropCallback func(w *Window, names []string)

// SetDropCallback sets the drop callback which is called when an object
// is dropped over the window.
func (w *Window) SetDropCallback(cbfun DropCallback) (previous DropCallback, err error) {
	previous = w.fDropHolder
	w.fDropHolder = cbfun
	if cbfun == nil {
		C.glfwSetDropCallback(w.data, nil)
	} else {
		C.glfwSetDropCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}
