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
	"slices"
	"strings"
	"testing"

	"github.com/go-text/typesetting/font/opentype"

	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// bitmapTablesRemoved returns the font data with the bitmap strike tables
// removed, so that the same glyphs render from their outlines.
func bitmapTablesRemoved(t *testing.T, data []byte) []byte {
	t.Helper()

	ld, err := opentype.NewLoader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	strikes := map[string]bool{
		"EBDT": true,
		"EBLC": true,
		"EBSC": true,
		"CBDT": true,
		"CBLC": true,
		"sbix": true,
	}
	var tables []opentype.Table
	for _, tag := range ld.Tables() {
		if strikes[tag.String()] {
			continue
		}
		content, err := ld.RawTable(tag)
		if err != nil {
			t.Fatal(err)
		}
		tables = append(tables, opentype.Table{
			Tag:     tag,
			Content: content,
		})
	}
	return opentype.WriteTTF(tables)
}

// glyphInk returns the visible part of a glyph image as rows of '#' for a
// pixel with any coverage and '.' for a transparent one.
func glyphInk(t *testing.T, str string, face *text.GoTextFace) string {
	t.Helper()

	glyphs := text.AppendGlyphs(nil, str, face, nil)
	if len(glyphs) != 1 {
		t.Fatalf("got %d glyphs for %q, want 1", len(glyphs), str)
	}
	img := glyphs[0].Image
	if img == nil {
		t.Fatalf("%q has no glyph image", str)
	}

	b := img.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	img.ReadPixels(pix)
	covered := func(x, y int) bool {
		return pix[4*(y*b.Dx()+x)+3] != 0
	}

	minX, minY, maxX, maxY := b.Dx(), b.Dy(), -1, -1
	for y := range b.Dy() {
		for x := range b.Dx() {
			if !covered(x, y) {
				continue
			}
			minX = min(minX, x)
			minY = min(minY, y)
			maxX = max(maxX, x)
			maxY = max(maxY, y)
		}
	}
	if maxX < 0 {
		t.Fatalf("%q has no visible pixel", str)
	}

	var sb strings.Builder
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			if covered(x, y) {
				sb.WriteByte('#')
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// rotateClockwise returns the rows of an image rotated 90 degrees clockwise.
func rotateClockwise(rows string) string {
	lines := strings.Split(strings.TrimSuffix(rows, "\n"), "\n")
	var sb strings.Builder
	for x := range len(lines[0]) {
		for _, line := range slices.Backward(lines) {
			sb.WriteByte(line[x])
		}
		sb.WriteByte('\n')
	}
	return sb.String()
}

// TestBitmapFontSideways tests that a glyph of a bitmap font is rotated in a
// sideways run. A bitmap strike cannot be rotated, so the glyph must fall
// back to its outline rasterized with the sideways rotation applied, and not
// to the unrotated outline drawn into the already rotated bounds.
//
// The expected image is built independently of that rotation: the outline of
// the same glyph is rendered horizontally by the same font with its bitmap
// tables removed, and rotated here.
func TestBitmapFontSideways(t *testing.T) {
	// Terminus has a bitmap strike at 16ppem, so a horizontal run renders
	// that strike and a sideways run needs the outline fallback.
	const (
		str  = "F"
		size = 16
	)

	bitmapSource, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.TerminusTTF_ttf))
	if err != nil {
		t.Fatal(err)
	}
	outlineSource, err := text.NewGoTextFaceSource(bytes.NewReader(bitmapTablesRemoved(t, fonts.TerminusTTF_ttf)))
	if err != nil {
		t.Fatal(err)
	}

	// Both runs rasterize the same segments, and the glyph is placed on a
	// pixel in both of them, so no rasterization tolerance is needed.
	want := rotateClockwise(glyphInk(t, str, &text.GoTextFace{
		Source: outlineSource,
		Size:   size,
	}))
	got := glyphInk(t, str, &text.GoTextFace{
		Source:    bitmapSource,
		Size:      size,
		Direction: text.DirectionTopToBottomAndLeftToRight,
	})
	if got != want {
		t.Errorf("the sideways glyph of a bitmap font is not the rotated outline:\ngot:\n%swant:\n%s", got, want)
	}
}
