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

package driver

import (
	"errors"

	"github.com/hajimehoshi/ebiten/internal/affine"
	"github.com/hajimehoshi/ebiten/internal/graphics"
	"github.com/hajimehoshi/ebiten/internal/shaderir"
	"github.com/hajimehoshi/ebiten/internal/thread"
)

type Region struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

type Graphics interface {
	SetThread(thread *thread.Thread)
	Begin()
	End()
	SetTransparent(transparent bool)
	SetVertices(vertices []float32, indices []uint16)
	NewImage(width, height int) (Image, error)
	NewScreenFramebufferImage(width, height int) (Image, error)
	Reset() error
	SetVsyncEnabled(enabled bool)
	FramebufferYDirection() YDirection
	NeedsRestoring() bool
	IsGL() bool
	HasHighPrecisionFloat() bool
	MaxImageSize() int
	InvalidImageID() ImageID

	NewShader(program *shaderir.Program) (Shader, error)

	// Draw draws an image onto another image.
	//
	// TODO: Merge this into DrawShader.
	Draw(dst, src ImageID, indexLen int, indexOffset int, mode CompositeMode, colorM *affine.ColorM, filter Filter, address Address, sourceRegion Region) error

	// DrawShader draws the shader.
	//
	// uniforms represents a colletion of uniform variables. The values must be one of these types:
	//
	//   * float32
	//   * []float32
	DrawShader(dst ImageID, srcs [graphics.ShaderImageNum]ImageID, offsets [graphics.ShaderImageNum - 1][2]float32, shader ShaderID, indexLen int, indexOffset int, sourceRegion Region, mode CompositeMode, uniforms []interface{}) error
}

// GraphicsNotReady represents that the graphics driver is not ready for recovering from the context lost.
var GraphicsNotReady = errors.New("graphics not ready")

type Image interface {
	ID() ImageID
	Dispose()
	IsInvalidated() bool
	Pixels() ([]byte, error)
	ReplacePixels(args []*ReplacePixelsArgs)
}

type ImageID int

type ReplacePixelsArgs struct {
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
