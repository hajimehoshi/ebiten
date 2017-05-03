// Copyright 2017 The Ebiten Authors
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

package restorable

import (
	"sync"

	"github.com/hajimehoshi/ebiten/internal/opengl"
)

type images struct {
	images      map[*Image]struct{}
	m           sync.Mutex
	lastChecked *Image
}

var theImages = &images{
	images: map[*Image]struct{}{},
}

func ResolveStalePixels(context *opengl.Context) error {
	return theImages.resolveStalePixels(context)
}

func Restore(context *opengl.Context) error {
	return theImages.restore(context)
}

func ClearVolatileImages() {
	theImages.clearVolatileImages()
}

func (i *images) add(img *Image) {
	i.m.Lock()
	defer i.m.Unlock()
	i.images[img] = struct{}{}
}

func (i *images) remove(img *Image) {
	i.m.Lock()
	defer i.m.Unlock()
	delete(i.images, img)
}

func (i *images) resolveStalePixels(context *opengl.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	i.lastChecked = nil
	for img := range i.images {
		if err := img.resolveStalePixels(context); err != nil {
			return err
		}
	}
	return nil
}

func (i *images) resetPixelsIfDependingOn(target *Image) {
	i.m.Lock()
	defer i.m.Unlock()
	if target == nil {
		// disposed
		return
	}
	if i.lastChecked == target {
		return
	}
	i.lastChecked = target
	for img := range i.images {
		img.makeStaleIfDependingOn(target)
	}
}

func (i *images) restore(context *opengl.Context) error {
	i.m.Lock()
	defer i.m.Unlock()
	// Framebuffers/textures cannot be disposed since framebuffers/textures that
	// don't belong to the current context.
	imagesWithoutDependency := []*Image{}
	imagesWithDependency := []*Image{}
	for img := range i.images {
		if img.hasDependency() {
			imagesWithDependency = append(imagesWithDependency, img)
		} else {
			imagesWithoutDependency = append(imagesWithoutDependency, img)
		}
	}
	// Images depending on other images should be processed first.
	for _, img := range imagesWithoutDependency {
		if err := img.restore(context); err != nil {
			return err
		}
	}
	for _, img := range imagesWithDependency {
		if err := img.restore(context); err != nil {
			return err
		}
	}
	return nil
}

func (i *images) clearVolatileImages() {
	i.m.Lock()
	defer i.m.Unlock()
	for img := range i.images {
		img.clearIfVolatile()
	}
}
