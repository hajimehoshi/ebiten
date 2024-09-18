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

package metal

import (
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

type view struct {
	window uintptr
	uiview uintptr

	windowChanged bool
	vsyncDisabled bool

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
}

func (v *view) colorPixelFormat() mtl.PixelFormat {
	return v.ml.PixelFormat()
}

func (v *view) initialize(device mtl.Device, colorSpace graphicsdriver.ColorSpace) error {
	v.device = device

	ml, err := ca.NewMetalLayer(colorSpace)
	if err != nil {
		return err
	}
	v.ml = ml
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

	// presentsWithTransaction doesn't work in the fullscreen mode (#1745, #1974).
	// presentsWithTransaction doesn't work with vsync off (#1196).
	// nextDrawable took more than one second if the window has other controls like NSTextView (#1029).
	v.ml.SetPresentsWithTransaction(false)

	v.ml.SetMaximumDrawableCount(v.maximumDrawableCount())

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
