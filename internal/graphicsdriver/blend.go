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
	BlendFactorSourceColor      BlendFactor
	BlendFactorSourceAlpha      BlendFactor
	BlendFactorDestinationColor BlendFactor
	BlendFactorDestinationAlpha BlendFactor
	BlendOperationColor         BlendOperation
	BlendOperationAlpha         BlendOperation
}

type BlendFactor int

const (
	BlendFactorZero BlendFactor = iota
	BlendFactorOne
	BlendFactorSourceAlpha
	BlendFactorDestinationAlpha
	BlendFactorOneMinusSourceAlpha
	BlendFactorOneMinusDestinationAlpha
	BlendFactorDestinationColor
)

type BlendOperation int

const (
	BlendOperationAdd BlendOperation = iota

	// TODO: Add more operators
)

var BlendSourceOver = Blend{
	BlendFactorSourceColor:      BlendFactorOne,
	BlendFactorSourceAlpha:      BlendFactorOne,
	BlendFactorDestinationColor: BlendFactorOneMinusSourceAlpha,
	BlendFactorDestinationAlpha: BlendFactorOneMinusSourceAlpha,
	BlendOperationColor:         BlendOperationAdd,
	BlendOperationAlpha:         BlendOperationAdd,
}

var BlendClear = Blend{
	BlendFactorSourceColor:      BlendFactorZero,
	BlendFactorSourceAlpha:      BlendFactorZero,
	BlendFactorDestinationColor: BlendFactorZero,
	BlendFactorDestinationAlpha: BlendFactorZero,
	BlendOperationColor:         BlendOperationAdd,
	BlendOperationAlpha:         BlendOperationAdd,
}

var BlendCopy = Blend{
	BlendFactorSourceColor:      BlendFactorOne,
	BlendFactorSourceAlpha:      BlendFactorOne,
	BlendFactorDestinationColor: BlendFactorZero,
	BlendFactorDestinationAlpha: BlendFactorZero,
	BlendOperationColor:         BlendOperationAdd,
	BlendOperationAlpha:         BlendOperationAdd,
}
