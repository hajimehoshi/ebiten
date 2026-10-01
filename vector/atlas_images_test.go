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
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"testing"
)

func TestAtlasReleasesUnusedImages(t *testing.T) {
	var a atlas
	a.atlasImages = []*ebiten.Image{
		ebiten.NewImage(16, 16),
		ebiten.NewImage(16, 16),
		ebiten.NewImage(16, 16),
	}
	old := a.atlasImages
	for _, img := range old {
		defer img.Deallocate()
	}
	var p Path
	p.MoveTo(0, 0)
	p.LineTo(16, 0)
	p.LineTo(16, 16)
	p.LineTo(0, 16)
	p.Close()
	bounds := image.Rect(0, 0, 16, 16)
	a.setPaths(bounds, []*Path{&p}, []image.Rectangle{bounds}, false)
	if len(a.atlasImages) != 1 {
		t.Fatalf("got %d images, want 1", len(a.atlasImages))
	}
	if a.atlasImages[0] != old[0] {
		t.Error("the active image was not reused")
	}
	for i := 1; i < len(old); i++ {
		if old[i] != nil {
			t.Errorf("unused image %d is still referenced", i)
		}
	}
}
