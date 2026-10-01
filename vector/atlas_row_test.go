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

func TestAtlasRowHeight(t *testing.T) {
	for _, antialias := range []bool{false, true} {
		var a atlas
		paths := make([]*Path, 5)
		bounds := make([]image.Rectangle, len(paths))
		for i := range paths {
			height := 16
			if i == 0 {
				height = 1024
			}
			p := &Path{}
			p.MoveTo(0, 0)
			p.LineTo(2048, 0)
			p.LineTo(2048, float32(height))
			p.LineTo(0, float32(height))
			p.Close()
			paths[i] = p
			bounds[i] = image.Rect(0, 0, 2048, height)
		}
		a.setPaths(image.Rect(0, 0, 2048, 2048), paths, bounds, antialias)
		for _, img := range a.atlasImages {
			defer img.Deallocate()
		}
		if got := len(a.atlasImages); got != 1 {
			t.Errorf("antialias=%v: atlas image count: got %d, want 1", antialias, got)
		}
		for i := 1; i < len(paths); i++ {
			got := a.atlasRegions[a.pathIndexToAtlasRegionIndex[i]].imageBounds.Min.Y
			want := 1024 + (i-1)*16
			if got != want {
				t.Errorf("antialias=%v: path %d Y: got %d, want %d", antialias, i, got, want)
			}
		}
	}
}
