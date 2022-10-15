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
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/builtinshader"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// Filter represents the type of texture filter to be used when an image is maginified or minified.
type Filter int

const (
	// FilterNearest represents nearest (crisp-edged) filter
	FilterNearest Filter = Filter(builtinshader.FilterNearest)

	// FilterLinear represents linear filter
	FilterLinear Filter = Filter(builtinshader.FilterLinear)
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
	CompositeModeSourceOver CompositeMode = iota

	// c_out = 0
	CompositeModeClear

	// c_out = c_src
	CompositeModeCopy

	// c_out = c_dst
	CompositeModeDestination

	// c_out = c_src × (1 - α_dst) + c_dst
	CompositeModeDestinationOver

	// c_out = c_src × α_dst
	CompositeModeSourceIn

	// c_out = c_dst × α_src
	CompositeModeDestinationIn

	// c_out = c_src × (1 - α_dst)
	CompositeModeSourceOut

	// c_out = c_dst × (1 - α_src)
	CompositeModeDestinationOut

	// c_out = c_src × α_dst + c_dst × (1 - α_src)
	CompositeModeSourceAtop

	// c_out = c_src × (1 - α_dst) + c_dst × α_src
	CompositeModeDestinationAtop

	// c_out = c_src × (1 - α_dst) + c_dst × (1 - α_src)
	CompositeModeXor

	// Sum of source and destination (a.k.a. 'plus' or 'additive')
	// c_out = c_src + c_dst
	CompositeModeLighter

	// The product of source and destination (a.k.a 'multiply blend mode')
	// c_out = c_src * c_dst
	CompositeModeMultiply
)

func (c CompositeMode) blend() graphicsdriver.Blend {
	src, dst := c.blendFactors()
	return graphicsdriver.Blend{
		BlendFactorSourceColor:      src,
		BlendFactorSourceAlpha:      src,
		BlendFactorDestinationColor: dst,
		BlendFactorDestinationAlpha: dst,
		BlendOperationColor:         graphicsdriver.BlendOperationAdd,
		BlendOperationAlpha:         graphicsdriver.BlendOperationAdd,
	}
}

func (c CompositeMode) blendFactors() (src graphicsdriver.BlendFactor, dst graphicsdriver.BlendFactor) {
	switch c {
	case CompositeModeSourceOver:
		return graphicsdriver.BlendFactorOne, graphicsdriver.BlendFactorOneMinusSourceAlpha
	case CompositeModeClear:
		return graphicsdriver.BlendFactorZero, graphicsdriver.BlendFactorZero
	case CompositeModeCopy:
		return graphicsdriver.BlendFactorOne, graphicsdriver.BlendFactorZero
	case CompositeModeDestination:
		return graphicsdriver.BlendFactorZero, graphicsdriver.BlendFactorOne
	case CompositeModeDestinationOver:
		return graphicsdriver.BlendFactorOneMinusDestinationAlpha, graphicsdriver.BlendFactorOne
	case CompositeModeSourceIn:
		return graphicsdriver.BlendFactorDestinationAlpha, graphicsdriver.BlendFactorZero
	case CompositeModeDestinationIn:
		return graphicsdriver.BlendFactorZero, graphicsdriver.BlendFactorSourceAlpha
	case CompositeModeSourceOut:
		return graphicsdriver.BlendFactorOneMinusDestinationAlpha, graphicsdriver.BlendFactorZero
	case CompositeModeDestinationOut:
		return graphicsdriver.BlendFactorZero, graphicsdriver.BlendFactorOneMinusSourceAlpha
	case CompositeModeSourceAtop:
		return graphicsdriver.BlendFactorDestinationAlpha, graphicsdriver.BlendFactorOneMinusSourceAlpha
	case CompositeModeDestinationAtop:
		return graphicsdriver.BlendFactorOneMinusDestinationAlpha, graphicsdriver.BlendFactorSourceAlpha
	case CompositeModeXor:
		return graphicsdriver.BlendFactorOneMinusDestinationAlpha, graphicsdriver.BlendFactorOneMinusSourceAlpha
	case CompositeModeLighter:
		return graphicsdriver.BlendFactorOne, graphicsdriver.BlendFactorOne
	case CompositeModeMultiply:
		return graphicsdriver.BlendFactorDestinationColor, graphicsdriver.BlendFactorZero
	default:
		panic(fmt.Sprintf("ebiten: invalid composite mode: %d", c))
	}
}

// GraphicsLibrary represets graphics libraries supported by the engine.
type GraphicsLibrary = ui.GraphicsLibrary

const (
	// GraphicsLibraryUnknown represents the state at which graphics library cannot be determined,
	// e.g. hasn't loaded yet or failed to initialize.
	GraphicsLibraryUnknown GraphicsLibrary = ui.GraphicsLibraryUnknown

	// GraphicsLibraryOpenGL represents the graphics library OpenGL.
	GraphicsLibraryOpenGL GraphicsLibrary = ui.GraphicsLibraryOpenGL

	// GraphicsLibraryDirectX represents the graphics library Microsoft DirectX.
	GraphicsLibraryDirectX GraphicsLibrary = ui.GraphicsLibraryDirectX

	// GraphicsLibraryMetal represents the graphics library Apple's Metal.
	GraphicsLibraryMetal GraphicsLibrary = ui.GraphicsLibraryMetal
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
