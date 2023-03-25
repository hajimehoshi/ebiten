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
	"math"

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

func (p *Pixels) AppendRegion(regions []image.Rectangle) []image.Rectangle {
	if p.pixelsRecords == nil {
		return regions
	}
	return p.pixelsRecords.appendRegions(regions)
}

// drawTrianglesHistoryItem is an item for history of draw-image commands.
type drawTrianglesHistoryItem struct {
	images    [graphics.ShaderImageCount]*Image
	offsets   [graphics.ShaderImageCount - 1][2]float32
	vertices  []float32
	indices   []uint16
	blend     graphicsdriver.Blend
	dstRegion graphicsdriver.Region
	srcRegion graphicsdriver.Region
	shader    *Shader
	uniforms  []uint32
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
	// reading pixels from GPU are expensive operations. Volatile images can skip such operations, but the image content
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

	// staleRegions indicates the regions to restore.
	// staleRegions is valid only when stale is true.
	// staleRegions is not used when AlwaysReadPixelsFromGPU() returns true.
	staleRegions []image.Rectangle

	// pixelsCache is cached byte slices for pixels.
	// pixelsCache is just a cache to avoid allocations (#2375).
	//
	// A key is the region and a value is a byte slice for the region.
	//
	// It is fine to reuse the same byte slice for the same region for basePixels,
	// as old pixels for the same region will be invalidated at basePixel.AddOrReplace.
	pixelsCache map[image.Rectangle][]byte

	// regionsCache is cached regions.
	// regionsCache is just a cache to avoid allocations (#2375).
	regionsCache []image.Rectangle

	imageType ImageType
}

// NewImage creates a white image with the given size.
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
func (i *Image) Extend(width, height int) *Image {
	if i.width >= width && i.height >= height {
		return i
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
	newImg.DrawTriangles(srcs, offsets, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, NearestFilterShader, nil, false)
	i.Dispose()

	return newImg
}

// quadVertices returns vertices to render a quad. These values are passed to graphicscommand.Image.
func quadVertices(src *Image, dx0, dy0, dx1, dy1, sx0, sy0, sx1, sy1, cr, cg, cb, ca float32) []float32 {
	if src == nil {
		return []float32{
			dx0, dy0, 0, 0, cr, cg, cb, ca,
			dx1, dy0, 0, 0, cr, cg, cb, ca,
			dx0, dy1, 0, 0, cr, cg, cb, ca,
			dx1, dy1, 0, 0, cr, cg, cb, ca,
		}
	}
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
	// This needs to use 'InternalSize' to render the whole region, or edges are unexpectedly cleared on some
	// devices.
	dw, dh := i.InternalSize()
	vs := quadVertices(nil, 0, 0, float32(dw), float32(dh), 0, 0, 0, 0, 0, 0, 0, 0)
	is := graphics.QuadIndices()
	var offsets [graphics.ShaderImageCount - 1][2]float32
	dstRegion := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  float32(dw),
		Height: float32(dh),
	}
	i.DrawTriangles([graphics.ShaderImageCount]*graphicscommand.Image{}, offsets, vs, is, graphicsdriver.BlendClear, dstRegion, graphicsdriver.Region{}, clearShader.shader, nil, false)
}

// BasePixelsForTesting returns the image's basePixels for testing.
func (i *Image) BasePixelsForTesting() *Pixels {
	return &i.basePixels
}

// makeStale makes the image stale.
func (i *Image) makeStale(rect image.Rectangle) {
	i.stale = true

	// If ReadPixels always reads pixels from GPU, staleRegions are never used.
	if AlwaysReadPixelsFromGPU() {
		return
	}

	origNum := len(i.staleRegions)
	i.staleRegions = i.appendRegionsForDrawTriangles(i.staleRegions)
	if !rect.Empty() {
		i.staleRegions = append(i.staleRegions, rect)
	}

	i.clearDrawTrianglesHistory()

	// Clear pixels to save memory.
	for _, r := range i.staleRegions[origNum:] {
		if r.Empty() {
			continue
		}
		i.basePixels.Clear(r.Min.X, r.Min.Y, r.Dx(), r.Dy())
	}

	// Remove duplicated regions to avoid unnecessary reading pixels from GPU.
	n := removeDuplicatedRegions(i.staleRegions)
	i.staleRegions = i.staleRegions[:n]

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
	// For this purpose, images should remember which part of that is used for DrawTriangles.
	theImages.makeStaleIfDependingOn(i)

	if pixels != nil {
		i.image.WritePixels(pixels, x, y, width, height)
	} else {
		// TODO: When pixels == nil, we don't have to care the pixel state there. In such cases, the image
		// accepts only WritePixels and not Fill or DrawTriangles.
		// TODO: Separate Image struct into two: images for WritePixels-only, and the others.
		i.image.WritePixels(make([]byte, 4*width*height), x, y, width, height)
	}

	// Even if the image is already stale, call makeStale to extend the stale region.
	if !needsRestoring() || !i.needsRestoring() || i.stale {
		i.makeStale(image.Rect(x, y, x+width, y+height))
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
		i.staleRegions = i.staleRegions[:0]
		return
	}

	// Records for DrawTriangles cannot come before records for WritePixels.
	if len(i.drawTrianglesHistory) > 0 {
		i.makeStale(image.Rect(x, y, x+width, y+height))
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
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageCount]*Image, offsets [graphics.ShaderImageCount - 1][2]float32, vertices []float32, indices []uint16, blend graphicsdriver.Blend, dstRegion, srcRegion graphicsdriver.Region, shader *Shader, uniforms []uint32, evenOdd bool) {
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

	// Even if the image is already stale, call makeStale to extend the stale region.
	if srcstale || !needsRestoring() || !i.needsRestoring() || i.stale {
		i.makeStale(regionToRectangle(dstRegion))
	} else {
		i.appendDrawTrianglesHistory(srcs, offsets, vertices, indices, blend, dstRegion, srcRegion, shader, uniforms, evenOdd)
	}

	var imgs [graphics.ShaderImageCount]*graphicscommand.Image
	for i, src := range srcs {
		if src == nil {
			continue
		}
		imgs[i] = src.image
	}
	i.image.DrawTriangles(imgs, offsets, vertices, indices, blend, dstRegion, srcRegion, shader.shader, uniforms, evenOdd)
}

// appendDrawTrianglesHistory appends a draw-image history item to the image.
func (i *Image) appendDrawTrianglesHistory(srcs [graphics.ShaderImageCount]*Image, offsets [graphics.ShaderImageCount - 1][2]float32, vertices []float32, indices []uint16, blend graphicsdriver.Blend, dstRegion, srcRegion graphicsdriver.Region, shader *Shader, uniforms []uint32, evenOdd bool) {
	if i.stale || !i.needsRestoring() {
		panic("restorable: an image must not be stale or need restoring at appendDrawTrianglesHistory")
	}
	if AlwaysReadPixelsFromGPU() {
		panic("restorable: appendDrawTrianglesHistory must not be called when AlwaysReadPixelsFromGPU() returns true")
	}

	// TODO: Would it be possible to merge draw image history items?
	const maxDrawTrianglesHistoryCount = 1024
	if len(i.drawTrianglesHistory)+1 > maxDrawTrianglesHistoryCount {
		i.makeStale(regionToRectangle(dstRegion))
		return
	}
	// All images must be resolved and not stale each after frame.
	// So we don't have to care if image is stale or not here.

	vs := make([]float32, len(vertices))
	copy(vs, vertices)

	is := make([]uint16, len(indices))
	copy(is, indices)

	us := make([]uint32, len(uniforms))
	copy(us, uniforms)

	item := &drawTrianglesHistoryItem{
		images:    srcs,
		offsets:   offsets,
		vertices:  vs,
		indices:   is,
		blend:     blend,
		dstRegion: dstRegion,
		srcRegion: srcRegion,
		shader:    shader,
		uniforms:  us,
		evenOdd:   evenOdd,
	}
	i.drawTrianglesHistory = append(i.drawTrianglesHistory, item)
}

func (i *Image) readPixelsFromGPUIfNeeded(graphicsDriver graphicsdriver.Graphics) error {
	if len(i.drawTrianglesHistory) > 0 || i.stale {
		if err := i.readPixelsFromGPU(graphicsDriver); err != nil {
			return err
		}
	}
	return nil
}

func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, x, y, width, height int) error {
	if AlwaysReadPixelsFromGPU() {
		if err := i.image.ReadPixels(graphicsDriver, pixels, x, y, width, height); err != nil {
			return err
		}
		return nil
	}

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
		// There is no new region to make stale.
		i.makeStale(image.Rectangle{})
	}
}

// makeStaleIfDependingOnShader makes the image stale if the image depends on shader.
func (i *Image) makeStaleIfDependingOnShader(shader *Shader) {
	if i.stale {
		return
	}
	if i.dependsOnShader(shader) {
		// There is no new region to make stale.
		i.makeStale(image.Rectangle{})
	}
}

// readPixelsFromGPU reads the pixels from GPU and resolves the image's 'stale' state.
func (i *Image) readPixelsFromGPU(graphicsDriver graphicsdriver.Graphics) error {
	var rs []image.Rectangle
	if i.stale {
		rs = i.staleRegions
	} else {
		i.regionsCache = i.appendRegionsForDrawTriangles(i.regionsCache)
		defer func() {
			i.regionsCache = i.regionsCache[:0]
		}()
		n := removeDuplicatedRegions(i.regionsCache)
		rs = i.regionsCache[:n]
	}

	for _, r := range rs {
		if r.Empty() {
			continue
		}

		if i.pixelsCache == nil {
			i.pixelsCache = map[image.Rectangle][]byte{}
		}

		pix, ok := i.pixelsCache[r]
		if !ok {
			pix = make([]byte, 4*r.Dx()*r.Dy())
			i.pixelsCache[r] = pix
		}
		if err := i.image.ReadPixels(graphicsDriver, pix, r.Min.X, r.Min.Y, r.Dx(), r.Dy()); err != nil {
			return err
		}
		i.basePixels.AddOrReplace(pix, r.Min.X, r.Min.Y, r.Dx(), r.Dy())
	}

	i.clearDrawTrianglesHistory()
	i.stale = false
	i.staleRegions = i.staleRegions[:0]
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

// dependingImages returns all images that is depended on the image.
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
		i.staleRegions = i.staleRegions[:0]
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
	clearImage(gimg)

	i.basePixels.Apply(gimg)

	for _, c := range i.drawTrianglesHistory {
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
		gimg.DrawTriangles(imgs, c.offsets, c.vertices, c.indices, c.blend, c.dstRegion, c.srcRegion, c.shader.shader, c.uniforms, c.evenOdd)
	}

	// In order to clear the draw-triangles history, read pixels from GPU.
	if len(i.drawTrianglesHistory) > 0 {
		i.regionsCache = i.appendRegionsForDrawTriangles(i.regionsCache)
		defer func() {
			i.regionsCache = i.regionsCache[:0]
		}()
		n := removeDuplicatedRegions(i.regionsCache)
		rs := i.regionsCache[:n]

		for _, r := range rs {
			if r.Empty() {
				continue
			}

			if i.pixelsCache == nil {
				i.pixelsCache = map[image.Rectangle][]byte{}
			}

			pix, ok := i.pixelsCache[r]
			if !ok {
				pix = make([]byte, 4*r.Dx()*r.Dy())
				i.pixelsCache[r] = pix
			}
			if err := gimg.ReadPixels(graphicsDriver, pix, r.Min.X, r.Min.Y, r.Dx(), r.Dy()); err != nil {
				return err
			}
			i.basePixels.AddOrReplace(pix, r.Min.X, r.Min.Y, r.Dx(), r.Dy())
		}
	}

	i.image = gimg
	i.clearDrawTrianglesHistory()
	i.stale = false
	i.staleRegions = i.staleRegions[:0]
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
	i.pixelsCache = nil
	i.clearDrawTrianglesHistory()
	i.stale = false
	i.staleRegions = i.staleRegions[:0]
}

// isInvalidated returns a boolean value indicating whether the image is invalidated.
//
// If an image is invalidated, GL context is lost and all the images should be restored asap.
func (i *Image) isInvalidated(graphicsDriver graphicsdriver.Graphics) (bool, error) {
	// IsInvalidated flushes the commands internally.
	return i.image.IsInvalidated(graphicsDriver)
}

func (i *Image) Dump(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool, rect image.Rectangle) (string, error) {
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

func (i *Image) appendRegionsForDrawTriangles(regions []image.Rectangle) []image.Rectangle {
	n := len(regions)
	for _, d := range i.drawTrianglesHistory {
		r := regionToRectangle(d.dstRegion)
		if r.Empty() {
			continue
		}
		regions = append(regions, r)
	}

	nn := removeDuplicatedRegions(regions[n:])
	return regions[:n+nn]
}

func regionToRectangle(region graphicsdriver.Region) image.Rectangle {
	return image.Rect(
		int(math.Floor(float64(region.X))),
		int(math.Floor(float64(region.Y))),
		int(math.Ceil(float64(region.X+region.Width))),
		int(math.Ceil(float64(region.Y+region.Height))))
}

// removeDuplicatedRegions removes duplicated regions and returns the new size of the slice.
// If a region covers other regions, the covered regions are removed.
func removeDuplicatedRegions(regions []image.Rectangle) int {
	for i, r := range regions {
		if r.Empty() {
			continue
		}
		for j, rr := range regions {
			if i == j {
				continue
			}
			if rr.Empty() {
				continue
			}
			if rr.In(r) {
				regions[j] = image.Rectangle{}
			}
		}
	}

	n := 0
	for _, r := range regions {
		if r.Empty() {
			continue
		}
		regions[n] = r
		n++
	}

	return n
}
