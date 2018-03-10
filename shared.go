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

package ebiten

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

type sharedImage struct {
	restorable *restorable.Image
	page       *packing.Page
}

var (
	theSharedImages = []*sharedImage{}
)

type sharedImagePart struct {
	sharedImage *sharedImage

	// If node is nil, the image is not shared.
	node *packing.Node
}

func (s *sharedImagePart) ensureNotShared() {
	if s.node == nil {
		return
	}

	x, y, w, h := s.region()
	newImg := restorable.NewImage(w, h, false)
	newImg.DrawImage(s.sharedImage.restorable, x, y, w, h, nil, nil, opengl.CompositeModeCopy, graphics.FilterNearest)

	s.Dispose()
	s.sharedImage = &sharedImage{
		restorable: newImg,
	}
}

func (s *sharedImagePart) region() (x, y, width, height int) {
	if s.node == nil {
		w, h := s.sharedImage.restorable.Size()
		return 0, 0, w, h
	}
	return s.node.Region()
}

func (s *sharedImagePart) Size() (width, height int) {
	_, _, w, h := s.region()
	return w, h
}

func (s *sharedImagePart) DrawImage(img *sharedImagePart, sx0, sy0, sx1, sy1 int, geom *affine.GeoM, colorm *affine.ColorM, mode opengl.CompositeMode, filter graphics.Filter) {
	dx, dy, _, _ := img.region()
	sx0 += dx
	sy0 += dy
	sx1 += dx
	sy1 += dy
	s.sharedImage.restorable.DrawImage(img.sharedImage.restorable, sx0, sy0, sx1, sy1, geom, colorm, mode, filter)
}

func (s *sharedImagePart) ReplacePixels(p []byte) {
	x, y, w, h := s.region()
	if l := 4 * w * h; len(p) != l {
		panic(fmt.Sprintf("ebiten: len(p) was %d but must be %d", len(p), l))
	}
	s.sharedImage.restorable.ReplacePixels(p, x, y, w, h)
}

func (s *sharedImagePart) At(x, y int) (color.Color, error) {
	ox, oy, w, h := s.region()
	if x < 0 || y < 0 || x >= w || y >= h {
		return color.RGBA{}, nil
	}
	return s.sharedImage.restorable.At(x+ox, y+oy)
}

func (s *sharedImagePart) isDisposed() bool {
	return s.sharedImage == nil
}

func (s *sharedImagePart) Dispose() {
	if s.isDisposed() {
		return
	}

	defer func() {
		s.sharedImage = nil
		s.node = nil
	}()

	if s.node == nil {
		s.sharedImage.restorable.Dispose()
		return
	}

	s.sharedImage.page.Free(s.node)
	if !s.sharedImage.page.IsEmpty() {
		return
	}

	index := -1
	for i, sh := range theSharedImages {
		if sh == s.sharedImage {
			index = i
			break
		}
	}
	if index == -1 {
		panic("not reached")
	}
	theSharedImages = append(theSharedImages[:index], theSharedImages[index+1:]...)
}

func (s *sharedImagePart) IsInvalidated() (bool, error) {
	return s.sharedImage.restorable.IsInvalidated()
}

var sharedImageLock sync.Mutex

func newSharedImagePart(width, height int) *sharedImagePart {
	const maxSize = 2048

	sharedImageLock.Lock()
	defer sharedImageLock.Unlock()

	if width > maxSize || height > maxSize {
		s := &sharedImage{
			restorable: restorable.NewImage(width, height, false),
		}
		return &sharedImagePart{
			sharedImage: s,
		}
	}

	for _, s := range theSharedImages {
		if n := s.page.Alloc(width, height); n != nil {
			return &sharedImagePart{
				sharedImage: s,
				node:        n,
			}
		}
	}
	s := &sharedImage{
		restorable: restorable.NewImage(maxSize, maxSize, false),
		page:       packing.NewPage(maxSize),
	}
	theSharedImages = append(theSharedImages, s)

	n := s.page.Alloc(width, height)
	if n == nil {
		panic("not reached")
	}
	return &sharedImagePart{
		sharedImage: s,
		node:        n,
	}
}
