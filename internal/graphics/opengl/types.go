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

type Filter int
type ShaderType int
type BufferType int
type BufferUsage int
type Mode int
type operation int

type Context struct {
	Nearest            Filter
	Linear             Filter
	VertexShader       ShaderType
	FragmentShader     ShaderType
	ArrayBuffer        BufferType
	ElementArrayBuffer BufferType
	DynamicDraw        BufferUsage
	StaticDraw         BufferUsage
	Triangles          Mode
	Lines              Mode
	zero               operation
	one                operation
	srcAlpha           operation
	dstAlpha           operation
	oneMinusSrcAlpha   operation
	oneMinusDstAlpha   operation
	context
}

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

func (c *Context) operations(mode CompositeMode) (src operation, dst operation) {
	switch mode {
	case CompositeModeSourceOver:
		return c.one, c.oneMinusSrcAlpha
	case CompositeModeClear:
		return c.zero, c.zero
	case CompositeModeCopy:
		return c.one, c.zero
	case CompositeModeDestination:
		return c.zero, c.one
	case CompositeModeDestinationOver:
		return c.oneMinusDstAlpha, c.one
	case CompositeModeSourceIn:
		return c.dstAlpha, c.zero
	case CompositeModeDestinationIn:
		return c.zero, c.srcAlpha
	case CompositeModeSourceOut:
		return c.oneMinusDstAlpha, c.zero
	case CompositeModeDestinationOut:
		return c.zero, c.oneMinusSrcAlpha
	case CompositeModeSourceAtop:
		return c.dstAlpha, c.oneMinusSrcAlpha
	case CompositeModeDestinationAtop:
		return c.oneMinusDstAlpha, c.srcAlpha
	case CompositeModeXor:
		return c.oneMinusDstAlpha, c.oneMinusSrcAlpha
	case CompositeModeLighter:
		return c.one, c.one
	default:
		panic("not reach")
	}
}
