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
)

// Context owns the EGL display, surface, and OpenGL ES 3 context.
type Context struct {
	display       uintptr
	surface       uintptr
	context       uintptr
	width, height int
	swapInterval  int
}

// NewContext loads the common EGL entry points. The caller selects the native
// display and window before creating the surface and context.
func NewContext() (*Context, error) {
	if err := load(); err != nil {
		return nil, err
	}
	return &Context{swapInterval: -1}, nil
}

// Initialize binds OpenGL ES to the native EGL display.
func (c *Context) Initialize(display uintptr) error {
	if display == 0 {
		return fmt.Errorf("egl: no display: %w", c.LastError())
	}
	var major, minor int32
	if !eglInitialize(display, &major, &minor) {
		return fmt.Errorf("egl: eglInitialize failed: %w", c.LastError())
	}
	c.display = display
	if !eglBindAPI(OpenGLESAPI) {
		return fmt.Errorf("egl: eglBindAPI failed: %w", c.LastError())
	}
	return nil
}

// ChooseConfig asks for one matching configuration. Some fbdev EGL drivers
// expect a config buffer rather than a count-only query.
func (c *Context) ChooseConfig(attribs []int32) (uintptr, error) {
	var config uintptr
	var num int32
	if !eglChooseConfig(c.display, &attribs[0], &config, 1, &num) {
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
	if !eglChooseConfig(c.display, &attribs[0], nil, 0, &num) {
		return nil, fmt.Errorf("egl: eglChooseConfig failed: %w", c.LastError())
	}
	if num <= 0 {
		return nil, errors.New("egl: no matching EGL config")
	}
	configs := make([]uintptr, num)
	if !eglChooseConfig(c.display, &attribs[0], &configs[0], int32(len(configs)), &num) {
		return nil, fmt.Errorf("egl: eglChooseConfig failed: %w", c.LastError())
	}
	if num <= 0 {
		return nil, errors.New("egl: no matching EGL config")
	}
	return configs[:num], nil
}

// ConfigAttrib returns an attribute of an EGL configuration.
func (c *Context) ConfigAttrib(config uintptr, attribute int32) (int32, error) {
	var value int32
	if !eglGetConfigAttrib(c.display, config, attribute, &value) {
		return 0, fmt.Errorf("egl: eglGetConfigAttrib failed: %w", c.LastError())
	}
	return value, nil
}

// CreateWindowSurface creates a window surface for the selected configuration.
func (c *Context) CreateWindowSurface(config, window uintptr, attribs []int32) error {
	var p *int32
	if len(attribs) != 0 {
		p = &attribs[0]
	}
	c.surface = eglCreateWindowSurface(c.display, config, window, p)
	if c.surface == 0 {
		return fmt.Errorf("egl: eglCreateWindowSurface failed: %w", c.LastError())
	}
	return nil
}

// QuerySurface returns an attribute of the current surface.
func (c *Context) QuerySurface(attribute int32) (int32, error) {
	var value int32
	if !eglQuerySurface(c.display, c.surface, attribute, &value) {
		return 0, fmt.Errorf("egl: eglQuerySurface failed: %w", c.LastError())
	}
	return value, nil
}

// CreateES3Context creates an OpenGL ES 3 context for the selected configuration.
func (c *Context) CreateES3Context(config uintptr) error {
	attribs := []int32{ContextClientVersion, 3, None}
	c.context = eglCreateContext(c.display, config, 0, &attribs[0])
	if c.context == 0 {
		return fmt.Errorf("egl: eglCreateContext failed: %w", c.LastError())
	}
	return nil
}

// SetSize records the surface size in pixels.
func (c *Context) SetSize(width, height int) { c.width, c.height = width, height }

// Size returns the surface size in pixels.
func (c *Context) Size() (int, int) { return c.width, c.height }

// MakeContextCurrent makes this context current on the calling thread.
func (c *Context) MakeContextCurrent() error {
	if !eglMakeCurrent(c.display, c.surface, c.surface, c.context) {
		return fmt.Errorf("egl: eglMakeCurrent failed: %w", c.LastError())
	}
	return nil
}

// ClearCurrentContext detaches the current context from the calling thread.
func (c *Context) ClearCurrentContext() error {
	if !eglMakeCurrent(c.display, 0, 0, 0) {
		return fmt.Errorf("egl: eglMakeCurrent failed: %w", c.LastError())
	}
	return nil
}

// SwapInterval sets the requested interval between buffer swaps.
func (c *Context) SwapInterval(interval int) error {
	if c.swapInterval == interval {
		return nil
	}
	if !eglSwapInterval(c.display, int32(interval)) {
		return fmt.Errorf("egl: eglSwapInterval failed: %w", c.LastError())
	}
	c.swapInterval = interval
	return nil
}

// SwapBuffers presents the current surface through EGL.
func (c *Context) SwapBuffers() error {
	if !eglSwapBuffers(c.display, c.surface) {
		return fmt.Errorf("egl: eglSwapBuffers failed: %w", c.LastError())
	}
	return nil
}

// Unbind releases the current context before a backend frees scanout buffers.
func (c *Context) Unbind() {
	if c.display != 0 && (c.context != 0 || c.surface != 0) {
		_ = c.ClearCurrentContext()
	}
}

// Close releases the EGL context, surface, and display.
func (c *Context) Close() error {
	if c.display != 0 {
		c.Unbind()
		if c.context != 0 {
			eglDestroyContext(c.display, c.context)
			c.context = 0
		}
		if c.surface != 0 {
			eglDestroySurface(c.display, c.surface)
			c.surface = 0
		}
		eglTerminate(c.display)
		c.display = 0
	}
	return nil
}

type eglError int32

func (e eglError) Error() string { return fmt.Sprintf("EGL error 0x%x", int32(e)) }

// LastError returns the last error reported by EGL.
func (c *Context) LastError() error {
	code := eglGetError()
	if code == Success {
		return errors.New("EGL reported no error")
	}
	return eglError(code)
}
