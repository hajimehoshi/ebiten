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
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"testing"

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
		})
	}
}
