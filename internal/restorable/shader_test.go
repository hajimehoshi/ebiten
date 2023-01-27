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

package restorable_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/restorable"
	etesting "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func clearImage(img *restorable.Image, w, h int) {
	emptyImage := restorable.NewImage(3, 3, restorable.ImageTypeRegular)
	defer emptyImage.Dispose()

	dx0 := float32(0)
	dy0 := float32(0)
	dx1 := float32(w)
	dy1 := float32(h)
	sx0 := float32(1)
	sy0 := float32(1)
	sx1 := float32(2)
	sy1 := float32(2)
	vs := []float32{
		dx0, dy0, sx0, sy0, 0, 0, 0, 0,
		dx1, dy0, sx1, sy0, 0, 0, 0, 0,
		dx0, dy1, sx0, sy1, 0, 0, 0, 0,
		dx1, dy1, sx1, sy1, 0, 0, 0, 0,
	}
	is := graphics.QuadIndices()
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  float32(w),
		Height: float32(h),
	}
	img.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{emptyImage}, [graphics.ShaderImageCount - 1][2]float32{}, vs, is, graphicsdriver.BlendClear, dr, graphicsdriver.Region{}, restorable.NearestFilterShader, nil, false)
}

func TestShader(t *testing.T) {
	img := restorable.NewImage(1, 1, restorable.ImageTypeRegular)
	defer img.Dispose()

	s := restorable.NewShader(etesting.ShaderProgramFill(0xff, 0, 0, 0xff))
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  1,
		Height: 1,
	}
	img.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(nil, 1, 1, 0, 0), graphics.QuadIndices(), graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, s, nil, false)

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	want := color.RGBA{R: 0xff, A: 0xff}
	got := pixelsToColor(img.BasePixelsForTesting(), 0, 0, 1, 1)
	if !sameColors(got, want, 1) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestShaderChain(t *testing.T) {
	const num = 10
	imgs := []*restorable.Image{}
	for i := 0; i < num; i++ {
		img := restorable.NewImage(1, 1, restorable.ImageTypeRegular)
		defer img.Dispose()
		imgs = append(imgs, img)
	}

	imgs[0].WritePixels([]byte{0xff, 0, 0, 0xff}, 0, 0, 1, 1)

	s := restorable.NewShader(etesting.ShaderProgramImages(1))
	for i := 0; i < num-1; i++ {
		dr := graphicsdriver.Region{
			X:      0,
			Y:      0,
			Width:  1,
			Height: 1,
		}
		imgs[i+1].DrawTriangles([graphics.ShaderImageCount]*restorable.Image{imgs[i]}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(imgs[i], 1, 1, 0, 0), graphics.QuadIndices(), graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, s, nil, false)
	}

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	for i, img := range imgs {
		want := color.RGBA{R: 0xff, A: 0xff}
		got := pixelsToColor(img.BasePixelsForTesting(), 0, 0, 1, 1)
		if !sameColors(got, want, 1) {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}
}

func TestShaderMultipleSources(t *testing.T) {
	var srcs [graphics.ShaderImageCount]*restorable.Image
	for i := range srcs {
		srcs[i] = restorable.NewImage(1, 1, restorable.ImageTypeRegular)
	}
	srcs[0].WritePixels([]byte{0x40, 0, 0, 0xff}, 0, 0, 1, 1)
	srcs[1].WritePixels([]byte{0, 0x80, 0, 0xff}, 0, 0, 1, 1)
	srcs[2].WritePixels([]byte{0, 0, 0xc0, 0xff}, 0, 0, 1, 1)

	dst := restorable.NewImage(1, 1, restorable.ImageTypeRegular)

	s := restorable.NewShader(etesting.ShaderProgramImages(3))
	var offsets [graphics.ShaderImageCount - 1][2]float32
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  1,
		Height: 1,
	}
	dst.DrawTriangles(srcs, offsets, quadVertices(srcs[0], 1, 1, 0, 0), graphics.QuadIndices(), graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, s, nil, false)

	// Clear one of the sources after DrawTriangles. dst should not be affected.
	clearImage(srcs[0], 1, 1)

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	want := color.RGBA{R: 0x40, G: 0x80, B: 0xc0, A: 0xff}
	got := pixelsToColor(dst.BasePixelsForTesting(), 0, 0, 1, 1)
	if !sameColors(got, want, 1) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestShaderMultipleSourcesOnOneTexture(t *testing.T) {
	src := restorable.NewImage(3, 1, restorable.ImageTypeRegular)
	src.WritePixels([]byte{
		0x40, 0, 0, 0xff,
		0, 0x80, 0, 0xff,
		0, 0, 0xc0, 0xff,
	}, 0, 0, 3, 1)
	srcs := [graphics.ShaderImageCount]*restorable.Image{src, src, src}

	dst := restorable.NewImage(1, 1, restorable.ImageTypeRegular)

	s := restorable.NewShader(etesting.ShaderProgramImages(3))
	offsets := [graphics.ShaderImageCount - 1][2]float32{
		{1, 0},
		{2, 0},
	}
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  1,
		Height: 1,
	}
	dst.DrawTriangles(srcs, offsets, quadVertices(srcs[0], 1, 1, 0, 0), graphics.QuadIndices(), graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, s, nil, false)

	// Clear one of the sources after DrawTriangles. dst should not be affected.
	clearImage(srcs[0], 3, 1)

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	want := color.RGBA{R: 0x40, G: 0x80, B: 0xc0, A: 0xff}
	got := pixelsToColor(dst.BasePixelsForTesting(), 0, 0, 1, 1)
	if !sameColors(got, want, 1) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestShaderDispose(t *testing.T) {
	img := restorable.NewImage(1, 1, restorable.ImageTypeRegular)
	defer img.Dispose()

	s := restorable.NewShader(etesting.ShaderProgramFill(0xff, 0, 0, 0xff))
	dr := graphicsdriver.Region{
		X:      0,
		Y:      0,
		Width:  1,
		Height: 1,
	}
	img.DrawTriangles([graphics.ShaderImageCount]*restorable.Image{}, [graphics.ShaderImageCount - 1][2]float32{}, quadVertices(nil, 1, 1, 0, 0), graphics.QuadIndices(), graphicsdriver.BlendCopy, dr, graphicsdriver.Region{}, s, nil, false)

	// Dispose the shader. This should invalidate all the images using this shader i.e., all the images become
	// stale.
	s.Dispose()

	if err := restorable.ResolveStaleImages(ui.GraphicsDriverForTesting(), false); err != nil {
		t.Fatal(err)
	}
	if err := restorable.RestoreIfNeeded(ui.GraphicsDriverForTesting()); err != nil {
		t.Fatal(err)
	}

	want := color.RGBA{R: 0xff, A: 0xff}
	got := pixelsToColor(img.BasePixelsForTesting(), 0, 0, 1, 1)
	if !sameColors(got, want, 1) {
		t.Errorf("got %v, want %v", got, want)
	}
}
