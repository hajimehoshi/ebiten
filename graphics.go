// Copyright 2014 Hajime Hoshi
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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// Filter represents the type of texture filter to be used when an image is maginified or minified.
type Filter int

const (
	// FilterNearest represents nearest (crisp-edged) filter
	FilterNearest Filter = Filter(graphicsdriver.FilterNearest)

	// FilterLinear represents linear filter
	FilterLinear Filter = Filter(graphicsdriver.FilterLinear)
)

// CompositeMode represents Porter-Duff composition mode.
type CompositeMode int

// This name convention follows CSS compositing: https://drafts.fxtf.org/compositing-2/.
//
// In the comments,
// c_src, c_dst and c_out represent alpha-premultiplied RGB values of source, destination and output respectively. α_src and α_dst represent alpha values of source and destination respectively.
const (
	// Regular alpha blending
	// c_out = c_src + c_dst × (1 - α_src)
	CompositeModeSourceOver CompositeMode = CompositeMode(graphicsdriver.CompositeModeSourceOver)

	// c_out = 0
	CompositeModeClear CompositeMode = CompositeMode(graphicsdriver.CompositeModeClear)

	// c_out = c_src
	CompositeModeCopy CompositeMode = CompositeMode(graphicsdriver.CompositeModeCopy)

	// c_out = c_dst
	CompositeModeDestination CompositeMode = CompositeMode(graphicsdriver.CompositeModeDestination)

	// c_out = c_src × (1 - α_dst) + c_dst
	CompositeModeDestinationOver CompositeMode = CompositeMode(graphicsdriver.CompositeModeDestinationOver)

	// c_out = c_src × α_dst
	CompositeModeSourceIn CompositeMode = CompositeMode(graphicsdriver.CompositeModeSourceIn)

	// c_out = c_dst × α_src
	CompositeModeDestinationIn CompositeMode = CompositeMode(graphicsdriver.CompositeModeDestinationIn)

	// c_out = c_src × (1 - α_dst)
	CompositeModeSourceOut CompositeMode = CompositeMode(graphicsdriver.CompositeModeSourceOut)

	// c_out = c_dst × (1 - α_src)
	CompositeModeDestinationOut CompositeMode = CompositeMode(graphicsdriver.CompositeModeDestinationOut)

	// c_out = c_src × α_dst + c_dst × (1 - α_src)
	CompositeModeSourceAtop CompositeMode = CompositeMode(graphicsdriver.CompositeModeSourceAtop)

	// c_out = c_src × (1 - α_dst) + c_dst × α_src
	CompositeModeDestinationAtop CompositeMode = CompositeMode(graphicsdriver.CompositeModeDestinationAtop)

	// c_out = c_src × (1 - α_dst) + c_dst × (1 - α_src)
	CompositeModeXor CompositeMode = CompositeMode(graphicsdriver.CompositeModeXor)

	// Sum of source and destination (a.k.a. 'plus' or 'additive')
	// c_out = c_src + c_dst
	CompositeModeLighter CompositeMode = CompositeMode(graphicsdriver.CompositeModeLighter)

	// The product of source and destination (a.k.a 'multiply blend mode')
	// c_out = c_src * c_dst
	CompositeModeMultiply CompositeMode = CompositeMode(graphicsdriver.CompositeModeMultiply)
)

// GraphicsLibrary represets graphics libraries supported by the engine.
type GraphicsLibrary = ui.GraphicsLibrary

const (
	// GraphicsLibraryUnknown represents the state at which graphics library cannot be determined,
	// e.g. hasn't loaded yet or failed to initialize.
	GraphicsLibraryUnknown = ui.GraphicsLibraryUnknown

	// GraphicsLibraryOpenGL represents the graphics library OpenGL.
	GraphicsLibraryOpenGL = ui.GraphicsLibraryOpenGL

	// GraphicsLibraryDirectX represents the graphics library Microsoft DirectX.
	GraphicsLibraryDirectX = ui.GraphicsLibraryDirectX

	// GraphicsLibraryMetal represents the graphics library Apple's Metal.
	GraphicsLibraryMetal = ui.GraphicsLibraryMetal
)

// DebugInfo is a struct to store debug info about the graphics.
type DebugInfo struct {
	// GraphicsLibrary represents the graphics library currently in use.
	GraphicsLibrary GraphicsLibrary
}

// ReadDebugInfo writes debug info (e.g. current graphics library) into a provided struct.
func ReadDebugInfo(d *DebugInfo) {
	d.GraphicsLibrary = ui.GetGraphicsLibrary()
}
