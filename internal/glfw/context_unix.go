// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd || openbsd

package glfw

//#include <stdlib.h>
//#define GLFW_INCLUDE_NONE
//#include "glfw3_unix.h"
import "C"

import (
	"unsafe"
)

// MakeContextCurrent makes the context of the window current.
// Originally GLFW 3 passes a null pointer to detach the context.
// But since we're using receivers, DetachCurrentContext should
// be used instead.
func (w *Window) MakeContextCurrent() error {
	C.glfwMakeContextCurrent(w.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// DetachCurrentContext detaches the current context.
func DetachCurrentContext() error {
	C.glfwMakeContextCurrent(nil)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// GetCurrentContext returns the window whose context is current.
func GetCurrentContext() (*Window, error) {
	w := C.glfwGetCurrentContext()
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	if w == nil {
		return nil, nil
	}
	return windows.get(w), nil
}

// SwapBuffers swaps the front and back buffers of the window. If the
// swap interval is greater than zero, the GPU driver waits the specified number
// of screen updates before swapping the buffers.
func (w *Window) SwapBuffers() error {
	C.glfwSwapBuffers(w.data)
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// SwapInterval sets the swap interval for the current context, i.e. the number
// of screen updates to wait before swapping the buffers of a window and
// returning from SwapBuffers. This is sometimes called
// 'vertical synchronization', 'vertical retrace synchronization' or 'vsync'.
//
// Contexts that support either of the WGL_EXT_swap_control_tear and
// GLX_EXT_swap_control_tear extensions also accept negative swap intervals,
// which allow the driver to swap even if a frame arrives a little bit late.
// You can check for the presence of these extensions using
// ExtensionSupported. For more information about swap tearing,
// see the extension specifications.
//
// Some GPU drivers do not honor the requested swap interval, either because of
// user settings that override the request or due to bugs in the driver.
func (w *Window) SwapInterval(interval int) error {
	C.glfwSwapInterval(w.data, C.int(interval))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return err
	}
	return nil
}

// ExtensionSupported reports whether the specified OpenGL or context creation
// API extension is supported by the current context. For example, on Windows
// both the OpenGL and WGL extension strings are checked.
//
// As this functions searches one or more extension strings on each call, it is
// recommended that you cache its results if it's going to be used frequently.
// The extension strings will not change during the lifetime of a context, so
// there is no danger in doing this.
func (w *Window) ExtensionSupported(extension string) (bool, error) {
	e := C.CString(extension)
	defer C.free(unsafe.Pointer(e))
	ret := C.glfwExtensionSupported(w.data, e) != 0
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return false, err
	}
	return ret, nil
}

// GetProcAddress returns the address of the specified OpenGL or OpenGL ES core
// or extension function, if it is supported by the current context.
//
// A context must be current on the calling thread. Calling this function
// without a current context will cause a GLFW_NO_CURRENT_CONTEXT error.
//
// This function is used to provide GL proc resolving capabilities to an
// external C library.
func GetProcAddress(procname string) (unsafe.Pointer, error) {
	p := C.CString(procname)
	defer C.free(unsafe.Pointer(p))
	ret := unsafe.Pointer(C.glfwGetProcAddress(p))
	if err := fetchErrorIgnoringPlatformError(); err != nil {
		return nil, err
	}
	return ret, nil
}
