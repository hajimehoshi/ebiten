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
	"path/filepath"

	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
)

// forceRestoring reports whether restoring forcely happens or not.
var forceRestoring = false

// needsRestoring reports whether restoring process works or not.
func needsRestoring() bool {
	if forceRestoring {
		return true
	}
	return graphicscommand.NeedsRestoring()
}

// EnableRestoringForTesting forces to enable restoring for testing.
func EnableRestoringForTesting() {
	forceRestoring = true
}

// images is a set of Image objects.
type images struct {
	images      map[*Image]struct{}
	shaders     map[*Shader]struct{}
	lastTarget  *Image
	contextLost bool
}

// theImages represents the images for the current process.
var theImages = &images{
	images:  map[*Image]struct{}{},
	shaders: map[*Shader]struct{}{},
}

// ResolveStaleImages flushes the queued draw commands and resolves
// all stale images.
//
// ResolveStaleImages is intended to be called at the end of a frame.
func ResolveStaleImages() error {
	if err := graphicscommand.FlushCommands(); err != nil {
		return err
	}
	if !needsRestoring() {
		return nil
	}
	return theImages.resolveStaleImages()
}

// RestoreIfNeeded restores the images.
//
// Restoring means to make all *graphicscommand.Image objects have their textures and framebuffers.
func RestoreIfNeeded() error {
	if !needsRestoring() {
		return nil
	}

	if !forceRestoring {
		var r bool

		if canDetectContextLostExplicitly {
			r = theImages.contextLost
		} else {
			// As isInvalidated() is expensive, call this only for one image.
			// This assumes that if there is one image that is invalidated, all images are invalidated.
			for img := range theImages.images {
				// The screen image might not have a texture. Skip this.
				if img.screen {
					continue
				}
				var err error
				r, err = img.isInvalidated()
				if err != nil {
					return err
				}
				break
			}
		}

		if !r {
			return nil
		}
	}

	err := graphicscommand.ResetGraphicsDriverState()
	if err == driver.GraphicsNotReady {
		return nil
	}
	if err != nil {
		return err
	}
	return theImages.restore()
}

// DumpImages dumps all the current images to the specified directory.
//
// This is for testing usage.
func DumpImages(dir string) error {
	for img := range theImages.images {
		if err := img.Dump(filepath.Join(dir, "*.png"), false); err != nil {
			return err
		}
	}
	return nil
}

// add adds img to the images.
func (i *images) add(img *Image) {
	i.images[img] = struct{}{}
}

func (i *images) addShader(shader *Shader) {
	i.shaders[shader] = struct{}{}
}

// remove removes img from the images.
func (i *images) remove(img *Image) {
	i.makeStaleIfDependingOn(img)
	delete(i.images, img)
}

func (i *images) removeShader(shader *Shader) {
	i.makeStaleIfDependingOnShader(shader)
	delete(i.shaders, shader)
}

// resolveStaleImages resolves stale images.
func (i *images) resolveStaleImages() error {
	i.lastTarget = nil
	for img := range i.images {
		if err := img.resolveStale(); err != nil {
			return err
		}
	}
	return nil
}

// makeStaleIfDependingOn makes all the images stale that depend on target.
//
// When target is modified, all images depending on target can't be restored with target.
// makeStaleIfDependingOn is called in such situation.
func (i *images) makeStaleIfDependingOn(target *Image) {
	if target == nil {
		panic("restorable: target must not be nil at makeStaleIfDependingOn")
	}
	if i.lastTarget == target {
		return
	}
	i.lastTarget = target
	for img := range i.images {
		img.makeStaleIfDependingOn(target)
	}
}

// makeStaleIfDependingOn makes all the images stale that depend on shader.
func (i *images) makeStaleIfDependingOnShader(shader *Shader) {
	if shader == nil {
		panic("restorable: shader must not be nil at makeStaleIfDependingOnShader")
	}
	for img := range i.images {
		img.makeStaleIfDependingOnShader(shader)
	}
}

// restore restores the images.
//
// Restoring means to make all *graphicscommand.Image objects have their textures and framebuffers.
func (i *images) restore() error {
	if !needsRestoring() {
		panic("restorable: restore cannot be called when restoring is disabled")
	}

	// Dispose all the shaders ahead of restoring. A current shader ID and a new shader ID can be duplicated.
	for s := range i.shaders {
		if needsDisposingWhenRestoring {
			s.shader.Dispose()
		}
		s.shader = nil
	}
	for s := range i.shaders {
		s.restore()
	}

	// Dispose all the images ahead of restoring. A current texture ID and a new texture ID can be duplicated.
	// TODO: Write a test to confirm that ID duplication never happens.
	for i := range i.images {
		if needsDisposingWhenRestoring {
			i.image.Dispose()
		}
		i.image = nil
	}

	// Let's do topological sort based on dependencies of drawing history.
	// It is assured that there are not loops since cyclic drawing makes images stale.
	type edge struct {
		source *Image
		target *Image
	}
	images := map[*Image]struct{}{}
	for i := range i.images {
		if !i.priority {
			images[i] = struct{}{}
		}
	}
	edges := map[edge]struct{}{}
	for t := range images {
		for s := range t.dependingImages() {
			edges[edge{source: s, target: t}] = struct{}{}
		}
	}

	sorted := []*Image{}
	for i := range i.images {
		if i.priority {
			sorted = append(sorted, i)
		}
	}
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

	i.contextLost = false

	return nil
}

// InitializeGraphicsDriverState initializes the graphics driver state.
func InitializeGraphicsDriverState() error {
	return graphicscommand.ResetGraphicsDriverState()
}

// OnContextLost is called when the context lost is detected in an explicit way.
func OnContextLost() {
	if !canDetectContextLostExplicitly {
		panic("restorable: OnContextLost cannot be called in this environment")
	}
	theImages.contextLost = true
}
