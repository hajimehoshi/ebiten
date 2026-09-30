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

// Package gbm renders on a Linux DRM/KMS display through GBM and a vendor EGL
// implementation, for systems with no window system but a GBM-capable DRM device
// (Mali and other Mesa/panfrost-class GPUs).
package gbm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

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
		if mode.typ&_DRM_MODE_TYPE_PREFERRED != 0 {
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
		if conn.connection == _DRM_MODE_CONNECTED && len(modes) > 0 {
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
