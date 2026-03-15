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
	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver/metal/mtl"
)

var (
	classCAMetalLayer             = objc.GetClass("CAMetalLayer")
	classCAMetalDisplayLink       = objc.GetClass("CAMetalDisplayLink")
	classCAMetalDisplayLinkUpdate = objc.GetClass("CAMetalDisplayLinkUpdate")
)

var (
	selPixelFormat                = objc.RegisterName("pixelFormat")
	selSetDevice                  = objc.RegisterName("setDevice:")
	selSetOpaque                  = objc.RegisterName("setOpaque:")
	selSetPixelFormat             = objc.RegisterName("setPixelFormat:")
	selNew                        = objc.RegisterName("new")
	selSetColorspace              = objc.RegisterName("setColorspace:")
	selSetMaximumDrawableCount    = objc.RegisterName("setMaximumDrawableCount:")
	selSetDisplaySyncEnabled      = objc.RegisterName("setDisplaySyncEnabled:")
	selSetDrawableSize            = objc.RegisterName("setDrawableSize:")
	selNextDrawable               = objc.RegisterName("nextDrawable")
	selPresentsWithTransaction    = objc.RegisterName("presentsWithTransaction")
	selSetPresentsWithTransaction = objc.RegisterName("setPresentsWithTransaction:")
	selSetFramebufferOnly         = objc.RegisterName("setFramebufferOnly:")
	selTexture                    = objc.RegisterName("texture")
	selPresent                    = objc.RegisterName("present")
	selAlloc                      = objc.RegisterName("alloc")
	selInitWithMetalLayer         = objc.RegisterName("initWithMetalLayer:")
	selSetDelegate                = objc.RegisterName("setDelegate:")
	selAddToOneLoopForMode        = objc.RegisterName("addToRunLoop:forMode:")
	selRemoveFromRunLoopForMode   = objc.RegisterName("removeFromRunLoop:forMode:")
	selSetPaused                  = objc.RegisterName("setPaused:")
	selDrawable                   = objc.RegisterName("drawable")
	selRelease                    = objc.RegisterName("release")
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
func NewMetalLayer(colorSpace color.ColorSpace) (MetalLayer, error) {
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
	case color.ColorSpaceSRGB:
		kCGColorSpaceSRGB, err := purego.Dlsym(coreGraphics, "kCGColorSpaceSRGB")
		if err != nil {
			return MetalLayer{}, err
		}
		colorSpaceSym = kCGColorSpaceSRGB
	case color.ColorSpaceDisplayP3:
		kCGColorSpaceDisplayP3, err := purego.Dlsym(coreGraphics, "kCGColorSpaceDisplayP3")
		if err != nil {
			return MetalLayer{}, err
		}
		colorSpaceSym = kCGColorSpaceDisplayP3
	default:
		return MetalLayer{}, fmt.Errorf("ca: unsupported color space: %d", colorSpace)
	}

	layer := objc.ID(classCAMetalLayer).Send(selNew)
	// setColorspace: is available from iOS 13.0?
	// https://github.com/hajimehoshi/ebiten/commit/3af351a2aa31e30affd433429c42130015b302f3
	// TODO: Enable this on iOS as well.
	if runtime.GOOS != "ios" {
		// Dlsym returns pointer to symbol so dereference it.
		colorspace, _, _ := purego.SyscallN(cgColorSpaceCreateWithName, **(**uintptr)(unsafe.Pointer(&colorSpaceSym)))
		layer.Send(selSetColorspace, colorspace)
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
	return mtl.PixelFormat(ml.metalLayer.Send(selPixelFormat))
}

// SetDevice sets the Metal device responsible for the layer's drawable resources.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478163-device?language=objc.
func (ml MetalLayer) SetDevice(device mtl.Device) {
	ml.metalLayer.Send(selSetDevice, uintptr(device.Device()))
}

// SetOpaque a Boolean value indicating whether the layer contains completely opaque content.
func (ml MetalLayer) SetOpaque(opaque bool) {
	ml.metalLayer.Send(selSetOpaque, opaque)
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
		panic(fmt.Sprintf("ca: invalid pixel format %d", pf))
	}
	ml.metalLayer.Send(selSetPixelFormat, uint(pf))
}

// SetMaximumDrawableCount controls the number of Metal drawables in the resource pool
// managed by Core Animation.
//
// It can set to 2 or 3 only. SetMaximumDrawableCount panics for other values.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2938720-maximumdrawablecount?language=objc.
func (ml MetalLayer) SetMaximumDrawableCount(count int) {
	if count < 2 || count > 3 {
		panic(fmt.Sprintf("ca: failed trying to set maximumDrawableCount to %d outside of the valid range of [2, 3]", count))
	}
	ml.metalLayer.Send(selSetMaximumDrawableCount, count)
}

// SetDisplaySyncEnabled controls whether the Metal layer and its drawables
// are synchronized with the display's refresh rate.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/2887087-displaysyncenabled?language=objc.
func (ml MetalLayer) SetDisplaySyncEnabled(enabled bool) {
	if runtime.GOOS == "ios" {
		return
	}
	ml.metalLayer.Send(selSetDisplaySyncEnabled, enabled)
}

// SetDrawableSize sets the size, in pixels, of textures for rendering layer content.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478174-drawablesize?language=objc.
func (ml MetalLayer) SetDrawableSize(width, height int) {
	ml.metalLayer.Send(selSetDrawableSize, cocoa.CGSize{Width: cocoa.CGFloat(width), Height: cocoa.CGFloat(height)})
}

// NextDrawable returns a Metal drawable.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478172-nextdrawable?language=objc.
func (ml MetalLayer) NextDrawable() (MetalDrawable, error) {
	md := ml.metalLayer.Send(selNextDrawable)
	if md == 0 {
		return MetalDrawable{}, errors.New("nextDrawable returned nil")
	}
	return MetalDrawable{md}, nil
}

// PresentsWithTransaction returns a Boolean value that determines whether the layer presents its content using a Core Animation transaction.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478157-presentswithtransaction?language=objc
func (ml MetalLayer) PresentsWithTransaction() bool {
	return ml.metalLayer.Send(selPresentsWithTransaction) != 0
}

// SetPresentsWithTransaction sets a Boolean value that determines whether the layer presents its content using a Core Animation transaction.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478157-presentswithtransaction?language=objc
func (ml MetalLayer) SetPresentsWithTransaction(presentsWithTransaction bool) {
	ml.metalLayer.Send(selSetPresentsWithTransaction, presentsWithTransaction)
}

// SetFramebufferOnly sets a Boolean value that determines whether the layer’s textures are used only for rendering.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametallayer/1478168-framebufferonly?language=objc
func (ml MetalLayer) SetFramebufferOnly(framebufferOnly bool) {
	ml.metalLayer.Send(selSetFramebufferOnly, framebufferOnly)
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
	return mtl.NewTexture(md.metalDrawable.Send(selTexture))
}

// Present presents the drawable onscreen as soon as possible.
//
// Reference: https://developer.apple.com/documentation/metal/mtldrawable/1470284-present?language=objc.
func (md MetalDrawable) Present() {
	md.metalDrawable.Send(selPresent)
}

// MetalDisplayLink is a class your Metal app uses to register for callbacks to synchronize its animations for a display.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametaldisplaylink?language=objc
type MetalDisplayLink struct {
	objc.ID
}

// SetDelegate sets an instance of a type your app implements that responds to the system’s callbacks.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametaldisplaylink/delegate?language=objc
func (m MetalDisplayLink) SetDelegate(delegate objc.ID) {
	m.Send(selSetDelegate, delegate)
}

// AddToRunLoop registers the display link with a run loop.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametaldisplaylink/add(to:formode:)?language=objc
func (m MetalDisplayLink) AddToRunLoop(runLoop cocoa.NSRunLoop, mode cocoa.NSRunLoopMode) {
	m.Send(selAddToOneLoopForMode, runLoop, mode)
}

// RemoveFromRunLoop removes a mode’s display link from a run loop.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametaldisplaylink/remove(from:formode:)?language=objc
func (m MetalDisplayLink) RemoveFromRunLoop(runLoop cocoa.NSRunLoop, mode cocoa.NSRunLoopMode) {
	m.Send(selRemoveFromRunLoopForMode, runLoop, mode)
}

// SetPaused sets a Boolean value that indicates whether the system suspends the display link’s notifications to the target.
//
// https://developer.apple.com/documentation/quartzcore/cametaldisplaylink/ispaused?language=objc
func (m MetalDisplayLink) SetPaused(paused bool) {
	m.Send(selSetPaused, paused)
}

func (m MetalDisplayLink) Release() {
	m.Send(selRelease)
}

// NewMetalDisplayLink creates a display link for Metal from a Core Animation layer.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametaldisplaylink/init(metallayer:)?language=objc
func NewMetalDisplayLink(metalLayer MetalLayer) MetalDisplayLink {
	displayLink := objc.ID(classCAMetalDisplayLink).Send(selAlloc).Send(selInitWithMetalLayer, metalLayer.metalLayer)
	return MetalDisplayLink{displayLink}
}

// MetalDisplayLinkUpdate stores information about a single update from a Metal display link instance.
//
// Reference: https://developer.apple.com/documentation/quartzcore/cametaldisplaylink/update?language=objc
type MetalDisplayLinkUpdate struct {
	objc.ID
}

// Drawable returns the Metal drawable your app uses to render the next frame.
//
// https://developer.apple.com/documentation/quartzcore/cametaldisplaylink/update/drawable?language=objc
func (m MetalDisplayLinkUpdate) Drawable() MetalDrawable {
	return MetalDrawable{m.Send(selDrawable)}
}
