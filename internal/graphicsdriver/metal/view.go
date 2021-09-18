// Copyright 2019 The Ebiten Authors
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

//go:build darwin
// +build darwin

package metal

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

type view struct {
	window uintptr
	uiview uintptr

	windowChanged bool
	vsyncDisabled bool
	fullscreen    bool

	device mtl.Device
	ml     ca.MetalLayer

	once sync.Once
}

func (v *view) setDrawableSize(width, height int) {
	v.ml.SetDrawableSize(width, height)
}

func (v *view) getMTLDevice() mtl.Device {
	return v.device
}

func (v *view) setDisplaySyncEnabled(enabled bool) {
	if !v.vsyncDisabled == enabled {
		return
	}
	v.forceSetDisplaySyncEnabled(enabled)
}

func (v *view) forceSetDisplaySyncEnabled(enabled bool) {
	v.ml.SetDisplaySyncEnabled(enabled)
	v.vsyncDisabled = !enabled

	// setting presentsWithTransaction true makes the FPS stable (#1196). We're not sure why...
	v.updatePresentsWithTransaction()
}

func (v *view) setFullscreen(fullscreen bool) {
	if v.fullscreen == fullscreen {
		return
	}
	v.fullscreen = fullscreen
	v.updatePresentsWithTransaction()
}

func (v *view) updatePresentsWithTransaction() {
	v.ml.SetPresentsWithTransaction(v.usePresentsWithTransaction())
	v.ml.SetMaximumDrawableCount(v.maximumDrawableCount())
}

func (v *view) colorPixelFormat() mtl.PixelFormat {
	return v.ml.PixelFormat()
}

func (v *view) reset() error {
	var err error
	v.device, err = mtl.CreateSystemDefaultDevice()
	if err != nil {
		return err
	}

	v.ml = ca.MakeMetalLayer()
	v.ml.SetDevice(v.device)
	// https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat
	//
	// The pixel format for a Metal layer must be MTLPixelFormatBGRA8Unorm,
	// MTLPixelFormatBGRA8Unorm_sRGB, MTLPixelFormatRGBA16Float, MTLPixelFormatBGRA10_XR, or
	// MTLPixelFormatBGRA10_XR_sRGB.
	v.ml.SetPixelFormat(mtl.PixelFormatBGRA8UNorm)

	// The vsync state might be reset. Set the state again (#1364).
	v.forceSetDisplaySyncEnabled(!v.vsyncDisabled)
	v.ml.SetFramebufferOnly(true)

	return nil
}

func (v *view) nextDrawable() ca.MetalDrawable {
	d, err := v.ml.NextDrawable()
	if err != nil {
		// Drawable is nil. This can happen at the initial state. Let's wait and see.
		return ca.MetalDrawable{}
	}
	return d
}

func (v *view) presentsWithTransaction() bool {
	return v.ml.PresentsWithTransaction()
}
