// Copyright 2026 The Ebitengine Authors
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

package egl

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

const (
	None                 = 0x3038
	SurfaceType          = 0x3033
	WindowBit            = 0x0004
	RenderableType       = 0x3040
	OpenGLES3Bit         = 0x0040
	RedSize              = 0x3024
	GreenSize            = 0x3023
	BlueSize             = 0x3022
	AlphaSize            = 0x3021
	NativeVisualID       = 0x302e
	OpenGLESAPI          = 0x30a0
	ContextClientVersion = 0x3098
	Width                = 0x3057
	Height               = 0x3056
	Success              = 0x3000
)

var (
	eglInitialize          func(display uintptr, major, minor *int32) bool
	eglTerminate           func(display uintptr) bool
	eglBindAPI             func(api int32) bool
	eglChooseConfig        func(display uintptr, attribList *int32, configs *uintptr, configSize int32, numConfig *int32) bool
	eglGetConfigAttrib     func(display, config uintptr, attribute int32, value *int32) bool
	eglCreateWindowSurface func(display, config, win uintptr, attribList *int32) uintptr
	eglCreateContext       func(display, config, shareContext uintptr, attribList *int32) uintptr
	eglDestroySurface      func(display, surface uintptr) bool
	eglDestroyContext      func(display, ctx uintptr) bool
	eglMakeCurrent         func(display, draw, read, ctx uintptr) bool
	eglSwapBuffers         func(display, surface uintptr) bool
	eglSwapInterval        func(display uintptr, interval int32) bool
	eglQuerySurface        func(display, surface uintptr, attribute int32, value *int32) bool
	eglGetError            func() int32
)

var (
	libEGL   uintptr
	loadOnce sync.Once
	loadErr  error
)

func load() error {
	loadOnce.Do(func() {
		loadErr = loadImpl()
	})
	return loadErr
}

func loadImpl() error {
	var errs []error
	for _, name := range []string{"libEGL.so.1", "libEGL.so"} {
		lib, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		libEGL = lib
		break
	}
	if libEGL == 0 {
		return fmt.Errorf("egl: failed to load libEGL: %w", errors.Join(errs...))
	}
	for _, f := range []struct {
		ptr  any
		name string
	}{
		{&eglInitialize, "eglInitialize"},
		{&eglTerminate, "eglTerminate"},
		{&eglBindAPI, "eglBindAPI"},
		{&eglChooseConfig, "eglChooseConfig"},
		{&eglGetConfigAttrib, "eglGetConfigAttrib"},
		{&eglCreateWindowSurface, "eglCreateWindowSurface"},
		{&eglCreateContext, "eglCreateContext"},
		{&eglDestroySurface, "eglDestroySurface"},
		{&eglDestroyContext, "eglDestroyContext"},
		{&eglMakeCurrent, "eglMakeCurrent"},
		{&eglSwapBuffers, "eglSwapBuffers"},
		{&eglSwapInterval, "eglSwapInterval"},
		{&eglQuerySurface, "eglQuerySurface"},
		{&eglGetError, "eglGetError"},
	} {
		if err := registerFunc(f.ptr, f.name); err != nil {
			return err
		}
	}
	return nil
}

func registerFunc(ptr any, name string) error {
	sym, err := purego.Dlsym(libEGL, name)
	if err != nil {
		return fmt.Errorf("egl: %s not found in libEGL: %w", name, err)
	}
	if sym == 0 {
		return fmt.Errorf("egl: %s not found in libEGL", name)
	}
	purego.RegisterFunc(ptr, sym)
	return nil
}

// RegisterFunc loads a backend-specific EGL entry point from libEGL.
func (c *Context) RegisterFunc(ptr any, name string) error {
	return registerFunc(ptr, name)
}

// RegisterProcFunc loads an EGL extension entry point through eglGetProcAddress.
func (c *Context) RegisterProcFunc(ptr any, name string) error {
	var getProcAddress func(*byte) uintptr
	if err := registerFunc(&getProcAddress, "eglGetProcAddress"); err != nil {
		return err
	}
	nameBytes := append([]byte(name), 0)
	sym := getProcAddress(&nameBytes[0])
	if sym == 0 {
		return fmt.Errorf("egl: %s not found through eglGetProcAddress", name)
	}
	purego.RegisterFunc(ptr, sym)
	return nil
}
