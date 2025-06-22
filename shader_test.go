// Copyright 2020 The Ebiten Authors
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

package ebiten_test

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
)

func TestShaderFill(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w/2, h/2, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if i < w/2 && j < h/2 {
				want = color.RGBA{R: 0xff, A: 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderFillWithDrawImage(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	src := ebiten.NewImage(w/2, h/2)
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dst.DrawRectShader(w/2, h/2, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if i < w/2 && j < h/2 {
				want = color.RGBA{R: 0xff, A: 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2525
func TestShaderWithDrawImageDoesNotWreckTextureUnits(t *testing.T) {
	const w, h = 16, 16
	rect := image.Rectangle{Max: image.Point{X: w, Y: h}}

	dst := ebiten.NewImageWithOptions(rect, &ebiten.NewImageOptions{Unmanaged: true})
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return imageSrc0At(srcPos)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	src0 := ebiten.NewImageWithOptions(rect, &ebiten.NewImageOptions{Unmanaged: true})
	src0.Fill(color.RGBA{R: 25, G: 0xff, B: 25, A: 0xff})
	src1 := ebiten.NewImageWithOptions(rect, &ebiten.NewImageOptions{Unmanaged: true})
	src1.Fill(color.RGBA{R: 0xff, A: 0xff})
	op := &ebiten.DrawRectShaderOptions{}
	op.CompositeMode = ebiten.CompositeModeCopy
	op.Images[0] = src0
	op.Images[1] = src1
	dst.DrawRectShader(w, h, s, op)
	op.Images[0] = src1
	op.Images[1] = nil
	dst.DrawRectShader(w, h, s, op) // dst should now be identical to src1.

	// With issue #2525, instead, GL_TEXTURE0 is active but with src0 bound
	// while binding src1 gets skipped!
	// This means that src0, not src1, got copied to dst.

	// Demonstrate the bug with a write to src1, which will actually end up on src0.
	// Validated later.
	var buf []byte
	for i := 0; i < w*h; i++ {
		buf = append(buf, 2, 5, 2, 5)
	}
	src1.WritePixels(buf)

	// Verify that src1 was copied to dst.
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, A: 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// Fix up texture unit assignment by binding a different texture.
	op.Images[0] = src1
	dst.DrawRectShader(w, h, s, op)
	op.Images[0] = src0
	dst.DrawRectShader(w, h, s, op)

	// Verify that src0 was copied to dst and not overwritten above.
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 25, G: 0xff, B: 25, A: 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderFillWithDrawTriangles(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	src := ebiten.NewImage(w/2, h/2)
	op := &ebiten.DrawTrianglesShaderOptions{}
	op.Images[0] = src

	vs := []ebiten.Vertex{
		{
			DstX:   0,
			DstY:   0,
			SrcX:   0,
			SrcY:   0,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   w,
			DstY:   0,
			SrcX:   w / 2,
			SrcY:   0,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   0,
			DstY:   h,
			SrcX:   0,
			SrcY:   h / 2,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
		{
			DstX:   w,
			DstY:   h,
			SrcX:   w / 2,
			SrcY:   h / 2,
			ColorR: 1,
			ColorG: 1,
			ColorB: 1,
			ColorA: 1,
		},
	}
	is := []uint16{0, 1, 2, 1, 2, 3}

	dst.DrawTrianglesShader(vs, is, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, A: 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderFunction(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func clr(red float) (float, float, float, float) {
	return red, 0, 0, 1
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(clr(1))
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, A: 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderUninitializedUniformVariables(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var U vec4

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return U
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderMatrix(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var a, b mat4
	a[0] = vec4(0.125, 0.0625, 0.0625, 0.0625)
	a[1] = vec4(0.25, 0.25, 0.0625, 0.1875)
	a[2] = vec4(0.1875, 0.125, 0.25, 0.25)
	a[3] = vec4(0.0625, 0.1875, 0.125, 0.25)
	b[0] = vec4(0.0625, 0.125, 0.0625, 0.125)
	b[1] = vec4(0.125, 0.1875, 0.25, 0.0625)
	b[2] = vec4(0.125, 0.125, 0.1875, 0.1875)
	b[3] = vec4(0.25, 0.0625, 0.125, 0.0625)
	return vec4((a * b * vec4(1, 1, 1, 1)).xyz, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	src := ebiten.NewImage(w, h)
	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 87, G: 82, B: 71, A: 255}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderSubImage(t *testing.T) {
	const w, h = 16, 16

	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	r := imageSrc0At(srcPos).r
	g := imageSrc1At(srcPos).g
	return vec4(r, g, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	src0 := ebiten.NewImage(w, h)
	pix0 := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if 2 <= i && i < 10 && 3 <= j && j < 11 {
				pix0[4*(j*w+i)] = 0xff
				pix0[4*(j*w+i)+1] = 0
				pix0[4*(j*w+i)+2] = 0
				pix0[4*(j*w+i)+3] = 0xff
			}
		}
	}
	src0.WritePixels(pix0)
	src0 = src0.SubImage(image.Rect(2, 3, 10, 11)).(*ebiten.Image)

	src1 := ebiten.NewImage(w, h)
	pix1 := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if 6 <= i && i < 14 && 8 <= j && j < 16 {
				pix1[4*(j*w+i)] = 0
				pix1[4*(j*w+i)+1] = 0xff
				pix1[4*(j*w+i)+2] = 0
				pix1[4*(j*w+i)+3] = 0xff
			}
		}
	}
	src1.WritePixels(pix1)
	src1 = src1.SubImage(image.Rect(6, 8, 14, 16)).(*ebiten.Image)

	testPixels := func(testname string, dst *ebiten.Image) {
		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				got := dst.At(i, j).(color.RGBA)
				var want color.RGBA
				if i < w/2 && j < h/2 {
					want = color.RGBA{R: 0xff, G: 0xff, A: 0xff}
				}
				if got != want {
					t.Errorf("%s dst.At(%d, %d): got: %v, want: %v", testname, i, j, got, want)
				}
			}
		}
	}

	t.Run("DrawRectShader", func(t *testing.T) {
		dst := ebiten.NewImage(w, h)
		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = src0
		op.Images[1] = src1
		dst.DrawRectShader(w/2, h/2, s, op)
		testPixels("DrawRectShader", dst)
	})

	t.Run("DrawTrianglesShader", func(t *testing.T) {
		dst := ebiten.NewImage(w, h)
		vs := []ebiten.Vertex{
			{
				DstX:   0,
				DstY:   0,
				SrcX:   2,
				SrcY:   3,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   w / 2,
				DstY:   0,
				SrcX:   10,
				SrcY:   3,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   0,
				DstY:   h / 2,
				SrcX:   2,
				SrcY:   11,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   w / 2,
				DstY:   h / 2,
				SrcX:   10,
				SrcY:   11,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
		}
		is := []uint16{0, 1, 2, 1, 2, 3}

		op := &ebiten.DrawTrianglesShaderOptions{}
		op.Images[0] = src0
		op.Images[1] = src1
		dst.DrawTrianglesShader(vs, is, s, op)
		testPixels("DrawTrianglesShader", dst)
	})
}

// Issue #1404
func TestShaderDerivatives(t *testing.T) {
	t.Skip("the results of dfdx, dfdy, and fwidth are indeterministic (#2583)")

	const w, h = 16, 16

	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	p := imageSrc0At(srcPos)
	return vec4(abs(dfdx(p.r)), abs(dfdy(p.g)), 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if i < w/2 {
				pix[4*(j*w+i)] = 0xff
			}
			if j < h/2 {
				pix[4*(j*w+i)+1] = 0xff
			}
			pix[4*(j*w+i)+3] = 0xff
		}
	}
	src.WritePixels(pix)

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dst.DrawRectShader(w, h, s, op)

	// The results of the edges might be unreliable. Skip the edges.
	for j := 1; j < h-1; j++ {
		for i := 1; i < w-1; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{A: 0xff}
			if i == w/2-1 || i == w/2 {
				want.R = 0xff
			}
			if j == h/2-1 || j == h/2 {
				want.G = 0xff
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #1701
func TestShaderDerivatives2(t *testing.T) {
	t.Skip("the results of dfdx, dfdy, and fwidth are indeterministic (#2583)")

	const w, h = 16, 16

	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

// This function uses dfdx and then should not be in GLSL's vertex shader (#1701).
func Foo(p vec4) vec4 {
	return vec4(abs(dfdx(p.r)), abs(dfdy(p.g)), 0, 1)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	p := imageSrc0At(srcPos)
	return Foo(p)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)
	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if i < w/2 {
				pix[4*(j*w+i)] = 0xff
			}
			if j < h/2 {
				pix[4*(j*w+i)+1] = 0xff
			}
			pix[4*(j*w+i)+3] = 0xff
		}
	}
	src.WritePixels(pix)

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dst.DrawRectShader(w, h, s, op)

	// The results of the edges might be unreliable. Skip the edges.
	for j := 1; j < h-1; j++ {
		for i := 1; i < w-1; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{A: 0xff}
			if i == w/2-1 || i == w/2 {
				want.R = 0xff
			}
			if j == h/2-1 || j == h/2 {
				want.G = 0xff
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #1754
func TestShaderUniformFirstElement(t *testing.T) {
	shaders := []struct {
		Name     string
		Shader   string
		Uniforms map[string]any
	}{
		{
			Name: "float array",
			Shader: `//kage:unit pixels

package main

var C [2]float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(C[0], 1, 1, 1)
}`,
			Uniforms: map[string]any{
				"C": []float32{1, 1},
			},
		},
		{
			Name: "float one-element array",
			Shader: `//kage:unit pixels

package main

var C [1]float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(C[0], 1, 1, 1)
}`,
			Uniforms: map[string]any{
				"C": []float32{1},
			},
		},
		{
			Name: "matrix array",
			Shader: `//kage:unit pixels

package main

var C [2]mat2

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(C[0][0][0], 1, 1, 1)
}`,
			Uniforms: map[string]any{
				"C": []float32{1, 0, 0, 0, 0, 0, 0, 0},
			},
		},
	}

	for _, shader := range shaders {
		shader := shader
		t.Run(shader.Name, func(t *testing.T) {
			const w, h = 1, 1

			dst := ebiten.NewImage(w, h)
			defer dst.Deallocate()

			s, err := ebiten.NewShader([]byte(shader.Shader))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Deallocate()

			op := &ebiten.DrawRectShaderOptions{}
			op.Uniforms = shader.Uniforms
			dst.DrawRectShader(w, h, s, op)
			if got, want := dst.At(0, 0), (color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}); got != want {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}

// Issue #2006
func TestShaderFuncMod(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	r := mod(-0.25, 1.0)
	return vec4(r, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w/2, h/2, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if i < w/2 && j < h/2 {
				want = color.RGBA{R: 0xc0, A: 0xff}
			}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderMatrixInitialize(t *testing.T) {
	const w, h = 16, 16

	src := ebiten.NewImage(w, h)
	src.Fill(color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff})

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return mat4(2) * imageSrc0At(srcPos);
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x20, G: 0x40, B: 0x60, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2029
func TestShaderModVectorAndFloat(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	r := mod(vec3(0.25, 0.5, 0.75), 0.5)
	return vec4(r, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x40, B: 0x40, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderTextureAt(t *testing.T) {
	const w, h = 16, 16

	src := ebiten.NewImage(w, h)
	src.Fill(color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff})

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func textureAt(uv vec2) vec4 {
	return imageSrc0UnsafeAt(uv)
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return textureAt(srcPos)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderAtan2(t *testing.T) {
	const w, h = 16, 16

	src := ebiten.NewImage(w, h)
	src.Fill(color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff})

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	y := vec4(1, 1, 1, 1)
	x := vec4(1, 1, 1, 1)
	return atan2(y, x)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			v := byte(math.Floor(0xff * math.Pi / 4))
			want := color.RGBA{R: v, G: v, B: v, A: v}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderUniformMatrix2(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var Mat2 mat2
var F float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(F * Mat2 * vec2(1), 1, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Mat2": []float32{
			1.0 / 256.0, 2.0 / 256.0,
			3.0 / 256.0, 4.0 / 256.0,
		},
		"F": float32(2),
	}
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 8, G: 12, B: 0xff, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderUniformMatrix2Array(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var Mat2 [2]mat2
var F float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(F * Mat2[0] * Mat2[1] * vec2(1), 1, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Mat2": []float32{
			1.0 / 256.0, 2.0 / 256.0,
			3.0 / 256.0, 4.0 / 256.0,
			5.0 / 256.0, 6.0 / 256.0,
			7.0 / 256.0, 8.0 / 256.0,
		},
		"F": float32(256),
	}
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 54, G: 80, B: 0xff, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderUniformMatrix3(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var Mat3 mat3
var F float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(F * Mat3 * vec3(1), 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Mat3": []float32{
			1.0 / 256.0, 2.0 / 256.0, 3.0 / 256.0,
			4.0 / 256.0, 5.0 / 256.0, 6.0 / 256.0,
			7.0 / 256.0, 8.0 / 256.0, 9.0 / 256.0,
		},
		"F": float32(2),
	}
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 24, G: 30, B: 36, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderUniformMatrix3Array(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var Mat3 [2]mat3
var F float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(F * Mat3[0] * Mat3[1] * vec3(1), 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Mat3": []float32{
			1.0 / 256.0, 2.0 / 256.0, 3.0 / 256.0,
			4.0 / 256.0, 5.0 / 256.0, 6.0 / 256.0,
			7.0 / 256.0, 8.0 / 256.0, 9.0 / 256.0,
			10.0 / 256.0, 11.0 / 256.0, 12.0 / 256.0,
			13.0 / 256.0, 14.0 / 256.0, 15.0 / 256.0,
			16.0 / 256.0, 17.0 / 256.0, 18.0 / 256.0,
		},
		"F": float32(3),
	}
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 6, G: 8, B: 9, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderUniformMatrix4(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var Mat4 mat4
var F float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return F * Mat4 * vec4(1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Mat4": []float32{
			1.0 / 256.0, 2.0 / 256.0, 3.0 / 256.0, 4.0 / 256.0,
			5.0 / 256.0, 6.0 / 256.0, 7.0 / 256.0, 8.0 / 256.0,
			9.0 / 256.0, 10.0 / 256.0, 11.0 / 256.0, 12.0 / 256.0,
			13.0 / 256.0, 14.0 / 256.0, 15.0 / 256.0, 16.0 / 256.0,
		},
		"F": float32(4),
	}
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 112, G: 128, B: 143, A: 159}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderUniformMatrix4Array(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var Mat4 [2]mat4
var F float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return F * Mat4[0] * Mat4[1] * vec4(1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Mat4": []float32{
			1.0 / 256.0, 2.0 / 256.0, 3.0 / 256.0, 4.0 / 256.0,
			5.0 / 256.0, 6.0 / 256.0, 7.0 / 256.0, 8.0 / 256.0,
			9.0 / 256.0, 10.0 / 256.0, 11.0 / 256.0, 12.0 / 256.0,
			13.0 / 256.0, 14.0 / 256.0, 15.0 / 256.0, 16.0 / 256.0,
			17.0 / 256.0, 18.0 / 256.0, 19.0 / 256.0, 20.0 / 256.0,
			21.0 / 256.0, 22.0 / 256.0, 23.0 / 256.0, 24.0 / 256.0,
			25.0 / 256.0, 26.0 / 256.0, 27.0 / 256.0, 28.0 / 256.0,
			29.0 / 256.0, 30.0 / 256.0, 31.0 / 256.0, 32.0 / 256.0,
		},
		"F": float32(4),
	}
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 44, G: 50, B: 56, A: 62}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderUniformMatrixIndexer(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var Mat4 mat4

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return Mat4[1][2] * vec4(1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"Mat4": []float32{
			1.0 / 256.0, 2.0 / 256.0, 3.0 / 256.0, 4.0 / 256.0,
			5.0 / 256.0, 6.0 / 256.0, 7.0 / 256.0, 8.0 / 256.0,
			9.0 / 256.0, 10.0 / 256.0, 11.0 / 256.0, 12.0 / 256.0,
			13.0 / 256.0, 14.0 / 256.0, 15.0 / 256.0, 16.0 / 256.0,
		},
	}
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 7, G: 7, B: 7, A: 7}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderOptionsNegativeBounds(t *testing.T) {
	const w, h = 16, 16

	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	r := imageSrc0At(srcPos).r
	g := imageSrc1At(srcPos).g
	return vec4(r, g, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	const offset0 = -4
	src0 := ebiten.NewImageWithOptions(image.Rect(offset0, offset0, w+offset0, h+offset0), nil)
	pix0 := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if 2 <= i && i < 10 && 3 <= j && j < 11 {
				pix0[4*(j*w+i)] = 0xff
				pix0[4*(j*w+i)+1] = 0
				pix0[4*(j*w+i)+2] = 0
				pix0[4*(j*w+i)+3] = 0xff
			}
		}
	}
	src0.WritePixels(pix0)
	src0 = src0.SubImage(image.Rect(2+offset0, 3+offset0, 10+offset0, 11+offset0)).(*ebiten.Image)

	const offset1 = -6
	src1 := ebiten.NewImageWithOptions(image.Rect(offset1, offset1, w+offset1, h+offset1), nil)
	pix1 := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if 6 <= i && i < 14 && 8 <= j && j < 16 {
				pix1[4*(j*w+i)] = 0
				pix1[4*(j*w+i)+1] = 0xff
				pix1[4*(j*w+i)+2] = 0
				pix1[4*(j*w+i)+3] = 0xff
			}
		}
	}
	src1.WritePixels(pix1)
	src1 = src1.SubImage(image.Rect(6+offset1, 8+offset1, 14+offset1, 16+offset1)).(*ebiten.Image)

	const offset2 = -2
	testPixels := func(testname string, dst *ebiten.Image) {
		for j := offset2; j < h+offset2; j++ {
			for i := offset2; i < w+offset2; i++ {
				got := dst.At(i, j).(color.RGBA)
				var want color.RGBA
				if 0 <= i && i < w/2 && 0 <= j && j < h/2 {
					want = color.RGBA{R: 0xff, G: 0xff, A: 0xff}
				}
				if got != want {
					t.Errorf("%s dst.At(%d, %d): got: %v, want: %v", testname, i, j, got, want)
				}
			}
		}
	}

	t.Run("DrawRectShader", func(t *testing.T) {
		dst := ebiten.NewImageWithOptions(image.Rect(offset2, offset2, w+offset2, h+offset2), nil)
		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = src0
		op.Images[1] = src1
		dst.DrawRectShader(w/2, h/2, s, op)
		testPixels("DrawRectShader", dst)
	})

	t.Run("DrawTrianglesShader", func(t *testing.T) {
		dst := ebiten.NewImageWithOptions(image.Rect(offset2, offset2, w+offset2, h+offset2), nil)
		vs := []ebiten.Vertex{
			{
				DstX:   0,
				DstY:   0,
				SrcX:   2 + offset0,
				SrcY:   3 + offset0,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   w / 2,
				DstY:   0,
				SrcX:   10 + offset0,
				SrcY:   3 + offset0,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   0,
				DstY:   h / 2,
				SrcX:   2 + offset0,
				SrcY:   11 + offset0,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   w / 2,
				DstY:   h / 2,
				SrcX:   10 + offset0,
				SrcY:   11 + offset0,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
		}
		is := []uint16{0, 1, 2, 1, 2, 3}

		op := &ebiten.DrawTrianglesShaderOptions{}
		op.Images[0] = src0
		op.Images[1] = src1
		dst.DrawTrianglesShader(vs, is, s, op)
		testPixels("DrawTrianglesShader", dst)
	})
}

// Issue #2186
func TestShaderVectorEqual(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := vec3(1)
	b := vec3(1)
	if a == b {
		return vec4(1, 0, 0, 1)
	} else {
		return vec4(0, 1, 0, 1)
	}
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #1969
func TestShaderDiscard(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	dst.Fill(color.RGBA{R: 0xff, A: 0xff})

	src := ebiten.NewImage(w, h)
	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			if i >= w/2 || j >= h/2 {
				continue
			}
			pix[4*(j*w+i)+3] = 0xff
		}
	}
	src.WritePixels(pix)

	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	p := imageSrc0At(srcPos)
	if p.a == 0 {
		discard()
	} else {
		return vec4(0, 1, 0, 1)
	}
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{G: 0xff, A: 0xff}
			if i >= w/2 || j >= h/2 {
				want = color.RGBA{R: 0xff, A: 0xff}
			}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2245, #2247
func TestShaderDrawRect(t *testing.T) {
	const (
		dstW = 16
		dstH = 16
		srcW = 8
		srcH = 8
	)

	dst := ebiten.NewImage(dstW, dstH)
	src := ebiten.NewImage(srcW, srcH)

	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	// Adjust srcPos into [0, 1].
	srcPos -= imageSrc0Origin()
	srcPos /= imageSrc0Size()
	if srcPos.x >= 0.5 && srcPos.y >= 0.5 {
		return vec4(1, 0, 0, 1)
	}
	return vec4(0, 1, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	const (
		offsetX = (dstW - srcW) / 2
		offsetY = (dstH - srcH) / 2
	)
	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(offsetX, offsetY)
	op.Images[0] = src
	dst.DrawRectShader(srcW, srcH, s, op)
	for j := 0; j < dstH; j++ {
		for i := 0; i < dstW; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if offsetX <= i && i < offsetX+srcW && offsetY <= j && j < offsetY+srcH {
				if offsetX+srcW/2 <= i && offsetY+srcH/2 <= j {
					want = color.RGBA{R: 0xff, A: 0xff}
				} else {
					want = color.RGBA{G: 0xff, A: 0xff}
				}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderDrawRectColorScale(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return color
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.ColorScale.SetR(4.0 / 8.0)
	op.ColorScale.SetG(5.0 / 8.0)
	op.ColorScale.SetB(6.0 / 8.0)
	op.ColorScale.SetA(7.0 / 8.0)
	op.ColorScale.ScaleWithColor(color.RGBA{R: 0x40, G: 0x80, B: 0xc0, A: 0xff})
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x20, G: 0x50, B: 0x90, A: 0xe0}
			if !sameColors(got, want, 1) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderUniformInt(t *testing.T) {
	const ints = `//kage:unit pixels

package main

var U0 int
var U1 int
var U2 int
var U3 int

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(float(U0)/255.0, float(U1)/255.0, float(U2)/255.0, float(U3)/255.0)
}
`

	const intArray = `//kage:unit pixels

package main

var U [4]int

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(float(U[0])/255.0, float(U[1])/255.0, float(U[2])/255.0, float(U[3])/255.0)
}
`

	const intVec = `//kage:unit pixels

package main

var U0 ivec4
var U1 [2]ivec3

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(float(U0.x)/255.0, float(U0.y)/255.0, float(U1[0].z)/255.0, float(U1[1].x)/255.0)
}
`

	testCases := []struct {
		Name     string
		Uniforms map[string]any
		Shader   string
		Want     color.RGBA
	}{
		{
			Name: "0xff",
			Uniforms: map[string]any{
				"U0": 0xff,
				"U1": 0xff,
				"U2": 0xff,
				"U3": 0xff,
			},
			Shader: ints,
			Want:   color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		},
		{
			Name: "int",
			Uniforms: map[string]any{
				"U0": int8(0x24),
				"U1": int16(0x3f),
				"U2": int32(0x6a),
				"U3": int64(0x88),
			},
			Shader: ints,
			Want:   color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x88},
		},
		{
			Name: "uint",
			Uniforms: map[string]any{
				"U0": uint8(0x85),
				"U1": uint16(0xa3),
				"U2": uint32(0x08),
				"U3": uint64(0xd3),
			},
			Shader: ints,
			Want:   color.RGBA{R: 0x85, G: 0xa3, B: 0x08, A: 0xd3},
		},
		{
			Name: "0xff,slice",
			Uniforms: map[string]any{
				"U": []int{0xff, 0xff, 0xff, 0xff},
			},
			Shader: intArray,
			Want:   color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		},
		{
			Name: "int,slice",
			Uniforms: map[string]any{
				"U": []int16{0x24, 0x3f, 0x6a, 0x88},
			},
			Shader: intArray,
			Want:   color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x88},
		},
		{
			Name: "uint,slice",
			Uniforms: map[string]any{
				"U": []uint8{0x85, 0xa3, 0x08, 0xd3},
			},
			Shader: intArray,
			Want:   color.RGBA{R: 0x85, G: 0xa3, B: 0x08, A: 0xd3},
		},
		{
			Name: "0xff,array",
			Uniforms: map[string]any{
				"U": [...]int{0xff, 0xff, 0xff, 0xff},
			},
			Shader: intArray,
			Want:   color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		},
		{
			Name: "int,array",
			Uniforms: map[string]any{
				"U": [...]int16{0x24, 0x3f, 0x6a, 0x88},
			},
			Shader: intArray,
			Want:   color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x88},
		},
		{
			Name: "uint,array",
			Uniforms: map[string]any{
				"U": [...]uint8{0x85, 0xa3, 0x08, 0xd3},
			},
			Shader: intArray,
			Want:   color.RGBA{R: 0x85, G: 0xa3, B: 0x08, A: 0xd3},
		},
		{
			Name: "0xff,array",
			Uniforms: map[string]any{
				"U": [...]int{0xff, 0xff, 0xff, 0xff},
			},
			Shader: intArray,
			Want:   color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		},
		{
			Name: "int,array",
			Uniforms: map[string]any{
				"U": [...]int16{0x24, 0x3f, 0x6a, 0x88},
			},
			Shader: intArray,
			Want:   color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x88},
		},
		{
			Name: "uint,array",
			Uniforms: map[string]any{
				"U": [...]uint8{0x85, 0xa3, 0x08, 0xd3},
			},
			Shader: intArray,
			Want:   color.RGBA{R: 0x85, G: 0xa3, B: 0x08, A: 0xd3},
		},
		{
			Name: "0xff,ivec",
			Uniforms: map[string]any{
				"U0": [...]int{0xff, 0xff, 0xff, 0xff},
				"U1": [...]int{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
			},
			Shader: intVec,
			Want:   color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		},
		{
			Name: "int,ivec",
			Uniforms: map[string]any{
				"U0": [...]int16{0x24, 0x3f, 0x6a, 0x88},
				"U1": [...]int16{0x85, 0xa3, 0x08, 0xd3, 0x13, 0x19},
			},
			Shader: intVec,
			Want:   color.RGBA{R: 0x24, G: 0x3f, B: 0x08, A: 0xd3},
		},
		{
			Name: "uint,ivec",
			Uniforms: map[string]any{
				"U0": [...]uint8{0x24, 0x3f, 0x6a, 0x88},
				"U1": [...]uint8{0x85, 0xa3, 0x08, 0xd3, 0x13, 0x19},
			},
			Shader: intVec,
			Want:   color.RGBA{R: 0x24, G: 0x3f, B: 0x08, A: 0xd3},
		},
	}
	for _, tc := range testCases {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			const w, h = 1, 1

			dst := ebiten.NewImage(w, h)
			defer dst.Deallocate()

			s, err := ebiten.NewShader([]byte(tc.Shader))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Deallocate()

			op := &ebiten.DrawRectShaderOptions{}
			op.Uniforms = tc.Uniforms
			dst.DrawRectShader(w, h, s, op)
			if got, want := dst.At(0, 0).(color.RGBA), tc.Want; !sameColors(got, want, 1) {
				t.Errorf("got: %v, want: %v", got, want)
			}
		})
	}
}

// Issue #2463
func TestShaderUniformVec3Array(t *testing.T) {
	const shader = `//kage:unit pixels

package main

var U [4]vec3

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(U[0].x/255.0, U[1].y/255.0, U[2].z/255.0, U[3].x/255.0)
}
`
	const w, h = 1, 1

	dst := ebiten.NewImage(w, h)
	defer dst.Deallocate()

	s, err := ebiten.NewShader([]byte(shader))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Deallocate()

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"U": []float32{
			0x24, 0x3f, 0x6a,
			0x88, 0x85, 0xa3,
			0x08, 0xd3, 0x13,
			0x19, 0x8a, 0x2e,
		},
	}
	dst.DrawRectShader(w, h, s, op)
	if got, want := dst.At(0, 0).(color.RGBA), (color.RGBA{R: 0x24, G: 0x85, B: 0x13, A: 0x19}); !sameColors(got, want, 1) {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestShaderIVecMod(t *testing.T) {
	cases := []struct {
		source string
		want   color.RGBA
	}{
		{
			source: `a := ivec4(0x24, 0x3f, 0x6a, 0x88)
return vec4(a)/255`,
			want: color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x88},
		},
		{
			source: `a := ivec4(0x24, 0x3f, 0x6a, 0x88)
a %= 0x85
return vec4(a)/255`,
			want: color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x03},
		},
		{
			source: `a := ivec4(0x24, 0x3f, 0x6a, 0x88)
a %= ivec4(0x85, 0xa3, 0x08, 0xd3)
return vec4(a)/255`,
			want: color.RGBA{R: 0x24, G: 0x3f, B: 0x02, A: 0x88},
		},
		{
			source: `a := ivec4(0x24, 0x3f, 0x6a, 0x88)
b := a % 0x85
return vec4(b)/255`,
			want: color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0x03},
		},
		{
			source: `a := ivec4(0x24, 0x3f, 0x6a, 0x88)
b := a % ivec4(0x85, 0xa3, 0x08, 0xd3)
return vec4(b)/255`,
			want: color.RGBA{R: 0x24, G: 0x3f, B: 0x02, A: 0x88},
		},
	}

	for _, tc := range cases {
		shader := fmt.Sprintf(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	%s
}
`, tc.source)
		const w, h = 1, 1

		dst := ebiten.NewImage(w, h)
		defer dst.Deallocate()

		s, err := ebiten.NewShader([]byte(shader))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Deallocate()

		op := &ebiten.DrawRectShaderOptions{}
		dst.DrawRectShader(w, h, s, op)
		if got, want := dst.At(0, 0).(color.RGBA), tc.want; !sameColors(got, want, 1) {
			t.Errorf("%s: got: %v, want: %v", tc.source, got, want)
		}
	}
}

func TestShaderTexelAndPixel(t *testing.T) {
	const dstW, dstH = 13, 17
	const srcW, srcH = 19, 23
	dstTexel := ebiten.NewImage(dstW, dstH)
	dstPixel := ebiten.NewImage(dstW, dstH)
	src := ebiten.NewImage(srcW, srcH)

	shaderTexel, err := ebiten.NewShader([]byte(fmt.Sprintf(`//kage:unit texels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	pos := (srcPos - imageSrc0Origin()) / imageSrc0Size()
	pos *= vec2(%d, %d)
	pos /= 255
	return vec4(pos.x, pos.y, 0, 1)
}
`, srcW, srcH)))
	if err != nil {
		t.Fatal(err)
	}
	shaderPixel, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	pos := srcPos - imageSrc0Origin()
	pos /= 255
	return vec4(pos.x, pos.y, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dstTexel.DrawRectShader(src.Bounds().Dx(), src.Bounds().Dy(), shaderTexel, op)
	dstPixel.DrawRectShader(src.Bounds().Dx(), src.Bounds().Dy(), shaderPixel, op)

	for j := 0; j < dstH; j++ {
		for i := 0; i < dstW; i++ {
			c0 := dstTexel.At(i, j).(color.RGBA)
			c1 := dstPixel.At(i, j).(color.RGBA)
			if !sameColors(c0, c1, 1) {
				t.Errorf("dstTexel.At(%d, %d) %v != dstPixel.At(%d, %d) %v", i, j, c0, i, j, c1)
			}
		}
	}
}

func TestShaderDifferentTextureSizes(t *testing.T) {
	src0 := ebiten.NewImageWithOptions(image.Rect(0, 0, 20, 4000), &ebiten.NewImageOptions{
		Unmanaged: true,
	}).SubImage(image.Rect(4, 1025, 6, 1028)).(*ebiten.Image)
	defer src0.Deallocate()

	src1 := ebiten.NewImageWithOptions(image.Rect(0, 0, 4000, 20), &ebiten.NewImageOptions{
		Unmanaged: true,
	}).SubImage(image.Rect(2047, 7, 2049, 10)).(*ebiten.Image)
	defer src1.Deallocate()

	src0.Fill(color.RGBA{0x10, 0x20, 0x30, 0xff})
	src1.Fill(color.RGBA{0x30, 0x20, 0x10, 0xff})

	for _, unit := range []string{"texels", "pixels"} {
		unit := unit
		t.Run(fmt.Sprintf("unit %s", unit), func(t *testing.T) {
			shader, err := ebiten.NewShader([]byte(fmt.Sprintf(`//kage:unit %s

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return imageSrc0At(srcPos) + imageSrc1At(srcPos)
}
`, unit)))
			if err != nil {
				t.Fatal(err)
			}
			defer shader.Deallocate()

			dst := ebiten.NewImage(2, 3)
			defer dst.Deallocate()

			op := &ebiten.DrawRectShaderOptions{}
			op.Images[0] = src0
			op.Images[1] = src1
			dst.DrawRectShader(2, 3, shader, op)

			for j := 0; j < 3; j++ {
				for i := 0; i < 2; i++ {
					got := dst.At(i, j).(color.RGBA)
					want := color.RGBA{0x40, 0x40, 0x40, 0xff}
					if !sameColors(got, want, 1) {
						t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
					}
				}
			}
		})
	}
}

func TestShaderIVec(t *testing.T) {
	const w, h = 16, 16
	dst := ebiten.NewImage(w, h)
	src := ebiten.NewImage(w, h)

	pix := make([]byte, 4*w*h)
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			pix[4*(j*w+i)] = byte(i)
			pix[4*(j*w+i)+1] = byte(j)
			pix[4*(j*w+i)+3] = 0xff
		}
	}
	src.WritePixels(pix)

	// Test that ivec2 can take any float values that can be casted to integers.
	// This seems the common behavior in shading languages like GLSL, Metal, and HLSL.
	shader, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	pos := ivec2(3, 4)
	return imageSrc0At(vec2(pos) + imageSrc0Origin())
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dst.DrawRectShader(w, h, shader, op)

	got := dst.At(0, 0).(color.RGBA)
	want := color.RGBA{3, 4, 0, 0xff}
	if got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestShaderUniformSizes(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var U vec4
var V [3]float

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(0)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		uniforms map[string]any
		err      bool
	}{
		{
			uniforms: nil,
			err:      false,
		},
		{
			uniforms: map[string]any{
				"U": 1,
			},
			err: true,
		},
		{
			uniforms: map[string]any{
				"U": "1",
			},
			err: true,
		},
		{
			uniforms: map[string]any{
				"U": []int32{1, 2, 3},
			},
			err: true,
		},
		{
			uniforms: map[string]any{
				"U": []int32{1, 2, 3, 4},
			},
			err: false,
		},
		{
			uniforms: map[string]any{
				"U": []int32{1, 2, 3, 4, 5},
			},
			err: true,
		},
		{
			uniforms: map[string]any{
				"V": 1,
			},
			err: true,
		},
		{
			uniforms: map[string]any{
				"V": "1",
			},
			err: true,
		},
		{
			uniforms: map[string]any{
				"V": []int32{1, 2},
			},
			err: true,
		},
		{
			uniforms: map[string]any{
				"V": []int32{1, 2, 3},
			},
			err: false,
		},
		{
			uniforms: map[string]any{
				"V": []int32{1, 2, 3, 4},
			},
			err: true,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("%v", tc.uniforms), func(t *testing.T) {
			defer func() {
				r := recover()
				if r != nil && !tc.err {
					t.Errorf("DrawRectShader must not panic but did")
				} else if r == nil && tc.err {
					t.Errorf("DrawRectShader must panic but does not")
				}
			}()
			op := &ebiten.DrawRectShaderOptions{}
			op.Uniforms = tc.uniforms
			dst.DrawRectShader(w, h, s, op)
		})
	}
}

// Issue #2709
func TestShaderUniformDefaultValue(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var U vec4

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return U
}
`))
	if err != nil {
		t.Fatal(err)
	}

	// Draw with a uniform variable value.
	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"U": [...]float32{1, 1, 1, 1},
	}
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// Draw without a uniform variable value. In this case, the uniform variable value should be 0.
	dst.Clear()
	op.Uniforms = nil
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{0, 0, 0, 0}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2166
func TestShaderDrawRectWithoutSource(t *testing.T) {
	const (
		dstW = 16
		dstH = 16
		srcW = 8
		srcH = 8
	)

	src := ebiten.NewImage(srcW, srcH)

	for _, unit := range []string{"pixels", "texels"} {
		s, err := ebiten.NewShader([]byte(fmt.Sprintf(`//kage:unit %s

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	t := srcPos

	size := imageSrc0Size()

	// If the unit is texels and no source images are specified, size is always 0.
	if size == vec2(0) {
		// Even in this case, t is in pixels (0, 0) to (8, 8).
		if t.x >= 4 && t.y >= 4 {
			return vec4(1, 0, 1, 1)
		}
		return vec4(0, 1, 1, 1)
	}

	// Adjust srcPos into [0, 1].
	t -= imageSrc0Origin()
	if size != vec2(0) {
		t /= size
	}
	if t.x >= 0.5 && t.y >= 0.5 {
		return vec4(1, 0, 0, 1)
	}
	return vec4(0, 1, 0, 1)
}
`, unit)))
		if err != nil {
			t.Fatal(err)
		}

		for _, withSrc := range []bool{false, true} {
			withSrc := withSrc
			title := "WithSrc,unit=" + unit
			if !withSrc {
				title = "WithoutSrc,unit=" + unit
			}
			t.Run(title, func(t *testing.T) {
				dst := ebiten.NewImage(dstW, dstH)
				const (
					offsetX = (dstW - srcW) / 2
					offsetY = (dstH - srcH) / 2
				)
				op := &ebiten.DrawRectShaderOptions{}
				op.GeoM.Translate(offsetX, offsetY)
				if withSrc {
					op.Images[0] = src
				}
				dst.DrawRectShader(srcW, srcH, s, op)
				for j := 0; j < dstH; j++ {
					for i := 0; i < dstW; i++ {
						got := dst.At(i, j).(color.RGBA)
						var want color.RGBA
						if offsetX <= i && i < offsetX+srcW && offsetY <= j && j < offsetY+srcH {
							var blue byte
							if !withSrc && unit == "texels" {
								blue = 0xff
							}
							if offsetX+srcW/2 <= i && offsetY+srcH/2 <= j {
								want = color.RGBA{0xff, 0, blue, 0xff}
							} else {
								want = color.RGBA{0, 0xff, blue, 0xff}
							}
						}
						if got != want {
							t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
						}
					}
				}
			})
		}
	}
}

// Issue #2719
func TestShaderMatrixDivFloat(t *testing.T) {
	const w, h = 16, 16

	src := ebiten.NewImage(w, h)
	src.Fill(color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0xff})

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var x = 2.0
	return mat4(3) / x * imageSrc0At(srcPos);
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Images[0] = src
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x18, G: 0x30, B: 0x48, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderDifferentSourceSizes(t *testing.T) {
	src0 := ebiten.NewImageWithOptions(image.Rect(0, 0, 20, 4000), &ebiten.NewImageOptions{
		Unmanaged: true,
	}).SubImage(image.Rect(4, 1025, 7, 1029)).(*ebiten.Image) // 3x4
	defer src0.Deallocate()

	src1 := ebiten.NewImageWithOptions(image.Rect(0, 0, 4000, 20), &ebiten.NewImageOptions{
		Unmanaged: true,
	}).SubImage(image.Rect(2047, 7, 2049, 10)).(*ebiten.Image) // 2x3
	defer src1.Deallocate()

	src0.Fill(color.RGBA{0x10, 0x20, 0x30, 0xff})
	src1.Fill(color.RGBA{0x30, 0x20, 0x10, 0xff})

	for _, unit := range []string{"texels", "pixels"} {
		unit := unit
		t.Run(fmt.Sprintf("unit %s", unit), func(t *testing.T) {
			if unit == "texels" {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("DrawTrianglesShader must panic with different sizes but not (unit=%s)", unit)
					}
				}()
			}
			shader, err := ebiten.NewShader([]byte(fmt.Sprintf(`//kage:unit %s

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return imageSrc0At(srcPos) + imageSrc1At(srcPos)
}
`, unit)))
			if err != nil {
				t.Fatal(err)
			}
			defer shader.Deallocate()

			dst := ebiten.NewImage(3, 4)
			defer dst.Deallocate()

			op := &ebiten.DrawTrianglesShaderOptions{}
			op.Images[0] = src0
			op.Images[1] = src1
			vs := []ebiten.Vertex{
				{
					DstX:   0,
					DstY:   0,
					SrcX:   4,
					SrcY:   1025,
					ColorR: 1,
					ColorG: 1,
					ColorB: 1,
					ColorA: 1,
				},
				{
					DstX:   3,
					DstY:   0,
					SrcX:   7,
					SrcY:   1025,
					ColorR: 1,
					ColorG: 1,
					ColorB: 1,
					ColorA: 1,
				},
				{
					DstX:   0,
					DstY:   4,
					SrcX:   4,
					SrcY:   1029,
					ColorR: 1,
					ColorG: 1,
					ColorB: 1,
					ColorA: 1,
				},
				{
					DstX:   3,
					DstY:   4,
					SrcX:   7,
					SrcY:   1029,
					ColorR: 1,
					ColorG: 1,
					ColorB: 1,
					ColorA: 1,
				},
			}
			is := []uint16{0, 1, 2, 1, 2, 3}
			dst.DrawTrianglesShader(vs, is, shader, op)

			if unit == "texel" {
				t.Fatal("not reached")
			}

			for j := 0; j < 4; j++ {
				for i := 0; i < 3; i++ {
					got := dst.At(i, j).(color.RGBA)
					var want color.RGBA
					if i < 2 && j < 3 {
						want = color.RGBA{0x40, 0x40, 0x40, 0xff}
					} else {
						want = color.RGBA{0x10, 0x20, 0x30, 0xff}
					}
					if !sameColors(got, want, 1) {
						t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
					}
				}
			}
		})
	}
}

// Issue #2752
func TestShaderBitwiseOperator(t *testing.T) {
	const w, h = 16, 16

	src := ebiten.NewImage(w, h)
	src.Fill(color.RGBA{R: 0x24, G: 0x3f, B: 0x6a, A: 0xff})

	for _, assign := range []bool{false, true} {
		assign := assign
		name := "op"
		if assign {
			name = "op+assign"
		}
		t.Run(name, func(t *testing.T) {
			var code string
			if assign {
				code = `	v.rgb &= 0x5a
	v.rgb |= 0x30
	v.rgb ^= 0x8d`
			} else {
				code = `	v.rgb = v.rgb & 0x5a
	v.rgb = v.rgb | 0x30
	v.rgb = v.rgb ^ 0x8d`
			}

			s, err := ebiten.NewShader([]byte(fmt.Sprintf(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	v := ivec4(imageSrc0At(srcPos) * 0xff)
%s
	return vec4(v) / 0xff;
}
`, code)))
			if err != nil {
				t.Fatal(err)
			}

			dst := ebiten.NewImage(w, h)
			op := &ebiten.DrawRectShaderOptions{}
			op.Images[0] = src
			dst.DrawRectShader(w, h, s, op)

			for j := 0; j < h; j++ {
				for i := 0; i < w; i++ {
					got := dst.At(i, j).(color.RGBA)
					want := color.RGBA{R: 0xbd, G: 0xb7, B: 0xf7, A: 0xff}
					if !sameColors(got, want, 2) {
						t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
					}
				}
			}
		})
	}
}

func TestShaderDispose(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w/2, h/2, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if i < w/2 && j < h/2 {
				want = color.RGBA{R: 0xff, A: 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	s.Dispose()

	dst.Clear()

	defer func() {
		if e := recover(); e == nil {
			panic("DrawRectShader with a disposed shader must panic but not")
		}
	}()

	dst.DrawRectShader(w/2, h/2, s, nil)
}

func TestShaderDeallocate(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w/2, h/2, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if i < w/2 && j < h/2 {
				want = color.RGBA{R: 0xff, A: 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// Even after Deallocate is called, the shader is still available.
	s.Deallocate()

	dst.Clear()
	dst.DrawRectShader(w/2, h/2, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			var want color.RGBA
			if i < w/2 && j < h/2 {
				want = color.RGBA{R: 0xff, A: 0xff}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2923
func TestShaderReturnArray(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func foo() [4]float {
	return [4]float{0.25, 0.5, 0.75, 1}
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := foo()
	return vec4(a[0], a[1], a[2], a[3])
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x40, G: 0x80, B: 0xc0, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2798
func TestShaderInvalidPremultipliedAlphaColor(t *testing.T) {
	// This test checks the rendering result when the shader returns an invalid premultiplied alpha color.
	// The result values are kept and not clamped.

	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(1, 0.75, 0.5, 0.25)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, G: 0xc0, B: 0x80, A: 0x40}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	dst.Clear()
	s, err = ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	return vec4(1, 0.75, 0.5, 0)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, G: 0xc0, B: 0x80, A: 0}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2933
func TestShaderIncDecStmt(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 0
	a++
	b := -0.5
	b++
	c := ivec2(0)
	c++
	d := vec2(-0.25)
	d++
	return vec4(float(a), b, float(c.x), d.y)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, G: 0x80, B: 0xff, A: 0xc0}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	dst.Clear()

	s, err = ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 1
	a--
	b := 1.5
	b--
	c := ivec2(1)
	c--
	d := vec2(1.25)
	d--
	return vec4(float(a), b, float(c.x), d.y)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x00, G: 0x80, B: 0x00, A: 0x40}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

// Issue #2934
func TestShaderAssignConst(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 0.0
	a = 1
	b, c := 0.0, 0.0
	b, c = 1, 1
	d := 0.0
	d += 1
	return vec4(a, b, c, d)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderCustomValues(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4, custom vec4) vec4 {
	return custom
}
`))
	if err != nil {
		t.Fatal(err)
	}

	clr := color.RGBA{R: 0x10, G: 0x20, B: 0x30, A: 0x40}
	dst.DrawTrianglesShader([]ebiten.Vertex{
		{
			DstX:    0,
			DstY:    0,
			SrcX:    0,
			SrcY:    0,
			ColorR:  1,
			ColorG:  1,
			ColorB:  1,
			ColorA:  1,
			Custom0: float32(clr.R) / 0xff,
			Custom1: float32(clr.G) / 0xff,
			Custom2: float32(clr.B) / 0xff,
			Custom3: float32(clr.A) / 0xff,
		},
		{
			DstX:    w,
			DstY:    0,
			SrcX:    w,
			SrcY:    0,
			ColorR:  1,
			ColorG:  1,
			ColorB:  1,
			ColorA:  1,
			Custom0: float32(clr.R) / 0xff,
			Custom1: float32(clr.G) / 0xff,
			Custom2: float32(clr.B) / 0xff,
			Custom3: float32(clr.A) / 0xff,
		},
		{
			DstX:    0,
			DstY:    h,
			SrcX:    0,
			SrcY:    h,
			ColorR:  1,
			ColorG:  1,
			ColorB:  1,
			ColorA:  1,
			Custom0: float32(clr.R) / 0xff,
			Custom1: float32(clr.G) / 0xff,
			Custom2: float32(clr.B) / 0xff,
			Custom3: float32(clr.A) / 0xff,
		},
		{
			DstX:    w,
			DstY:    h,
			SrcX:    w,
			SrcY:    h,
			ColorR:  1,
			ColorG:  1,
			ColorB:  1,
			ColorA:  1,
			Custom0: float32(clr.R) / 0xff,
			Custom1: float32(clr.G) / 0xff,
			Custom2: float32(clr.B) / 0xff,
			Custom3: float32(clr.A) / 0xff,
		},
	}, []uint16{0, 1, 2, 1, 2, 3}, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := clr
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderFragmentLessArguments(t *testing.T) {
	const w, h = 16, 16

	s0, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment() vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}
	s1, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4) vec4 {
	return vec4(0, 1, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}
	s2, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2) vec4 {
	return vec4(0, 0, 1, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst := ebiten.NewImage(w, h)
	for idx, s := range []*ebiten.Shader{s0, s1, s2} {
		dst.Clear()
		dst.DrawTrianglesShader([]ebiten.Vertex{
			{
				DstX:   0,
				DstY:   0,
				SrcX:   0,
				SrcY:   0,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   w,
				DstY:   0,
				SrcX:   w,
				SrcY:   0,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   0,
				DstY:   h,
				SrcX:   0,
				SrcY:   h,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
			{
				DstX:   w,
				DstY:   h,
				SrcX:   w,
				SrcY:   h,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			},
		}, []uint16{0, 1, 2, 1, 2, 3}, s, nil)

		for j := 0; j < h; j++ {
			for i := 0; i < w; i++ {
				got := dst.At(i, j).(color.RGBA)
				var want color.RGBA
				switch idx {
				case 0:
					want = color.RGBA{R: 0xff, A: 0xff}
				case 1:
					want = color.RGBA{G: 0xff, A: 0xff}
				case 2:
					want = color.RGBA{B: 0xff, A: 0xff}
				}
				if !sameColors(got, want, 2) {
					t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
				}
			}
		}
	}
}

func TestShaderArray(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := [4]float{1}
	b := [4]float{1, 1}
	c := [4]float{1, 1, 1}
	d := [4]float{1, 1, 1, 1}
	return vec4(a[3], b[3], c[3], d[3])
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func BenchmarkBuiltinShader(b *testing.B) {
	// Create a shader to cache the shader compilation result.
	_ = ebiten.BuiltinShader(builtinshader.FilterNearest, builtinshader.AddressUnsafe, false)
	for i := 0; i < b.N; i++ {
		_ = ebiten.BuiltinShader(builtinshader.FilterNearest, builtinshader.AddressUnsafe, false)
	}
}

// Issue #3251
func TestShaderSwap(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := 0.25
	b := 0.5
	a, b = b, a
	return vec4(a, b, 0.75, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x80, G: 0x40, B: 0xc0, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderVectorAndScalarMinMax(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels
package main
func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := min(vec2(0.375, 0.5), 0.25)
	b := max(vec2(0.625, 0.5), 0.75)
	return vec4(a, b)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x40, G: 0x40, B: 0xc0, A: 0xc0}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderVariadicMinMax(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels
package main
func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	a := min(0.25, 0.375, 0.5, 0.625, 0.75)
	b := max(0.75, 0.625, 0.5, 0.375, 0.25)
	return vec4(float(a), float(b), 0.75, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	dst.DrawRectShader(w, h, s, nil)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x40, G: 0xc0, B: 0xc0, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderUniformBool(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

var B1 bool
var B2 [2]bool
var B3 bool

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	var r, g, b, a float
	if B1 {
		r = 1.0
	}
	if B2[0] {
		g = 1.0
	}
	if B2[1] {
		b = 1.0
	}
	if B3 {
		a = 1.0
	}
	return vec4(r, g, b, a)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	op := &ebiten.DrawRectShaderOptions{}
	op.Uniforms = map[string]any{
		"B1": true,
		"B2": [2]bool{false, true},
		"B3": true,
	}
	dst.DrawRectShader(w, h, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0xff, G: 0, B: 0xff, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderFrontFacing(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`//kage:unit pixels

package main

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	if frontfacing() {
		return vec4(0.5, 0, 0, 1)
	}
	return vec4(0, 0.5, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	vs := []ebiten.Vertex{
		{
			DstX: 0,
			DstY: 0,
		},
		{
			DstX: w,
			DstY: 0,
		},
		{
			DstX: 0,
			DstY: h,
		},
		{
			DstX: w,
			DstY: h,
		},
	}
	op := &ebiten.DrawTrianglesShaderOptions{}
	op.Blend = ebiten.BlendLighter
	dst.DrawTrianglesShader32(vs, []uint32{0, 1, 2, 1, 2, 3}, s, op)
	dst.DrawTrianglesShader32(vs, []uint32{2, 1, 0, 3, 2, 1}, s, op)

	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := dst.At(i, j).(color.RGBA)
			want := color.RGBA{R: 0x80, G: 0x80, B: 0x00, A: 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}
