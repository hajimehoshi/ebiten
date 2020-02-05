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

var graphicsDriver driver.Graphics

func SetGraphicsDriver(graphics driver.Graphics) {
	graphicsDriver = graphics
}

var (
	minSize = 0
	maxSize = 0
)

func min(a, b int) int {
	if a < b {
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

	ox, oy, w, h := i.region()
	dx0 := float32(0)
	dy0 := float32(0)
	dx1 := float32(w)
	dy1 := float32(h)
	sx0 := float32(ox)
	sy0 := float32(oy)
	sx1 := float32(ox + w)
	sy1 := float32(oy + h)
	newImg := restorable.NewImage(w, h, i.volatile)
	vs := []float32{
		dx0, dy0, sx0, sy0, sx0, sy0, sx1, sy1, 1, 1, 1, 1,
		dx1, dy0, sx1, sy0, sx0, sy0, sx1, sy1, 1, 1, 1, 1,
		dx0, dy1, sx0, sy1, sx0, sy0, sx1, sy1, 1, 1, 1, 1,
		dx1, dy1, sx1, sy1, sx0, sy0, sx1, sy1, 1, 1, 1, 1,
	}
	is := graphics.QuadIndices()
	newImg.DrawTriangles(i.backend.restorable, vs, is, nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressClampToZero)

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

	newI := NewImage(i.width, i.height, i.volatile)
	pixels := make([]byte, 4*i.width*i.height)
	for y := 0; y < i.height; y++ {
		for x := 0; x < i.width; x++ {
			r, g, b, a, err := i.at(x, y)
			if err != nil {
				return err
			}
			pixels[4*(x+i.width*y)] = r
			pixels[4*(x+i.width*y)+1] = g
			pixels[4*(x+i.width*y)+2] = b
			pixels[4*(x+i.width*y)+3] = a
		}
	}
	newI.replacePixels(pixels)
	newI.moveTo(i)
	i.nonUpdatedCount = 0
	return nil
}

func (i *Image) region() (x, y, width, height int) {
	if i.backend == nil {
		panic("shareable: backend must not be nil: not allocated yet?")
	}
	if !i.isShared() {
		return 0, 0, i.width, i.height
	}
	return i.node.Region()
}

// DrawTriangles draws triangles with the given image.
//
// The vertex floats are:
//
//   0:  Destination X in pixels
//   1:  Destination Y in pixels
//   2:  Source X in pixels (the upper-left is (0, 0))
//   3:  Source Y in pixels
//   4:  Bounds of the source min X in pixels
//   5:  Bounds of the source min Y in pixels
//   6:  Bounds of the source max X in pixels
//   7:  Bounds of the source max Y in pixels
//   8:  Color R [0.0-1.0]
//   9:  Color G
//   10: Color B
//   11: Color Y
func (i *Image) DrawTriangles(img *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode driver.CompositeMode, filter driver.Filter, address driver.Address) {
	backendsM.Lock()
	// Do not use defer for performance.

	if img.disposed {
		panic("shareable: the drawing source image must not be disposed (DrawTriangles)")
	}
	if i.disposed {
		panic("shareable: the drawing target image must not be disposed (DrawTriangles)")
	}
	if img.backend == nil {
		img.allocate(true)
	}

	i.ensureNotShared()

	// Compare i and img after ensuring i is not shared, or
	// i and img might share the same texture even though i != img.
	if i.backend.restorable == img.backend.restorable {
		panic("shareable: Image.DrawTriangles: img must be different from the receiver")
	}

	ox, oy, _, _ := img.region()
	oxf, oyf := float32(ox), float32(oy)
	n := len(vertices) / graphics.VertexFloatNum
	for i := 0; i < n; i++ {
		vertices[i*graphics.VertexFloatNum+2] += oxf
		vertices[i*graphics.VertexFloatNum+3] += oyf
		vertices[i*graphics.VertexFloatNum+4] += oxf
		vertices[i*graphics.VertexFloatNum+5] += oyf
		vertices[i*graphics.VertexFloatNum+6] += oxf
		vertices[i*graphics.VertexFloatNum+7] += oyf
	}

	i.backend.restorable.DrawTriangles(img.backend.restorable, vertices, indices, colorm, mode, filter, address)

	i.nonUpdatedCount = 0
	delete(imagesToMakeShared, i)

	if !img.isShared() && img.shareable() {
		imagesToMakeShared[img] = struct{}{}
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

func (i *Image) ReplacePixels(p []byte) {
	backendsM.Lock()
	defer backendsM.Unlock()
	i.replacePixels(p)
}

func (i *Image) replacePixels(p []byte) {
	if i.disposed {
		panic("shareable: the image must not be disposed at replacePixels")
	}
	if i.backend == nil {
		if p == nil {
			return
		}
		i.allocate(true)
	}

	x, y, w, h := i.region()
	if p != nil {
		if l := 4 * w * h; len(p) != l {
			panic(fmt.Sprintf("shareable: len(p) must be %d but %d", l, len(p)))
		}
	}
	i.backend.restorable.ReplacePixels(p, x, y, w, h)
}

func (i *Image) At(x, y int) (byte, byte, byte, byte, error) {
	backendsM.Lock()
	r, g, b, a, err := i.at(x, y)
	backendsM.Unlock()
	return r, g, b, a, err
}

func (i *Image) at(x, y int) (byte, byte, byte, byte, error) {
	if i.backend == nil {
		return 0, 0, 0, 0, nil
	}

	ox, oy, w, h := i.region()
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
	// deferred doesn't have to be, and should not be protected by a mutex.
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
		i.backend.restorable.ClearPixels(i.region())
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

func NewImage(width, height int, volatile bool) *Image {
	// Actual allocation is done lazily, and the lock is not needed.
	return &Image{
		width:    width,
		height:   height,
		volatile: volatile,
	}
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
	return i.width <= maxSize && i.height <= maxSize
}

func (i *Image) allocate(shareable bool) {
	if i.backend != nil {
		panic("shareable: the image is already allocated")
	}

	runtime.SetFinalizer(i, (*Image).MarkDisposed)

	if i.screen {
		i.backend = &backend{
			restorable: restorable.NewScreenFramebufferImage(i.width, i.height),
		}
		return
	}

	if !shareable || !i.shareable() {
		i.backend = &backend{
			restorable: restorable.NewImage(i.width, i.height, i.volatile),
		}
		return
	}

	for _, b := range theBackends {
		if n, ok := b.TryAlloc(i.width, i.height); ok {
			i.backend = b
			i.node = n
			return
		}
	}
	size := minSize
	for i.width > size || i.height > size {
		if size == maxSize {
			panic(fmt.Sprintf("shareable: the image being shared is too big: width: %d, height: %d", i.width, i.height))
		}
		size *= 2
	}

	b := &backend{
		restorable: restorable.NewImage(size, size, i.volatile),
		page:       packing.NewPage(size, maxSize),
	}
	theBackends = append(theBackends, b)

	n := b.page.Alloc(i.width, i.height)
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
		if graphicsDriver.HasHighPrecisionFloat() {
			minSize = 1024
			// Use 4096 as a maximum size whatever size the graphics driver accepts. There are
			// not enough evidences that bigger textures works correctly.
			maxSize = min(4096, graphicsDriver.MaxImageSize())
		} else {
			minSize = 512
			maxSize = 512
		}
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
