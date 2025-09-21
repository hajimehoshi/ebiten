// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd || openbsd

package glfw

// #include <stdlib.h>
// #define GLFW_INCLUDE_NONE
// #include "glfw3_unix.h"
//
// void goWindowPosCB(void* window, int xpos, int ypos);
// void goWindowSizeCB(void* window, int width, int height);
// void goWindowCloseCB(void* window);
// void goWindowRefreshCB(void* window);
// void goWindowFocusCB(void* window, int focused);
// void goWindowIconifyCB(void* window, int iconified);
// void goFramebufferSizeCB(void* window, int width, int height);
// void goWindowMaximizeCB(void* window, int maximized);
// void goWindowContentScaleCB(void* window, float x, float y);
//
// #cgo noescape glfwSetWindowPosCallbackCB
// static void glfwSetWindowPosCallbackCB(GLFWwindow *window) {
//   glfwSetWindowPosCallback(window, (GLFWwindowposfun)goWindowPosCB);
// }
//
// #cgo noescape glfwSetWindowSizeCallbackCB
// static void glfwSetWindowSizeCallbackCB(GLFWwindow *window) {
//   glfwSetWindowSizeCallback(window, (GLFWwindowsizefun)goWindowSizeCB);
// }
//
// #cgo noescape glfwSetWindowCloseCallbackCB
// static void glfwSetWindowCloseCallbackCB(GLFWwindow *window) {
//   glfwSetWindowCloseCallback(window, (GLFWwindowclosefun)goWindowCloseCB);
// }
//
// #cgo noescape glfwSetWindowRefreshCallbackCB
// static void glfwSetWindowRefreshCallbackCB(GLFWwindow *window) {
//   glfwSetWindowRefreshCallback(window, (GLFWwindowrefreshfun)goWindowRefreshCB);
// }
//
// #cgo noescape glfwSetWindowFocusCallbackCB
// static void glfwSetWindowFocusCallbackCB(GLFWwindow *window) {
//   glfwSetWindowFocusCallback(window, (GLFWwindowfocusfun)goWindowFocusCB);
// }
//
// #cgo noescape glfwSetWindowIconifyCallbackCB
// static void glfwSetWindowIconifyCallbackCB(GLFWwindow *window) {
//   glfwSetWindowIconifyCallback(window, (GLFWwindowiconifyfun)goWindowIconifyCB);
// }
//
// #cgo noescape glfwSetFramebufferSizeCallbackCB
// static void glfwSetFramebufferSizeCallbackCB(GLFWwindow *window) {
//   glfwSetFramebufferSizeCallback(window, (GLFWframebuffersizefun)goFramebufferSizeCB);
// }
//
// #cgo noescape glfwSetWindowMaximizeCallbackCB
// static void glfwSetWindowMaximizeCallbackCB(GLFWwindow *window) {
//   glfwSetWindowMaximizeCallback(window, (GLFWwindowmaximizefun)goWindowMaximizeCB);
// }
//
// #cgo noescape glfwSetWindowContentScaleCallbackCB
// static void glfwSetWindowContentScaleCallbackCB(GLFWwindow *window) {
//   glfwSetWindowContentScaleCallback(window, (GLFWwindowcontentscalefun)goWindowContentScaleCB);
// }
import "C"

import (
	"errors"
	"image"
	"sync"
	"unsafe"
)

// Internal window list stuff
type windowList struct {
	l sync.Mutex
	m map[*C.GLFWwindow]*Window
}

var windows = windowList{m: map[*C.GLFWwindow]*Window{}}

func (w *windowList) put(wnd *Window) {
	w.l.Lock()
	defer w.l.Unlock()
	w.m[wnd.data] = wnd
}

func (w *windowList) remove(wnd *C.GLFWwindow) {
	w.l.Lock()
	defer w.l.Unlock()
	delete(w.m, wnd)
}

func (w *windowList) get(wnd *C.GLFWwindow) *Window {
	w.l.Lock()
	defer w.l.Unlock()
	return w.m[wnd]
}

// Window represents a window.
type Window struct {
	data *C.GLFWwindow

	// Window.
	fPosHolder             func(w *Window, xpos int, ypos int)
	fSizeHolder            func(w *Window, width int, height int)
	fFramebufferSizeHolder func(w *Window, width int, height int)
	fCloseHolder           func(w *Window)
	fMaximizeHolder        func(w *Window, maximized bool)
	fContentScaleHolder    func(w *Window, x float32, y float32)
	fRefreshHolder         func(w *Window)
	fFocusHolder           func(w *Window, focused bool)
	fIconifyHolder         func(w *Window, iconified bool)

	// Input.
	fMouseButtonHolder func(w *Window, button MouseButton, action Action, mod ModifierKey)
	fCursorPosHolder   func(w *Window, xpos float64, ypos float64)
	fCursorEnterHolder func(w *Window, entered bool)
	fScrollHolder      func(w *Window, xoff float64, yoff float64)
	fKeyHolder         func(w *Window, key Key, scancode int, action Action, mods ModifierKey)
	fCharHolder        func(w *Window, char rune)
	fCharModsHolder    func(w *Window, char rune, mods ModifierKey)
	fDropHolder        func(w *Window, names []string)
}

// Handle returns a *C.GLFWwindow reference (i.e. the GLFW window itself).
// This can be used for passing the GLFW window handle to external libraries
// like vulkan-go.
func (w *Window) Handle() unsafe.Pointer {
	return unsafe.Pointer(w.data)
}

// GoWindow creates a Window from a *C.GLFWwindow reference.
// Used when an external C library is calling your Go handlers.
func GoWindow(window unsafe.Pointer) *Window {
	return &Window{data: (*C.GLFWwindow)(window)}
}

//export goWindowPosCB
func goWindowPosCB(window unsafe.Pointer, xpos, ypos C.int) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fPosHolder(w, int(xpos), int(ypos))
}

//export goWindowSizeCB
func goWindowSizeCB(window unsafe.Pointer, width, height C.int) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fSizeHolder(w, int(width), int(height))
}

//export goFramebufferSizeCB
func goFramebufferSizeCB(window unsafe.Pointer, width, height C.int) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fFramebufferSizeHolder(w, int(width), int(height))
}

//export goWindowCloseCB
func goWindowCloseCB(window unsafe.Pointer) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fCloseHolder(w)
}

//export goWindowMaximizeCB
func goWindowMaximizeCB(window unsafe.Pointer, maximized C.int) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fMaximizeHolder(w, maximized != 0)
}

//export goWindowRefreshCB
func goWindowRefreshCB(window unsafe.Pointer) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fRefreshHolder(w)
}

//export goWindowFocusCB
func goWindowFocusCB(window unsafe.Pointer, focused C.int) {
	w := windows.get((*C.GLFWwindow)(window))
	isFocused := focused != 0
	w.fFocusHolder(w, isFocused)
}

//export goWindowIconifyCB
func goWindowIconifyCB(window unsafe.Pointer, iconified C.int) {
	isIconified := iconified != 0
	w := windows.get((*C.GLFWwindow)(window))
	w.fIconifyHolder(w, isIconified)
}

//export goWindowContentScaleCB
func goWindowContentScaleCB(window unsafe.Pointer, x C.float, y C.float) {
	w := windows.get((*C.GLFWwindow)(window))
	w.fContentScaleHolder(w, float32(x), float32(y))
}

// DefaultWindowHints resets all window hints to their default values.
//
// This function may only be called from the main thread.
func DefaultWindowHints() error {
	C.glfwDefaultWindowHints()
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// WindowHint sets hints for the next call to CreateWindow. The hints,
// once set, retain their values until changed by a call to WindowHint or
// DefaultWindowHints, or until the library is terminated with Terminate.
//
// This function may only be called from the main thread.
func WindowHint(target Hint, hint int) error {
	C.glfwWindowHint(C.int(target), C.int(hint))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// WindowHintString sets hints for the next call to CreateWindow. The hints,
// once set, retain their values until changed by a call to this function or
// DefaultWindowHints, or until the library is terminated.
//
// Only string type hints can be set with this function. Integer value hints are
// set with WindowHint.
//
// This function does not check whether the specified hint values are valid. If
// you set hints to invalid values this will instead be reported by the next
// call to CreateWindow.
//
// Some hints are platform specific. These may be set on any platform but they
// will only affect their specific platform. Other platforms will ignore them.
// Setting these hints requires no platform specific headers or functions.
//
// This function must only be called from the main thread.
func WindowHintString(hint Hint, value string) error {
	str := C.CString(value)
	defer C.free(unsafe.Pointer(str))
	C.glfwWindowHintString(C.int(hint), str)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// CreateWindow creates a window and its associated context. Most of the options
// controlling how the window and its context should be created are specified
// through Hint.
//
// Successful creation does not change which context is current. Before you can
// use the newly created context, you need to make it current using
// MakeContextCurrent.
//
// Note that the created window and context may differ from what you requested,
// as not all parameters and hints are hard constraints. This includes the size
// of the window, especially for full screen windows. To retrieve the actual
// attributes of the created window and context, use queries like
// Window.GetAttrib and Window.GetSize.
//
// To create the window at a specific position, make it initially invisible using
// the Visible window hint, set its position and then show it.
//
// If a fullscreen window is active, the screensaver is prohibited from starting.
//
// Windows: If the executable has an icon resource named GLFW_ICON, it will be
// set as the icon for the window. If no such icon is present, the IDI_WINLOGO
// icon will be used instead.
//
// Mac OS X: The GLFW window has no icon, as it is not a document window, but the
// dock icon will be the same as the application bundle's icon. Also, the first
// time a window is opened the menu bar is populated with common commands like
// Hide, Quit and About. The (minimal) about dialog uses information from the
// application's bundle. For more information on bundles, see the Bundle
// Programming Guide provided by Apple.
//
// This function may only be called from the main thread.
func CreateWindow(width, height int, title string, monitor *Monitor, share *Window) (*Window, error) {
	var (
		m *C.GLFWmonitor
		s *C.GLFWwindow
	)

	t := C.CString(title)
	defer C.free(unsafe.Pointer(t))

	if monitor != nil {
		m = monitor.data
	}

	if share != nil {
		s = share.data
	}

	w := C.glfwCreateWindow(C.int(width), C.int(height), t, m, s)
	if w == nil {
		if err := fetchErrorIgnoringPlatformError(); err != nil {
			return nil, err
		}
		return nil, nil
	}

	wnd := &Window{data: w}
	windows.put(wnd)
	return wnd, nil
}

// Destroy destroys the specified window and its context. On calling this
// function, no further callbacks will be called for that window.
//
// This function may only be called from the main thread.
func (w *Window) Destroy() error {
	windows.remove(w.data)
	C.glfwDestroyWindow(w.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// ShouldClose reports the value of the close flag of the specified window.
func (w *Window) ShouldClose() (bool, error) {
	ret := C.glfwWindowShouldClose(w.data) != 0
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return false, err
	}
	return ret, nil
}

// SetShouldClose sets the value of the close flag of the window. This can be
// used to override the user's attempt to close the window, or to signal that it
// should be closed.
func (w *Window) SetShouldClose(value bool) error {
	if !value {
		C.glfwSetWindowShouldClose(w.data, C.int(False))
	} else {
		C.glfwSetWindowShouldClose(w.data, C.int(True))
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// SetTitle sets the window title, encoded as UTF-8, of the window.
//
// This function may only be called from the main thread.
func (w *Window) SetTitle(title string) error {
	t := C.CString(title)
	defer C.free(unsafe.Pointer(t))
	C.glfwSetWindowTitle(w.data, t)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// SetIcon sets the icon of the specified window. If passed an array of candidate images,
// those of or closest to the sizes desired by the system are selected. If no images are
// specified, the window reverts to its default icon.
//
// The image is ideally provided in the form of *image.NRGBA.
// The pixels are 32-bit, little-endian, non-premultiplied RGBA, i.e. eight
// bits per channel with the red channel first. They are arranged canonically
// as packed sequential rows, starting from the top-left corner. If the image
// type is not *image.NRGBA, it will be converted to it.
//
// The desired image sizes varies depending on platform and system settings. The selected
// images will be rescaled as needed. Good sizes include 16x16, 32x32 and 48x48.
func (w *Window) SetIcon(images []image.Image) error {
	count := len(images)
	glfwImgs := make([]C.GLFWimage, 0, count)

	for _, img := range images {
		glfwImg, free := imageToGLFWImage(img)
		defer free()
		glfwImgs = append(glfwImgs, glfwImg)
	}

	var p *C.GLFWimage
	if count > 0 {
		p = &glfwImgs[0]
	}
	C.glfwSetWindowIcon(w.data, C.int(count), p)

	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// GetPos returns the position, in screen coordinates, of the upper-left
// corner of the client area of the window.
func (w *Window) GetPos() (x, y int, err error) {
	var xpos, ypos C.int
	C.glfwGetWindowPos(w.data, &xpos, &ypos)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, 0, err
	}
	return int(xpos), int(ypos), nil
}

// SetPos sets the position, in screen coordinates, of the upper-left corner
// of the client area of the window.
//
// If it is a full screen window, this function does nothing.
//
// If you wish to set an initial window position you should create a hidden
// window (using Hint and Visible), set its position and then show it.
//
// It is very rarely a good idea to move an already visible window, as it will
// confuse and annoy the user.
//
// The window manager may put limits on what positions are allowed.
//
// This function may only be called from the main thread.
func (w *Window) SetPos(xpos, ypos int) error {
	C.glfwSetWindowPos(w.data, C.int(xpos), C.int(ypos))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// GetSize returns the size, in screen coordinates, of the client area of the
// specified window.
func (w *Window) GetSize() (width, height int, err error) {
	var wi, h C.int
	C.glfwGetWindowSize(w.data, &wi, &h)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, 0, err
	}
	return int(wi), int(h), nil
}

// SetSize sets the size, in screen coordinates, of the client area of the
// window.
//
// For full screen windows, this function selects and switches to the resolution
// closest to the specified size, without affecting the window's context. As the
// context is unaffected, the bit depths of the framebuffer remain unchanged.
//
// The window manager may put limits on what window sizes are allowed.
//
// This function may only be called from the main thread.
func (w *Window) SetSize(width, height int) error {
	C.glfwSetWindowSize(w.data, C.int(width), C.int(height))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// SetSizeLimits sets the size limits of the client area of the specified window.
// If the window is full screen or not resizable, this function does nothing.
//
// The size limits are applied immediately and may cause the window to be resized.
func (w *Window) SetSizeLimits(minw, minh, maxw, maxh int) error {
	C.glfwSetWindowSizeLimits(w.data, C.int(minw), C.int(minh), C.int(maxw), C.int(maxh))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// SetAspectRatio sets the required aspect ratio of the client area of the specified window.
// If the window is full screen or not resizable, this function does nothing.
//
// The aspect ratio is specified as a numerator and a denominator and both values must be greater
// than zero. For example, the common 16:9 aspect ratio is specified as 16 and 9, respectively.
//
// If the numerator and denominator is set to glfw.DontCare then the aspect ratio limit is disabled.
//
// The aspect ratio is applied immediately and may cause the window to be resized.
func (w *Window) SetAspectRatio(numer, denom int) error {
	C.glfwSetWindowAspectRatio(w.data, C.int(numer), C.int(denom))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// GetFramebufferSize retrieves the size, in pixels, of the framebuffer of the
// specified window.
func (w *Window) GetFramebufferSize() (width, height int, err error) {
	var wi, h C.int
	C.glfwGetFramebufferSize(w.data, &wi, &h)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, 0, err
	}
	return int(wi), int(h), nil
}

// GetFrameSize retrieves the size, in screen coordinates, of each edge of the frame
// of the specified window. This size includes the title bar, if the window has one.
// The size of the frame may vary depending on the window-related hints used to create it.
//
// Because this function retrieves the size of each window frame edge and not the offset
// along a particular coordinate axis, the retrieved values will always be zero or positive.
func (w *Window) GetFrameSize() (left, top, right, bottom int, err error) {
	var l, t, r, b C.int
	C.glfwGetWindowFrameSize(w.data, &l, &t, &r, &b)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, 0, 0, 0, err
	}
	return int(l), int(t), int(r), int(b), nil
}

// GetContentScale function retrieves the content scale for the specified
// window. The content scale is the ratio between the current DPI and the
// platform's default DPI. If you scale all pixel dimensions by this scale then
// your content should appear at an appropriate size. This is especially
// important for text and any UI elements.
//
// This function may only be called from the main thread.
func (w *Window) GetContentScale() (float32, float32) {
	var x, y C.float
	C.glfwGetWindowContentScale(w.data, &x, &y)
	return float32(x), float32(y)
}

// GetOpacity function returns the opacity of the window, including any
// decorations.
//
// The opacity (or alpha) value is a positive finite number between zero and
// one, where zero is fully transparent and one is fully opaque. If the system
// does not support whole window transparency, this function always returns one.
//
// The initial opacity value for newly created windows is one.
//
// This function may only be called from the main thread.
func (w *Window) GetOpacity() float32 {
	return float32(C.glfwGetWindowOpacity(w.data))
}

// SetOpacity function sets the opacity of the window, including any
// decorations. The opacity (or alpha) value is a positive finite number between
// zero and one, where zero is fully transparent and one is fully opaque.
//
// The initial opacity value for newly created windows is one.
//
// A window created with framebuffer transparency may not use whole window
// transparency. The results of doing this are undefined.
//
// This function may only be called from the main thread.
func (w *Window) SetOpacity(opacity float32) {
	C.glfwSetWindowOpacity(w.data, C.float(opacity))
}

// RequestAttention function requests user attention to the specified
// window. On platforms where this is not supported, attention is requested to
// the application as a whole.
//
// Once the user has given attention, usually by focusing the window or
// application, the system will end the request automatically.
//
// This function must only be called from the main thread.
func (w *Window) RequestAttention() error {
	C.glfwRequestWindowAttention(w.data)
	return nil
}

// Focus brings the specified window to front and sets input focus.
// The window should already be visible and not iconified.
//
// By default, both windowed and full screen mode windows are focused when initially created.
// Set the glfw.Focused to disable this behavior.
//
// Do not use this function to steal focus from other applications unless you are certain that
// is what the user wants. Focus stealing can be extremely disruptive.
func (w *Window) Focus() error {
	C.glfwFocusWindow(w.data)
	return nil
}

// Iconify iconifies/minimizes the window, if it was previously restored. If it
// is a full screen window, the original monitor resolution is restored until the
// window is restored. If the window is already iconified, this function does
// nothing.
//
// This function may only be called from the main thread.
func (w *Window) Iconify() error {
	C.glfwIconifyWindow(w.data)
	return nil
}

// Maximize maximizes the specified window if it was previously not maximized.
// If the window is already maximized, this function does nothing.
//
// If the specified window is a full screen window, this function does nothing.
func (w *Window) Maximize() error {
	C.glfwMaximizeWindow(w.data)
	return nil
}

// Restore restores the window, if it was previously iconified/minimized. If it
// is a full screen window, the resolution chosen for the window is restored on
// the selected monitor. If the window is already restored, this function does
// nothing.
//
// This function may only be called from the main thread.
func (w *Window) Restore() error {
	C.glfwRestoreWindow(w.data)
	return nil
}

// Show makes the window visible, if it was previously hidden. If the window is
// already visible or is in full screen mode, this function does nothing.
//
// This function may only be called from the main thread.
func (w *Window) Show() error {
	C.glfwShowWindow(w.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// Hide hides the window, if it was previously visible. If the window is already
// hidden or is in full screen mode, this function does nothing.
//
// This function may only be called from the main thread.
func (w *Window) Hide() error {
	C.glfwHideWindow(w.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// GetMonitor returns the handle of the monitor that the window is in
// fullscreen on.
//
// Returns nil if the window is in windowed mode.
func (w *Window) GetMonitor() (*Monitor, error) {
	m := C.glfwGetWindowMonitor(w.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	if m == nil {
		return nil, nil
	}
	return &Monitor{m}, nil
}

// SetMonitor sets the monitor that the window uses for full screen mode or,
// if the monitor is NULL, makes it windowed mode.
//
// When setting a monitor, this function updates the width, height and refresh
// rate of the desired video mode and switches to the video mode closest to it.
// The window position is ignored when setting a monitor.
//
// When the monitor is NULL, the position, width and height are used to place
// the window client area. The refresh rate is ignored when no monitor is specified.
// If you only wish to update the resolution of a full screen window or the size of
// a windowed mode window, see window.SetSize.
//
// When a window transitions from full screen to windowed mode, this function
// restores any previous window settings such as whether it is decorated, floating,
// resizable, has size or aspect ratio limits, etc..
func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) error {
	var m *C.GLFWmonitor
	if monitor == nil {
		m = nil
	} else {
		m = monitor.data
	}
	C.glfwSetWindowMonitor(w.data, m, C.int(xpos), C.int(ypos), C.int(width), C.int(height), C.int(refreshRate))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// GetAttrib returns an attribute of the window. There are many attributes,
// some related to the window and others to its context.
func (w *Window) GetAttrib(attrib Hint) (int, error) {
	ret := int(C.glfwGetWindowAttrib(w.data, C.int(attrib)))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, err
	}
	return ret, nil
}

// SetAttrib function sets the value of an attribute of the specified window.
//
// The supported attributes are Decorated, Resizeable, Floating and AutoIconify.
//
// Some of these attributes are ignored for full screen windows. The new value
// will take effect if the window is later made windowed.
//
// Some of these attributes are ignored for windowed mode windows. The new value
// will take effect if the window is later made full screen.
//
// This function may only be called from the main thread.
func (w *Window) SetAttrib(attrib Hint, value int) error {
	C.glfwSetWindowAttrib(w.data, C.int(attrib), C.int(value))
	return nil
}

// SetUserPointer sets the user-defined pointer of the window. The current value
// is retained until the window is destroyed. The initial value is nil.
func (w *Window) SetUserPointer(pointer unsafe.Pointer) error {
	C.glfwSetWindowUserPointer(w.data, pointer)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// GetUserPointer returns the current value of the user-defined pointer of the
// window. The initial value is nil.
func (w *Window) GetUserPointer() (unsafe.Pointer, error) {
	ret := C.glfwGetWindowUserPointer(w.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return ret, nil
}

// PosCallback is the window position callback.
type PosCallback func(w *Window, xpos int, ypos int)

// SetPosCallback sets the position callback of the window, which is called
// when the window is moved. The callback is provided with the screen position
// of the upper-left corner of the client area of the window.
func (w *Window) SetPosCallback(cbfun PosCallback) (previous PosCallback, err error) {
	previous = w.fPosHolder
	w.fPosHolder = cbfun
	if cbfun == nil {
		C.glfwSetWindowPosCallback(w.data, nil)
	} else {
		C.glfwSetWindowPosCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// SizeCallback is the window size callback.
type SizeCallback func(w *Window, width int, height int)

// SetSizeCallback sets the size callback of the window, which is called when
// the window is resized. The callback is provided with the size, in screen
// coordinates, of the client area of the window.
func (w *Window) SetSizeCallback(cbfun SizeCallback) (previous SizeCallback, err error) {
	previous = w.fSizeHolder
	w.fSizeHolder = cbfun
	if cbfun == nil {
		C.glfwSetWindowSizeCallback(w.data, nil)
	} else {
		C.glfwSetWindowSizeCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// FramebufferSizeCallback is the framebuffer size callback.
type FramebufferSizeCallback func(w *Window, width int, height int)

// SetFramebufferSizeCallback sets the framebuffer resize callback of the specified
// window, which is called when the framebuffer of the specified window is resized.
func (w *Window) SetFramebufferSizeCallback(cbfun FramebufferSizeCallback) (previous FramebufferSizeCallback, err error) {
	previous = w.fFramebufferSizeHolder
	w.fFramebufferSizeHolder = cbfun
	if cbfun == nil {
		C.glfwSetFramebufferSizeCallback(w.data, nil)
	} else {
		C.glfwSetFramebufferSizeCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// CloseCallback is the window close callback.
type CloseCallback func(w *Window)

// SetCloseCallback sets the close callback of the window, which is called when
// the user attempts to close the window, for example by clicking the close
// widget in the title bar.
//
// The close flag is set before this callback is called, but you can modify it at
// any time with SetShouldClose.
//
// Mac OS X: Selecting Quit from the application menu will trigger the close
// callback for all windows.
func (w *Window) SetCloseCallback(cbfun CloseCallback) (previous CloseCallback, err error) {
	previous = w.fCloseHolder
	w.fCloseHolder = cbfun
	if cbfun == nil {
		C.glfwSetWindowCloseCallback(w.data, nil)
	} else {
		C.glfwSetWindowCloseCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// MaximizeCallback is the function signature for window maximize callback
// functions.
type MaximizeCallback func(w *Window, maximized bool)

// SetMaximizeCallback sets the maximization callback of the specified window,
// which is called when the window is maximized or restored.
//
// This function must only be called from the main thread.
func (w *Window) SetMaximizeCallback(cbfun MaximizeCallback) MaximizeCallback {
	previous := w.fMaximizeHolder
	w.fMaximizeHolder = cbfun
	if cbfun == nil {
		C.glfwSetWindowMaximizeCallback(w.data, nil)
	} else {
		C.glfwSetWindowMaximizeCallbackCB(w.data)
	}
	return previous
}

// ContentScaleCallback is the function signature for window content scale
// callback functions.
type ContentScaleCallback func(w *Window, x float32, y float32)

// SetContentScaleCallback function sets the window content scale callback of
// the specified window, which is called when the content scale of the specified
// window changes.
//
// This function must only be called from the main thread.
func (w *Window) SetContentScaleCallback(cbfun ContentScaleCallback) (ContentScaleCallback, error) {
	previous := w.fContentScaleHolder
	w.fContentScaleHolder = cbfun
	if cbfun == nil {
		C.glfwSetWindowContentScaleCallback(w.data, nil)
	} else {
		C.glfwSetWindowContentScaleCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// RefreshCallback is the window refresh callback.
type RefreshCallback func(w *Window)

// SetRefreshCallback sets the refresh callback of the window, which
// is called when the client area of the window needs to be redrawn, for example
// if the window has been exposed after having been covered by another window.
//
// On compositing window systems such as Aero, Compiz or Aqua, where the window
// contents are saved off-screen, this callback may be called only very
// infrequently or never at all.
func (w *Window) SetRefreshCallback(cbfun RefreshCallback) (previous RefreshCallback, err error) {
	previous = w.fRefreshHolder
	w.fRefreshHolder = cbfun
	if cbfun == nil {
		C.glfwSetWindowRefreshCallback(w.data, nil)
	} else {
		C.glfwSetWindowRefreshCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// FocusCallback is the window focus callback.
type FocusCallback func(w *Window, focused bool)

// SetFocusCallback sets the focus callback of the window, which is called when
// the window gains or loses focus.
//
// After the focus callback is called for a window that lost focus, synthetic key
// and mouse button release events will be generated for all such that had been
// pressed. For more information, see SetKeyCallback and SetMouseButtonCallback.
func (w *Window) SetFocusCallback(cbfun FocusCallback) (previous FocusCallback, err error) {
	previous = w.fFocusHolder
	w.fFocusHolder = cbfun
	if cbfun == nil {
		C.glfwSetWindowFocusCallback(w.data, nil)
	} else {
		C.glfwSetWindowFocusCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// IconifyCallback is the window iconification callback.
type IconifyCallback func(w *Window, iconified bool)

// SetIconifyCallback sets the iconification callback of the window, which is
// called when the window is iconified or restored.
func (w *Window) SetIconifyCallback(cbfun IconifyCallback) (previous IconifyCallback, err error) {
	previous = w.fIconifyHolder
	w.fIconifyHolder = cbfun
	if cbfun == nil {
		C.glfwSetWindowIconifyCallback(w.data, nil)
	} else {
		C.glfwSetWindowIconifyCallbackCB(w.data)
	}
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return previous, nil
}

// SetClipboardString sets the system clipboard to the specified UTF-8 encoded
// string.
//
// Ownership to the Window is no longer necessary, see
// glfw.SetClipboardString(string)
//
// This function may only be called from the main thread.
func (w *Window) SetClipboardString(str string) error {
	cp := C.CString(str)
	defer C.free(unsafe.Pointer(cp))
	C.glfwSetClipboardString(w.data, cp)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// GetClipboardString returns the contents of the system clipboard, if it
// contains or is convertible to a UTF-8 encoded string.
//
// Ownership to the Window is no longer necessary, see
// glfw.GetClipboardString()
//
// This function may only be called from the main thread.
func (w *Window) GetClipboardString() (string, error) {
	cs := C.glfwGetClipboardString(w.data)
	if cs == nil {
		if err := fetchErrorIgnoringPlatformError(); err != nil {
			if errors.Is(err, FormatUnavailable) {
				return "", nil
			}
			return "", err
		}
		return "", nil
	}
	return C.GoString(cs), nil
}

// PollEvents processes only those events that have already been received and
// then returns immediately. Processing events will cause the window and input
// callbacks associated with those events to be called.
//
// This function is not required for joystick input to work.
//
// This function may not be called from a callback.
//
// This function may only be called from the main thread.
func PollEvents() error {
	C.glfwPollEvents()
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// WaitEvents puts the calling thread to sleep until at least one event has been
// received. Once one or more events have been recevied, it behaves as if
// PollEvents was called, i.e. the events are processed and the function then
// returns immediately. Processing events will cause the window and input
// callbacks associated with those events to be called.
//
// Since not all events are associated with callbacks, this function may return
// without a callback having been called even if you are monitoring all
// callbacks.
//
// This function may not be called from a callback.
//
// This function may only be called from the main thread.
func WaitEvents() error {
	C.glfwWaitEvents()
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// WaitEventsTimeout puts the calling thread to sleep until at least one event is available in the
// event queue, or until the specified timeout is reached. If one or more events are available,
// it behaves exactly like PollEvents, i.e. the events in the queue are processed and the function
// then returns immediately. Processing events will cause the window and input callbacks associated
// with those events to be called.
//
// The timeout value must be a positive finite number.
//
// Since not all events are associated with callbacks, this function may return without a callback
// having been called even if you are monitoring all callbacks.
//
// On some platforms, a window move, resize or menu operation will cause event processing to block.
// This is due to how event processing is designed on those platforms. You can use the window
// refresh callback to redraw the contents of your window when necessary during such operations.
//
// On some platforms, certain callbacks may be called outside of a call to one of the event
// processing functions.
//
// If no windows exist, this function returns immediately. For synchronization of threads in
// applications that do not create windows, use native Go primitives.
//
// Event processing is not required for joystick input to work.
func WaitEventsTimeout(timeout float64) error {
	C.glfwWaitEventsTimeout(C.double(timeout))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// PostEmptyEvent posts an empty event from the current thread to the main
// thread event queue, causing WaitEvents to return.
//
// If no windows exist, this function returns immediately. For synchronization of threads in
// applications that do not create windows, use native Go primitives.
//
// This function may be called from secondary threads.
func PostEmptyEvent() error {
	C.glfwPostEmptyEvent()
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}
