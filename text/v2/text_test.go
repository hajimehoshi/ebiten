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
	"bytes"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hajimehoshi/bitmapfont/v3"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
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

	f := text.NewGoXFace(bitmapfont.Face)
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
	text.Draw(img, "Hello", text.NewGoXFace(bitmapfont.Face), op)

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

const testGoXFaceSize = 6

type testGoXFace struct{}

func (f *testGoXFace) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	dr = image.Rect(0, 0, testGoXFaceSize, testGoXFaceSize)
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
	advance = fixed.I(testGoXFaceSize)
	ok = true
	return
}

func (f *testGoXFace) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	bounds = fixed.R(0, 0, testGoXFaceSize, testGoXFaceSize)
	advance = fixed.I(testGoXFaceSize)
	ok = true
	return
}

func (f *testGoXFace) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	return fixed.I(testGoXFaceSize), true
}

func (f *testGoXFace) Kern(r0, r1 rune) fixed.Int26_6 {
	if r1 == 'b' {
		return fixed.I(-testGoXFaceSize)
	}
	return 0
}

func (f *testGoXFace) Close() error {
	return nil
}

func (f *testGoXFace) Metrics() font.Metrics {
	return font.Metrics{
		Height:     fixed.I(testGoXFaceSize),
		Ascent:     0,
		Descent:    fixed.I(testGoXFaceSize),
		XHeight:    0,
		CapHeight:  fixed.I(testGoXFaceSize),
		CaretSlope: image.Pt(0, 1),
	}
}

// Issue #1378
func TestNegativeKern(t *testing.T) {
	f := text.NewGoXFace(&testGoXFace{})
	dst := ebiten.NewImage(testGoXFaceSize*2, testGoXFaceSize)

	// With testGoXFace, 'b' is rendered at the previous position as 0xff.
	// 'a' is rendered at the current position as 0x80.
	op := &text.DrawOptions{}
	op.GeoM.Translate(0, 0)
	text.Draw(dst, "ab", f, op)
	for j := 0; j < testGoXFaceSize; j++ {
		for i := 0; i < testGoXFaceSize; i++ {
			got := dst.At(i, j)
			want := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}

	// The glyph 'a' should be treated correctly.
	op = &text.DrawOptions{}
	op.GeoM.Translate(testGoXFaceSize, 0)
	text.Draw(dst, "a", f, op)
	for j := 0; j < testGoXFaceSize; j++ {
		for i := testGoXFaceSize; i < testGoXFaceSize*2; i++ {
			got := dst.At(i, j)
			want := color.RGBA{R: 0x80, G: 0x80, B: 0x80, A: 0x80}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

type unhashableGoXFace func()

const unhashableGoXFaceSize = 10

func (u *unhashableGoXFace) Glyph(dot fixed.Point26_6, r rune) (dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	dr = image.Rect(0, 0, unhashableGoXFaceSize, unhashableGoXFaceSize)
	a := image.NewAlpha(dr)
	for j := dr.Min.Y; j < dr.Max.Y; j++ {
		for i := dr.Min.X; i < dr.Max.X; i++ {
			a.SetAlpha(i, j, color.Alpha{A: 0xff})
		}
	}
	mask = a
	advance = fixed.I(unhashableGoXFaceSize)
	ok = true
	return
}

func (u *unhashableGoXFace) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	bounds = fixed.R(0, 0, unhashableGoXFaceSize, unhashableGoXFaceSize)
	advance = fixed.I(unhashableGoXFaceSize)
	ok = true
	return
}

func (u *unhashableGoXFace) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	return fixed.I(unhashableGoXFaceSize), true
}

func (u *unhashableGoXFace) Kern(r0, r1 rune) fixed.Int26_6 {
	return 0
}

func (u *unhashableGoXFace) Close() error {
	return nil
}

func (u *unhashableGoXFace) Metrics() font.Metrics {
	return font.Metrics{
		Height:     fixed.I(unhashableGoXFaceSize),
		Ascent:     0,
		Descent:    fixed.I(unhashableGoXFaceSize),
		XHeight:    0,
		CapHeight:  fixed.I(unhashableGoXFaceSize),
		CaretSlope: image.Pt(0, 1),
	}
}

// Issue #2669
func TestUnhashableFace(t *testing.T) {
	var face unhashableGoXFace
	f := text.NewGoXFace(&face)
	dst := ebiten.NewImage(unhashableGoXFaceSize*2, unhashableGoXFaceSize*2)
	text.Draw(dst, "a", f, nil)

	for j := 0; j < unhashableGoXFaceSize*2; j++ {
		for i := 0; i < unhashableGoXFaceSize*2; i++ {
			got := dst.At(i, j)
			var want color.RGBA
			if i < unhashableGoXFaceSize && j < unhashableGoXFaceSize {
				want = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
			}
			if got != want {
				t.Errorf("At(%d, %d): got: %v, want: %v", i, j, got, want)
			}
		}
	}
}

func TestConvertToFixed26_6(t *testing.T) {
	testCases := []struct {
		In  float64
		Out fixed.Int26_6
	}{
		{
			In:  0,
			Out: 0,
		},
		{
			In:  0.25,
			Out: fixed.I(1) / 4,
		},
		{
			In:  0.5,
			Out: fixed.I(1) / 2,
		},
		{
			In:  1.25,
			Out: fixed.I(1) * 5 / 4,
		},
		{
			In:  1,
			Out: fixed.I(1),
		},
		{
			In:  -0.25,
			Out: fixed.I(-1) / 4,
		},
		{
			In:  -0.5,
			Out: fixed.I(-1) / 2,
		},
		{
			In:  -1,
			Out: fixed.I(-1),
		},
		{
			In:  -1.25,
			Out: fixed.I(-1) * 5 / 4,
		},
	}

	for _, tc := range testCases {
		got := text.Float32ToFixed26_6(float32(tc.In))
		want := tc.Out
		if got != want {
			t.Errorf("Float32ToFixed26_6(%v): got: %v, want: %v", tc.In, got, want)
		}

		got = text.Float64ToFixed26_6(tc.In)
		want = tc.Out
		if got != want {
			t.Errorf("Float32ToFixed26_6(%v): got: %v, want: %v", tc.In, got, want)
		}
	}
}

func TestConvertToFloat(t *testing.T) {
	testCases := []struct {
		In  fixed.Int26_6
		Out float64
	}{
		{
			In:  0,
			Out: 0,
		},
		{
			In:  fixed.I(1) / 4,
			Out: 0.25,
		},
		{
			In:  fixed.I(1) / 2,
			Out: 0.5,
		},
		{
			In:  fixed.I(1) * 5 / 4,
			Out: 1.25,
		},
		{
			In:  fixed.I(1),
			Out: 1,
		},
		{
			In:  fixed.I(-1) / 4,
			Out: -0.25,
		},
		{
			In:  fixed.I(-1) / 2,
			Out: -0.5,
		},
		{
			In:  fixed.I(-1),
			Out: -1,
		},
		{
			In:  fixed.I(-1) * 5 / 4,
			Out: -1.25,
		},
	}

	for _, tc := range testCases {
		got := text.Fixed26_6ToFloat32(tc.In)
		want := float32(tc.Out)
		if got != want {
			t.Errorf("Fixed26_6ToFloat32(%v): got: %v, want: %v", tc.In, got, want)
		}

		got64 := text.Fixed26_6ToFloat64(tc.In)
		want64 := tc.Out
		if got64 != want64 {
			t.Errorf("Fixed26_6ToFloat64(%v): got: %v, want: %v", tc.In, got64, want64)
		}
	}
}

// Issue #2954
func TestDrawOptionsNotModified(t *testing.T) {
	img := ebiten.NewImage(30, 30)

	op := &text.DrawOptions{}
	text.Draw(img, "Hello", text.NewGoXFace(bitmapfont.Face), op)

	if got, want := op.GeoM, (ebiten.GeoM{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
	if got, want := op.ColorScale, (ebiten.ColorScale{}); got != want {
		t.Errorf("got: %v, want: %v", got, want)
	}
}

func TestGoXFaceMetrics(t *testing.T) {
	const size = 100

	fontFiles := []string{
		// MPLUS1p-Regular.ttf is an old version of M+ 1p font, and this doesn't have metadata.
		"MPLUS1p-Regular.ttf",
		"Roboto-Regular.ttf",
	}

	for _, fontFile := range fontFiles {
		fontFile := fontFile
		t.Run(fontFile, func(t *testing.T) {
			fontdata, err := os.ReadFile(filepath.Join("testdata", fontFile))
			if err != nil {
				t.Fatal(err)
			}

			sfntFont, err := opentype.Parse(fontdata)
			if err != nil {
				t.Fatal(err)
			}
			opentypeFace, err := opentype.NewFace(sfntFont, &opentype.FaceOptions{
				Size: size,
				DPI:  72,
			})
			if err != nil {
				t.Fatal(err)
			}
			goXFace := text.NewGoXFace(opentypeFace)
			goXMetrics := goXFace.Metrics()
			if goXMetrics.XHeight <= 0 {
				t.Errorf("GoXFace's XHeight must be positive but not: %f", goXMetrics.XHeight)
			}
			if goXMetrics.CapHeight <= 0 {
				t.Errorf("GoXFace's CapHeight must be positive but not: %f", goXMetrics.CapHeight)
			}

			goTextFaceSource, err := text.NewGoTextFaceSource(bytes.NewBuffer(fontdata))
			if err != nil {
				t.Fatal(err)
			}
			goTextFace := &text.GoTextFace{
				Source: goTextFaceSource,
				Size:   size,
			}
			goTextMetrics := goTextFace.Metrics()
			if goTextMetrics.XHeight <= 0 {
				t.Errorf("GoTextFace's XHeight must be positive but not: %f", goTextMetrics.XHeight)
			}
			if goTextMetrics.CapHeight <= 0 {
				t.Errorf("GoTextFace's CapHeight must be positive but not: %f", goTextMetrics.CapHeight)
			}

			if math.Abs(goXMetrics.XHeight-goTextMetrics.XHeight) >= 0.1 {
				t.Errorf("XHeight values don't match: %f (GoXFace) vs %f (GoTextFace)", goXMetrics.XHeight, goTextMetrics.XHeight)
			}
			if math.Abs(goXMetrics.CapHeight-goTextMetrics.CapHeight) >= 0.1 {
				t.Errorf("CapHeight values don't match: %f (GoXFace) vs %f (GoTextFace)", goXMetrics.CapHeight, goTextMetrics.CapHeight)
			}

			// Check that a MultiFace should have the same metrics.
			multiFace, err := text.NewMultiFace(goTextFace)
			if err != nil {
				t.Fatal(err)
			}
			if got := multiFace.Metrics(); got != goTextMetrics {
				t.Errorf("got: %v, want: %v", got, goTextMetrics)
			}
		})
	}
}

func BenchmarkDrawText(b *testing.B) {
	var txt string
	for i := 0; i < 32; i++ {
		txt += "The quick brown fox jumps over the lazy dog.\n"
	}
	screen := ebiten.NewImage(16, 16)
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		b.Fatal(err)
	}
	f := &text.GoTextFace{
		Source: source,
		Size:   10,
	}
	op := &text.DrawOptions{}
	for i := 0; i < b.N; i++ {
		text.Draw(screen, txt, f, op)
	}
}