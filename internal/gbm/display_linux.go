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

// Package gbm renders on a Linux DRM/KMS display through GBM and a vendor EGL
// implementation, for systems with no window system but a GBM-capable DRM device
// (Mali and other Mesa/panfrost-class GPUs).
package gbm

import (
	"fmt"
	"os"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	drmModeConnected = 1

	gbmFormatXRGB8888 = 0x34325258 // fourcc 'X','R','2','4'
	gbmUseScanout     = 1 << 0
	gbmUseRendering   = 1 << 2
)

// These mirror libdrm structures and use native pointer alignment.
type drmModeRes struct {
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
	encoderID      uint32
	encoderType    uint32
	crtcID         uint32
	possibleCrtcs  uint32
	possibleClones uint32
}

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
	SetCrtc        func(fd int32, crtc, fb, x, y uint32, conns *uint32, count int32, mode *drmModeModeInfo) int32
	PageFlip       func(fd int32, crtc, fb, flags uint32, user uintptr) int32
	AddFB2         func(fd int32, w, h, format uint32, handles, pitches, offsets *uint32, bufID *uint32, flags uint32) int32
	AddFB2WithMods func(fd int32, w, h, format uint32, handles, pitches, offsets *uint32, modifiers *uint64, bufID *uint32, flags uint32) int32
	RmFB           func(fd int32, id uint32) int32
	FreeResources  func(p *drmModeRes)
	FreeConnector  func(p *drmModeConnector)
	FreeEncoder    func(p *drmModeEncoder)
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
	drml drmLib
	gbml gbmLib
)

func regFunc(fptr any, lib uintptr, name string) error {
	sym, err := purego.Dlsym(lib, name)
	if err != nil || sym == 0 {
		return fmt.Errorf("gbm: symbol %s not found: %w", name, err)
	}
	purego.RegisterFunc(fptr, sym)
	return nil
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
	if drml.GetResources != nil {
		return nil
	}
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
		{&drml.SetCrtc, libdrm, "drmModeSetCrtc"},
		{&drml.PageFlip, libdrm, "drmModePageFlip"},
		{&drml.AddFB2, libdrm, "drmModeAddFB2"},
		{&drml.AddFB2WithMods, libdrm, "drmModeAddFB2WithModifiers"},
		{&drml.RmFB, libdrm, "drmModeRmFB"},
		{&drml.FreeResources, libdrm, "drmModeFreeResources"},
		{&drml.FreeConnector, libdrm, "drmModeFreeConnector"},
		{&drml.FreeEncoder, libdrm, "drmModeFreeEncoder"},
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
		{&gbml.BoGetModifier, libgbm, "gbm_bo_get_modifier"},
	} {
		if err := regFunc(f.p, f.lib, f.name); err != nil {
			return err
		}
	}
	return nil
}

type Display struct {
	file    *os.File
	fd      int32
	gbmDev  uintptr
	connID  uint32
	crtcID  uint32
	mode    drmModeModeInfo
	width   int
	height  int
	refresh int
}

func crtcForEncoder(enc *drmModeEncoder, crtcs []uint32) uint32 {
	if enc.crtcID != 0 {
		return enc.crtcID
	}
	for i, crtc := range crtcs {
		if i >= 32 {
			break
		}
		if enc.possibleCrtcs&(uint32(1)<<i) != 0 {
			return crtc
		}
	}
	return 0
}

func findConnectorCRTC(fd int32, conn *drmModeConnector, crtcs []uint32) uint32 {
	// Prefer the active route, as some drivers expose incomplete masks.
	if conn.encoderID != 0 {
		if enc := drml.GetEncoder(fd, conn.encoderID); enc != nil {
			crtc := crtcForEncoder(enc, crtcs)
			drml.FreeEncoder(enc)
			if crtc != 0 {
				return crtc
			}
		}
	}

	for _, encoderID := range uint32Slice(conn.encoders, conn.countEncoders) {
		if encoderID == conn.encoderID {
			continue
		}
		enc := drml.GetEncoder(fd, encoderID)
		if enc == nil {
			continue
		}
		crtc := crtcForEncoder(enc, crtcs)
		drml.FreeEncoder(enc)
		if crtc != 0 {
			return crtc
		}
	}
	return 0
}

func OpenDisplay() (*Display, error) {
	if err := loadDRMGBM(); err != nil {
		return nil, err
	}

	node := "/dev/dri/card0"
	if v := os.Getenv("EBITENGINE_DRM_DEVICE"); v != "" {
		node = v
	}
	f, err := os.OpenFile(node, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("gbm: failed to open %s: %w", node, err)
	}
	fd := int32(f.Fd())

	if r := drml.SetMaster(fd); r != 0 {
		_ = f.Close()
		return nil, fmt.Errorf("gbm: drmSetMaster failed: %d", r)
	}
	closeFile := func() {
		drml.DropMaster(fd)
		_ = f.Close()
	}

	res := drml.GetResources(fd)
	if res == nil {
		closeFile()
		return nil, fmt.Errorf("gbm: drmModeGetResources failed (no KMS, or another client holds the device)")
	}
	defer drml.FreeResources(res)

	connectors := uint32Slice(res.connectors, res.countConnectors)
	crtcs := uint32Slice(res.crtcs, res.countCrtcs)

	d := &Display{file: f, fd: fd}
	found := false
	for _, cid := range connectors {
		conn := drml.GetConnector(fd, cid)
		if conn == nil {
			continue
		}
		modes := modeSlice(conn.modes, conn.countModes)
		if conn.connection == drmModeConnected && len(modes) > 0 {
			// Copy the mode out before the connector is freed.
			d.mode = modes[0]
			d.width = int(d.mode.hdisplay)
			d.height = int(d.mode.vdisplay)
			d.refresh = int(d.mode.vrefresh)
			d.connID = cid
			d.crtcID = findConnectorCRTC(fd, conn, crtcs)
			found = d.crtcID != 0
			drml.FreeConnector(conn)
			if found {
				break
			}
			continue
		}
		drml.FreeConnector(conn)
	}
	if !found {
		closeFile()
		return nil, fmt.Errorf("gbm: no connected connector with a mode and compatible CRTC")
	}

	d.gbmDev = gbml.CreateDevice(fd)
	if d.gbmDev == 0 {
		closeFile()
		return nil, fmt.Errorf("gbm: gbm_create_device failed")
	}
	return d, nil
}

func (d *Display) Size() (width, height int) { return d.width, d.height }

func (d *Display) RefreshRate() int { return d.refresh }

func (d *Display) Close() error {
	if d.gbmDev != 0 {
		gbml.DestroyDevice(d.gbmDev)
		d.gbmDev = 0
	}
	if d.file != nil {
		drml.DropMaster(d.fd)
		err := d.file.Close()
		d.file = nil
		return err
	}
	return nil
}
