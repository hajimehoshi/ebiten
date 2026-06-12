// Copyright 2024 The Ebitengine Authors
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

package buffered_test

import (
	"bytes"
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/buffered"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestMain(m *testing.M) {
	t.MainWithRunLoop(m)
}

func TestUnsyncedPixels(t *testing.T) {
	dst := buffered.NewImage(16, 16, atlas.ImageTypeRegular)

	// Add an entry for dots at (0, 0).
	dst.WritePixels([]byte{0xff, 0xff, 0xff, 0xff}, image.Rect(0, 0, 1, 1))

	// Merge the entry into the cached pixels.
	// The entry for dots is now gone in the current implementation.
	ok, err := dst.ReadPixels(ui.Get().GraphicsDriverForTesting(), make([]byte, 4*16*16), image.Rect(0, 0, 16, 16))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ReadPixels failed")
	}

	// Call WritePixels with the outside region of (0, 0).
	dst.WritePixels(make([]byte, 4*2*2), image.Rect(1, 1, 3, 3))

	// Flush unsynced pixel cache.
	src := buffered.NewImage(16, 16, atlas.ImageTypeRegular)
	vs := make([]float32, 4*graphics.VertexFloatCount)
	graphics.QuadVerticesFromDstAndSrc(vs, 0, 0, 16, 16, 0, 0, 16, 16, 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, 16, 16)
	sr := [graphics.ShaderSrcImageCount]image.Rectangle{image.Rect(0, 0, 16, 16)}
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*buffered.Image{src}, vs, is, graphicsdriver.BlendSourceOver, dr, sr, atlas.NearestFilterShader, nil)

	// Check the result is correct.
	var got [4]byte
	ok, err = dst.ReadPixels(ui.Get().GraphicsDriverForTesting(), got[:], image.Rect(0, 0, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ReadPixels failed")
	}
	want := [4]byte{0xff, 0xff, 0xff, 0xff}
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestReadPixelsOnLargeImage(t *testing.T) {
	// A pixel cache for an image whose whole pixel data exceeds MaxPixelsCacheSize is divided into tiles.
	const width = 1024
	height := buffered.MaxPixelsCacheSize/(4*width) + 1
	img := buffered.NewImage(width, height, atlas.ImageTypeRegular)

	// Write a 2x2 block at (2, 2). This is written to the GPU directly.
	pix := make([]byte, 4*2*2)
	for i := range pix {
		pix[i] = 0x40
	}
	img.WritePixels(pix, image.Rect(2, 2, 4, 4))

	// Add an entry for dots at (3, 3).
	img.WritePixels([]byte{0xff, 0xff, 0xff, 0xff}, image.Rect(3, 3, 4, 4))

	want := make([]byte, 4*4*4)
	for _, p := range []image.Point{{2, 2}, {3, 2}, {2, 3}} {
		idx := 4 * ((p.Y-1)*4 + (p.X - 1))
		for i := range 4 {
			want[idx+i] = 0x40
		}
	}
	idx := 4 * ((3-1)*4 + (3 - 1))
	for i := range 4 {
		want[idx+i] = 0xff
	}

	// Read the same region twice. The entry for dots must be applied to the result both times.
	for range 2 {
		got := make([]byte, 4*4*4)
		ok, err := img.ReadPixels(ui.Get().GraphicsDriverForTesting(), got, image.Rect(1, 1, 5, 5))
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("ReadPixels failed")
		}
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	// Only the tile containing the read region must be cached.
	if got, want := img.CachedPixelsSizeForTesting(), 4*buffered.DividedTileSize*buffered.DividedTileSize; got != want {
		t.Errorf("cached pixels size: got: %d, want: %d", got, want)
	}
}

func TestReadPixelsOnLargeImageAcrossTiles(t *testing.T) {
	const width = 1024
	height := buffered.MaxPixelsCacheSize/(4*width) + 1
	img := buffered.NewImage(width, height, atlas.ImageTypeRegular)

	// Write a 4x4 block crossing the tile boundaries at x=256 and y=256. This is written to the GPU directly.
	pix := make([]byte, 4*4*4)
	for i := range pix {
		pix[i] = 0x40
	}
	img.WritePixels(pix, image.Rect(254, 254, 258, 258))

	// Add an entry for dots at (256, 256).
	img.WritePixels([]byte{0xff, 0xff, 0xff, 0xff}, image.Rect(256, 256, 257, 257))

	// Reading one pixel caches only the tile containing it.
	var dot [4]byte
	ok, err := img.ReadPixels(ui.Get().GraphicsDriverForTesting(), dot[:], image.Rect(0, 0, 1, 1))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("ReadPixels failed")
	}
	if dot != ([4]byte{}) {
		t.Errorf("got: %v, want: %v", dot, [4]byte{})
	}
	if got, want := img.CachedPixelsSizeForTesting(), 4*buffered.DividedTileSize*buffered.DividedTileSize; got != want {
		t.Errorf("cached pixels size: got: %d, want: %d", got, want)
	}

	want := make([]byte, 4*8*8)
	for y := 254; y < 258; y++ {
		for x := 254; x < 258; x++ {
			v := byte(0x40)
			if x == 256 && y == 256 {
				v = 0xff
			}
			idx := 4 * ((y-252)*8 + (x - 252))
			for i := range 4 {
				want[idx+i] = v
			}
		}
	}

	// Read the same region twice. The entry for dots must be applied to the result both times.
	for range 2 {
		got := make([]byte, 4*8*8)
		ok, err := img.ReadPixels(ui.Get().GraphicsDriverForTesting(), got, image.Rect(252, 252, 260, 260))
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Fatal("ReadPixels failed")
		}
		if !bytes.Equal(got, want) {
			t.Errorf("got: %v, want: %v", got, want)
		}
	}

	// The four tiles around (256, 256), one of which was already cached by the first read, must be cached.
	if got, want := img.CachedPixelsSizeForTesting(), 4*4*buffered.DividedTileSize*buffered.DividedTileSize; got != want {
		t.Errorf("cached pixels size: got: %d, want: %d", got, want)
	}
}
