// SPDX-License-Identifier: BSD-3-Clause
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build (freebsd || linux || netbsd || openbsd) && wayland

package glfw

//#include <stdlib.h>
//#define GLFW_EXPOSE_NATIVE_WAYLAND
//#define GLFW_EXPOSE_NATIVE_EGL
//#define GLFW_INCLUDE_NONE
//#include "glfw/include/GLFW/glfw3.h"
//#include "glfw/include/GLFW/glfw3native.h"
import "C"

func GetWaylandDisplay() *C.struct_wl_display {
	ret := C.glfwGetWaylandDisplay()
	panicError()
	return ret
}

func (m *Monitor) GetWaylandMonitor() *C.struct_wl_output {
	ret := C.glfwGetWaylandMonitor(m.data)
	panicError()
	return ret
}

func (w *Window) GetWaylandWindow() *C.struct_wl_surface {
	ret := C.glfwGetWaylandWindow(w.data)
	panicError()
	return ret
}

func GetEGLDisplay() C.EGLDisplay {
	ret := C.glfwGetEGLDisplay()
	panicError()
	return ret
}

func (w *Window) GetEGLContext() C.EGLContext {
	ret := C.glfwGetEGLContext(w.data)
	panicError()
	return ret
}

func (w *Window) GetEGLSurface() C.EGLSurface {
	ret := C.glfwGetEGLSurface(w.data)
	panicError()
	return ret
}
