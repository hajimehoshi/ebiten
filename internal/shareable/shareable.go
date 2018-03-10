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

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/opengl"
	"github.com/hajimehoshi/ebiten/internal/packing"
	"github.com/hajimehoshi/ebiten/internal/restorable"
	"github.com/hajimehoshi/ebiten/internal/sync"
)

type Image struct {
	restorable *restorable.Image
	page       *packing.Page
}

var (
	// theSharedImages is a set of actually shared images.
	theSharedImages = []*Image{}
)

type ImagePart struct {
	shareableImage *Image

	// If node is nil, the image is not shared.
	node *packing.Node
}

func (s *ImagePart) ensureNotShared() {
	if s.node == nil {
		return
	}

	x, y, w, h := s.region()
	newImg := restorable.NewImage(w, h, false)
	newImg.DrawImage(s.shareableImage.restorable, x, y, w, h, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)

	s.Dispose()
	s.shareableImage = &Image{
		restorable: newImg,
	}
}

func (s *ImagePart) region() (x, y, width, height int) {
	if s.node == nil {
		w, h := s.shareableImage.restorable.Size()
		return 0, 0, w, h
	}
	return s.node.Region()
}

func (s *ImagePart) Size() (width, height int) {
	_, _, w, h := s.region()
	return w, h
}

func (s *ImagePart) DrawImage(img *ImagePart, sx0, sy0, sx1, sy1 int, geom *affine.GeoM, colorm *affine.ColorM, mode opengl.CompositeMode, filter graphics.Filter) {
	s.ensureNotShared()

	// Compare i and img after ensuring i is not shared, or
	// i and img might share the same texture even though i != img.
	if s.shareableImage.restorable == img.shareableImage.restorable {
		panic("shareable: Image.DrawImage: img must be different from the receiver")
	}

	dx, dy, _, _ := img.region()
	sx0 += dx
	sy0 += dy
	sx1 += dx
	sy1 += dy
	s.shareableImage.restorable.DrawImage(img.shareableImage.restorable, sx0, sy0, sx1, sy1, geom, colorm, mode, filter)
}

func (s *ImagePart) ReplacePixels(p []byte) {
	x, y, w, h := s.region()
	if l := 4 * w * h; len(p) != l {
		panic(fmt.Sprintf("shareable: len(p) was %d but must be %d", len(p), l))
	}
	s.shareableImage.restorable.ReplacePixels(p, x, y, w, h)
}

func (s *ImagePart) At(x, y int) (color.Color, error) {
	ox, oy, w, h := s.region()
	if x < 0 || y < 0 || x >= w || y >= h {
		return color.RGBA{}, nil
	}
	return s.shareableImage.restorable.At(x+ox, y+oy)
}

func (s *ImagePart) isDisposed() bool {
	return s.shareableImage == nil
}

func (s *ImagePart) Dispose() {
	if s.isDisposed() {
		return
	}

	defer func() {
		s.shareableImage = nil
		s.node = nil
	}()

	if s.node == nil {
		s.shareableImage.restorable.Dispose()
		return
	}

	s.shareableImage.page.Free(s.node)
	if !s.shareableImage.page.IsEmpty() {
		return
	}

	index := -1
	for i, sh := range theSharedImages {
		if sh == s.shareableImage {
			index = i
			break
		}
	}
	if index == -1 {
		panic("not reached")
	}
	theSharedImages = append(theSharedImages[:index], theSharedImages[index+1:]...)
}

func (s *ImagePart) IsInvalidated() (bool, error) {
	return s.shareableImage.restorable.IsInvalidated()
}

var shareableImageLock sync.Mutex

func NewImagePart(width, height int) *ImagePart {
	const maxSize = 2048

	shareableImageLock.Lock()
	defer shareableImageLock.Unlock()

	if width > maxSize || height > maxSize {
		s := &Image{
			restorable: restorable.NewImage(width, height, false),
		}
		return &ImagePart{
			shareableImage: s,
		}
	}

	for _, s := range theSharedImages {
		if n := s.page.Alloc(width, height); n != nil {
			return &ImagePart{
				shareableImage: s,
				node:           n,
			}
		}
	}
	s := &Image{
		restorable: restorable.NewImage(maxSize, maxSize, false),
		page:       packing.NewPage(maxSize),
	}
	theSharedImages = append(theSharedImages, s)

	n := s.page.Alloc(width, height)
	if n == nil {
		panic("not reached")
	}
	return &ImagePart{
		shareableImage: s,
		node:           n,
	}
}

func NewVolatileImagePart(width, height int) *ImagePart {
	r := restorable.NewImage(width, height, true)
	return &ImagePart{
		shareableImage: &Image{
			restorable: r,
		},
	}
}

func NewScreenFramebufferImagePart(width, height int) *ImagePart {
	r := restorable.NewScreenFramebufferImage(width, height)
	return &ImagePart{
		shareableImage: &Image{
			restorable: r,
		},
	}
}
