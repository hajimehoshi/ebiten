// Copyright 2026 The Ebitengine Authors
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
	"image"
	"maps"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// maxPixelsCacheSize is the maximum size in bytes for caching the whole pixels of an image as a single tile.
// A pixel cache for an image whose whole pixel data exceeds this size is divided into multiple tiles,
// so that reading a small region does not require a whole-image allocation and a whole-image GPU read.
// The value must be bigger than 4 * 1920 * 1080 (full HD), so that a screen-sized image is cached as a single tile.
const maxPixelsCacheSize = 4 * 2048 * 2048

// dividedTileSize is the width and height in pixels of a tile in a pixel cache divided into multiple tiles.
const dividedTileSize = 256

// tile is a unit of cached pixels.
type tile struct {
	// pixels is cached pixels for ReadPixels.
	pixels []byte

	// needsWriteBack represents whether pixels include writes that are not applied to the GPU yet.
	// The pixels of a tile needing a write-back are the primary data of the pixels for ReadPixels.
	// Otherwise, the pixels are a copy of the GPU pixels.
	needsWriteBack bool
}

// dot is a pending write of a single pixel.
type dot struct {
	pos image.Point
	clr [4]byte
}

// tileLoad is a load of a tile's pixels from the GPU.
type tileLoad struct {
	// index is the index of the tile.
	index image.Point

	// pixels receives the GPU pixels of the tile.
	pixels []byte

	// dots are the dots in the tile at the start of the load, which the GPU pixels do not include.
	dots []dot
}

// tilesLoad is a load of tiles from the GPU, started by startReadPixels and finished by finishReadPixels.
type tilesLoad struct {
	// generation is the generation of the cache at the start of the load.
	generation uint64

	tiles []tileLoad

	// results are the read-backs of tiles, in the same order.
	results []*PixelReadback
}

// pixelCache manages the CPU-side pixel data of an image.
type pixelCache struct {
	// width and height are the size of the image in pixels.
	width  int
	height int

	// dots is a buffer for drawing a lot of dots.
	// An entry in this map is the primary data of pixels for ReadPixels.
	// An entry never exists at a position whose tile is cached in tiles.
	dots map[image.Point][4]byte

	// tiles is cached pixels keyed by the tile index.
	tiles map[image.Point]*tile

	// generation changes whenever the pixels of the image change.
	generation uint64
}

// init initializes the cache for an image of the given size.
func (c *pixelCache) init(width, height int) {
	c.width = width
	c.height = height
}

// tileSize returns the width and height in pixels of a tile.
// For a small image, a single tile covers the whole image.
func (c *pixelCache) tileSize() (int, int) {
	if 4*c.width*c.height > maxPixelsCacheSize {
		return dividedTileSize, dividedTileSize
	}
	return c.width, c.height
}

// tileIndex returns the index of the tile containing pos.
func (c *pixelCache) tileIndex(pos image.Point) image.Point {
	w, h := c.tileSize()
	return image.Point{X: pos.X / w, Y: pos.Y / h}
}

// tileRegion returns the region of the tile at idx, clipped by the image bounds.
func (c *pixelCache) tileRegion(idx image.Point) image.Rectangle {
	w, h := c.tileSize()
	r := image.Rect(idx.X*w, idx.Y*h, (idx.X+1)*w, (idx.Y+1)*h)
	return r.Intersect(image.Rect(0, 0, c.width, c.height))
}

// startReadPixels reads the pixels in region into pixels from the cache, and starts loading the tiles in
// region that are not cached. The pixels of the loaded tiles are read by finishReadPixels after the
// read-backs of the returned load complete. The returned load is nil when no tile needs loading.
func (c *pixelCache) startReadPixels(img *ui.Image, pixels []byte, region image.Rectangle) *tilesLoad {
	if region.Empty() {
		return nil
	}

	if region.Dx() == 1 && region.Dy() == 1 {
		if clr, ok := c.dots[region.Min]; ok {
			copy(pixels, clr[:])
			return nil
		}
	}

	// Do not call writeBackPixelsIfNeeded here. This would slow (image/draw).Draw.
	// See ebiten.TestImageDrawOver.

	var load *tilesLoad
	minTile := c.tileIndex(region.Min)
	maxTile := c.tileIndex(region.Max.Sub(image.Point{X: 1, Y: 1}))
	for ty := minTile.Y; ty <= maxTile.Y; ty++ {
		for tx := minTile.X; tx <= maxTile.X; tx++ {
			idx := image.Point{X: tx, Y: ty}
			if t, ok := c.tiles[idx]; ok {
				c.copyTilePixels(pixels, region, idx, t.pixels)
				continue
			}
			r := c.tileRegion(idx)
			pix := make([]byte, 4*r.Dx()*r.Dy())
			result := &PixelReadback{
				result: img.ReadPixelsAsync(pix, r),
			}
			if load == nil {
				load = &tilesLoad{
					generation: c.generation,
				}
			}
			load.tiles = append(load.tiles, tileLoad{
				index:  idx,
				pixels: pix,
				dots:   c.dotsIn(r),
			})
			load.results = append(load.results, result)
		}
	}
	return load
}

// finishReadPixels reads the pixels in region of the tiles loaded by load into pixels.
// The read-backs of load must have completed successfully.
func (c *pixelCache) finishReadPixels(load *tilesLoad, pixels []byte, region image.Rectangle) {
	// The loaded pixels and the dots are the pixels at the start of the load, so they are cached only
	// when the pixels have not changed since then.
	cacheable := load.generation == c.generation
	for _, l := range load.tiles {
		r := c.tileRegion(l.index)
		for _, d := range l.dots {
			i := 4 * ((d.pos.Y-r.Min.Y)*r.Dx() + (d.pos.X - r.Min.X))
			copy(l.pixels[i:i+4], d.clr[:])
		}
		c.copyTilePixels(pixels, region, l.index, l.pixels)

		if !cacheable {
			continue
		}
		// Another read might have cached the same pixels during the load.
		if _, ok := c.tiles[l.index]; ok {
			continue
		}
		// The dots are merged into the tile.
		for _, d := range l.dots {
			delete(c.dots, d.pos)
		}
		if c.tiles == nil {
			c.tiles = map[image.Point]*tile{}
		}
		c.tiles[l.index] = &tile{
			pixels:         l.pixels,
			needsWriteBack: len(l.dots) > 0,
		}
	}
}

// dotsIn returns the dots in region.
func (c *pixelCache) dotsIn(region image.Rectangle) []dot {
	var dots []dot
	for pos, clr := range c.dots {
		if !pos.In(region) {
			continue
		}
		dots = append(dots, dot{
			pos: pos,
			clr: clr,
		})
	}
	return dots
}

// copyTilePixels copies the pixels in region of the tile at idx, whose pixels are tilePixels, into pixels
// of region.
func (c *pixelCache) copyTilePixels(pixels []byte, region image.Rectangle, idx image.Point, tilePixels []byte) {
	r := c.tileRegion(idx)
	ir := r.Intersect(region)
	lineWidth := 4 * ir.Dx()
	for y := ir.Min.Y; y < ir.Max.Y; y++ {
		srcX := 4 * ((y-r.Min.Y)*r.Dx() + (ir.Min.X - r.Min.X))
		dstX := 4 * ((y-region.Min.Y)*region.Dx() + (ir.Min.X - region.Min.X))
		copy(pixels[dstX:dstX+lineWidth], tilePixels[srcX:srcX+lineWidth])
	}
}

// writePixels updates the cache for a write to region, and writes the pixels to the GPU if necessary.
func (c *pixelCache) writePixels(img *ui.Image, pix []byte, region image.Rectangle) {
	c.generation++

	// Writing one pixel is a special case.
	// Do not write the pixels to the GPU, as (image/draw).Image's functions might call WritePixels with pixels one by one.
	if region.Dx() == 1 && region.Dy() == 1 {
		// If the tile at the position is cached, update this instead of adding an entry to dots.
		if t, ok := c.tiles[c.tileIndex(region.Min)]; ok {
			r := c.tileRegion(c.tileIndex(region.Min))
			i := 4 * ((region.Min.Y-r.Min.Y)*r.Dx() + (region.Min.X - r.Min.X))
			t.pixels[i] = pix[0]
			t.pixels[i+1] = pix[1]
			t.pixels[i+2] = pix[2]
			t.pixels[i+3] = pix[3]
			t.needsWriteBack = true
			delete(c.dots, region.Min)
			return
		}

		if c.dots == nil {
			c.dots = map[image.Point][4]byte{}
		}

		var clr [4]byte
		copy(clr[:], pix)
		c.dots[region.Min] = clr

		if len(c.dots) >= 10000 {
			c.writeBackPixelsIfNeeded(img)
		}
		return
	}

	// If a tile is cached, this indicates ReadPixels was called and might be called again later.
	// Keep and update the cached pixels in this case.
	for idx, t := range c.tiles {
		r := c.tileRegion(idx)
		ir := r.Intersect(region)
		if ir.Empty() {
			continue
		}
		lineWidth := 4 * ir.Dx()
		for y := ir.Min.Y; y < ir.Max.Y; y++ {
			dstX := 4 * ((y-r.Min.Y)*r.Dx() + (ir.Min.X - r.Min.X))
			srcX := 4 * ((y-region.Min.Y)*region.Dx() + (ir.Min.X - region.Min.X))
			copy(t.pixels[dstX:dstX+lineWidth], pix[srcX:srcX+lineWidth])
		}
		// needsWriteBack can NOT be set false as the outside pixels of the region are not written by writePixels here.
		// See the test TestImageUnsyncedPixels.
	}

	// Even if no tile is cached, do not create a new tile.
	// It is in theory possible to copy the argument pixels, but this tends to consume a lot of memory.
	// Avoid this unless ReadPixels is called.

	// Remove entries in the dots buffer that are overwritten by this writePixels call.
	maps.DeleteFunc(c.dots, func(pos image.Point, _ [4]byte) bool {
		return pos.In(region)
	})

	img.WritePixels(pix, region)
}

// writeBackPixelsIfNeeded applies the pending pixel writes to the GPU.
// After writeBackPixelsIfNeeded, dots is cleared, but cached tiles might remain.
func (c *pixelCache) writeBackPixelsIfNeeded(img *ui.Image) {
	for idx, t := range c.tiles {
		if !t.needsWriteBack {
			continue
		}
		img.WritePixels(t.pixels, c.tileRegion(idx))
		t.needsWriteBack = false
	}

	if len(c.dots) == 0 {
		return
	}

	l := len(c.dots)
	vs := make([]float32, l*4*graphics.VertexFloatCount)
	is := make([]uint32, l*6)
	sx, sy := float32(1), float32(1)
	var idx int
	for p, clr := range c.dots {
		dx := float32(p.X)
		dy := float32(p.Y)
		crf := float32(clr[0]) / 0xff
		cgf := float32(clr[1]) / 0xff
		cbf := float32(clr[2]) / 0xff
		caf := float32(clr[3]) / 0xff

		vidx := 4 * idx
		iidx := 6 * idx

		vs[graphics.VertexFloatCount*vidx] = dx
		vs[graphics.VertexFloatCount*vidx+1] = dy
		vs[graphics.VertexFloatCount*vidx+2] = sx
		vs[graphics.VertexFloatCount*vidx+3] = sy
		vs[graphics.VertexFloatCount*vidx+4] = crf
		vs[graphics.VertexFloatCount*vidx+5] = cgf
		vs[graphics.VertexFloatCount*vidx+6] = cbf
		vs[graphics.VertexFloatCount*vidx+7] = caf

		vs[graphics.VertexFloatCount*(vidx+1)] = dx + 1
		vs[graphics.VertexFloatCount*(vidx+1)+1] = dy
		vs[graphics.VertexFloatCount*(vidx+1)+2] = sx + 1
		vs[graphics.VertexFloatCount*(vidx+1)+3] = sy
		vs[graphics.VertexFloatCount*(vidx+1)+4] = crf
		vs[graphics.VertexFloatCount*(vidx+1)+5] = cgf
		vs[graphics.VertexFloatCount*(vidx+1)+6] = cbf
		vs[graphics.VertexFloatCount*(vidx+1)+7] = caf

		vs[graphics.VertexFloatCount*(vidx+2)] = dx
		vs[graphics.VertexFloatCount*(vidx+2)+1] = dy + 1
		vs[graphics.VertexFloatCount*(vidx+2)+2] = sx
		vs[graphics.VertexFloatCount*(vidx+2)+3] = sy + 1
		vs[graphics.VertexFloatCount*(vidx+2)+4] = crf
		vs[graphics.VertexFloatCount*(vidx+2)+5] = cgf
		vs[graphics.VertexFloatCount*(vidx+2)+6] = cbf
		vs[graphics.VertexFloatCount*(vidx+2)+7] = caf

		vs[graphics.VertexFloatCount*(vidx+3)] = dx + 1
		vs[graphics.VertexFloatCount*(vidx+3)+1] = dy + 1
		vs[graphics.VertexFloatCount*(vidx+3)+2] = sx + 1
		vs[graphics.VertexFloatCount*(vidx+3)+3] = sy + 1
		vs[graphics.VertexFloatCount*(vidx+3)+4] = crf
		vs[graphics.VertexFloatCount*(vidx+3)+5] = cgf
		vs[graphics.VertexFloatCount*(vidx+3)+6] = cbf
		vs[graphics.VertexFloatCount*(vidx+3)+7] = caf

		is[iidx] = uint32(vidx)
		is[iidx+1] = uint32(vidx + 1)
		is[iidx+2] = uint32(vidx + 2)
		is[iidx+3] = uint32(vidx + 1)
		is[iidx+4] = uint32(vidx + 2)
		is[iidx+5] = uint32(vidx + 3)

		idx++
	}

	srcs := [graphics.ShaderSrcImageCount]*ui.Image{whiteImage}
	dr := image.Rect(0, 0, c.width, c.height)
	sr := image.Rect(0, 0, whiteImageSize, whiteImageSize)
	blend := graphicsdriver.BlendCopy
	img.DrawTriangles(srcs, vs, is, blend, dr, [graphics.ShaderSrcImageCount]image.Rectangle{sr}, ui.NearestFilterShader, nil, true)

	clear(c.dots)
}

// reset clears the cached tiles after the GPU pixels change, keeping the allocated capacity for reuse.
// The dots must be empty and no tile must need a write-back, as they are pending writes that must not be discarded.
func (c *pixelCache) reset() {
	if len(c.dots) > 0 {
		panic("ebiten: the dots must be empty at reset")
	}
	for _, t := range c.tiles {
		if t.needsWriteBack {
			panic("ebiten: no tile must need a write-back at reset")
		}
	}
	clear(c.tiles)
	c.generation++
}

// deallocate clears the cache, releasing the allocations.
func (c *pixelCache) deallocate() {
	c.dots = nil
	c.tiles = nil
	c.generation++
}
