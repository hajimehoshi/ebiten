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
//
// Deprecated: as of v2.5. Use Blend instead.
type CompositeMode int

const (
	// CompositeModeCustom indicates to refer Blend.
	CompositeModeCustom CompositeMode = iota

	// Deprecated: as of v2.5. Use BlendSourceOver instead.
	CompositeModeSourceOver

	// Deprecated: as of v2.5. Use BlendClear instead.
	CompositeModeClear

	// Deprecated: as of v2.5. Use BlendCopy instead.
	CompositeModeCopy

	// Deprecated: as of v2.5. Use BlendDestination instead.
	CompositeModeDestination

	// Deprecated: as of v2.5. Use BlendDestinationOver instead.
	CompositeModeDestinationOver

	// Deprecated: as of v2.5. Use BlendSourceIn instead.
	CompositeModeSourceIn

	// Deprecated: as of v2.5. Use BlendDestinationIn instead.
	CompositeModeDestinationIn

	// Deprecated: as of v2.5. Use BlendSourceOut instead.
	CompositeModeSourceOut

	// Deprecated: as of v2.5. Use BlendDestinationOut instead.
	CompositeModeDestinationOut

	// Deprecated: as of v2.5. Use BlendSourceAtop instead.
	CompositeModeSourceAtop

	// Deprecated: as of v2.5. Use BlendDestinationAtop instead.
	CompositeModeDestinationAtop

	// Deprecated: as of v2.5. Use BlendXor instead.
	CompositeModeXor

	// Deprecated: as of v2.5. Use BlendLighter instead.
	CompositeModeLighter

	// Deprecated: as of v2.5. Use Blend with BlendFactorDestinationColor and BlendFactorZero instead.
	CompositeModeMultiply
)

func (c CompositeMode) blend() Blend {
	switch c {
	case CompositeModeSourceOver:
		return BlendSourceOver
	case CompositeModeClear:
		return BlendClear
	case CompositeModeCopy:
		return BlendCopy
	case CompositeModeDestination:
		return BlendDestination
	case CompositeModeDestinationOver:
		return BlendDestinationOver
	case CompositeModeSourceIn:
		return BlendSourceIn
	case CompositeModeDestinationIn:
		return BlendDestinationIn
	case CompositeModeSourceOut:
		return BlendSourceOut
	case CompositeModeDestinationOut:
		return BlendDestinationOut
	case CompositeModeSourceAtop:
		return BlendSourceAtop
	case CompositeModeDestinationAtop:
		return BlendDestinationAtop
	case CompositeModeXor:
		return BlendXor
	case CompositeModeLighter:
		return BlendLighter
	case CompositeModeMultiply:
		return Blend{
			BlendFactorSourceRGB:        BlendFactorDestinationColor,
			BlendFactorSourceAlpha:      BlendFactorDestinationColor,
			BlendFactorDestinationRGB:   BlendFactorZero,
			BlendFactorDestinationAlpha: BlendFactorZero,
			BlendOperationRGB:           BlendOperationAdd,
			BlendOperationAlpha:         BlendOperationAdd,
		}
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
