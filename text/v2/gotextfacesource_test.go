// Copyright 2026 The Ebitengine Authors
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
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestGlyphImageCacheConcurrent(t *testing.T) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	face := &text.GoTextFace{Source: src, Size: 16}
	glyphs := text.AppendLazyGlyphs(nil, "Hello, 世界!", face, nil)

	const goroutines = 8
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for _, gl := range glyphs {
				// A space or a control character has no image.
				if gl.ImageBounds.Empty() {
					continue
				}
				if gl.Image() == nil {
					t.Error("Image() returned nil for a glyph which should have an image")
				}
			}
		})
	}
	wg.Wait()
}

func TestGlyphImageCacheSizeEviction(t *testing.T) {
	for _, tps := range []int{30, 120} {
		t.Run(fmt.Sprint(tps), func(t *testing.T) {
			src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
			if err != nil {
				t.Fatal(err)
			}

			dst := ebiten.NewImage(64, 64)

			// Draw with many distinct sizes, like a game animating its font size.
			const drawnSizeCount = 100
			for i := range drawnSizeCount {
				face := &text.GoTextFace{Source: src, Size: 12 + float64(i)/4}
				text.Draw(dst, "Hello", face, nil)
			}
			if got := text.GlyphImageCacheCount(src); got == 0 {
				t.Fatal("no glyph image cache was created")
			}

			// Pass explicit logical times to simulate frames.
			// Each frame uses a new size, and one size is kept in use all the time.
			const (
				firstTick = 1000
				tickCount = 300
				hotSize   = 12
				// The tested rates retain fewer than 128 recent sizes.
				maxCacheCount = 128
			)
			for i := range tickCount {
				now := ebiten.Duration(firstTick+i) * ebiten.DurationSecond / ebiten.Duration(tps)
				text.TouchGlyphImageCache(&text.GoTextFace{Source: src, Size: hotSize}, now)
				text.TouchGlyphImageCache(&text.GoTextFace{Source: src, Size: 1000 + float64(i)}, now)
				if got := text.GlyphImageCacheCount(src); got > maxCacheCount {
					t.Errorf("the number of the glyph image caches must be <= %d but was %d at time %d", maxCacheCount, got, now)
				}
			}

			if !text.HasGlyphImageCache(src, hotSize) {
				t.Errorf("the cache for the size %v must not be dropped", float64(hotSize))
			}
			if staleSize := 1000.0; text.HasGlyphImageCache(src, staleSize) {
				t.Errorf("the cache for the size %v must be dropped", staleSize)
			}

		})
	}
}

func TestGlyphImageCacheSideways(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "MPLUS1p-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}

	pixels := func(img *ebiten.Image) []byte {
		if img == nil {
			return nil
		}
		b := img.Bounds()
		pix := make([]byte, 4*b.Dx()*b.Dy())
		img.ReadPixels(pix)
		return pix
	}

	const (
		str  = "L"
		size = 200
	)
	for _, horizontalFirst := range []bool{true, false} {
		src := newGoTextFaceSourceForTest(t, data)
		horizontal := &text.GoTextFace{
			Source: src,
			Size:   size,
		}
		vertical := &text.GoTextFace{
			Source:    src,
			Size:      size,
			Direction: text.DirectionTopToBottomAndLeftToRight,
		}
		var hg, vg []text.Glyph
		if horizontalFirst {
			hg = text.AppendGlyphs(nil, str, horizontal, nil)
			vg = text.AppendGlyphs(nil, str, vertical, nil)
		} else {
			vg = text.AppendGlyphs(nil, str, vertical, nil)
			hg = text.AppendGlyphs(nil, str, horizontal, nil)
		}
		if len(hg) != 1 || len(vg) != 1 {
			t.Fatalf("got %d horizontal and %d vertical glyphs, want 1 each", len(hg), len(vg))
		}
		if hg[0].Image == nil || vg[0].Image == nil {
			t.Fatal("a glyph image is nil")
		}
		hb := hg[0].Image.Bounds()
		vb := vg[0].Image.Bounds()
		if hb.Dx() != vb.Dy() || hb.Dy() != vb.Dx() {
			t.Errorf("horizontal first: %t: vertical glyph image size %dx%d is not the horizontal size %dx%d rotated", horizontalFirst, vb.Dx(), vb.Dy(), hb.Dx(), hb.Dy())
		}
		if bytes.Equal(pixels(hg[0].Image), pixels(vg[0].Image)) {
			t.Errorf("horizontal first: %t: the vertical glyph image must not be the horizontal glyph image", horizontalFirst)
		}
	}
}

// variableFontData returns the bytes of a font with variation axes.
func variableFontData(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "RobotoFlex.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func newGoTextFaceSourceForTest(t *testing.T, data []byte) *text.GoTextFaceSource {
	t.Helper()
	src, err := text.NewGoTextFaceSource(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func parseTestFont(t *testing.T, data []byte) *font.Font {
	t.Helper()
	l, err := opentype.NewLoader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	f, err := font.NewFont(l)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func goTextFaceMetrics(t *testing.T, f *font.Font, size float64, variations []font.Variation) text.Metrics {
	t.Helper()

	face := font.NewFace(f)
	face.SetVariations(variations)

	var m text.Metrics
	if h, ok := face.FontHExtents(); ok {
		m.HLineGap = float64(h.LineGap)
		m.HAscent = float64(h.Ascender)
		m.HDescent = float64(-h.Descender)
	}
	if v, ok := face.FontVExtents(); ok {
		m.VLineGap = float64(v.LineGap)
		m.VAscent = float64(v.Ascender)
		m.VDescent = float64(-v.Descender)
	}
	m.XHeight = float64(face.LineMetric(font.XHeight))
	m.CapHeight = float64(face.LineMetric(font.CapHeight))

	scale := size / float64(face.Upem())
	m.HLineGap *= scale
	m.HAscent *= scale
	m.HDescent *= scale
	m.VLineGap *= scale
	m.VAscent *= scale
	m.VDescent *= scale
	m.XHeight *= scale
	m.CapHeight *= scale
	return m
}

func newVariedGoTextFace(src *text.GoTextFaceSource, size float64, variations []font.Variation) *text.GoTextFace {
	face := &text.GoTextFace{Source: src, Size: size}
	for _, v := range variations {
		face.SetVariation(text.Tag(v.Tag), v.Value)
	}
	return face
}

func TestGoTextFaceSourceMetricsWithVariations(t *testing.T) {
	data := variableFontData(t)
	f := parseTestFont(t, data)
	const (
		size   = 16
		sample = "Hello, world!"
	)
	wght := text.MustParseTag("wght")

	src := newGoTextFaceSourceForTest(t, data)
	defaultFace := &text.GoTextFace{Source: src, Size: size}
	defaultMetrics := defaultFace.Metrics()
	if got, want := defaultMetrics, goTextFaceMetrics(t, f, size, nil); got != want {
		t.Errorf("default Metrics(): got: %v, want: %v", got, want)
	}

	var changed bool
	for _, weight := range []float32{100, 400, 900} {
		variations := []font.Variation{
			{
				Tag:   font.Tag(wght),
				Value: weight,
			},
		}
		varied := newVariedGoTextFace(src, size, variations)
		got := varied.Metrics()
		want := goTextFaceMetrics(t, f, size, variations)
		if got != want {
			t.Errorf("wght=%v Metrics(): got: %v, want: %v", weight, got, want)
		}
		if got.XHeight != defaultMetrics.XHeight || got.CapHeight != defaultMetrics.CapHeight {
			changed = true
		}

		fresh := newVariedGoTextFace(newGoTextFaceSourceForTest(t, data), size, variations)
		if got, want := text.Advance(sample, varied), text.Advance(sample, fresh); got != want {
			t.Errorf("wght=%v Advance after Metrics(): got: %v, want: %v", weight, got, want)
		}

		text.Measure(sample, varied, 0)
		if got := defaultFace.Metrics(); got != defaultMetrics {
			t.Errorf("default Metrics() after shaping wght=%v: got: %v, want: %v", weight, got, defaultMetrics)
		}
		if got := (&text.GoTextFace{Source: src, Size: size}).Metrics(); got != defaultMetrics {
			t.Errorf("new default Metrics() after shaping wght=%v: got: %v, want: %v", weight, got, defaultMetrics)
		}
	}
	if !changed {
		t.Error("RobotoFlex wght did not change XHeight or CapHeight")
	}

	regularFont := parseTestFont(t, goregular.TTF)
	regular := newGoTextFaceSourceForTest(t, goregular.TTF)
	variations := []font.Variation{
		{
			Tag:   font.Tag(wght),
			Value: 900,
		},
	}
	got := newVariedGoTextFace(regular, size, variations).Metrics()
	if want := goTextFaceMetrics(t, regularFont, size, variations); got != want {
		t.Errorf("non-variable font Metrics(): got: %v, want: %v", got, want)
	}
	if plain := (&text.GoTextFace{Source: regular, Size: size}).Metrics(); got != plain {
		t.Errorf("variation changed a non-variable font: plain %v, varied %v", plain, got)
	}
}

func TestGoTextFaceSourceMetricsConcurrentWithShaping(t *testing.T) {
	data := variableFontData(t)
	f := parseTestFont(t, data)
	const size = 16
	wght := text.MustParseTag("wght")
	wdth := text.MustParseTag("wdth")

	variations := [][]font.Variation{
		nil,
		{
			{
				Tag:   font.Tag(wght),
				Value: 900,
			},
		},
		{
			{
				Tag:   font.Tag(wght),
				Value: 250,
			},
			{
				Tag:   font.Tag(wdth),
				Value: 120,
			},
		},
	}

	src := newGoTextFaceSourceForTest(t, data)
	faces := make([]*text.GoTextFace, len(variations))
	wants := make([]text.Metrics, len(variations))
	for i, v := range variations {
		faces[i] = newVariedGoTextFace(src, size, v)
		wants[i] = goTextFaceMetrics(t, f, size, v)
	}

	const (
		goroutines = 4
		iterations = 20
		sample     = "Hello, world!"
	)
	var wg sync.WaitGroup
	for i := range faces {
		face := faces[i]
		want := wants[i]
		for range goroutines {
			wg.Go(func() {
				for range iterations {
					if got := face.Metrics(); got != want {
						t.Errorf("Metrics(): got: %v, want: %v", got, want)
					}
					text.Measure(sample, face, 2)
				}
			})
		}
	}
	wg.Wait()
}
