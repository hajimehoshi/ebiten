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
	"errors"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type Region struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

const (
	InvalidImageID  = 0
	InvalidShaderID = 0
)

type ColorM interface {
	IsIdentity() bool
	At(i, j int) float32
	Elements(body *[16]float32, translate *[4]float32)
}

type Graphics interface {
	Initialize() error
	Begin() error
	End(present bool) error
	SetTransparent(transparent bool)
	SetVertices(vertices []float32, indices []uint16) error
	NewImage(width, height int) (Image, error)
	NewScreenFramebufferImage(width, height int) (Image, error)
	SetVsyncEnabled(enabled bool)
	SetFullscreen(fullscreen bool)
	FramebufferYDirection() YDirection
	NeedsRestoring() bool
	NeedsClearingScreen() bool
	IsGL() bool
	IsDirectX() bool
	MaxImageSize() int

	NewShader(program *shaderir.Program) (Shader, error)

	// DrawTriangles draws an image onto another image with the given parameters.
	DrawTriangles(dst ImageID, srcs [graphics.ShaderImageCount]ImageID, offsets [graphics.ShaderImageCount - 1][2]float32, shader ShaderID, indexLen int, indexOffset int, mode CompositeMode, colorM ColorM, filter Filter, address Address, dstRegion, srcRegion Region, uniforms [][]float32, evenOdd bool) error
}

// GraphicsNotReady represents that the graphics driver is not ready for recovering from the context lost.
var GraphicsNotReady = errors.New("graphics not ready")

type Image interface {
	ID() ImageID
	Dispose()
	IsInvalidated() bool
	ReadPixels(buf []byte) error
	WritePixels(args []*WritePixelsArgs) error
}

type ImageID int

type WritePixelsArgs struct {
	Pixels []byte
	X      int
	Y      int
	Width  int
	Height int
}

type YDirection int

const (
	Upward YDirection = iota
	Downward
)

type Shader interface {
	ID() ShaderID
	Dispose()
}

type ShaderID int
