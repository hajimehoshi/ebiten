// Copyright 2018 The Ebiten Authors
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

package atlas

import (
	"fmt"
	"image"
	"math"
	"math/bits"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/packing"
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

var (
	minSourceSize      = 0
	minDestinationSize = 0
	maxSize            = 0
)

func appendDeferred(f func()) {
	deferredM.Lock()
	defer deferredM.Unlock()
	deferred = append(deferred, f)
}

func flushDeferred() {
	deferredM.Lock()
	fs := deferred
	deferred = nil
	deferredM.Unlock()

	for _, f := range fs {
		f()
	}
}

// baseCountToPutOnSourceBackend represents the base time duration when the image can be put onto an atlas.
// Actual time duration is increased in an exponential way for each usage as a rendering target.
const baseCountToPutOnSourceBackend = 10

func putImagesOnSourceBackend() {
	// The counter usedAsDestinationCount is updated at most once per frame (#2676).
	imagesUsedAsDestination.forEach(func(i *Image) {
		// i.backend can be nil after deallocate is called.
		if i.backend == nil {
			i.usedAsDestinationCount = 0
			return
		}
		// This counter is not updated when the backend is created in this frame.
		if !i.backendCreatedInThisFrame && i.usedAsDestinationCount < math.MaxInt {
			i.usedAsDestinationCount++
		}
		i.backendCreatedInThisFrame = false
	})
	imagesUsedAsDestination.clear()

	imagesToPutOnSourceBackend.forEach(func(i *Image) {
		// i.backend can be nil after deallocate is called.
		if i.backend == nil {
			i.usedAsSourceCount = 0
			return
		}
		if i.usedAsSourceCount < math.MaxInt {
			i.usedAsSourceCount++
		}
		if i.usedAsSourceCount >= baseCountToPutOnSourceBackend*(1<<uint(min(i.usedAsDestinationCount, 31))) {
			i.putOnSourceBackend()
			i.usedAsSourceCount = 0
		}
	})
	imagesToPutOnSourceBackend.clear()
}

// backend is a big texture atlas that can have multiple images.
// backend is a texture in GPU.
type backend struct {
	// restorable is an atlas on which there might be multiple images.
	restorable *restorable.Image

	// page is an atlas map. Each part is called a node.
	// If page is nil, the backend's image is isolated and not on an atlas.
	page *packing.Page

	// source reports whether this backend is mainly used a rendering source, but this is not 100%.
	//
	// If a non-source (destination) image is used as a source many times,
	// the image's backend might be turned into a source backend to optimize draw calls.
	source bool

	// sourceInThisFrame reports whether this backend is used as a source in this frame.
	// sourceInThisFrame is reset every frame.
	sourceInThisFrame bool
}

func (b *backend) tryAlloc(width, height int) (*packing.Node, bool) {
	if b.page == nil {
		return nil, false
	}
	n := b.page.Alloc(width, height)
	if n == nil {
		// The page can't be extended anymore. Return as failure.
		return nil, false
	}

	b.restorable = b.restorable.Extend(b.page.Size())

	return n, true
}

var (
	// backendsM is a mutex for critical sections of the backend and packing.Node objects.
	backendsM sync.Mutex

	// inFrame indicates whether the current state is in between BeginFrame and EndFrame or not.
	// If inFrame is false, function calls on an image should be deferred until the next BeginFrame.
	inFrame bool

	initOnce sync.Once

	// theBackends is a set of atlases.
	theBackends []*backend

	imagesToPutOnSourceBackend imageSmallSet

	imagesUsedAsDestination imageSmallSet

	deferred []func()

	// deferredM is a mutex for the slice operations. This must not be used for other usages.
	deferredM sync.Mutex
)

// ImageType represents the type of an image.
type ImageType int

const (
	// ImageTypeRegular is a regular image, that can be on a big texture atlas (backend).
	ImageTypeRegular ImageType = iota

	// ImageTypeScreen is a screen image that is not on an atlas.
	// A screen image is also unmanaged.
	ImageTypeScreen

	// ImageTypeVolatile is a volatile image that is cleared every frame.
	// A volatile image is also unmanaged.
	ImageTypeVolatile

	// ImageTypeUnmanaged is an unmanaged image that is not on an atlas.
	ImageTypeUnmanaged
)

// Image is a rectangle pixel set that might be on an atlas.
type Image struct {
	*imageImpl

	cleanup runtime.Cleanup
}

type imageImpl struct {
	width     int
	height    int
	imageType ImageType

	backend                   *backend
	backendCreatedInThisFrame bool

	node *packing.Node

	// usedAsSourceCount represents how long the image is used as a rendering source and kept not modified with
	// DrawTriangles.
	// In the current implementation, if an image is being modified by DrawTriangles, the image is separated from
	// a restorable image on an atlas by ensureIsolatedFromSource.
	//
	// The type is int64 instead of int to avoid overflow when comparing the limitation.
	//
	// usedAsSourceCount is increased if the image is used as a rendering source, or set to 0 if the image is
	// modified.
	//
	// WritePixels doesn't affect this value since WritePixels can be done on images on an atlas.
	usedAsSourceCount int64

	// usedAsDestinationCount represents how many times an image is used as a rendering destination at DrawTriangles.
	// usedAsDestinationCount affects the calculation when to put the image onto a texture atlas again.
	//
	// usedAsDestinationCount is never reset.
	usedAsDestinationCount int
}

// moveTo moves its content to the given image dst.
// After moveTo is called, the image i is no longer available.
//
// moveTo is similar to C++'s move semantics.
func (i *Image) moveTo(dst *Image) {
	dst.deallocateImpl()
	dst.cleanup.Stop()

	impl := *i.imageImpl
	dst.imageImpl = &impl
	if dst.backend != nil {
		dst.cleanup = runtime.AddCleanup(dst, (*imageImpl).cleanup, dst.imageImpl)
	}

	// i is no longer available but the finalizer must not be called
	// since i and dst share the same backend and the same node.
	i.cleanup.Stop()
}

func (i *imageImpl) isOnAtlas() bool {
	return i.node != nil
}

func (i *Image) isOnSourceBackend() bool {
	if i.backend == nil {
		return false
	}
	return i.backend.source
}

func (i *Image) resetUsedAsSourceCount() {
	i.usedAsSourceCount = 0
	imagesToPutOnSourceBackend.remove(i)
}

func (i *imageImpl) paddingSize() int {
	if i.imageType == ImageTypeRegular {
		return 1
	}
	return 0
}

func (i *Image) ensureIsolatedFromSource(backends []*backend) {
	i.resetUsedAsSourceCount()

	// imagesUsedAsDestination affects the counter usedAsDestination.
	// The larger this counter is, the harder it is for the image to be transferred to the source backend.
	imagesUsedAsDestination.add(i)

	if i.backend == nil {
		// `sourceInThisFrame` of `backends` should be true, so `backends` should be in `bs`.
		var bs []*backend
		for _, b := range theBackends {
			if b.sourceInThisFrame {
				bs = append(bs, b)
			}
		}
		i.allocate(bs, false)
		i.backendCreatedInThisFrame = true
		return
	}

	if !i.isOnAtlas() {
		return
	}

	// Check if i has the same backend as the given backends.
	var needsIsolation bool
	for _, b := range backends {
		if i.backend == b {
			needsIsolation = true
			break
		}
	}
	if !needsIsolation {
		return
	}

	newI := NewImage(i.width, i.height, i.imageType)

	// Call allocate explicitly in order to have an isolated backend from the specified backends.
	// `sourceInThisFrame` of `backends` should be true, so `backends` should be in `bs`.
	bs := []*backend{i.backend}
	for _, b := range theBackends {
		if b.sourceInThisFrame {
			bs = append(bs, b)
		}
	}
	newI.allocate(bs, false)

	w, h := float32(i.width), float32(i.height)
	vs := make([]float32, 4*graphics.VertexFloatCount)
	graphics.QuadVerticesFromDstAndSrc(vs, 0, 0, w, h, 0, 0, w, h, 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, i.width, i.height)
	sr := image.Rect(0, 0, i.width, i.height)

	newI.drawTriangles([graphics.ShaderSrcImageCount]*Image{i}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{sr}, NearestFilterShader, nil, graphicsdriver.FillRuleFillAll, restorable.HintOverwriteDstRegion)
	newI.moveTo(i)
}

func (i *Image) putOnSourceBackend() {
	if i.backend == nil {
		i.allocate(nil, true)
		return
	}

	if i.isOnSourceBackend() {
		return
	}

	if !i.canBePutOnAtlas() {
		panic("atlas: putOnSourceBackend cannot be called on a image that cannot be on an atlas")
	}

	if i.imageType != ImageTypeRegular {
		panic(fmt.Sprintf("atlas: the image type must be ImageTypeRegular but %d", i.imageType))
	}

	newI := NewImage(i.width, i.height, ImageTypeRegular)
	newI.allocate(nil, true)

	w, h := float32(i.width), float32(i.height)
	vs := make([]float32, 4*graphics.VertexFloatCount)
	graphics.QuadVerticesFromDstAndSrc(vs, 0, 0, w, h, 0, 0, w, h, 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, i.width, i.height)
	sr := image.Rect(0, 0, i.width, i.height)
	newI.drawTriangles([graphics.ShaderSrcImageCount]*Image{i}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{sr}, NearestFilterShader, nil, graphicsdriver.FillRuleFillAll, restorable.HintOverwriteDstRegion)

	newI.moveTo(i)
	i.usedAsSourceCount = 0

	if !i.isOnSourceBackend() {
		panic("atlas: i must be on a source backend but not")
	}
}

func (i *imageImpl) regionWithPadding() image.Rectangle {
	if i.backend == nil {
		panic("atlas: backend must not be nil: not allocated yet?")
	}
	if !i.isOnAtlas() {
		return image.Rect(0, 0, i.width+i.paddingSize(), i.height+i.paddingSize())
	}
	return i.node.Region()
}

// DrawTriangles draws triangles with the given image.
//
// The vertex floats are:
//
//	0: Destination X in pixels
//	1: Destination Y in pixels
//	2: Source X in pixels (the upper-left is (0, 0))
//	3: Source Y in pixels
//	4: Color R [0.0-1.0]
//	5: Color G
//	6: Color B
//	7: Color Y
func (i *Image) DrawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule, hint restorable.Hint) {
	backendsM.Lock()
	defer backendsM.Unlock()

	if !inFrame {
		vs := make([]float32, len(vertices))
		copy(vs, vertices)
		is := make([]uint32, len(indices))
		copy(is, indices)
		us := make([]uint32, len(uniforms))
		copy(us, uniforms)

		appendDeferred(func() {
			i.drawTriangles(srcs, vs, is, blend, dstRegion, srcRegions, shader, us, fillRule, hint)
		})
		return
	}

	i.drawTriangles(srcs, vertices, indices, blend, dstRegion, srcRegions, shader, uniforms, fillRule, hint)
}

func (i *Image) drawTriangles(srcs [graphics.ShaderSrcImageCount]*Image, vertices []float32, indices []uint32, blend graphicsdriver.Blend, dstRegion image.Rectangle, srcRegions [graphics.ShaderSrcImageCount]image.Rectangle, shader *Shader, uniforms []uint32, fillRule graphicsdriver.FillRule, hint restorable.Hint) {
	backends := make([]*backend, 0, len(srcs))
	for _, src := range srcs {
		if src == nil {
			continue
		}
		if src.backend == nil {
			// It is possible to spcify i.backend as a forbidden backend, but this might prevent a good allocation for a source image.
			// If the backend becomes the same as i's, i's backend will be changed at ensureIsolatedFromSource.
			src.allocate(nil, true)
		}
		backends = append(backends, src.backend)
		src.backend.sourceInThisFrame = true
	}

	i.ensureIsolatedFromSource(backends)

	for _, src := range srcs {
		// Compare i and source images after ensuring i is not on an atlas, or
		// i and a source image might share the same atlas even though i != src.
		if src != nil && i.backend.restorable == src.backend.restorable {
			panic("atlas: Image.DrawTriangles: source must be different from the receiver")
		}
	}

	r := i.regionWithPadding()
	// TODO: Check if dstRegion does not to violate the region.
	dstRegion = dstRegion.Add(r.Min)

	dx, dy := float32(r.Min.X), float32(r.Min.Y)

	var oxf, oyf float32
	if srcs[0] != nil {
		r := srcs[0].regionWithPadding()
		oxf, oyf = float32(r.Min.X), float32(r.Min.Y)
		n := len(vertices)
		for i := 0; i < n; i += graphics.VertexFloatCount {
			vertices[i] += dx
			vertices[i+1] += dy
			vertices[i+2] += oxf
			vertices[i+3] += oyf
		}
		if shader.unit == shaderir.Texels {
			sw, sh := srcs[0].backend.restorable.InternalSize()
			swf, shf := float32(sw), float32(sh)
			for i := 0; i < n; i += graphics.VertexFloatCount {
				vertices[i+2] /= swf
				vertices[i+3] /= shf
			}
		}
	} else {
		n := len(vertices)
		for i := 0; i < n; i += graphics.VertexFloatCount {
			vertices[i] += dx
			vertices[i+1] += dy
		}
	}

	var imgs [graphics.ShaderSrcImageCount]*restorable.Image
	for i, src := range srcs {
		if src == nil {
			continue
		}

		// A source region can be deliberately empty when this is not needed in order to avoid unexpected
		// performance issue (#1293).
		// TODO: This should no longer be needed but is kept just in case. Remove this later.
		if !srcRegions[i].Empty() {
			r := src.regionWithPadding()
			srcRegions[i] = srcRegions[i].Add(r.Min)
		}
		imgs[i] = src.backend.restorable
		if !src.isOnSourceBackend() && src.canBePutOnAtlas() {
			// src might already registered, but assigning it again is not harmful.
			imagesToPutOnSourceBackend.add(src)
		}
	}

	i.backend.restorable.DrawTriangles(imgs, vertices, indices, blend, dstRegion, srcRegions, shader.ensureShader(), uniforms, fillRule, hint)
}

// WritePixels replaces the pixels on the image.
func (i *Image) WritePixels(pix []byte, region image.Rectangle) {
	backendsM.Lock()
	defer backendsM.Unlock()

	if !inFrame {
		copied := make([]byte, len(pix))
		copy(copied, pix)

		appendDeferred(func() {
			i.writePixels(copied, region)
		})
		return
	}

	i.writePixels(pix, region)
}

func (i *Image) writePixels(pix []byte, region image.Rectangle) {
	if l := 4 * region.Dx() * region.Dy(); len(pix) != l {
		panic(fmt.Sprintf("atlas: len(p) must be %d but %d", l, len(pix)))
	}

	i.resetUsedAsSourceCount()

	if i.backend == nil {
		if pix == nil {
			return
		}
		// Allocate as a source as this image will likely be used as a source.
		i.allocate(nil, true)
	}

	r := i.regionWithPadding()

	if !region.Eq(image.Rect(0, 0, i.width, i.height)) || i.paddingSize() == 0 {
		region = region.Add(r.Min)

		if pix == nil {
			i.backend.restorable.ClearPixels(region)
			return
		}

		// Copy pixels in the case when pix is modified before the graphics command is executed.
		pix2 := graphics.NewManagedBytes(len(pix), func(bs []byte) {
			copy(bs, pix)
		})
		i.backend.restorable.WritePixels(pix2, region)
		return
	}

	// TODO: These loops assume that paddingSize is 1.
	// TODO: Is clearing edges explicitly really needed?
	const paddingSize = 1
	if paddingSize != i.paddingSize() {
		panic(fmt.Sprintf("atlas: writePixels assumes the padding is always 1 but the actual padding was %d", i.paddingSize()))
	}

	pixb := graphics.NewManagedBytes(4*r.Dx()*r.Dy(), func(bs []byte) {
		// Clear the edges. bs might not be zero-cleared.
		rowPixels := 4 * r.Dx()
		for i := 0; i < rowPixels; i++ {
			bs[rowPixels*(r.Dy()-1)+i] = 0
		}
		for j := 1; j < r.Dy(); j++ {
			bs[rowPixels*j-4] = 0
			bs[rowPixels*j-3] = 0
			bs[rowPixels*j-2] = 0
			bs[rowPixels*j-1] = 0
		}

		// Copy the content.
		for j := 0; j < region.Dy(); j++ {
			copy(bs[4*j*r.Dx():], pix[4*j*region.Dx():4*(j+1)*region.Dx()])
		}
	})
	i.backend.restorable.WritePixels(pixb, r)
}

func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, region image.Rectangle) (ok bool, err error) {
	backendsM.Lock()
	defer backendsM.Unlock()

	if !inFrame {
		// Not ready to read pixels. Try this later.
		return false, nil
	}

	// In the tests, BeginFrame might not be called often and then images might not be disposed (#2292).
	// To prevent memory leaks, flush the deferred functions here.
	flushDeferred()

	if i.backend == nil || i.backend.restorable == nil {
		for i := range pixels {
			pixels[i] = 0
		}
		return true, nil
	}

	if err := i.backend.restorable.ReadPixels(graphicsDriver, pixels, region.Add(i.regionWithPadding().Min)); err != nil {
		return false, err
	}
	return true, nil
}

// Deallocate deallocates the internal state.
// Even after this call, the image is still available as a new cleared image.
func (i *Image) Deallocate() {
	i.cleanup.Stop()

	backendsM.Lock()
	defer backendsM.Unlock()

	if !inFrame {
		appendDeferred(func() {
			i.deallocateImpl()
		})
		return
	}

	i.deallocateImpl()
}

func (i *imageImpl) deallocateImpl() {
	defer func() {
		i.backend = nil
		i.node = nil
	}()

	i.usedAsSourceCount = 0
	i.usedAsDestinationCount = 0

	if i.backend == nil {
		// Not allocated yet.
		return
	}

	if i.isOnAtlas() {
		i.backend.page.Free(i.node)
		if !i.backend.page.IsEmpty() {
			// As this part can be reused, this should be cleared explicitly.
			r := i.regionWithPadding()
			i.backend.restorable.ClearPixels(r)
			return
		}
	}

	i.backend.restorable.Dispose()

	for idx, sh := range theBackends {
		if sh == i.backend {
			copy(theBackends[idx:], theBackends[idx+1:])
			theBackends[len(theBackends)-1] = nil
			theBackends = theBackends[:len(theBackends)-1]
			return
		}
	}

	panic("atlas: backend not found at an image being deallocated")
}

func NewImage(width, height int, imageType ImageType) *Image {
	// Actual allocation is done lazily, and the lock is not needed.
	return &Image{
		imageImpl: &imageImpl{
			width:     width,
			height:    height,
			imageType: imageType,
		},
	}
}

func (i *Image) canBePutOnAtlas() bool {
	if minSourceSize == 0 || minDestinationSize == 0 || maxSize == 0 {
		panic("atlas: min*Size or maxSize must be initialized")
	}
	if i.imageType != ImageTypeRegular {
		return false
	}
	return i.width+i.paddingSize() <= maxSize && i.height+i.paddingSize() <= maxSize
}

func (i *imageImpl) cleanup() {
	// A function from finalizer must not be blocked, but disposing operation can be blocked.
	// Defer this operation until it becomes safe. (#913)
	appendDeferred(func() {
		i.deallocateImpl()
	})
}

func (i *Image) allocate(forbiddenBackends []*backend, asSource bool) {
	if i.backend != nil {
		panic("atlas: the image is already allocated")
	}

	i.cleanup = runtime.AddCleanup(i, (*imageImpl).cleanup, i.imageImpl)

	if i.imageType == ImageTypeScreen {
		if asSource {
			panic("atlas: a screen image cannot be created as a source")
		}
		// A screen image doesn't have a padding.
		i.backend = &backend{
			restorable: restorable.NewImage(i.width, i.height, restorable.ImageTypeScreen),
		}
		theBackends = append(theBackends, i.backend)
		return
	}

	wp := i.width + i.paddingSize()
	hp := i.height + i.paddingSize()

	if !i.canBePutOnAtlas() {
		if wp > maxSize || hp > maxSize {
			panic(fmt.Sprintf("atlas: the image being put on an atlas is too big: width: %d, height: %d", i.width, i.height))
		}

		typ := restorable.ImageTypeRegular
		if i.imageType == ImageTypeVolatile {
			typ = restorable.ImageTypeVolatile
		}
		i.backend = &backend{
			restorable: restorable.NewImage(wp, hp, typ),
			source:     asSource && typ == restorable.ImageTypeRegular,
		}
		theBackends = append(theBackends, i.backend)
		return
	}

	// Check if an existing backend is available.
loop:
	for _, b := range theBackends {
		if b.source != asSource {
			continue
		}
		for _, bb := range forbiddenBackends {
			if b == bb {
				continue loop
			}
		}

		if n, ok := b.tryAlloc(wp, hp); ok {
			i.backend = b
			i.node = n
			return
		}
	}

	var width, height int
	if asSource {
		width, height = minSourceSize, minSourceSize
	} else {
		width, height = minDestinationSize, minDestinationSize
	}
	for wp > width {
		if width == maxSize {
			panic(fmt.Sprintf("atlas: the image being put on an atlas is too big: width: %d, height: %d", i.width, i.height))
		}
		width *= 2
	}
	for hp > height {
		if height == maxSize {
			panic(fmt.Sprintf("atlas: the image being put on an atlas is too big: width: %d, height: %d", i.width, i.height))
		}
		height *= 2
	}

	typ := restorable.ImageTypeRegular
	if i.imageType == ImageTypeVolatile {
		typ = restorable.ImageTypeVolatile
	}
	b := &backend{
		restorable: restorable.NewImage(width, height, typ),
		page:       packing.NewPage(width, height, maxSize),
		source:     asSource,
	}
	theBackends = append(theBackends, b)

	n := b.page.Alloc(wp, hp)
	if n == nil {
		panic("atlas: Alloc result must not be nil at allocate")
	}
	i.backend = b
	i.node = n
}

func (i *Image) DumpScreenshot(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool) (string, error) {
	backendsM.Lock()
	defer backendsM.Unlock()

	if !inFrame {
		panic("atlas: DumpScreenshots must be called in between BeginFrame and EndFrame")
	}

	return i.backend.restorable.Dump(graphicsDriver, path, blackbg, image.Rect(0, 0, i.width, i.height))
}

func EndFrame() error {
	backendsM.Lock()
	defer backendsM.Unlock()
	defer func() {
		inFrame = false
	}()

	if !inFrame {
		panic("atlas: inFrame must be true in EndFrame")
	}

	for _, b := range theBackends {
		b.sourceInThisFrame = false
	}

	return nil
}

func SwapBuffers(graphicsDriver graphicsdriver.Graphics) error {
	func() {
		backendsM.Lock()
		defer backendsM.Unlock()

		if inFrame {
			panic("atlas: inFrame must be false in SwapBuffer")
		}
	}()

	if err := restorable.SwapBuffers(graphicsDriver); err != nil {
		return err
	}
	return nil
}

func floorPowerOf2(x int) int {
	if x <= 0 {
		return 0
	}
	return 1 << (bits.Len(uint(x)) - 1)
}

func BeginFrame(graphicsDriver graphicsdriver.Graphics) error {
	backendsM.Lock()
	defer backendsM.Unlock()

	if inFrame {
		panic("atlas: inFrame must be false in BeginFrame")
	}

	inFrame = true

	var err error
	initOnce.Do(func() {
		err = restorable.InitializeGraphicsDriverState(graphicsDriver)
		if err != nil {
			return
		}
		if len(theBackends) != 0 {
			panic("atlas: all the images must be not on an atlas before the game starts")
		}

		// min*Size and maxSize can already be set for testings.
		if minSourceSize == 0 {
			minSourceSize = 1024
		}
		if minDestinationSize == 0 {
			minDestinationSize = 16
		}
		if maxSize == 0 {
			maxSize = floorPowerOf2(restorable.MaxImageSize(graphicsDriver))
		}
	})
	if err != nil {
		return err
	}

	// Restore images first before other image manipulations (#2075).
	if err := restorable.RestoreIfNeeded(graphicsDriver); err != nil {
		return err
	}

	flushDeferred()
	putImagesOnSourceBackend()

	return nil
}

func DumpImages(graphicsDriver graphicsdriver.Graphics, dir string) (string, error) {
	backendsM.Lock()
	defer backendsM.Unlock()

	if !inFrame {
		panic("atlas: DumpImages must be called in between BeginFrame and EndFrame")
	}

	return restorable.DumpImages(graphicsDriver, dir)
}

func TotalGPUImageMemoryUsageInBytes() int64 {
	backendsM.Lock()
	defer backendsM.Unlock()

	var sum int64
	for _, b := range theBackends {
		if b.restorable == nil {
			continue
		}
		w, h := b.restorable.InternalSize()
		sum += 4 * int64(w) * int64(h)
	}
	return sum
}
