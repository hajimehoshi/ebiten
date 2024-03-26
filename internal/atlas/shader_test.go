// Copyright 2021 The Ebiten Authors
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

package atlas_test

import (
	"image"
	"image/color"
	"runtime"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/atlas"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	etesting "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestShaderFillTwice(t *testing.T) {
	const w, h = 1, 1

	dst := atlas.NewImage(w, h, atlas.ImageTypeRegular)

	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	g := ui.Get().GraphicsDriverForTesting()
	s0 := atlas.NewShader(etesting.ShaderProgramFill(0xff, 0xff, 0xff, 0xff))
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderImageCount]image.Rectangle{}, s0, nil, graphicsdriver.FillAll)

	// Vertices must be recreated (#1755)
	vs = quadVertices(w, h, 0, 0, 1)
	s1 := atlas.NewShader(etesting.ShaderProgramFill(0x80, 0x80, 0x80, 0xff))
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderImageCount]image.Rectangle{}, s1, nil, graphicsdriver.FillAll)

	pix := make([]byte, 4*w*h)
	ok, err := dst.ReadPixels(g, pix, image.Rect(0, 0, w, h))
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Fatal("ReadPixels failed")
	}
	if got, want := (color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}), (color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageDrawTwice(t *testing.T) {
	const w, h = 1, 1

	dst := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	src0 := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	src0.WritePixels([]byte{0xff, 0xff, 0xff, 0xff}, image.Rect(0, 0, w, h))
	src1 := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	src1.WritePixels([]byte{0x80, 0x80, 0x80, 0xff}, image.Rect(0, 0, w, h))

	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src0}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderImageCount]image.Rectangle{}, atlas.NearestFilterShader, nil, graphicsdriver.FillAll)

	// Vertices must be recreated (#1755)
	vs = quadVertices(w, h, 0, 0, 1)
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src1}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderImageCount]image.Rectangle{}, atlas.NearestFilterShader, nil, graphicsdriver.FillAll)

	pix := make([]byte, 4*w*h)
	ok, err := dst.ReadPixels(ui.Get().GraphicsDriverForTesting(), pix, image.Rect(0, 0, w, h))
	if err != nil {
		t.Error(err)
	}
	if !ok {
		t.Fatal("ReadPixels failed")
	}
	if got, want := (color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}), (color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestGCShader(t *testing.T) {
	s := atlas.NewShader(etesting.ShaderProgramFill(0xff, 0xff, 0xff, 0xff))

	// Use the shader to initialize it.
	const w, h = 1, 1
	dst := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderImageCount]image.Rectangle{}, s, nil, graphicsdriver.FillAll)

	// Ensure other objects are GCed, as GC appends deferred functions for collected objects.
	ensureGC()

	// Get the difference of the number of deferred functions before and after s is GCed.
	c := atlas.DeferredFuncCountForTesting()
	runtime.KeepAlive(s)
	ensureGC()

	diff := atlas.DeferredFuncCountForTesting() - c
	if got, want := diff, 1; got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}
}
