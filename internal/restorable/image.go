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
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
)

type Pixels struct {
	pixels []byte

	length int

	// color is used only when pixels == nil
	color color.RGBA
}

func (p *Pixels) CopyFrom(pix []byte, from int) {
	if p.pixels == nil {
		p.pixels = make([]byte, p.length)
	}
	copy(p.pixels[from:from+len(pix)], pix)
}

func (p *Pixels) At(i int) byte {
	if i < 0 || p.length <= i {
		panic(fmt.Sprintf("restorable: index out of range: %d for length: %d", i, p.length))
	}
	if p.pixels != nil {
		return p.pixels[i]
	}
	switch i % 4 {
	case 0:
		return p.color.R
	case 1:
		return p.color.G
	case 2:
		return p.color.B
	case 3:
		return p.color.A
	default:
		panic("not reached")
	}
}

// drawTrianglesHistoryItem is an item for history of draw-image commands.
type drawTrianglesHistoryItem struct {
	image    *Image
	vertices []float32
	indices  []uint16
	colorm   *affine.ColorM
	mode     driver.CompositeMode
	filter   driver.Filter
	address  driver.Address
}

// Image represents an image that can be restored when GL context is lost.
type Image struct {
	image *graphicscommand.Image

	basePixels *Pixels

	// drawTrianglesHistory is a set of draw-image commands.
	// TODO: This should be merged with the similar command queue in package graphics (#433).
	drawTrianglesHistory []*drawTrianglesHistoryItem

	// stale indicates whether the image needs to be synced with GPU as soon as possible.
	stale bool

	// volatile indicates whether the image is cleared whenever a frame starts.
	volatile bool

	// screen indicates whether the image is used as an actual screen.
	screen bool

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

	// As emptyImage is the source at fillImage, initialize this with ReplacePixels, not fillImage.
	// This operation is also important when restoring emptyImage.
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
	i.clearForInitialization()
	theImages.add(i)
	return i
}

// Extend extends the image by the given size.
// Extend creates a new image with the given size and copies the pixels of the given source image.
// Extend disposes itself after its call.
//
// If the given size (width and height) is smaller than the source image, ExtendImage panics.
//
// The image must be ReplacePixels-only image. Extend panics when Fill or DrawTriangles are applied on the image.
//
// Extend panics when the image is stale.
func (i *Image) Extend(width, height int) *Image {
	w, h := i.Size()
	if w > width || h > height {
		panic(fmt.Sprintf("restorable: the original size (%d, %d) cannot be extended to (%d, %d)", w, h, width, height))
	}

	if i.stale {
		panic("restorable: Extend at a stale image is forbidden")
	}

	if len(i.drawTrianglesHistory) > 0 {
		panic("restorable: Extend after DrawTriangles is forbidden")
	}

	if i.basePixels != nil && i.basePixels.color.A > 0 {
		panic("restorable: Extend after Fill is forbidden")
	}

	newImg := NewImage(width, height)

	// Do not use DrawTriangles here. ReplacePixels will be called on a part of newImg later, and it looked like
	// ReplacePixels on a part of image deletes other region that are rendered by DrawTriangles (#593, #758).
	newImg.image.CopyPixels(i.image)

	// Copy basePixels.
	newImg.basePixels = &Pixels{
		pixels: make([]byte, 4*width*height),
		length: 4 * width * height,
	}
	pix := i.basePixels.pixels
	idx := 0
	for j := 0; j < h; j++ {
		newImg.basePixels.CopyFrom(pix[4*j*w:4*(j+1)*w], idx)
		idx += 4 * width
	}

	i.Dispose()

	return newImg
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
	i.clearForInitialization()
	theImages.add(i)
	return i
}

func (i *Image) Fill(r, g, b, a byte) {
	theImages.makeStaleIfDependingOn(i)
	i.fill(r, g, b, a)
}

// clearForInitialization clears the underlying image for initialization.
func (i *Image) clearForInitialization() {
	// As this is for initialization, drawing history doesn't have to be adjusted.
	i.fill(0, 0, 0, 0)
}

// fillImage fills a graphicscommand.Image with the specified color.
// This does nothing to do with a restorable.Image's rendering state.
func fillImage(img *graphicscommand.Image, r, g, b, a byte) {
	if img == emptyImage.image {
		panic("restorable: fillImage cannot be called on emptyImage")
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

	// There are not 'drawTrianglesHistoryItem's for this image and emptyImage.
	// As emptyImage is a priority image, this is restored before other regular images are restored.

	// The rendering target size needs to be its 'internal' size instead of the exposed size to avoid glitches on
	// mobile platforms (See the change 1e1f309a).
	dw, dh := img.InternalSize()
	sw, sh := emptyImage.Size()
	vs := make([]float32, 4*graphics.VertexFloatNum)
	graphics.PutQuadVertices(vs, emptyImage, 0, 0, sw, sh,
		float32(dw)/float32(sw), 0, 0, float32(dh)/float32(sh), 0, 0,
		rf, gf, bf, af)
	is := graphics.QuadIndices()
	c := driver.CompositeModeCopy
	if a == 0 {
		// The first DrawTriangles must be clear mode for initialization.
		// TODO: Can the graphicscommand package hide this knowledge?
		c = driver.CompositeModeClear
	}
	img.DrawTriangles(emptyImage.image, vs, is, nil, c, driver.FilterNearest, driver.AddressClampToZero)
}

func (i *Image) fill(r, g, b, a byte) {
	if i.priority {
		panic("restorable: clear cannot be called on a priority image")
	}

	fillImage(i.image, r, g, b, a)

	w, h := i.Size()
	i.basePixels = &Pixels{
		color:  color.RGBA{r, g, b, a},
		length: 4 * w * h,
	}
	i.drawTrianglesHistory = nil
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
	return i.image.InternalSize()
}

func (i *Image) PutVertex(vs []float32, dx, dy, sx, sy float32, bx0, by0, bx1, by1 float32, cr, cg, cb, ca float32) {
	// Specifying a range explicitly here is redundant but this helps optimization
	// to eliminate boundary checks.
	//
	// VertexFloatNum is better than 12 in terms of code maintenanceability, but in GopherJS, optimization
	// might not work.
	vs = vs[0:12]

	w, h := i.internalSize()
	vs[0] = dx
	vs[1] = dy
	vs[2] = sx / float32(w)
	vs[3] = sy / float32(h)
	vs[4] = bx0 / float32(w)
	vs[5] = by0 / float32(h)
	vs[6] = bx1 / float32(w)
	vs[7] = by1 / float32(h)
	vs[8] = cr
	vs[9] = cg
	vs[10] = cb
	vs[11] = ca
}

// makeStale makes the image stale.
func (i *Image) makeStale() {
	i.basePixels = nil
	i.drawTrianglesHistory = nil
	i.stale = true

	// Don't have to call makeStale recursively here.
	// Restoring is done after topological sorting is done.
	// If an image depends on another stale image, this means that
	// the former image can be restored from the latest state of the latter image.
}

// ClearPixels clears the specified region by ReplacePixels.
func (i *Image) ClearPixels(x, y, width, height int) {
	// TODO: Allocating bytes for all pixels are wasteful. Allocate memory only for required regions (#897).
	i.ReplacePixels(make([]byte, 4*width*height), x, y, width, height)
}

// ReplacePixels replaces the image pixels with the given pixels slice.
//
// ReplacePixels for a part is forbidden if the image is rendered with DrawTriangles or Fill.
func (i *Image) ReplacePixels(pixels []byte, x, y, width, height int) {
	if pixels == nil {
		panic("restorable: pixels must not be nil")
	}

	w, h := i.image.Size()
	if width <= 0 || height <= 0 {
		panic("restorable: width/height must be positive")
	}
	if x < 0 || y < 0 || w <= x || h <= y || x+width <= 0 || y+height <= 0 || w < x+width || h < y+height {
		panic(fmt.Sprintf("restorable: out of range x: %d, y: %d, width: %d, height: %d", x, y, width, height))
	}

	// TODO: Avoid making other images stale if possible. (#514)
	// For this purpuse, images should remember which part of that is used for DrawTriangles.
	theImages.makeStaleIfDependingOn(i)

	i.image.ReplacePixels(pixels, x, y, width, height)

	if x == 0 && y == 0 && width == w && height == h {
		if i.basePixels == nil {
			i.basePixels = &Pixels{
				length: 4 * w * h,
			}
		}
		i.basePixels.CopyFrom(pixels, 0)
		i.basePixels.color = color.RGBA{}
		i.drawTrianglesHistory = nil
		i.stale = false
		return
	}

	if len(i.drawTrianglesHistory) > 0 {
		panic("restorable: ReplacePixels for a part after DrawTriangles is forbidden")
	}

	if i.basePixels != nil && i.basePixels.color.A > 0 {
		panic("restorable: ReplacePixels for a part after Fill is forbidden")
	}

	if i.stale {
		// TODO: panic here?
		return
	}

	idx := 4 * (y*w + x)
	if i.basePixels == nil {
		i.basePixels = &Pixels{
			length: 4 * w * h,
		}
	}
	for j := 0; j < height; j++ {
		i.basePixels.CopyFrom(pixels[4*j*width:4*(j+1)*width], idx)
		idx += 4 * w
	}
}

// DrawTriangles draws a given image img to the image.
func (i *Image) DrawTriangles(img *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	if i.priority {
		panic("restorable: DrawTriangles cannot be called on a priority image")
	}
	if len(vertices) == 0 {
		return
	}
	theImages.makeStaleIfDependingOn(i)

	if img.stale || img.volatile || i.screen || !needsRestoring() || i.volatile {
		i.makeStale()
	} else {
		i.appendDrawTrianglesHistory(img, vertices, indices, colorm, mode, filter, address)
	}
	i.image.DrawTriangles(img.image, vertices, indices, colorm, mode, filter, address)
}

// appendDrawTrianglesHistory appends a draw-image history item to the image.
func (i *Image) appendDrawTrianglesHistory(image *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	if i.stale || i.volatile || i.screen {
		return
	}
	// TODO: Would it be possible to merge draw image history items?
	const maxDrawTrianglesHistoryNum = 1024
	if len(i.drawTrianglesHistory)+1 > maxDrawTrianglesHistoryNum {
		i.makeStale()
		return
	}
	// All images must be resolved and not stale each after frame.
	// So we don't have to care if image is stale or not here.
	item := &drawTrianglesHistoryItem{
		image:    image,
		vertices: vertices,
		indices:  indices,
		colorm:   colorm,
		mode:     mode,
		filter:   filter,
		address:  address,
	}
	i.drawTrianglesHistory = append(i.drawTrianglesHistory, item)
}

func (i *Image) readPixelsFromGPUIfNeeded() {
	if i.basePixels == nil || len(i.drawTrianglesHistory) > 0 || i.stale {
		graphicscommand.FlushCommands()
		i.readPixelsFromGPU()
		i.drawTrianglesHistory = nil
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
	i.drawTrianglesHistory = nil
	i.stale = false
}

// resolveStale resolves the image's 'stale' state.
func (i *Image) resolveStale() {
	if !needsRestoring() {
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
	for _, c := range i.drawTrianglesHistory {
		if c.image == target {
			return true
		}
	}
	return false
}

// dependingImages returns all images that is depended by the image.
func (i *Image) dependingImages() map[*Image]struct{} {
	r := map[*Image]struct{}{}
	for _, c := range i.drawTrianglesHistory {
		r[c.image] = struct{}{}
	}
	return r
}

// hasDependency returns a boolean value indicating whether the image depends on another image.
func (i *Image) hasDependency() bool {
	if i.stale {
		return false
	}
	return len(i.drawTrianglesHistory) > 0
}

// Restore restores *graphicscommand.Image from the pixels using its state.
func (i *Image) restore() error {
	w, h := i.image.Size()
	if i.screen {
		// The screen image should also be recreated because framebuffer might
		// be changed.
		i.image = graphicscommand.NewScreenFramebufferImage(w, h)
		i.basePixels = nil
		i.drawTrianglesHistory = nil
		i.stale = false
		return nil
	}
	if i.volatile {
		i.image = graphicscommand.NewImage(w, h)
		i.clearForInitialization()
		return nil
	}
	if i.stale {
		// TODO: panic here?
		return errors.New("restorable: pixels must not be stale when restoring")
	}

	gimg := graphicscommand.NewImage(w, h)
	if i.basePixels != nil {
		if i.basePixels.pixels != nil {
			// If ReplacePixels is the first command, the image doesn't have be cleared.
			gimg.ReplacePixels(i.basePixels.pixels, 0, 0, w, h)
		} else {
			// Clear the image explicitly.
			fillImage(gimg, 0, 0, 0, 0)
			r := i.basePixels.color.R
			g := i.basePixels.color.G
			b := i.basePixels.color.B
			a := i.basePixels.color.A
			if a > 0 {
				fillImage(gimg, r, g, b, a)
			}
		}
	} else {
		// Clear the image explicitly.
		fillImage(gimg, 0, 0, 0, 0)
	}
	for _, c := range i.drawTrianglesHistory {
		if c.image.hasDependency() {
			panic("restorable: all dependencies must be already resolved but not")
		}
		gimg.DrawTriangles(c.image.image, c.vertices, c.indices, c.colorm, c.mode, c.filter, c.address)
	}

	if len(i.drawTrianglesHistory) > 0 {
		pix := gimg.Pixels()
		i.basePixels = &Pixels{
			pixels: pix,
			length: len(pix),
		}
	}

	i.image = gimg
	i.drawTrianglesHistory = nil
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
	i.drawTrianglesHistory = nil
	i.stale = false
}

// isInvalidated returns a boolean value indicating whether the image is invalidated.
//
// If an image is invalidated, GL context is lost and all the images should be restored asap.
func (i *Image) isInvalidated() bool {
	// FlushCommands is required because c.offscreen.impl might not have an actual texture.
	graphicscommand.FlushCommands()
	return i.image.IsInvalidated()
}
