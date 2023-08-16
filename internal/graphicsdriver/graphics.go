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

package graphicsdriver

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Region struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

type DstRegion struct {
	Region     Region
	IndexCount int
}

const (
	InvalidImageID  = 0
	InvalidShaderID = 0
)

type Graphics interface {
	Initialize() error
	Begin() error
	End(present bool) error
	SetTransparent(transparent bool)
	SetVertices(vertices []float32, indices []uint16) error
	NewImage(width, height int) (Image, error)
	NewScreenFramebufferImage(width, height int) (Image, error)
	SetVsyncEnabled(enabled bool)
	NeedsRestoring() bool
	NeedsClearingScreen() bool
	IsGL() bool
	IsDirectX() bool
	MaxImageSize() int

	NewShader(program *shaderir.Program) (Shader, error)

	// DrawTriangles draws an image onto another image with the given parameters.
	DrawTriangles(dst ImageID, srcs [graphics.ShaderImageCount]ImageID, shader ShaderID, dstRegions []DstRegion, indexOffset int, blend Blend, uniforms []uint32, evenOdd bool) error
}

type Resetter interface {
	Reset() error
}

type Image interface {
	ID() ImageID
	Dispose()
	IsInvalidated() bool
	ReadPixels(args []PixelsArgs) error
	WritePixels(args []PixelsArgs) error
}

type ImageID int

type PixelsArgs struct {
	Pixels []byte
	Region image.Rectangle
}

type Shader interface {
	ID() ShaderID
	Dispose()
}

type ShaderID int
