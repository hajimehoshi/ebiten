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
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	etesting "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

var nearestFilterShader *graphicscommand.Shader

func init() {
	ir, err := graphics.CompileShader([]byte(builtinshader.ShaderSource(builtinshader.FilterNearest, builtinshader.AddressUnsafe)))
	if err != nil {
		panic(fmt.Sprintf("graphicscommand: compiling the nearest shader failed: %v", err))
	}
	nearestFilterShader = graphicscommand.NewShader(ir, "")
}

func TestMain(m *testing.M) {
	etesting.MainWithRunLoop(m)
}

func quadVertices(w, h float32) []float32 {
	vs := make([]float32, 8*graphics.VertexFloatCount)
	graphics.QuadVerticesFromDstAndSrc(vs, 0, 0, w, h, 0, 0, w, h, 1, 1, 1, 1)
	return vs
}

func TestClear(t *testing.T) {
	const w, h = 1024, 1024
	src := graphicscommand.NewImage(w/2, h/2, false, "")
	dst := graphicscommand.NewImage(w, h, false, "")

	vs := quadVertices(w/2, h/2)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	sr := image.Rect(0, 0, w/2, h/2)
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*graphicscommand.Image{src}, vs, is, graphicsdriver.BlendClear, dr, [graphics.ShaderSrcImageCount]image.Rectangle{sr}, nearestFilterShader, nil)

	pix := make([]byte, 4*w*h)
	if err := dst.ReadPixels(ui.Get().GraphicsDriverForTesting(), []graphicsdriver.PixelsArgs{
		{
			Pixels: pix,
			Region: image.Rect(0, 0, w, h),
		},
	}); err != nil {
		t.Fatal(err)
	}
	for j := range h / 2 {
		for i := range w / 2 {
			idx := 4 * (i + w*j)
			got := color.RGBA{R: pix[idx], G: pix[idx+1], B: pix[idx+2], A: pix[idx+3]}
			var want color.RGBA
			if got != want {
				t.Errorf("dst.At(%d, %d) after DrawTriangles: got %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestWritePixelsPartAfterDrawTriangles(t *testing.T) {
	const w, h = 32, 32
	clr := graphicscommand.NewImage(w, h, false, "")
	src := graphicscommand.NewImage(w/2, h/2, false, "")
	dst := graphicscommand.NewImage(w, h, false, "")
	vs := quadVertices(w/2, h/2)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	sr0 := image.Rect(0, 0, w, h)
	sr1 := image.Rect(0, 0, w/2, h/2)
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*graphicscommand.Image{clr}, vs, is, graphicsdriver.BlendClear, dr, [graphics.ShaderSrcImageCount]image.Rectangle{sr0}, nearestFilterShader, nil)
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*graphicscommand.Image{src}, vs, is, graphicsdriver.BlendSourceOver, dr, [graphics.ShaderSrcImageCount]image.Rectangle{sr1}, nearestFilterShader, nil)
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
	clr := graphicscommand.NewImage(w, h, false, "")
	dst := graphicscommand.NewImage(w, h, false, "")
	vs := quadVertices(w, h)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)
	sr := image.Rect(0, 0, w, h)
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*graphicscommand.Image{clr}, vs, is, graphicsdriver.BlendClear, dr, [graphics.ShaderSrcImageCount]image.Rectangle{sr}, nearestFilterShader, nil)

	g := ui.Get().GraphicsDriverForTesting()
	s := graphicscommand.NewShader(etesting.ShaderProgramFill(0xff, 0, 0, 0xff), "")
	dst.DrawTriangles([graphics.ShaderSrcImageCount]*graphicscommand.Image{}, vs, is, graphicsdriver.BlendSourceOver, dr, [graphics.ShaderSrcImageCount]image.Rectangle{}, s, nil)

	pix := make([]byte, 4*w*h)
	if err := dst.ReadPixels(g, []graphicsdriver.PixelsArgs{
		{
			Pixels: pix,
			Region: image.Rect(0, 0, w, h),
		},
	}); err != nil {
		t.Fatal(err)
	}
	for j := range h {
		for i := range w {
			idx := 4 * (i + w*j)
			got := color.RGBA{R: pix[idx], G: pix[idx+1], B: pix[idx+2], A: pix[idx+3]}
			want := color.RGBA{R: 0xff, A: 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d) after DrawTriangles: got %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #3036
func TestSuccessiveWritePixels(t *testing.T) {
	const w, h = 32, 32
	dst := graphicscommand.NewImage(w, h, false, "")

	dst.WritePixels(graphics.NewManagedBytes(4, func(bs []byte) {
		for i := range bs {
			bs[i] = 0
		}
	}), image.Rect(0, 0, 1, 1))
	if got, want := len(dst.BufferedWritePixelsArgsForTesting()), 1; got != want {
		t.Errorf("len(dst.BufferedWritePixelsArgsForTesting()): got %d, want: %d", got, want)
	}

	dst.WritePixels(graphics.NewManagedBytes(4, func(bs []byte) {
		for i := range bs {
			bs[i] = 0
		}
	}), image.Rect(1, 1, 2, 2))
	if got, want := len(dst.BufferedWritePixelsArgsForTesting()), 2; got != want {
		t.Errorf("len(dst.BufferedWritePixelsArgsForTesting()): got %d, want: %d", got, want)
	}

	dst.WritePixels(graphics.NewManagedBytes(4, func(bs []byte) {
		for i := range bs {
			bs[i] = 0
		}
	}), image.Rect(0, 0, 1, 1))
	if got, want := len(dst.BufferedWritePixelsArgsForTesting()), 2; got != want {
		t.Errorf("len(dst.BufferedWritePixelsArgsForTesting()): got %d, want: %d", got, want)
	}

	dst.WritePixels(graphics.NewManagedBytes(4, func(bs []byte) {
		for i := range bs {
			bs[i] = 0
		}
	}), image.Rect(0, 0, 1, 1))
	if got, want := len(dst.BufferedWritePixelsArgsForTesting()), 2; got != want {
		t.Errorf("len(dst.BufferedWritePixelsArgsForTesting()): got %d, want: %d", got, want)
	}

	dst.WritePixels(graphics.NewManagedBytes(4, func(bs []byte) {
		for i := range bs {
			bs[i] = 0
		}
	}), image.Rect(0, 0, 2, 2))
	if got, want := len(dst.BufferedWritePixelsArgsForTesting()), 1; got != want {
		t.Errorf("len(dst.BufferedWritePixelsArgsForTesting()): got %d, want: %d", got, want)
	}
}

func flushEndFrame(t *testing.T) {
	t.Helper()
	if err := graphicscommand.FlushCommands(ui.Get().GraphicsDriverForTesting(), graphicsdriver.FlushModeEndFrame); err != nil {
		t.Fatal(err)
	}
}

func requireReadPixelsAsyncResult(t *testing.T, result *graphicscommand.PixelReadback) error {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if done, err := result.Poll(); done {
			return err
		}
		flushEndFrame(t)
		time.Sleep(time.Millisecond)
	}
	return fmt.Errorf("test: read-back timed out")
}

func fillImage(t *testing.T, img *graphicscommand.Image, clr color.RGBA) {
	t.Helper()
	const w, h = 16, 16
	pix := make([]byte, 4*w*h)
	for i := range len(pix) / 4 {
		pix[4*i] = clr.R
		pix[4*i+1] = clr.G
		pix[4*i+2] = clr.B
		pix[4*i+3] = clr.A
	}
	img.WritePixels(graphics.NewManagedBytes(len(pix), func(bs []byte) {
		copy(bs, pix)
	}), image.Rect(0, 0, w, h))
}

func TestReadPixelsAsyncWithRealDriver(t *testing.T) {
	const w, h = 16, 16
	region := image.Rect(0, 0, w, h)

	t.Run("ReusedBuffersInFlight", func(t *testing.T) {
		const count = 4
		for round := range 2 {
			pixels := make([][]byte, count)
			results := make([]*graphicscommand.PixelReadback, count)
			for i := range count {
				img := graphicscommand.NewImage(w, h, false, "")
				fillImage(t, img, color.RGBA{R: byte(0x40 + count*round + i), A: 0xff})
				pixels[i] = make([]byte, 4*w*h)
				results[i] = img.ReadPixelsAsync([]graphicsdriver.PixelsArgs{{
					Pixels: pixels[i],
					Region: region,
				}})
			}
			for i, result := range results {
				if err := requireReadPixelsAsyncResult(t, result); err != nil {
					t.Errorf("round %d, read-back %d: %v", round, i, err)
					continue
				}
				if got, want := pixels[i][0], byte(0x40+count*round+i); got != want {
					t.Errorf("round %d, read-back %d: pixels[0] = %#x, want %#x", round, i, got, want)
				}
			}
		}
	})

	t.Run("MultipleRegions", func(t *testing.T) {
		const w, h = 17, 11
		region := image.Rect(0, 0, w, h)
		img := graphicscommand.NewImage(w, h, false, "")
		pixels := graphics.NewManagedBytes(4*w*h, func(bs []byte) {
			for y := range h {
				for x := range w {
					i := 4 * (y*w + x)
					copy(bs[i:], []byte{byte(x), byte(y), 123, 255})
				}
			}
		})
		img.WritePixels(pixels, region)
		args := []graphicsdriver.PixelsArgs{
			{
				Pixels: make([]byte, 3*5*4),
				Region: image.Rect(2, 3, 5, 8),
			},
			{
				Pixels: make([]byte, w*h*4),
				Region: region,
			},
		}
		result := img.ReadPixelsAsync(args)
		img.WritePixels(graphics.NewManagedBytes(4*w*h, func(bs []byte) {
			clear(bs)
		}), region)
		cleared := bytes.Repeat([]byte{0xff}, 4*w*h)
		clearedCh := img.ReadPixelsAsync([]graphicsdriver.PixelsArgs{
			{
				Pixels: cleared,
				Region: region,
			},
		})
		img.Dispose()
		if err := requireReadPixelsAsyncResult(t, result); err != nil {
			t.Error(err)
			return
		}
		if err := requireReadPixelsAsyncResult(t, clearedCh); err != nil {
			t.Error(err)
			return
		}
		if !bytes.Equal(cleared, make([]byte, len(cleared))) {
			t.Error("the cleared image must be all zeros")
		}
		for _, arg := range args {
			for y := arg.Region.Min.Y; y < arg.Region.Max.Y; y++ {
				for x := arg.Region.Min.X; x < arg.Region.Max.X; x++ {
					i := 4 * ((y-arg.Region.Min.Y)*arg.Region.Dx() + x - arg.Region.Min.X)
					want := []byte{byte(x), byte(y), 123, 255}
					if got := arg.Pixels[i : i+4]; !bytes.Equal(got, want) {
						t.Errorf("region %v at (%d, %d): got %v, want %v", arg.Region, x, y, got, want)
					}
				}
			}
		}
	})

	t.Run("ManyInFlight", func(t *testing.T) {
		const count = 16
		clrs := make([]color.RGBA, count)
		pixels := make([][]byte, count)
		results := make([]*graphicscommand.PixelReadback, count)
		for i := range count {
			clr := color.RGBA{R: byte(0x40 + i), G: byte(0x80 + i), B: byte(0xc0 + i), A: 0xff}
			clrs[i] = clr
			img := graphicscommand.NewImage(w, h, false, "")
			fillImage(t, img, clr)
			pixels[i] = make([]byte, 4*w*h)
			results[i] = img.ReadPixelsAsync([]graphicsdriver.PixelsArgs{{
				Pixels: pixels[i],
				Region: region,
			}})
		}
		for i, result := range results {
			if err := requireReadPixelsAsyncResult(t, result); err != nil {
				t.Errorf("read-back %d: %v", i, err)
				continue
			}
			want := []byte{clrs[i].R, clrs[i].G, clrs[i].B, clrs[i].A}
			for j, p := range pixels[i] {
				if p != want[j%4] {
					t.Errorf("read-back %d: pixels[%d] = %#x, want %#x", i, j, p, want[j%4])
				}
			}
		}
	})

	t.Run("IgnoredResults", func(t *testing.T) {
		img := graphicscommand.NewImage(w, h, false, "")
		fillImage(t, img, color.RGBA{R: 0x55, A: 0xff})

		const count = 32
		for range count {
			img.ReadPixelsAsync([]graphicsdriver.PixelsArgs{{
				Pixels: make([]byte, 4*w*h),
				Region: region,
			}})
		}
		flushEndFrame(t)

		pix := make([]byte, 4*w*h)
		result := img.ReadPixelsAsync([]graphicsdriver.PixelsArgs{{
			Pixels: pix,
			Region: region,
		}})
		if err := requireReadPixelsAsyncResult(t, result); err != nil {
			t.Error(err)
			return
		}
		if p := pix[0]; p != 0x55 {
			t.Errorf("pixels[0] = %#x, want %#x", p, byte(0x55))
		}
	})
}

func TestShaderUniformsAfterFramesWithoutPresenting(t *testing.T) {
	const w, h = 16, 16
	g := ui.Get().GraphicsDriverForTesting()

	ir, err := graphics.CompileShader([]byte(`//kage:unit pixels

package main

var Color vec4

func Fragment(dstPos vec4, src0Pos vec2, color vec4) vec4 {
	return Color
}
`))
	if err != nil {
		t.Fatal(err)
	}
	s := graphicscommand.NewShader(ir, "")
	dst := graphicscommand.NewImage(w, h, false, "")
	vs := quadVertices(w, h)
	is := graphics.QuadIndices()
	dr := image.Rect(0, 0, w, h)

	endFrame := func() {
		t.Helper()
		flushEndFrame(t)
		if err := graphicscommand.FlushAndPollPixelReadbacks(g); err != nil {
			t.Fatal(err)
		}
	}
	endFrame()

	clrs := []color.RGBA{
		{R: 0xff, A: 0xff},
		{R: 0xff, A: 0xff},
		{G: 0xff, A: 0xff},
	}
	for i, clr := range clrs {
		if i > 0 {
			endFrame()
		}
		uniforms := []uint32{
			math.Float32bits(float32(clr.R) / 0xff),
			math.Float32bits(float32(clr.G) / 0xff),
			math.Float32bits(float32(clr.B) / 0xff),
			math.Float32bits(float32(clr.A) / 0xff),
		}
		dst.DrawTriangles([graphics.ShaderSrcImageCount]*graphicscommand.Image{}, vs, is, graphicsdriver.BlendCopy, dr, [graphics.ShaderSrcImageCount]image.Rectangle{}, s, uniforms)
	}

	pix := make([]byte, 4*w*h)
	if err := dst.ReadPixels(g, []graphicsdriver.PixelsArgs{
		{
			Pixels: pix,
			Region: dr,
		},
	}); err != nil {
		t.Fatal(err)
	}
	want := clrs[len(clrs)-1]
	for j := range h {
		for i := range w {
			idx := 4 * (i + w*j)
			got := color.RGBA{R: pix[idx], G: pix[idx+1], B: pix[idx+2], A: pix[idx+3]}
			if got != want {
				t.Errorf("dst.At(%d, %d) after %d frames ending without presenting: got %v, want: %v", i, j, len(clrs)-1, got, want)
			}
		}
	}
}
