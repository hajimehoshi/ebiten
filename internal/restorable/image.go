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
	"sort"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

type Pixels struct {
	pixelsRecords *pixelsRecords
}

// Apply applies the Pixels state to the given image especially for restoration.
func (p *Pixels) Apply(img *graphicscommand.Image) {
	// Pixels doesn't clear the image. This is a caller's responsibility.

	if p.pixelsRecords == nil {
		return
	}
	p.pixelsRecords.apply(img)
}

func (p *Pixels) AddOrReplace(pix *graphics.ManagedBytes, region image.Rectangle) {
	if p.pixelsRecords == nil {
		p.pixelsRecords = &pixelsRecords{}
	}
	p.pixelsRecords.addOrReplace(pix, region)
}

func (p *Pixels) Clear(region image.Rectangle) {
	// Note that we don't care whether the region is actually removed or not here. There is an actual case that
	// the region is allocated but nothing is rendered. See TestDisposeImmediately at shareable package.
	if p.pixelsRecords == nil {
		return
	}
	p.pixelsRecords.clear(region)
}

func (p *Pixels) ReadPixels(pixels []byte, region image.Rectangle, imageWidth, imageHeight int) {
	if p.pixelsRecords == nil {
		for i := range pixels {
			pixels[i] = 0
		}
		return
	}
	p.pixelsRecords.readPixels(pixels, region, imageWidth, imageHeight)
}

func (p *Pixels) AppendRegion(regions []image.Rectangle) []image.Rectangle {
	if p.pixelsRecords == nil {
		return regions
	}
	return p.pixelsRecords.appendRegions(regions)
}

func (p *Pixels) Dispose() {
	if p.pixelsRecords == nil {
		return
	}
	p.pixelsRecords.dispose()
}

// drawTrianglesHistoryItem is an item for history of draw-image commands.
type drawTrianglesHistoryItem struct {
	srcImages  [graphics.ShaderSrcImageCount]*Image
	vertices   []float32
	indices    []uint32
	blend      graphicsdriver.Blend
	dstRegion  image.Rectangle
	srcRegions [graphics.ShaderSrcImageCount]image.Rectangle
	shader     *Shader
	uniforms   []uint32
	fillRule   graphicsdriver.FillRule
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

// Hint is a hint to optimize the info to restore the image.
type Hint int

const (
	// HintNone indicates that there is no hint.
	HintNone Hint = iota

	// HintOverwriteDstRegion indicates that the destination region is overwritten.
	// HintOverwriteDstRegion helps to reduce the size of the draw-image history.
	HintOverwriteDstRegion
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

// NewImage creates an emtpy image with the given size.
//
// The returned image is cleared.
//
// Note that Dispose is not called automatically.
func NewImage(width, height int, imageType ImageType) *Image {
	if !graphicsDriverInitialized {
		panic("restorable: graphics driver must be ready at NewImage but not")
	}

	var attribute string
	if needsRestoration() {
		switch imageType {
		case ImageTypeVolatile:
			attribute = "volatile"
		}
	}
	i := &Image{
		image:     graphicscommand.NewImage(width, height, imageType == ImageTypeScreen, attribute),
		width:     width,
		height:    height,
		imageType: imageType,
	}

	// This needs to use 'InternalSize' to render the whole region, or edges are unexpectedly cleared on some
	// devices.
	iw, ih := i.image.InternalSize()
	clearImage(i.image, image.Rect(0, 0, iw, ih))
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
	srcs := [graphics.ShaderSrcImageCount]*Image{i}
	sw, sh := i.image.InternalSize()
	vs := make([]float32, 4*graphics.VertexFloatCount)
	graphics.QuadVerticesFromDstAndSrc(vs, 0, 0, float32(sw), float32(sh), 0, 0, float32(sw), float32(sh), 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, sw, sh)
	newImg.DrawTriangles(srcs, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{}, NearestFilterShader, nil, graphicsdriver.FillRuleFillAll, HintOverwriteDstRegion)
	i.Dispose()

	return newImg
}

func clearImage(i *graphicscommand.Image, region image.Rectangle) {
	vs := make([]float32, 4*graphics.VertexFloatCount)
	graphics.QuadVerticesFromDstAndSrc(vs, float32(region.Min.X), float32(region.Min.Y), float32(region.Max.X), float32(region.Max.Y), 0, 0, 0, 0, 0, 0, 0, 0)
	is := graphics.QuadIndices()
	i.DrawTriangles([graphics.ShaderSrcImageCount]*graphicscommand.Image{}, vs, is, graphicsdriver.BlendClear, region, [graphics.ShaderSrcImageCount]image.Rectangle{}, clearShader.shader, nil, graphicsdriver.FillRuleFillAll)
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

	if !i.needsRestoration() {
		return
	}

	origSize := len(i.staleRegions)
	i.staleRegions = i.appendRegionsForDrawTriangles(i.staleRegions)
	if !rect.Empty() {
		i.staleRegions = append(i.staleRegions, rect)
	}

	i.clearDrawTrianglesHistory()

	// Clear pixels to save memory.
	for _, r := range i.staleRegions[origSize:] {
		i.basePixels.Clear(r)
	}

	// Don't have to call makeStale recursively here.
	// Restoration is done after topological sorting is done.
	// If an image depends on another stale image, this means that
	// the former image can be restored from the latest state of the latter image.
}

// ClearPixels clears the specified region by WritePixels.
func (i *Image) ClearPixels(region image.Rectangle) {
	i.WritePixels(nil, region)
}

func (i *Image) needsRestoration() bool {
	return i.imageType == ImageTypeRegular
}

// WritePixels replaces the image pixels with the given pixels slice.
//
// The specified region must not be overlapped with other regions by WritePixels.
func (i *Image) WritePixels(pixels *graphics.ManagedBytes, region image.Rectangle) {
	if region.Dx() <= 0 || region.Dy() <= 0 {
		panic("restorable: width/height must be positive")
	}
	w, h := i.width, i.height
	if !region.In(image.Rect(0, 0, w, h)) {
		panic(fmt.Sprintf("restorable: out of range %v", region))
	}

	theImages.makeStaleIfDependingOnAtRegion(i, region)

	if pixels != nil {
		i.image.WritePixels(pixels, region)
	} else {
		clearImage(i.image, region)
	}

	if !needsRestoration() || !i.needsRestoration() {
		i.makeStale(region)
		return
	}

	if region.Eq(image.Rect(0, 0, w, h)) {
		if pixels != nil {
			// Clone a ManagedBytes as the package graphicscommand has a different lifetime management.
			i.basePixels.AddOrReplace(pixels.Clone(), region)
		} else {
			i.basePixels.Clear(region)
		}
		i.clearDrawTrianglesHistory()
		i.stale = false
		i.staleRegions = i.staleRegions[:0]
		return
	}

	if i.stale {
		// Even if the image is already stale, call makeStale to extend the stale region.
		i.makeStale(region)
		return
	}

	i.removeDrawTrianglesHistoryItems(region)

	// Records for DrawTriangles cannot come before records for WritePixels.
	if len(i.drawTrianglesHistory) > 0 {
		i.makeStale(region)
		return
	}

	if pixels != nil {
		// Clone a ManagedBytes as the package graphicscommand has a different lifetime management.
		i.basePixels.AddOrReplace(pixels.Clone(), region)
	} else {
		i.basePixels.Clear(region)
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
func (i *Image) DrawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule, hint Hint) {
	if len(vertices) == 0 {
		return
	}

	// makeStaleIfDependingOnAtRegion is not available here.
	// This might create cyclic dependency.
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
	if srcstale || !needsRestoration() || !i.needsRestoration() {
		i.makeStale(dstRegion)
	} else if i.stale {
		var overwrite bool
		if hint == HintOverwriteDstRegion {
			overwrite = i.areStaleRegionsIncludedIn(dstRegion)
		}
		if overwrite {
			i.basePixels.Clear(dstRegion)
			i.clearDrawTrianglesHistory()
			i.stale = false
			i.staleRegions = i.staleRegions[:0]
		} else {
			// Even if the image is already stale, call makeStale to extend the stale region.
			i.makeStale(dstRegion)
		}
	}

	if !i.stale {
		i.appendDrawTrianglesHistory(srcs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, fillRule, hint)
	}

	var imgs [graphics.ShaderSrcImageCount]*graphicscommand.Image
	for i, src := range srcs {
		if src == nil {
			continue
		}
		imgs[i] = src.image
	}
	i.image.DrawTriangles(imgs, vertices, indices, blend, dstRegion, srcRegions, shader.shader, uniforms, fillRule)
}

func (i *Image) areStaleRegionsIncludedIn(r image.Rectangle) bool {
	if !i.stale {
		return false
	}
	for _, sr := range i.staleRegions {
		if !sr.In(r) {
			return false
		}
	}
	return true
}

// removeDrawTrianglesHistoryItems removes draw-image history items whose destination regions are in the given region.
// This is useful when the destination region is overwritten soon later.
func (i *Image) removeDrawTrianglesHistoryItems(region image.Rectangle) {
	for idx, c := range i.drawTrianglesHistory {
		if c.dstRegion.In(region) {
			i.drawTrianglesHistory[idx] = nil
		}
	}
	var n int
	for _, c := range i.drawTrianglesHistory {
		if c == nil {
			continue
		}
		i.drawTrianglesHistory[n] = c
		n++
	}
	i.drawTrianglesHistory = i.drawTrianglesHistory[:n]
}

// appendDrawTrianglesHistory appends a draw-image history item to the image.
func (i *Image) appendDrawTrianglesHistory(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule, hint Hint) {
	if i.stale || !i.needsRestoration() {
		panic("restorable: an image must not be stale or need restoration at appendDrawTrianglesHistory")
	}
	if AlwaysReadPixelsFromGPU() {
		panic("restorable: appendDrawTrianglesHistory must not be called when AlwaysReadPixelsFromGPU() returns true")
	}

	// If the command overwrites the destination region, remove the history items that are in the region.
	if hint == HintOverwriteDstRegion {
		i.removeDrawTrianglesHistoryItems(dstRegion)
	}

	// TODO: Would it be possible to merge draw image history items?
	const maxDrawTrianglesHistoryCount = 1024
	if len(i.drawTrianglesHistory)+1 > maxDrawTrianglesHistoryCount {
		i.makeStale(dstRegion)
		return
	}
	// All images must be resolved and not stale each after frame.
	// So we don't have to care if image is stale or not here.

	vs := make([]float32, len(vertices))
	copy(vs, vertices)

	is := make([]uint32, len(indices))
	copy(is, indices)

	us := make([]uint32, len(uniforms))
	copy(us, uniforms)

	item := &drawTrianglesHistoryItem{
		srcImages:  srcs,
		vertices:   vs,
		indices:    is,
		blend:      blend,
		dstRegion:  dstRegion,
		srcRegions: srcRegions,
		shader:     shader,
		uniforms:   us,
		fillRule:   fillRule,
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

func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, region image.Rectangle) error {
	if AlwaysReadPixelsFromGPU() || !i.needsRestoration() {
		if err := i.image.ReadPixels(graphicsDriver, []graphicsdriver.PixelsArgs{
			{
				Pixels: pixels,
				Region: region,
			},
		}); err != nil {
			return err
		}
		return nil
	}

	if err := i.readPixelsFromGPUIfNeeded(graphicsDriver); err != nil {
		return err
	}
	if got, want := len(pixels), 4*region.Dx()*region.Dy(); got != want {
		return fmt.Errorf("restorable: len(pixels) must be %d but %d at ReadPixels", want, got)
	}
	i.basePixels.ReadPixels(pixels, region, i.width, i.height)
	return nil
}

// makeStaleIfDependingOn makes the image stale if the image depends on src.
func (i *Image) makeStaleIfDependingOn(src *Image) {
	if i.stale {
		return
	}
	if i.dependsOn(src) {
		// There is no new region to make stale.
		i.makeStale(image.Rectangle{})
	}
}

// makeStaleIfDependingOnAtRegion makes the image stale if the image depends on src at srcRegion.
func (i *Image) makeStaleIfDependingOnAtRegion(src *Image, srcRegion image.Rectangle) {
	if i.stale {
		return
	}
	if i.dependsOnAtRegion(src, srcRegion) {
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
		rs = i.regionsCache
	}

	// Remove duplications. Is this heavy?
	rs = rs[:removeDuplicatedRegions(rs)]

	args := make([]graphicsdriver.PixelsArgs, 0, len(rs))
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

		args = append(args, graphicsdriver.PixelsArgs{
			Pixels: pix,
			Region: r,
		})
	}

	if err := i.image.ReadPixels(graphicsDriver, args); err != nil {
		return err
	}

	for _, a := range args {
		bs := graphics.NewManagedBytes(len(a.Pixels), func(bs []byte) {
			copy(bs, a.Pixels)
		})
		i.basePixels.AddOrReplace(bs, a.Region)
	}

	i.clearDrawTrianglesHistory()
	i.stale = false
	i.staleRegions = i.staleRegions[:0]
	return nil
}

// resolveStale resolves the image's 'stale' state.
func (i *Image) resolveStale(graphicsDriver graphicsdriver.Graphics) error {
	if !needsRestoration() {
		return nil
	}
	if !i.needsRestoration() {
		return nil
	}
	if !i.stale {
		return nil
	}
	return i.readPixelsFromGPU(graphicsDriver)
}

// dependsOn reports whether the image depends on src.
func (i *Image) dependsOn(src *Image) bool {
	for _, c := range i.drawTrianglesHistory {
		for _, img := range c.srcImages {
			if img != src {
				continue
			}
			return true
		}
	}
	return false
}

// dependsOnAtRegion reports whether the image depends on src at srcRegion.
func (i *Image) dependsOnAtRegion(src *Image, srcRegion image.Rectangle) bool {
	for _, c := range i.drawTrianglesHistory {
		for i, img := range c.srcImages {
			if img != src {
				continue
			}
			if c.srcRegions[i].Overlaps(srcRegion) {
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
		for _, img := range c.srcImages {
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
		i.image = graphicscommand.NewImage(w, h, true, "")
		i.basePixels.Dispose()
		i.basePixels = Pixels{}
		i.clearDrawTrianglesHistory()
		i.stale = false
		i.staleRegions = i.staleRegions[:0]
		return nil
	case ImageTypeVolatile:
		i.image = graphicscommand.NewImage(w, h, false, "volatile")
		iw, ih := i.image.InternalSize()
		clearImage(i.image, image.Rect(0, 0, iw, ih))
		return nil
	}

	if i.stale {
		panic("restorable: pixels must not be stale when restoring")
	}

	gimg := graphicscommand.NewImage(w, h, false, "")
	// Clear the image explicitly.
	iw, ih := gimg.InternalSize()
	clearImage(gimg, image.Rect(0, 0, iw, ih))

	i.basePixels.Apply(gimg)

	for _, c := range i.drawTrianglesHistory {
		var imgs [graphics.ShaderSrcImageCount]*graphicscommand.Image
		for i, img := range c.srcImages {
			if img == nil {
				continue
			}
			if img.hasDependency() {
				panic("restorable: all dependencies must be already resolved but not")
			}
			imgs[i] = img.image
		}
		gimg.DrawTriangles(imgs, c.vertices, c.indices, c.blend, c.dstRegion, c.srcRegions, c.shader.shader, c.uniforms, c.fillRule)
	}

	// In order to clear the draw-triangles history, read pixels from GPU.
	if len(i.drawTrianglesHistory) > 0 {
		i.regionsCache = i.appendRegionsForDrawTriangles(i.regionsCache)
		defer func() {
			i.regionsCache = i.regionsCache[:0]
		}()

		args := make([]graphicsdriver.PixelsArgs, 0, len(i.regionsCache))
		for _, r := range i.regionsCache {
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
			args = append(args, graphicsdriver.PixelsArgs{
				Pixels: pix,
				Region: r,
			})
		}

		if err := gimg.ReadPixels(graphicsDriver, args); err != nil {
			return err
		}

		for _, a := range args {
			bs := graphics.NewManagedBytes(len(a.Pixels), func(bs []byte) {
				copy(bs, a.Pixels)
			})
			i.basePixels.AddOrReplace(bs, a.Region)
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
	i.basePixels.Dispose()
	i.basePixels = Pixels{}
	i.pixelsCache = nil
	i.clearDrawTrianglesHistory()
	i.stale = false
	i.staleRegions = i.staleRegions[:0]
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
	for _, d := range i.drawTrianglesHistory {
		if d.dstRegion.Empty() {
			continue
		}
		regions = append(regions, d.dstRegion)
	}
	return regions
}

// removeDuplicatedRegions removes duplicated regions and returns a shrunk slice.
// If a region covers preceding regions, the covered regions are removed.
func removeDuplicatedRegions(regions []image.Rectangle) int {
	// Sweep and prune algorithm

	sort.Slice(regions, func(i, j int) bool {
		return regions[i].Min.X < regions[j].Min.X
	})

	for i, r := range regions {
		if r.Empty() {
			continue
		}
		for j := i + 1; j < len(regions); j++ {
			rr := regions[j]
			if rr.Empty() {
				continue
			}
			if r.Max.X <= rr.Min.X {
				break
			}
			if rr.In(r) {
				regions[j] = image.Rectangle{}
			} else if r.In(rr) {
				regions[i] = image.Rectangle{}
				break
			}
		}
	}

	var n int
	for _, r := range regions {
		if r.Empty() {
			continue
		}
		regions[n] = r
		n++
	}

	return n
}
