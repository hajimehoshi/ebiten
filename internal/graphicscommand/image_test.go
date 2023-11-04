// Copyright 2018 The Ebiten Authors
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

package graphicscommand_test

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	etesting "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

var nearestFilterShader *graphicscommand.Shader

func init() {
	ir, err := graphics.CompileShader([]byte(builtinshader.Shader(builtinshader.FilterNearest, builtinshader.AddressUnsafe, false)))
	if err != nil {
		panic(fmt.Sprintf("graphicscommand: compiling the nearest shader failed: %v", err))
	}
	nearestFilterShader = graphicscommand.NewShader(ir)
}

func TestMain(m *testing.M) {
	etesting.MainWithRunLoop(m)
}

func quadVertices(w, h float32) []float32 {
	return []float32{
		0, 0, 0, 0, 1, 1, 1, 1,
		w, 0, w, 0, 1, 1, 1, 1,
		0, w, 0, h, 1, 1, 1, 1,
		w, h, w, h, 1, 1, 1, 1,
	}
}

func TestClear(t *testing.T) {
	const w, h = 1024, 1024
	src := graphicscommand.NewImage(w/2, h/2, false)
	dst := graphicscommand.NewImage(w, h, false)

	vs := quadVertices(w/2, h/2)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	dst.DrawTriangles([graphics.ShaderImageCount]*graphicscommand.Image{src}, vs, is, graphicsdriver.BlendClear, dr, [graphics.ShaderImageCount]image.Rectangle{}, nearestFilterShader, nil, graphicsdriver.FillAll)

	pix := make([]byte, 4*w*h)
	if err := dst.ReadPixels(ui.Get().GraphicsDriverForTesting(), []graphicsdriver.PixelsArgs{
		{
			Pixels: pix,
			Region: image.Rect(0, 0, w, h),
		},
	}); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h/2; j++ {
		for i := 0; i < w/2; i++ {
			idx := 4 * (i + w*j)
			got := color.RGBA{R: pix[idx], G: pix[idx+1], B: pix[idx+2], A: pix[idx+3]}
			want := color.RGBA{}
			if got != want {
				t.Errorf("dst.At(%d, %d) after DrawTriangles: got %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestWritePixelsPartAfterDrawTriangles(t *testing.T) {
	const w, h = 32, 32
	clr := graphicscommand.NewImage(w, h, false)
	src := graphicscommand.NewImage(w/2, h/2, false)
	dst := graphicscommand.NewImage(w, h, false)
	vs := quadVertices(w/2, h/2)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	dst.DrawTriangles([graphics.ShaderImageCount]*graphicscommand.Image{clr}, vs, is, graphicsdriver.BlendClear, dr, [graphics.ShaderImageCount]image.Rectangle{}, nearestFilterShader, nil, graphicsdriver.FillAll)
	dst.DrawTriangles([graphics.ShaderImageCount]*graphicscommand.Image{src}, vs, is, graphicsdriver.BlendSourceOver, dr, [graphics.ShaderImageCount]image.Rectangle{}, nearestFilterShader, nil, graphicsdriver.FillAll)
	bs := graphics.NewManagedBytes(4, func(bs []byte) {
		for i := range bs {
			bs[i] = 0
		}
	})
	dst.WritePixels(bs, image.Rect(0, 0, 1, 1))

	// TODO: Check the result.
}

func TestShader(t *testing.T) {
	const w, h = 16, 16
	clr := graphicscommand.NewImage(w, h, false)
	dst := graphicscommand.NewImage(w, h, false)
	vs := quadVertices(w, h)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	dst.DrawTriangles([graphics.ShaderImageCount]*graphicscommand.Image{clr}, vs, is, graphicsdriver.BlendClear, dr, [graphics.ShaderImageCount]image.Rectangle{}, nearestFilterShader, nil, graphicsdriver.FillAll)

	g := ui.Get().GraphicsDriverForTesting()
	s := graphicscommand.NewShader(etesting.ShaderProgramFill(0xff, 0, 0, 0xff))
	dst.DrawTriangles([graphics.ShaderImageCount]*graphicscommand.Image{}, vs, is, graphicsdriver.BlendSourceOver, dr, [graphics.ShaderImageCount]image.Rectangle{}, s, nil, graphicsdriver.FillAll)

	pix := make([]byte, 4*w*h)
	if err := dst.ReadPixels(g, []graphicsdriver.PixelsArgs{
		{
			Pixels: pix,
			Region: image.Rect(0, 0, w, h),
		},
	}); err != nil {
		t.Fatal(err)
	}
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			idx := 4 * (i + w*j)
			got := color.RGBA{R: pix[idx], G: pix[idx+1], B: pix[idx+2], A: pix[idx+3]}
			want := color.RGBA{R: 0xff, A: 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d) after DrawTriangles: got %v, want: %v", i, j, got, want)
			}
		}
	}
}
