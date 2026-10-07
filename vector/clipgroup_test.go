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

package vector_test

import (
	"fmt"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Resolve only after all clipping and isolated groups have been composed.
// Linear minification with mipmaps disabled does not average every sample.
func boxResolveSamples(pixels []byte, width, scale int) []byte {
	size := width / scale
	resolved := make([]byte, 4*size*size)
	for y := range size {
		for x := range size {
			for c := range 4 {
				var sum int
				for sy := range scale {
					for sx := range scale {
						sum += int(pixels[4*((y*scale+sy)*width+x*scale+sx)+c])
					}
				}
				resolved[4*(y*size+x)+c] = byte((sum + scale*scale/2) / (scale * scale))
			}
		}
	}
	return resolved
}

func TestClipNestedPointSampledGroups(t *testing.T) {
	for _, scale := range []int{1, 2, 4, 16} {
		for _, offset := range []float32{0, 0.375} {
			for _, overlap := range []bool{false, true} {
				for _, clipChildren := range []bool{false, true} {
					t.Run(fmt.Sprintf("scale=%d/offset=%g/overlap=%t/clipChildren=%t", scale, offset, overlap, clipChildren), func(t *testing.T) {
						const logicalSize = 48
						size := logicalSize * scale
						newImage := func() *ebiten.Image {
							img := ebiten.NewImage(size, size)
							t.Cleanup(img.Deallocate)
							return img
						}
						circle := func(x float32) *vector.Path {
							var path vector.Path
							path.Arc((x+offset)*float32(scale), (24+offset)*float32(scale), 17*float32(scale), 0, 2*math.Pi, vector.Clockwise)
							path.Close()
							return &path
						}
						first := circle(24)
						second := first
						if overlap {
							second = circle(32)
						}
						rectangle := clipRectangle(0, 0, (30+offset)*float32(scale), float32(size))
						innerClip := &vector.PathClip{
							Path: first,
						}
						outerClip := &vector.PathClip{
							Path: rectangle,
						}
						var clip vector.Clip
						if clipChildren {
							clip = intersectionClip(innerClip, intersectionClip(outerClip, innerClip))
						}
						inner, outer, got := newImage(), newImage(), newImage()
						red := &vector.DrawPathOptions{
							Clip: clip,
						}
						red.ColorScale.Scale(1, 0, 0, 1)
						blue := &vector.DrawPathOptions{
							Clip: clip,
						}
						blue.ColorScale.Scale(0, 0, 1, 1)
						vector.FillPath(inner, first, nil, red)
						vector.FillPath(inner, second, nil, blue)

						// Each group retains the point-sample grid, including its clip mask.
						mask := newImage()
						vector.FillPath(mask, first, nil, nil)
						maskOp := &ebiten.DrawImageOptions{
							Blend: ebiten.BlendDestinationIn,
						}
						inner.DrawImage(mask, maskOp)
						opacity := &ebiten.DrawImageOptions{}
						opacity.ColorScale.ScaleAlpha(0.5)
						outer.DrawImage(inner, opacity)
						mask.Clear()
						vector.FillPath(mask, rectangle, nil, nil)
						outer.DrawImage(mask, maskOp)
						got.DrawImage(outer, opacity)

						// Independent per-sample oracle: opaque blue replaces red where the
						// second child covers it. Two isolated half-opacity groups yield 1/4
						// alpha everywhere in the clipped union, including overlapping children.
						a, b, c := newImage(), newImage(), newImage()
						vector.FillPath(a, first, nil, nil)
						vector.FillPath(b, second, nil, nil)
						vector.FillPath(c, rectangle, nil, nil)
						ap, bp, cp := clipPixels(a), clipPixels(b), clipPixels(c)
						want := make([]byte, len(ap))
						for i := 0; i < len(want); i += 4 {
							if ap[i+3] == 0 || cp[i+3] == 0 {
								continue
							}
							want[i+3] = 64
							if bp[i+3] != 0 {
								want[i+2] = 64
							} else {
								want[i] = 64
							}
						}
						actual := clipPixels(got)
						check := func(label string, actual, expected []byte) {
							t.Helper()
							var maxError int
							for i, v := range actual {
								delta := int(v) - int(expected[i])
								if delta < 0 {
									delta = -delta
								}
								maxError = max(maxError, delta)
							}
							if maxError > 1 {
								t.Errorf("%s: maximum premultiplied RGBA error = %d, want <= 1", label, maxError)
							}
						}
						check("point samples", actual, want)
						check("box resolve", boxResolveSamples(actual, size, scale), boxResolveSamples(want, size, scale))
						center := 4 * ((24*scale)*size + 24*scale)
						if actual[center+3] < 63 || actual[center+2] < 63 {
							t.Errorf("overlap center = %v, want opaque blue at quarter group opacity", actual[center:center+4])
						}
					})
				}
			}
		}
	}
}
