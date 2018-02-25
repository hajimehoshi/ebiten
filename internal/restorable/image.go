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
	"image/color"
	"math"
	"runtime"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	emath "github.com/hajimehoshi/ebiten/internal/math"
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

	paddingX0 float64
	paddingY0 float64
	paddingX1 float64
	paddingY1 float64
}

// NewImage creates an empty image with the given size.
func NewImage(width, height int, volatile bool) *Image {
	i := &Image{
		image:    graphics.NewImage(width, height),
		volatile: volatile,
	}
	theImages.add(i)
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

// NewScreenFramebufferImage creates a special image that framebuffer is one for the screen.
func NewScreenFramebufferImage(width, height int, paddingX0, paddingY0, paddingX1, paddingY1 float64) *Image {
	i := &Image{
		image:     graphics.NewScreenFramebufferImage(width, height),
		volatile:  true,
		screen:    true,
		paddingX0: paddingX0,
		paddingY0: paddingY0,
		paddingX1: paddingX1,
		paddingY1: paddingY1,
	}
	theImages.add(i)
	runtime.SetFinalizer(i, (*Image).Dispose)
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
}

var (
	dummyImage  = graphics.NewImage(16, 16)
	clearColorM = &affine.ColorM{}
)

func init() {
	clearColorM.Scale(0, 0, 0, 0)
}

// clearIfVolatile clears the image if the image is volatile.
func (i *Image) clearIfVolatile() {
	if !i.volatile {
		return
	}
	i.basePixels = nil
	i.drawImageHistory = nil
	i.stale = false
	if i.image == nil {
		panic("not reached")
	}

	w, h := i.image.Size()
	x0 := float32(0)
	y0 := float32(0)
	x1 := float32(w + int(math.Ceil(i.paddingX0+i.paddingX1)))
	y1 := float32(h + int(math.Ceil(i.paddingY0+i.paddingY1)))
	// For the rule of values, see vertices.go.
	clearVertices := []float32{
		x0, y0, 0, 0, 1, 1,
		x1, y0, 1, 0, 0, 1,
		x0, y1, 0, 1, 1, 0,
		x1, y1, 1, 1, 0, 0,
	}
	i.image.DrawImage(dummyImage, clearVertices, clearColorM, opengl.CompositeModeCopy, graphics.FilterNearest)
}

// ReplacePixels replaces the image pixels with the given pixels slice.
func (i *Image) ReplacePixels(pixels []byte) {
	theImages.makeStaleIfDependingOn(i)
	i.image.ReplacePixels(pixels)
	i.basePixels = pixels
	i.drawImageHistory = nil
	i.stale = false
}

// DrawImage draws a given image img to the image.
func (i *Image) DrawImage(img *Image, vertices []float32, colorm *affine.ColorM, mode opengl.CompositeMode, filter graphics.Filter) {
	theImages.makeStaleIfDependingOn(i)
	if img.stale || img.volatile || !IsRestoringEnabled() {
		i.makeStale()
	} else {
		i.appendDrawImageHistory(img, vertices, colorm, mode, filter)
	}
	i.image.DrawImage(img.image, vertices, colorm, mode, filter)
}

// appendDrawImageHistory appends a draw-image history item to the image.
func (i *Image) appendDrawImageHistory(image *Image, vertices []float32, colorm *affine.ColorM, mode opengl.CompositeMode, filter graphics.Filter) {
	if i.stale || i.volatile {
		return
	}
	if len(i.drawImageHistory) > 0 {
		last := i.drawImageHistory[len(i.drawImageHistory)-1]
		if last.canMerge(image, colorm, mode, filter) {
			last.vertices = append(last.vertices, vertices...)
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
		vertices: vertices,
		colorm:   *colorm,
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
	w2, h2 := emath.NextPowerOf2Int(w), emath.NextPowerOf2Int(h)
	if x < 0 || y < 0 || w2 <= x || h2 <= y {
		return color.RGBA{}, nil
	}
	if i.basePixels == nil || i.drawImageHistory != nil || i.stale {
		if err := i.readPixelsFromGPU(i.image); err != nil {
			return color.RGBA{}, err
		}
	}
	idx := 4*x + 4*y*w2
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
func (i *Image) readPixelsFromGPU(image *graphics.Image) error {
	var err error
	i.basePixels, err = image.Pixels()
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
	if !i.stale {
		return nil
	}
	return i.readPixelsFromGPU(i.image)
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
		gimg.ReplacePixels(i.basePixels)
	} else {
		// Clear the image explicitly.
		pix := make([]uint8, w*h*4)
		gimg.ReplacePixels(pix)
	}
	for _, c := range i.drawImageHistory {
		// All dependencies must be already resolved.
		if c.image.hasDependency() {
			panic("not reached")
		}
		gimg.DrawImage(c.image.image, c.vertices, &c.colorm, c.mode, c.filter)
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
	theImages.makeStaleIfDependingOn(i)
	i.image.Dispose()
	i.image = nil
	i.basePixels = nil
	i.drawImageHistory = nil
	i.stale = false
	theImages.remove(i)
	runtime.SetFinalizer(i, nil)
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
