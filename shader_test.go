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
)

func TestShaderFill(t *testing.T) {
	const w, h = 16, 16

	dst := ebiten.NewImage(w, h)
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return imageSrc0At(texCoord)
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
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

func clr(red float) (float, float, float, float) {
	return red, 0, 0, 1
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

var U vec4

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	r := imageSrc0At(texCoord).r
	g := imageSrc1At(texCoord).g
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

	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	p := imageSrc0At(texCoord)
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

	s, err := ebiten.NewShader([]byte(`package main

// This function uses dfdx and then should not be in GLSL's vertex shader (#1701).
func Foo(p vec4) vec4 {
	return vec4(abs(dfdx(p.r)), abs(dfdy(p.g)), 0, 1)
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	p := imageSrc0At(texCoord)
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
			Shader: `package main

var C [2]float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(C[0], 1, 1, 1)
}`,
			Uniforms: map[string]any{
				"C": []float32{1, 1},
			},
		},
		{
			Name: "float one-element array",
			Shader: `package main

var C [1]float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(C[0], 1, 1, 1)
}`,
			Uniforms: map[string]any{
				"C": []float32{1},
			},
		},
		{
			Name: "matrix array",
			Shader: `package main

var C [2]mat2

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
			defer dst.Dispose()

			s, err := ebiten.NewShader([]byte(shader.Shader))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Dispose()

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
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return mat4(2) * imageSrc0At(texCoord);
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
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

func textureAt(uv vec2) vec4 {
	return imageSrc0UnsafeAt(uv)
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return textureAt(texCoord)
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
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

var Mat2 mat2
var F float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

var Mat2 [2]mat2
var F float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

var Mat3 mat3
var F float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

var Mat3 [2]mat3
var F float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

var Mat4 mat4
var F float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	s, err := ebiten.NewShader([]byte(`package main

var Mat4 [2]mat4
var F float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

func TestShaderOptionsNegativeBounds(t *testing.T) {
	const w, h = 16, 16

	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	r := imageSrc0At(texCoord).r
	g := imageSrc1At(texCoord).g
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
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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

	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	p := imageSrc0At(texCoord)
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

	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	// Adjust texCoord into [0, 1].
	origin, size := imageSrcRegionOnTexture()
	texCoord -= origin
	texCoord /= size
	if texCoord.x >= 0.5 && texCoord.y >= 0.5 {
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
	s, err := ebiten.NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
	const ints = `package main

var U0 int
var U1 int
var U2 int
var U3 int

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(float(U0)/255.0, float(U1)/255.0, float(U2)/255.0, float(U3)/255.0)
}
`

	const intArray = `package main

var U [4]int

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(float(U[0])/255.0, float(U[1])/255.0, float(U[2])/255.0, float(U[3])/255.0)
}
`

	const intVec = `package main

var U0 ivec4
var U1 [2]ivec3

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
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
			defer dst.Dispose()

			s, err := ebiten.NewShader([]byte(tc.Shader))
			if err != nil {
				t.Fatal(err)
			}
			defer s.Dispose()

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
	const shader = `package main

var U [4]vec3

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(U[0].x/255.0, U[1].y/255.0, U[2].z/255.0, U[3].x/255.0)
}
`
	const w, h = 1, 1

	dst := ebiten.NewImage(w, h)
	defer dst.Dispose()

	s, err := ebiten.NewShader([]byte(shader))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Dispose()

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
		shader := fmt.Sprintf(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	%s
}
`, tc.source)
		const w, h = 1, 1

		dst := ebiten.NewImage(w, h)
		defer dst.Dispose()

		s, err := ebiten.NewShader([]byte(shader))
		if err != nil {
			t.Fatal(err)
		}
		defer s.Dispose()

		op := &ebiten.DrawRectShaderOptions{}
		dst.DrawRectShader(w, h, s, op)
		if got, want := dst.At(0, 0).(color.RGBA), tc.want; !sameColors(got, want, 1) {
			t.Errorf("%s: got: %v, want: %v", tc.source, got, want)
		}
	}
}
