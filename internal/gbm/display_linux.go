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

// Package gbm renders on a Linux DRM/KMS display through GBM and a vendor EGL
// implementation, for systems with no window system but a GBM-capable DRM device
// (Mali and other Mesa/panfrost-class GPUs).
package gbm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"structs"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	drmModeConnected     = 1
	drmModeTypePreferred = 1 << 3

	gbmFormatXRGB8888 = 0x34325258 // fourcc 'X','R','2','4'
	gbmUseScanout     = 1 << 0
	gbmUseRendering   = 1 << 2
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

type Display struct {
	file     *os.File
	fd       int32
	gbmDev   uintptr
	connID   uint32
	crtcID   uint32
	mode     drmModeModeInfo
	oldCRTC  drmModeCrtc
	oldConns []uint32
	width    int
	height   int
	refresh  int
}

func preferredMode(modes []drmModeModeInfo) drmModeModeInfo {
	for _, mode := range modes {
		if mode.typ&drmModeTypePreferred != 0 {
			return mode
		}
	}
	return modes[0]
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

	nodes := []string{os.Getenv("EBITENGINE_DRM_DEVICE")}
	if nodes[0] == "" {
		var err error
		nodes, err = filepath.Glob("/dev/dri/card[0-9]*")
		if err != nil {
			return nil, fmt.Errorf("gbm: listing DRM devices: %w", err)
		}
	}
	if len(nodes) == 0 {
		return nil, errors.New("gbm: no DRM card devices found")
	}
	var errs []error
	for _, node := range nodes {
		d, err := openDisplayNode(node)
		if err == nil {
			return d, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", node, err))
	}
	return nil, fmt.Errorf("gbm: no usable DRM device: %w", errors.Join(errs...))
}

func openDisplayNode(node string) (*Display, error) {
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
			d.mode = preferredMode(modes)
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
	old := drml.GetCrtc(fd, d.crtcID)
	if old == nil {
		closeFile()
		return nil, fmt.Errorf("gbm: drmModeGetCrtc failed")
	}
	d.oldCRTC = *old
	drml.FreeCrtc(old)
	if d.oldCRTC.modeValid != 0 {
		for _, cid := range connectors {
			conn := drml.GetConnector(fd, cid)
			if conn == nil {
				continue
			}
			encoderID := conn.encoderID
			drml.FreeConnector(conn)
			if encoderID == 0 {
				continue
			}
			enc := drml.GetEncoder(fd, encoderID)
			if enc == nil {
				continue
			}
			if enc.crtcID == d.crtcID {
				d.oldConns = append(d.oldConns, cid)
			}
			drml.FreeEncoder(enc)
		}
	}

	d.gbmDev = gbml.CreateDevice(fd)
	if d.gbmDev == 0 {
		closeFile()
		return nil, fmt.Errorf("gbm: gbm_create_device failed")
	}
	return d, nil
}

func (d *Display) restoreCRTC() error {
	var conns *uint32
	var mode *drmModeModeInfo
	if d.oldCRTC.modeValid != 0 {
		mode = &d.oldCRTC.mode
		if len(d.oldConns) != 0 {
			conns = &d.oldConns[0]
		}
	}
	if r := drml.SetCrtc(d.fd, d.crtcID, d.oldCRTC.bufferID, d.oldCRTC.x, d.oldCRTC.y, conns, int32(len(d.oldConns)), mode); r != 0 {
		return fmt.Errorf("gbm: restoring CRTC failed: %d", r)
	}
	return nil
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
