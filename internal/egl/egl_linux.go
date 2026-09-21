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

// Package egl manages the EGL context shared by the fbdev and GBM backends.
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

// Context owns the EGL display, surface, and OpenGL ES 3 context.
type Context struct {
	api           api
	lib           uintptr
	display       uintptr
	surface       uintptr
	context       uintptr
	width, height int
	swapInterval  int
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

func (c *Context) Initialize(display uintptr) error {
	if display == 0 {
		return fmt.Errorf("egl: no display: %w", c.LastError())
	}
	var major, minor int32
	if !c.api.Initialize(display, &major, &minor) {
		return fmt.Errorf("egl: eglInitialize failed: %w", c.LastError())
	}
	c.display = display
	if !c.api.BindAPI(OpenGLESAPI) {
		return fmt.Errorf("egl: eglBindAPI failed: %w", c.LastError())
	}
	return nil
}

// ChooseConfig asks for one matching configuration. Some fbdev EGL drivers
// expect a config buffer rather than a count-only query.
func (c *Context) ChooseConfig(attribs []int32) (uintptr, error) {
	var config uintptr
	var num int32
	if !c.api.ChooseConfig(c.display, &attribs[0], &config, 1, &num) {
		return 0, fmt.Errorf("egl: eglChooseConfig failed: %w", c.LastError())
	}
	if num <= 0 {
		return 0, errors.New("egl: no matching EGL config")
	}
	return config, nil
}

// ChooseConfigs returns every matching configuration so a backend can select
// the one compatible with its native window format.
func (c *Context) ChooseConfigs(attribs []int32) ([]uintptr, error) {
	var num int32
	if !c.api.ChooseConfig(c.display, &attribs[0], nil, 0, &num) {
		return nil, fmt.Errorf("egl: eglChooseConfig failed: %w", c.LastError())
	}
	if num <= 0 {
		return nil, errors.New("egl: no matching EGL config")
	}
	configs := make([]uintptr, num)
	if !c.api.ChooseConfig(c.display, &attribs[0], &configs[0], int32(len(configs)), &num) {
		return nil, fmt.Errorf("egl: eglChooseConfig failed: %w", c.LastError())
	}
	if num <= 0 {
		return nil, errors.New("egl: no matching EGL config")
	}
	return configs[:num], nil
}

func (c *Context) ConfigAttrib(config uintptr, attribute int32) (int32, error) {
	var value int32
	if !c.api.GetConfigAttrib(c.display, config, attribute, &value) {
		return 0, fmt.Errorf("egl: eglGetConfigAttrib failed: %w", c.LastError())
	}
	return value, nil
}

func (c *Context) CreateWindowSurface(config, window uintptr, attribs []int32) error {
	var p *int32
	if len(attribs) != 0 {
		p = &attribs[0]
	}
	c.surface = c.api.CreateWindowSurface(c.display, config, window, p)
	if c.surface == 0 {
		return fmt.Errorf("egl: eglCreateWindowSurface failed: %w", c.LastError())
	}
	return nil
}

func (c *Context) QuerySurface(attribute int32) (int32, error) {
	var value int32
	if !c.api.QuerySurface(c.display, c.surface, attribute, &value) {
		return 0, fmt.Errorf("egl: eglQuerySurface failed: %w", c.LastError())
	}
	return value, nil
}

func (c *Context) CreateES3Context(config uintptr) error {
	attribs := []int32{ContextClientVersion, 3, None}
	c.context = c.api.CreateContext(c.display, config, 0, &attribs[0])
	if c.context == 0 {
		return fmt.Errorf("egl: eglCreateContext failed: %w", c.LastError())
	}
	return nil
}

func (c *Context) SetSize(width, height int) { c.width, c.height = width, height }
func (c *Context) Size() (int, int)          { return c.width, c.height }

func (c *Context) MakeContextCurrent() error {
	if !c.api.MakeCurrent(c.display, c.surface, c.surface, c.context) {
		return fmt.Errorf("egl: eglMakeCurrent failed: %w", c.LastError())
	}
	return nil
}

func (c *Context) SwapInterval(interval int) error {
	if c.swapInterval == interval {
		return nil
	}
	if !c.api.SwapInterval(c.display, int32(interval)) {
		return fmt.Errorf("egl: eglSwapInterval failed: %w", c.LastError())
	}
	c.swapInterval = interval
	return nil
}

func (c *Context) SwapBuffers() error {
	if !c.api.SwapBuffers(c.display, c.surface) {
		return fmt.Errorf("egl: eglSwapBuffers failed: %w", c.LastError())
	}
	return nil
}

// Unbind releases the current context before a backend frees scanout buffers.
func (c *Context) Unbind() {
	if c.display != 0 && (c.context != 0 || c.surface != 0) {
		c.api.MakeCurrent(c.display, 0, 0, 0)
	}
}

func (c *Context) Close() error {
	if c.display != 0 {
		c.Unbind()
		if c.context != 0 {
			c.api.DestroyContext(c.display, c.context)
			c.context = 0
		}
		if c.surface != 0 {
			c.api.DestroySurface(c.display, c.surface)
			c.surface = 0
		}
		c.api.Terminate(c.display)
		c.display = 0
	}
	if c.lib != 0 {
		lib := c.lib
		c.lib = 0
		return purego.Dlclose(lib)
	}
	return nil
}

type eglError int32

func (e eglError) Error() string { return fmt.Sprintf("EGL error 0x%x", int32(e)) }

func (c *Context) LastError() error {
	code := c.api.GetError()
	if code == Success {
		return errors.New("EGL reported no error")
	}
	return eglError(code)
}
