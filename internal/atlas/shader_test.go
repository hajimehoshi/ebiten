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
	"image/color"
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
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	g := ui.GraphicsDriverForTesting()
	s0 := atlas.NewShader(etesting.ShaderProgramFill(0xff, 0xff, 0xff, 0xff))
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{}, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, s0, nil, false)

	// Vertices must be recreated (#1755)
	vs = quadVertices(w, h, 0, 0, 1)
	s1 := atlas.NewShader(etesting.ShaderProgramFill(0x80, 0x80, 0x80, 0xff))
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{}, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, s1, nil, false)

	pix := make([]byte, 4*w*h)
	if err := dst.ReadPixels(g, pix, 0, 0, w, h); err != nil {
		t.Error(err)
	}
	if got, want := (color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}), (color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestImageDrawTwice(t *testing.T) {
	const w, h = 1, 1

	dst := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	src0 := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	src0.WritePixels([]byte{0xff, 0xff, 0xff, 0xff}, 0, 0, w, h)
	src1 := atlas.NewImage(w, h, atlas.ImageTypeRegular)
	src1.WritePixels([]byte{0x80, 0x80, 0x80, 0xff}, 0, 0, w, h)

	vs := quadVertices(w, h, 0, 0, 1)
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  w,
		Height: h,
	}
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src0}, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, atlas.NearestFilterShader, nil, false)

	// Vertices must be recreated (#1755)
	vs = quadVertices(w, h, 0, 0, 1)
	dst.DrawTriangles([graphics.ShaderImageCount]*atlas.Image{src1}, vs, is, graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, [graphics.ShaderImageCount - 1][2]float32{}, atlas.NearestFilterShader, nil, false)

	pix := make([]byte, 4*w*h)
	if err := dst.ReadPixels(ui.GraphicsDriverForTesting(), pix, 0, 0, w, h); err != nil {
		t.Error(err)
	}
	if got, want := (color.RGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}), (color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0xff}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}
