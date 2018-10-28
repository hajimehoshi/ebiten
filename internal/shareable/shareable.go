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
	"image"
	"image/color"
	"runtime"
	"sync"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphicsutil"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/packing"
	"github.com/hajimehoshi/ebiten/internal/restorable"
)

type backend struct {
	restorable *restorable.Image

	// If page is nil, the backend is not shared.
	page *packing.Page
}

func (b *backend) TryAlloc(width, height int) (*packing.Node, bool) {
	// If the region is allocated without any extention, it's fine.
	if n := b.page.Alloc(width, height); n != nil {
		return n, true
	}

	// Simulate the extending the page and calculate the appropriate page size.
	page := b.page.Clone()
	nExtended := 0
	for {
		if !page.Extend() {
			// The page can't be extended any more. Return as failure.
			return nil, false
		}
		nExtended++
		if n := page.Alloc(width, height); n != nil {
			break
		}
	}

	for i := 0; i < nExtended; i++ {
		b.page.Extend()
	}
	s := b.page.Size()
	newImg := restorable.NewImage(s, s, false)
	oldImg := b.restorable
	w, h := oldImg.Size()
	vs := graphicsutil.QuadVertices(w, h, 0, 0, w, h, 1, 0, 0, 1, 0, 0, 1, 1, 1, 1)
	is := graphicsutil.QuadIndices()
	newImg.DrawImage(oldImg, vs, is, nil, opengl.CompositeModeCopy, graphics.FilterNearest)
	oldImg.Dispose()
	b.restorable = newImg

	n := b.page.Alloc(width, height)
	if n == nil {
		panic("not reached")
	}
	return n, true
}

var (
	// backendsM is a mutex for critical sections of the backend and packing.Node objects.
	backendsM sync.Mutex

	// theBackends is a set of actually shared images.
	theBackends = []*backend{}
)

type Image struct {
	width    int
	height   int
	disposed bool

	backend *backend

	node          *packing.Node
	countForShare int

	neverShared bool
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

func (i *Image) IsSharedForTesting() bool {
	backendsM.Lock()
	defer backendsM.Unlock()
	return i.isShared()
}

func (i *Image) ensureNotShared() {
	if i.backend == nil {
		i.allocate(false)
		return
	}

	if !i.isShared() {
		return
	}

	x, y, w, h := i.region()
	newImg := restorable.NewImage(w, h, false)
	vw, vh := i.backend.restorable.Size()
	vs := graphicsutil.QuadVertices(vw, vh, x, y, x+w, y+h, 1, 0, 0, 1, 0, 0, 1, 1, 1, 1)
	is := graphicsutil.QuadIndices()
	newImg.DrawImage(i.backend.restorable, vs, is, nil, opengl.CompositeModeCopy, graphics.FilterNearest)

	i.dispose(false)
	i.backend = &backend{
		restorable: newImg,
	}
}

func (i *Image) forceShared() {
	if i.backend == nil {
		i.allocate(true)
		return
	}

	if i.isShared() {
		return
	}

	if !i.shareable() {
		panic("not reached")
	}

	newI := NewImage(i.width, i.height)
	pixels := make([]byte, 4*i.width*i.height)
	for y := 0; y < i.height; y++ {
		for x := 0; x < i.width; x++ {
			c := i.at(x, y)
			pixels[4*(x+i.width*y)] = c.R
			pixels[4*(x+i.width*y)+1] = c.G
			pixels[4*(x+i.width*y)+2] = c.B
			pixels[4*(x+i.width*y)+3] = c.A
		}
	}
	newI.replacePixels(pixels)
	newI.moveTo(i)
	i.countForShare = 0
}

func (i *Image) region() (x, y, width, height int) {
	if i.backend == nil {
		panic("not reached")
	}
	if !i.isShared() {
		w, h := i.backend.restorable.Size()
		return 0, 0, w, h
	}
	return i.node.Region()
}

func (i *Image) Size() (width, height int) {
	return i.width, i.height
}

func (i *Image) QuadVertices(sx0, sy0, sx1, sy1 int, a, b, c, d, tx, ty float32, cr, cg, cb, ca float32) []float32 {
	if i.backend == nil {
		i.allocate(true)
	}
	ox, oy, _, _ := i.region()
	w, h := i.backend.restorable.SizePowerOf2()
	return graphicsutil.QuadVertices(w, h, sx0+ox, sy0+oy, sx1+ox, sy1+oy, a, b, c, d, tx, ty, cr, cg, cb, ca)
}

func (i *Image) Vertex(dx, dy, sx, sy float32, cr, cg, cb, ca float32) []float32 {
	if i.backend == nil {
		i.allocate(true)
	}
	ox, oy, _, _ := i.region()
	w, h := i.backend.restorable.SizePowerOf2()
	return graphicsutil.Vertex(w, h, dx, dy, sx+float32(ox), sy+float32(oy), cr, cg, cb, ca)
}

const MaxCountForShare = 10

func (i *Image) DrawImage(img *Image, vertices []float32, indices []uint16, colorm *affine.ColorM, mode opengl.CompositeMode, filter graphics.Filter) {
	backendsM.Lock()
	defer backendsM.Unlock()

	if img.disposed {
		panic("shareable: the drawing source image must not be disposed")
	}
	if i.disposed {
		panic("shareable: the drawing target image must not be disposed")
	}
	if img.backend == nil {
		img.allocate(true)
	}

	i.ensureNotShared()

	// Compare i and img after ensuring i is not shared, or
	// i and img might share the same texture even though i != img.
	if i.backend.restorable == img.backend.restorable {
		panic("shareable: Image.DrawImage: img must be different from the receiver")
	}

	i.backend.restorable.DrawImage(img.backend.restorable, vertices, indices, colorm, mode, filter)

	i.countForShare = 0

	// TODO: Reusing shared images is temporarily suspended for performance. See #661.
	//
	// if !img.isShared() && img.shareable() {
	//	img.countForShare++
	//	if img.countForShare >= MaxCountForShare {
	//		img.forceShared()
	//		img.countForShare = 0
	//	}
	// }
}

func (i *Image) ReplacePixels(p []byte) {
	backendsM.Lock()
	defer backendsM.Unlock()
	i.replacePixels(p)
}

func (i *Image) replacePixels(p []byte) {
	if i.disposed {
		panic("shareable: the image must not be disposed")
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
			panic(fmt.Sprintf("shareable: len(p) was %d but must be %d", len(p), l))
		}
	}
	i.backend.restorable.ReplacePixels(p, x, y, w, h)
}

func (i *Image) At(x, y int) color.RGBA {
	backendsM.Lock()
	defer backendsM.Unlock()
	return i.at(x, y)
}

func (i *Image) at(x, y int) color.RGBA {
	if i.backend == nil {
		return color.RGBA{}
	}

	ox, oy, w, h := i.region()
	if x < 0 || y < 0 || x >= w || y >= h {
		return color.RGBA{}
	}

	return i.backend.restorable.At(x+ox, y+oy)
}

func (i *Image) Dispose() {
	backendsM.Lock()
	defer backendsM.Unlock()
	i.dispose(true)
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
		x0, y0, x1, y1 := i.region()
		i.backend.restorable.ReplacePixels(nil, x0, y0, x1, y1)
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
		panic("not reached")
	}
	theBackends = append(theBackends[:index], theBackends[index+1:]...)
}

func (i *Image) IsVolatile() bool {
	backendsM.Lock()
	defer backendsM.Unlock()
	if i.backend == nil {
		// Not allocated yet. Only non-volatile images can do lazy allocation so far.
		return false
	}
	return i.backend.restorable.IsVolatile()
}

func (i *Image) IsInvalidated() (bool, error) {
	backendsM.Lock()
	defer backendsM.Unlock()
	v, err := i.backend.restorable.IsInvalidated()
	return v, err
}

func NewImage(width, height int) *Image {
	// Actual allocation is done lazily.
	return &Image{
		width:  width,
		height: height,
	}
}

func (i *Image) shareable() bool {
	if i.neverShared {
		return false
	}
	return i.width <= maxSize && i.height <= maxSize
}

func (i *Image) allocate(shareable bool) {
	if i.backend != nil {
		panic("not reached")
	}

	if !shareable || !i.shareable() {
		i.backend = &backend{
			restorable: restorable.NewImage(i.width, i.height, false),
		}
		runtime.SetFinalizer(i, (*Image).Dispose)
		return
	}

	for _, b := range theBackends {
		if n, ok := b.TryAlloc(i.width, i.height); ok {
			i.backend = b
			i.node = n
			runtime.SetFinalizer(i, (*Image).Dispose)
			return
		}
	}
	size := initSize
	for i.width > size || i.height > size {
		if size == maxSize {
			panic("not reached")
		}
		size *= 2
	}

	b := &backend{
		restorable: restorable.NewImage(size, size, false),
		page:       packing.NewPage(size, maxSize),
	}
	theBackends = append(theBackends, b)

	n := b.page.Alloc(i.width, i.height)
	if n == nil {
		panic("not reached")
	}
	i.backend = b
	i.node = n
	runtime.SetFinalizer(i, (*Image).Dispose)
	return
}

func NewVolatileImage(width, height int) *Image {
	backendsM.Lock()
	defer backendsM.Unlock()

	r := restorable.NewImage(width, height, true)
	i := &Image{
		width:  width,
		height: height,
		backend: &backend{
			restorable: r,
		},
		neverShared: true,
	}
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

func NewScreenFramebufferImage(width, height int) *Image {
	backendsM.Lock()
	defer backendsM.Unlock()

	r := restorable.NewScreenFramebufferImage(width, height)
	i := &Image{
		width:  width,
		height: height,
		backend: &backend{
			restorable: r,
		},
		neverShared: true,
	}
	runtime.SetFinalizer(i, (*Image).Dispose)
	return i
}

func InitializeGLState() error {
	backendsM.Lock()
	defer backendsM.Unlock()
	return restorable.InitializeGLState()
}

func ResolveStaleImages() {
	backendsM.Lock()
	defer backendsM.Unlock()
	restorable.ResolveStaleImages()
}

func IsRestoringEnabled() bool {
	// As IsRestoringEnabled is an immutable state, no need to lock here.
	return restorable.IsRestoringEnabled()
}

func Restore() error {
	backendsM.Lock()
	defer backendsM.Unlock()
	return restorable.Restore()
}

func Images() []image.Image {
	backendsM.Lock()
	defer backendsM.Unlock()
	return restorable.Images()
}

func Error() error {
	backendsM.Lock()
	defer backendsM.Unlock()
	return restorable.Error()
}
