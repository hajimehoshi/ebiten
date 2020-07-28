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

func (t *Type) Equal(rhs *Type) bool {
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

func (t *Type) String() string {
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
	case Mat2:
		return "mat2"
	case Mat3:
		return "mat3"
	case Mat4:
		return "mat4"
	case Array:
		return fmt.Sprintf("%s[%d]", t.Sub[0].String(), t.Length)
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
		return fmt.Sprintf("?(unknown type: %d)", t)
	}
}

func (t *Type) serialize() string {
	return t.String()
}

func (t *Type) Glsl() string {
	switch t.Main {
	case Array:
		return fmt.Sprintf("%s[%d]", t.Sub[0].Glsl(), t.Length)
	case Struct:
		panic("shaderir: a struct is not implemented")
	default:
		return t.Main.glsl()
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
	Mat2
	Mat3
	Mat4
	Array
	Struct
)

func (t BasicType) glsl() string {
	switch t {
	case None:
		return "?(none)"
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
	case Mat2:
		return "mat2"
	case Mat3:
		return "mat3"
	case Mat4:
		return "mat4"
	case Array:
		return "?(array)"
	case Struct:
		return "?(struct)"
	default:
		return fmt.Sprintf("?(unknown type: %d)", t)
	}
}
