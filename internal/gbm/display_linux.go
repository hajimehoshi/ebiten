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

func u16At(p unsafe.Pointer, o int) uint16 { return *(*uint16)(unsafe.Add(p, o)) }
func u32At(p unsafe.Pointer, o int) uint32 { return *(*uint32)(unsafe.Add(p, o)) }
func i32At(p unsafe.Pointer, o int) int32  { return *(*int32)(unsafe.Add(p, o)) }
func ptrAt(p unsafe.Pointer, o int) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Add(p, o))
}
func u32Index(p unsafe.Pointer, i int) uint32 {
	return *(*uint32)(unsafe.Add(p, i*4))
}

type drmLib struct {
	GetResources   func(fd int32) unsafe.Pointer
	GetConnector   func(fd int32, id uint32) unsafe.Pointer
	GetEncoder     func(fd int32, id uint32) unsafe.Pointer
	SetCrtc        func(fd int32, crtc, fb, x, y uint32, conns *uint32, count int32, mode unsafe.Pointer) int32
	PageFlip       func(fd int32, crtc, fb, flags uint32, user uintptr) int32
	AddFB2         func(fd int32, w, h, format uint32, handles, pitches, offsets *uint32, bufID *uint32, flags uint32) int32
	AddFB2WithMods func(fd int32, w, h, format uint32, handles, pitches, offsets *uint32, modifiers *uint64, bufID *uint32, flags uint32) int32
	RmFB           func(fd int32, id uint32) int32
	FreeResources  func(p unsafe.Pointer)
	FreeConnector  func(p unsafe.Pointer)
	FreeEncoder    func(p unsafe.Pointer)
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
	mode    [68]byte
	width   int
	height  int
	refresh int
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

	drml.SetMaster(fd)

	res := drml.GetResources(fd)
	if res == nil {
		_ = f.Close()
		return nil, fmt.Errorf("gbm: drmModeGetResources failed (no KMS, or another client holds the device)")
	}
	defer drml.FreeResources(res)

	countConns := int(i32At(res, 32))
	connsPtr := ptrAt(res, 40)
	countCrtcs := int(i32At(res, 16))
	crtcsPtr := ptrAt(res, 24)

	d := &Display{file: f, fd: fd}
	found := false
	for i := 0; i < countConns; i++ {
		cid := u32Index(connsPtr, i)
		conn := drml.GetConnector(fd, cid)
		if conn == nil {
			continue
		}
		if i32At(conn, 16) == drmModeConnected && int(i32At(conn, 32)) > 0 {
			modes := ptrAt(conn, 40)
			// Copy the mode out before the connector is freed.
			copy(d.mode[:], unsafe.Slice((*byte)(modes), 68))
			d.width = int(u16At(modes, 4))    // hdisplay
			d.height = int(u16At(modes, 14))  // vdisplay
			d.refresh = int(u32At(modes, 48)) // vrefresh
			d.connID = cid
			if encID := u32At(conn, 4); encID != 0 {
				if enc := drml.GetEncoder(fd, encID); enc != nil {
					d.crtcID = u32At(enc, 8)
					drml.FreeEncoder(enc)
				}
			}
			found = true
			drml.FreeConnector(conn)
			break
		}
		drml.FreeConnector(conn)
	}
	if !found {
		_ = f.Close()
		return nil, fmt.Errorf("gbm: no connected connector with a mode")
	}
	if d.crtcID == 0 && countCrtcs > 0 {
		d.crtcID = u32Index(crtcsPtr, 0)
	}
	if d.crtcID == 0 {
		_ = f.Close()
		return nil, fmt.Errorf("gbm: no CRTC available")
	}

	d.gbmDev = gbml.CreateDevice(fd)
	if d.gbmDev == 0 {
		_ = f.Close()
		return nil, fmt.Errorf("gbm: gbm_create_device failed")
	}
	return d, nil
}

func (d *Display) Size() (width, height int) { return d.width, d.height }

func (d *Display) RefreshRate() int { return d.refresh }

func (d *Display) modePointer() unsafe.Pointer { return unsafe.Pointer(&d.mode[0]) }

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
