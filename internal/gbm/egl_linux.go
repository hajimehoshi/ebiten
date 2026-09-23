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
	*egl.Context
	d          *Display
	gbmSurface uintptr

	swapInterval             int
	modesetDone              bool
	asyncPageFlipUnsupported bool
	prevBo                   uintptr
	prevFB                   uint32
	eventBuf                 [64]byte
}

func NewContext(d *Display) (*Context, error) {
	e, err := egl.NewContext()
	if err != nil {
		return nil, err
	}
	c := &Context{Context: e, d: d, swapInterval: -1}
	fail := func(err error) (*Context, error) {
		return nil, errors.Join(fmt.Errorf("gbm: %w", err), c.Close())
	}
	var getPlatformDisplay func(platform uint32, nativeDisplay uintptr, attribList *int) uintptr
	if err := e.RegisterFunc(&getPlatformDisplay, "eglGetPlatformDisplay"); err != nil {
		return fail(err)
	}
	if err := e.Initialize(getPlatformDisplay(_EGL_PLATFORM_GBM, d.gbmDev, nil)); err != nil {
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
	configs, err := c.Context.ChooseConfigs(attribs)
	if err != nil {
		return 0, err
	}
	// Mesa needs the config's native visual ID to match the GBM format.
	for _, config := range configs {
		vis, err := c.Context.ConfigAttrib(config, egl.NativeVisualID)
		if err == nil && uint32(vis) == gbmFormatXRGB8888 {
			return config, nil
		}
	}
	return configs[0], nil
}

func (c *Context) SwapInterval(interval int) error {
	if err := c.Context.SwapInterval(interval); err != nil {
		return err
	}
	c.swapInterval = interval
	return nil
}

func (c *Context) SwapBuffers() error {
	if err := c.Context.SwapBuffers(); err != nil {
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
	c.Context.Unbind()
	if c.prevBo != 0 {
		drml.RmFB(c.d.fd, c.prevFB)
		gbml.ReleaseBuffer(c.gbmSurface, c.prevBo)
		c.prevBo = 0
	}
	err := c.Context.Close()
	if c.gbmSurface != 0 {
		gbml.SurfaceDestroy(c.gbmSurface)
		c.gbmSurface = 0
	}
	return err
}
