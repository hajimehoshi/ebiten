// Copyright 2023 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build nintendosdk

package opengl

// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
//
// #include <EGL/egl.h>
// #include <EGL/eglext.h>
import "C"

import (
	"fmt"
)

type egl struct {
	display C.EGLDisplay
	surface C.EGLSurface
	context C.EGLContext
}

func newEGL(nativeWindowHandle uintptr) (*egl, error) {
	e := &egl{}

	e.display = C.eglGetDisplay(C.NativeDisplayType(C.EGL_DEFAULT_DISPLAY))
	if e.display == 0 {
		return nil, fmt.Errorf("opengl: eglGetDisplay failed")
	}

	if r := C.eglInitialize(e.display, nil, nil); r == 0 {
		return nil, fmt.Errorf("opengl: eglInitialize failed")
	}

	configAttribs := []C.EGLint{
		C.EGL_RENDERABLE_TYPE, C.EGL_OPENGL_BIT,
		C.EGL_SURFACE_TYPE, C.EGL_WINDOW_BIT,
		C.EGL_RED_SIZE, 8,
		C.EGL_GREEN_SIZE, 8,
		C.EGL_BLUE_SIZE, 8,
		C.EGL_ALPHA_SIZE, 8,
		C.EGL_NONE}
	var numConfigs C.EGLint
	var config C.EGLConfig
	if r := C.eglChooseConfig(e.display, &configAttribs[0], &config, 1, &numConfigs); r == 0 {
		return nil, fmt.Errorf("opengl: eglChooseConfig failed")
	}
	if numConfigs != 1 {
		return nil, fmt.Errorf("opengl: eglChooseConfig failed: numConfigs must be 1 but %d", numConfigs)
	}

	e.surface = C.eglCreateWindowSurface(e.display, config, C.NativeWindowType(nativeWindowHandle), nil)
	if e.surface == C.EGLSurface(C.EGL_NO_SURFACE) {
		return nil, fmt.Errorf("opengl: eglCreateWindowSurface failed")
	}

	// Set the current rendering API.
	if r := C.eglBindAPI(C.EGL_OPENGL_API); r == 0 {
		return nil, fmt.Errorf("opengl: eglBindAPI failed")
	}

	// Create new context and set it as current.
	contextAttribs := []C.EGLint{
		// Set target graphics api version.
		C.EGL_CONTEXT_MAJOR_VERSION, 3,
		C.EGL_CONTEXT_MINOR_VERSION, 2,
		// For debug callback
		C.EGL_CONTEXT_FLAGS_KHR, C.EGL_CONTEXT_OPENGL_DEBUG_BIT_KHR,
		C.EGL_NONE}
	e.context = C.eglCreateContext(e.display, config, C.EGLContext(C.EGL_NO_CONTEXT), &contextAttribs[0])
	if e.context == C.EGLContext(C.EGL_NO_CONTEXT) {
		return nil, fmt.Errorf("opengl: eglCreateContext failed: error: %d", C.eglGetError())
	}

	return e, nil
}

func (e *egl) makeContextCurrent() error {
	if r := C.eglMakeCurrent(e.display, e.surface, e.surface, e.context); r == 0 {
		return fmt.Errorf("opengl: eglMakeCurrent failed")
	}
	return nil
}

func (e *egl) swapBuffers() {
	C.eglSwapBuffers(e.display, e.surface)
}
