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
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-text/typesetting/font/opentype"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// TestDrawColorGlyph tests that color glyphs are rendered in color (#2649).
// Each ChromaCheck font maps one rune to a glyph filling the em square
// above the baseline with a color identifying the glyph format.
func TestDrawColorGlyph(t *testing.T) {
	for _, tc := range []struct {
		font string
		str  string
		size float64
		want color.RGBA
	}{
		// An OpenType SVG glyph.
		{font: "chromacheck-svg.ttf", str: "\uE902", size: 16, want: color.RGBA{R: 0x32, A: 0xff}},
		// A CBDT bitmap glyph, at the strike size (80) and at scaled sizes.
		{font: "chromacheck-cbdt.ttf", str: "\uE903", size: 80, want: color.RGBA{R: 0x64, A: 0xff}},
		{font: "chromacheck-cbdt.ttf", str: "\uE903", size: 16, want: color.RGBA{R: 0x64, A: 0xff}},
		{font: "chromacheck-cbdt.ttf", str: "\uE903", size: 120, want: color.RGBA{R: 0x64, A: 0xff}},
		// An sbix bitmap glyph, at the strike size (300) and at a scaled size.
		{font: "chromacheck-sbix.ttf", str: "\uE901", size: 300, want: color.RGBA{R: 0x96, A: 0xff}},
		{font: "chromacheck-sbix.ttf", str: "\uE901", size: 16, want: color.RGBA{R: 0x96, A: 0xff}},
		// A COLRv0 layered glyph.
		{font: "chromacheck-colr.ttf", str: "\uE900", size: 16, want: color.RGBA{R: 0xc8, A: 0xff}},
	} {
		t.Run(fmt.Sprintf("%s/%v", tc.font, tc.size), func(t *testing.T) {
			f, err := os.Open(filepath.Join("testdata", tc.font))
			if err != nil {
				t.Fatal(err)
			}
			defer func() {
				_ = f.Close()
			}()

			src, err := text.NewGoTextFaceSource(f)
			if err != nil {
				t.Fatal(err)
			}

			face := &text.GoTextFace{
				Source: src,
				Size:   tc.size,
			}

			glyphs := text.AppendGlyphs(nil, tc.str, face, nil)
			if len(glyphs) != 1 {
				t.Fatalf("len(glyphs): got: %d, want: 1", len(glyphs))
			}
			g := glyphs[0]
			if g.Image == nil {
				t.Fatal("g.Image is nil")
			}
			if !g.Colored {
				t.Error("Colored must be true for a color glyph")
			}

			lazyGlyphs := text.AppendLazyGlyphs(nil, tc.str, face, nil)
			if len(lazyGlyphs) != 1 {
				t.Fatalf("len(lazyGlyphs): got: %d, want: 1", len(lazyGlyphs))
			}
			if !lazyGlyphs[0].Colored() {
				t.Error("Colored() must be true for a color glyph")
			}

			b := g.Image.Bounds()
			if got, want := b.Dx(), int(tc.size); got < want || got > want+1 {
				t.Errorf("image width: got: %d, want: %d or %d", got, want, want+1)
			}
			got := g.Image.At(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)
			if got != tc.want {
				t.Errorf("got: %v, want: %v", got, tc.want)
			}

			// The glyph must also render in color via Draw.
			dst := ebiten.NewImage(int(tc.size)*2, int(tc.size)*2)
			defer dst.Deallocate()
			text.Draw(dst, tc.str, face, nil)
			m := face.Metrics()
			gotDst := dst.At(int(tc.size)/2, int(m.HAscent)-int(tc.size)/2)
			if gotDst != tc.want {
				t.Errorf("Draw: got: %v, want: %v", gotDst, tc.want)
			}

			// The text color must not tint a color glyph (#3494).
			dst.Clear()
			op := &text.DrawOptions{}
			op.ColorScale.ScaleWithColor(color.Black)
			text.Draw(dst, tc.str, face, op)
			gotDst = dst.At(int(tc.size)/2, int(m.HAscent)-int(tc.size)/2)
			if gotDst != tc.want {
				t.Errorf("Draw with a black color scale: got: %v, want: %v", gotDst, tc.want)
			}

			// The alpha of the color scale must still apply to a color glyph.
			dst.Clear()
			op = &text.DrawOptions{}
			op.ColorScale.ScaleAlpha(0.5)
			text.Draw(dst, tc.str, face, op)
			gotDst = dst.At(int(tc.size)/2, int(m.HAscent)-int(tc.size)/2)
			wantAlpha := color.RGBA{
				R: uint8(int(tc.want.R) / 2),
				G: uint8(int(tc.want.G) / 2),
				B: uint8(int(tc.want.B) / 2),
				A: uint8(int(tc.want.A) / 2),
			}
			if got, ok := gotDst.(color.RGBA); !ok || colorDiff(got, wantAlpha) > 1 {
				t.Errorf("Draw with a half alpha: got: %v, want: %v", gotDst, wantAlpha)
			}
		})
	}
}

func TestGlyphColoredGrayscale(t *testing.T) {
	f, err := os.Open(filepath.Join("testdata", "Roboto-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = f.Close()
	}()

	src, err := text.NewGoTextFaceSource(f)
	if err != nil {
		t.Fatal(err)
	}

	face := &text.GoTextFace{
		Source: src,
		Size:   16,
	}
	for _, g := range text.AppendGlyphs(nil, "Sushi", face, nil) {
		if g.Colored {
			t.Errorf("Colored must be false for a grayscale glyph (GID: %d)", g.GID)
		}
	}
	for _, g := range text.AppendLazyGlyphs(nil, "Sushi", face, nil) {
		if g.Colored() {
			t.Errorf("Colored() must be false for a grayscale glyph (GID: %d)", g.GID)
		}
	}
}

func newCOLRv0FaceSource(t *testing.T, paletteIndices []uint16) *text.GoTextFaceSource {
	t.Helper()

	b, err := os.ReadFile(filepath.Join("testdata", "chromacheck-colr.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	ld, err := opentype.NewLoader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}

	const (
		gid                 = 1
		colrHeaderSize      = 14
		baseGlyphRecordSize = 6
	)
	colr := binary.BigEndian.AppendUint16(nil, 0)
	colr = binary.BigEndian.AppendUint16(colr, 1)
	colr = binary.BigEndian.AppendUint32(colr, colrHeaderSize)
	colr = binary.BigEndian.AppendUint32(colr, colrHeaderSize+baseGlyphRecordSize)
	colr = binary.BigEndian.AppendUint16(colr, uint16(len(paletteIndices)))
	colr = binary.BigEndian.AppendUint16(colr, gid)
	colr = binary.BigEndian.AppendUint16(colr, 0)
	colr = binary.BigEndian.AppendUint16(colr, uint16(len(paletteIndices)))
	for _, p := range paletteIndices {
		colr = binary.BigEndian.AppendUint16(colr, gid)
		colr = binary.BigEndian.AppendUint16(colr, p)
	}

	var tables []opentype.Table
	for _, tag := range ld.Tables() {
		content, err := ld.RawTable(tag)
		if err != nil {
			t.Fatal(err)
		}
		if tag == opentype.MustNewTag("COLR") {
			content = colr
		}
		tables = append(tables, opentype.Table{
			Tag:     tag,
			Content: content,
		})
	}

	src, err := text.NewGoTextFaceSource(bytes.NewReader(opentype.WriteTTF(tables)))
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func TestDrawCOLRv0ForegroundLayer(t *testing.T) {
	for _, tc := range []struct {
		name           string
		paletteIndices []uint16
		colored        bool
		want           color.RGBA
	}{
		{
			name:           "foreground",
			paletteIndices: []uint16{0xffff},
			colored:        false,
			want:           color.RGBA{G: 0xff, A: 0xff},
		},
		{
			name:           "mixed",
			paletteIndices: []uint16{0xffff, 0},
			colored:        true,
			want:           color.RGBA{R: 0xc8, A: 0xff},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const (
				str  = "\uE900"
				size = 16
			)
			face := &text.GoTextFace{
				Source: newCOLRv0FaceSource(t, tc.paletteIndices),
				Size:   size,
			}

			glyphs := text.AppendGlyphs(nil, str, face, nil)
			if len(glyphs) != 1 {
				t.Fatalf("len(glyphs): got: %d, want: 1", len(glyphs))
			}
			if got := glyphs[0].Colored; got != tc.colored {
				t.Errorf("Colored: got: %t, want: %t", got, tc.colored)
			}
			lazyGlyphs := text.AppendLazyGlyphs(nil, str, face, nil)
			if len(lazyGlyphs) != 1 {
				t.Fatalf("len(lazyGlyphs): got: %d, want: 1", len(lazyGlyphs))
			}
			if got := lazyGlyphs[0].Colored(); got != tc.colored {
				t.Errorf("Colored(): got: %t, want: %t", got, tc.colored)
			}

			dst := ebiten.NewImage(size*2, size*2)
			defer dst.Deallocate()
			op := &text.DrawOptions{}
			op.ColorScale.ScaleWithColor(color.RGBA{G: 0xff, A: 0xff})
			text.Draw(dst, str, face, op)
			m := face.Metrics()
			if got := dst.At(size/2, int(m.HAscent)-size/2); got != tc.want {
				t.Errorf("Draw with a green color scale: got: %v, want: %v", got, tc.want)
			}
		})
	}
}

// colorDiff returns the maximum difference among the RGBA components.
func colorDiff(a, b color.RGBA) int {
	diff := func(x, y uint8) int {
		if x > y {
			return int(x - y)
		}
		return int(y - x)
	}
	return max(diff(a.R, b.R), diff(a.G, b.G), diff(a.B, b.B), diff(a.A, b.A))
}
