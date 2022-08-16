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
				want = color.RGBA{0xff, 0, 0, 0xff}
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
				want = color.RGBA{0xff, 0, 0, 0xff}
			}
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
			want := color.RGBA{0xff, 0, 0, 0xff}
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
			want := color.RGBA{0xff, 0, 0, 0xff}
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
			want := color.RGBA{87, 82, 71, 255}
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
					want = color.RGBA{0xff, 0xff, 0, 0xff}
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
			want := color.RGBA{0, 0, 0, 0xff}
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
			want := color.RGBA{0, 0, 0, 0xff}
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
		Uniforms map[string]interface{}
	}{
		{
			Name: "float array",
			Shader: `package main

var C [2]float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(C[0], 1, 1, 1)
}`,
			Uniforms: map[string]interface{}{
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
			Uniforms: map[string]interface{}{
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
			Uniforms: map[string]interface{}{
				"C": []float32{1, 0, 0, 0, 0, 0, 0, 0},
			},
		},
	}

	for _, shader := range shaders {
		shader := shader
		t.Run(shader.Name, func(t *testing.T) {
			const w, h = 1, 1

			dst := ebiten.NewImage(w, h)
			s, err := ebiten.NewShader([]byte(shader.Shader))
			if err != nil {
				t.Fatal(err)
			}

			op := &ebiten.DrawRectShaderOptions{}
			op.Uniforms = shader.Uniforms
			dst.DrawRectShader(w, h, s, op)
			if got, want := dst.At(0, 0), (color.RGBA{0xff, 0xff, 0xff, 0xff}); got != want {
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
				want = color.RGBA{0xc0, 0, 0, 0xff}
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
	src.Fill(color.RGBA{0x10, 0x20, 0x30, 0xff})

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
			want := color.RGBA{0x20, 0x40, 0x60, 0xff}
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
			want := color.RGBA{0x40, 0, 0x40, 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderTextureAt(t *testing.T) {
	const w, h = 16, 16

	src := ebiten.NewImage(w, h)
	src.Fill(color.RGBA{0x10, 0x20, 0x30, 0xff})

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
			want := color.RGBA{0x10, 0x20, 0x30, 0xff}
			if !sameColors(got, want, 2) {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestShaderAtan2(t *testing.T) {
	const w, h = 16, 16

	src := ebiten.NewImage(w, h)
	src.Fill(color.RGBA{0x10, 0x20, 0x30, 0xff})

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
			want := color.RGBA{v, v, v, v}
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
	op.Uniforms = map[string]interface{}{
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
			want := color.RGBA{8, 12, 0xff, 0xff}
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
	op.Uniforms = map[string]interface{}{
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
			want := color.RGBA{54, 80, 0xff, 0xff}
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
	op.Uniforms = map[string]interface{}{
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
			want := color.RGBA{24, 30, 36, 0xff}
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
	op.Uniforms = map[string]interface{}{
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
			want := color.RGBA{6, 8, 9, 0xff}
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
	op.Uniforms = map[string]interface{}{
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
			want := color.RGBA{112, 128, 143, 159}
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
	op.Uniforms = map[string]interface{}{
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
			want := color.RGBA{44, 50, 56, 62}
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
					want = color.RGBA{0xff, 0xff, 0, 0xff}
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
			want := color.RGBA{0xff, 0, 0x00, 0xff}
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
	dst.Fill(color.RGBA{0xff, 0, 0, 0xff})

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
			want := color.RGBA{0, 0xff, 0x00, 0xff}
			if i >= w/2 || j >= h/2 {
				want = color.RGBA{0xff, 0, 0x00, 0xff}
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
					want = color.RGBA{0xff, 0, 0, 0xff}
				} else {
					want = color.RGBA{0, 0xff, 0, 0xff}
				}
			}
			if got != want {
				t.Errorf("dst.At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}
