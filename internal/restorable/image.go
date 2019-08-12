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

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
)

type Pixels struct {
	rectToPixels *rectToPixels
}

// Apply applies the Pixels state to the given image especially for restoring.
func (p *Pixels) Apply(img *graphicscommand.Image) {
	if p.rectToPixels == nil {
		clearImage(img)
		return
	}

	p.rectToPixels.apply(img)
}

func (p *Pixels) AddOrReplace(pix []byte, x, y, width, height int) {
	if p.rectToPixels == nil {
		p.rectToPixels = &rectToPixels{}
	}
	p.rectToPixels.addOrReplace(pix, x, y, width, height)
}

func (p *Pixels) Remove(x, y, width, height int) {
	// Note that we don't care whether the region is actually removed or not here. There is an actual case that
	// the region is allocated but nothing is rendered. See TestDisposeImmediately at shareable package.
	if p.rectToPixels == nil {
		return
	}
	p.rectToPixels.remove(x, y, width, height)
}

func (p *Pixels) At(i, j int) (byte, byte, byte, byte) {
	if p.rectToPixels != nil {
		if r, g, b, a, ok := p.rectToPixels.at(i, j); ok {
			return r, g, b, a
		}
	}
	return 0, 0, 0, 0
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

	basePixels Pixels

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

	// As emptyImage is the source at clearImage, initialize this with ReplacePixels, not clearImage.
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
	// As this should not affect the information for restoring, this doesn't have to be deferred.

	i := &Image{
		image: graphicscommand.NewImage(width, height),
	}
	i.clear()
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

	newImg := NewImage(width, height)
	i.basePixels.Apply(newImg.image)

	newImg.basePixels = i.basePixels

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
	// As this should not affect the information for restoring, this doesn't have to be deferred.

	i := &Image{
		image:  graphicscommand.NewScreenFramebufferImage(width, height),
		screen: true,
	}
	i.clear()
	theImages.add(i)
	return i
}

func (i *Image) Clear() {
	select {
	case theImages.deferCh <- struct{}{}:
		break
	default:
		theImages.deferUntilBeginFrame(i.Clear)
		return
	}

	defer func() {
		<-theImages.deferCh
	}()

	theImages.makeStaleIfDependingOn(i)
	i.clear()
}

// clearImage clears a graphicscommand.Image.
// This does nothing to do with a restorable.Image's rendering state.
func clearImage(img *graphicscommand.Image) {
	if img == emptyImage.image {
		panic("restorable: clearImage cannot be called on emptyImage")
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
		0, 0, 0, 0)
	is := graphics.QuadIndices()
	// The first DrawTriangles must be clear mode for initialization.
	// TODO: Can the graphicscommand package hide this knowledge?
	img.DrawTriangles(emptyImage.image, vs, is, nil, driver.CompositeModeClear, driver.FilterNearest, driver.AddressClampToZero)
}

func (i *Image) clear() {
	if i.priority {
		panic("restorable: clear cannot be called on a priority image")
	}

	clearImage(i.image)
	i.ResetRestoringState()
}

// ResetRestoringState resets all the information for restoring.
// ResetRestoringState doen't affect the underlying image.
//
// After ResetRestoringState, the image is assumed to be cleared.
func (i *Image) ResetRestoringState() {
	i.basePixels = Pixels{}
	i.drawTrianglesHistory = nil
	i.stale = false
}

func (i *Image) IsVolatile() bool {
	return i.volatile
}

// BasePixelsForTesting returns the image's basePixels for testing.
func (i *Image) BasePixelsForTesting() *Pixels {
	return &i.basePixels
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
	i.basePixels = Pixels{}
	i.drawTrianglesHistory = nil
	i.stale = true

	// Don't have to call makeStale recursively here.
	// Restoring is done after topological sorting is done.
	// If an image depends on another stale image, this means that
	// the former image can be restored from the latest state of the latter image.
}

// ClearPixels clears the specified region by ReplacePixels.
func (i *Image) ClearPixels(x, y, width, height int) {
	i.ReplacePixels(nil, x, y, width, height)
}

// ReplacePixels replaces the image pixels with the given pixels slice.
//
// ReplacePixels for a part is forbidden if the image is rendered with DrawTriangles or Fill.
func (i *Image) ReplacePixels(pixels []byte, x, y, width, height int) {
	select {
	case theImages.deferCh <- struct{}{}:
		break
	default:
		theImages.deferUntilBeginFrame(func() {
			i.ReplacePixels(pixels, x, y, width, height)
		})
		return
	}

	defer func() {
		<-theImages.deferCh
	}()

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

	if pixels != nil {
		i.image.ReplacePixels(pixels, x, y, width, height)
	} else {
		// TODO: When pixels == nil, we don't have to care the pixel state there. In such cases, the image
		// accepts only ReplacePixels and not Fill or DrawTriangles.
		// TODO: Separate Image struct into two: images for only-ReplacePixels, and the others.
		i.image.ReplacePixels(make([]byte, 4*width*height), x, y, width, height)
	}

	if x == 0 && y == 0 && width == w && height == h {
		if pixels != nil {
			i.basePixels.AddOrReplace(pixels, 0, 0, w, h)
		} else {
			i.basePixels.Remove(0, 0, w, h)
		}
		i.drawTrianglesHistory = nil
		i.stale = false
		return
	}

	// It looked like ReplacePixels on a part of image deletes other region that are rendered by DrawTriangles
	// (#593, #758).

	if len(i.drawTrianglesHistory) > 0 {
		panic("restorable: ReplacePixels for a part after DrawTriangles is forbidden")
	}

	if i.stale {
		// TODO: panic here?
		return
	}

	if pixels != nil {
		i.basePixels.AddOrReplace(pixels, x, y, width, height)
	} else {
		i.basePixels.Remove(x, y, width, height)
	}
}

// DrawTriangles draws a given image img to the image.
func (i *Image) DrawTriangles(img *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	select {
	case theImages.deferCh <- struct{}{}:
		break
	default:
		theImages.deferUntilBeginFrame(func() {
			i.DrawTriangles(img, vertices, indices, colorm, mode, filter, address)
		})
		return
	}

	defer func() {
		<-theImages.deferCh
	}()

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
	if len(i.drawTrianglesHistory) > 0 || i.stale {
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
	// As this should not affect the information for restoring, this doesn't have to be deferred.
	// TODO: If there are deferred operations, At doesn't return the correct color. Fix this (#913).

	w, h := i.image.Size()
	if x < 0 || y < 0 || w <= x || h <= y {
		return 0, 0, 0, 0
	}

	i.readPixelsFromGPUIfNeeded()

	return i.basePixels.At(x, y)
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
	w, h := i.Size()
	i.basePixels = Pixels{}
	i.basePixels.AddOrReplace(i.image.Pixels(), 0, 0, w, h)
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
	w, h := i.Size()

	// Dispose the internal image after getting its size for safety.
	i.image.Dispose()

	if i.screen {
		// The screen image should also be recreated because framebuffer might
		// be changed.
		i.image = graphicscommand.NewScreenFramebufferImage(w, h)
		i.basePixels = Pixels{}
		i.drawTrianglesHistory = nil
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
	// Clear the image explicitly.
	clearImage(gimg)
	i.basePixels.Apply(gimg)

	for _, c := range i.drawTrianglesHistory {
		if c.image.hasDependency() {
			panic("restorable: all dependencies must be already resolved but not")
		}
		gimg.DrawTriangles(c.image.image, c.vertices, c.indices, c.colorm, c.mode, c.filter, c.address)
	}

	if len(i.drawTrianglesHistory) > 0 {
		i.basePixels = Pixels{}
		i.basePixels.AddOrReplace(gimg.Pixels(), 0, 0, w, h)
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
	select {
	case theImages.deferCh <- struct{}{}:
		break
	default:
		theImages.deferUntilBeginFrame(i.Dispose)
		return
	}
	defer func() {
		<-theImages.deferCh
	}()

	theImages.remove(i)
	i.image.Dispose()
	i.image = nil
	i.basePixels = Pixels{}
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

func (i *Image) Dump(path string) error {
	return i.image.Dump(path)
}
