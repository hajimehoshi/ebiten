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
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/packing"
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
)

var (
	minSourceSize      = 0
	minDestinationSize = 0
	maxSize            = 0
)

type temporaryBytes struct {
	pixels           []byte
	pos              int
	notFullyUsedTime int
}

var theTemporaryBytes temporaryBytes

func temporaryBytesSize(size int) int {
	l := 16
	for l < size {
		l *= 2
	}
	return l
}

// alloc allocates the pixels and returns it.
// Be careful that the returned pixels might not be zero-cleared.
func (t *temporaryBytes) alloc(size int) []byte {
	if len(t.pixels) < t.pos+size {
		t.pixels = make([]byte, max(len(t.pixels)*2, temporaryBytesSize(size)))
		t.pos = 0
	}
	pix := t.pixels[t.pos : t.pos+size]
	t.pos += size
	return pix
}

func (t *temporaryBytes) resetAtFrameEnd() {
	const maxNotFullyUsedTime = 60

	if temporaryBytesSize(t.pos) < len(t.pixels) {
		if t.notFullyUsedTime < maxNotFullyUsedTime {
			t.notFullyUsedTime++
		}
	} else {
		t.notFullyUsedTime = 0
	}

	// Let the pixels GCed if this is not used for a while.
	if t.notFullyUsedTime == maxNotFullyUsedTime && len(t.pixels) > 0 {
		t.pixels = nil
		t.notFullyUsedTime = 0
	}

	// Reset the position and reuse the allocated bytes.
	// t.pixels should already be sent to GPU, then this can be reused.
	t.pos = 0
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

func putImagesOnSourceBackend(graphicsDriver graphicsdriver.Graphics) error {
	for i := range imagesToPutOnSourceBackend {
		i.usedAsSourceCount++
		if i.usedAsSourceCount >= baseCountToPutOnSourceBackend*(1<<uint(min(i.destinationCount, 31))) {
			if err := i.putOnSourceBackend(graphicsDriver); err != nil {
				return err
			}
			i.usedAsSourceCount = 0
			delete(imagesToPutOnSourceBackend, i)
		}
	}

	// Reset the images. The images will be registered again when it is used as a rendering source.
	for k := range imagesToPutOnSourceBackend {
		delete(imagesToPutOnSourceBackend, k)
	}
	return nil
}

type backend struct {
	// restorable is an atlas on which there might be multiple images.
	restorable *restorable.Image

	// page is an atlas map. Each part is called a node.
	// If page is nil, the backend's image is isolated and not on an atlas.
	page *packing.Page

	// source reports whether this backend is mainly used a rendering source, but this is not 100%.
	// If a non-source (destination) image is used as a source many times,
	// the image's backend might be turned into a source backend to optimize draw calls.
	source bool
}

func (b *backend) tryAlloc(width, height int) (*packing.Node, bool) {
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

	initOnce sync.Once

	// theBackends is a set of atlases.
	theBackends []*backend

	// theSourceBackendsForOneFrame is a temporary set of backends that are used as sources in one frame.
	// theSourceBackendsForOneFrame is reset every frame.
	theSourceBackendsForOneFrame = map[*backend]struct{}{}

	imagesToPutOnSourceBackend = map[*Image]struct{}{}

	deferred []func()

	// deferredM is a mutex for the slice operations. This must not be used for other usages.
	deferredM sync.Mutex
)

func init() {
	// Lock the mutex before a frame begins.
	//
	// In each frame, restoring images and resolving images happen respectively:
	//
	//   [Restore -> Resolve] -> [Restore -> Resolve] -> ...
	//
	// Between each frame, any image operations are not permitted, or stale images would remain when restoring
	// (#913).
	backendsM.Lock()
}

type ImageType int

const (
	ImageTypeRegular ImageType = iota
	ImageTypeScreen
	ImageTypeVolatile
	ImageTypeUnmanaged
)

// Image is a rectangle pixel set that might be on an atlas.
type Image struct {
	width     int
	height    int
	imageType ImageType
	disposed  bool

	backend *backend

	node *packing.Node

	// usedAsSourceCount represents how long the image is used as a rendering source and kept not modified with
	// DrawTriangles.
	// In the current implementation, if an image is being modified by DrawTriangles, the image is separated from
	// a restorable image on an atlas by ensureIsolatedFromSource.
	//
	// usedAsSourceCount is increased if the image is used as a rendering source, or set to 0 if the image is
	// modified.
	//
	// WritePixels doesn't affect this value since WritePixels can be done on images on an atlas.
	usedAsSourceCount int

	// destinationCount represents how many times an image switches its backend when the image is used
	// as a rendering destination at DrawTriangles.
	// destinationCount affects the calculation when to put the image onto a texture atlas again.
	destinationCount int
}

// moveTo moves its content to the given image dst.
// After moveTo is called, the image i is no longer available.
//
// moveTo is similar to C++'s move semantics.
func (i *Image) moveTo(dst *Image) {
	dst.dispose(false)
	*dst = *i

	// i is no longer available but Dispose must not be called
	// since i and dst have the same values as node.
	runtime.SetFinalizer(i, nil)
}

func (i *Image) isOnAtlas() bool {
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
	delete(imagesToPutOnSourceBackend, i)
}

func (i *Image) paddingSize() int {
	if i.imageType == ImageTypeRegular {
		return 1
	}
	return 0
}

func (i *Image) ensureIsolatedFromSource(backends []*backend) {
	i.resetUsedAsSourceCount()

	if i.backend == nil {
		// `theSourceBackendsForOneFrame` already includes `backends`.
		bs := make([]*backend, 0, len(theSourceBackendsForOneFrame))
		for b := range theSourceBackendsForOneFrame {
			bs = append(bs, b)
		}
		i.allocate(bs, false)
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
	// `theSourceBackendsForOneFrame` already includes `backends`.
	bs := make([]*backend, 0, 1+len(theSourceBackendsForOneFrame))
	bs = append(bs, i.backend)
	for b := range theSourceBackendsForOneFrame {
		bs = append(bs, b)
	}
	newI.allocate(bs, false)

	w, h := float32(i.width), float32(i.height)
	vs := make([]float32, 4*graphics.VertexFloatCount)
	graphics.QuadVertices(vs, 0, 0, w, h, 1, 0, 0, 1, 0, 0, 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}

	origBackend := i.backend
	newI.drawTriangles([graphics.ShaderImageCount]*Image{i}, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, NearestFilterShader, nil, false, true)
	delete(theSourceBackendsForOneFrame, origBackend)

	newI.moveTo(i)

	// Count up only when the backend is switched. (#2586).
	// This counting up must be done after moveTo.
	i.destinationCount++
}

func (i *Image) putOnSourceBackend(graphicsDriver graphicsdriver.Graphics) error {
	if i.backend == nil {
		i.allocate(nil, true)
		return nil
	}

	if i.isOnSourceBackend() {
		return nil
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
	graphics.QuadVertices(vs, 0, 0, w, h, 1, 0, 0, 1, 0, 0, 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	newI.drawTriangles([graphics.ShaderImageCount]*Image{i}, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, NearestFilterShader, nil, false, true)

	newI.moveTo(i)
	i.usedAsSourceCount = 0

	if !i.isOnSourceBackend() {
		panic("atlas: i must be on a source backend but not")
	}

	return nil
}

func (i *Image) regionWithPadding() (x, y, width, height int) {
	if i.backend == nil {
		panic("atlas: backend must not be nil: not allocated yet?")
	}
	if !i.isOnAtlas() {
		return 0, 0, i.width + 2*i.paddingSize(), i.height + 2*i.paddingSize()
	}
	return i.node.Region()
}

func (i *Image) processSrc(src *Image) {
	if src == nil {
		return
	}
	if src.disposed {
		panic("atlas: the drawing source image must not be disposed (DrawTriangles)")
	}

	if src.backend == nil {
		backends := make([]*backend, 0, 1)
		if i.backend != nil {
			backends = append(backends, i.backend)
		}
		src.allocate(backends, true)
	}

	// Compare i and source images after ensuring i is not on an atlas, or
	// i and a source image might share the same atlas even though i != src.
	if i.backend.restorable == src.backend.restorable {
		panic("atlas: Image.DrawTriangles: source must be different from the receiver")
	}
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
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageCount]*Image, vertices []float32, indices []uint16, blend graphicsdriver.Blend, dstRegion, srcRegion graphicsdriver.Region, subimageOffsets [graphics.ShaderImageCount - 1][2]float32, shader *Shader, uniforms []uint32, evenOdd bool) {
	backendsM.Lock()
	defer backendsM.Unlock()
	i.drawTriangles(srcs, vertices, indices, blend, dstRegion, srcRegion, subimageOffsets, shader, uniforms, evenOdd, false)
}

func (i *Image) drawTriangles(srcs [graphics.ShaderImageCount]*Image, vertices []float32, indices []uint16, blend graphicsdriver.Blend, dstRegion, srcRegion graphicsdriver.Region, subimageOffsets [graphics.ShaderImageCount - 1][2]float32, shader *Shader, uniforms []uint32, evenOdd bool, keepOnAtlas bool) {
	if i.disposed {
		panic("atlas: the drawing target image must not be disposed (DrawTriangles)")
	}

	backends := make([]*backend, 0, len(srcs))
	for _, src := range srcs {
		if src == nil {
			continue
		}
		if src.backend == nil {
			// It is possible to spcify i.backend as a forbidden backend, but this might prevent a good allocation for a source image.
			// If the backend becomes the same as i's, this will be changed later.
			src.allocate(nil, true)
		}
		backends = append(backends, src.backend)
		theSourceBackendsForOneFrame[src.backend] = struct{}{}
	}
	i.ensureIsolatedFromSource(backends)

	for _, src := range srcs {
		i.processSrc(src)
	}

	x, y, _, _ := i.regionWithPadding()
	ps := i.paddingSize()
	dx, dy := float32(x+ps), float32(y+ps)
	// TODO: Check if dstRegion does not to violate the region.

	dstRegion.X += dx
	dstRegion.Y += dy

	var oxf, oyf float32
	if srcs[0] != nil {
		ox, oy, _, _ := srcs[0].regionWithPadding()
		ps := srcs[0].paddingSize()
		oxf, oyf = float32(ox+ps), float32(oy+ps)
		sw, sh := srcs[0].backend.restorable.InternalSize()
		swf, shf := float32(sw), float32(sh)
		n := len(vertices)
		for i := 0; i < n; i += graphics.VertexFloatCount {
			vertices[i] = vertices[i] + dx
			vertices[i+1] = vertices[i+1] + dy
			vertices[i+2] = (vertices[i+2] + oxf) / swf
			vertices[i+3] = (vertices[i+3] + oyf) / shf
		}
		// srcRegion can be deliberately empty when this is not needed in order to avoid unexpected
		// performance issue (#1293).
		if srcRegion.Width != 0 && srcRegion.Height != 0 {
			srcRegion.X += oxf
			srcRegion.Y += oyf
		}
	} else {
		n := len(vertices)
		for i := 0; i < n; i += graphics.VertexFloatCount {
			vertices[i] = vertices[i] + dx
			vertices[i+1] = vertices[i+1] + dy
		}
	}

	var offsets [graphics.ShaderImageCount - 1][2]float32
	var imgs [graphics.ShaderImageCount]*restorable.Image
	for i, subimageOffset := range subimageOffsets {
		src := srcs[i+1]
		if src == nil {
			continue
		}
		ox, oy, _, _ := src.regionWithPadding()
		ps := src.paddingSize()
		offsets[i][0] = float32(ox+ps) - oxf + subimageOffset[0]
		offsets[i][1] = float32(oy+ps) - oyf + subimageOffset[1]
	}
	for i, src := range srcs {
		if src == nil {
			continue
		}
		imgs[i] = src.backend.restorable
	}

	i.backend.restorable.DrawTriangles(imgs, offsets, vertices, indices, blend, dstRegion, srcRegion, shader.shader, uniforms, evenOdd)

	for _, src := range srcs {
		if src == nil {
			continue
		}
		if !src.isOnSourceBackend() && src.canBePutOnAtlas() {
			// src might already registered, but assigning it again is not harmful.
			imagesToPutOnSourceBackend[src] = struct{}{}
		}
	}
}

// WritePixels replaces the pixels on the image.
func (i *Image) WritePixels(pix []byte, x, y, width, height int) {
	backendsM.Lock()
	defer backendsM.Unlock()
	i.writePixels(pix, x, y, width, height)
}

func (i *Image) writePixels(pix []byte, x, y, width, height int) {
	if i.disposed {
		panic("atlas: the image must not be disposed at writePixels")
	}

	if l := 4 * width * height; len(pix) != l {
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

	px, py, pw, ph := i.regionWithPadding()

	if x != 0 || y != 0 || width != i.width || height != i.height || i.paddingSize() == 0 {
		x += px + i.paddingSize()
		y += py + i.paddingSize()

		if pix == nil {
			i.backend.restorable.WritePixels(nil, x, y, width, height)
			return
		}

		// Copy pixels in the case when pix is modified before the graphics command is executed.
		pix2 := theTemporaryBytes.alloc(len(pix))
		copy(pix2, pix)
		i.backend.restorable.WritePixels(pix2, x, y, width, height)
		return
	}

	pixb := theTemporaryBytes.alloc(4 * pw * ph)

	// Clear the edges. pixb might not be zero-cleared.
	// TODO: These loops assume that paddingSize is 1.
	// TODO: Is clearing edges explicitly really needed?
	const paddingSize = 1
	if paddingSize != i.paddingSize() {
		panic(fmt.Sprintf("atlas: writePixels assumes the padding is always 1 but the actual padding was %d", i.paddingSize()))
	}
	rowPixels := 4 * pw
	for i := 0; i < rowPixels; i++ {
		pixb[i] = 0
		pixb[rowPixels*(ph-1)+i] = 0
	}
	for j := 1; j < ph-1; j++ {
		pixb[rowPixels*j] = 0
		pixb[rowPixels*j+1] = 0
		pixb[rowPixels*j+2] = 0
		pixb[rowPixels*j+3] = 0
		pixb[rowPixels*(j+1)-4] = 0
		pixb[rowPixels*(j+1)-3] = 0
		pixb[rowPixels*(j+1)-2] = 0
		pixb[rowPixels*(j+1)-1] = 0
	}

	// Copy the content.
	for j := 0; j < height; j++ {
		copy(pixb[4*((j+paddingSize)*pw+paddingSize):], pix[4*j*width:4*(j+1)*width])
	}

	x += px
	y += py
	i.backend.restorable.WritePixels(pixb, x, y, pw, ph)
}

func (i *Image) ReadPixels(graphicsDriver graphicsdriver.Graphics, pixels []byte, x, y, width, height int) error {
	backendsM.Lock()
	defer backendsM.Unlock()

	// In the tests, BeginFrame might not be called often and then images might not be disposed (#2292).
	// To prevent memory leaks, flush the deferred functions here.
	flushDeferred()

	if i.backend == nil || i.backend.restorable == nil {
		for i := range pixels {
			pixels[i] = 0
		}
		return nil
	}

	ps := i.paddingSize()
	ox, oy, _, _ := i.regionWithPadding()
	x += ox + ps
	y += oy + ps
	return i.backend.restorable.ReadPixels(graphicsDriver, pixels, x, y, width, height)
}

// MarkDisposed marks the image as disposed. The actual operation is deferred.
// MarkDisposed can be called from finalizers.
//
// A function from finalizer must not be blocked, but disposing operation can be blocked.
// Defer this operation until it becomes safe. (#913)
func (i *Image) MarkDisposed() {
	// As MarkDisposed can be invoked from finalizers, backendsM should not be used.
	deferredM.Lock()
	deferred = append(deferred, func() {
		i.dispose(true)
	})
	deferredM.Unlock()
}

func (i *Image) dispose(markDisposed bool) {
	defer func() {
		if markDisposed {
			i.disposed = true
		}
		i.backend = nil
		i.node = nil
		if markDisposed {
			runtime.SetFinalizer(i, nil)
		}
	}()

	i.resetUsedAsSourceCount()

	if i.disposed {
		return
	}

	if i.backend == nil {
		// Not allocated yet.
		return
	}

	if !i.isOnAtlas() {
		i.backend.restorable.Dispose()
		return
	}

	i.backend.page.Free(i.node)
	if !i.backend.page.IsEmpty() {
		// As this part can be reused, this should be cleared explicitly.
		i.backend.restorable.ClearPixels(i.regionWithPadding())
		return
	}

	i.backend.restorable.Dispose()

	delete(theSourceBackendsForOneFrame, i.backend)

	for idx, sh := range theBackends {
		if sh == i.backend {
			theBackends = append(theBackends[:idx], theBackends[idx+1:]...)
			return
		}
	}

	panic("atlas: backend not found at an image being disposed")
}

func NewImage(width, height int, imageType ImageType) *Image {
	// Actual allocation is done lazily, and the lock is not needed.
	return &Image{
		width:     width,
		height:    height,
		imageType: imageType,
	}
}

func (i *Image) canBePutOnAtlas() bool {
	if minSourceSize == 0 || minDestinationSize == 0 || maxSize == 0 {
		panic("atlas: min*Size or maxSize must be initialized")
	}
	if i.imageType != ImageTypeRegular {
		return false
	}
	return i.width+2*i.paddingSize() <= maxSize && i.height+2*i.paddingSize() <= maxSize
}

func (i *Image) allocate(forbiddenBackends []*backend, asSource bool) {
	if i.backend != nil {
		panic("atlas: the image is already allocated")
	}

	runtime.SetFinalizer(i, (*Image).MarkDisposed)

	if i.imageType == ImageTypeScreen {
		if asSource {
			panic("atlas: a screen image cannot be created as a source")
		}
		// A screen image doesn't have a padding.
		i.backend = &backend{
			restorable: restorable.NewImage(i.width, i.height, restorable.ImageTypeScreen),
		}
		return
	}

	if !i.canBePutOnAtlas() {
		if i.width+2*i.paddingSize() > maxSize || i.height+2*i.paddingSize() > maxSize {
			panic(fmt.Sprintf("atlas: the image being put on an atlas is too big: width: %d, height: %d", i.width, i.height))
		}

		typ := restorable.ImageTypeRegular
		if i.imageType == ImageTypeVolatile {
			typ = restorable.ImageTypeVolatile
		}
		i.backend = &backend{
			restorable: restorable.NewImage(i.width+2*i.paddingSize(), i.height+2*i.paddingSize(), typ),
			source:     asSource && typ == restorable.ImageTypeRegular,
		}
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

		if n, ok := b.tryAlloc(i.width+2*i.paddingSize(), i.height+2*i.paddingSize()); ok {
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
	for i.width+2*i.paddingSize() > width {
		if width == maxSize {
			panic(fmt.Sprintf("atlas: the image being put on an atlas is too big: width: %d, height: %d", i.width, i.height))
		}
		width *= 2
	}
	for i.height+2*i.paddingSize() > height {
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

	n := b.page.Alloc(i.width+2*i.paddingSize(), i.height+2*i.paddingSize())
	if n == nil {
		panic("atlas: Alloc result must not be nil at allocate")
	}
	i.backend = b
	i.node = n
}

func (i *Image) DumpScreenshot(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool) (string, error) {
	backendsM.Lock()
	defer backendsM.Unlock()

	return i.backend.restorable.Dump(graphicsDriver, path, blackbg, image.Rect(i.paddingSize(), i.paddingSize(), i.width+i.paddingSize(), i.height+i.paddingSize()))
}

func EndFrame(graphicsDriver graphicsdriver.Graphics) error {
	backendsM.Lock()

	if err := restorable.ResolveStaleImages(graphicsDriver, true); err != nil {
		return err
	}

	theTemporaryBytes.resetAtFrameEnd()

	for b := range theSourceBackendsForOneFrame {
		delete(theSourceBackendsForOneFrame, b)
	}

	return nil
}

func floorPowerOf2(x int) int {
	if x <= 0 {
		return 0
	}
	p2 := 1
	for p2*2 <= x {
		p2 *= 2
	}
	return p2
}

func BeginFrame(graphicsDriver graphicsdriver.Graphics) error {
	defer backendsM.Unlock()

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
	if err := putImagesOnSourceBackend(graphicsDriver); err != nil {
		return err
	}

	return nil
}

func DumpImages(graphicsDriver graphicsdriver.Graphics, dir string) (string, error) {
	backendsM.Lock()
	defer backendsM.Unlock()
	return restorable.DumpImages(graphicsDriver, dir)
}
