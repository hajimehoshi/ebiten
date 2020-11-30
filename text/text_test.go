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

package text_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/bitmapfont/v2"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
	. "github.com/hajimehoshi/ebiten/v2/text"
)

func TestMain(m *testing.M) {
	t.MainWithRunLoop(m)
}

func TestTextColor(t *testing.T) {
	clr := color.RGBA{0x80, 0x80, 0x80, 0x80}
	img := ebiten.NewImage(30, 30)
	Draw(img, "Hello", bitmapfont.Face, 12, 12, clr)

	w, h := img.Size()
	allTransparent := true
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img.At(i, j)
			want1 := color.RGBA{0x80, 0x80, 0x80, 0x80}
			want2 := color.RGBA{}
			if got != want1 && got != want2 {
				t.Errorf("img At(%d, %d): got %v; want %v or %v", i, j, got, want1, want2)
			}
			if got == want1 {
				allTransparent = false
			}
		}
	}
	if allTransparent {
		t.Fail()
	}
}

const testFaceSize = 6

type testFace struct{}

func (f *testFace) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	dr = image.Rect(0, 0, testFaceSize, testFaceSize)
	a := image.NewAlpha(dr)
	switch r {
	case 'a':
		for j := 0; j < testFaceSize; j++ {
			for i := 0; i < testFaceSize; i++ {
				a.SetAlpha(i, j, color.Alpha{0x80})
			}
		}
	case 'b':
		for j := 0; j < testFaceSize; j++ {
			for i := 0; i < testFaceSize; i++ {
				a.SetAlpha(i, j, color.Alpha{0xff})
			}
		}
	}
	mask = a
	advance = fixed.I(testFaceSize)
	ok = true
	return
}

func (f *testFace) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	bounds = fixed.R(0, 0, testFaceSize, testFaceSize)
	advance = fixed.I(testFaceSize)
	ok = true
	return
}

func (f *testFace) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	return fixed.I(testFaceSize), true
}

func (f *testFace) Kern(r0, r1 rune) fixed.Int26_6 {
	if r1 == 'b' {
		return fixed.I(-testFaceSize)
	}
	return 0
}

func (f *testFace) Close() error {
	return nil
}

func (f *testFace) Metrics() font.Metrics {
	return font.Metrics{
		Height:     fixed.I(testFaceSize),
		Ascent:     fixed.I(testFaceSize),
		Descent:    0,
		XHeight:    0,
		CapHeight:  fixed.I(testFaceSize),
		CaretSlope: image.Pt(0, 1),
	}
}

func TestTextOverlap(t *testing.T) {
	f := &testFace{}
	dst := ebiten.NewImage(testFaceSize*2, testFaceSize)

	// With testFace, 'b' is rendered at the previous position as 0xff.
	// 'a' is rendered at the current position as 0x80.
	Draw(dst, "ab", f, 0, 0, color.White)
	for j := 0; j < testFaceSize; j++ {
		for i := 0; i < testFaceSize; i++ {
			got := dst.At(i, j)
			want := color.RGBA{0xff, 0xff, 0xff, 0xff}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// The glyph 'a' should be treated correctly.
	Draw(dst, "a", f, testFaceSize, 0, color.White)
	for j := 0; j < testFaceSize; j++ {
		for i := testFaceSize; i < testFaceSize*2; i++ {
			got := dst.At(i, j)
			want := color.RGBA{0x80, 0x80, 0x80, 0x80}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}
