// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build freebsd || linux || netbsd || openbsd

package glfw

//#include <stdlib.h>
//#define GLFW_EXPOSE_NATIVE_X11
//#define GLFW_EXPOSE_NATIVE_GLX
//#define GLFW_INCLUDE_NONE
//#include "glfw3_unix.h"
//#include "glfw3native_unix.h"
import "C"
import "unsafe"

func GetX11Display() (*C.Display, error) {
	ret := C.glfwGetX11Display()
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return ret, nil
}

// GetX11Adapter returns the RRCrtc of the monitor.
func (m *Monitor) GetX11Adapter() (C.RRCrtc, error) {
	ret := C.glfwGetX11Adapter(m.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, err
	}
	return ret, nil
}

// GetX11Monitor returns the RROutput of the monitor.
func (m *Monitor) GetX11Monitor() (C.RROutput, error) {
	ret := C.glfwGetX11Monitor(m.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, err
	}
	return ret, nil
}

// GetX11Window returns the Window of the window.
func (w *Window) GetX11Window() (C.Window, error) {
	ret := C.glfwGetX11Window(w.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, err
	}
	return ret, nil
}

// GetGLXContext returns the GLXContext of the window.
func (w *Window) GetGLXContext() (C.GLXContext, error) {
	ret := C.glfwGetGLXContext(w.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return ret, nil
}

// GetGLXWindow returns the GLXWindow of the window.
func (w *Window) GetGLXWindow() (C.GLXWindow, error) {
	ret := C.glfwGetGLXWindow(w.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return 0, err
	}
	return ret, nil
}

// SetX11SelectionString sets the X11 selection string.
func SetX11SelectionString(str string) {
	s := C.CString(str)
	defer C.free(unsafe.Pointer(s))
	C.glfwSetX11SelectionString(s)
}

// GetX11SelectionString gets the X11 selection string.
func GetX11SelectionString() string {
	s := C.glfwGetX11SelectionString()
	return C.GoString(s)
}
