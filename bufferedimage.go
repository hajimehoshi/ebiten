// Copyright 2019 The Ebiten Authors
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

package ebiten

import (
	"fmt"
	"image"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// whiteImageSize is the width and height of whiteImage.
const whiteImageSize = 3

// whiteImage is a white image to draw the pending dots of a pixel cache.
var whiteImage *ui.Image

func init() {
	whiteImage = ui.Get().NewImage(whiteImageSize, whiteImageSize, atlas.ImageTypeRegular)
	pix := make([]byte, 4*whiteImageSize*whiteImageSize)
	for i := range pix {
		pix[i] = 0xff
	}
	whiteImage.WritePixels(pix, image.Rect(0, 0, whiteImageSize, whiteImageSize))
}

// bufferedImage is the internal image shared by an Image and its sub-images.
//
// bufferedImage keeps the CPU-side cache of the pixels and the pending pixel writes of a ui.Image.
// Draws to, writes to, and reads from the ui.Image go through bufferedImage, so the pending writes
// are applied before the ui.Image's pixels are used, and the cache is invalidated when the ui.Image
// is modified.
type bufferedImage struct {
	image *ui.Image

	// mu protects cache.
	//
	// mu is never held while waiting for a frame, since the frame might need mu to draw the image.
	// mu is never held while locking another bufferedImage's mu.
	mu    sync.Mutex
	cache pixelCache
}

func newBufferedImage(width, height int, imageType atlas.ImageType) *bufferedImage {
	b := &bufferedImage{
		image: ui.Get().NewImage(width, height, imageType),
	}
	b.cache.init(width, height)
	return b
}

func (b *bufferedImage) Deallocate() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.image.Deallocate()
	b.cache.deallocate()
}

// theReadPixelsError is the first error at a synchronous pixel read.
var theReadPixelsError atomic.Pointer[error]

// panicOnErrorOnReadingPixels indicates whether an error at a synchronous pixel read panics.
// This value is set only in tests.
var panicOnErrorOnReadingPixels bool

// readPixelsError returns the first error at a synchronous pixel read, if any.
func readPixelsError() error {
	if err := theReadPixelsError.Load(); err != nil {
		return *err
	}
	return nil
}

// setReadPixelsError records err at a synchronous pixel read, which ends the game at the next Update.
func setReadPixelsError(err error) {
	if panicOnErrorOnReadingPixels {
		panic(err)
	}
	theReadPixelsError.CompareAndSwap(nil, &err)
}

// ReadPixels reads the pixels in region into pixels, from the cache when possible.
// An error is reported as the error of the game.
func (b *bufferedImage) ReadPixels(pixels []byte, region image.Rectangle) {
	// Check the error existence and avoid unnecessary calls.
	if readPixelsError() != nil {
		return
	}
	if !isRunGameStarted() {
		panic("ebiten: ReadPixels cannot be called before the game starts")
	}

	load := b.startReadPixels(pixels, region)
	b.finishReadPixels(load, pixels, region)
}

// startReadPixels reads the cached pixels in region into pixels, and starts loading the other pixels.
func (b *bufferedImage) startReadPixels(pixels []byte, region image.Rectangle) *tilesLoad {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.cache.startReadPixels(b.image, pixels, region)
}

// finishReadPixels waits for load, and reads the loaded pixels in region into pixels.
// An error is reported as the error of the game.
func (b *bufferedImage) finishReadPixels(load *tilesLoad, pixels []byte, region image.Rectangle) {
	if load == nil {
		return
	}
	// Waiting for the read-backs can need the next frame, so mu must not be held here.
	for _, r := range load.results {
		if err := r.Wait(); err != nil {
			setReadPixelsError(err)
			return
		}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.finishReadPixels(load, pixels, region)
}

// ReadPixelsAsync queues a pixel read-back and returns its result.
//
// Read-back must neither read from nor populate the CPU pixel cache. Pending CPU writes are
// flushed to the GPU before the read; GPU pixels go directly into the caller's buffer.
func (b *bufferedImage) ReadPixelsAsync(pixels []byte, region image.Rectangle) *graphicscommand.PixelReadback {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.writeBackPixelsIfNeeded(b.image)
	return b.image.ReadPixelsAsync(pixels, region)
}

func (b *bufferedImage) DumpScreenshot(name string, blackbg bool) (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.writeBackPixelsIfNeeded(b.image)
	return b.image.DumpScreenshot(name, blackbg)
}

// WritePixels replaces the pixels at the specified region.
func (b *bufferedImage) WritePixels(pix []byte, region image.Rectangle) {
	if l := 4 * region.Dx() * region.Dy(); len(pix) != l {
		panic(fmt.Sprintf("ebiten: len(pix) was %d but must be %d", len(pix), l))
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.writePixels(b.image, pix, region)
}

// DrawTriangles draws the src images with the given vertices.
//
// Copying vertices and indices is the caller's responsibility.
func (b *bufferedImage) DrawTriangles(srcs [graphics.ShaderSrcImageCount]*bufferedImage, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *ui.Shader, uniforms []uint32, canSkipMipmap bool) {
	var imgs [graphics.ShaderSrcImageCount]*ui.Image
	for i, src := range srcs {
		if src == nil {
			continue
		}
		// src's pending pixel writes have to be applied to the GPU,
		// but the cache doesn't have to be cleared since src is not modified in this function.
		src.writeBackPixelsIfNeeded()
		imgs[i] = src.image
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.writeBackPixelsIfNeeded(b.image)
	b.image.DrawTriangles(imgs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, canSkipMipmap)

	// After rendering, the pixel cache is no longer valid.
	b.cache.reset()
}

// Fill fills the region with the given premultiplied-alpha color.
func (b *bufferedImage) Fill(cr, cg, cb, ca float32, region image.Rectangle) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.writeBackPixelsIfNeeded(b.image)
	b.image.Fill(cr, cg, cb, ca, region)
	b.cache.reset()
}

// drawOnUIImage calls drawer to draw on the ui.Image directly, with dstRegion as the region of the ui.Image.
func (b *bufferedImage) drawOnUIImage(drawer ui.EbitenImageDrawer, dstRegion image.Rectangle) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.writeBackPixelsIfNeeded(b.image)
	drawer.DrawOnImage(b.image, dstRegion)
	b.cache.reset()
}

// writeBackPixelsIfNeeded applies the pending pixel writes to the GPU.
func (b *bufferedImage) writeBackPixelsIfNeeded() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.cache.writeBackPixelsIfNeeded(b.image)
}
