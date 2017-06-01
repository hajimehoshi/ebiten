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
	"github.com/hajimehoshi/ebiten/internal/sync"
)

type images struct {
	images      map[*Image]struct{}
	lastChecked *Image
	m           sync.Mutex
}

var theImages = &images{
	images: map[*Image]struct{}{},
}

func ResolveStalePixels() error {
	return theImages.resolveStalePixels()
}

func Restore() error {
	return theImages.restore()
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

func (i *images) resolveStalePixels() error {
	i.m.Lock()
	defer i.m.Unlock()
	i.lastChecked = nil
	for img := range i.images {
		if err := img.resolveStalePixels(); err != nil {
			return err
		}
	}
	return nil
}

func (i *images) resetPixelsIfDependingOn(target *Image) {
	// Avoid defer for performance
	i.m.Lock()
	if target == nil {
		// disposed
		i.m.Unlock()
		return
	}
	if i.lastChecked == target {
		i.m.Unlock()
		return
	}
	i.lastChecked = target
	for img := range i.images {
		// TODO: This seems not enough: What if img becomes stale but what about
		// other images depend on img? (#357)
		img.makeStaleIfDependingOn(target)
	}
	i.m.Unlock()
}

func (i *images) restore() error {
	i.m.Lock()
	defer i.m.Unlock()
	// Framebuffers/textures cannot be disposed since framebuffers/textures that
	// don't belong to the current context.

	// Let's do topological sort based on dependencies of drawing history.
	// There should not be a loop since cyclic drawing makes images stale.
	current := map[*Image]struct{}{}
	toBeDetermined := map[*Image]struct{}{}
	sorted := []*Image{}
	for img := range i.images {
		if img.hasDependency() {
			toBeDetermined[img] = struct{}{}
			continue
		}
		current[img] = struct{}{}
		sorted = append(sorted, img)
	}
	// TODO: How to confirm that there is no loop?
	for len(current) > 0 {
		next := map[*Image]struct{}{}
		// TODO: This is inefficient. Get all edges from the source first.
		// (*Image).childImages?
		for source := range current {
			for target := range toBeDetermined {
				if target.dependsOn(source) {
					next[target] = struct{}{}
				}
			}
		}
		for img := range next {
			sorted = append(sorted, img)
			delete(toBeDetermined, img)
		}
		current = next
	}
	if len(toBeDetermined) > 0 {
		panic("not reached")
	}
	for _, img := range sorted {
		if err := img.restore(); err != nil {
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
