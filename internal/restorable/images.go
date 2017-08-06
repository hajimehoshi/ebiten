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
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/sync"
)

// restoringEnabled indicates if restoring happens or not.
var restoringEnabled = true // This value is overridden at enabled_*.go.

func IsRestoringEnabled() bool {
	// This value is updated only at init or EnableRestoringForTesting.
	// No need to lock here.
	return restoringEnabled
}

func EnableRestoringForTesting() {
	restoringEnabled = true
}

type images struct {
	images      map[*Image]struct{}
	lastChecked *Image
	m           sync.Mutex
}

var theImages = &images{
	images: map[*Image]struct{}{},
}

func FlushAndResolveStalePixels() error {
	if err := graphics.FlushCommands(); err != nil {
		return err
	}
	return theImages.resolveStalePixels()
}

func Restore() error {
	if err := graphics.ResetGLState(); err != nil {
		return err
	}
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
	if !IsRestoringEnabled() {
		panic("not reached")
	}
	// Framebuffers/textures cannot be disposed since framebuffers/textures that
	// don't belong to the current context.

	// Let's do topological sort based on dependencies of drawing history.
	// There should not be a loop since cyclic drawing makes images stale.
	type edge struct {
		source *Image
		target *Image
	}
	images := map[*Image]struct{}{}
	for i := range i.images {
		images[i] = struct{}{}
	}
	edges := map[edge]struct{}{}
	for t := range images {
		for s := range t.dependingImages() {
			edges[edge{source: s, target: t}] = struct{}{}
		}
	}
	sorted := []*Image{}
	for len(images) > 0 {
		// current repesents images that have no incoming edges.
		current := map[*Image]struct{}{}
		for i := range images {
			current[i] = struct{}{}
		}
		for e := range edges {
			if _, ok := current[e.target]; ok {
				delete(current, e.target)
			}
		}
		for i := range current {
			delete(images, i)
			sorted = append(sorted, i)
		}
		removed := []edge{}
		for e := range edges {
			if _, ok := current[e.source]; ok {
				removed = append(removed, e)
			}
		}
		for _, e := range removed {
			delete(edges, e)
		}
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

func ResetGLState() error {
	return graphics.ResetGLState()
}
