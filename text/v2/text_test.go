// Copyright 2023 The Ebitengine Authors
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
	"regexp"
	"strings"
	"testing"

	"github.com/hajimehoshi/bitmapfont/v3"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"

	"github.com/hajimehoshi/ebiten/v2"
	t "github.com/hajimehoshi/ebiten/v2/internal/testing"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestMain(m *testing.M) {
	t.MainWithRunLoop(m)
}

func TestGlyphIndex(t *testing.T) {
	const sampleText = `The quick brown fox jumps
over the lazy dog.`

	f := text.NewStdFace(bitmapfont.Face)
	got := sampleText
	for _, g := range text.AppendGlyphs(nil, sampleText, f, nil) {
		got = got[:g.StartIndexInBytes] + strings.Repeat(" ", g.EndIndexInBytes-g.StartIndexInBytes) + got[g.EndIndexInBytes:]
	}
	want := regexp.MustCompile(`\S`).ReplaceAllString(sampleText, " ")
	if got != want {
		t.Errorf("got: %q, want: %q", got, want)
	}
}

func TestTextColor(t *testing.T) {
	clr := color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}
	img := ebiten.NewImage(30, 30)
	op := &text.DrawOptions{}
	op.GeoM.Translate(0, 0)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(img, "Hello", text.NewStdFace(bitmapfont.Face), op)

	w, h := img.Bounds().Dx(), img.Bounds().Dy()
	allTransparent := true
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			got := img.At(i, j)
			want1 := color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}
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

const testStdFaceSize = 6

type testStdFace struct{}

func (f *testStdFace) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	dr = image.Rect(0, 0, testStdFaceSize, testStdFaceSize)
	a := image.NewAlpha(dr)
	switch r {
	case 'a':
		for j := dr.Min.Y; j < dr.Max.Y; j++ {
			for i := dr.Min.X; i < dr.Max.X; i++ {
				a.SetAlpha(i, j, color.Alpha{A: 0x80})
			}
		}
	case 'b':
		for j := dr.Min.Y; j < dr.Max.Y; j++ {
			for i := dr.Min.X; i < dr.Max.X; i++ {
				a.SetAlpha(i, j, color.Alpha{A: 0xff})
			}
		}
	}
	mask = a
	advance = fixed.I(testStdFaceSize)
	ok = true
	return
}

func (f *testStdFace) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	bounds = fixed.R(0, 0, testStdFaceSize, testStdFaceSize)
	advance = fixed.I(testStdFaceSize)
	ok = true
	return
}

func (f *testStdFace) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	return fixed.I(testStdFaceSize), true
}

func (f *testStdFace) Kern(r0, r1 rune) fixed.Int26_6 {
	if r1 == 'b' {
		return fixed.I(-testStdFaceSize)
	}
	return 0
}

func (f *testStdFace) Close() error {
	return nil
}

func (f *testStdFace) Metrics() font.Metrics {
	return font.Metrics{
		Height:     fixed.I(testStdFaceSize),
		Ascent:     0,
		Descent:    fixed.I(testStdFaceSize),
		XHeight:    0,
		CapHeight:  fixed.I(testStdFaceSize),
		CaretSlope: image.Pt(0, 1),
	}
}

// Issue #1378
func TestNegativeKern(t *testing.T) {
	f := text.NewStdFace(&testStdFace{})
	dst := ebiten.NewImage(testStdFaceSize*2, testStdFaceSize)

	// With testStdFace, 'b' is rendered at the previous position as 0xff.
	// 'a' is rendered at the current position as 0x80.
	op := &text.DrawOptions{}
	op.GeoM.Translate(0, 0)
	text.Draw(dst, "ab", f, op)
	for j := 0; j < testStdFaceSize; j++ {
		for i := 0; i < testStdFaceSize; i++ {
			got := dst.At(i, j)
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// The glyph 'a' should be treated correctly.
	op = &text.DrawOptions{}
	op.GeoM.Translate(testStdFaceSize, 0)
	text.Draw(dst, "a", f, op)
	for j := 0; j < testStdFaceSize; j++ {
		for i := testStdFaceSize; i < testStdFaceSize*2; i++ {
			got := dst.At(i, j)
			want := color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

type unhashableStdFace func()

const unhashableStdFaceSize = 10

func (u *unhashableStdFace) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	dr = image.Rect(0, 0, unhashableStdFaceSize, unhashableStdFaceSize)
	a := image.NewAlpha(dr)
	for j := dr.Min.Y; j < dr.Max.Y; j++ {
		for i := dr.Min.X; i < dr.Max.X; i++ {
			a.SetAlpha(i, j, color.Alpha{A: 0xff})
		}
	}
	mask = a
	advance = fixed.I(unhashableStdFaceSize)
	ok = true
	return
}

func (u *unhashableStdFace) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	bounds = fixed.R(0, 0, unhashableStdFaceSize, unhashableStdFaceSize)
	advance = fixed.I(unhashableStdFaceSize)
	ok = true
	return
}

func (u *unhashableStdFace) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	return fixed.I(unhashableStdFaceSize), true
}

func (u *unhashableStdFace) Kern(r0, r1 rune) fixed.Int26_6 {
	return 0
}

func (u *unhashableStdFace) Close() error {
	return nil
}

func (u *unhashableStdFace) Metrics() font.Metrics {
	return font.Metrics{
		Height:     fixed.I(unhashableStdFaceSize),
		Ascent:     0,
		Descent:    fixed.I(unhashableStdFaceSize),
		XHeight:    0,
		CapHeight:  fixed.I(unhashableStdFaceSize),
		CaretSlope: image.Pt(0, 1),
	}
}

// Issue #2669
func TestUnhashableFace(t *testing.T) {
	var face unhashableStdFace
	f := text.NewStdFace(&face)
	dst := ebiten.NewImage(unhashableStdFaceSize*2, unhashableStdFaceSize*2)
	text.Draw(dst, "a", f, nil)

	for j := 0; j < unhashableStdFaceSize*2; j++ {
		for i := 0; i < unhashableStdFaceSize*2; i++ {
			got := dst.At(i, j)
			var want color.RGBA
			if i < unhashableStdFaceSize && j < unhashableStdFaceSize {
				want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}
