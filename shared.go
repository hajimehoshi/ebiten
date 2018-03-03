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
	"github.com/hajimehoshi/ebiten/internal/bsp"
	"github.com/hajimehoshi/ebiten/internal/restorable"
)

type sharedImage struct {
	restorable *restorable.Image
	page       bsp.Page
}

var (
	theSharedImages = []*sharedImage{}
)

type sharedImagePart struct {
	sharedImage *sharedImage
	node        *bsp.Node
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

func newSharedImagePart(width, height int) *sharedImagePart {
	// TODO: Lock!
	if width > bsp.MaxSize || height > bsp.MaxSize {
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
		restorable: restorable.NewImage(bsp.MaxSize, bsp.MaxSize, false),
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
