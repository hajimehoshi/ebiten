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
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/affine"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/packing"
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
)

const (
	// paddingSize represents the size of padding around an image.
	// Every image or node except for a screen image has its padding.
	paddingSize = 1
)

var (
	minSize = 0
	maxSize = 0
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

// alloc allocates the pixels and reutrns it.
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

func resolveDeferred() {
	deferredM.Lock()
	fs := deferred
	deferred = nil
	deferredM.Unlock()

	for _, f := range fs {
		f()
	}
}

// baseCountToPutOnAtlas represents the base time duration when the image can be put onto an atlas.
// Actual time duration is increased in an exponential way for each usages as a rendering target.
const baseCountToPutOnAtlas = 10

func putImagesOnAtlas(graphicsDriver graphicsdriver.Graphics) error {
	for i := range imagesToPutOnAtlas {
		i.usedAsSourceCount++
		if i.usedAsSourceCount >= baseCountToPutOnAtlas*(1<<uint(min(i.isolatedCount, 31))) {
			if err := i.putOnAtlas(graphicsDriver); err != nil {
				return err
			}
			i.usedAsSourceCount = 0
			delete(imagesToPutOnAtlas, i)
		}
	}

	// Reset the images. The images will be registered again when it is used as a rendering source.
	for k := range imagesToPutOnAtlas {
		delete(imagesToPutOnAtlas, k)
	}
	return nil
}

type backend struct {
	// restorable is an atlas on which there might be multiple images.
	restorable *restorable.Image

	// page is an atlas map. Each part is called a node.
	// If page is nil, the backend's image is isolated and not on an atlas.
	page *packing.Page
}

func (b *backend) tryAlloc(width, height int) (*packing.Node, bool) {
	// If the region is allocated without any extension, that's fine.
	if n := b.page.Alloc(width, height); n != nil {
		return n, true
	}

	nExtended := 1
	var n *packing.Node
	for {
		if !b.page.Extend(nExtended) {
			// The page can't be extended any more. Return as failure.
			return nil, false
		}
		nExtended++
		n = b.page.Alloc(width, height)
		if n != nil {
			b.page.CommitExtension()
			break
		}
		b.page.RollbackExtension()
	}

	s := b.page.Size()
	b.restorable = b.restorable.Extend(s, s)

	if n == nil {
		panic("atlas: Alloc result must not be nil at TryAlloc")
	}
	return n, true
}

var (
	// backendsM is a mutex for critical sections of the backend and packing.Node objects.
	backendsM sync.Mutex

	initOnce sync.Once

	// theBackends is a set of atlases.
	theBackends = []*backend{}

	imagesToPutOnAtlas = map[*Image]struct{}{}

	deferred []func()

	// deferredM is a mutext for the slice operations. This must not be used for other usages.
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

// Image is a rectangle pixel set that might be on an atlas.
type Image struct {
	width       int
	height      int
	disposed    bool
	independent bool
	volatile    bool
	screen      bool

	backend *backend

	node *packing.Node

	// usedAsSourceCount represents how long the image is used as a rendering source and kept not modified with
	// DrawTriangles.
	// In the current implementation, if an image is being modified by DrawTriangles, the image is separated from
	// a restorable image on an atlas by ensureIsolated.
	//
	// usedAsSourceCount is increased if the image is used as a rendering source, or set to 0 if the image is
	// modified.
	//
	// ReplacePixels doesn't affect this value since ReplacePixels can be done on images on an atlas.
	usedAsSourceCount int

	// isolatedCount represents how many times the image on a texture atlas is changed into an isolated image.
	// isolatedCount affects the calculation when to put the image onto a texture atlas again.
	isolatedCount int
}

// moveTo moves its content to the given image dst.
// After moveTo is called, the image i is no longer available.
//
// moveTo is smilar to C++'s move semantics.
func (i *Image) moveTo(dst *Image) {
	dst.dispose(false)
	*dst = *i

	// i is no longer available but Dispose must not be called
	// since i and dst have the same values like node.
	runtime.SetFinalizer(i, nil)
}

func (i *Image) isOnAtlas() bool {
	return i.node != nil
}

func (i *Image) resetUsedAsSourceCount() {
	i.usedAsSourceCount = 0
	delete(imagesToPutOnAtlas, i)
}

func (i *Image) ensureIsolated() {
	i.resetUsedAsSourceCount()

	if i.backend == nil {
		i.allocate(false)
		return
	}

	if !i.isOnAtlas() {
		return
	}

	ox, oy, w, h := i.regionWithPadding()
	dx0 := float32(0)
	dy0 := float32(0)
	dx1 := float32(w)
	dy1 := float32(h)
	sx0 := float32(ox)
	sy0 := float32(oy)
	sx1 := float32(ox + w)
	sy1 := float32(oy + h)
	sw, sh := i.backend.restorable.InternalSize()
	sx0 /= float32(sw)
	sy0 /= float32(sh)
	sx1 /= float32(sw)
	sy1 /= float32(sh)
	newImg := restorable.NewImage(w, h)
	newImg.SetVolatile(i.volatile)
	vs := []float32{
		dx0, dy0, sx0, sy0, 1, 1, 1, 1,
		dx1, dy0, sx1, sy0, 1, 1, 1, 1,
		dx0, dy1, sx0, sy1, 1, 1, 1, 1,
		dx1, dy1, sx1, sy1, 1, 1, 1, 1,
	}
	is := graphics.QuadIndices()
	srcs := [graphics.ShaderImageNum]*restorable.Image{i.backend.restorable}
	var offsets [graphics.ShaderImageNum - 1][2]float32
	dstRegion := graphicsdriver.Region{
		X:      paddingSize,
		Y:      paddingSize,
		Width:  float32(w - 2*paddingSize),
		Height: float32(h - 2*paddingSize),
	}
	newImg.DrawTriangles(srcs, offsets, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dstRegion, graphicsdriver.Region{}, nil, nil, false)

	i.dispose(false)
	i.backend = &backend{
		restorable: newImg,
	}

	i.isolatedCount++
}

func (i *Image) putOnAtlas(graphicsDriver graphicsdriver.Graphics) error {
	if i.backend == nil {
		i.allocate(true)
		return nil
	}

	if i.isOnAtlas() {
		return nil
	}

	if !i.canBePutOnAtlas() {
		panic("atlas: putOnAtlas cannot be called on a image that cannot be on an atlas")
	}

	newI := NewImage(i.width, i.height)
	newI.SetVolatile(i.volatile)

	if restorable.NeedsRestoring() {
		// If the underlying graphics driver requires restoring from the context lost, the pixel data is
		// needed. An image on an atlas must have its complete pixel data in this case.
		pixels := make([]byte, 4*i.width*i.height)
		for y := 0; y < i.height; y++ {
			for x := 0; x < i.width; x++ {
				r, g, b, a, err := i.at(graphicsDriver, x+paddingSize, y+paddingSize)
				if err != nil {
					return err
				}
				pixels[4*(i.width*y+x)] = r
				pixels[4*(i.width*y+x)+1] = g
				pixels[4*(i.width*y+x)+2] = b
				pixels[4*(i.width*y+x)+3] = a
			}
		}
		newI.replacePixels(pixels, nil)
	} else {
		// If the underlying graphics driver doesn't require restoring from the context lost, just a regular
		// rendering works.
		w, h := float32(i.width), float32(i.height)
		vs := graphics.QuadVertices(0, 0, w, h, 1, 0, 0, 1, 0, 0, 1, 1, 1, 1)
		is := graphics.QuadIndices()
		dr := graphicsdriver.Region{
			X:      0,
			Y:      0,
			Width:  w,
			Height: h,
		}
		newI.drawTriangles([graphics.ShaderImageNum]*Image{i}, vs, is, affine.ColorMIdentity{}, graphicsdriver.CompositeModeCopy, graphicsdriver.FilterNearest, graphicsdriver.AddressUnsafe, dr, graphicsdriver.Region{}, [graphics.ShaderImageNum - 1][2]float32{}, nil, nil, false, true)
	}

	newI.moveTo(i)
	i.usedAsSourceCount = 0
	return nil
}

func (i *Image) regionWithPadding() (x, y, width, height int) {
	if i.backend == nil {
		panic("atlas: backend must not be nil: not allocated yet?")
	}
	if !i.isOnAtlas() {
		return 0, 0, i.width + 2*paddingSize, i.height + 2*paddingSize
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
		src.allocate(true)
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
//   0: Destination X in pixels
//   1: Destination Y in pixels
//   2: Source X in pixels (the upper-left is (0, 0))
//   3: Source Y in pixels
//   4: Color R [0.0-1.0]
//   5: Color G
//   6: Color B
//   7: Color Y
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageNum]*Image, vertices []float32, indices []uint16, colorm affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, subimageOffsets [graphics.ShaderImageNum - 1][2]float32, shader *Shader, uniforms [][]float32, evenOdd bool) {
	backendsM.Lock()
	defer backendsM.Unlock()
	i.drawTriangles(srcs, vertices, indices, colorm, mode, filter, address, dstRegion, srcRegion, subimageOffsets, shader, uniforms, evenOdd, false)
}

func (i *Image) drawTriangles(srcs [graphics.ShaderImageNum]*Image, vertices []float32, indices []uint16, colorm affine.ColorM, mode graphicsdriver.CompositeMode, filter graphicsdriver.Filter, address graphicsdriver.Address, dstRegion, srcRegion graphicsdriver.Region, subimageOffsets [graphics.ShaderImageNum - 1][2]float32, shader *Shader, uniforms [][]float32, evenOdd bool, keepOnAtlas bool) {
	if i.disposed {
		panic("atlas: the drawing target image must not be disposed (DrawTriangles)")
	}
	if keepOnAtlas {
		if i.backend == nil {
			i.allocate(true)
		}
	} else {
		i.ensureIsolated()
	}

	for _, src := range srcs {
		i.processSrc(src)
	}

	var dx, dy float32
	// A screen image doesn't have its padding.
	if !i.screen {
		x, y, _, _ := i.regionWithPadding()
		dx = float32(x) + paddingSize
		dy = float32(y) + paddingSize
		// TODO: Check if dstRegion does not to violate the region.
	}
	dstRegion.X += dx
	dstRegion.Y += dy

	var oxf, oyf float32
	if srcs[0] != nil {
		ox, oy, _, _ := srcs[0].regionWithPadding()
		ox += paddingSize
		oy += paddingSize
		oxf, oyf = float32(ox), float32(oy)
		sw, sh := srcs[0].backend.restorable.InternalSize()
		swf, shf := float32(sw), float32(sh)
		n := len(vertices)
		for i := 0; i < n; i += graphics.VertexFloatNum {
			vertices[i] = adjustDestinationPixel(vertices[i] + dx)
			vertices[i+1] = adjustDestinationPixel(vertices[i+1] + dy)
			vertices[i+2] = (vertices[i+2] + oxf) / swf
			vertices[i+3] = (vertices[i+3] + oyf) / shf
		}
		// srcRegion can be delibarately empty when this is not needed in order to avoid unexpected
		// performance issue (#1293).
		if srcRegion.Width != 0 && srcRegion.Height != 0 {
			srcRegion.X += oxf
			srcRegion.Y += oyf
		}
	} else {
		n := len(vertices)
		for i := 0; i < n; i += graphics.VertexFloatNum {
			vertices[i] = adjustDestinationPixel(vertices[i] + dx)
			vertices[i+1] = adjustDestinationPixel(vertices[i+1] + dy)
		}
	}

	var offsets [graphics.ShaderImageNum - 1][2]float32
	var s *restorable.Shader
	var imgs [graphics.ShaderImageNum]*restorable.Image
	if shader == nil {
		// Fast path for rendering without a shader (#1355).
		imgs[0] = srcs[0].backend.restorable
	} else {
		for i, subimageOffset := range subimageOffsets {
			src := srcs[i+1]
			if src == nil {
				continue
			}
			ox, oy, _, _ := src.regionWithPadding()
			offsets[i][0] = float32(ox) + paddingSize - oxf + subimageOffset[0]
			offsets[i][1] = float32(oy) + paddingSize - oyf + subimageOffset[1]
		}
		s = shader.shader
		for i, src := range srcs {
			if src == nil {
				continue
			}
			imgs[i] = src.backend.restorable
		}
	}

	i.backend.restorable.DrawTriangles(imgs, offsets, vertices, indices, colorm, mode, filter, address, dstRegion, srcRegion, s, uniforms, evenOdd)

	for _, src := range srcs {
		if src == nil {
			continue
		}
		if !src.isOnAtlas() && src.canBePutOnAtlas() {
			// src might already registered, but assiging it again is not harmful.
			imagesToPutOnAtlas[src] = struct{}{}
		}
	}
}

// ReplacePixels replaces the pixels on the image.
// ReplacePixels cannot take a region due to the current implementation.
// internal/restorable.Image has to record the areas of replaced pixels, and the areas must not be overlapped so far.
func (i *Image) ReplacePixels(pix []byte, mask []byte) {
	backendsM.Lock()
	defer backendsM.Unlock()
	i.replacePixels(pix, mask)
}

func (i *Image) replacePixels(pix []byte, mask []byte) {
	if i.disposed {
		panic("atlas: the image must not be disposed at replacePixels")
	}

	i.resetUsedAsSourceCount()

	if i.backend == nil {
		if pix == nil {
			return
		}
		i.allocate(true)
	}

	px, py, pw, ph := i.regionWithPadding()
	if pix == nil {
		if mask != nil {
			panic("atlas: mask must be nil when pix is nil")
		}
		i.backend.restorable.ReplacePixels(nil, nil, px, py, pw, ph)
		return
	}

	ow, oh := pw-2*paddingSize, ph-2*paddingSize
	if l := 4 * ow * oh; len(pix) != l {
		panic(fmt.Sprintf("atlas: len(p) must be %d but %d", l, len(pix)))
	}

	pixb := theTemporaryBytes.alloc(4 * pw * ph)

	// Clear the edges. pixb might not be zero-cleared.
	// TODO: These loops assume that paddingSize is 1.
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
	for j := 0; j < oh; j++ {
		copy(pixb[4*((j+paddingSize)*pw+paddingSize):], pix[4*j*ow:4*(j+1)*ow])
	}

	// Add the paddings to the mask if needed.
	if mask != nil {
		origMask := mask
		mask = theTemporaryBytes.alloc((pw*ph-1)/8 + 1)
		for i := 0; i < pw; i++ {
			// Top edge
			idx := i
			mask[idx/8] |= 1 << idx % 8
			// Bottom edge
			idx = (ph-1)*pw + i
			mask[idx/8] |= 1 << idx % 8
		}
		for j := 1; j < ph-1; j++ {
			// Left edge
			idx := j * pw
			mask[idx/8] |= 1 << idx % 8
			// Right edge
			idx = j*pw + pw - 1
			mask[idx/8] |= 1 << idx % 8

			// Content
			for i := 1; i < pw-1; i++ {
				idx := j*pw + i
				origIdx := (j-paddingSize)*(pw-paddingSize*2) + i - paddingSize
				origValue := (origMask[origIdx/8] >> (origIdx % 8)) & 1
				mask[idx/8] |= origValue << (idx % 8)
			}
		}
	}

	i.backend.restorable.ReplacePixels(pixb, mask, px, py, pw, ph)
}

func (img *Image) Pixels(graphicsDriver graphicsdriver.Graphics) ([]byte, error) {
	backendsM.Lock()
	defer backendsM.Unlock()

	x := paddingSize
	y := paddingSize

	bs := make([]byte, 4*img.width*img.height)
	idx := 0
	for j := y; j < y+img.height; j++ {
		for i := x; i < x+img.width; i++ {
			r, g, b, a, err := img.at(graphicsDriver, i, j)
			if err != nil {
				return nil, err
			}
			bs[4*idx] = r
			bs[4*idx+1] = g
			bs[4*idx+2] = b
			bs[4*idx+3] = a
			idx++
		}
	}
	return bs, nil
}

func (i *Image) at(graphicsDriver graphicsdriver.Graphics, x, y int) (byte, byte, byte, byte, error) {
	if i.backend == nil {
		return 0, 0, 0, 0, nil
	}

	ox, oy, w, h := i.regionWithPadding()
	if x < 0 || y < 0 || x >= w || y >= h {
		return 0, 0, 0, 0, nil
	}

	return i.backend.restorable.At(graphicsDriver, x+ox, y+oy)
}

// MarkDisposed marks the image as disposed. The actual operation is deferred.
// MarkDisposed can be called from finalizers.
//
// A function from finalizer must not be blocked, but disposing operation can be blocked.
// Defer this operation until it becomes safe. (#913)
func (i *Image) MarkDisposed() {
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
	index := -1
	for idx, sh := range theBackends {
		if sh == i.backend {
			index = idx
			break
		}
	}
	if index == -1 {
		panic("atlas: backend not found at an image being disposed")
	}
	theBackends = append(theBackends[:index], theBackends[index+1:]...)
}

func NewImage(width, height int) *Image {
	// Actual allocation is done lazily, and the lock is not needed.
	return &Image{
		width:  width,
		height: height,
	}
}

func (i *Image) SetIndependent(independent bool) {
	i.independent = independent
}

func (i *Image) SetVolatile(volatile bool) {
	i.volatile = volatile
	if i.backend == nil {
		return
	}
	if i.volatile {
		i.ensureIsolated()
	}
	i.backend.restorable.SetVolatile(i.volatile)
}

func (i *Image) canBePutOnAtlas() bool {
	if minSize == 0 || maxSize == 0 {
		panic("atlas: minSize or maxSize must be initialized")
	}
	if i.independent {
		return false
	}
	if i.volatile {
		return false
	}
	if i.screen {
		return false
	}
	return i.width+2*paddingSize <= maxSize && i.height+2*paddingSize <= maxSize
}

func (i *Image) allocate(putOnAtlas bool) {
	if i.backend != nil {
		panic("atlas: the image is already allocated")
	}

	runtime.SetFinalizer(i, (*Image).MarkDisposed)

	if i.screen {
		// A screen image doesn't have a padding.
		i.backend = &backend{
			restorable: restorable.NewScreenFramebufferImage(i.width, i.height),
		}
		return
	}

	if !putOnAtlas || !i.canBePutOnAtlas() {
		i.backend = &backend{
			restorable: restorable.NewImage(i.width+2*paddingSize, i.height+2*paddingSize),
		}
		i.backend.restorable.SetVolatile(i.volatile)
		return
	}

	for _, b := range theBackends {
		if n, ok := b.tryAlloc(i.width+2*paddingSize, i.height+2*paddingSize); ok {
			i.backend = b
			i.node = n
			return
		}
	}
	size := minSize
	for i.width+2*paddingSize > size || i.height+2*paddingSize > size {
		if size == maxSize {
			panic(fmt.Sprintf("atlas: the image being put on an atlas is too big: width: %d, height: %d", i.width, i.height))
		}
		size *= 2
	}

	b := &backend{
		restorable: restorable.NewImage(size, size),
		page:       packing.NewPage(size, maxSize),
	}
	b.restorable.SetVolatile(i.volatile)
	theBackends = append(theBackends, b)

	n := b.page.Alloc(i.width+2*paddingSize, i.height+2*paddingSize)
	if n == nil {
		panic("atlas: Alloc result must not be nil at allocate")
	}
	i.backend = b
	i.node = n
}

func (i *Image) DumpScreenshot(graphicsDriver graphicsdriver.Graphics, path string, blackbg bool) error {
	backendsM.Lock()
	defer backendsM.Unlock()

	return i.backend.restorable.Dump(graphicsDriver, path, blackbg, image.Rect(paddingSize, paddingSize, paddingSize+i.width, paddingSize+i.height))
}

func NewScreenFramebufferImage(width, height int) *Image {
	// Actual allocation is done lazily.
	i := &Image{
		width:  width,
		height: height,
		screen: true,
	}
	return i
}

func EndFrame(graphicsDriver graphicsdriver.Graphics) error {
	backendsM.Lock()

	theTemporaryBytes.resetAtFrameEnd()

	return restorable.ResolveStaleImages(graphicsDriver)
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
		minSize = 1024
		maxSize = restorable.MaxImageSize(graphicsDriver)
	})
	if err != nil {
		return err
	}

	// Restore images first before other image manipulations (#2075).
	if err := restorable.RestoreIfNeeded(graphicsDriver); err != nil {
		return err
	}

	resolveDeferred()
	if err := putImagesOnAtlas(graphicsDriver); err != nil {
		return err
	}

	return nil
}

func DumpImages(graphicsDriver graphicsdriver.Graphics, dir string) error {
	backendsM.Lock()
	defer backendsM.Unlock()
	return restorable.DumpImages(graphicsDriver, dir)
}

func adjustDestinationPixel(x float32) float32 {
	// Avoid the center of the pixel, which is problematic (#929, #1171).
	// Instead, align the vertices with about 1/3 pixels.
	ix := float32(math.Floor(float64(x)))
	frac := x - ix
	switch {
	case frac < 3.0/16.0:
		return ix
	case frac < 8.0/16.0:
		return ix + 5.0/16.0
	case frac < 13.0/16.0:
		return ix + 11.0/16.0
	default:
		return ix + 16.0/16.0
	}
}
