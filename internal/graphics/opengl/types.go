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

type CompositionMode int

const (
	CompositionModeSourceOver CompositionMode = iota
	CompositionModeClear
	CompositionModeCopy
	CompositionModeDesination
	CompositionModeDesinationOver
	CompositionModeSourceIn
	CompositionModeDestinationIn
	CompositionModeSourceOut
	CompositionModeDestinationOut
	CompositionModeSourceAtop
	CompositionModeDestinationAtop
	CompositionModeXor
	CompositionModeLighter
)

func (c *Context) operations(mode CompositionMode) (src operation, dst operation) {
	switch mode {
	case CompositionModeSourceOver:
		return c.one, c.oneMinusSrcAlpha
	case CompositionModeClear:
		return c.zero, c.zero
	case CompositionModeCopy:
		return c.one, c.zero
	case CompositionModeDesination:
		return c.zero, c.one
	case CompositionModeDesinationOver:
		return c.one, c.oneMinusDstAlpha
	case CompositionModeSourceIn:
		return c.dstAlpha, c.zero
	case CompositionModeDestinationIn:
		return c.zero, c.srcAlpha
	case CompositionModeSourceOut:
		return c.oneMinusDstAlpha, c.zero
	case CompositionModeDestinationOut:
		return c.zero, c.oneMinusSrcAlpha
	case CompositionModeSourceAtop:
		return c.dstAlpha, c.oneMinusSrcAlpha
	case CompositionModeDestinationAtop:
		return c.oneMinusDstAlpha, c.srcAlpha
	case CompositionModeXor:
		return c.oneMinusDstAlpha, c.oneMinusSrcAlpha
	case CompositionModeLighter:
		return c.one, c.one
	default:
		panic("not reach")
	}
}
