// Copyright 2022 The Ebitengine Authors
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

type Blend struct {
	BlendFactorSourceRGB        BlendFactor
	BlendFactorSourceAlpha      BlendFactor
	BlendFactorDestinationRGB   BlendFactor
	BlendFactorDestinationAlpha BlendFactor
	BlendOperationRGB           BlendOperation
	BlendOperationAlpha         BlendOperation
}

// BlendFactor and BlendOperation must be synced with internal/graphicsdriver/playstation5/graphics_playstation5.h.

type BlendFactor byte

const (
	BlendFactorZero BlendFactor = iota
	BlendFactorOne
	BlendFactorSourceColor
	BlendFactorOneMinusSourceColor
	BlendFactorSourceAlpha
	BlendFactorOneMinusSourceAlpha
	BlendFactorDestinationColor
	BlendFactorOneMinusDestinationColor
	BlendFactorDestinationAlpha
	BlendFactorOneMinusDestinationAlpha
	BlendFactorSourceAlphaSaturated
)

type BlendOperation byte

const (
	BlendOperationAdd BlendOperation = iota
	BlendOperationSubtract
	BlendOperationReverseSubtract
	BlendOperationMin
	BlendOperationMax
)

var (
	BlendSourceOver = Blend{
		BlendFactorSourceRGB:        BlendFactorOne,
		BlendFactorSourceAlpha:      BlendFactorOne,
		BlendFactorDestinationRGB:   BlendFactorOneMinusSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorOneMinusSourceAlpha,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendClear = Blend{
		BlendFactorSourceRGB:        BlendFactorZero,
		BlendFactorSourceAlpha:      BlendFactorZero,
		BlendFactorDestinationRGB:   BlendFactorZero,
		BlendFactorDestinationAlpha: BlendFactorZero,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendCopy = Blend{
		BlendFactorSourceRGB:        BlendFactorOne,
		BlendFactorSourceAlpha:      BlendFactorOne,
		BlendFactorDestinationRGB:   BlendFactorZero,
		BlendFactorDestinationAlpha: BlendFactorZero,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendDestination = Blend{
		BlendFactorSourceRGB:        BlendFactorZero,
		BlendFactorSourceAlpha:      BlendFactorZero,
		BlendFactorDestinationRGB:   BlendFactorOne,
		BlendFactorDestinationAlpha: BlendFactorOne,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendDestinationOver = Blend{
		BlendFactorSourceRGB:        BlendFactorOneMinusDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorDestinationRGB:   BlendFactorOne,
		BlendFactorDestinationAlpha: BlendFactorOne,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendSourceIn = Blend{
		BlendFactorSourceRGB:        BlendFactorDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorDestinationAlpha,
		BlendFactorDestinationRGB:   BlendFactorZero,
		BlendFactorDestinationAlpha: BlendFactorZero,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendDestinationIn = Blend{
		BlendFactorSourceRGB:        BlendFactorZero,
		BlendFactorSourceAlpha:      BlendFactorZero,
		BlendFactorDestinationRGB:   BlendFactorSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorSourceAlpha,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendSourceOut = Blend{
		BlendFactorSourceRGB:        BlendFactorOneMinusDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorDestinationRGB:   BlendFactorZero,
		BlendFactorDestinationAlpha: BlendFactorZero,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendDestinationOut = Blend{
		BlendFactorSourceRGB:        BlendFactorZero,
		BlendFactorSourceAlpha:      BlendFactorZero,
		BlendFactorDestinationRGB:   BlendFactorOneMinusSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorOneMinusSourceAlpha,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendSourceAtop = Blend{
		BlendFactorSourceRGB:        BlendFactorDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorDestinationAlpha,
		BlendFactorDestinationRGB:   BlendFactorOneMinusSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorOneMinusSourceAlpha,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendDestinationAtop = Blend{
		BlendFactorSourceRGB:        BlendFactorOneMinusDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorDestinationRGB:   BlendFactorSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorSourceAlpha,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendXor = Blend{
		BlendFactorSourceRGB:        BlendFactorOneMinusDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorDestinationRGB:   BlendFactorOneMinusSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorOneMinusSourceAlpha,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	BlendLighter = Blend{
		BlendFactorSourceRGB:        BlendFactorOne,
		BlendFactorSourceAlpha:      BlendFactorOne,
		BlendFactorDestinationRGB:   BlendFactorOne,
		BlendFactorDestinationAlpha: BlendFactorOne,
		BlendOperationRGB:           BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}
)
