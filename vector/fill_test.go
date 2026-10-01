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
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/internal/imagebridge"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// waitForEmptyFillPathsStates waits until the pending fill states of collected images are released.
func waitForEmptyFillPathsStates(t *testing.T) bool {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for {
		runtime.GC()
		if vector.FillPathsStateCount() == 0 && vector.CallbackTokenCount() == 0 {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(time.Millisecond)
	}
}

func TestFillPathDoesNotRetainDestination(t *testing.T) {
	// Images that other tests left behind might still be registered. Wait for them to be released.
	if !waitForEmptyFillPathsStates(t) {
		t.Skip("the states of the other tests are not released")
	}

	func() {
		dst := ebiten.NewImage(16, 16)
		var path vector.Path
		path.MoveTo(0, 0)
		path.LineTo(16, 0)
		path.LineTo(16, 16)
		path.Close()
		op := &vector.DrawPathOptions{}
		op.ColorScale.ScaleWithColor(color.White)
		vector.FillPath(dst, &path, nil, op)

		if got, want := vector.FillPathsStateCount(), 1; got != want {
			t.Errorf("vector.FillPathsStateCount(): got: %d, want: %d", got, want)
		}
		if got, want := vector.CallbackTokenCount(), 1; got != want {
			t.Errorf("vector.CallbackTokenCount(): got: %d, want: %d", got, want)
		}

		runtime.KeepAlive(dst)
	}()

	// The destination image is no longer used. The states must be released after the image is collected.
	if !waitForEmptyFillPathsStates(t) {
		t.Errorf("the states must be released after the destination image is collected: fill paths states: %d, callback tokens: %d", vector.FillPathsStateCount(), vector.CallbackTokenCount())
	}
}

func TestFillPathAfterLargePath(t *testing.T) {
	const size = 256

	fillRect := func(dst *ebiten.Image, r image.Rectangle, antialias bool) {
		var path vector.Path
		path.MoveTo(float32(r.Min.X), float32(r.Min.Y))
		path.LineTo(float32(r.Max.X), float32(r.Min.Y))
		path.LineTo(float32(r.Max.X), float32(r.Max.Y))
		path.LineTo(float32(r.Min.X), float32(r.Max.Y))
		path.Close()
		op := &vector.DrawPathOptions{}
		op.AntiAlias = antialias
		op.ColorScale.ScaleWithColor(color.White)
		vector.FillPath(dst, &path, nil, op)
	}

	for _, antialias := range []bool{false, true} {
		t.Run(fmt.Sprintf("antialias=%t", antialias), func(t *testing.T) {
			dst := ebiten.NewImage(size, size)
			defer dst.Deallocate()

			// A path covering the whole destination makes the stencil atlas image large.
			fillRect(dst, image.Rect(0, 0, size, size), antialias)
			// Clear flushes the pending path.
			dst.Clear()

			// A smaller path in a later flush reuses the large atlas image.
			// Stale stencil data would show up around the path.
			// The atlas is shared, so the expected pixels are computed here.
			small := image.Rect(4, 4, 12, 12)
			fillRect(dst, small, antialias)
			got := make([]byte, 4*size*size)
			dst.ReadPixels(got)

			var mismatches int
			var firstX, firstY int
			for y := range size {
				for x := range size {
					var want byte
					if image.Pt(x, y).In(small) {
						want = 0xff
					}
					i := 4 * (y*size + x)
					if bytes.Equal(got[i:i+4], []byte{want, want, want, want}) {
						continue
					}
					if mismatches == 0 {
						firstX, firstY = x, y
					}
					mismatches++
				}
			}
			if mismatches > 0 {
				i := 4 * (firstY*size + firstX)
				t.Errorf("%d pixels differ; first at (%d, %d): got: %v, in the path: %t", mismatches, firstX, firstY, got[i:i+4], image.Pt(firstX, firstY).In(small))
			}
		})
	}
}

func TestFillPathZeroCoverage(t *testing.T) {
	for _, aa := range []bool{false, true} {
		for _, rule := range []vector.FillRule{vector.FillRuleNonZero, vector.FillRuleEvenOdd} {
			t.Run(fmt.Sprintf("aa=%t/rule=%d", aa, rule), func(t *testing.T) {
				dst := ebiten.NewImage(64, 64)
				defer dst.Deallocate()
				dst.Fill(color.White)
				var path vector.Path
				path.Arc(24.375, 24.375, 17, 0, 2*math.Pi, vector.Clockwise)
				path.Close()
				paint := color.RGBA{
					R: 70,
					G: 160,
					B: 240,
					A: 255,
				}
				op := &vector.DrawPathOptions{
					AntiAlias: aa,
					Blend:     ebiten.BlendCopy,
				}
				op.ColorScale.ScaleWithColor(paint)
				vector.FillPath(dst, &path, &vector.FillOptions{
					FillRule: rule,
				}, op)
				if r, g, b, a := dst.At(8, 8).RGBA(); r != 0xffff || g != 0xffff || b != 0xffff || a != 0xffff {
					t.Errorf("pixel outside circle: got %v, want white", dst.At(8, 8))
				}
				if got := dst.At(24, 24); got != paint {
					t.Errorf("pixel inside circle: got %v, want %v", got, paint)
				}
			})
		}
	}
}

func TestDrawEmptyPath(t *testing.T) {
	var empty, reset vector.Path
	reset.MoveTo(1, 1)
	reset.LineTo(7, 1)
	reset.LineTo(7, 7)
	reset.Close()
	reset.Reset()
	paths := []struct {
		name string
		path *vector.Path
	}{
		{
			name: "Nil",
		},
		{
			name: "ZeroValue",
			path: &empty,
		},
		{
			name: "Reset",
			path: &reset,
		},
	}

	drawings := []struct {
		name string
		draw func(*ebiten.Image, *vector.Path, *vector.DrawPathOptions)
	}{
		{
			name: "FillPath/NilOptions",
			draw: func(dst *ebiten.Image, path *vector.Path, op *vector.DrawPathOptions) {
				vector.FillPath(dst, path, nil, op)
			},
		},
		{
			name: "FillPath/EvenOdd",
			draw: func(dst *ebiten.Image, path *vector.Path, op *vector.DrawPathOptions) {
				vector.FillPath(dst, path, &vector.FillOptions{
					FillRule: vector.FillRuleEvenOdd,
				}, op)
			},
		},
		{
			name: "StrokePath/NilOptions",
			draw: func(dst *ebiten.Image, path *vector.Path, op *vector.DrawPathOptions) {
				vector.StrokePath(dst, path, nil, op)
			},
		},
		{
			name: "StrokePath/PositiveWidth",
			draw: func(dst *ebiten.Image, path *vector.Path, op *vector.DrawPathOptions) {
				vector.StrokePath(dst, path, &vector.StrokeOptions{
					Width: 2,
				}, op)
			},
		},
	}
	blends := []struct {
		name  string
		blend ebiten.Blend
	}{
		{
			name: "NilOptions",
		},
		{
			name: "Default",
		},
		{
			name:  "SourceOver",
			blend: ebiten.BlendSourceOver,
		},
		{
			name:  "Clear",
			blend: ebiten.BlendClear,
		},
		{
			name:  "Copy",
			blend: ebiten.BlendCopy,
		},
		{
			name:  "Destination",
			blend: ebiten.BlendDestination,
		},
		{
			name:  "DestinationOver",
			blend: ebiten.BlendDestinationOver,
		},
		{
			name:  "SourceIn",
			blend: ebiten.BlendSourceIn,
		},
		{
			name:  "DestinationIn",
			blend: ebiten.BlendDestinationIn,
		},
		{
			name:  "SourceOut",
			blend: ebiten.BlendSourceOut,
		},
		{
			name:  "DestinationOut",
			blend: ebiten.BlendDestinationOut,
		},
		{
			name:  "SourceAtop",
			blend: ebiten.BlendSourceAtop,
		},
		{
			name:  "DestinationAtop",
			blend: ebiten.BlendDestinationAtop,
		},
		{
			name:  "Xor",
			blend: ebiten.BlendXor,
		},
		{
			name:  "Lighter",
			blend: ebiten.BlendLighter,
		},
	}
	for _, path := range paths {
		for _, drawing := range drawings {
			for _, blend := range blends {
				for _, aa := range []bool{false, true} {
					if blend.name == "NilOptions" && aa {
						continue
					}
					for _, queued := range []bool{false, true} {
						t.Run(fmt.Sprintf("%s/%s/%s/aa=%t/queued=%t", path.name, drawing.name, blend.name, aa, queued), func(t *testing.T) {
							const size = 8
							dst := ebiten.NewImage(size, size)
							defer dst.Deallocate()
							background := color.RGBA{R: 32, G: 64, B: 96, A: 128}
							dst.Fill(background)
							if queued {
								vector.FillRect(dst, 2, 2, 4, 4, color.White, false)
								vector.StrokeLine(dst, 1, 1, 7, 1, 2, color.White, false)
							}

							var op *vector.DrawPathOptions
							if blend.name != "NilOptions" {
								op = &vector.DrawPathOptions{
									AntiAlias: aa,
									Blend:     blend.blend,
								}
							}
							// An empty path must leave queued drawings deferred until the destination is used.
							bridge := imagebridge.Get[*ebiten.Image]()
							var uses int
							token := bridge.AddUsage(dst, func(*ebiten.Image) {
								uses++
							})
							drawing.draw(dst, path.path, op)
							bridge.RemoveUsage(dst, token)
							if uses != 0 {
								t.Errorf("empty path used the destination %d times, want 0", uses)
							}

							got := make([]byte, 4*size*size)
							dst.ReadPixels(got)
							for y := range size {
								for x := range size {
									want := background
									if queued && ((2 <= x && x < 6 && 2 <= y && y < 6) || (1 <= x && x < 7 && y < 2)) {
										want = color.RGBA{R: 255, G: 255, B: 255, A: 255}
									}
									i := 4 * (y*size + x)
									if !bytes.Equal(got[i:i+4], []byte{want.R, want.G, want.B, want.A}) {
										t.Errorf("pixel (%d, %d): got %v, want %v", x, y, got[i:i+4], want)
									}
								}
							}
						})
					}
				}
			}
		}
	}
}
