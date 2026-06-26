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

package vmhost_test

import (
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/vmhost"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
)

// makePrefix builds a preserved-uniform prefix with source image 0's texture size and region set.
func makePrefix(texW, texH, ox, oy, sw, sh float32) []uint32 {
	u := make([]uint32, graphics.PreservedUniformDwordCount)
	u[2] = math.Float32bits(texW) // source 0 texture size
	u[3] = math.Float32bits(texH)
	u[graphics.SourceImageRegionOriginUniformDwordIndex] = math.Float32bits(ox)
	u[graphics.SourceImageRegionOriginUniformDwordIndex+1] = math.Float32bits(oy)
	u[graphics.SourceImageRegionSizeUniformDwordIndex] = math.Float32bits(sw)
	u[graphics.SourceImageRegionSizeUniformDwordIndex+1] = math.Float32bits(sh)
	return u
}

func TestSrcRegionFromUniforms(t *testing.T) {
	want := image.Rect(4, 8, 14, 20) // origin (4,8), size (10,12)

	t.Run("pixels", func(t *testing.T) {
		// Pixel-unit regions are stored in pixels; passing a zero texture size reads them directly.
		u := makePrefix(64, 32, 4, 8, 10, 12)
		got, ok := vmhost.SrcRegionFromUniforms(u, 0, 0, 0)
		if !ok {
			t.Fatal("ok = false, want true")
		}
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("texels (normalized by texture size)", func(t *testing.T) {
		// Same region, but stored normalized by the 64x32 texture; pass that size to scale it back.
		u := makePrefix(64, 32, 4.0/64, 8.0/32, 10.0/64, 12.0/32)
		got, ok := vmhost.SrcRegionFromUniforms(u, 0, 64, 32)
		if !ok {
			t.Fatal("ok = false, want true")
		}
		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("missing prefix", func(t *testing.T) {
		if _, ok := vmhost.SrcRegionFromUniforms([]uint32{0, 0}, 0, 0, 0); ok {
			t.Error("ok = true, want false for a too-short uniform slice")
		}
	})
}
