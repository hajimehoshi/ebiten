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
	node        *packing.Node
}

func (s *sharedImagePart) image() *restorable.Image {
	return s.sharedImage.restorable
}

func (s *sharedImagePart) region() (x, y, width, height int) {
	return s.node.Region()
}

func (s *sharedImagePart) Dispose() {
	s.sharedImage.page.Free(s.node)
	if s.sharedImage.page.IsEmpty() {
		s.sharedImage.restorable.Dispose()
		s.sharedImage.restorable = nil
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
}

var sharedImageLock sync.Mutex

func newSharedImagePart(width, height int) *sharedImagePart {
	const maxSize = 2048

	sharedImageLock.Lock()
	defer sharedImageLock.Unlock()

	if width > maxSize || height > maxSize {
		return nil
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
