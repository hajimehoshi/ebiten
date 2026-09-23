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
	"runtime"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
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
