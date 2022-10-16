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
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type Pixels struct {
	pixelsRecords *pixelsRecords
}

// Apply applies the Pixels state to the given image especially for restoring.
func (p *Pixels) Apply(img *graphicscommand.Image) {
	// Pixels doesn't clear the image. This is a caller's responsibility.

	if p.pixelsRecords == nil {
		return
	}
	p.pixelsRecords.apply(img)
}

func (p *Pixels) AddOrReplace(pix []byte, x, y, width, height int) {
	if p.pixelsRecords == nil {
		p.pixelsRecords = &pixelsRecords{}
	}
	p.pixelsRecords.addOrReplace(pix, x, y, width, height)
}

func (p *Pixels) Clear(x, y, width, height int) {
	// Note that we don't care whether the region is actually removed or not here. There is an actual case that
	// the region is allocated but nothing is rendered. See TestDisposeImmediately at shareable package.
	if p.pixelsRecords == nil {
		return
	}
	p.pixelsRecords.clear(x, y, width, height)
}

func (p *Pixels) ReadPixels(pixels []byte, x, y, width, height, imageWidth, imageHeight int) {
	if p.pixelsRecords == nil {
		for i := range pixels {
			pixels[i] = 0
		}
		return
	}
	p.pixelsRecords.readPixels(pixels, x, y, width, height, imageWidth, imageHeight)
}

// drawTrianglesHistoryItem is an item for history of draw-image commands.
type drawTrianglesHistoryItem struct {
	images    [graphics.ShaderImageCount]*Image
	offsets   [graphics.ShaderImageCount - 1][2]float32
	vertices  []float32
	indices   []uint16
	colorm    affine.ColorM
	mode      graphicsdriver.CompositeMode
	filter    graphicsdriver.Filter
	address   graphicsdriver.Address
	dstRegion graphicsdriver.Region
	srcRegion graphicsdriver.Region
	shader    *Shader
	uniforms  [][]float32
	evenOdd   bool
}

type ImageType int

const (
	// ImageTypeRegular indicates the image is a regular image.
	ImageTypeRegular ImageType = iota

	// ImageTypeScreen indicates the image is used as an actual screen.
	ImageTypeScreen

	// ImageTypeVolatile indicates the image is cleared whenever a frame starts.
	//
	// Regular non-volatile images need to record drawing history or read its pixels from GPU if necessary so that all
	// the images can be restored automatically from the context lost. However, such recording the drawing history or
	// reading pixels from GPU are expensive operations. Volatile images can skip such oprations, but the image content
	// is cleared every frame instead.
	ImageTypeVolatile
)

// Image represents an image that can be restored when GL context is lost.
type Image struct {
	image *graphicscommand.Image

	width  int
	height int

	basePixels Pixels

	// drawTrianglesHistory is a set of draw-image commands.
	// TODO: This should be merged with the similar command queue in package graphics (#433).
	drawTrianglesHistory []*drawTrianglesHistoryItem

	// stale indicates whether the image needs to be synced with GPU as soon as possible.
	stale bool

	imageType ImageType

	// priority indicates whether the image is restored in high priority when context-lost happens.
	priority bool
}

var emptyImage *Image

func ensureEmptyImage() *Image {
	if emptyImage != nil {
		return emptyImage
	}

	// Initialize the empty image lazily. Some functions like needsRestoring might not work at the initial phase.

	// w and h are the empty image's size. They indicate the 1x1 image with 1px padding around.
	const w, h = 3, 3
	emptyImage = &Image{
		image:    graphicscommand.NewImage(w, h, false),
		width:    w,
		height:   h,
		priority: true,
	}
	pix := make([]byte, 4*w*h)
	for i := range pix {
		pix[i] = 0xff
	}

	// As emptyImage is the source at clearImage, initialize this with WritePixels, not clearImage.
	// This operation is also important when restoring emptyImage.
	emptyImage.WritePixels(pix, 0, 0, w, h)
	theImages.add(emptyImage)
	return emptyImage
}

// NewImage creates an empty image with the given size.
//
// The returned image is cleared.
//
// Note that Dispose is not called automatically.
func NewImage(width, height int, imageType ImageType) *Image {
	if !graphicsDriverInitialized {
		panic("restorable: graphics driver must be ready at NewImage but not")
	}

	i := &Image{
		image:     graphicscommand.NewImage(width, height, imageType == ImageTypeScreen),
		width:     width,
		height:    height,
		imageType: imageType,
	}
	clearImage(i.image)
	theImages.add(i)
	return i
}

// Extend extends the image by the given size.
// Extend creates a new image with the given size and copies the pixels of the given source image.
// Extend disposes itself after its call.
//
// If the given size (width and height) is smaller than the source image, ExtendImage panics.
//
// The image must be WritePixels-only image. Extend panics when Fill or DrawTriangles are applied on the image.
//
// Extend panics when the image is stale.
func (i *Image) Extend(width, height int) *Image {
	if i.width > width || i.height > height {
		panic(fmt.Sprintf("restorable: the original size (%d, %d) cannot be extended to (%d, %d)", i.width, i.height, width, height))
	}

	newImg := NewImage(width, height, i.imageType)

	// Use DrawTriangles instead of WritePixels because the image i might be stale and not have its pixels
	// information.
	srcs := [graphics.ShaderImageCount]*Image{i}
	var offsets [graphics.ShaderImageCount - 1][2]float32
	sw, sh := i.image.InternalSize()
	vs := quadVertices(i, 0, 0, float32(sw), float32(sh), 0, 0, float32(sw), float32(sh), 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  float32(sw),
		Height: float32(sh),
	}
	newImg.DrawTriangles(srcs, offsets, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, nil, nil, false)

	// Overwrite the history as if the image newImg is created only by WritePixels. Now drawTrianglesHistory
	// and basePixels cannot be mixed.
	newImg.clearDrawTrianglesHistory()
	newImg.basePixels = i.basePixels
	newImg.stale = i.stale

	i.Dispose()

	return newImg
}

// quadVertices returns vertices to render a quad. These values are passed to graphicscommand.Image.
func quadVertices(src *Image, dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1, cr, cg, cb, ca float32) []float32 {
	sw, sh := src.InternalSize()
	swf, shf := float32(sw), float32(sh)
	return []float32{
		dx0, dy0, sx0 / swf, sy0 / shf, cr, cg, cb, ca,
		dx1, dy0, sx1 / swf, sy0 / shf, cr, cg, cb, ca,
		dx0, dy1, sx0 / swf, sy1 / shf, cr, cg, cb, ca,
		dx1, dy1, sx1 / swf, sy1 / shf, cr, cg, cb, ca,
	}
}

func clearImage(i *graphicscommand.Image) {
	emptyImage := ensureEmptyImage()

	if i == emptyImage.image {
		panic("restorable: fillImage cannot be called on emptyImage")
	}

	// This needs to use 'InternalSize' to render the whole region, or edges are unexpectedly cleared on some
	// devices.
	dw, dh := i.InternalSize()
	sw, sh := emptyImage.width, emptyImage.height
	vs := quadVertices(emptyImage, 0, 0, float32(dw), float32(dh), 1, 1, float32(sw-1), float32(sh-1), 0, 0, 0, 0)
	is := graphics.QuadIndices()
	srcs := [graphics.ShaderImageCount]*graphicscommand.Image{emptyImage.image}
	var offsets [graphics.ShaderImageCount - 1][2]float32
	dstRegion := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  float32(dw),
		Height: float32(dh),
	}
	i.DrawTriangles(srcs, offsets, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeClear, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dstRegion, graphicsdriver.Region{}, nil, nil, false)
}

// BasePixelsForTesting returns the image's basePixels for testing.
func (i *Image) BasePixelsForTesting() *Pixels {
	return &i.basePixels
}

// makeStale makes the image stale.
func (i *Image) makeStale() {
	i.basePixels = Pixels{}
	i.clearDrawTrianglesHistory()
	i.stale = true

	// Don't have to call makeStale recursively here.
	// Restoring is done after topological sorting is done.
	// If an image depends on another stale image, this means that
	// the former image can be restored from the latest state of the latter image.
}

// ClearPixels clears the specified region by WritePixels.
func (i *Image) ClearPixels(x, y, width, height int) {
	i.WritePixels(nil, x, y, width, height)
}

func (i *Image) needsRestoring() bool {
	return i.imageType == ImageTypeRegular
}

// WritePixels replaces the image pixels with the given pixels slice.
//
// The specified region must not be overlapped with other regions by WritePixels.
func (i *Image) WritePixels(pixels []byte, x, y, width, height int) {
	if width <= 0 || height <= 0 {
		panic("restorable: width/height must be positive")
	}
	w, h := i.width, i.height
	if x < 0 || y < 0 || w <= x || h <= y || x+width <= 0 || y+height <= 0 || w < x+width || h < y+height {
		panic(fmt.Sprintf("restorable: out of range x: %d, y: %d, width: %d, height: %d", x, y, width, height))
	}

	// TODO: Avoid making other images stale if possible. (#514)
	// For this purpuse, images should remember which part of that is used for DrawTriangles.
	theImages.makeStaleIfDependingOn(i)

	if pixels != nil {
		i.image.WritePixels(pixels, x, y, width, height)
	} else {
		// TODO: When pixels == nil, we don't have to care the pixel state there. In such cases, the image
		// accepts only WritePixels and not Fill or DrawTriangles.
		// TODO: Separate Image struct into two: images for WritePixels-only, and the others.
		i.image.WritePixels(make([]byte, 4*width*height), x, y, width, height)
	}

	if !needsRestoring() || !i.needsRestoring() {
		i.makeStale()
		return
	}

	if x == 0 && y == 0 && width == w && height == h {
		if pixels != nil {
			// pixels can point to a shared region.
			// This function is responsible to copy this.
			copiedPixels := make([]byte, len(pixels))
			copy(copiedPixels, pixels)
			i.basePixels.AddOrReplace(copiedPixels, 0, 0, w, h)
		} else {
			i.basePixels.Clear(0, 0, w, h)
		}
		i.clearDrawTrianglesHistory()
		i.stale = false
		return
	}

	// drawTrianglesHistory and basePixels cannot be mixed.
	if len(i.drawTrianglesHistory) > 0 {
		i.makeStale()
		return
	}

	if i.stale {
		// TODO: panic here?
		return
	}

	if pixels != nil {
		// pixels can point to a shared region.
		// This function is responsible to copy this.
		copiedPixels := make([]byte, len(pixels))
		copy(copiedPixels, pixels)
		i.basePixels.AddOrReplace(copiedPixels, x, y, width, height)
	} else {
		i.basePixels.Clear(x, y, width, height)
	}
}

// DrawTriangles draws triangles with the given image.
//
// The vertex floats are:
//
//	0: Destination X in pixels
//	1: Destination Y in pixels
//	2: Source X in texels
//	3: Source Y in texels
//	4: Color R [0.0-1.0]
//	5: Color G
//	6: Color B
//	7: Color Y
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageCount]*Image, offsets [graphics.ShaderImageCount - 1][2]float32, vertices []float32, indices []uint16, colorm affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, shader *Shader, uniforms [][]float32, evenOdd bool) {
	if i.priority {
		panic("restorable: DrawTriangles cannot be called on a priority image")
	}
	if len(vertices) == 0 {
		return
	}
	theImages.makeStaleIfDependingOn(i)

	// TODO: Add tests to confirm this logic.
	var srcstale bool
	for _, src := range srcs {
		if src == nil {
			continue
		}
		if src.stale || src.imageType == ImageTypeVolatile {
			srcstale = true
			break
		}
	}

	if srcstale || !needsRestoring() || !i.needsRestoring() {
		i.makeStale()
	} else {
		i.appendDrawTrianglesHistory(srcs, offsets, vertices, indices, colorm, mode, filter, address, dstRegion, srcRegion, shader, uniforms, evenOdd)
	}

	var s *graphicscommand.Shader
	var imgs [graphics.ShaderImageCount]*graphicscommand.Image
	if shader == nil {
		// Fast path for rendering without a shader (#1355).
		imgs[0] = srcs[0].image
	} else {
		for i, src := range srcs {
			if src == nil {
				continue
			}
			imgs[i] = src.image
		}
		s = shader.shader
	}
	i.image.DrawTriangles(imgs, offsets, vertices, indices, colorm, mode, filter, address, dstRegion, srcRegion, s, uniforms, evenOdd)
}

// appendDrawTrianglesHistory appends a draw-image history item to the image.
func (i *Image) appendDrawTrianglesHistory(srcs [graphics.ShaderImageCount]*Image, offsets [graphics.ShaderImageCount - 1][2]float32, vertices []float32, indices []uint16, colorm affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, shader *Shader, uniforms [][]float32, evenOdd bool) {
	if i.stale || !i.needsRestoring() {
		return
	}

	// TODO: Would it be possible to merge draw image history items?
	const maxDrawTrianglesHistoryCount = 1024
	if len(i.drawTrianglesHistory)+1 > maxDrawTrianglesHistoryCount {
		i.makeStale()
		return
	}
	// All images must be resolved and not stale each after frame.
	// So we don't have to care if image is stale or not here.

	vs := make([]float32, len(vertices))
	copy(vs, vertices)

	is := make([]uint16, len(indices))
	copy(is, indices)

	item := &drawTrianglesHistoryItem{
		images:    srcs,
		offsets:   offsets,
		vertices:  vs,
		indices:   is,
		colorm:    colorm,
		mode:      mode,
		filter:    filter,
		address:   address,
		dstRegion: dstRegion,
		srcRegion: srcRegion,
		shader:    shader,
		uniforms:  uniforms,
		evenOdd:   evenOdd,
	}
	i.drawTrianglesHistory = append(i.drawTrianglesHistory, item)
}

func (i *Image) readPixelsFromGPUIfNeeded(graphicsDriver graphicsdriver.Graphics) error {
	if len(i.drawTrianglesHistory) > 0 || i.stale {
		if err := graphicscommand.FlushCommands(graphicsDriver, false); err != nil {
			return err
		}
		if err := i.readPixelsFromGPU(graphicsDriver); err != nil {
			return err
		}
	}
	return nil
}

func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, x, y, width, height int) error {
	if err := i.readPixelsFromGPUIfNeeded(graphicsDriver); err != nil {
		return err
	}
	if got, want := len(pixels), 4*width*height; got != want {
		return fmt.Errorf("restorable: len(pixels) must be %d but %d at ReadPixels", want, got)
	}
	i.basePixels.ReadPixels(pixels, x, y, width, height, i.width, i.height)
	return nil
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

// makeStaleIfDependingOnShader makes the image stale if the image depends on shader.
func (i *Image) makeStaleIfDependingOnShader(shader *Shader) {
	if i.stale {
		return
	}
	if i.dependsOnShader(shader) {
		i.makeStale()
	}
}

// readPixelsFromGPU reads the pixels from GPU and resolves the image's 'stale' state.
func (i *Image) readPixelsFromGPU(graphicsDriver graphicsdriver.Graphics) error {
	pix := make([]byte, 4*i.width*i.height)
	if err := i.image.ReadPixels(graphicsDriver, pix); err != nil {
		return err
	}
	i.basePixels = Pixels{}
	i.basePixels.AddOrReplace(pix, 0, 0, i.width, i.height)
	i.clearDrawTrianglesHistory()
	i.stale = false
	return nil
}

// resolveStale resolves the image's 'stale' state.
func (i *Image) resolveStale(graphicsDriver graphicsdriver.Graphics) error {
	if !needsRestoring() {
		return nil
	}
	if !i.needsRestoring() {
		return nil
	}
	if !i.stale {
		return nil
	}
	return i.readPixelsFromGPU(graphicsDriver)
}

// dependsOn reports whether the image depends on target.
func (i *Image) dependsOn(target *Image) bool {
	for _, c := range i.drawTrianglesHistory {
		for _, img := range c.images {
			if img == nil {
				continue
			}
			if img == target {
				return true
			}
		}
	}
	return false
}

// dependsOnShader reports whether the image depends on shader.
func (i *Image) dependsOnShader(shader *Shader) bool {
	for _, c := range i.drawTrianglesHistory {
		if c.shader == shader {
			return true
		}
	}
	return false
}

// dependingImages returns all images that is depended by the image.
func (i *Image) dependingImages() map[*Image]struct{} {
	r := map[*Image]struct{}{}
	for _, c := range i.drawTrianglesHistory {
		for _, img := range c.images {
			if img == nil {
				continue
			}
			r[img] = struct{}{}
		}
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
func (i *Image) restore(graphicsDriver graphicsdriver.Graphics) error {
	w, h := i.width, i.height
	// Do not dispose the image here. The image should be already disposed.

	switch i.imageType {
	case ImageTypeScreen:
		// The screen image should also be recreated because framebuffer might
		// be changed.
		i.image = graphicscommand.NewImage(w, h, true)
		i.basePixels = Pixels{}
		i.clearDrawTrianglesHistory()
		i.stale = false
		return nil
	case ImageTypeVolatile:
		i.image = graphicscommand.NewImage(w, h, false)
		clearImage(i.image)
		return nil
	}

	if i.stale {
		panic("restorable: pixels must not be stale when restoring")
	}

	gimg := graphicscommand.NewImage(w, h, false)
	// Clear the image explicitly.
	if i != ensureEmptyImage() {
		// As clearImage uses emptyImage, clearImage cannot be called on emptyImage.
		// It is OK to skip this since emptyImage has its entire pixel information.
		clearImage(gimg)
	}
	i.basePixels.Apply(gimg)

	for _, c := range i.drawTrianglesHistory {
		var s *graphicscommand.Shader
		if c.shader != nil {
			s = c.shader.shader
		}

		var imgs [graphics.ShaderImageCount]*graphicscommand.Image
		for i, img := range c.images {
			if img == nil {
				continue
			}
			if img.hasDependency() {
				panic("restorable: all dependencies must be already resolved but not")
			}
			imgs[i] = img.image
		}
		gimg.DrawTriangles(imgs, c.offsets, c.vertices, c.indices, c.colorm, c.mode, c.filter, c.address, c.dstRegion, c.srcRegion, s, c.uniforms, c.evenOdd)
	}

	if len(i.drawTrianglesHistory) > 0 {
		i.basePixels = Pixels{}
		pix := make([]byte, 4*w*h)
		if err := gimg.ReadPixels(graphicsDriver, pix); err != nil {
			return err
		}
		i.basePixels.AddOrReplace(pix, 0, 0, w, h)
	}

	i.image = gimg
	i.clearDrawTrianglesHistory()
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
	i.basePixels = Pixels{}
	i.clearDrawTrianglesHistory()
	i.stale = false
}

// isInvalidated returns a boolean value indicating whether the image is invalidated.
//
// If an image is invalidated, GL context is lost and all the images should be restored asap.
func (i *Image) isInvalidated(graphicsDriver graphicsdriver.Graphics) (bool, error) {
	// FlushCommands is required because c.offscreen.impl might not have an actual texture.
	if err := graphicscommand.FlushCommands(graphicsDriver, false); err != nil {
		return false, err
	}
	return i.image.IsInvalidated(), nil
}

func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) error {
	return i.image.Dump(graphicsDriver, path, blackbg, rect)
}

func (i *Image) clearDrawTrianglesHistory() {
	// Clear the items explicitly, or the references might remain (#1803).
	for idx := range i.drawTrianglesHistory {
		i.drawTrianglesHistory[idx] = nil
	}
	i.drawTrianglesHistory = i.drawTrianglesHistory[:0]
}

func (i *Image) InternalSize() (int, int) {
	return i.image.InternalSize()
}
