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

	"golang.org/x/sys/unix"

	"github.com/hajimehoshi/ebiten/v2/internal/egl"
)

// nativeWindow is the native window type the framebuffer drivers that want one
// expect a pointer to.
type nativeWindow struct {
	_      structs.HostLayout
	width  uint16
	height uint16
}

// Context is an EGL context presenting to a framebuffer device.
type Context struct {
	*egl.Context

	// Drivers can retain this pointer until the surface is destroyed.
	window []byte
}

// NewContext creates an OpenGL ES 3 context covering the display.
func NewContext(d *Display) (*Context, error) {
	e, err := egl.NewContext()
	if err != nil {
		return nil, err
	}
	c := &Context{Context: e}
	fail := func(err error) (*Context, error) {
		return nil, errors.Join(fmt.Errorf("fbdev: %w", err), c.Close())
	}

	var getDisplay func(displayID uintptr) uintptr
	if err := e.RegisterFunc(&getDisplay, "eglGetDisplay"); err != nil {
		return fail(err)
	}
	// EGL_DEFAULT_DISPLAY: there is no display server to name.
	if err := e.Initialize(getDisplay(0)); err != nil {
		return fail(err)
	}

	red, green, blue := d.BitsPerColor()
	attribs := []int32{
		egl.SurfaceType, egl.WindowBit,
		egl.RenderableType, egl.OpenGLES3Bit,
		egl.RedSize, int32(red),
		egl.GreenSize, int32(green),
		egl.BlueSize, int32(blue),
		egl.None,
	}
	config, err := e.ChooseConfig(attribs)
	if err != nil {
		return fail(err)
	}

	if err := c.createWindow(d); err != nil {
		return fail(err)
	}
	// Some drivers dereference a native window; others require a null window.
	// Try the pointer first so a driver that needs one does not crash.
	var surfaceErr error
	for _, win := range []uintptr{c.nativeWindowPointer(), 0} {
		if surfaceErr = e.CreateWindowSurface(config, win, []int32{egl.None}); surfaceErr == nil {
			break
		}
	}
	if surfaceErr != nil {
		return fail(surfaceErr)
	}

	width, err := e.QuerySurface(egl.Width)
	if err != nil {
		return fail(err)
	}
	height, err := e.QuerySurface(egl.Height)
	if err != nil {
		return fail(err)
	}
	if width <= 0 || height <= 0 {
		return fail(fmt.Errorf("the EGL surface reported an empty size %dx%d", width, height))
	}
	e.SetSize(int(width), int(height))
	if err := e.CreateES3Context(config); err != nil {
		return fail(err)
	}
	return c, nil
}

func (c *Context) createWindow(d *Display) error {
	width, height := d.Size()
	if width > math.MaxUint16 || height > math.MaxUint16 {
		return fmt.Errorf("a display of %dx%d does not fit in a native window", width, height)
	}
	buf, err := unix.Mmap(-1, 0, int(unsafe.Sizeof(nativeWindow{})), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		return fmt.Errorf("failed to allocate a native window: %w", err)
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

// Close releases EGL before freeing the memory a driver may have retained.
func (c *Context) Close() error {
	err := c.Context.Close()
	if c.window != nil {
		err = errors.Join(err, unix.Munmap(c.window))
		c.window = nil
	}
	return err
}
