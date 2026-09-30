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

type api struct {
	Initialize          func(display uintptr, major, minor *int32) bool
	Terminate           func(display uintptr) bool
	BindAPI             func(api int32) bool
	ChooseConfig        func(display uintptr, attribList *int32, configs *uintptr, configSize int32, numConfig *int32) bool
	GetConfigAttrib     func(display, config uintptr, attribute int32, value *int32) bool
	CreateWindowSurface func(display, config, win uintptr, attribList *int32) uintptr
	CreateContext       func(display, config, shareContext uintptr, attribList *int32) uintptr
	DestroySurface      func(display, surface uintptr) bool
	DestroyContext      func(display, ctx uintptr) bool
	MakeCurrent         func(display, draw, read, ctx uintptr) bool
	SwapBuffers         func(display, surface uintptr) bool
	SwapInterval        func(display uintptr, interval int32) bool
	QuerySurface        func(display, surface uintptr, attribute int32, value *int32) bool
	GetError            func() int32
}

// NewContext loads the common EGL entry points. The caller selects the native
// display and window before creating the surface and context.
func NewContext() (*Context, error) {
	c := &Context{swapInterval: -1}
	var errs []error
	for _, name := range []string{"libEGL.so.1", "libEGL.so"} {
		lib, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		c.lib = lib
		break
	}
	if c.lib == 0 {
		return nil, fmt.Errorf("egl: failed to load libEGL: %w", errors.Join(errs...))
	}
	for _, f := range []struct {
		ptr  any
		name string
	}{
		{&c.api.Initialize, "eglInitialize"},
		{&c.api.Terminate, "eglTerminate"},
		{&c.api.BindAPI, "eglBindAPI"},
		{&c.api.ChooseConfig, "eglChooseConfig"},
		{&c.api.GetConfigAttrib, "eglGetConfigAttrib"},
		{&c.api.CreateWindowSurface, "eglCreateWindowSurface"},
		{&c.api.CreateContext, "eglCreateContext"},
		{&c.api.DestroySurface, "eglDestroySurface"},
		{&c.api.DestroyContext, "eglDestroyContext"},
		{&c.api.MakeCurrent, "eglMakeCurrent"},
		{&c.api.SwapBuffers, "eglSwapBuffers"},
		{&c.api.SwapInterval, "eglSwapInterval"},
		{&c.api.QuerySurface, "eglQuerySurface"},
		{&c.api.GetError, "eglGetError"},
	} {
		if err := c.RegisterFunc(f.ptr, f.name); err != nil {
			return nil, errors.Join(err, c.Close())
		}
	}
	return c, nil
}

// RegisterFunc loads a backend-specific EGL entry point from the same library.
func (c *Context) RegisterFunc(ptr any, name string) error {
	sym, err := purego.Dlsym(c.lib, name)
	if err != nil {
		return fmt.Errorf("egl: %s not found in libEGL: %w", name, err)
	}
	if sym == 0 {
		return fmt.Errorf("egl: %s not found in libEGL", name)
	}
	purego.RegisterFunc(ptr, sym)
	return nil
}

// RegisterProcFunc loads an EGL extension entry point through eglGetProcAddress.
func (c *Context) RegisterProcFunc(ptr any, name string) error {
	var getProcAddress func(*byte) uintptr
	if err := c.RegisterFunc(&getProcAddress, "eglGetProcAddress"); err != nil {
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
