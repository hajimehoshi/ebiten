// Copyright 2018 The Ebiten Authors
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

// Package ca provides access to Apple's Core Animation API (https://developer.apple.com/documentation/quartzcore).
//
// This package is in very early stages of development.
// It's a minimal implementation with scope limited to
// supporting the movingtriangle example.
package ca

import (
	"errors"
	"unsafe"

	"github.com/hajimehoshi/ebiten/internal/graphicsdriver/metal/mtl"
)

// Suppress the warnings about availability guard with -Wno-unguarded-availability-new.
// It is because old Xcode (8 or older?) does not accept @available syntax.

// #cgo CFLAGS: -Wno-unguarded-availability-new
// #cgo !ios CFLAGS: -mmacosx-version-min=10.12
// #cgo LDFLAGS: -framework QuartzCore -framework Foundation -framework CoreGraphics
//
// #include "ca.h"
import "C"

// Layer is an object that manages image-based content and
// allows you to perform animations on that content.
//
// Reference: https://developer.apple.com/documentation/quartzcore/calayer.
type Layer interface {
	// Layer returns the underlying CALayer * pointer.
	Layer() unsafe.Pointer
}

// MetalLayer is a Core Animation Metal layer, a layer that manages a pool of Metal drawables.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer.
type MetalLayer struct {
	metalLayer unsafe.Pointer
}

// MakeMetalLayer creates a new Core Animation Metal layer.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer.
func MakeMetalLayer() MetalLayer {
	return MetalLayer{C.MakeMetalLayer()}
}

// Layer implements the Layer interface.
func (ml MetalLayer) Layer() unsafe.Pointer { return ml.metalLayer }

// PixelFormat returns the pixel format of textures for rendering layer content.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat.
func (ml MetalLayer) PixelFormat() mtl.PixelFormat {
	return mtl.PixelFormat(C.MetalLayer_PixelFormat(ml.metalLayer))
}

// SetDevice sets the Metal device responsible for the layer's drawable resources.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478163-device.
func (ml MetalLayer) SetDevice(device mtl.Device) {
	C.MetalLayer_SetDevice(ml.metalLayer, device.Device())
}

// SetOpaque a Boolean value indicating whether the layer contains completely opaque content.
func (ml MetalLayer) SetOpaque(opaque bool) {
	if opaque {
		C.MetalLayer_SetOpaque(ml.metalLayer, 1)
	} else {
		C.MetalLayer_SetOpaque(ml.metalLayer, 0)
	}
}

// SetPixelFormat controls the pixel format of textures for rendering layer content.
//
// The pixel format for a Metal layer must be PixelFormatBGRA8UNorm, PixelFormatBGRA8UNormSRGB,
// PixelFormatRGBA16Float, PixelFormatBGRA10XR, or PixelFormatBGRA10XRSRGB.
// SetPixelFormat panics for other values.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat.
func (ml MetalLayer) SetPixelFormat(pf mtl.PixelFormat) {
	e := C.MetalLayer_SetPixelFormat(ml.metalLayer, C.uint16_t(pf))
	if e != nil {
		panic(errors.New(C.GoString(e)))
	}
}

// SetMaximumDrawableCount controls the number of Metal drawables in the resource pool
// managed by Core Animation.
//
// It can set to 2 or 3 only. SetMaximumDrawableCount panics for other values.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2938720-maximumdrawablecount.
func (ml MetalLayer) SetMaximumDrawableCount(count int) {
	e := C.MetalLayer_SetMaximumDrawableCount(ml.metalLayer, C.uint_t(count))
	if e != nil {
		panic(errors.New(C.GoString(e)))
	}
}

// SetDisplaySyncEnabled controls whether the Metal layer and its drawables
// are synchronized with the display's refresh rate.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2887087-displaysyncenabled.
func (ml MetalLayer) SetDisplaySyncEnabled(enabled bool) {
	switch enabled {
	case true:
		C.MetalLayer_SetDisplaySyncEnabled(ml.metalLayer, 1)
	case false:
		C.MetalLayer_SetDisplaySyncEnabled(ml.metalLayer, 0)
	}
}

// SetDrawableSize sets the size, in pixels, of textures for rendering layer content.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478174-drawablesize.
func (ml MetalLayer) SetDrawableSize(width, height int) {
	C.MetalLayer_SetDrawableSize(ml.metalLayer, C.double(width), C.double(height))
}

// NextDrawable returns a Metal drawable.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478172-nextdrawable.
func (ml MetalLayer) NextDrawable() (MetalDrawable, error) {
	md := C.MetalLayer_NextDrawable(ml.metalLayer)
	if md == nil {
		return MetalDrawable{}, errors.New("nextDrawable returned nil")
	}

	return MetalDrawable{md}, nil
}

// MetalDrawable is a displayable resource that can be rendered or written to by Metal.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametaldrawable.
type MetalDrawable struct {
	metalDrawable unsafe.Pointer
}

// Drawable implements the mtl.Drawable interface.
func (md MetalDrawable) Drawable() unsafe.Pointer { return md.metalDrawable }

// Texture returns a Metal texture object representing the drawable object's content.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametaldrawable/1478159-texture.
func (md MetalDrawable) Texture() mtl.Texture {
	return mtl.NewTexture(C.MetalDrawable_Texture(md.metalDrawable))
}
