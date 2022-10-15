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
	"fmt"
)

type CompositeMode int

const (
	CompositeModeUnknown    CompositeMode = iota - 1
	CompositeModeSourceOver               // This value must be 0 (= initial value)
	CompositeModeClear
	CompositeModeCopy
	CompositeModeDestination
	CompositeModeDestinationOver
	CompositeModeSourceIn
	CompositeModeDestinationIn
	CompositeModeSourceOut
	CompositeModeDestinationOut
	CompositeModeSourceAtop
	CompositeModeDestinationAtop
	CompositeModeXor
	CompositeModeLighter
	CompositeModeMultiply

	CompositeModeMax = CompositeModeMultiply
)

type BlendFactor int

const (
	BlendFactorZero BlendFactor = iota
	BlendFactorOne
	BlendFactorSrcAlpha
	BlendFactorDstAlpha
	BlendFactorOneMinusSrcAlpha
	BlendFactorOneMinusDstAlpha
	BlendFactorDstColor
)

func (c CompositeMode) BlendFactors() (src BlendFactor, dst BlendFactor) {
	switch c {
	case CompositeModeSourceOver:
		return BlendFactorOne, BlendFactorOneMinusSrcAlpha
	case CompositeModeClear:
		return BlendFactorZero, BlendFactorZero
	case CompositeModeCopy:
		return BlendFactorOne, BlendFactorZero
	case CompositeModeDestination:
		return BlendFactorZero, BlendFactorOne
	case CompositeModeDestinationOver:
		return BlendFactorOneMinusDstAlpha, BlendFactorOne
	case CompositeModeSourceIn:
		return BlendFactorDstAlpha, BlendFactorZero
	case CompositeModeDestinationIn:
		return BlendFactorZero, BlendFactorSrcAlpha
	case CompositeModeSourceOut:
		return BlendFactorOneMinusDstAlpha, BlendFactorZero
	case CompositeModeDestinationOut:
		return BlendFactorZero, BlendFactorOneMinusSrcAlpha
	case CompositeModeSourceAtop:
		return BlendFactorDstAlpha, BlendFactorOneMinusSrcAlpha
	case CompositeModeDestinationAtop:
		return BlendFactorOneMinusDstAlpha, BlendFactorSrcAlpha
	case CompositeModeXor:
		return BlendFactorOneMinusDstAlpha, BlendFactorOneMinusSrcAlpha
	case CompositeModeLighter:
		return BlendFactorOne, BlendFactorOne
	case CompositeModeMultiply:
		return BlendFactorDstColor, BlendFactorZero
	default:
		panic(fmt.Sprintf("graphicsdriver: invalid composite mode: %d", c))
	}
}
