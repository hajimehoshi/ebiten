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

// GraphicsCommandKind identifies a recorded graphics command.
type GraphicsCommandKind int

const (
	GraphicsCommandKindInitialize GraphicsCommandKind = iota
	GraphicsCommandKindBegin
	GraphicsCommandKindEnd
	GraphicsCommandKindSetTransparent
	GraphicsCommandKindSetVertices
	GraphicsCommandKindNewImage
	GraphicsCommandKindNewScreenFramebufferImage
	GraphicsCommandKindNewShader
	GraphicsCommandKindDrawTriangles
	GraphicsCommandKindSetVsyncEnabled
	GraphicsCommandKindWritePixels
	GraphicsCommandKindReadPixels
	GraphicsCommandKindDisposeImage
	GraphicsCommandKindDisposeShader
)

func (k GraphicsCommandKind) String() string {
	switch k {
	case GraphicsCommandKindInitialize:
		return "Initialize"
	case GraphicsCommandKindBegin:
		return "Begin"
	case GraphicsCommandKindEnd:
		return "End"
	case GraphicsCommandKindSetTransparent:
		return "SetTransparent"
	case GraphicsCommandKindSetVertices:
		return "SetVertices"
	case GraphicsCommandKindNewImage:
		return "NewImage"
	case GraphicsCommandKindNewScreenFramebufferImage:
		return "NewScreenFramebufferImage"
	case GraphicsCommandKindNewShader:
		return "NewShader"
	case GraphicsCommandKindDrawTriangles:
		return "DrawTriangles"
	case GraphicsCommandKindSetVsyncEnabled:
		return "SetVsyncEnabled"
	case GraphicsCommandKindWritePixels:
		return "WritePixels"
	case GraphicsCommandKindReadPixels:
		return "ReadPixels"
	case GraphicsCommandKindDisposeImage:
		return "DisposeImage"
	case GraphicsCommandKindDisposeShader:
		return "DisposeShader"
	default:
		return fmt.Sprintf("GraphicsCommandKind(%d)", int(k))
	}
}

// GraphicsCommand is a single recorded graphics command: the guest-side remote driver produces these,
// they cross the wire, and the host replays them.
//
// Only the fields relevant to Kind are populated; the rest stay at their zero values.
type GraphicsCommand struct {
	Kind GraphicsCommandKind

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

	// ShaderSource is the pixel-unit Kage source a shader was compiled from, set by NewShader.
	// The host recompiles it to recreate the shader (the compiled IR is not forwarded).
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
