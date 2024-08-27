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

// Package ca provides access to Apple's Core Animation API (https://developer.apple.com/documentation/quartzcore).
//
// This package is in very early stages of development.
// It's a minimal implementation with scope limited to
// supporting the movingtriangle example.
package ca

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

// Layer is an object that manages image-based content and
// allows you to perform animations on that content.
//
// Reference: https://developer.apple.com/documentation/quartzcore/calayer?language=objc.
type Layer interface {
	// Layer returns the underlying CALayer * pointer.
	Layer() unsafe.Pointer
}

// MetalLayer is a Core Animation Metal layer, a layer that manages a pool of Metal drawables.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer?language=objc.
type MetalLayer struct {
	metalLayer objc.ID
}

// NewMetalLayer creates a new Core Animation Metal layer.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer?language=objc.
func NewMetalLayer(colorSpace graphicsdriver.ColorSpace) (MetalLayer, error) {
	coreGraphics, err := purego.Dlopen("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return MetalLayer{}, err
	}

	cgColorSpaceCreateWithName, err := purego.Dlsym(coreGraphics, "CGColorSpaceCreateWithName")
	if err != nil {
		return MetalLayer{}, err
	}

	cgColorSpaceRelease, err := purego.Dlsym(coreGraphics, "CGColorSpaceRelease")
	if err != nil {
		return MetalLayer{}, err
	}

	var colorSpaceSym uintptr
	switch colorSpace {
	case graphicsdriver.ColorSpaceSRGB:
		kCGColorSpaceSRGB, err := purego.Dlsym(coreGraphics, "kCGColorSpaceSRGB")
		if err != nil {
			return MetalLayer{}, err
		}
		colorSpaceSym = kCGColorSpaceSRGB
	default:
		fallthrough
	case graphicsdriver.ColorSpaceDisplayP3:
		kCGColorSpaceDisplayP3, err := purego.Dlsym(coreGraphics, "kCGColorSpaceDisplayP3")
		if err != nil {
			return MetalLayer{}, err
		}
		colorSpaceSym = kCGColorSpaceDisplayP3
	}

	layer := objc.ID(objc.GetClass("CAMetalLayer")).Send(objc.RegisterName("new"))
	// setColorspace: is available from iOS 13.0?
	// https://github.com/hajimehoshi/ebiten/commit/3af351a2aa31e30affd433429c42130015b302f3
	// TODO: Enable this on iOS as well.
	if runtime.GOOS != "ios" {
		// Dlsym returns pointer to symbol so dereference it.
		colorspace, _, _ := purego.SyscallN(cgColorSpaceCreateWithName, **(**uintptr)(unsafe.Pointer(&colorSpaceSym)))
		layer.Send(objc.RegisterName("setColorspace:"), colorspace)
		purego.SyscallN(cgColorSpaceRelease, colorspace)
	}
	return MetalLayer{layer}, nil
}

// Layer implements the Layer interface.
func (ml MetalLayer) Layer() unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&ml.metalLayer))
}

// PixelFormat returns the pixel format of textures for rendering layer content.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat?language=objc.
func (ml MetalLayer) PixelFormat() mtl.PixelFormat {
	return mtl.PixelFormat(ml.metalLayer.Send(objc.RegisterName("pixelFormat")))
}

// SetDevice sets the Metal device responsible for the layer's drawable resources.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478163-device?language=objc.
func (ml MetalLayer) SetDevice(device mtl.Device) {
	ml.metalLayer.Send(objc.RegisterName("setDevice:"), uintptr(device.Device()))
}

// SetOpaque a Boolean value indicating whether the layer contains completely opaque content.
func (ml MetalLayer) SetOpaque(opaque bool) {
	ml.metalLayer.Send(objc.RegisterName("setOpaque:"), opaque)
}

// SetPixelFormat controls the pixel format of textures for rendering layer content.
//
// The pixel format for a Metal layer must be PixelFormatBGRA8UNorm, PixelFormatBGRA8UNormSRGB,
// PixelFormatRGBA16Float, PixelFormatBGRA10XR, or PixelFormatBGRA10XRSRGB.
// SetPixelFormat panics for other values.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478155-pixelformat?language=objc.
func (ml MetalLayer) SetPixelFormat(pf mtl.PixelFormat) {
	switch pf {
	case mtl.PixelFormatRGBA8UNorm, mtl.PixelFormatRGBA8UNormSRGB, mtl.PixelFormatBGRA8UNorm, mtl.PixelFormatBGRA8UNormSRGB, mtl.PixelFormatStencil8:
	default:
		panic(errors.New(fmt.Sprintf("invalid pixel format %d", pf)))
	}
	ml.metalLayer.Send(objc.RegisterName("setPixelFormat:"), uint(pf))
}

// SetMaximumDrawableCount controls the number of Metal drawables in the resource pool
// managed by Core Animation.
//
// It can set to 2 or 3 only. SetMaximumDrawableCount panics for other values.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2938720-maximumdrawablecount?language=objc.
func (ml MetalLayer) SetMaximumDrawableCount(count int) {
	if count < 2 || count > 3 {
		panic(errors.New(fmt.Sprintf("failed trying to set maximumDrawableCount to %d outside of the valid range of [2, 3]", count)))
	}
	ml.metalLayer.Send(objc.RegisterName("setMaximumDrawableCount:"), count)
}

// SetDisplaySyncEnabled controls whether the Metal layer and its drawables
// are synchronized with the display's refresh rate.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2887087-displaysyncenabled?language=objc.
func (ml MetalLayer) SetDisplaySyncEnabled(enabled bool) {
	if runtime.GOOS == "ios" {
		return
	}
	ml.metalLayer.Send(objc.RegisterName("setDisplaySyncEnabled:"), enabled)
}

// SetDrawableSize sets the size, in pixels, of textures for rendering layer content.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478174-drawablesize?language=objc.
func (ml MetalLayer) SetDrawableSize(width, height int) {
	// TODO: once objc supports calling functions with struct arguments replace this with just a ID.Send call
	var sel_setDrawableSize = objc.RegisterName("setDrawableSize:")
	sig := cocoa.NSMethodSignature_instanceMethodSignatureForSelector(objc.ID(objc.GetClass("CAMetalLayer")), sel_setDrawableSize)
	inv := cocoa.NSInvocation_invocationWithMethodSignature(sig)
	inv.SetTarget(ml.metalLayer)
	inv.SetSelector(sel_setDrawableSize)
	inv.SetArgumentAtIndex(unsafe.Pointer(&cocoa.CGSize{Width: cocoa.CGFloat(width), Height: cocoa.CGFloat(height)}), 2)
	inv.Invoke()
}

// NextDrawable returns a Metal drawable.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478172-nextdrawable?language=objc.
func (ml MetalLayer) NextDrawable() (MetalDrawable, error) {
	md := ml.metalLayer.Send(objc.RegisterName("nextDrawable"))
	if md == 0 {
		return MetalDrawable{}, errors.New("nextDrawable returned nil")
	}
	return MetalDrawable{md}, nil
}

// PresentsWithTransaction returns a Boolean value that determines whether the layer presents its content using a Core Animation transaction.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478157-presentswithtransaction?language=objc
func (ml MetalLayer) PresentsWithTransaction() bool {
	return ml.metalLayer.Send(objc.RegisterName("presentsWithTransaction")) != 0
}

// SetPresentsWithTransaction sets a Boolean value that determines whether the layer presents its content using a Core Animation transaction.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478157-presentswithtransaction?language=objc
func (ml MetalLayer) SetPresentsWithTransaction(presentsWithTransaction bool) {
	ml.metalLayer.Send(objc.RegisterName("setPresentsWithTransaction:"), presentsWithTransaction)
}

// SetFramebufferOnly sets a Boolean value that determines whether the layer’s textures are used only for rendering.
//
// https://developer.apple.com/documentation/quartzcore/cametallayer/1478168-framebufferonly?language=objc
func (ml MetalLayer) SetFramebufferOnly(framebufferOnly bool) {
	ml.metalLayer.Send(objc.RegisterName("setFramebufferOnly:"), framebufferOnly)
}

// MetalDrawable is a displayable resource that can be rendered or written to by Metal.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametaldrawable?language=objc.
type MetalDrawable struct {
	metalDrawable objc.ID
}

// Drawable implements the mtl.Drawable interface.
func (md MetalDrawable) Drawable() unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&md.metalDrawable))
}

// Texture returns a Metal texture object representing the drawable object's content.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametaldrawable/1478159-texture?language=objc.
func (md MetalDrawable) Texture() mtl.Texture {
	return mtl.NewTexture(md.metalDrawable.Send(objc.RegisterName("texture")))
}

// Present presents the drawable onscreen as soon as possible.
//
// Reference: https://developer.apple.com/documentation/metal/mtldrawable/1470284-present?language=objc.
func (md MetalDrawable) Present() {
	md.metalDrawable.Send(objc.RegisterName("present"))
}
