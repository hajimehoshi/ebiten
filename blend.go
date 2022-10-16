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

package ebiten

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
)

// Blend is a blending way of the source color and the destination color.
//
// The default (zero) value is source-over (regular alpha blending).
type Blend struct {
	// BlendFactorSourceColor is a factor for source RGB values.
	BlendFactorSourceColor BlendFactor

	// BlendFactorSourceAlpha is a factor for source alpha values.
	BlendFactorSourceAlpha BlendFactor

	// BlendFactorDestinationColor is a factor for destination RGB values.
	BlendFactorDestinationColor BlendFactor

	// BlendFactorDestinationAlpha is a factor for destination apha values.
	BlendFactorDestinationAlpha BlendFactor

	// BlendOperationColor is an operation for source and destination RGB values.
	BlendOperationColor BlendOperation

	// BlendOperationAlpha is an operation for source and destination alpha values.
	BlendOperationAlpha BlendOperation
}

func (b Blend) internalBlend() graphicsdriver.Blend {
	return graphicsdriver.Blend{
		BlendFactorSourceColor:      b.BlendFactorSourceColor.internalBlendFactor(true),
		BlendFactorSourceAlpha:      b.BlendFactorSourceAlpha.internalBlendFactor(true),
		BlendFactorDestinationColor: b.BlendFactorDestinationColor.internalBlendFactor(false),
		BlendFactorDestinationAlpha: b.BlendFactorDestinationAlpha.internalBlendFactor(false),
		BlendOperationColor:         b.BlendOperationColor.internalBlendOperation(),
		BlendOperationAlpha:         b.BlendOperationAlpha.internalBlendOperation(),
	}
}

// BlendFactor is a factor for source and destination color values.
type BlendFactor byte

const (
	// BlendFactorDefault is the default factor value.
	// The actual value depends on which source or destination this value is used.
	BlendFactorDefault BlendFactor = iota
	BlendFactorZero
	BlendFactorOne
	BlendFactorSourceAlpha
	BlendFactorDestinationAlpha
	BlendFactorOneMinusSourceAlpha
	BlendFactorOneMinusDestinationAlpha
	BlendFactorDestinationColor
)

func (b BlendFactor) internalBlendFactor(source bool) graphicsdriver.BlendFactor {
	switch b {
	case BlendFactorDefault:
		// The default is the source-over composition (regular alpha blending).
		if source {
			return graphicsdriver.BlendFactorOne
		}
		return graphicsdriver.BlendFactorOneMinusSourceAlpha
	case BlendFactorZero:
		return graphicsdriver.BlendFactorZero
	case BlendFactorOne:
		return graphicsdriver.BlendFactorOne
	case BlendFactorSourceAlpha:
		return graphicsdriver.BlendFactorSourceAlpha
	case BlendFactorDestinationAlpha:
		return graphicsdriver.BlendFactorDestinationAlpha
	case BlendFactorOneMinusSourceAlpha:
		return graphicsdriver.BlendFactorOneMinusSourceAlpha
	case BlendFactorOneMinusDestinationAlpha:
		return graphicsdriver.BlendFactorOneMinusDestinationAlpha
	case BlendFactorDestinationColor:
		return graphicsdriver.BlendFactorDestinationColor
	default:
		panic(fmt.Sprintf("ebiten: invalid blend factor: %d", b))
	}
}

// BlendFactor is an operation for source and destination color values.
type BlendOperation byte

const (
	// BlendOperationAdd represents adding the source and destination color.
	// c_out = factor_src × c_src + factor_dst × c_dst
	BlendOperationAdd BlendOperation = iota
)

func (b BlendOperation) internalBlendOperation() graphicsdriver.BlendOperation {
	switch b {
	case BlendOperationAdd:
		return graphicsdriver.BlendOperationAdd
	default:
		panic(fmt.Sprintf("ebiten: invalid blend operation: %d", b))
	}
}

// This name convention follows CSS compositing: https://drafts.fxtf.org/compositing-2/.
//
// In the comments,
// c_src, c_dst and c_out represent alpha-premultiplied RGB values of source, destination and output respectively. α_src and α_dst represent alpha values of source and destination respectively.
var (
	// BlendSourceOver represents the regular alpha blending.
	// c_out = c_src + c_dst × (1 - α_src)
	BlendSourceOver = Blend{
		BlendFactorSourceColor:      BlendFactorOne,
		BlendFactorSourceAlpha:      BlendFactorOne,
		BlendFactorDestinationColor: BlendFactorOneMinusSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorOneMinusSourceAlpha,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = 0
	BlendClear = Blend{
		BlendFactorSourceColor:      BlendFactorZero,
		BlendFactorSourceAlpha:      BlendFactorZero,
		BlendFactorDestinationColor: BlendFactorZero,
		BlendFactorDestinationAlpha: BlendFactorZero,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = c_src
	BlendCopy = Blend{
		BlendFactorSourceColor:      BlendFactorOne,
		BlendFactorSourceAlpha:      BlendFactorOne,
		BlendFactorDestinationColor: BlendFactorZero,
		BlendFactorDestinationAlpha: BlendFactorZero,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = c_dst
	BlendDestination = Blend{
		BlendFactorSourceColor:      BlendFactorZero,
		BlendFactorSourceAlpha:      BlendFactorZero,
		BlendFactorDestinationColor: BlendFactorOne,
		BlendFactorDestinationAlpha: BlendFactorOne,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = c_src × (1 - α_dst) + c_dst
	BlendDestinationOver = Blend{
		BlendFactorSourceColor:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorDestinationColor: BlendFactorOne,
		BlendFactorDestinationAlpha: BlendFactorOne,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = c_src × α_dst
	BlendSourceIn = Blend{
		BlendFactorSourceColor:      BlendFactorDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorDestinationAlpha,
		BlendFactorDestinationColor: BlendFactorZero,
		BlendFactorDestinationAlpha: BlendFactorZero,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = c_dst × α_src
	BlendDestinationIn = Blend{
		BlendFactorSourceColor:      BlendFactorZero,
		BlendFactorSourceAlpha:      BlendFactorZero,
		BlendFactorDestinationColor: BlendFactorSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorSourceAlpha,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = c_src × (1 - α_dst)
	BlendSourceOut = Blend{
		BlendFactorSourceColor:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorDestinationColor: BlendFactorZero,
		BlendFactorDestinationAlpha: BlendFactorZero,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = c_dst × (1 - α_src)
	BlendDestinationOut = Blend{
		BlendFactorSourceColor:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorDestinationColor: BlendFactorZero,
		BlendFactorDestinationAlpha: BlendFactorZero,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = c_src × α_dst + c_dst × (1 - α_src)
	BlendSourceAtop = Blend{
		BlendFactorSourceColor:      BlendFactorDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorDestinationAlpha,
		BlendFactorDestinationColor: BlendFactorOneMinusSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorOneMinusSourceAlpha,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = c_src × (1 - α_dst) + c_dst × α_src
	BlendDestinationAtop = Blend{
		BlendFactorSourceColor:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorDestinationColor: BlendFactorSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorSourceAlpha,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// c_out = c_src × (1 - α_dst) + c_dst × (1 - α_src)
	BlendXor = Blend{
		BlendFactorSourceColor:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorSourceAlpha:      BlendFactorOneMinusDestinationAlpha,
		BlendFactorDestinationColor: BlendFactorOneMinusSourceAlpha,
		BlendFactorDestinationAlpha: BlendFactorOneMinusSourceAlpha,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}

	// Sum of source and destination (a.k.a. 'plus' or 'additive')
	// c_out = c_src + c_dst
	BlendLighter = Blend{
		BlendFactorSourceColor:      BlendFactorOne,
		BlendFactorSourceAlpha:      BlendFactorOne,
		BlendFactorDestinationColor: BlendFactorOne,
		BlendFactorDestinationAlpha: BlendFactorOne,
		BlendOperationColor:         BlendOperationAdd,
		BlendOperationAlpha:         BlendOperationAdd,
	}
)
