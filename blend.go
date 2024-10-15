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
// The final color is calculated like this:
//
//	c_src: source RGB values
//	c_dst: destination RGB values
//	c_out: result RGB values
//	α_src: source alpha values
//	α_dst: destination alpha values
//	α_out: result alpha values
//
//	c_out = BlendOperationRGB((BlendFactorSourceRGB) × c_src, (BlendFactorDestinationRGB) × c_dst)
//	α_out = BlendOperationAlpha((BlendFactorSourceAlpha) × α_src, (BlendFactorDestinationAlpha) × α_dst)
//
// A blend factor is a factor for source and color destination color values.
// The default is source-over (regular alpha blending).
//
// A blend operation is a binary operator of a source color and a destination color.
// The default is adding.
type Blend struct {
	// BlendFactorSourceRGB is a factor for source RGB values.
	BlendFactorSourceRGB BlendFactor

	// BlendFactorSourceAlpha is a factor for source alpha values.
	BlendFactorSourceAlpha BlendFactor

	// BlendFactorDestinationRGB is a factor for destination RGB values.
	BlendFactorDestinationRGB BlendFactor

	// BlendFactorDestinationAlpha is a factor for destination alpha values.
	BlendFactorDestinationAlpha BlendFactor

	// BlendOperationRGB is an operation for source and destination RGB values.
	BlendOperationRGB BlendOperation

	// BlendOperationAlpha is an operation for source and destination alpha values.
	BlendOperationAlpha BlendOperation
}

var (
	defaultBlendInternalBlend = graphicsdriver.Blend{
		BlendFactorSourceRGB:        BlendFactorDefault.internalBlendFactor(true),
		BlendFactorSourceAlpha:      BlendFactorDefault.internalBlendFactor(true),
		BlendFactorDestinationRGB:   BlendFactorDefault.internalBlendFactor(false),
		BlendFactorDestinationAlpha: BlendFactorDefault.internalBlendFactor(false),
		BlendOperationRGB:           BlendOperationAdd.internalBlendOperation(),
		BlendOperationAlpha:         BlendOperationAdd.internalBlendOperation(),
	}
)

func (b Blend) internalBlend() graphicsdriver.Blend {
	// A shortcut for the most common blend.
	if b == (Blend{}) {
		return defaultBlendInternalBlend
	}
	return graphicsdriver.Blend{
		BlendFactorSourceRGB:        b.BlendFactorSourceRGB.internalBlendFactor(true),
		BlendFactorSourceAlpha:      b.BlendFactorSourceAlpha.internalBlendFactor(true),
		BlendFactorDestinationRGB:   b.BlendFactorDestinationRGB.internalBlendFactor(false),
		BlendFactorDestinationAlpha: b.BlendFactorDestinationAlpha.internalBlendFactor(false),
		BlendOperationRGB:           b.BlendOperationRGB.internalBlendOperation(),
		BlendOperationAlpha:         b.BlendOperationAlpha.internalBlendOperation(),
	}
}

// BlendFactor is a factor for source and destination color values.
type BlendFactor byte

const (
	// BlendFactorDefault is the default factor value.
	// The actual value depends on which source or destination this value is used.
	BlendFactorDefault BlendFactor = iota

	// BlendFactorZero is a factor:
	//
	//     0
	BlendFactorZero

	// BlendFactorOne is a factor:
	//
	//     1
	BlendFactorOne

	// BlendFactorSourceColor is a factor:
	//
	//     (source RGBA)
	BlendFactorSourceColor

	// BlendFactorOneMinusSourceColor is a factor:
	//
	//     1 - (source color)
	BlendFactorOneMinusSourceColor

	// BlendFactorSourceAlpha is a factor:
	//
	//     (source alpha)
	BlendFactorSourceAlpha

	// BlendFactorOneMinusSourceAlpha is a factor:
	//
	//     1 - (source alpha)
	BlendFactorOneMinusSourceAlpha

	// BlendFactorDestinationColor is a factor:
	//
	//     (destination RGBA)
	BlendFactorDestinationColor

	// BlendFactorOneMinusDestinationColor is a factor:
	//
	//     1 - (destination RGBA)
	BlendFactorOneMinusDestinationColor

	// BlendFactorDestinationAlpha is a factor:
	//
	//     (destination alpha)
	BlendFactorDestinationAlpha

	// BlendFactorOneMinusDestinationAlpha is a factor:
	//
	//     1 - (destination alpha)
	BlendFactorOneMinusDestinationAlpha

	// TODO: Add BlendFactorSourceAlphaSaturated. This might not work well on some platforms like Steam SDK (#2382).
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
	case BlendFactorSourceColor:
		return graphicsdriver.BlendFactorSourceColor
	case BlendFactorOneMinusSourceColor:
		return graphicsdriver.BlendFactorOneMinusSourceColor
	case BlendFactorSourceAlpha:
		return graphicsdriver.BlendFactorSourceAlpha
	case BlendFactorOneMinusSourceAlpha:
		return graphicsdriver.BlendFactorOneMinusSourceAlpha
	case BlendFactorDestinationColor:
		return graphicsdriver.BlendFactorDestinationColor
	case BlendFactorOneMinusDestinationColor:
		return graphicsdriver.BlendFactorOneMinusDestinationColor
	case BlendFactorDestinationAlpha:
		return graphicsdriver.BlendFactorDestinationAlpha
	case BlendFactorOneMinusDestinationAlpha:
		return graphicsdriver.BlendFactorOneMinusDestinationAlpha
	default:
		panic(fmt.Sprintf("ebiten: invalid blend factor: %d", b))
	}
}

func internalBlendFactorToBlendFactor(blendFactor graphicsdriver.BlendFactor) BlendFactor {
	switch blendFactor {
	case graphicsdriver.BlendFactorZero:
		return BlendFactorZero
	case graphicsdriver.BlendFactorOne:
		return BlendFactorOne
	case graphicsdriver.BlendFactorSourceColor:
		return BlendFactorSourceColor
	case graphicsdriver.BlendFactorOneMinusSourceColor:
		return BlendFactorOneMinusSourceColor
	case graphicsdriver.BlendFactorSourceAlpha:
		return BlendFactorSourceAlpha
	case graphicsdriver.BlendFactorOneMinusSourceAlpha:
		return BlendFactorOneMinusSourceAlpha
	case graphicsdriver.BlendFactorDestinationColor:
		return BlendFactorDestinationColor
	case graphicsdriver.BlendFactorOneMinusDestinationColor:
		return BlendFactorOneMinusDestinationColor
	case graphicsdriver.BlendFactorDestinationAlpha:
		return BlendFactorDestinationAlpha
	case graphicsdriver.BlendFactorOneMinusDestinationAlpha:
		return BlendFactorOneMinusDestinationAlpha
	default:
		panic(fmt.Sprintf("ebiten: invalid blend factor: %d", blendFactor))
	}
}

// BlendOperation is an operation for source and destination color values.
type BlendOperation byte

const (
	// BlendOperationAdd represents adding the source and destination color.
	//
	//     c_out = (BlendFactorSourceRGB) × c_src + (BlendFactorDestinationRGB) × c_dst
	//     α_out = (BlendFactorSourceAlpha) × α_src + (BlendFactorDestinationAlpha) × α_dst
	BlendOperationAdd BlendOperation = iota

	// BlendOperationSubtract represents subtracting the source and destination color.
	//
	//     c_out = (BlendFactorSourceRGB) × c_src - (BlendFactorDestinationRGB) × c_dst
	//     α_out = (BlendFactorSourceAlpha) × α_src - (BlendFactorDestinationAlpha) × α_dst
	BlendOperationSubtract

	// BlendOperationReverseSubtract represents subtracting the source and destination color in a reversed order.
	//
	//     c_out = (BlendFactorDestinationRGB) × c_dst - (BlendFactorSourceRGB) × c_src
	//     α_out = (BlendFactorDestinationAlpha) × α_dst - (BlendFactorSourceAlpha) × α_src
	BlendOperationReverseSubtract

	// BlendOperationMin represents a minimum function for the source and destination color.
	// If BlendOperationMin is specified, blend factors are not used.
	//
	//     c_out = min(c_dst, c_src)
	//     α_out = min(α_dst, α_src)
	BlendOperationMin

	// BlendOperationMax represents a maximum function for the source and destination color.
	// If BlendOperationMax is specified, blend factors are not used.
	//
	//     c_out = max(c_dst, c_src)
	//     α_out = max(α_dst, α_src)
	BlendOperationMax
)

func (b BlendOperation) internalBlendOperation() graphicsdriver.BlendOperation {
	switch b {
	case BlendOperationAdd:
		return graphicsdriver.BlendOperationAdd
	case BlendOperationSubtract:
		return graphicsdriver.BlendOperationSubtract
	case BlendOperationReverseSubtract:
		return graphicsdriver.BlendOperationReverseSubtract
	case BlendOperationMin:
		return graphicsdriver.BlendOperationMin
	case BlendOperationMax:
		return graphicsdriver.BlendOperationMax
	default:
		panic(fmt.Sprintf("ebiten: invalid blend operation: %d", b))
	}
}

func internalBlendOperationToBlendOperation(blendOperation graphicsdriver.BlendOperation) BlendOperation {
	switch blendOperation {
	case graphicsdriver.BlendOperationAdd:
		return BlendOperationAdd
	case graphicsdriver.BlendOperationSubtract:
		return BlendOperationSubtract
	case graphicsdriver.BlendOperationReverseSubtract:
		return BlendOperationReverseSubtract
	case graphicsdriver.BlendOperationMin:
		return BlendOperationMin
	case graphicsdriver.BlendOperationMax:
		return BlendOperationMax
	default:
		panic(fmt.Sprintf("ebiten: invalid blend operation: %d", blendOperation))
	}
}

func internalBlendToBlend(blend graphicsdriver.Blend) Blend {
	return Blend{
		BlendFactorSourceRGB:        internalBlendFactorToBlendFactor(blend.BlendFactorSourceRGB),
		BlendFactorSourceAlpha:      internalBlendFactorToBlendFactor(blend.BlendFactorSourceAlpha),
		BlendFactorDestinationRGB:   internalBlendFactorToBlendFactor(blend.BlendFactorDestinationRGB),
		BlendFactorDestinationAlpha: internalBlendFactorToBlendFactor(blend.BlendFactorDestinationAlpha),
		BlendOperationRGB:           internalBlendOperationToBlendOperation(blend.BlendOperationRGB),
		BlendOperationAlpha:         internalBlendOperationToBlendOperation(blend.BlendOperationAlpha),
	}
}

// This name convention follows CSS compositing: https://drafts.fxtf.org/compositing-2/.
//
// In the comments,
// c_src, c_dst and c_out represent alpha-premultiplied RGB values of source, destination and output respectively. α_src and α_dst represent alpha values of source and destination respectively.
var (
	// BlendSourceOver is a preset Blend for the regular alpha blending.
	//
	//     c_out = c_src + c_dst × (1 - α_src)
	//     α_out = α_src + α_dst × (1 - α_src)
	BlendSourceOver = internalBlendToBlend(graphicsdriver.BlendSourceOver)

	// BlendClear is a preset Blend for Porter Duff's 'clear'.
	//
	//     c_out = 0
	//     α_out = 0
	BlendClear = internalBlendToBlend(graphicsdriver.BlendClear)

	// BlendCopy is a preset Blend for Porter Duff's 'copy'.
	//
	//     c_out = c_src
	//     α_out = α_src
	BlendCopy = internalBlendToBlend(graphicsdriver.BlendCopy)

	// BlendDestination is a preset Blend for Porter Duff's 'destination'.
	//
	//     c_out = c_dst
	//     α_out = α_dst
	BlendDestination = internalBlendToBlend(graphicsdriver.BlendDestination)

	// BlendDestinationOver is a preset Blend for Porter Duff's 'destination-over'.
	//
	//     c_out = c_src × (1 - α_dst) + c_dst
	//     α_out = α_src × (1 - α_dst) + α_dst
	BlendDestinationOver = internalBlendToBlend(graphicsdriver.BlendDestinationOver)

	// BlendSourceIn is a preset Blend for Porter Duff's 'source-in'.
	//
	//     c_out = c_src × α_dst
	//     α_out = α_src × α_dst
	BlendSourceIn = internalBlendToBlend(graphicsdriver.BlendSourceIn)

	// BlendDestinationIn is a preset Blend for Porter Duff's 'destination-in'.
	//
	//     c_out = c_dst × α_src
	//     α_out = α_dst × α_src
	BlendDestinationIn = internalBlendToBlend(graphicsdriver.BlendDestinationIn)

	// BlendSourceOut is a preset Blend for Porter Duff's 'source-out'.
	//
	//     c_out = c_src × (1 - α_dst)
	//     α_out = α_src × (1 - α_dst)
	BlendSourceOut = internalBlendToBlend(graphicsdriver.BlendSourceOut)

	// BlendDestinationOut is a preset Blend for Porter Duff's 'destination-out'.
	//
	//     c_out = c_dst × (1 - α_src)
	//     α_out = α_dst × (1 - α_src)
	BlendDestinationOut = internalBlendToBlend(graphicsdriver.BlendDestinationOut)

	// BlendSourceAtop is a preset Blend for Porter Duff's 'source-atop'.
	//
	//     c_out = c_src × α_dst + c_dst × (1 - α_src)
	//     α_out = α_src × α_dst + α_dst × (1 - α_src)
	BlendSourceAtop = internalBlendToBlend(graphicsdriver.BlendSourceAtop)

	// BlendDestinationAtop is a preset Blend for Porter Duff's 'destination-atop'.
	//
	//     c_out = c_src × (1 - α_dst) + c_dst × α_src
	//     α_out = α_src × (1 - α_dst) + α_dst × α_src
	BlendDestinationAtop = internalBlendToBlend(graphicsdriver.BlendDestinationAtop)

	// BlendXor is a preset Blend for Porter Duff's 'xor'.
	//
	//     c_out = c_src × (1 - α_dst) + c_dst × (1 - α_src)
	//     α_out = α_src × (1 - α_dst) + α_dst × (1 - α_src)
	BlendXor = internalBlendToBlend(graphicsdriver.BlendXor)

	// BlendLighter is a preset Blend for Porter Duff's 'lighter'.
	// This is sum of source and destination (a.k.a. 'plus' or 'additive')
	//
	//     c_out = c_src + c_dst
	//     α_out = α_src + α_dst
	BlendLighter = internalBlendToBlend(graphicsdriver.BlendLighter)
)
