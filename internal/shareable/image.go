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

package shareable

import (
	"fmt"
	"image/color"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/packing"
	"github.com/hajimehoshi/ebiten/internal/restorable"
)

const (
	// paddingSize represents the size of padding around an image.
	// Every image or node except for a screen image has its padding.
	paddingSize = 1
)

var graphicsDriver driver.Graphics

func SetGraphicsDriver(graphics driver.Graphics) {
	graphicsDriver = graphics
}

var (
	minSize = 0
	maxSize = 0
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func init() {
	hooks.AppendHookOnBeforeUpdate(func() error {
		backendsM.Lock()
		defer backendsM.Unlock()

		resolveDeferred()
		return makeImagesShared()
	})
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

// MaxCountForShare represents the time duration when the image can become shared.
//
// This value is exported for testing.
const MaxCountForShare = 10

func makeImagesShared() error {
	for i := range imagesToMakeShared {
		i.nonUpdatedCount++
		if i.nonUpdatedCount >= MaxCountForShare {
			if err := i.makeShared(); err != nil {
				return err
			}
		}
		delete(imagesToMakeShared, i)
	}
	return nil
}

type backend struct {
	restorable *restorable.Image

	// If page is nil, the backend is not shared.
	page *packing.Page
}

func (b *backend) TryAlloc(width, height int) (*packing.Node, bool) {
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
		panic("shareable: Alloc result must not be nil at TryAlloc")
	}
	return n, true
}

var (
	// backendsM is a mutex for critical sections of the backend and packing.Node objects.
	backendsM sync.Mutex

	initOnce sync.Once

	// theBackends is a set of actually shared images.
	theBackends = []*backend{}

	imagesToMakeShared = map[*Image]struct{}{}

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

type Image struct {
	width    int
	height   int
	disposed bool
	volatile bool
	screen   bool

	backend *backend

	node *packing.Node

	// nonUpdatedCount represents how long the image is kept not modified with DrawTriangles.
	// In the current implementation, if an image is being modified by DrawTriangles, the image is separated from
	// a shared (restorable) image by ensureNotShared.
	//
	// nonUpdatedCount is increased every frame if the image is not modified, or set to 0 if the image is
	// modified.
	//
	// ReplacePixels doesn't affect this value since ReplacePixels can be done on shared images.
	nonUpdatedCount int
}

func (i *Image) moveTo(dst *Image) {
	dst.dispose(false)
	*dst = *i

	// i is no longer available but Dispose must not be called
	// since i and dst have the same values like node.
	runtime.SetFinalizer(i, nil)
}

func (i *Image) isShared() bool {
	return i.node != nil
}

func (i *Image) ensureNotShared() {
	if i.backend == nil {
		i.allocate(false)
		return
	}

	if !i.isShared() {
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
	newImg.DrawTriangles(srcs, offsets, vs, is, nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressUnsafe, driver.Region{}, nil, nil)

	i.dispose(false)
	i.backend = &backend{
		restorable: newImg,
	}
}

func (i *Image) makeShared() error {
	if i.backend == nil {
		i.allocate(true)
		return nil
	}

	if i.isShared() {
		return nil
	}

	if !i.shareable() {
		panic("shareable: makeShared cannot be called on a non-shareable image")
	}

	newI := NewImage(i.width, i.height)
	newI.SetVolatile(i.volatile)
	pixels := make([]byte, 4*i.width*i.height)
	for y := 0; y < i.height; y++ {
		for x := 0; x < i.width; x++ {
			r, g, b, a, err := i.at(x+paddingSize, y+paddingSize)
			if err != nil {
				return err
			}
			pixels[4*(i.width*y+x)] = r
			pixels[4*(i.width*y+x)+1] = g
			pixels[4*(i.width*y+x)+2] = b
			pixels[4*(i.width*y+x)+3] = a
		}
	}
	newI.replacePixels(pixels)
	newI.moveTo(i)
	i.nonUpdatedCount = 0
	return nil
}

func (i *Image) regionWithPadding() (x, y, width, height int) {
	if i.backend == nil {
		panic("shareable: backend must not be nil: not allocated yet?")
	}
	if !i.isShared() {
		return 0, 0, i.width + 2*paddingSize, i.height + 2*paddingSize
	}
	return i.node.Region()
}

func (i *Image) processSrc(src *Image) {
	if src == nil {
		return
	}
	if src.disposed {
		panic("shareable: the drawing source image must not be disposed (DrawTriangles)")
	}
	if src.backend == nil {
		src.allocate(true)
	}

	// Compare i and source images after ensuring i is not shared, or
	// i and a source image might share the same texture even though i != src.
	if i.backend.restorable == src.backend.restorable {
		panic("shareable: Image.DrawTriangles: source must be different from the receiver")
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
func (i *Image) DrawTriangles(srcs [graphics.ShaderImageNum]*Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address, sourceRegion driver.Region, subimageOffsets [graphics.ShaderImageNum - 1][2]float32, shader *Shader, uniforms []interface{}) {
	backendsM.Lock()
	// Do not use defer for performance.

	if i.disposed {
		panic("shareable: the drawing target image must not be disposed (DrawTriangles)")
	}
	i.ensureNotShared()

	for _, src := range srcs {
		i.processSrc(src)
	}

	var dx, dy float32
	// A screen image doesn't have its padding.
	if !i.screen {
		dx = paddingSize
		dy = paddingSize
	}

	var oxf, oyf float32
	if srcs[0] != nil {
		ox, oy, _, _ := srcs[0].regionWithPadding()
		ox += paddingSize
		oy += paddingSize
		oxf, oyf = float32(ox), float32(oy)
		n := len(vertices) / graphics.VertexFloatNum
		for i := 0; i < n; i++ {
			vertices[i*graphics.VertexFloatNum+0] += dx
			vertices[i*graphics.VertexFloatNum+1] += dy
			vertices[i*graphics.VertexFloatNum+2] += oxf
			vertices[i*graphics.VertexFloatNum+3] += oyf
		}
		// sourceRegion can be delibarately empty when this is not needed in order to avoid unexpected
		// performance issue (#1293).
		if sourceRegion.Width != 0 && sourceRegion.Height != 0 {
			sourceRegion.X += oxf
			sourceRegion.Y += oyf
		}
	} else {
		n := len(vertices) / graphics.VertexFloatNum
		for i := 0; i < n; i++ {
			vertices[i*graphics.VertexFloatNum+0] += dx
			vertices[i*graphics.VertexFloatNum+1] += dy
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

	i.backend.restorable.DrawTriangles(imgs, offsets, vertices, indices, colorm, mode, filter, address, sourceRegion, s, uniforms)

	i.nonUpdatedCount = 0
	delete(imagesToMakeShared, i)

	for _, src := range srcs {
		if src == nil {
			continue
		}
		if !src.isShared() && src.shareable() {
			imagesToMakeShared[src] = struct{}{}
		}
	}

	backendsM.Unlock()
}

func (i *Image) Fill(clr color.RGBA) {
	backendsM.Lock()
	defer backendsM.Unlock()

	if i.disposed {
		panic("shareable: the drawing target image must not be disposed (Fill)")
	}
	if i.backend == nil {
		if _, _, _, a := clr.RGBA(); a == 0 {
			return
		}
	}

	i.ensureNotShared()

	// As *restorable.Image is an independent image, it is fine to fill the entire image.
	i.backend.restorable.Fill(clr)

	i.nonUpdatedCount = 0
	delete(imagesToMakeShared, i)
}

func (i *Image) ReplacePixels(pix []byte) {
	backendsM.Lock()
	defer backendsM.Unlock()
	i.replacePixels(pix)
}

func (i *Image) replacePixels(pix []byte) {
	if i.disposed {
		panic("shareable: the image must not be disposed at replacePixels")
	}
	if i.backend == nil {
		if pix == nil {
			return
		}
		i.allocate(true)
	}

	x, y, w, h := i.regionWithPadding()
	if pix == nil {
		i.backend.restorable.ReplacePixels(nil, x, y, w, h)
		return
	}

	ow, oh := w-2*paddingSize, h-2*paddingSize
	if l := 4 * ow * oh; len(pix) != l {
		panic(fmt.Sprintf("shareable: len(p) must be %d but %d", l, len(pix)))
	}

	// Add a padding around the image.
	pixb := make([]byte, 4*w*h)
	for j := 0; j < oh; j++ {
		copy(pixb[4*((j+paddingSize)*w+paddingSize):], pix[4*j*ow:4*(j+1)*ow])
	}

	i.backend.restorable.ReplacePixels(pixb, x, y, w, h)
}

func (img *Image) Pixels(x, y, width, height int) ([]byte, error) {
	backendsM.Lock()
	defer backendsM.Unlock()

	x += paddingSize
	y += paddingSize

	bs := make([]byte, 4*width*height)
	idx := 0
	for j := y; j < y+height; j++ {
		for i := x; i < x+width; i++ {
			r, g, b, a, err := img.at(i, j)
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

func (i *Image) at(x, y int) (byte, byte, byte, byte, error) {
	if i.backend == nil {
		return 0, 0, 0, 0, nil
	}

	ox, oy, w, h := i.regionWithPadding()
	if x < 0 || y < 0 || x >= w || y >= h {
		return 0, 0, 0, 0, nil
	}

	return i.backend.restorable.At(x+ox, y+oy)
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

	if i.disposed {
		return
	}

	if i.backend == nil {
		// Not allocated yet.
		return
	}

	if !i.isShared() {
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
		panic("shareable: backend not found at an image being disposed")
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

func (i *Image) SetVolatile(volatile bool) {
	i.volatile = volatile
	if i.backend == nil {
		return
	}
	if i.volatile {
		i.ensureNotShared()
	}
	i.backend.restorable.SetVolatile(i.volatile)
}

func (i *Image) shareable() bool {
	if minSize == 0 || maxSize == 0 {
		panic("shareable: minSize or maxSize must be initialized")
	}
	if i.volatile {
		return false
	}
	if i.screen {
		return false
	}
	return i.width+2*paddingSize <= maxSize && i.height+2*paddingSize <= maxSize
}

func (i *Image) allocate(shareable bool) {
	if i.backend != nil {
		panic("shareable: the image is already allocated")
	}

	runtime.SetFinalizer(i, (*Image).MarkDisposed)

	if i.screen {
		// A screen image doesn't have a padding.
		i.backend = &backend{
			restorable: restorable.NewScreenFramebufferImage(i.width, i.height),
		}
		return
	}

	if !shareable || !i.shareable() {
		i.backend = &backend{
			restorable: restorable.NewImage(i.width+2*paddingSize, i.height+2*paddingSize),
		}
		i.backend.restorable.SetVolatile(i.volatile)
		return
	}

	for _, b := range theBackends {
		if n, ok := b.TryAlloc(i.width+2*paddingSize, i.height+2*paddingSize); ok {
			i.backend = b
			i.node = n
			return
		}
	}
	size := minSize
	for i.width+2*paddingSize > size || i.height+2*paddingSize > size {
		if size == maxSize {
			panic(fmt.Sprintf("shareable: the image being shared is too big: width: %d, height: %d", i.width, i.height))
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
		panic("shareable: Alloc result must not be nil at allocate")
	}
	i.backend = b
	i.node = n
}

func (i *Image) Dump(path string, blackbg bool) error {
	backendsM.Lock()
	defer backendsM.Unlock()

	return i.backend.restorable.Dump(path, blackbg)
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

func EndFrame() error {
	backendsM.Lock()

	return restorable.ResolveStaleImages()
}

func BeginFrame() error {
	defer backendsM.Unlock()

	var err error
	initOnce.Do(func() {
		err = restorable.InitializeGraphicsDriverState()
		if err != nil {
			return
		}
		if len(theBackends) != 0 {
			panic("shareable: all the images must be not-shared before the game starts")
		}
		minSize = 1024
		maxSize = max(minSize, graphicsDriver.MaxImageSize())
	})
	if err != nil {
		return err
	}

	return restorable.RestoreIfNeeded()
}

func DumpImages(dir string) error {
	backendsM.Lock()
	defer backendsM.Unlock()
	return restorable.DumpImages(dir)
}
