// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

package glfw

/*
#define GLFW_EXPOSE_NATIVE_COCOA
#define GLFW_EXPOSE_NATIVE_NSGL
#include "glfw3_unix.h"
#include "glfw3native_unix.h"

// workaround wrappers needed due to a cgo and/or LLVM bug.
// See: https://github.com/go-gl/glfw/issues/136
static void *workaround_glfwGetCocoaWindow(GLFWwindow *w) {
	return (void *)glfwGetCocoaWindow(w);
}
static void *workaround_glfwGetNSGLContext(GLFWwindow *w) {
	return (void *)glfwGetNSGLContext(w);
}
*/
import "C"
import "unsafe"

// GetCocoaMonitor returns the CGDirectDisplayID of the monitor.
func (m *Monitor) GetCocoaMonitor() (uintptr, error) {
	ret := uintptr(C.glfwGetCocoaMonitor(m.data))
	return ret, fetchErrorIgnoringPlatformError()
}

// GetCocoaWindow returns the NSWindow of the window.
func (w *Window) GetCocoaWindow() (uintptr, error) {
	ret := uintptr(C.workaround_glfwGetCocoaWindow(w.data))
	return ret, fetchErrorIgnoringPlatformError()
}

// GetNSGLContext returns the NSOpenGLContext of the window.
func (w *Window) GetNSGLContext() (unsafe.Pointer, error) {
	ret := C.workaround_glfwGetNSGLContext(w.data)
	return ret, fetchErrorIgnoringPlatformError()
}
