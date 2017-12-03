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

package opengl

type (
	ShaderType  int
	BufferType  int
	BufferUsage int
	Mode        int
	operation   int
)

type CompositeMode int

const (
	CompositeModeSourceOver CompositeMode = iota // This value must be 0 (= initial value)
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
	CompositeModeUnknown
)

func operations(mode CompositeMode) (src operation, dst operation) {
	switch mode {
	case CompositeModeSourceOver:
		return one, oneMinusSrcAlpha
	case CompositeModeClear:
		return zero, zero
	case CompositeModeCopy:
		return one, zero
	case CompositeModeDestination:
		return zero, one
	case CompositeModeDestinationOver:
		return oneMinusDstAlpha, one
	case CompositeModeSourceIn:
		return dstAlpha, zero
	case CompositeModeDestinationIn:
		return zero, srcAlpha
	case CompositeModeSourceOut:
		return oneMinusDstAlpha, zero
	case CompositeModeDestinationOut:
		return zero, oneMinusSrcAlpha
	case CompositeModeSourceAtop:
		return dstAlpha, oneMinusSrcAlpha
	case CompositeModeDestinationAtop:
		return oneMinusDstAlpha, srcAlpha
	case CompositeModeXor:
		return oneMinusDstAlpha, oneMinusSrcAlpha
	case CompositeModeLighter:
		return one, one
	default:
		panic("not reached")
	}
}

type DataType int

func (d DataType) SizeInBytes() int {
	switch d {
	case Short:
		return 2
	case Float:
		return 4
	default:
		panic("not reached")
	}
}
