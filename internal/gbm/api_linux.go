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

//go:build amd64 || arm64

package gbm

import (
	"fmt"
	"structs"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	_EGL_PLATFORM_GBM_KHR = 0x31d7

	_DRM_MODE_CONNECTED       = 1
	_DRM_MODE_TYPE_PREFERRED  = 1 << 3
	_DRM_MODE_PAGE_FLIP_EVENT = 0x01
	_DRM_MODE_PAGE_FLIP_ASYNC = 0x02
	_DRM_MODE_FB_MODIFIERS    = 0x02
	_DRM_FORMAT_MOD_INVALID   = 0x00ffffffffffffff

	_GBM_FORMAT_XRGB8888  = 0x34325258 // fourcc 'X','R','2','4'
	_GBM_BO_USE_SCANOUT   = 1 << 0
	_GBM_BO_USE_RENDERING = 1 << 2
)

// These mirror libdrm structures and use native pointer alignment.
type drmModeRes struct {
	_               structs.HostLayout
	countFBs        int32
	fbs             *uint32
	countCrtcs      int32
	crtcs           *uint32
	countConnectors int32
	connectors      *uint32
	countEncoders   int32
	encoders        *uint32
	minWidth        uint32
	maxWidth        uint32
	minHeight       uint32
	maxHeight       uint32
}

type drmModeModeInfo struct {
	_                                      structs.HostLayout
	clock                                  uint32
	hdisplay, hsyncStart, hsyncEnd, htotal uint16
	hskew                                  uint16
	vdisplay, vsyncStart, vsyncEnd, vtotal uint16
	vscan                                  uint16
	vrefresh                               uint32
	flags                                  uint32
	typ                                    uint32
	name                                   [32]byte
}

type drmModeConnector struct {
	_               structs.HostLayout
	connectorID     uint32
	encoderID       uint32
	connectorType   uint32
	connectorTypeID uint32
	connection      uint32
	mmWidth         uint32
	mmHeight        uint32
	subpixel        uint32
	countModes      int32
	modes           *drmModeModeInfo
	countProps      int32
	props           *uint32
	propValues      *uint64
	countEncoders   int32
	encoders        *uint32
}

type drmModeEncoder struct {
	_              structs.HostLayout
	encoderID      uint32
	encoderType    uint32
	crtcID         uint32
	possibleCrtcs  uint32
	possibleClones uint32
}

type drmModeCrtc struct {
	_         structs.HostLayout
	crtcID    uint32
	bufferID  uint32
	x, y      uint32
	width     uint32
	height    uint32
	modeValid int32
	mode      drmModeModeInfo
	gammaSize int32
}

const (
	drmPointerSize                  = unsafe.Sizeof(uintptr(0))
	drmModeResSize                  = unsafe.Sizeof(drmModeRes{})
	drmModeResCountConnectorsOffset = unsafe.Offsetof(drmModeRes{}.countConnectors)
	drmModeResConnectorsOffset      = unsafe.Offsetof(drmModeRes{}.connectors)
	drmModeConnectorSize            = unsafe.Sizeof(drmModeConnector{})
	drmModeConnectorModesOffset     = unsafe.Offsetof(drmModeConnector{}.modes)
	drmModeModeInfoSize             = unsafe.Sizeof(drmModeModeInfo{})
	drmModeModeInfoVRefreshOffset   = unsafe.Offsetof(drmModeModeInfo{}.vrefresh)
	drmModeCrtcSize                 = unsafe.Sizeof(drmModeCrtc{})

	wantDRMModeResSize                  = 8*drmPointerSize + 16
	wantDRMModeResCountConnectorsOffset = 4 * drmPointerSize
	wantDRMModeResConnectorsOffset      = 5 * drmPointerSize
	wantDRMModeConnectorSize            = 7*drmPointerSize + 32
	wantDRMModeConnectorModesOffset     = drmPointerSize + 32
)

var (
	_ [0]byte = [wantDRMModeResSize - drmModeResSize]byte{}
	_ [0]byte = [wantDRMModeResCountConnectorsOffset - drmModeResCountConnectorsOffset]byte{}
	_ [0]byte = [wantDRMModeResConnectorsOffset - drmModeResConnectorsOffset]byte{}
	_ [0]byte = [wantDRMModeConnectorSize - drmModeConnectorSize]byte{}
	_ [0]byte = [wantDRMModeConnectorModesOffset - drmModeConnectorModesOffset]byte{}
	_ [0]byte = [68 - drmModeModeInfoSize]byte{}
	_ [0]byte = [24 - drmModeModeInfoVRefreshOffset]byte{}
	_ [0]byte = [100 - drmModeCrtcSize]byte{}
)

func uint32Slice(p *uint32, n int32) []uint32 {
	if p == nil || n <= 0 {
		return nil
	}
	return unsafe.Slice(p, int(n))
}

func modeSlice(p *drmModeModeInfo, n int32) []drmModeModeInfo {
	if p == nil || n <= 0 {
		return nil
	}
	return unsafe.Slice(p, int(n))
}

type drmLib struct {
	GetResources   func(fd int32) *drmModeRes
	GetConnector   func(fd int32, id uint32) *drmModeConnector
	GetEncoder     func(fd int32, id uint32) *drmModeEncoder
	GetCrtc        func(fd int32, id uint32) *drmModeCrtc
	SetCrtc        func(fd int32, crtc, fb, x, y uint32, conns *uint32, count int32, mode *drmModeModeInfo) int32
	PageFlip       func(fd int32, crtc, fb, flags uint32, user uintptr) int32
	AddFB2         func(fd int32, w, h, format uint32, handles, pitches, offsets *uint32, bufID *uint32, flags uint32) int32
	AddFB2WithMods func(fd int32, w, h, format uint32, handles, pitches, offsets *uint32, modifiers *uint64, bufID *uint32, flags uint32) int32
	RmFB           func(fd int32, id uint32) int32
	FreeResources  func(p *drmModeRes)
	FreeConnector  func(p *drmModeConnector)
	FreeEncoder    func(p *drmModeEncoder)
	FreeCrtc       func(p *drmModeCrtc)
	SetMaster      func(fd int32) int32
	DropMaster     func(fd int32) int32
}

type gbmLib struct {
	CreateDevice   func(fd int32) uintptr
	DestroyDevice  func(dev uintptr)
	SurfaceCreate  func(dev uintptr, w, h, format, flags uint32) uintptr
	SurfaceDestroy func(surf uintptr)
	LockFront      func(surf uintptr) uintptr
	ReleaseBuffer  func(surf, bo uintptr)
	BoGetStride    func(bo uintptr) uint32
	BoGetHandle    func(bo uintptr) uint64
	BoGetModifier  func(bo uintptr) uint64
}

var (
	drml     drmLib
	gbml     gbmLib
	loadOnce sync.Once
	loadErr  error
)

func regFunc(fptr any, lib uintptr, name string) error {
	sym, err := purego.Dlsym(lib, name)
	if err != nil || sym == 0 {
		return fmt.Errorf("gbm: symbol %s not found: %w", name, err)
	}
	purego.RegisterFunc(fptr, sym)
	return nil
}

func regOptionalFunc(fptr any, lib uintptr, name string) {
	sym, err := purego.Dlsym(lib, name)
	if err == nil && sym != 0 {
		purego.RegisterFunc(fptr, sym)
	}
}

func dlopenAny(names ...string) (uintptr, error) {
	var errs error
	for _, n := range names {
		h, err := purego.Dlopen(n, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			return h, nil
		}
		errs = err
	}
	return 0, fmt.Errorf("gbm: failed to load %v: %w", names, errs)
}

func loadDRMGBM() error {
	loadOnce.Do(func() {
		loadErr = loadDRMGBMImpl()
	})
	return loadErr
}

func loadDRMGBMImpl() error {
	libdrm, err := dlopenAny("libdrm.so.2", "libdrm.so")
	if err != nil {
		return err
	}
	libgbm, err := dlopenAny("libgbm.so.1", "libgbm.so")
	if err != nil {
		return err
	}
	for _, f := range []struct {
		p    any
		lib  uintptr
		name string
	}{
		{&drml.GetResources, libdrm, "drmModeGetResources"},
		{&drml.GetConnector, libdrm, "drmModeGetConnector"},
		{&drml.GetEncoder, libdrm, "drmModeGetEncoder"},
		{&drml.GetCrtc, libdrm, "drmModeGetCrtc"},
		{&drml.SetCrtc, libdrm, "drmModeSetCrtc"},
		{&drml.PageFlip, libdrm, "drmModePageFlip"},
		{&drml.AddFB2, libdrm, "drmModeAddFB2"},
		{&drml.RmFB, libdrm, "drmModeRmFB"},
		{&drml.FreeResources, libdrm, "drmModeFreeResources"},
		{&drml.FreeConnector, libdrm, "drmModeFreeConnector"},
		{&drml.FreeEncoder, libdrm, "drmModeFreeEncoder"},
		{&drml.FreeCrtc, libdrm, "drmModeFreeCrtc"},
		{&drml.SetMaster, libdrm, "drmSetMaster"},
		{&drml.DropMaster, libdrm, "drmDropMaster"},
		{&gbml.CreateDevice, libgbm, "gbm_create_device"},
		{&gbml.DestroyDevice, libgbm, "gbm_device_destroy"},
		{&gbml.SurfaceCreate, libgbm, "gbm_surface_create"},
		{&gbml.SurfaceDestroy, libgbm, "gbm_surface_destroy"},
		{&gbml.LockFront, libgbm, "gbm_surface_lock_front_buffer"},
		{&gbml.ReleaseBuffer, libgbm, "gbm_surface_release_buffer"},
		{&gbml.BoGetStride, libgbm, "gbm_bo_get_stride"},
		{&gbml.BoGetHandle, libgbm, "gbm_bo_get_handle"},
	} {
		if err := regFunc(f.p, f.lib, f.name); err != nil {
			return err
		}
	}
	regOptionalFunc(&drml.AddFB2WithMods, libdrm, "drmModeAddFB2WithModifiers")
	regOptionalFunc(&gbml.BoGetModifier, libgbm, "gbm_bo_get_modifier")
	return nil
}
