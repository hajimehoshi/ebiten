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

	"github.com/hajimehoshi/ebiten/v2/internal/color"
	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type DstRegion struct {
	Region     image.Rectangle
	IndexCount int
}

const (
	InvalidImageID  = 0
	InvalidShaderID = 0
)

// FlushMode specifies whether a command batch completes or presents a frame.
type FlushMode int

const (
	// FlushModeIntermediate submits commands without completing the frame.
	FlushModeIntermediate FlushMode = iota
	// FlushModeEndFrame completes the frame without presenting it.
	FlushModeEndFrame
	// FlushModePresent completes and presents the frame.
	FlushModePresent
)

type Graphics interface {
	Initialize() error
	ColorSpace() color.ColorSpace
	Begin() error
	// End ends a command batch with the given flush mode.
	End(mode FlushMode) error
	SetTransparent(transparent bool)
	SetVertices(vertices []float32, indices []uint32) error
	NewImage(width, height int) (Image, error)
	NewScreenFramebufferImage(width, height int) (Image, error)
	SetVsyncEnabled(enabled bool)
	NeedsClearingScreen() bool
	MaxImageSize() int

	NewShader(program *shaderir.Program) (Shader, error)

	// DrawTriangles draws an image onto another image with the given parameters.
	DrawTriangles(dst ImageID, srcs [graphics.ShaderSrcImageCount]ImageID, shader ShaderID, dstRegions []DstRegion, indexOffset int, blend Blend, uniforms []uint32) error
}

type Resetter interface {
	Reset() error
}

type Image interface {
	ID() ImageID
	Dispose()
	ReadPixels(args []PixelsArgs) error
	WritePixels(args []PixelsArgs) error
}

// PixelsReadback represents a pixel read-back that has been started by AsyncPixelsReader and whose
// pixels are not available yet.
type PixelsReadback interface {
	// Poll reports whether the read pixels are available. Poll must not block.
	//
	// When Poll reports true, the read pixels are available and Copy must be called to obtain them.
	// When Poll reports an error, the read pixels are never available.
	Poll() (done bool, err error)

	// Copy copies the read pixels to args. Copy must be called only after Poll reported done, and
	// must be called at most once.
	Copy(args []PixelsArgs) error

	// Discard releases the resources for the read-back without copying the pixels. The contents of
	// the arguments of the read-back are left unspecified.
	//
	// Discard must be called exactly once, either after Copy or instead of it.
	Discard()
}

// AsyncPixelsReader is an optional interface for a graphics driver image that can start a pixel
// read-back without waiting for the GPU to finish.
//
// A graphics driver that doesn't implement this interface reads pixels synchronously, which is
// correct but blocks the render thread until the GPU finishes.
type AsyncPixelsReader interface {
	// ReadPixelsAsync starts reading pixels and returns without waiting for the GPU.
	//
	// The read pixels include the drawing commands preceding this call and exclude the following
	// ones, as the reads are recorded at the current position of the command stream.
	//
	// ReadPixelsAsync must be called on the render thread.
	//
	// ReadPixelsAsync must retain the resources the read-back needs until the returned
	// PixelsReadback is discarded. In particular, the image may be disposed right after
	// ReadPixelsAsync returns.
	ReadPixelsAsync(args []PixelsArgs) (PixelsReadback, error)
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
