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
	"image"
	"image/color"
	"runtime"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/math"
	"github.com/hajimehoshi/ebiten/internal/opengl"
)

// MaxImageSize represents the maximum width/height of an image.
const MaxImageSize = graphics.MaxImageSize

// QuadVertexSizeInBytes returns the byte size of vertices for a quadrilateral.
func QuadVertexSizeInBytes() int {
	return graphics.QuadVertexSizeInBytes()
}

// drawImageHistoryItem is an item for history of draw-image commands.
type drawImageHistoryItem struct {
	image    *Image
	vertices []float32
	colorm   affine.ColorM
	mode     opengl.CompositeMode
}

// canMerge returns a boolean value indicating whether the drawImageHistoryItem d
// can be merged with the given conditions.
func (d *drawImageHistoryItem) canMerge(image *Image, colorm *affine.ColorM, mode opengl.CompositeMode) bool {
	if d.image != image {
		return false
	}
	if !d.colorm.Equals(colorm) {
		return false
	}
	if d.mode != mode {
		return false
	}
	return true
}

// Image represents an image that can be restored when GL context is lost.
type Image struct {
	image  *graphics.Image
	filter opengl.Filter

	// baseImage and baseColor are exclusive.
	basePixels       []uint8
	baseColor        color.RGBA
	drawImageHistory []*drawImageHistoryItem

	// stale indicates whether the image needs to be synced with GPU as soon as possible.
	stale bool

	// volatile indicates whether the image is cleared whenever a frame starts.
	volatile bool

	// screen indicates whether the image is used as an actual screen.
	screen bool

	offsetX float64
	offsetY float64
}

// NewImage creates an empty image with the given size and filter.
func NewImage(width, height int, filter opengl.Filter, volatile bool) *Image {
	i := &Image{
		image:    graphics.NewImage(width, height, filter),
		filter:   filter,
		volatile: volatile,
	}
	theImages.add(i)
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

// NewImageFromImage creates an image with source image.
func NewImageFromImage(source image.Image, filter opengl.Filter) *Image {
	size := source.Bounds().Size()
	width, height := size.X, size.Y
	rgbaImg := CopyImage(source)
	w2, h2 := math.NextPowerOf2Int(width), math.NextPowerOf2Int(height)
	p := make([]uint8, 4*w2*h2)
	for j := 0; j < height; j++ {
		copy(p[j*w2*4:(j+1)*w2*4], rgbaImg.Pix[j*rgbaImg.Stride:])
	}
	i := &Image{
		image:      graphics.NewImageFromImage(rgbaImg, width, height, filter),
		basePixels: p,
		filter:     filter,
	}
	theImages.add(i)
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

// NewScreenFramebufferImage creates a special image that framebuffer is one for the screen.
func NewScreenFramebufferImage(width, height int, offsetX, offsetY float64) *Image {
	i := &Image{
		image:    graphics.NewScreenFramebufferImage(width, height, offsetX, offsetY),
		volatile: true,
		screen:   true,
		offsetX:  offsetX,
		offsetY:  offsetY,
	}
	theImages.add(i)
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

// BasePixelsForTesting returns the image's basePixels for testing.
func (p *Image) BasePixelsForTesting() []uint8 {
	return p.basePixels
}

// Size returns the image's size.
func (p *Image) Size() (int, int) {
	return p.image.Size()
}

// makeStale makes the image stale.
func (p *Image) makeStale() {
	p.basePixels = nil
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = true
}

// clearIfVolatile clears the image if the image is volatile.
func (p *Image) clearIfVolatile() {
	if !p.volatile {
		return
	}
	p.basePixels = nil
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	if p.image == nil {
		panic("not reached")
	}
	p.image.Fill(0, 0, 0, 0)
}

// Fill fills the image with the given color.
func (p *Image) Fill(r, g, b, a uint8) {
	theImages.makeStaleIfDependingOn(p)
	p.basePixels = nil
	p.baseColor = color.RGBA{r, g, b, a}
	p.drawImageHistory = nil
	p.stale = false
	p.image.Fill(r, g, b, a)
}

// ReplacePixels replaces the image pixels with the given pixels slice.
func (p *Image) ReplacePixels(pixels []uint8) {
	theImages.makeStaleIfDependingOn(p)
	p.image.ReplacePixels(pixels)
	p.basePixels = pixels
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
}

// DrawImage draws a given image img to the image.
func (p *Image) DrawImage(img *Image, vertices []float32, colorm *affine.ColorM, mode opengl.CompositeMode) {
	theImages.makeStaleIfDependingOn(p)
	if img.stale || img.volatile || !IsRestoringEnabled() {
		p.makeStale()
	} else {
		p.appendDrawImageHistory(img, vertices, colorm, mode)
	}
	p.image.DrawImage(img.image, vertices, colorm, mode)
}

// appendDrawImageHistory appends a draw-image history item to the image.
func (p *Image) appendDrawImageHistory(image *Image, vertices []float32, colorm *affine.ColorM, mode opengl.CompositeMode) {
	if p.stale || p.volatile {
		return
	}
	if len(p.drawImageHistory) > 0 {
		last := p.drawImageHistory[len(p.drawImageHistory)-1]
		if last.canMerge(image, colorm, mode) {
			last.vertices = append(last.vertices, vertices...)
			return
		}
	}
	const maxDrawImageHistoryNum = 100
	if len(p.drawImageHistory)+1 > maxDrawImageHistoryNum {
		p.makeStale()
		return
	}
	// All images must be resolved and not stale each after frame.
	// So we don't have to care if image is stale or not here.
	item := &drawImageHistoryItem{
		image:    image,
		vertices: vertices,
		colorm:   *colorm,
		mode:     mode,
	}
	p.drawImageHistory = append(p.drawImageHistory, item)
}

// At returns a color value at (x, y).
//
// Note that this must not be called until context is available.
func (p *Image) At(x, y int) (color.RGBA, error) {
	w, h := p.image.Size()
	w2, h2 := math.NextPowerOf2Int(w), math.NextPowerOf2Int(h)
	if x < 0 || y < 0 || w2 <= x || h2 <= y {
		return color.RGBA{}, nil
	}
	if p.basePixels == nil || p.drawImageHistory != nil || p.stale {
		if err := p.readPixelsFromGPU(p.image); err != nil {
			return color.RGBA{}, err
		}
	}
	idx := 4*x + 4*y*w2
	r, g, b, a := p.basePixels[idx], p.basePixels[idx+1], p.basePixels[idx+2], p.basePixels[idx+3]
	return color.RGBA{r, g, b, a}, nil
}

// makeStaleIfDependingOn makes the image stale if the image depends on target.
func (p *Image) makeStaleIfDependingOn(target *Image) {
	if p.stale {
		return
	}
	if p.dependsOn(target) {
		p.makeStale()
	}
}

// readPixelsFromGPU reads the pixels from GPU and resolves the image's 'stale' state.
func (p *Image) readPixelsFromGPU(image *graphics.Image) error {
	var err error
	p.basePixels, err = image.Pixels()
	if err != nil {
		return err
	}
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	return nil
}

// resolveStale resolves the image's 'stale' state.
func (p *Image) resolveStale() error {
	if !IsRestoringEnabled() {
		return nil
	}
	if p.volatile {
		return nil
	}
	if !p.stale {
		return nil
	}
	return p.readPixelsFromGPU(p.image)
}

// dependsOn returns a boolean value indicating whether the image depends on target.
func (p *Image) dependsOn(target *Image) bool {
	for _, c := range p.drawImageHistory {
		if c.image == target {
			return true
		}
	}
	return false
}

// dependingImages returns all images that is depended by the image.
func (p *Image) dependingImages() map[*Image]struct{} {
	r := map[*Image]struct{}{}
	for _, c := range p.drawImageHistory {
		r[c.image] = struct{}{}
	}
	return r
}

// hasDependency returns a boolean value indicating whether the image depends on another image.
func (p *Image) hasDependency() bool {
	if p.stale {
		return false
	}
	return len(p.drawImageHistory) > 0
}

// Restore restores *graphics.Image from the pixels using its state.
func (p *Image) restore() error {
	w, h := p.image.Size()
	if p.screen {
		// The screen image should also be recreated because framebuffer might
		// be changed.
		p.image = graphics.NewScreenFramebufferImage(w, h, p.offsetX, p.offsetY)
		p.basePixels = nil
		p.baseColor = color.RGBA{}
		p.drawImageHistory = nil
		p.stale = false
		return nil
	}
	if p.volatile {
		p.image = graphics.NewImage(w, h, p.filter)
		p.basePixels = nil
		p.baseColor = color.RGBA{}
		p.drawImageHistory = nil
		p.stale = false
		return nil
	}
	if p.stale {
		// TODO: panic here?
		return errors.New("restorable: pixels must not be stale when restoring")
	}
	w2, h2 := math.NextPowerOf2Int(w), math.NextPowerOf2Int(h)
	img := image.NewRGBA(image.Rect(0, 0, w2, h2))
	if p.basePixels != nil {
		for j := 0; j < h; j++ {
			copy(img.Pix[j*img.Stride:], p.basePixels[j*w2*4:(j+1)*w2*4])
		}
	}
	gimg := graphics.NewImageFromImage(img, w, h, p.filter)
	if p.baseColor != (color.RGBA{}) {
		if p.basePixels != nil {
			panic("not reached")
		}
		gimg.Fill(p.baseColor.R, p.baseColor.G, p.baseColor.B, p.baseColor.A)
	}
	for _, c := range p.drawImageHistory {
		// All dependencies must be already resolved.
		if c.image.hasDependency() {
			panic("not reached")
		}
		gimg.DrawImage(c.image.image, c.vertices, &c.colorm, c.mode)
	}
	p.image = gimg

	var err error
	p.basePixels, err = gimg.Pixels()
	if err != nil {
		return err
	}
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	return nil
}

// Dispose disposes the image.
//
// After disposing, calling the funciton of the image causes unexpected results.
func (p *Image) Dispose() {
	theImages.makeStaleIfDependingOn(p)
	p.image.Dispose()
	p.image = nil
	p.basePixels = nil
	p.baseColor = color.RGBA{}
	p.drawImageHistory = nil
	p.stale = false
	theImages.remove(p)
	runtime.SetFinalizer(p, nil)
}

// IsInvalidated returns a boolean value indicating whether the image is invalidated.
//
// If an image is invalidated, GL context is lost and all the images should be restored asap.
func (p *Image) IsInvalidated() (bool, error) {
	// FlushCommands is required because c.offscreen.impl might not have an actual texture.
	if err := graphics.FlushCommands(); err != nil {
		return false, err
	}

	if !IsRestoringEnabled() {
		return false, nil
	}
	return p.image.IsInvalidated(), nil
}
