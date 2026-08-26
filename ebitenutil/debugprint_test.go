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

package ebitenutil_test

import (
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	itesting "github.com/hajimehoshi/ebiten/v2/internal/testing"
)

func TestMain(m *testing.M) {
	itesting.MainWithRunLoop(m)
}

// TestDebugPrintSkipsOutOfRange checks that DebugPrint, whose docstring says the supported rune
// range is U+0000..U+00FF, does not render runes outside that range. Without the skip, such
// runes silently fall through to a modular index lookup that creates a SubImage entry in an
// unbounded cache map.
func TestDebugPrintSkipsOutOfRange(t *testing.T) {
	for _, tc := range []struct {
		name string
		r    rune
		want bool
	}{
		{"in-range A", 'A', true},
		{"out-of-range U+3042", '\u3042', false},
	} {
		dst := ebiten.NewImage(16, 32)
		dst.Fill(color.RGBA{R: 0xff, A: 0xff})
		ebitenutil.DebugPrintAt(dst, string([]rune{tc.r}), 0, 0)
		// A non-red pixel anywhere in the first glyph column means something was rendered.
		nonRed := 0
		for y := 0; y < 32; y++ {
			for x := 0; x < 16; x++ {
				if r16, _, _, _ := dst.At(x, y).RGBA(); r16 != 0xffff {
					nonRed++
				}
			}
		}
		wantNonRed := tc.want
		var got string
		switch {
		case nonRed > 0:
			got = "rendered"
		default:
			got = "skipped"
		}
		if (nonRed > 0) != wantNonRed {
			t.Errorf("%s: %s, but the doc says the supported range is U+0000..U+00FF (non-red pixel count: %d)", tc.name, got, nonRed)
		}
		t.Logf("%-25s (%U %q): %s (%d non-red pixels)", tc.name, tc.r, tc.r, got, nonRed)
		dst.Deallocate()
	}
}
