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

package vector

import (
	"image"
	"testing"
)

func TestAtlasEmptyPaths(t *testing.T) {
	for _, antialias := range []bool{false, true} {
		a := &atlas{}
		paths := []*Path{{}, {}}
		bounds := []image.Rectangle{
			image.Rect(0, 0, 16, 16),
			image.Rect(0, 0, 16, 16),
		}
		a.setPaths(image.Rect(0, 0, 16, 16), paths, bounds, antialias)

		for i, img := range a.atlasImages {
			if img != nil {
				t.Errorf("antialias: %v: atlasImages[%d] must be nil when all paths are empty", antialias, i)
			}
		}
		for i := range paths {
			for j := range 2 {
				if got := a.stencilBufferImageAt(i, antialias, j); got != nil {
					t.Errorf("antialias: %v: stencilBufferImageAt(%d, %v, %d): got: %v, want: nil", antialias, i, antialias, j, got)
				}
			}
		}
	}
}

func TestAtlasEmptyPathsAfterNonEmptyPaths(t *testing.T) {
	a := &atlas{}

	var p Path
	p.MoveTo(0, 0)
	p.LineTo(16, 0)
	p.LineTo(16, 16)
	p.LineTo(0, 16)
	p.Close()
	a.setPaths(image.Rect(0, 0, 16, 16), []*Path{&p}, []image.Rectangle{image.Rect(0, 0, 16, 16)}, false)
	if len(a.atlasImages) == 0 {
		t.Fatal("atlasImages must not be empty for non-empty paths")
	}
	if a.atlasImages[0] == nil {
		t.Fatal("atlasImages[0] must not be nil for non-empty paths")
	}

	a.setPaths(image.Rect(0, 0, 16, 16), []*Path{{}}, []image.Rectangle{image.Rect(0, 0, 16, 16)}, false)
	for i, img := range a.atlasImages {
		if img != nil {
			t.Errorf("atlasImages[%d] must be nil when all paths become empty", i)
		}
	}
	if got := a.stencilBufferImageAt(0, false, 0); got != nil {
		t.Errorf("stencilBufferImageAt(0, false, 0): got: %v, want: nil", got)
	}
}
