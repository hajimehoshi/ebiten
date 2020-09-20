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
	"testing"

	. "github.com/hajimehoshi/ebiten"
)

func TestShaderFill(t *testing.T) {
	const w, h = 16, 16

	dst, _ := NewImage(w, h, FilterDefault)
	s, err := NewShader([]byte(`package main

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

	dst, _ := NewImage(w, h, FilterDefault)
	s, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	src, _ := NewImage(w/2, h/2, FilterDefault)
	op := &DrawRectShaderOptions{}
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

	dst, _ := NewImage(w, h, FilterDefault)
	s, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(1, 0, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	src, _ := NewImage(w/2, h/2, FilterDefault)
	op := &DrawTrianglesShaderOptions{}
	op.Images[0] = src

	vs := []Vertex{
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

	dst, _ := NewImage(w, h, FilterDefault)
	s, err := NewShader([]byte(`package main

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

func TestShaderShadowing(t *testing.T) {
	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var position vec4
	return position
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestShaderDuplicatedVariables(t *testing.T) {
	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var foo vec4
	var foo vec4
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var foo, foo vec4
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var foo vec4
	foo := vec4(0)
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Foo() (vec4, vec4) {
	return vec4(0), vec4(0)
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	foo, foo := Foo()
	return foo
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestShaderNoMain(t *testing.T) {
	if _, err := NewShader([]byte(`package main
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestShaderNoNewVariables(t *testing.T) {
	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_ := 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_, _ := 1, 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Foo() (int, int) {
	return 1, 1
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_, _ := Foo()
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	a, _ := 1, 1
	_ = a
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_, a := 1, 1
	_ = a
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}
}

func TestShaderWrongReturn(t *testing.T) {
	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return 0.0
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Foo() (float, float) {
	return 0
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Foo() float {
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestShaderMultipleValueReturn(t *testing.T) {
	if _, err := NewShader([]byte(`package main

func Foo() (float, float) {
	return 0.0
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Foo() float {
	return 0.0, 0.0
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Foo() (float, float, float) {
	return 0.0, 0.0
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Foo() (float, float) {
	return 0.0, 0.0
}

func Foo2() (float, float, float) {
	return Foo()
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Foo() (float, float, float) {
	return 0.0, 0.0, 0.0
}

func Foo2() (float, float, float) {
	return Foo()
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0.0)
}
`)); err != nil {
		t.Error(err)
	}
}

func TestShaderInit(t *testing.T) {
	if _, err := NewShader([]byte(`package main

func init() {
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestShaderUnspportedSyntax(t *testing.T) {
	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := func() {
	}
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	go func() {
	}()
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	ch := make(chan int)
	_ = ch
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 1i
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var x [4]float
	y := x[1:2]
	_ = y
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var x [4]float
	y := x[1:2:3]
	_ = y
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestShaderUninitializedUniformVariables(t *testing.T) {
	const w, h = 16, 16

	dst, _ := NewImage(w, h, FilterDefault)
	s, err := NewShader([]byte(`package main

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

func TestShaderForbidAssigningSpecialVariables(t *testing.T) {
	if _, err := NewShader([]byte(`package main

var U vec4

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	U = vec4(0)
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

var U vec4

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	U.x = 0
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

var U [2]vec4

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	U[0] = vec4(0)
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	texCoord = vec2(0)
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	texCoord.x = 0
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestShaderBoolLiteral(t *testing.T) {
	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	true := vec4(0)
	return true
}
`)); err != nil {
		t.Errorf("error must be nil but was non-nil")
	}
}

func TestShaderUnusedVariable(t *testing.T) {
	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 0
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 0
	x = 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := vec4(0)
	x.x = 1
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}

	// Increment statement treats a variable 'used'.
	// https://play.golang.org/p/2RuYMrSLjt3
	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 0
	x++
	return vec4(0)
}
`)); err != nil {
		t.Error(err)
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var a int
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var a, b int
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestShaderBlankLhs(t *testing.T) {
	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var x int = _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 1
	x = _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	x := 1 + _
	_ = x
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_++
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_ += 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}

	if _, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	_.x = 1
	return vec4(0)
}
`)); err == nil {
		t.Errorf("error must be non-nil but was nil")
	}
}

func TestShaderMatrix(t *testing.T) {
	const w, h = 16, 16

	dst, _ := NewImage(w, h, FilterDefault)
	s, err := NewShader([]byte(`package main

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

	src, _ := NewImage(w, h, FilterDefault)
	op := &DrawRectShaderOptions{}
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

	s, err := NewShader([]byte(`package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	r := imageSrc0At(texCoord).r
	g := imageSrc1At(texCoord).g
	return vec4(r, g, 0, 1)
}
`))
	if err != nil {
		t.Fatal(err)
	}

	src0, _ := NewImage(w, h, FilterDefault)
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
	src0.ReplacePixels(pix0)
	src0 = src0.SubImage(image.Rect(2, 3, 10, 11)).(*Image)

	src1, _ := NewImage(w, h, FilterDefault)
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
	src1.ReplacePixels(pix1)
	src1 = src1.SubImage(image.Rect(6, 8, 14, 16)).(*Image)

	testPixels := func(testname string, dst *Image) {
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
		dst, _ := NewImage(w, h, FilterDefault)
		op := &DrawRectShaderOptions{}
		op.Images[0] = src0
		op.Images[1] = src1
		dst.DrawRectShader(w/2, h/2, s, op)
		testPixels("DrawRectShader", dst)
	})

	t.Run("DrawTrianglesShader", func(t *testing.T) {
		dst, _ := NewImage(w, h, FilterDefault)
		vs := []Vertex{
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

		op := &DrawTrianglesShaderOptions{}
		op.Images[0] = src0
		op.Images[1] = src1
		dst.DrawTrianglesShader(vs, is, s, op)
		testPixels("DrawTrianglesShader", dst)
	})
}
