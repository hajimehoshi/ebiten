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

package fbdev

import (
	"errors"
	"fmt"
	"math"
	"structs"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/unix"
)

const (
	_EGL_NONE                   = 0x3038
	_EGL_SURFACE_TYPE           = 0x3033
	_EGL_WINDOW_BIT             = 0x0004
	_EGL_RENDERABLE_TYPE        = 0x3040
	_EGL_OPENGL_ES2_BIT         = 0x0004
	_EGL_RED_SIZE               = 0x3024
	_EGL_GREEN_SIZE             = 0x3023
	_EGL_BLUE_SIZE              = 0x3022
	_EGL_OPENGL_ES_API          = 0x30a0
	_EGL_CONTEXT_CLIENT_VERSION = 0x3098

	_EGL_HEIGHT = 0x3056
	_EGL_WIDTH  = 0x3057

	_EGL_SUCCESS = 0x3000
)

// nativeWindow is the native window type the framebuffer drivers that want one
// expect a pointer to.
type nativeWindow struct {
	_      structs.HostLayout
	width  uint16
	height uint16
}

type eglAPI struct {
	GetDisplay          func(displayID uintptr) uintptr
	Initialize          func(display uintptr, major, minor *int32) bool
	Terminate           func(display uintptr) bool
	BindAPI             func(api int32) bool
	ChooseConfig        func(display uintptr, attribList *int32, configs *uintptr, configSize int32, numConfig *int32) bool
	CreateWindowSurface func(display uintptr, config uintptr, win uintptr, attribList *int32) uintptr
	CreateContext       func(display uintptr, config uintptr, shareContext uintptr, attribList *int32) uintptr
	DestroySurface      func(display uintptr, surface uintptr) bool
	DestroyContext      func(display uintptr, ctx uintptr) bool
	MakeCurrent         func(display uintptr, draw, read, ctx uintptr) bool
	SwapBuffers         func(display uintptr, surface uintptr) bool
	SwapInterval        func(display uintptr, interval int32) bool
	QuerySurface        func(display uintptr, surface uintptr, attribute int32, value *int32) bool
	GetError            func() int32
}

// Context is an EGL context presenting to a framebuffer device.
type Context struct {
	egl eglAPI

	lib     uintptr
	display uintptr
	surface uintptr
	context uintptr

	// window is the memory a driver that wants a native window receives a
	// pointer to. It is mapped outside the Go heap since such a driver keeps
	// the pointer for as long as the surface exists.
	window []byte

	width  int
	height int

	swapInterval int
}

// NewContext creates an OpenGL ES context covering the display.
func NewContext(d *Display) (*Context, error) {
	c := &Context{
		swapInterval: -1,
	}

	if err := c.loadEGL(); err != nil {
		return nil, err
	}

	// EGL_DEFAULT_DISPLAY: there is no display server to name.
	c.display = c.egl.GetDisplay(0)
	if c.display == 0 {
		return nil, fmt.Errorf("fbdev: eglGetDisplay failed: %w", c.lastError())
	}

	var major, minor int32
	if !c.egl.Initialize(c.display, &major, &minor) {
		return nil, fmt.Errorf("fbdev: eglInitialize failed: %w", c.lastError())
	}

	if !c.egl.BindAPI(_EGL_OPENGL_ES_API) {
		err := fmt.Errorf("fbdev: eglBindAPI failed: %w", c.lastError())
		return nil, errors.Join(err, c.Close())
	}

	config, err := c.chooseConfig(d)
	if err != nil {
		return nil, errors.Join(err, c.Close())
	}

	if err := c.createWindow(d); err != nil {
		return nil, errors.Join(err, c.Close())
	}

	// The native window type is what the drivers here disagree on: some read
	// the dimensions through a pointer, while a null window system has no
	// window to describe and takes the display instead. Ask for a window
	// first, as a driver that wants one dereferences what it is given, and a
	// driver that wants none reports an error rather than crashing.
	attribs := []int32{_EGL_NONE}
	for _, win := range []uintptr{c.nativeWindowPointer(), 0} {
		c.surface = c.egl.CreateWindowSurface(c.display, config, win, &attribs[0])
		if c.surface != 0 {
			break
		}
	}
	if c.surface == 0 {
		err := fmt.Errorf("fbdev: eglCreateWindowSurface failed: %w", c.lastError())
		return nil, errors.Join(err, c.Close())
	}

	// The surface's own size is the one to render at: it is created without a
	// size wherever the driver takes no window.
	var width, height int32
	if !c.egl.QuerySurface(c.display, c.surface, _EGL_WIDTH, &width) || !c.egl.QuerySurface(c.display, c.surface, _EGL_HEIGHT, &height) {
		err := fmt.Errorf("fbdev: eglQuerySurface failed: %w", c.lastError())
		return nil, errors.Join(err, c.Close())
	}
	if width <= 0 || height <= 0 {
		err := fmt.Errorf("fbdev: the EGL surface reported an empty size %dx%d", width, height)
		return nil, errors.Join(err, c.Close())
	}
	c.width = int(width)
	c.height = int(height)

	// OpenGL ES 3 comes first as the graphics driver uses ES 3 features where
	// they are available.
	for _, version := range []int32{3, 2} {
		attribs := []int32{_EGL_CONTEXT_CLIENT_VERSION, version, _EGL_NONE}
		c.context = c.egl.CreateContext(c.display, config, 0, &attribs[0])
		if c.context != 0 {
			break
		}
	}
	if c.context == 0 {
		err := fmt.Errorf("fbdev: eglCreateContext failed: %w", c.lastError())
		return nil, errors.Join(err, c.Close())
	}

	return c, nil
}

func (c *Context) loadEGL() error {
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
		return fmt.Errorf("fbdev: failed to load libEGL: %w", errors.Join(errs...))
	}

	for _, f := range []struct {
		ptr  any
		name string
	}{
		{&c.egl.GetDisplay, "eglGetDisplay"},
		{&c.egl.Initialize, "eglInitialize"},
		{&c.egl.Terminate, "eglTerminate"},
		{&c.egl.BindAPI, "eglBindAPI"},
		{&c.egl.ChooseConfig, "eglChooseConfig"},
		{&c.egl.CreateWindowSurface, "eglCreateWindowSurface"},
		{&c.egl.CreateContext, "eglCreateContext"},
		{&c.egl.DestroySurface, "eglDestroySurface"},
		{&c.egl.DestroyContext, "eglDestroyContext"},
		{&c.egl.MakeCurrent, "eglMakeCurrent"},
		{&c.egl.SwapBuffers, "eglSwapBuffers"},
		{&c.egl.SwapInterval, "eglSwapInterval"},
		{&c.egl.QuerySurface, "eglQuerySurface"},
		{&c.egl.GetError, "eglGetError"},
	} {
		sym, err := purego.Dlsym(c.lib, f.name)
		if err != nil || sym == 0 {
			return fmt.Errorf("fbdev: %s not found in libEGL: %w", f.name, err)
		}
		purego.RegisterFunc(f.ptr, sym)
	}

	return nil
}

// chooseConfig asks the implementation for a config, taking the color depths
// from the display: a framebuffer device is often 16-bit, where a request for
// 8 bits per channel matches nothing.
func (c *Context) chooseConfig(d *Display) (uintptr, error) {
	red, green, blue := d.BitsPerColor()
	attribs := []int32{
		_EGL_SURFACE_TYPE, _EGL_WINDOW_BIT,
		_EGL_RENDERABLE_TYPE, _EGL_OPENGL_ES2_BIT,
		_EGL_RED_SIZE, int32(red),
		_EGL_GREEN_SIZE, int32(green),
		_EGL_BLUE_SIZE, int32(blue),
		_EGL_NONE,
	}

	var config uintptr
	var num int32
	if !c.egl.ChooseConfig(c.display, &attribs[0], &config, 1, &num) {
		return 0, fmt.Errorf("fbdev: eglChooseConfig failed: %w", c.lastError())
	}
	if num == 0 {
		return 0, fmt.Errorf("fbdev: no EGL config matches the display's %d/%d/%d bit color", red, green, blue)
	}
	return config, nil
}

func (c *Context) createWindow(d *Display) error {
	// The native window's fields are 16-bit, so a larger mode cannot be
	// expressed to a driver that reads them.
	width, height := d.Size()
	if width > math.MaxUint16 || height > math.MaxUint16 {
		return fmt.Errorf("fbdev: a display of %dx%d does not fit in a native window", width, height)
	}

	buf, err := unix.Mmap(-1, 0, int(unsafe.Sizeof(nativeWindow{})), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		return fmt.Errorf("fbdev: failed to allocate a native window: %w", err)
	}

	w := (*nativeWindow)(unsafe.Pointer(&buf[0]))
	w.width = uint16(width)
	w.height = uint16(height)

	c.window = buf
	return nil
}

func (c *Context) nativeWindowPointer() uintptr {
	return uintptr(unsafe.Pointer(&c.window[0]))
}

// Size returns the size of the surface in pixels.
func (c *Context) Size() (width, height int) {
	return c.width, c.height
}

// MakeContextCurrent binds the context to the calling thread.
func (c *Context) MakeContextCurrent() error {
	if !c.egl.MakeCurrent(c.display, c.surface, c.surface, c.context) {
		return fmt.Errorf("fbdev: eglMakeCurrent failed: %w", c.lastError())
	}
	return nil
}

// SwapInterval sets how many display refreshes a frame is shown for.
func (c *Context) SwapInterval(interval int) error {
	if c.swapInterval == interval {
		return nil
	}
	if !c.egl.SwapInterval(c.display, int32(interval)) {
		return fmt.Errorf("fbdev: eglSwapInterval failed: %w", c.lastError())
	}
	c.swapInterval = interval
	return nil
}

// SwapBuffers presents the rendered frame.
func (c *Context) SwapBuffers() error {
	if !c.egl.SwapBuffers(c.display, c.surface) {
		return fmt.Errorf("fbdev: eglSwapBuffers failed: %w", c.lastError())
	}
	return nil
}

// Close releases the context and its surface.
func (c *Context) Close() error {
	if c.display != 0 {
		if c.context != 0 || c.surface != 0 {
			c.egl.MakeCurrent(c.display, 0, 0, 0)
		}
		if c.context != 0 {
			c.egl.DestroyContext(c.display, c.context)
			c.context = 0
		}
		if c.surface != 0 {
			c.egl.DestroySurface(c.display, c.surface)
			c.surface = 0
		}
		c.egl.Terminate(c.display)
		c.display = 0
	}

	if c.window != nil {
		if err := unix.Munmap(c.window); err != nil {
			c.window = nil
			return fmt.Errorf("fbdev: failed to release the native window: %w", err)
		}
		c.window = nil
	}

	return nil
}

// eglError is an error reported by the EGL implementation.
type eglError int32

func (e eglError) Error() string {
	return fmt.Sprintf("EGL error 0x%x", int32(e))
}

func (c *Context) lastError() error {
	if c.egl.GetError == nil {
		return nil
	}
	code := c.egl.GetError()
	if code == _EGL_SUCCESS {
		return nil
	}
	return eglError(code)
}
