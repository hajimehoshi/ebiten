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
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/debug"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

// forceRestoration reports whether restoration forcibly happens or not.
// This is used only for testing.
var forceRestoration = false

// disabled indicates that restoration is disabled or not.
// Restoration is enabled by default for some platforms like Android for safety.
// Before SetGame, it is not possible to determine whether restoration is needed or not.
var disabled atomic.Bool

var disabledOnce sync.Once

// Disable disables restoration.
func Disable() {
	disabled.Store(true)
}

// needsRestoration reports whether restoration process works or not.
func needsRestoration() bool {
	if forceRestoration {
		return true
	}
	// TODO: If Vulkan is introduced, restoration might not be needed.
	if runtime.GOOS == "android" {
		return !disabled.Load()
	}
	return false
}

// AlwaysReadPixelsFromGPU reports whether ReadPixels always reads pixels from GPU or not.
func AlwaysReadPixelsFromGPU() bool {
	return !needsRestoration()
}

// images is a set of Image objects.
type images struct {
	images      map[*Image]struct{}
	shaders     map[*Shader]struct{}
	contextLost atomic.Bool
}

// theImages represents the images for the current process.
var theImages = &images{
	images:  map[*Image]struct{}{},
	shaders: map[*Shader]struct{}{},
}

func SwapBuffers(graphicsDriver graphicsdriver.Graphics) error {
	if debug.IsDebug {
		debug.FrameLogf("Internal image sizes:\n")
		imgs := make([]*graphicscommand.Image, 0, len(theImages.images))
		for i := range theImages.images {
			imgs = append(imgs, i.image)
		}
		graphicscommand.LogImagesInfo(imgs)
	}
	return resolveStaleImages(graphicsDriver, true)
}

// resolveStaleImages flushes the queued draw commands and resolves all stale images.
// If endFrame is true, the current screen might be used to present when flushing the commands.
func resolveStaleImages(graphicsDriver graphicsdriver.Graphics, endFrame bool) error {
	// When Disable is called, all the images data should be evicted once.
	if disabled.Load() {
		disabledOnce.Do(func() {
			for img := range theImages.images {
				img.makeStale(image.Rect(0, 0, img.width, img.height))
			}
		})
	}

	if err := graphicscommand.FlushCommands(graphicsDriver, endFrame); err != nil {
		return err
	}
	if !needsRestoration() {
		return nil
	}
	return theImages.resolveStaleImages(graphicsDriver)
}

// RestoreIfNeeded restores the images.
//
// Restoration means to make all *graphicscommand.Image objects have their textures and framebuffers.
func RestoreIfNeeded(graphicsDriver graphicsdriver.Graphics) error {
	if !needsRestoration() {
		return nil
	}

	if !forceRestoration && !theImages.contextLost.Load() {
		return nil
	}

	if err := graphicscommand.ResetGraphicsDriverState(graphicsDriver); err != nil {
		return err
	}
	return theImages.restore(graphicsDriver)
}

// DumpImages dumps all the current images to the specified directory.
//
// This is for testing usage.
func DumpImages(graphicsDriver graphicsdriver.Graphics, dir string) (string, error) {
	images := make([]*graphicscommand.Image, 0, len(theImages.images))
	for img := range theImages.images {
		images = append(images, img.image)
	}

	return graphicscommand.DumpImages(images, graphicsDriver, dir)
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
func (i *images) resolveStaleImages(graphicsDriver graphicsdriver.Graphics) error {
	for img := range i.images {
		if err := img.resolveStale(graphicsDriver); err != nil {
			return err
		}
	}
	return nil
}

// makeStaleIfDependingOn makes all the images stale that depend on src.
//
// When src is modified, all images depending on src can't be restored with src.
// makeStaleIfDependingOn is called in such situation.
func (i *images) makeStaleIfDependingOn(src *Image) {
	if src == nil {
		panic("restorable: src must not be nil at makeStaleIfDependingOn")
	}
	for img := range i.images {
		img.makeStaleIfDependingOn(src)
	}
}

// makeStaleIfDependingOnAtRegion makes all the images stale that depend on src at srcRegion.
//
// When src is modified, all images depending on src can't be restored with src at srcRegion.
// makeStaleIfDependingOnAtRegion is called in such situation.
func (i *images) makeStaleIfDependingOnAtRegion(src *Image, srcRegion image.Rectangle) {
	if src == nil {
		panic("restorable: src must not be nil at makeStaleIfDependingOnAtRegion")
	}
	for img := range i.images {
		img.makeStaleIfDependingOnAtRegion(src, srcRegion)
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
// Restoration means to make all *graphicscommand.Image objects have their textures and framebuffers.
func (i *images) restore(graphicsDriver graphicsdriver.Graphics) error {
	if !needsRestoration() {
		panic("restorable: restore cannot be called when restoration is disabled")
	}

	// Dispose all the shaders ahead of restoration. A current shader ID and a new shader ID can be duplicated.
	for s := range i.shaders {
		s.shader.Dispose()
		s.shader = nil
	}
	for s := range i.shaders {
		s.restore()
	}

	// Dispose all the images ahead of restoration. A current texture ID and a new texture ID can be duplicated.
	// TODO: Write a test to confirm that ID duplication never happens.
	for i := range i.images {
		i.image.Dispose()
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
		images[i] = struct{}{}
	}
	edges := map[edge]struct{}{}
	for t := range images {
		for s := range t.dependingImages() {
			edges[edge{source: s, target: t}] = struct{}{}
		}
	}

	var sorted []*Image
	for len(images) > 0 {
		// current represents images that have no incoming edges.
		current := map[*Image]struct{}{}
		for i := range images {
			current[i] = struct{}{}
		}
		for e := range edges {
			delete(current, e.target)
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
		if err := img.restore(graphicsDriver); err != nil {
			return err
		}
	}

	i.contextLost.Store(false)

	return nil
}

var graphicsDriverInitialized bool

// InitializeGraphicsDriverState initializes the graphics driver state.
func InitializeGraphicsDriverState(graphicsDriver graphicsdriver.Graphics) error {
	graphicsDriverInitialized = true
	return graphicscommand.InitializeGraphicsDriverState(graphicsDriver)
}

// MaxImageSize returns the maximum size of an image.
func MaxImageSize(graphicsDriver graphicsdriver.Graphics) int {
	return graphicscommand.MaxImageSize(graphicsDriver)
}

// OnContextLost is called when the context lost is detected in an explicit way.
func OnContextLost() {
	theImages.contextLost.Store(true)
}
