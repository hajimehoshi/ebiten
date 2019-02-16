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
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
)

type Pixels struct {
	pixels []byte

	length int

	// color is used only when pixels == nil
	color color.RGBA
}

func (p *Pixels) ensurePixels() {
	if p.pixels != nil {
		return
	}
	p.pixels = make([]byte, p.length)
	if p.color.A == 0 {
		return
	}
	for i := 0; i < p.length/4; i++ {
		p.pixels[4*i] = p.color.R
		p.pixels[4*i+1] = p.color.G
		p.pixels[4*i+2] = p.color.B
		p.pixels[4*i+3] = p.color.A
	}
}

func (p *Pixels) CopyFrom(pix []byte, from int) {
	p.ensurePixels()
	copy(p.pixels[from:from+len(pix)], pix)
}

func (p *Pixels) At(i int) byte {
	if i < 0 || p.length <= i {
		panic(fmt.Sprintf("restorable: index out of range: %d for length: %d", i, p.length))
	}
	if p.pixels == nil {
		switch i % 4 {
		case 0:
			return p.color.R
		case 1:
			return p.color.G
		case 2:
			return p.color.B
		case 3:
			return p.color.A
		}
	}
	return p.pixels[i]
}

func (p *Pixels) Slice() []byte {
	p.ensurePixels()
	return p.pixels
}

// drawImageHistoryItem is an item for history of draw-image commands.
type drawImageHistoryItem struct {
	image    *Image
	vertices []float32
	indices  []uint16
	colorm   *affine.ColorM
	mode     graphics.CompositeMode
	filter   graphics.Filter
	address  graphics.Address
}

// Image represents an image that can be restored when GL context is lost.
type Image struct {
	image *graphicscommand.Image

	basePixels *Pixels

	// drawImageHistory is a set of draw-image commands.
	// TODO: This should be merged with the similar command queue in package graphics (#433).
	drawImageHistory []*drawImageHistoryItem

	// stale indicates whether the image needs to be synced with GPU as soon as possible.
	stale bool

	// volatile indicates whether the image is cleared whenever a frame starts.
	volatile bool

	// screen indicates whether the image is used as an actual screen.
	screen bool

	w2 int
	h2 int

	// priority indicates whether the image is restored in high priority when context-lost happens.
	priority bool
}

var emptyImage *Image

func init() {
	const w, h = 16, 16
	emptyImage = &Image{
		image:    graphicscommand.NewImage(w, h),
		priority: true,
	}
	pix := make([]byte, 4*w*h)
	for i := range pix {
		pix[i] = 0xff
	}
	emptyImage.ReplacePixels(pix, 0, 0, w, h)
	theImages.add(emptyImage)
}

// NewImage creates an empty image with the given size.
//
// The returned image is cleared.
//
// Note that Dispose is not called automatically.
func NewImage(width, height int) *Image {
	i := &Image{
		image: graphicscommand.NewImage(width, height),
	}
	i.clear()
	theImages.add(i)
	return i
}

func (i *Image) MakeVolatile() {
	i.volatile = true
}

// NewScreenFramebufferImage creates a special image that framebuffer is one for the screen.
//
// The returned image is cleared.
//
// Note that Dispose is not called automatically.
func NewScreenFramebufferImage(width, height int) *Image {
	i := &Image{
		image:  graphicscommand.NewScreenFramebufferImage(width, height),
		screen: true,
	}
	i.clear()
	theImages.add(i)
	return i
}

func (i *Image) Fill(r, g, b, a uint8) {
	theImages.makeStaleIfDependingOn(i)
	i.fill(r, g, b, a)
}

func (i *Image) clear() {
	i.fill(0, 0, 0, 0)
}

func (i *Image) fill(r, g, b, a uint8) {
	if i.priority {
		panic("restorable: clear cannot be called on a priority image")
	}

	rf := float32(0)
	gf := float32(0)
	bf := float32(0)
	af := float32(0)
	if a > 0 {
		rf = float32(r) / float32(a)
		gf = float32(g) / float32(a)
		bf = float32(b) / float32(a)
		af = float32(a) / 0xff
	}

	// There are not 'drawImageHistoryItem's for this image and emptyImage.
	// As emptyImage is a priority image, this is restored before other regular images are restored.
	dw, dh := i.internalSize()
	sw, sh := emptyImage.Size()
	vs := graphics.QuadVertices(dw, dh, 0, 0, sw, sh,
		float32(dw)/float32(sw), 0, 0, float32(dh)/float32(sh), 0, 0,
		rf, gf, bf, af)
	is := graphics.QuadIndices()
	c := graphics.CompositeModeCopy
	if a == 0 {
		c = graphics.CompositeModeClear
	}
	i.image.DrawImage(emptyImage.image, vs, is, nil, c, graphics.FilterNearest, graphics.AddressClampToZero)

	w, h := i.Size()
	i.basePixels = &Pixels{
		color:  color.RGBA{r, g, b, a},
		length: 4 * w * h,
	}
	i.drawImageHistory = nil
	i.stale = false
}

func (i *Image) IsVolatile() bool {
	return i.volatile
}

// BasePixelsForTesting returns the image's basePixels for testing.
func (i *Image) BasePixelsForTesting() *Pixels {
	return i.basePixels
}

// Size returns the image's size.
func (i *Image) Size() (int, int) {
	return i.image.Size()
}

// internalSize returns the size of the internal texture.
func (i *Image) internalSize() (int, int) {
	if i.w2 == 0 || i.h2 == 0 {
		w, h := i.image.Size()
		i.w2 = graphics.InternalImageSize(w)
		i.h2 = graphics.InternalImageSize(h)
	}
	return i.w2, i.h2
}

func (i *Image) QuadVertices(sx0, sy0, sx1, sy1 int, a, b, c, d, tx, ty float32, cr, cg, cb, ca float32) []float32 {
	w, h := i.internalSize()
	return graphics.QuadVertices(w, h, sx0, sy0, sx1, sy1, a, b, c, d, tx, ty, cr, cg, cb, ca)
}

func (i *Image) PutVertex(vs []float32, dx, dy, sx, sy float32, bx0, by0, bx1, by1 float32, cr, cg, cb, ca float32) {
	w, h := i.internalSize()
	vs[0] = dx
	vs[1] = dy
	vs[2] = sx / float32(w)
	vs[3] = sy / float32(h)
	vs[4] = bx0 / float32(w)
	vs[5] = by0 / float32(h)
	vs[6] = bx1/float32(w) - 1.0/float32(w)/graphics.TexelAdjustmentFactor
	vs[7] = by1/float32(h) - 1.0/float32(h)/graphics.TexelAdjustmentFactor
	vs[8] = cr
	vs[9] = cg
	vs[10] = cb
	vs[11] = ca
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

func (i *Image) CopyPixels(src *Image) {
	// TODO: Avoid making other images stale if possible. (#514)
	// For this purpuse, images should remember which part of that is used for DrawImage.
	theImages.makeStaleIfDependingOn(i)

	i.image.CopyPixels(src.image)

	// As pixels should not be obtained here, making the image stale is inevitable.
	i.makeStale()
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

	if pixels == nil {
		pixels = make([]byte, 4*width*height)
	}
	i.image.ReplacePixels(pixels, x, y, width, height)

	if !IsRestoringEnabled() {
		i.makeStale()
		return
	}

	if x == 0 && y == 0 && width == w && height == h {
		if pixels != nil {
			if i.basePixels == nil {
				i.basePixels = &Pixels{
					length: 4 * w * h,
				}
			}
			i.basePixels.CopyFrom(pixels, 0)
		} else {
			// If basePixels is nil, the restored pixels are cleared.
			// See restore() implementation.
			i.basePixels = nil
		}
		i.drawImageHistory = nil
		i.stale = false
		return
	}

	if len(i.drawImageHistory) > 0 {
		panic("restorable: ReplacePixels for a part after DrawImage is forbidden")
	}

	if i.stale {
		return
	}

	idx := 4 * (y*w + x)
	if pixels != nil {
		if i.basePixels == nil {
			i.basePixels = &Pixels{
				length: 4 * w * h,
			}
		}
		for j := 0; j < height; j++ {
			i.basePixels.CopyFrom(pixels[4*j*width:4*(j+1)*width], idx)
			idx += 4 * w
		}
	} else if i.basePixels != nil {
		zeros := make([]byte, 4*width)
		for j := 0; j < height; j++ {
			i.basePixels.CopyFrom(zeros, idx)
			idx += 4 * w
		}
	}
}

// DrawImage draws a given image img to the image.
func (i *Image) DrawImage(img *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter, address graphics.Address) {
	if i.priority {
		panic("restorable: DrawImage cannot be called on a priority image")
	}
	if len(vertices) == 0 {
		return
	}
	theImages.makeStaleIfDependingOn(i)

	if img.stale || img.volatile || i.screen || !IsRestoringEnabled() || i.volatile {
		i.makeStale()
	} else {
		i.appendDrawImageHistory(img, vertices, indices, colorm, mode, filter, address)
	}
	i.image.DrawImage(img.image, vertices, indices, colorm, mode, filter, address)
}

// appendDrawImageHistory appends a draw-image history item to the image.
func (i *Image) appendDrawImageHistory(image *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode graphics.CompositeMode, filter graphics.Filter, address graphics.Address) {
	if i.stale || i.volatile || i.screen {
		return
	}
	// TODO: Would it be possible to merge draw image history items?
	const maxDrawImageHistoryNum = 1024
	if len(i.drawImageHistory)+1 > maxDrawImageHistoryNum {
		i.makeStale()
		return
	}
	// All images must be resolved and not stale each after frame.
	// So we don't have to care if image is stale or not here.
	item := &drawImageHistoryItem{
		image:    image,
		vertices: vertices,
		indices:  indices,
		colorm:   colorm,
		mode:     mode,
		filter:   filter,
		address:  address,
	}
	i.drawImageHistory = append(i.drawImageHistory, item)
}

func (i *Image) readPixelsFromGPUIfNeeded() {
	if i.basePixels == nil || len(i.drawImageHistory) > 0 || i.stale {
		graphicscommand.FlushCommands()
		i.readPixelsFromGPU()
		i.drawImageHistory = nil
		i.stale = false
	}
}

// At returns a color value at (x, y).
//
// Note that this must not be called until context is available.
func (i *Image) At(x, y int) (byte, byte, byte, byte) {
	w, h := i.image.Size()
	if x < 0 || y < 0 || w <= x || h <= y {
		return 0, 0, 0, 0
	}

	i.readPixelsFromGPUIfNeeded()

	// Even after readPixelsFromGPU, basePixels might be nil when OpenGL error happens.
	if i.basePixels == nil {
		return 0, 0, 0, 0
	}

	idx := 4*x + 4*y*w
	return i.basePixels.At(idx), i.basePixels.At(idx + 1), i.basePixels.At(idx + 2), i.basePixels.At(idx + 3)
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
func (i *Image) readPixelsFromGPU() {
	pix := i.image.Pixels()
	i.basePixels = &Pixels{
		pixels: pix,
		length: len(pix),
	}
	i.drawImageHistory = nil
	i.stale = false
}

// resolveStale resolves the image's 'stale' state.
func (i *Image) resolveStale() {
	if !IsRestoringEnabled() {
		return
	}

	if i.volatile {
		return
	}
	if i.screen {
		return
	}
	if !i.stale {
		return
	}
	i.readPixelsFromGPU()
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

// Restore restores *graphicscommand.Image from the pixels using its state.
func (i *Image) restore() error {
	w, h := i.image.Size()
	if i.screen {
		// The screen image should also be recreated because framebuffer might
		// be changed.
		i.image = graphicscommand.NewScreenFramebufferImage(w, h)
		i.basePixels = nil
		i.drawImageHistory = nil
		i.stale = false
		return nil
	}
	if i.volatile {
		i.image = graphicscommand.NewImage(w, h)
		i.clear()
		return nil
	}
	if i.stale {
		// TODO: panic here?
		return errors.New("restorable: pixels must not be stale when restoring")
	}

	gimg := graphicscommand.NewImage(w, h)
	if i.basePixels != nil {
		gimg.ReplacePixels(i.basePixels.Slice(), 0, 0, w, h)
	} else {
		// Clear the image explicitly.
		pix := make([]uint8, w*h*4)
		gimg.ReplacePixels(pix, 0, 0, w, h)
	}
	for _, c := range i.drawImageHistory {
		if c.image.hasDependency() {
			panic("restorable: all dependencies must be already resolved but not")
		}
		gimg.DrawImage(c.image.image, c.vertices, c.indices, c.colorm, c.mode, c.filter, c.address)
	}
	i.image = gimg

	pix := gimg.Pixels()
	i.basePixels = &Pixels{
		pixels: pix,
		length: len(pix),
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
	graphicscommand.FlushCommands()
	if !IsRestoringEnabled() {
		return false, nil
	}

	return i.image.IsInvalidated(), nil
}
