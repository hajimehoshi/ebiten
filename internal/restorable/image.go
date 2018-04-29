// Copyright 2016 The Ebiten Authors
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

package restorable

import (
	"errors"
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

// drawImageHistoryItem is an item for history of draw-image commands.
type drawImageHistoryItem struct {
	image    *Image
	vertices [][]float32
	colorm   *affine.ColorM
	mode     opengl.CompositeMode
	filter   graphics.Filter
}

// canMerge returns a boolean value indicating whether the drawImageHistoryItem d
// can be merged with the given conditions.
func (d *drawImageHistoryItem) canMerge(image *Image, colorm *affine.ColorM, mode opengl.CompositeMode, filter graphics.Filter) bool {
	if d.image != image {
		return false
	}
	if !d.colorm.Equals(colorm) {
		return false
	}
	if d.mode != mode {
		return false
	}
	if d.filter != filter {
		return false
	}
	return true
}

// Image represents an image that can be restored when GL context is lost.
type Image struct {
	image *graphics.Image

	basePixels []byte

	// drawImageHistory is a set of draw-image commands.
	// TODO: This should be merged with the similar command queue in package graphics (#433).
	drawImageHistory []*drawImageHistoryItem

	// stale indicates whether the image needs to be synced with GPU as soon as possible.
	stale bool

	// volatile indicates whether the image is cleared whenever a frame starts.
	volatile bool

	// screen indicates whether the image is used as an actual screen.
	screen bool
}

var dummyImage = newImageWithoutInit(16, 16, false)

// newImageWithoutInit creates an image without initialization.
//
// Note that Dispose is not called automatically.
func newImageWithoutInit(width, height int, volatile bool) *Image {
	i := &Image{
		image:    graphics.NewImage(width, height),
		volatile: volatile,
	}
	theImages.add(i)
	return i
}

// NewImage creates an empty image with the given size.
//
// The returned image is cleared.
//
// Note that Dispose is not called automatically.
func NewImage(width, height int, volatile bool) *Image {
	i := newImageWithoutInit(width, height, volatile)
	i.ReplacePixels(nil, 0, 0, width, height)
	return i
}

// NewScreenFramebufferImage creates a special image that framebuffer is one for the screen.
//
// The returned image is cleared.
//
// Note that Dispose is not called automatically.
func NewScreenFramebufferImage(width, height int) *Image {
	i := &Image{
		image:    graphics.NewScreenFramebufferImage(width, height),
		volatile: false,
		screen:   true,
	}
	theImages.add(i)
	i.ReplacePixels(nil, 0, 0, width, height)
	return i
}

// BasePixelsForTesting returns the image's basePixels for testing.
func (i *Image) BasePixelsForTesting() []byte {
	return i.basePixels
}

// Size returns the image's size.
func (i *Image) Size() (int, int) {
	return i.image.Size()
}

// makeStale makes the image stale.
func (i *Image) makeStale() {
	i.basePixels = nil
	i.drawImageHistory = nil
	i.stale = true

	// Don't have to call makeStale recursively here.
	// Restoring is done after topological sorting is done.
	// If an image depends on another stale image, this means that
	// the former image can be restored from the latest state of the latter image.
}

// ReplacePixels replaces the image pixels with the given pixels slice.
//
// If pixels is nil, ReplacePixels clears the specified reagion.
func (i *Image) ReplacePixels(pixels []byte, x, y, width, height int) {
	w, h := i.image.Size()
	if width <= 0 || height <= 0 {
		panic("restorable: width/height must be positive")
	}
	if x < 0 || y < 0 || w <= x || h <= y || x+width <= 0 || y+height <= 0 || w < x+width || h < y+height {
		panic(fmt.Sprintf("restorable: out of range x: %d, y: %d, width: %d, height: %d", x, y, width, height))
	}

	// TODO: Avoid making other images stale if possible. (#514)
	// For this purpuse, images should remember which part of that is used for DrawImage.
	theImages.makeStaleIfDependingOn(i)

	if pixels != nil {
		i.image.ReplacePixels(pixels, x, y, width, height)
	} else {
		// There is not 'drawImageHistoryItem' for this image and dummyImage.
		// This means dummyImage might not be restored yet when this image is restored.
		// However, that's ok since this image will be stale or have updated pixel data
		// and this image can be restored without dummyImage.
		w, h := dummyImage.Size()
		geom := (*affine.GeoM)(nil).Scale(float64(width)/float64(w), float64(height)/float64(h))
		geom = geom.Translate(float64(x), float64(y))
		colorm := (*affine.ColorM)(nil).Scale(0, 0, 0, 0)
		vs := vertices(w, h, 0, 0, w, h, geom)
		i.image.DrawImage(dummyImage.image, vs, colorm, opengl.CompositeModeCopy, graphics.FilterNearest)
	}

	if x == 0 && y == 0 && width == w && height == h {
		if i.basePixels == nil {
			i.basePixels = make([]byte, 4*w*h)
		}
		copy(i.basePixels, pixels)
		i.drawImageHistory = nil
		i.stale = false
		return
	}
	if i.basePixels == nil {
		i.makeStale()
		return
	}
	if len(i.drawImageHistory) > 0 {
		i.makeStale()
		return
	}
	idx := 4 * (y*w + x)
	if pixels != nil {
		for j := 0; j < height; j++ {
			copy(i.basePixels[idx:idx+4*width], pixels[4*j*width:4*(j+1)*width])
			idx += 4 * w
		}
	} else {
		zeros := make([]byte, 4*width)
		for j := 0; j < height; j++ {
			copy(i.basePixels[idx:idx+4*width], zeros)
			idx += 4 * w
		}
	}
	i.stale = false
}

// DrawImage draws a given image img to the image.
func (i *Image) DrawImage(img *Image, sx0, sy0, sx1, sy1 int, geom *affine.GeoM, colorm *affine.ColorM, mode opengl.CompositeMode, filter graphics.Filter) {
	w, h := img.Size()
	vs := vertices(w, h, sx0, sy0, sx1, sy1, geom)
	if vs == nil {
		return
	}
	theImages.makeStaleIfDependingOn(i)

	if img.stale || img.volatile || i.screen || !IsRestoringEnabled() {
		i.makeStale()
	} else {
		i.appendDrawImageHistory(img, vs, colorm, mode, filter)
	}
	i.image.DrawImage(img.image, vs, colorm, mode, filter)
}

// appendDrawImageHistory appends a draw-image history item to the image.
func (i *Image) appendDrawImageHistory(image *Image, vertices []float32, colorm *affine.ColorM, mode opengl.CompositeMode, filter graphics.Filter) {
	if i.stale || i.volatile || i.screen {
		return
	}
	if len(i.drawImageHistory) > 0 {
		last := i.drawImageHistory[len(i.drawImageHistory)-1]
		if last.canMerge(image, colorm, mode, filter) {
			last.vertices = append(last.vertices, vertices)
			return
		}
	}
	const maxDrawImageHistoryNum = 100
	if len(i.drawImageHistory)+1 > maxDrawImageHistoryNum {
		i.makeStale()
		return
	}
	// All images must be resolved and not stale each after frame.
	// So we don't have to care if image is stale or not here.
	item := &drawImageHistoryItem{
		image:    image,
		vertices: [][]float32{vertices},
		colorm:   colorm,
		mode:     mode,
		filter:   filter,
	}
	i.drawImageHistory = append(i.drawImageHistory, item)
}

// At returns a color value at (x, y).
//
// Note that this must not be called until context is available.
func (i *Image) At(x, y int) (color.RGBA, error) {
	w, h := i.image.Size()
	if x < 0 || y < 0 || w <= x || h <= y {
		return color.RGBA{}, nil
	}

	if i.basePixels == nil || i.drawImageHistory != nil || i.stale {
		if err := graphics.FlushCommands(); err != nil {
			return color.RGBA{}, err
		}
		if err := i.readPixelsFromGPU(); err != nil {
			return color.RGBA{}, err
		}
		i.drawImageHistory = nil
		i.stale = false
	}
	idx := 4*x + 4*y*w
	r, g, b, a := i.basePixels[idx], i.basePixels[idx+1], i.basePixels[idx+2], i.basePixels[idx+3]
	return color.RGBA{r, g, b, a}, nil
}

// makeStaleIfDependingOn makes the image stale if the image depends on target.
func (i *Image) makeStaleIfDependingOn(target *Image) {
	if i.stale {
		return
	}
	if i.dependsOn(target) {
		i.makeStale()
	}
}

// readPixelsFromGPU reads the pixels from GPU and resolves the image's 'stale' state.
func (i *Image) readPixelsFromGPU() error {
	var err error
	i.basePixels, err = i.image.Pixels()
	if err != nil {
		return err
	}
	i.drawImageHistory = nil
	i.stale = false
	return nil
}

// resolveStale resolves the image's 'stale' state.
func (i *Image) resolveStale() error {
	if !IsRestoringEnabled() {
		return nil
	}

	if i.volatile {
		return nil
	}
	if i.screen {
		return nil
	}
	if !i.stale {
		return nil
	}
	return i.readPixelsFromGPU()
}

// dependsOn returns a boolean value indicating whether the image depends on target.
func (i *Image) dependsOn(target *Image) bool {
	for _, c := range i.drawImageHistory {
		if c.image == target {
			return true
		}
	}
	return false
}

// dependingImages returns all images that is depended by the image.
func (i *Image) dependingImages() map[*Image]struct{} {
	r := map[*Image]struct{}{}
	for _, c := range i.drawImageHistory {
		r[c.image] = struct{}{}
	}
	return r
}

// hasDependency returns a boolean value indicating whether the image depends on another image.
func (i *Image) hasDependency() bool {
	if i.stale {
		return false
	}
	return len(i.drawImageHistory) > 0
}

// Restore restores *graphics.Image from the pixels using its state.
func (i *Image) restore() error {
	w, h := i.image.Size()
	if i.screen {
		// The screen image should also be recreated because framebuffer might
		// be changed.
		i.image = graphics.NewScreenFramebufferImage(w, h)
		i.basePixels = nil
		i.drawImageHistory = nil
		i.stale = false
		return nil
	}
	if i.volatile {
		i.image = graphics.NewImage(w, h)
		i.basePixels = nil
		i.drawImageHistory = nil
		i.stale = false
		return nil
	}
	if i.stale {
		// TODO: panic here?
		return errors.New("restorable: pixels must not be stale when restoring")
	}
	gimg := graphics.NewImage(w, h)
	if i.basePixels != nil {
		gimg.ReplacePixels(i.basePixels, 0, 0, w, h)
	} else {
		// Clear the image explicitly.
		pix := make([]uint8, w*h*4)
		gimg.ReplacePixels(pix, 0, 0, w, h)
	}
	for _, c := range i.drawImageHistory {
		// All dependencies must be already resolved.
		if c.image.hasDependency() {
			panic("not reached")
		}
		vs := []float32{}
		for _, v := range c.vertices {
			vs = append(vs, v...)
		}
		gimg.DrawImage(c.image.image, vs, c.colorm, c.mode, c.filter)
	}
	i.image = gimg

	var err error
	i.basePixels, err = gimg.Pixels()
	if err != nil {
		return err
	}
	i.drawImageHistory = nil
	i.stale = false
	return nil
}

// Dispose disposes the image.
//
// After disposing, calling the function of the image causes unexpected results.
func (i *Image) Dispose() {
	theImages.remove(i)

	i.image.Dispose()
	i.image = nil
	i.basePixels = nil
	i.drawImageHistory = nil
	i.stale = false
}

// IsInvalidated returns a boolean value indicating whether the image is invalidated.
//
// If an image is invalidated, GL context is lost and all the images should be restored asap.
func (i *Image) IsInvalidated() (bool, error) {
	// FlushCommands is required because c.offscreen.impl might not have an actual texture.
	if err := graphics.FlushCommands(); err != nil {
		return false, err
	}
	if !IsRestoringEnabled() {
		return false, nil
	}

	return i.image.IsInvalidated(), nil
}
