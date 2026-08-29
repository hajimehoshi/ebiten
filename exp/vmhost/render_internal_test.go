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

package vmhost

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/vmprotocol"
)

func TestRenderDrawTrianglesIndexOutOfRange(t *testing.T) {
	renderer := newFrameRenderer()
	renderer.images[1] = &hostImage{}
	renderer.shaders[1] = &hostShader{}
	renderer.vertices = make([]float32, 3*graphics.VertexFloatCount)
	renderer.indices = []uint32{0, 1, 2}

	cmd := vmprotocol.GraphicsCommand{
		Kind:        vmprotocol.GraphicsCommandKindDrawTriangles,
		Dst:         1,
		ShaderID:    1,
		IndexOffset: 2,
		DstRegions: []graphicsdriver.DstRegion{
			{Region: image.Rect(0, 0, 1, 1), IndexCount: 2},
		},
		Uniforms: make([]uint32, graphics.PreservedUniformDwordCount),
	}
	if err := renderer.renderOne(cmd); err == nil {
		t.Errorf("DrawTriangles with an out-of-range index must return an error")
	}

	cmd.IndexOffset = 0
	cmd.DstRegions = []graphicsdriver.DstRegion{
		{Region: image.Rect(0, 0, 1, 1), IndexCount: 1},
	}
	renderer.indices = []uint32{3}
	if err := renderer.renderOne(cmd); err == nil {
		t.Errorf("DrawTriangles with an out-of-range vertex must return an error")
	}
}

func TestRenderWritePixelsMismatchedRegions(t *testing.T) {
	renderer := newFrameRenderer()
	renderer.images[1] = &hostImage{}

	cmd := vmprotocol.GraphicsCommand{
		Kind:    vmprotocol.GraphicsCommandKindWritePixels,
		ImageID: 1,
		Regions: []image.Rectangle{
			image.Rect(0, 0, 1, 1),
			image.Rect(0, 0, 1, 1),
		},
		Pixels: [][]byte{
			make([]byte, 4),
		},
	}
	if err := renderer.renderOne(cmd); err == nil {
		t.Errorf("WritePixels with mismatched regions and pixels must return an error")
	}
}
