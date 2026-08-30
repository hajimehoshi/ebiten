// Copyright 2022 The Ebiten Authors
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

package ui

import (
	"cmp"
	"image"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/mipmap"
)

// panicOnErrorOnReadingPixels indicates whether reading pixels panics on an error or not.
// This value is set only on testing.
var panicOnErrorOnReadingPixels bool

func SetPanicOnErrorOnReadingPixelsForTesting(value bool) {
	panicOnErrorOnReadingPixels = value
}

var imageFromEbitenImage func(ebitenImage any) (*Image, image.Rectangle)

// SetImageFromEbitenImageFunc sets the implementation of [ImageFromEbitenImage]. Package ebiten calls
// this at initialization: this package cannot import ebiten, so the bridge is a function value.
func SetImageFromEbitenImageFunc(f func(ebitenImage any) (*Image, image.Rectangle)) {
	imageFromEbitenImage = f
}

// ImageFromEbitenImage returns the internal image and the adjusted bounds underlying a public
// ebiten.Image, or a nil image if the ebiten image is disposed.
func ImageFromEbitenImage(ebitenImage any) (*Image, image.Rectangle) {
	return imageFromEbitenImage(ebitenImage)
}

// lastImageID is the last ID given to an image.
var lastImageID atomic.Int64

type Image struct {
	ui *UserInterface

	mipmap    *mipmap.Mipmap
	width     int
	height    int
	imageType atlas.ImageType

	// id determines the order to lock mu among multiple images.
	id int64

	// mu protects the members below and the underlying mipmap image.
	//
	// An ebiten.Image and its sub-images share one ui.Image, so operations on an image and its
	// sub-images are serialized here even when they come from multiple goroutines (#3515).
	mu sync.Mutex

	// lastBlend is the lastly-used blend for mipmap.Image.
	lastBlend graphicsdriver.Blend

	// modifyCallback is a callback called when DrawTriangles or WritePixels is called.
	// modifyCallback is useful to detect whether the image is manipulated or not after a certain time.
	modifyCallback func()

	tmpVerticesForFill []float32
}

func (u *UserInterface) NewImage(width, height int, imageType atlas.ImageType) *Image {
	return &Image{
		ui:        u,
		mipmap:    mipmap.New(width, height, imageType),
		width:     width,
		height:    height,
		imageType: imageType,
		id:        lastImageID.Add(1),
		lastBlend: graphicsdriver.BlendSourceOver,
	}
}

// imagesLocker locks the mutexes of a destination image and its source images.
//
// The mutexes are locked in the order of the image IDs, so that two draws in the opposite
// directions cannot deadlock each other.
type imagesLocker struct {
	imgs  [graphics.ShaderSrcImageCount + 1]*Image
	count int
}

func (l *imagesLocker) lock(dst *Image, srcs [graphics.ShaderSrcImageCount]*Image) {
	l.imgs[0] = dst
	copy(l.imgs[1:], srcs[:])

	// Sorting gathers the nil sources at the end, and puts duplicates next to each other.
	// One ui.Image is shared by an image and its sub-images, so a source can be the same as the
	// destination or another source, and locking the same mutex twice would deadlock.
	slices.SortFunc(l.imgs[:], func(a, b *Image) int {
		switch {
		case a == b:
			return 0
		case a == nil:
			return 1
		case b == nil:
			return -1
		default:
			return cmp.Compare(a.id, b.id)
		}
	})
	imgs := slices.Compact(l.imgs[:])
	if imgs[len(imgs)-1] == nil {
		imgs = imgs[:len(imgs)-1]
	}

	l.count = len(imgs)
	for _, img := range imgs {
		img.mu.Lock()
	}
}

func (l *imagesLocker) unlock() {
	for _, img := range slices.Backward(l.imgs[:l.count]) {
		img.mu.Unlock()
	}
}

func (i *Image) Deallocate() {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.mipmap == nil {
		return
	}
	i.mipmap.Deallocate()
}

func (i *Image) DrawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, canSkipMipmap bool) {
	var l imagesLocker
	l.lock(i, srcs)
	defer l.unlock()

	i.drawTriangles(srcs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, canSkipMipmap)
}

// drawTriangles must be called with the mutexes of the image and the source images locked.
func (i *Image) drawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, canSkipMipmap bool) {
	if i.modifyCallback != nil {
		i.modifyCallback()
	}

	i.lastBlend = blend

	var srcMipmaps [graphics.ShaderSrcImageCount]*mipmap.Mipmap
	for i, src := range srcs {
		if src == nil {
			continue
		}
		srcMipmaps[i] = src.mipmap
	}

	i.mipmap.DrawTriangles(srcMipmaps, vertices, indices, blend, dstRegion, srcRegions, shader.shader, uniforms, canSkipMipmap)
}

func (i *Image) WritePixels(pix []byte, region image.Rectangle) {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.modifyCallback != nil {
		i.modifyCallback()
	}
	i.mipmap.WritePixels(pix, region)
}

func (i *Image) ReadPixels(pixels []byte, region image.Rectangle, useCache bool) {
	// Check the error existence and avoid unnecessary calls.
	if i.ui.error() != nil {
		return
	}

	if err := i.ui.readPixels(i, pixels, region, useCache); err != nil {
		if panicOnErrorOnReadingPixels {
			panic(err)
		}
		i.ui.setError(err)
	}
}

// readPixels reports whether the pixels were read.
//
// The caller must not hold the mutex: a failed read is retried in a frame, and the frame needs the
// mutex to draw the image.
func (i *Image) readPixels(pixels []byte, region image.Rectangle, useCache bool) (bool, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.mipmap.ReadPixels(i.ui.graphicsDriver, pixels, region, useCache)
}

func (i *Image) DumpScreenshot(name string, blackbg bool) (string, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	return i.ui.dumpScreenshot(i.mipmap, name, blackbg)
}

func (u *UserInterface) DumpImages(dir string) (string, error) {
	return u.dumpImages(dir)
}

func (i *Image) clear() {
	i.Fill(0, 0, 0, 0, image.Rect(0, 0, i.width, i.height))
}

func (i *Image) fillBlack() {
	i.Fill(0, 0, 0, 1, image.Rect(0, 0, i.width, i.height))
}

func (i *Image) Fill(r, g, b, a float32, region image.Rectangle) {
	// The white image is not locked: it is never modified after the initialization, and a fill
	// skips mipmaps, so drawing from it cannot modify it.
	i.mu.Lock()
	defer i.mu.Unlock()

	srcs := [graphics.ShaderSrcImageCount]*Image{i.ui.whiteImage}

	if len(i.tmpVerticesForFill) < 4*graphics.VertexFloatCount {
		i.tmpVerticesForFill = make([]float32, 4*graphics.VertexFloatCount)
	}
	// i.tmpVerticesForFill can be reused as this is sent to drawTriangles immediately.
	graphics.QuadVerticesFromSrcAndMatrix(
		i.tmpVerticesForFill,
		1, 1, float32(i.ui.whiteImage.width-1), float32(i.ui.whiteImage.height-1),
		float32(i.width), 0, 0, float32(i.height), 0, 0,
		r, g, b, a)
	is := graphics.QuadIndices()

	blend := graphicsdriver.BlendCopy
	// If possible, use BlendSourceOver to encourage batching (#2817).
	if a == 1 && i.lastBlend == graphicsdriver.BlendSourceOver {
		blend = graphicsdriver.BlendSourceOver
	}
	sr := image.Rect(0, 0, i.ui.whiteImage.width, i.ui.whiteImage.height)
	// i.lastBlend is updated in drawTriangles.
	i.drawTriangles(srcs, i.tmpVerticesForFill, is, blend, region, [graphics.ShaderSrcImageCount]image.Rectangle{sr}, NearestFilterShader, nil, true)
}
