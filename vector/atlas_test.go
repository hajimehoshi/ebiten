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
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func TestAtlasReleasesUnusedImages(t *testing.T) {
	var a vector.Atlas
	images := []*ebiten.Image{
		ebiten.NewImage(16, 16),
		ebiten.NewImage(16, 16),
		ebiten.NewImage(16, 16),
	}
	for _, img := range images {
		defer img.Deallocate()
	}
	a.SetAtlasImages(images)

	var p vector.Path
	p.MoveTo(0, 0)
	p.LineTo(16, 0)
	p.LineTo(16, 16)
	p.LineTo(0, 16)
	p.Close()
	bounds := image.Rect(0, 0, 16, 16)
	a.SetPaths(bounds, []*vector.Path{&p}, []image.Rectangle{bounds}, false)

	if got, want := len(a.AtlasImages()), 1; got != want {
		t.Fatalf("got: %d, want: %d", got, want)
	}
	if a.AtlasImages()[0] != images[0] {
		t.Error("the active image was not reused")
	}
	// The removed entries must not retain the unused images.
	for i := 1; i < len(images); i++ {
		if images[i] != nil {
			t.Errorf("unused image %d is still referenced", i)
		}
	}
}
