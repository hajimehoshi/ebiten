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

package vmprotocol

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2/internal/graphics"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

// DrawOp identifies a recorded graphics command.
type DrawOp int

const (
	DrawOpInitialize DrawOp = iota
	DrawOpBegin
	DrawOpEnd
	DrawOpSetTransparent
	DrawOpSetVertices
	DrawOpNewImage
	DrawOpNewScreenFramebufferImage
	DrawOpNewShader
	DrawOpDrawTriangles
	DrawOpSetVsyncEnabled
	DrawOpWritePixels
	DrawOpReadPixels
	DrawOpDisposeImage
	DrawOpDisposeShader
)

func (o DrawOp) String() string {
	switch o {
	case DrawOpInitialize:
		return "Initialize"
	case DrawOpBegin:
		return "Begin"
	case DrawOpEnd:
		return "End"
	case DrawOpSetTransparent:
		return "SetTransparent"
	case DrawOpSetVertices:
		return "SetVertices"
	case DrawOpNewImage:
		return "NewImage"
	case DrawOpNewScreenFramebufferImage:
		return "NewScreenFramebufferImage"
	case DrawOpNewShader:
		return "NewShader"
	case DrawOpDrawTriangles:
		return "DrawTriangles"
	case DrawOpSetVsyncEnabled:
		return "SetVsyncEnabled"
	case DrawOpWritePixels:
		return "WritePixels"
	case DrawOpReadPixels:
		return "ReadPixels"
	case DrawOpDisposeImage:
		return "DisposeImage"
	case DrawOpDisposeShader:
		return "DisposeShader"
	default:
		return fmt.Sprintf("DrawOp(%d)", int(o))
	}
}

// DrawCommand is a single recorded graphics command: the guest-side remote driver produces these, they
// cross the wire, and the host replays them.
//
// Only the fields relevant to Op are populated; the rest stay at their zero values.
type DrawCommand struct {
	Op DrawOp

	// Present is set by End.
	Present bool

	// Transparent is set by SetTransparent.
	Transparent bool

	// VsyncEnabled is set by SetVsyncEnabled.
	VsyncEnabled bool

	// Vertices and Indices are set by SetVertices. They hold copies of the data, as the caller
	// reuses its buffers across calls.
	Vertices []float32
	Indices  []uint32

	// ImageID/Width/Height/Screen are set by NewImage and NewScreenFramebufferImage.
	// ImageID is also set by WritePixels, ReadPixels, and DisposeImage.
	ImageID graphicsdriver.ImageID
	Width   int
	Height  int
	Screen  bool

	// ShaderID is set by NewShader, DrawTriangles, and DisposeShader.
	ShaderID graphicsdriver.ShaderID

	// ShaderSource is the Kage source a shader was compiled from, set by NewShader. The host
	// recompiles it to recreate the shader (the compiled IR is not forwarded).
	ShaderSource []byte

	// Dst/Srcs/DstRegions/IndexOffset/Blend/Uniforms are set by DrawTriangles. Uniforms holds a copy.
	Dst         graphicsdriver.ImageID
	Srcs        [graphics.ShaderSrcImageCount]graphicsdriver.ImageID
	DstRegions  []graphicsdriver.DstRegion
	IndexOffset int
	Blend       graphicsdriver.Blend
	Uniforms    []uint32

	// Regions is set by WritePixels and ReadPixels. Pixels holds a copy of the pixel bytes for each
	// region of a WritePixels.
	Regions []image.Rectangle
	Pixels  [][]byte
}
