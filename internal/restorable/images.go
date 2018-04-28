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
	"image"

	"github.com/hajimehoshi/ebiten/internal/graphics"
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

func Images() ([]image.Image, error) {
	var imgs []image.Image
	for img := range theImages.images {
		if img.volatile {
			continue
		}
		if img.screen {
			continue
		}

		w, h := img.Size()
		pix := make([]byte, 4*w*h)
		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				c, err := img.At(i, j)
				if err != nil {
					return nil, err
				}
				pix[4*(i+j*w)] = byte(c.R)
				pix[4*(i+j*w)+1] = byte(c.G)
				pix[4*(i+j*w)+2] = byte(c.B)
				pix[4*(i+j*w)+3] = byte(c.A)
			}
		}
		imgs = append(imgs, &image.RGBA{
			Pix:    pix,
			Stride: 4 * w,
			Rect:   image.Rect(0, 0, w, h),
		})
	}
	return imgs, nil
}

// add adds img to the images.
func (i *images) add(img *Image) {
	i.images[img] = struct{}{}
}

// remove removes img from the images.
func (i *images) remove(img *Image) {
	i.makeStaleIfDependingOnImpl(img)
	delete(i.images, img)
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
// When target is changed, all images depending on target can't be restored with target.
// makeStaleIfDependingOn is called in such situation.
func (i *images) makeStaleIfDependingOn(target *Image) {
	// Avoid defer for performance
	i.makeStaleIfDependingOnImpl(target)
}

func (i *images) makeStaleIfDependingOnImpl(target *Image) {
	if target == nil {
		panic("not reached")
	}
	if i.lastTarget == target {
		return
	}
	i.lastTarget = target
	for img := range i.images {
		img.makeStaleIfDependingOn(target)
	}
}

// restore restores the images.
//
// Restoring means to make all *graphics.Image objects have their textures and framebuffers.
func (i *images) restore() error {
	if !IsRestoringEnabled() {
		panic("not reached")
	}

	// Dispose image explicitly
	for img := range i.images {
		img.image.Dispose()
		// img.image can't be set nil here, or Size() panics when restoring.
	}

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
