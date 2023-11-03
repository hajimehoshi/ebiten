// Copyright 2023 The Ebitengine Authors
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

//go:build playstation5

package playstation5

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Graphics struct {
}

func NewGraphics() (*Graphics, error) {
	return &Graphics{}, nil
}

func (g *Graphics) Initialize() error {
	return nil
}

func (g *Graphics) Begin() error {
	return nil
}

func (g *Graphics) End(present bool) error {
	return nil
}

func (g *Graphics) SetTransparent(transparent bool) {
}

func (g *Graphics) SetVertices(vertices []float32, indices []uint32) error {
	return nil
}

func (g *Graphics) NewImage(width, height int) (graphicsdriver.Image, error) {
	return nil, nil
}

func (g *Graphics) NewScreenFramebufferImage(width, height int) (graphicsdriver.Image, error) {
	return nil, nil
}

func (g *Graphics) SetVsyncEnabled(enabled bool) {
}

func (g *Graphics) NeedsRestoring() bool {
	return false
}

func (g *Graphics) NeedsClearingScreen() bool {
	return false
}

func (g *Graphics) MaxImageSize() int {
	return 4096 // TODO
}

func (g *Graphics) NewShader(program *shaderir.Program) (graphicsdriver.Shader, error) {
	return nil, nil
}

func (g *Graphics) DrawTriangles(dst graphicsdriver.ImageID, srcs [graphics.ShaderImageCount]graphicsdriver.ImageID, shader graphicsdriver.ShaderID, dstRegions []graphicsdriver.DstRegion, indexOffset int, blend graphicsdriver.Blend, uniforms []uint32, evenOdd bool) error {
	return nil
}
