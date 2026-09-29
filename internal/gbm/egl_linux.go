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

//go:build (amd64 || arm64) && !android

package gbm

import (
	"errors"
	"fmt"

	"golang.org/x/sys/unix"

	"github.com/hajimehoshi/ebiten/v2/internal/egl"
)

const (
	_EGL_PLATFORM_GBM = 0x31d7

	_DRM_MODE_PAGE_FLIP_EVENT = 0x01
	_DRM_MODE_PAGE_FLIP_ASYNC = 0x02
	_DRM_MODE_FB_MODIFIERS    = 0x02
	_DRM_FORMAT_MOD_INVALID   = 0x00ffffffffffffff
)

// Context presents an EGL frame through GBM and KMS.
type Context struct {
	eglContext *egl.Context
	d          *Display
	gbmSurface uintptr

	swapInterval             int
	modesetDone              bool
	asyncPageFlipUnsupported bool
	prevBo                   uintptr
	prevFB                   uint32
	pendingBo                uintptr
	pendingFB                uint32
	eventBuf                 [64]byte
}

// NewContext creates an OpenGL ES 3 context for the GBM display.
func NewContext(d *Display) (*Context, error) {
	e, err := egl.NewContext()
	if err != nil {
		return nil, err
	}
	c := &Context{eglContext: e, d: d, swapInterval: -1}
	fail := func(err error) (*Context, error) {
		return nil, errors.Join(fmt.Errorf("gbm: %w", err), c.Close())
	}
	var display uintptr
	var getPlatformDisplay func(platform uint32, nativeDisplay uintptr, attribList *int) uintptr
	if err := e.RegisterFunc(&getPlatformDisplay, "eglGetPlatformDisplay"); err == nil {
		display = getPlatformDisplay(_EGL_PLATFORM_GBM, d.gbmDev, nil)
	} else {
		var getPlatformDisplayEXT func(platform uint32, nativeDisplay uintptr, attribList *int32) uintptr
		if extErr := e.RegisterProcFunc(&getPlatformDisplayEXT, "eglGetPlatformDisplayEXT"); extErr != nil {
			return fail(errors.Join(err, extErr))
		}
		display = getPlatformDisplayEXT(_EGL_PLATFORM_GBM, d.gbmDev, nil)
	}
	if err := e.Initialize(display); err != nil {
		return fail(err)
	}
	config, err := c.chooseConfig()
	if err != nil {
		return fail(err)
	}
	c.gbmSurface = gbml.SurfaceCreate(d.gbmDev, uint32(d.width), uint32(d.height), gbmFormatXRGB8888, gbmUseScanout|gbmUseRendering)
	if c.gbmSurface == 0 {
		return fail(errors.New("gbm_surface_create failed"))
	}
	if err := e.CreateWindowSurface(config, c.gbmSurface, nil); err != nil {
		return fail(err)
	}
	if err := e.CreateES3Context(config); err != nil {
		return fail(err)
	}
	e.SetSize(d.width, d.height)
	return c, nil
}

func (c *Context) chooseConfig() (uintptr, error) {
	attribs := []int32{
		egl.SurfaceType, egl.WindowBit,
		egl.RenderableType, egl.OpenGLES3Bit,
		egl.RedSize, 8, egl.GreenSize, 8, egl.BlueSize, 8, egl.AlphaSize, 0,
		egl.None,
	}
	configs, err := c.eglContext.ChooseConfigs(attribs)
	if err != nil {
		return 0, err
	}
	// Mesa needs the config's native visual ID to match the GBM format.
	for _, config := range configs {
		vis, err := c.eglContext.ConfigAttrib(config, egl.NativeVisualID)
		if err == nil && uint32(vis) == gbmFormatXRGB8888 {
			return config, nil
		}
	}
	return 0, fmt.Errorf("gbm: no EGL config has the XRGB8888 native visual")
}

// SwapInterval records the requested KMS presentation interval.
func (c *Context) SwapInterval(interval int) error {
	c.swapInterval = interval
	return nil
}

// Size returns the size of the surface in pixels.
func (c *Context) Size() (int, int) { return c.eglContext.Size() }

// MakeContextCurrent makes this context current on the calling thread.
func (c *Context) MakeContextCurrent() error { return c.eglContext.MakeContextCurrent() }

// Probe verifies that the selected buffer can be scanned out before the UI
// commits to this backend. The previous CRTC state is restored immediately.
func (c *Context) Probe() error {
	if err := c.MakeContextCurrent(); err != nil {
		return err
	}
	defer c.eglContext.Unbind()
	if err := c.SwapBuffers(); err != nil {
		return err
	}
	if err := c.d.restoreCRTC(); err != nil {
		return err
	}
	c.modesetDone = false
	if r := drml.RmFB(c.d.fd, c.prevFB); r != 0 {
		return fmt.Errorf("gbm: removing probe framebuffer failed: %d", r)
	}
	gbml.ReleaseBuffer(c.gbmSurface, c.prevBo)
	c.prevFB = 0
	c.prevBo = 0
	return nil
}

// SwapBuffers presents the current frame through KMS.
func (c *Context) SwapBuffers() error {
	if err := c.eglContext.SwapBuffers(); err != nil {
		return err
	}
	bo := gbml.LockFront(c.gbmSurface)
	if bo == 0 {
		return fmt.Errorf("gbm: gbm_surface_lock_front_buffer failed")
	}
	fb, err := c.addFB(bo)
	if err != nil {
		gbml.ReleaseBuffer(c.gbmSurface, bo)
		return err
	}
	release := func() {
		drml.RmFB(c.d.fd, fb)
		gbml.ReleaseBuffer(c.gbmSurface, bo)
	}

	if !c.modesetDone {
		if r := drml.SetCrtc(c.d.fd, c.d.crtcID, fb, 0, 0, &c.d.connID, 1, &c.d.mode); r != 0 {
			release()
			return fmt.Errorf("gbm: drmModeSetCrtc failed: %d", r)
		}
		c.modesetDone = true
	} else {
		flags := uint32(_DRM_MODE_PAGE_FLIP_EVENT)
		async := c.swapInterval == 0 && !c.asyncPageFlipUnsupported
		if async {
			flags |= _DRM_MODE_PAGE_FLIP_ASYNC
		}
		r := drml.PageFlip(c.d.fd, c.d.crtcID, fb, flags, 0)
		if r != 0 && async {
			c.asyncPageFlipUnsupported = true
			r = drml.PageFlip(c.d.fd, c.d.crtcID, fb, _DRM_MODE_PAGE_FLIP_EVENT, 0)
		}
		if r != 0 {
			release()
			return fmt.Errorf("gbm: drmModePageFlip failed: %d", r)
		}
		if _, err := unix.Read(int(c.d.fd), c.eventBuf[:]); err != nil {
			c.pendingBo = bo
			c.pendingFB = fb
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
	var mod uint64
	if gbml.BoGetModifier != nil {
		mod = gbml.BoGetModifier(bo)
	}
	handles := [4]uint32{handle}
	pitches := [4]uint32{stride}
	offsets := [4]uint32{0}
	var fb uint32
	var r int32
	if drml.AddFB2WithMods != nil && gbml.BoGetModifier != nil && mod != 0 && mod != _DRM_FORMAT_MOD_INVALID {
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

// Close restores the previous display state and releases EGL and GBM resources.
func (c *Context) Close() error {
	if c.eglContext == nil {
		return nil
	}
	c.eglContext.Unbind()
	var err error
	if c.modesetDone {
		err = errors.Join(err, c.d.restoreCRTC())
		c.modesetDone = false
	}
	if c.prevBo != 0 {
		if r := drml.RmFB(c.d.fd, c.prevFB); r != 0 {
			err = errors.Join(err, fmt.Errorf("gbm: removing framebuffer failed: %d", r))
		}
		gbml.ReleaseBuffer(c.gbmSurface, c.prevBo)
		c.prevBo = 0
		c.prevFB = 0
	}
	if c.pendingBo != 0 {
		if r := drml.RmFB(c.d.fd, c.pendingFB); r != 0 {
			err = errors.Join(err, fmt.Errorf("gbm: removing pending framebuffer failed: %d", r))
		}
		gbml.ReleaseBuffer(c.gbmSurface, c.pendingBo)
		c.pendingBo = 0
		c.pendingFB = 0
	}
	err = errors.Join(err, c.eglContext.Close())
	c.eglContext = nil
	if c.gbmSurface != 0 {
		gbml.SurfaceDestroy(c.gbmSurface)
		c.gbmSurface = 0
	}
	return err
}
