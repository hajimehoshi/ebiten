// Copyright 2018 The Ebiten Authors
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

package restorable_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/internal/driver"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/graphicscommand"
	. "github.com/hajimehoshi/ebiten/internal/restorable"
	etesting "github.com/hajimehoshi/ebiten/internal/testing"
)

func TestShader(t *testing.T) {
	if !graphicscommand.IsShaderAvailable() {
		t.Skip("shader is not available on this environment")
	}

	img := newImageFromImage(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	defer img.Dispose()

	ir := etesting.ShaderProgramFill(0xff, 0, 0, 0xff)
	s := NewShader(&ir)
	is := graphics.QuadIndices()
	us := map[int]interface{}{
		0: []float32{1, 1},
	}
	img.DrawTriangles(nil, quadVertices(1, 1, 0, 0), is, nil, driver.CompositeModeCopy, driver.FilterNearest, driver.AddressClampToZero, s, us)

	if err := ResolveStaleImages(); err != nil {
		t.Fatal(err)
	}
	if err := RestoreIfNeeded(); err != nil {
		t.Fatal(err)
	}

	want := color.RGBA{0xff, 0, 0, 0xff}
	got := pixelsToColor(img.BasePixelsForTesting(), 0, 0)
	if !sameColors(got, want, 1) {
		t.Errorf("got %v, want %v", got, want)
	}
}
