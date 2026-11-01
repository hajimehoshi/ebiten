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

package svgrender_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg/svgrender"
)

func TestFilteredImageAverage(t *testing.T) {
	// Odd areas exercise integer normalization, including near the source limit.
	for _, size := range []image.Point{{X: 7, Y: 5}, {X: 2053, Y: 2041}} {
		parent := image.NewRGBA(image.Rect(0, 0, size.X+2, size.Y+2))
		source := parent.SubImage(image.Rect(1, 1, size.X+1, size.Y+1)).(*image.RGBA)
		var sum [4]uint64
		for y := range size.Y {
			for x := range size.X {
				a := 128 + (x+y)%128
				c := color.RGBA{R: byte((3*x + y) % (a + 1)), G: byte((x + 7*y) % (a + 1)), B: byte((x ^ y) % (a + 1)), A: byte(a)}
				source.SetRGBA(x+1, y+1, c)
				sum[0] += uint64(c.R)
				sum[1] += uint64(c.G)
				sum[2] += uint64(c.B)
				sum[3] += uint64(c.A)
			}
		}
		count := uint64(size.X * size.Y)
		want := color.RGBA{R: byte((sum[0] + count/2) / count), G: byte((sum[1] + count/2) / count), B: byte((sum[2] + count/2) / count), A: byte((sum[3] + count/2) / count)}
		got := svgrender.FilteredImage(source, 1, 1).RGBAAt(0, 0)
		if got != want {
			t.Errorf("%v source: got %v, want exact average %v", size, got, want)
		}
	}
}
