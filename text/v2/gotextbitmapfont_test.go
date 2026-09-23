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
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/examples/resources/fonts"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

// inkBounds returns the bounding rectangle of the non-transparent pixels of
// a glyph image, in the image's coordinates. It returns false when the image
// is nil or has no visible pixel.
func inkBounds(t *testing.T, img *ebiten.Image) (image.Rectangle, bool) {
	t.Helper()

	if img == nil {
		return image.Rectangle{}, false
	}
	b := img.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	img.ReadPixels(pix)

	minX, minY, maxX, maxY := b.Dx(), b.Dy(), -1, -1
	for j := range b.Dy() {
		for i := range b.Dx() {
			if pix[4*(j*b.Dx()+i)+3] == 0 {
				continue
			}
			minX = min(minX, i)
			minY = min(minY, j)
			maxX = max(maxX, i)
			maxY = max(maxY, j)
		}
	}
	if maxX < 0 {
		return image.Rectangle{}, false
	}
	return image.Rect(minX, minY, maxX+1, maxY+1), true
}

// TestBitmapFontSideways tests that a glyph of a bitmap font is rotated in a
// sideways run. A bitmap strike cannot be rotated, so the glyph must fall
// back to its outline rasterized with the sideways rotation applied, not to
// the unrotated outline drawn into the already rotated bounds.
func TestBitmapFontSideways(t *testing.T) {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(fonts.TerminusTTF_ttf))
	if err != nil {
		t.Fatal(err)
	}

	// Terminus has bitmap strikes at these ppem values, so every size below
	// renders the strike in a horizontal run.
	for _, size := range []float64{12, 14, 16, 18, 20, 22, 24, 28, 32} {
		t.Run(fmt.Sprintf("%v", size), func(t *testing.T) {
			const str = "F"

			horizontal := &text.GoTextFace{
				Source: source,
				Size:   size,
			}
			vertical := &text.GoTextFace{
				Source:    source,
				Size:      size,
				Direction: text.DirectionTopToBottomAndLeftToRight,
			}

			hgs := text.AppendGlyphs(nil, str, horizontal, nil)
			vgs := text.AppendGlyphs(nil, str, vertical, nil)
			if len(hgs) != 1 || len(vgs) != 1 {
				t.Fatalf("got %d horizontal and %d vertical glyphs, want 1 each", len(hgs), len(vgs))
			}

			hInk, ok := inkBounds(t, hgs[0].Image)
			if !ok {
				t.Fatal("the horizontal glyph has no visible pixel")
			}
			vInk, ok := inkBounds(t, vgs[0].Image)
			if !ok {
				t.Fatal("the vertical glyph has no visible pixel")
			}

			// The sideways ink must be the horizontal ink turned on its side.
			// A bitmap strike and the outline of the same glyph are hinted
			// differently, so a few pixels of difference are allowed.
			const maxDiff = 2
			d := func(x int) int {
				if x < 0 {
					return -x
				}
				return x
			}
			if diff := d(vInk.Dx() - hInk.Dy()); diff > maxDiff {
				t.Errorf("the vertical glyph width %d is not the horizontal glyph height %d rotated: the diff is %d", vInk.Dx(), hInk.Dy(), diff)
			}
			if diff := d(vInk.Dy() - hInk.Dx()); diff > maxDiff {
				t.Errorf("the vertical glyph height %d is not the horizontal glyph width %d rotated: the diff is %d", vInk.Dy(), hInk.Dx(), diff)
			}
			if vInk.Dx() <= vInk.Dy() {
				t.Errorf("the vertical glyph ink %v must be wider than high, which means the glyph is not rotated", vInk)
			}
		})
	}
}
