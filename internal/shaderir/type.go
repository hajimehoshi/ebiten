// Copyright 2020 The Ebiten Authors
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

package shaderir

import (
	"fmt"
	"strings"
)

type Type struct {
	Main   BasicType
	Sub    []Type
	Length int
}

func (t Type) Equal(rhs *Type) bool {
	if t.Main != rhs.Main {
		return false
	}
	if t.Length != rhs.Length {
		return false
	}
	if len(t.Sub) != len(rhs.Sub) {
		return false
	}
	for i, s := range t.Sub {
		if !s.Equal(&rhs.Sub[i]) {
			return false
		}
	}
	return true
}

func (t Type) String() string {
	switch t.Main {
	case None:
		return "none"
	case Bool:
		return "bool"
	case Int:
		return "int"
	case Float:
		return "float"
	case Vec2:
		return "vec2"
	case Vec3:
		return "vec3"
	case Vec4:
		return "vec4"
	case IVec2:
		return "ivec2"
	case IVec3:
		return "ivec3"
	case IVec4:
		return "ivec4"
	case Mat2:
		return "mat2"
	case Mat3:
		return "mat3"
	case Mat4:
		return "mat4"
	case Array:
		return fmt.Sprintf("[%d]%s", t.Length, t.Sub[0].String())
	case Struct:
		str := "struct{"
		sub := make([]string, 0, len(t.Sub))
		for _, st := range t.Sub {
			sub = append(sub, st.String())
		}
		str += strings.Join(sub, ",")
		str += "}"
		return str
	default:
		return fmt.Sprintf("?(unknown type: %d)", t.Main)
	}
}

func (t Type) DwordCount() int {
	switch t.Main {
	case Bool:
		// The size of a bool varies by the shader language, but use 1 for simplicity.
		return 1
	case Int:
		return 1
	case Float:
		return 1
	case Vec2:
		return 2
	case Vec3:
		return 3
	case Vec4:
		return 4
	case IVec2:
		return 2
	case IVec3:
		return 3
	case IVec4:
		return 4
	case Mat2:
		return 4
	case Mat3:
		return 9
	case Mat4:
		return 16
	case Array:
		return t.Length * t.Sub[0].DwordCount()
	default: // TODO: Parse a struct correctly
		return -1
	}
}

func (t Type) IsFloatVector() bool {
	switch t.Main {
	case Vec2, Vec3, Vec4:
		return true
	}
	return false
}

func (t Type) IsIntVector() bool {
	switch t.Main {
	case IVec2, IVec3, IVec4:
		return true
	}
	return false
}

func (t Type) VectorElementCount() int {
	switch t.Main {
	case Vec2:
		return 2
	case Vec3:
		return 3
	case Vec4:
		return 4
	case IVec2:
		return 2
	case IVec3:
		return 3
	case IVec4:
		return 4
	default:
		return -1
	}
}

func (t Type) IsMatrix() bool {
	switch t.Main {
	case Mat2, Mat3, Mat4:
		return true
	}
	return false
}

func (t Type) MatrixSize() int {
	switch t.Main {
	case Mat2:
		return 2
	case Mat3:
		return 3
	case Mat4:
		return 4
	default:
		return -1
	}
}

type BasicType int

const (
	None BasicType = iota
	Bool
	Int
	Float
	Vec2
	Vec3
	Vec4
	IVec2
	IVec3
	IVec4
	Mat2
	Mat3
	Mat4
	Texture
	Array
	Struct
)

func descendantLocalVars(block, target *Block) ([]Type, bool) {
	if block == target {
		return block.LocalVars, true
	}

	var ts []Type
	for _, s := range block.Stmts {
		for _, b := range s.Blocks {
			if ts2, found := descendantLocalVars(b, target); found {
				n := b.LocalVarIndexOffset - block.LocalVarIndexOffset
				ts = append(ts, block.LocalVars[:n]...)
				ts = append(ts, ts2...)
				return ts, true
			}
		}
	}
	return nil, false
}

func localVariableType(p *Program, topBlock, block *Block, absidx int) Type {
	// TODO: Rename this function (truly-local variable?)
	var ts []Type
	for _, f := range p.Funcs {
		if f.Block == topBlock {
			ts = append(f.InParams, f.OutParams...)
			break
		}
	}

	ts2, _ := descendantLocalVars(topBlock, block)
	ts = append(ts, ts2...)
	return ts[absidx]
}

func (p *Program) LocalVariableType(topBlock, block *Block, idx int) Type {
	switch topBlock {
	case p.VertexFunc.Block:
		na := len(p.Attributes)
		nv := len(p.Varyings)
		switch {
		case idx < na:
			return p.Attributes[idx]
		case idx == na:
			return Type{Main: Vec4}
		case idx < na+nv+1:
			return p.Varyings[idx-na-1]
		default:
			return localVariableType(p, topBlock, block, idx-(na+nv+1))
		}
	case p.FragmentFunc.Block:
		nv := len(p.Varyings)
		switch {
		case idx == 0:
			return Type{Main: Vec4}
		case idx < nv+1:
			return p.Varyings[idx-1]
		default:
			return localVariableType(p, topBlock, block, idx-(nv+1))
		}
	default:
		return localVariableType(p, topBlock, block, idx)
	}
}
