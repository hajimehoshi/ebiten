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
	const size = 16
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

		text.Measure("Hello, world!", varied, 0)
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
}

func TestGoTextFaceSourceMetricsSharedSource(t *testing.T) {
	data := variableFontData(t)
	f := parseTestFont(t, data)
	const (
		size   = 16
		sample = "Hello, world!"
		weight = float32(900)
	)
	wght := text.MustParseTag("wght")
	variations := []font.Variation{
		{
			Tag:   font.Tag(wght),
			Value: weight,
		},
	}
	wantPlain := goTextFaceMetrics(t, f, size, nil)
	wantVaried := goTextFaceMetrics(t, f, size, variations)

	for _, variedFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("variedFirst=%t", variedFirst), func(t *testing.T) {
			src := newGoTextFaceSourceForTest(t, data)
			plain := &text.GoTextFace{Source: src, Size: size}
			varied := newVariedGoTextFace(src, size, variations)

			check := func(face *text.GoTextFace, want text.Metrics, name string) {
				t.Helper()
				if got := face.Metrics(); got != want {
					t.Errorf("%s Metrics(): got: %v, want: %v", name, got, want)
				}
			}

			if variedFirst {
				check(varied, wantVaried, "varied")
				check(plain, wantPlain, "plain")
			} else {
				check(plain, wantPlain, "plain")
				check(varied, wantVaried, "varied")
			}

			freshPlainSrc := newGoTextFaceSourceForTest(t, data)
			freshVariedSrc := newGoTextFaceSourceForTest(t, data)
			freshPlain := &text.GoTextFace{Source: freshPlainSrc, Size: size}
			freshVaried := newVariedGoTextFace(freshVariedSrc, size, variations)

			if variedFirst {
				text.Measure(sample, plain, 0)
				text.Measure(sample, varied, 0)
			} else {
				text.Measure(sample, varied, 0)
				text.Measure(sample, plain, 0)
			}
			if got, want := text.Advance(sample, plain), text.Advance(sample, freshPlain); got != want {
				t.Errorf("plain Advance: got: %v, want: %v", got, want)
			}
			if got, want := text.Advance(sample, varied), text.Advance(sample, freshVaried); got != want {
				t.Errorf("varied Advance: got: %v, want: %v", got, want)
			}

			check(plain, wantPlain, "plain after Measure")
			check(varied, wantVaried, "varied after Measure")
			check(varied, wantVaried, "varied cached")
			check(plain, wantPlain, "plain cached")
		})
	}
}

func TestGoTextFaceSourceMetricsVariationUpdate(t *testing.T) {
	data := variableFontData(t)
	f := parseTestFont(t, data)
	const size = 16
	wght := text.MustParseTag("wght")
	wdth := text.MustParseTag("wdth")

	src := newGoTextFaceSourceForTest(t, data)
	face := &text.GoTextFace{Source: src, Size: size}

	check := func(variations []font.Variation) {
		t.Helper()
		if got, want := face.Metrics(), goTextFaceMetrics(t, f, size, variations); got != want {
			t.Errorf("variations %v: got: %v, want: %v", variations, got, want)
		}
	}

	face.SetVariation(wght, 100)
	check([]font.Variation{
		{
			Tag:   font.Tag(wght),
			Value: 100,
		},
	})

	face.SetVariation(wght, 900)
	check([]font.Variation{
		{
			Tag:   font.Tag(wght),
			Value: 900,
		},
	})

	face.SetVariation(wdth, 70)
	check([]font.Variation{
		{
			Tag:   font.Tag(wght),
			Value: 900,
		},
		{
			Tag:   font.Tag(wdth),
			Value: 70,
		},
	})

	face.RemoveVariation(wght)
	check([]font.Variation{
		{
			Tag:   font.Tag(wdth),
			Value: 70,
		},
	})

	face.RemoveVariation(wdth)
	check(nil)

	wghtThenWdth := newVariedGoTextFace(src, size, []font.Variation{
		{
			Tag:   font.Tag(wght),
			Value: 250,
		},
		{
			Tag:   font.Tag(wdth),
			Value: 80,
		},
	})
	wdthThenWght := newVariedGoTextFace(src, size, []font.Variation{
		{
			Tag:   font.Tag(wdth),
			Value: 80,
		},
		{
			Tag:   font.Tag(wght),
			Value: 250,
		},
	})
	want := goTextFaceMetrics(t, f, size, []font.Variation{
		{
			Tag:   font.Tag(wght),
			Value: 250,
		},
		{
			Tag:   font.Tag(wdth),
			Value: 80,
		},
	})
	if got := wghtThenWdth.Metrics(); got != want {
		t.Errorf("wght then wdth: got: %v, want: %v", got, want)
	}
	if got := wdthThenWght.Metrics(); got != want {
		t.Errorf("wdth then wght: got: %v, want: %v", got, want)
	}
}

func TestGoTextFaceSourceMetricsScaling(t *testing.T) {
	variableData := variableFontData(t)
	variableFont := parseTestFont(t, variableData)
	regularFont := parseTestFont(t, goregular.TTF)
	wght := text.MustParseTag("wght")
	heavy := []font.Variation{
		{
			Tag:   font.Tag(wght),
			Value: 900,
		},
	}

	for _, tc := range []struct {
		name        string
		data        []byte
		font        *font.Font
		variations  []font.Variation
		nonVariable bool
	}{
		{
			name: "RobotoFlex default",
			data: variableData,
			font: variableFont,
		},
		{
			name:       "RobotoFlex wght900",
			data:       variableData,
			font:       variableFont,
			variations: heavy,
		},
		{
			name:        "goregular",
			data:        goregular.TTF,
			font:        regularFont,
			nonVariable: true,
		},
		{
			name:        "goregular varied",
			data:        goregular.TTF,
			font:        regularFont,
			variations:  heavy,
			nonVariable: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := newGoTextFaceSourceForTest(t, tc.data)
			for _, size := range []float64{0, 16, 32} {
				face := newVariedGoTextFace(src, size, tc.variations)
				want := goTextFaceMetrics(t, tc.font, size, tc.variations)
				if got := face.Metrics(); got != want {
					t.Errorf("size %v Metrics(): got: %v, want: %v", size, got, want)
				}
			}

			m16 := newVariedGoTextFace(src, 16, tc.variations).Metrics()
			m32 := newVariedGoTextFace(src, 32, tc.variations).Metrics()
			if m16.XHeight == 0 || m16.CapHeight == 0 {
				t.Fatalf("size 16 metrics have a zero x-height or cap-height: %v", m16)
			}
			if got, want := m32.XHeight/m16.XHeight, 2.0; got != want {
				t.Errorf("XHeight ratio: got: %v, want: %v", got, want)
			}
			if got, want := m32.CapHeight/m16.CapHeight, 2.0; got != want {
				t.Errorf("CapHeight ratio: got: %v, want: %v", got, want)
			}
			if got := newVariedGoTextFace(src, 0, tc.variations).Metrics(); got != (text.Metrics{}) {
				t.Errorf("size 0 Metrics(): got: %v, want zero", got)
			}

			if tc.nonVariable {
				plain := newVariedGoTextFace(src, 16, nil).Metrics()
				if plain != m16 {
					t.Errorf("variation changed a non-variable font: plain %v, varied %v", plain, m16)
				}
			}
		})
	}
}

func TestGoTextFaceSourceMetricsConcurrentWithShaping(t *testing.T) {
	data := variableFontData(t)
	f := parseTestFont(t, data)
	const size = 16
	wght := text.MustParseTag("wght")
	wdth := text.MustParseTag("wdth")

	type faceSpec struct {
		variations []font.Variation
		warm       bool
	}
	specs := []faceSpec{
		{
			warm: true,
		},
		{
			variations: []font.Variation{
				{
					Tag:   font.Tag(wght),
					Value: 100,
				},
			},
			warm: true,
		},
		{
			variations: []font.Variation{
				{
					Tag:   font.Tag(wght),
					Value: 900,
				},
			},
		},
		{
			variations: []font.Variation{
				{
					Tag:   font.Tag(wdth),
					Value: 70,
				},
			},
		},
		{
			variations: []font.Variation{
				{
					Tag:   font.Tag(wght),
					Value: 250,
				},
				{
					Tag:   font.Tag(wdth),
					Value: 120,
				},
			},
		},
	}

	src := newGoTextFaceSourceForTest(t, data)
	faces := make([]*text.GoTextFace, len(specs))
	wants := make([]text.Metrics, len(specs))
	for i, spec := range specs {
		faces[i] = newVariedGoTextFace(src, size, spec.variations)
		wants[i] = goTextFaceMetrics(t, f, size, spec.variations)
		if spec.warm {
			if got := faces[i].Metrics(); got != wants[i] {
				t.Fatalf("warm Metrics(): got: %v, want: %v", got, wants[i])
			}
		}
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
