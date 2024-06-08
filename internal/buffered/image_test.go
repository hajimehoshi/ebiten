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

	// Add an entry for dotsBuffer at (0, 0).
	dst.WritePixels([]byte{0xff, 0xff, 0xff, 0xff}, image.Rect(0, 0, 1, 1))

	// Merge the entry into the cached pixels.
	// The entry for dotsBuffer is now gone in the current implementation.
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
	graphics.QuadVertices(vs, 0, 0, 16, 16, 1, 0, 0, 1, 0, 0, 1, 1, 1, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, 16, 16)
	sr := [graphics.ShaderImageCount]image.Rectangle{image.Rect(0, 0, 16, 16)}
	dst.DrawTriangles([graphics.ShaderImageCount]*buffered.Image{src}, vs, is, graphicsdriver.BlendSourceOver, dr, sr, atlas.NearestFilterShader, nil, graphicsdriver.FillAll)

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
