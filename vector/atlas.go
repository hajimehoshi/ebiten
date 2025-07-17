// Copyright 2025 The Ebitengine Authors
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

package vector

import (
	"image"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/packing"
)

type atlasRegion struct {
	index  int
	bounds image.Rectangle
}

type atlas struct {
	pathBounds   []image.Rectangle
	atlasRegions []atlasRegion
	atlasImages  []*ebiten.Image
	pages        []*packing.Page
}

func roundUpAtlasSize(size int) int {
	if size < 16 {
		return 16
	}
	return int(math.Ceil(math.Pow(1.5, math.Ceil(math.Log(float64(size))/math.Log(1.5)))))
}

func (a *atlas) setPaths(dstBounds image.Rectangle, paths []*Path, antialias bool) {
	// Reset the members.
	a.pathBounds = slices.Delete(a.pathBounds, 0, len(a.pathBounds))
	a.atlasRegions = slices.Delete(a.atlasRegions, 0, len(a.atlasRegions))
	a.pages = slices.Delete(a.pages, 0, len(a.pages))

	if len(paths) == 0 {
		return
	}

	for _, p := range paths {
		pb := p.Bounds()
		a.pathBounds = append(a.pathBounds, pb)
		pb = pb.Intersect(dstBounds)
		if pb.Dx() == 0 || pb.Dy() == 0 {
			a.atlasRegions = append(a.atlasRegions, atlasRegion{})
			if antialias {
				a.atlasRegions = append(a.atlasRegions, atlasRegion{})
			}
			continue
		}
		index, node := a.allocNode(dstBounds, pb.Dx(), pb.Dy())
		a.atlasRegions = append(a.atlasRegions, atlasRegion{
			index:  index,
			bounds: node.Region(),
		})
		if antialias {
			index, node := a.allocNode(dstBounds, pb.Dx(), pb.Dy())
			a.atlasRegions = append(a.atlasRegions, atlasRegion{
				index:  index,
				bounds: node.Region(),
			})
		}
	}

	a.atlasImages = slices.Grow(a.atlasImages, len(a.pages))[:len(a.pages)]
	for i := range a.pages {
		r := a.pages[i].AllocatedRegion()
		var origWidth, origHeight int
		if a.atlasImages[i] != nil {
			origWidth = a.atlasImages[i].Bounds().Dx()
			origHeight = a.atlasImages[i].Bounds().Dy()
			if origWidth < r.Max.X || origHeight < r.Max.Y {
				a.atlasImages[i].Deallocate()
				a.atlasImages[i] = nil
			}
		}
		if a.atlasImages[i] != nil {
			a.atlasImages[i].Clear()
		} else {
			maxPageSize := a.maxPageSize(dstBounds)
			// Extend the bounds a little bit by roundUpAtlasSize to avoid creating an image too often.
			w := min(maxPageSize, max(roundUpAtlasSize(r.Max.X), origWidth))
			h := min(maxPageSize, max(roundUpAtlasSize(r.Max.Y), origHeight))
			a.atlasImages[i] = ebiten.NewImage(w, h)
		}
	}
}

func roundUpPowerOf2(n int) int {
	if n <= 0 {
		return 1
	}
	p := 1
	for p < n {
		p *= 2
	}
	return p
}

func (a *atlas) maxPageSize(dstBounds image.Rectangle) int {
	// 4096 is a very conservative value, but before the game starts, there is no way to know the maximum size of an image.
	return max(4096, roundUpPowerOf2(dstBounds.Dx()), roundUpPowerOf2(dstBounds.Dy()))
}

func (a *atlas) allocNode(dstBounds image.Rectangle, width, height int) (int, *packing.Node) {
	minSize := max(1024, roundUpPowerOf2(width), roundUpPowerOf2(height))
	if len(a.pages) == 0 {
		a.pages = append(a.pages, packing.NewPage(minSize, minSize, a.maxPageSize(dstBounds)))
	}

	node := a.pages[len(a.pages)-1].Alloc(width, height)
	if node != nil {
		return len(a.pages) - 1, node
	}

	a.pages = append(a.pages, packing.NewPage(minSize, minSize, a.maxPageSize(dstBounds)))
	node = a.pages[len(a.pages)-1].Alloc(width, height)
	if node != nil {
		return len(a.pages) - 1, node
	}

	panic("vector: failed to allocate a node for a path")
}

func (a *atlas) stencilBufferImageAt(i int) *ebiten.Image {
	r := a.atlasRegions[i]
	if r.bounds.Empty() {
		return nil
	}
	return a.atlasImages[r.index].SubImage(r.bounds).(*ebiten.Image)
}

func (a *atlas) pathBoundsAt(i int) image.Rectangle {
	return a.pathBounds[i]
}
