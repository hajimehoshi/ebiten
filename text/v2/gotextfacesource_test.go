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
	"os"
	"path/filepath"
	"sync"
	"testing"

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

	// A tick doesn't advance in a test, so pass explicit ticks to simulate frames.
	// Each frame uses a new size, and one size is kept in use all the time.
	const (
		firstTick = 1000
		tickCount = 300
		hotSize   = 12
		// A cache is dropped after it is unused for 60 ticks. 128 is an arbitrary bound above that.
		maxCacheCount = 128
	)
	for i := range tickCount {
		tick := int64(firstTick + i)
		text.TouchGlyphImageCache(&text.GoTextFace{Source: src, Size: hotSize}, tick)
		text.TouchGlyphImageCache(&text.GoTextFace{Source: src, Size: 1000 + float64(i)}, tick)
		if got := text.GlyphImageCacheCount(src); got > maxCacheCount {
			t.Fatalf("the number of the glyph image caches must be <= %d but was %d at tick %d", maxCacheCount, got, tick)
		}
	}

	if !text.HasGlyphImageCache(src, hotSize) {
		t.Errorf("the cache for the size %v must not be dropped", float64(hotSize))
	}
	if staleSize := 1000.0; text.HasGlyphImageCache(src, staleSize) {
		t.Errorf("the cache for the size %v must be dropped", staleSize)
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

func TestGoTextFaceSourceMetricsWithVariations(t *testing.T) {
	data := variableFontData(t)
	const size = 16

	want := (&text.GoTextFace{Source: newGoTextFaceSourceForTest(t, data), Size: size}).Metrics()

	wght := text.MustParseTag("wght")
	for _, weight := range []float32{100, 400, 900} {
		src := newGoTextFaceSourceForTest(t, data)
		varied := &text.GoTextFace{Source: src, Size: size}
		varied.SetVariation(wght, weight)
		text.Measure("Hello, world!", varied, 0)

		if got := (&text.GoTextFace{Source: src, Size: size}).Metrics(); got != want {
			t.Errorf("Metrics() after shaping with wght=%v: got: %v, want: %v", weight, got, want)
		}
	}
}

func TestGoTextFaceSourceMetricsConcurrentWithShaping(t *testing.T) {
	data := variableFontData(t)
	wght := text.MustParseTag("wght")

	// A source reads its metrics only once, so a fresh source is needed on
	// every iteration.
	const iterations = 50
	for i := range iterations {
		src := newGoTextFaceSourceForTest(t, data)
		varied := &text.GoTextFace{Source: src, Size: 16}
		varied.SetVariation(wght, float32(100+i*10))
		plain := &text.GoTextFace{Source: src, Size: 16}

		var wg sync.WaitGroup
		wg.Go(func() {
			text.Measure("Hello, world!", varied, 0)
		})
		wg.Go(func() {
			plain.Metrics()
		})
		wg.Wait()
	}
}
