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

// +build darwin

package metal

import (
	"sync"

	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/ca"
	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/mtl"
)

type view struct {
	window uintptr
	uiview uintptr

	windowChanged bool
	vsync         bool

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
	// TODO: Now SetVsyncEnabled is called only from the main thread, and d.t.Run is not available since
	// recursive function call via Run is forbidden.
	// Fix this to use d.t.Run to avoid confusion.
	v.ml.SetDisplaySyncEnabled(enabled)
	v.vsync = enabled
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
	v.ml.SetMaximumDrawableCount(3)

	// The vsync state might be reset. Set the state again (#1364).
	v.ml.SetDisplaySyncEnabled(v.vsync)
	return nil
}

func (v *view) drawable() ca.MetalDrawable {
	d, err := v.ml.NextDrawable()
	if err != nil {
		// Drawable is nil. This can happen at the initial state. Let's wait and see.
		return ca.MetalDrawable{}
	}
	return d
}
