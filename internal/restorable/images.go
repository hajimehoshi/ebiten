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
//
// This value is overridden at enabled_*.go.
var restoringEnabled = true

// IsRestoringEnabled returns a boolean value indicating whether
// restoring process works or not.
func IsRestoringEnabled() bool {
	// This value is updated only at init or EnableRestoringForTesting.
	// No need to lock here.
	return restoringEnabled
}

// EnableRestoringForTesting forces to enable restoring for testing.
func EnableRestoringForTesting() {
	restoringEnabled = true
}

// images is a set of Image objects.
type images struct {
	images     map[*Image]struct{}
	lastTarget *Image
	m          sync.Mutex
}

// theImages represents the images for the current process.
var theImages = &images{
	images: map[*Image]struct{}{},
}

// ResolveStaleImages flushes the queued draw commands and resolves
// all stale images.
//
// ResolveStaleImages is intended to be called at the end of a frame.
func ResolveStaleImages() error {
	if err := graphics.FlushCommands(); err != nil {
		return err
	}
	if !restoringEnabled {
		return nil
	}
	return theImages.resolveStaleImages()
}

// Restore restores the images.
//
// Restoring means to make all *graphics.Image objects have their textures and framebuffers.
func Restore() error {
	if err := graphics.ResetGLState(); err != nil {
		return err
	}
	return theImages.restore()
}

// add adds img to the images.
func (i *images) add(img *Image) {
	i.m.Lock()
	i.images[img] = struct{}{}
	i.m.Unlock()
}

// remove removes img from the images.
func (i *images) remove(img *Image) {
	i.m.Lock()
	delete(i.images, img)
	i.m.Unlock()
}

// resolveStaleImages resolves stale images.
func (i *images) resolveStaleImages() error {
	i.m.Lock()
	i.lastTarget = nil
	for img := range i.images {
		if err := img.resolveStale(); err != nil {
			i.m.Unlock()
			return err
		}
	}
	i.m.Unlock()
	return nil
}

// makeStaleIfDependingOn makes all the images stale that depend on target.
//
// When target is changed, all images depending on target can't be restored with target.
// makeStaleIfDependingOn is called in such situation.
func (i *images) makeStaleIfDependingOn(target *Image) {
	// Avoid defer for performance
	i.m.Lock()
	if target == nil {
		// disposed
		i.m.Unlock()
		return
	}
	if i.lastTarget == target {
		i.m.Unlock()
		return
	}
	i.lastTarget = target
	for img := range i.images {
		// TODO: This seems not enough: What if img becomes stale but what about
		// other images depend on img? (#357)
		img.makeStaleIfDependingOn(target)
	}
	i.m.Unlock()
}

// restore restores the images.
//
// Restoring means to make all *graphics.Image objects have their textures and framebuffers.
func (i *images) restore() error {
	i.m.Lock()
	defer i.m.Unlock()
	if !IsRestoringEnabled() {
		panic("not reached")
	}

	// Framebuffers/textures cannot be disposed since framebuffers/textures that
	// don't belong to the current context.

	// Let's do topological sort based on dependencies of drawing history.
	// It is assured that there are not loops since cyclic drawing makes images stale.
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

// InitializeGLState initializes the GL state.
func InitializeGLState() error {
	return graphics.ResetGLState()
}
