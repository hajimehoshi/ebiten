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
	"encoding/binary"
	"fmt"
	"maps"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TestSidewaysOutlineCacheAcrossSizes(t *testing.T) {
	newSource := func() *text.GoTextFaceSource {
		t.Helper()
		source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
		if err != nil {
			t.Fatal(err)
		}
		return source
	}
	vertices := func(source *text.GoTextFaceSource, size float64) []ebiten.Vertex {
		var path vector.Path
		text.AppendVectorPath(&path, "F", &text.GoTextFace{
			Source:    source,
			Size:      size,
			Direction: text.DirectionTopToBottomAndLeftToRight,
		}, nil)
		v, _ := path.AppendVerticesAndIndicesForFilling(nil, nil)
		return v
	}

	source := newSource()
	vertices(source, 1)
	got := vertices(source, 128)
	want := vertices(newSource(), 128)
	if len(want) == 0 {
		t.Fatal("expected an outline")
	}
	if !slices.Equal(got, want) {
		t.Error("sideways outline differs after rendering at another size")
	}
}

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
		face := &text.GoTextFace{
			Source: src,
			Size:   12 + float64(i)/4,
		}
		text.Draw(dst, "Hello", face, nil)
	}
	if got := text.GlyphImageCacheCount(src); got == 0 {
		t.Fatal("no glyph image cache was created")
	}

	// Pass explicit ticks to simulate updates.
	// Each frame uses a new size, and one size is kept in use all the time.
	const (
		firstTick = 1000
		tickCount = 300
		hotSize   = 12
		// The recent sizes fit below this bound.
		maxCacheCount = 128
	)
	for i := range tickCount {
		now := int64(firstTick + i)
		text.TouchGlyphImageCache(&text.GoTextFace{
			Source: src,
			Size:   hotSize,
		}, now)
		text.TouchGlyphImageCache(&text.GoTextFace{
			Source: src,
			Size:   1000 + float64(i),
		}, now)
		if got := text.GlyphImageCacheCount(src); got > maxCacheCount {
			t.Errorf("the number of the glyph image caches must be <= %d but was %d at tick %d", maxCacheCount, got, now)
		}
	}

	if !text.HasGlyphImageCache(src, hotSize) {
		t.Errorf("the cache for the size %v must not be dropped", float64(hotSize))
	}
	if staleSize := 1000.0; text.HasGlyphImageCache(src, staleSize) {
		t.Errorf("the cache for the size %v must be dropped", staleSize)
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

func TestGlyphImageCacheAfterSizeChange(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "chromacheck-sbix.ttf"))
	if err != nil {
		t.Fatal(err)
	}

	const (
		str     = "\U0000E901"
		oldSize = 16
		newSize = 64
	)
	src := newGoTextFaceSourceForTest(t, data)
	face := &text.GoTextFace{
		Source: src,
		Size:   oldSize,
	}

	glyphs := text.AppendLazyGlyphs(nil, str, face, nil)
	face.Size = newSize
	if img := glyphs[0].Image(); img == nil {
		t.Fatal("Image() = nil, want an image")
	}

	glyphs = text.AppendLazyGlyphs(nil, str, face, nil)
	img := glyphs[0].Image()
	if img == nil {
		t.Fatal("Image() = nil, want an image")
	}
	if got, want := img.Bounds().Size(), glyphs[0].ImageBounds.Size(); got != want {
		t.Errorf("the image size = %v, want %v", got, want)
	}
}

func TestGlyphImageCacheAfterVariationChange(t *testing.T) {
	data := variableFontData(t)
	face := &text.GoTextFace{
		Source: newGoTextFaceSourceForTest(t, data),
		Size:   32,
	}
	weight := text.MustParseTag("wght")
	face.SetVariation(weight, 100)
	glyphs := text.AppendLazyGlyphs(nil, "J", face, nil)
	face.SetVariation(weight, 900)
	if glyphs[0].Image() == nil {
		t.Fatal("Image() = nil, want an image")
	}

	got := text.AppendLazyGlyphs(nil, "J", face, nil)[0].Image()
	fresh := &text.GoTextFace{
		Source: newGoTextFaceSourceForTest(t, data),
		Size:   32,
	}
	fresh.SetVariation(weight, 900)
	want := text.AppendLazyGlyphs(nil, "J", fresh, nil)[0].Image()
	if got == nil || want == nil {
		t.Fatal("Image() = nil, want an image")
	}
	if got.Bounds().Size() != want.Bounds().Size() {
		t.Errorf("image size = %v, want %v", got.Bounds().Size(), want.Bounds().Size())
		return
	}
	gotPixels := make([]byte, 4*got.Bounds().Dx()*got.Bounds().Dy())
	wantPixels := make([]byte, len(gotPixels))
	got.ReadPixels(gotPixels)
	want.ReadPixels(wantPixels)
	if !bytes.Equal(gotPixels, wantPixels) {
		t.Error("glyph pixels differ from a fresh face with the same variation")
	}
}

func TestGoTextFaceCopyOptions(t *testing.T) {
	data := variableFontData(t)
	width, weight := text.MustParseTag("wdth"), text.MustParseTag("wght")
	kern, liga := text.MustParseTag("kern"), text.MustParseTag("liga")
	for _, tc := range []struct {
		name              string
		configure, change func(*text.GoTextFace)
	}{
		{
			name:      "set variation",
			configure: func(f *text.GoTextFace) { f.SetVariation(width, 50); f.SetVariation(weight, 100) },
			change:    func(f *text.GoTextFace) { f.SetVariation(width, 200) },
		},
		{
			name:      "remove variation",
			configure: func(f *text.GoTextFace) { f.SetVariation(width, 50); f.SetVariation(weight, 100) },
			change:    func(f *text.GoTextFace) { f.RemoveVariation(width) },
		},
		{
			name:      "set feature",
			configure: func(f *text.GoTextFace) { f.SetFeature(kern, 0); f.SetFeature(liga, 0) },
			change:    func(f *text.GoTextFace) { f.SetFeature(kern, 1) },
		},
		{
			name:      "remove feature",
			configure: func(f *text.GoTextFace) { f.SetFeature(kern, 0); f.SetFeature(liga, 0) },
			change:    func(f *text.GoTextFace) { f.RemoveFeature(kern) },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := &text.GoTextFace{Source: newGoTextFaceSourceForTest(t, data), Size: 32}
			want := &text.GoTextFace{Source: newGoTextFaceSourceForTest(t, data), Size: 32}
			tc.configure(original)
			tc.configure(want)
			copied := *original
			tc.change(&copied)
			// Shape only after changing the copy, so an already cached original layout
			// cannot hide mutations of the original face's options.
			if got, want := text.Advance("AVffi", original), text.Advance("AVffi", want); got != want {
				t.Errorf("changing the copied face changed the original advance: got %v, want %v", got, want)
			}
		})
	}
}

// Check both the option values and their cached key: a stale key alone can hide
// changes through a shared backing array.
func TestGoTextFaceOptionStorage(t *testing.T) {
	for _, kind := range []struct {
		name        string
		set         func(*text.GoTextFace, text.Tag, uint32)
		remove      func(*text.GoTextFace, text.Tag)
		state       func(*text.GoTextFace) ([]byte, string)
		encodeValue func(uint32) uint32
	}{
		{
			name:   "variations",
			set:    func(f *text.GoTextFace, tag text.Tag, value uint32) { f.SetVariation(tag, float32(value)) },
			remove: (*text.GoTextFace).RemoveVariation,
			state: func(f *text.GoTextFace) ([]byte, string) {
				values, key := text.GoTextFaceVariationState(f)
				var data []byte
				for _, v := range values {
					data = binary.LittleEndian.AppendUint32(data, uint32(v.Tag))
					data = binary.LittleEndian.AppendUint32(data, math.Float32bits(v.Value))
				}
				return data, key
			},
			encodeValue: func(v uint32) uint32 { return math.Float32bits(float32(v)) },
		},
		{
			name:   "features",
			set:    (*text.GoTextFace).SetFeature,
			remove: (*text.GoTextFace).RemoveFeature,
			state: func(f *text.GoTextFace) ([]byte, string) {
				values, key := text.GoTextFaceFeatureState(f)
				var data []byte
				for _, v := range values {
					data = binary.LittleEndian.AppendUint32(data, uint32(v.Tag))
					data = binary.LittleEndian.AppendUint32(data, v.Value)
				}
				return data, key
			},
			encodeValue: func(v uint32) uint32 { return v },
		},
	} {
		for _, count := range []int{1, 3, 4, 5, 6} {
			for _, change := range []string{"replace", "insert", "remove first", "remove middle", "remove last", "no-op set", "no-op remove"} {
				t.Run(fmt.Sprintf("%s/count=%d/%s", kind.name, count, change), func(t *testing.T) {
					var original text.GoTextFace
					wantOriginal := map[text.Tag]uint32{}
					// Reverse insertion must still produce a sorted key.
					for i := count; i > 0; i-- {
						tag := text.Tag(i * 10)
						kind.set(&original, tag, uint32(i))
						wantOriginal[tag] = uint32(i)
					}
					check := func(f *text.GoTextFace, want map[text.Tag]uint32) {
						t.Helper()
						tags := slices.Sorted(maps.Keys(want))
						var expected []byte
						for _, tag := range tags {
							expected = binary.LittleEndian.AppendUint32(expected, uint32(tag))
							expected = binary.LittleEndian.AppendUint32(expected, kind.encodeValue(want[tag]))
						}
						data, key := kind.state(f)
						if !bytes.Equal(data, expected) || key != string(expected) {
							t.Errorf("options/key: got %x/%x, want %x", data, key, expected)
						}
					}
					mutate := func(f *text.GoTextFace, want map[text.Tag]uint32, value uint32) {
						switch change {
						case "replace":
							kind.set(f, 10, value)
							want[10] = value
						case "insert":
							kind.set(f, 15, value)
							want[15] = value
						case "remove first":
							kind.remove(f, 10)
							delete(want, 10)
						case "remove middle":
							tag := text.Tag((count/2 + 1) * 10)
							kind.remove(f, tag)
							delete(want, tag)
						case "remove last":
							tag := text.Tag(count * 10)
							kind.remove(f, tag)
							delete(want, tag)
						case "no-op set":
							kind.set(f, 10, want[10])
						case "no-op remove":
							kind.remove(f, 999)
						}
					}
					copied := original
					wantCopied := maps.Clone(wantOriginal)
					mutate(&original, wantOriginal, 99)
					check(&original, wantOriginal)
					check(&copied, wantCopied)
					mutate(&copied, wantCopied, 77)
					check(&original, wantOriginal)
					check(&copied, wantCopied)
					// Copy again after a boundary transition, then mutate both values.
					again := original
					wantAgain := maps.Clone(wantOriginal)
					kind.set(&original, 999, 123)
					wantOriginal[999] = 123
					check(&again, wantAgain)
					kind.set(&again, 998, 456)
					wantAgain[998] = 456
					check(&original, wantOriginal)
					check(&again, wantAgain)
				})
			}
		}
	}
}

func TestGoTextFaceLazyVariationSnapshot(t *testing.T) {
	data := variableFontData(t)
	weight := text.MustParseTag("wght")
	for _, count := range []int{4, 5} {
		t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
			face := &text.GoTextFace{Source: newGoTextFaceSourceForTest(t, data), Size: 32}
			face.SetVariation(weight, 100)
			// Unrecognized tags leave this font unchanged but exercise both storage modes.
			for i := 1; i < count; i++ {
				face.SetVariation(text.Tag(i), float32(i))
			}
			glyphs := text.AppendLazyGlyphs(nil, "J", face, nil)
			copied := *face
			face.SetVariation(weight, 900)
			copied.SetVariation(weight, 500)
			got := glyphs[0].Image()
			fresh := &text.GoTextFace{Source: newGoTextFaceSourceForTest(t, data), Size: 32}
			fresh.SetVariation(weight, 100)
			want := text.AppendLazyGlyphs(nil, "J", fresh, nil)[0].Image()
			if got == nil || want == nil {
				t.Fatal("expected glyph images")
			}
			if got.Bounds().Size() != want.Bounds().Size() {
				t.Fatalf("snapshot image size: got %v, want %v", got.Bounds().Size(), want.Bounds().Size())
			}
			gotPixels := make([]byte, 4*got.Bounds().Dx()*got.Bounds().Dy())
			wantPixels := make([]byte, len(gotPixels))
			got.ReadPixels(gotPixels)
			want.ReadPixels(wantPixels)
			if !bytes.Equal(gotPixels, wantPixels) {
				t.Error("face mutation changed the pending glyph's variation snapshot")
			}
		})
	}
}

func TestGoTextFaceVariationSpecialValues(t *testing.T) {
	for _, count := range []int{4, 5} {
		for _, tc := range []struct {
			name          string
			before, after uint32
			want          uint32
		}{
			{name: "positive zero", before: 0, after: 0x80000000, want: 0},
			{name: "negative zero", before: 0x80000000, after: 0, want: 0x80000000},
			{name: "NaN", before: 0x7fc00001, after: 0x7fc00002, want: 0x7fc00002},
		} {
			t.Run(fmt.Sprintf("count=%d/%s", count, tc.name), func(t *testing.T) {
				var original text.GoTextFace
				original.SetVariation(1, math.Float32frombits(tc.before))
				for i := 2; i <= count; i++ {
					original.SetVariation(text.Tag(i), float32(i))
				}
				copied := original
				copied.SetVariation(1, math.Float32frombits(tc.after))
				for _, state := range []struct {
					name string
					face *text.GoTextFace
					want uint32
				}{
					{name: "original", face: &original, want: tc.before},
					{name: "copied", face: &copied, want: tc.want},
				} {
					values, key := text.GoTextFaceVariationState(state.face)
					if got := math.Float32bits(values[0].Value); got != state.want {
						t.Errorf("%s value bits: got %08x, want %08x", state.name, got, state.want)
					}
					if got := binary.LittleEndian.Uint32([]byte(key[4:8])); got != state.want {
						t.Errorf("%s key bits: got %08x, want %08x", state.name, got, state.want)
					}
				}
			})
		}
	}
}

func TestGoTextFaceUnchangedOptionsAllocateNothing(t *testing.T) {
	for _, count := range []int{4, 5} {
		t.Run(fmt.Sprintf("count=%d", count), func(t *testing.T) {
			var face text.GoTextFace
			for i := 1; i <= count; i++ {
				face.SetVariation(text.Tag(i), float32(i))
				face.SetFeature(text.Tag(i), uint32(i))
			}
			if got := testing.AllocsPerRun(100, func() {
				face.SetVariation(1, 1)
				face.SetFeature(1, 1)
				face.RemoveVariation(999)
				face.RemoveFeature(999)
			}); got != 0 {
				t.Errorf("unchanged options: got %v allocations, want 0", got)
			}
		})
	}
}
