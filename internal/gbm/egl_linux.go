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

package gbm

import (
	"errors"
	"fmt"

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
	_EGL_ALPHA_SIZE             = 0x3021
	_EGL_NATIVE_VISUAL_ID       = 0x302E
	_EGL_OPENGL_ES_API          = 0x30A0
	_EGL_CONTEXT_CLIENT_VERSION = 0x3098
	_EGL_PLATFORM_GBM           = 0x31D7
	_EGL_SUCCESS                = 0x3000

	_DRM_MODE_PAGE_FLIP_EVENT = 0x01
	_DRM_MODE_FB_MODIFIERS    = 0x02
	_DRM_FORMAT_MOD_INVALID   = 0x00ffffffffffffff
)

type eglAPI struct {
	GetPlatformDisplay  func(platform uint32, nativeDisplay uintptr, attribList *int) uintptr
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
	GetError            func() int32
}

type Context struct {
	d   *Display
	egl eglAPI
	lib uintptr

	display    uintptr
	config     uintptr
	surface    uintptr
	context    uintptr
	gbmSurface uintptr

	width  int
	height int

	swapInterval int
	modesetDone  bool
	prevBo       uintptr
	prevFB       uint32
	eventBuf     [64]byte
}

func NewContext(d *Display) (*Context, error) {
	c := &Context{d: d, swapInterval: -1}
	if err := c.loadEGL(); err != nil {
		return nil, err
	}

	c.display = c.egl.GetPlatformDisplay(_EGL_PLATFORM_GBM, d.gbmDev, nil)
	if c.display == 0 {
		return nil, fmt.Errorf("gbm: eglGetPlatformDisplay failed: %w", c.lastError())
	}
	var major, minor int32
	if !c.egl.Initialize(c.display, &major, &minor) {
		return nil, fmt.Errorf("gbm: eglInitialize failed: %w", c.lastError())
	}
	if !c.egl.BindAPI(_EGL_OPENGL_ES_API) {
		return nil, errors.Join(fmt.Errorf("gbm: eglBindAPI failed: %w", c.lastError()), c.Close())
	}

	config, err := c.chooseConfig()
	if err != nil {
		return nil, errors.Join(err, c.Close())
	}
	c.config = config

	c.gbmSurface = gbml.SurfaceCreate(d.gbmDev, uint32(d.width), uint32(d.height), gbmFormatXRGB8888, gbmUseScanout|gbmUseRendering)
	if c.gbmSurface == 0 {
		return nil, errors.Join(fmt.Errorf("gbm: gbm_surface_create failed"), c.Close())
	}

	c.surface = c.egl.CreateWindowSurface(c.display, config, c.gbmSurface, nil)
	if c.surface == 0 {
		return nil, errors.Join(fmt.Errorf("gbm: eglCreateWindowSurface failed: %w", c.lastError()), c.Close())
	}

	for _, version := range []int32{3, 2} {
		attribs := []int32{_EGL_CONTEXT_CLIENT_VERSION, version, _EGL_NONE}
		c.context = c.egl.CreateContext(c.display, config, 0, &attribs[0])
		if c.context != 0 {
			break
		}
	}
	if c.context == 0 {
		return nil, errors.Join(fmt.Errorf("gbm: eglCreateContext failed: %w", c.lastError()), c.Close())
	}

	c.width = d.width
	c.height = d.height
	return c, nil
}

func (c *Context) loadEGL() error {
	lib, err := dlopenAny("libEGL.so.1", "libEGL.so")
	if err != nil {
		return err
	}
	c.lib = lib
	for _, f := range []struct {
		ptr  any
		name string
	}{
		{&c.egl.GetPlatformDisplay, "eglGetPlatformDisplay"},
		{&c.egl.Initialize, "eglInitialize"},
		{&c.egl.Terminate, "eglTerminate"},
		{&c.egl.BindAPI, "eglBindAPI"},
		{&c.egl.ChooseConfig, "eglChooseConfig"},
		{&c.egl.GetConfigAttrib, "eglGetConfigAttrib"},
		{&c.egl.CreateWindowSurface, "eglCreateWindowSurface"},
		{&c.egl.CreateContext, "eglCreateContext"},
		{&c.egl.DestroySurface, "eglDestroySurface"},
		{&c.egl.DestroyContext, "eglDestroyContext"},
		{&c.egl.MakeCurrent, "eglMakeCurrent"},
		{&c.egl.SwapBuffers, "eglSwapBuffers"},
		{&c.egl.SwapInterval, "eglSwapInterval"},
		{&c.egl.GetError, "eglGetError"},
	} {
		sym, err := purego.Dlsym(lib, f.name)
		if err != nil || sym == 0 {
			return fmt.Errorf("gbm: %s not found in libEGL: %w", f.name, err)
		}
		purego.RegisterFunc(f.ptr, sym)
	}
	return nil
}

func (c *Context) chooseConfig() (uintptr, error) {
	attribs := []int32{
		_EGL_SURFACE_TYPE, _EGL_WINDOW_BIT,
		_EGL_RENDERABLE_TYPE, _EGL_OPENGL_ES2_BIT,
		_EGL_RED_SIZE, 8, _EGL_GREEN_SIZE, 8, _EGL_BLUE_SIZE, 8, _EGL_ALPHA_SIZE, 0,
		_EGL_NONE,
	}
	configs := make([]uintptr, 32)
	var num int32
	if !c.egl.ChooseConfig(c.display, &attribs[0], &configs[0], int32(len(configs)), &num) || num == 0 {
		return 0, fmt.Errorf("gbm: eglChooseConfig found no config: %w", c.lastError())
	}
	// Mesa needs the config's native visual id to be the gbm format.
	for i := 0; i < int(num); i++ {
		var vis int32
		if c.egl.GetConfigAttrib(c.display, configs[i], _EGL_NATIVE_VISUAL_ID, &vis) && uint32(vis) == gbmFormatXRGB8888 {
			return configs[i], nil
		}
	}
	return configs[0], nil
}

func (c *Context) Size() (width, height int) { return c.width, c.height }

func (c *Context) MakeContextCurrent() error {
	if !c.egl.MakeCurrent(c.display, c.surface, c.surface, c.context) {
		return fmt.Errorf("gbm: eglMakeCurrent failed: %w", c.lastError())
	}
	return nil
}

func (c *Context) SwapInterval(interval int) error {
	if c.swapInterval == interval {
		return nil
	}
	if c.egl.SwapInterval != nil {
		c.egl.SwapInterval(c.display, int32(interval))
	}
	c.swapInterval = interval
	return nil
}

func (c *Context) SwapBuffers() error {
	if !c.egl.SwapBuffers(c.display, c.surface) {
		return fmt.Errorf("gbm: eglSwapBuffers failed: %w", c.lastError())
	}
	bo := gbml.LockFront(c.gbmSurface)
	if bo == 0 {
		return fmt.Errorf("gbm: gbm_surface_lock_front_buffer failed")
	}
	fb, err := c.addFB(bo)
	if err != nil {
		return err
	}

	if !c.modesetDone {
		// The first frame sets the mode; later frames page-flip and wait.
		if r := drml.SetCrtc(c.d.fd, c.d.crtcID, fb, 0, 0, &c.d.connID, 1, c.d.modePointer()); r != 0 {
			return fmt.Errorf("gbm: drmModeSetCrtc failed: %d", r)
		}
		c.modesetDone = true
	} else {
		if r := drml.PageFlip(c.d.fd, c.d.crtcID, fb, _DRM_MODE_PAGE_FLIP_EVENT, 0); r != 0 {
			return fmt.Errorf("gbm: drmModePageFlip failed: %d", r)
		}
		if _, err := unix.Read(int(c.d.fd), c.eventBuf[:]); err != nil {
			return fmt.Errorf("gbm: waiting for page flip failed: %w", err)
		}
	}

	if c.prevBo != 0 {
		drml.RmFB(c.d.fd, c.prevFB)
		gbml.ReleaseBuffer(c.gbmSurface, c.prevBo)
	}
	c.prevBo = bo
	c.prevFB = fb
	return nil
}

func (c *Context) addFB(bo uintptr) (uint32, error) {
	handle := uint32(gbml.BoGetHandle(bo))
	stride := gbml.BoGetStride(bo)
	mod := gbml.BoGetModifier(bo)
	handles := [4]uint32{handle}
	pitches := [4]uint32{stride}
	offsets := [4]uint32{0}
	var fb uint32
	var r int32
	if mod != 0 && mod != _DRM_FORMAT_MOD_INVALID {
		mods := [4]uint64{mod}
		r = drml.AddFB2WithMods(c.d.fd, uint32(c.d.width), uint32(c.d.height), gbmFormatXRGB8888, &handles[0], &pitches[0], &offsets[0], &mods[0], &fb, _DRM_MODE_FB_MODIFIERS)
	} else {
		r = drml.AddFB2(c.d.fd, uint32(c.d.width), uint32(c.d.height), gbmFormatXRGB8888, &handles[0], &pitches[0], &offsets[0], &fb, 0)
	}
	if r != 0 {
		return 0, fmt.Errorf("gbm: drmModeAddFB2 failed: %d", r)
	}
	return fb, nil
}

func (c *Context) Close() error {
	if c.display != 0 {
		c.egl.MakeCurrent(c.display, 0, 0, 0)
		if c.prevBo != 0 {
			drml.RmFB(c.d.fd, c.prevFB)
			gbml.ReleaseBuffer(c.gbmSurface, c.prevBo)
			c.prevBo = 0
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
	if c.gbmSurface != 0 {
		gbml.SurfaceDestroy(c.gbmSurface)
		c.gbmSurface = 0
	}
	return nil
}

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
