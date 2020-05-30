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

func (t *Type) serialize() string {
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
	case Texture2D:
		return "sampler2D"
	case Array:
		return fmt.Sprintf("%s[%d]", t.Sub[0].serialize(), t.Length)
	case Struct:
		str := "struct{"
		sub := make([]string, 0, len(t.Sub))
		for _, st := range t.Sub {
			sub = append(sub, st.serialize())
		}
		str += strings.Join(sub, ",")
		str += "}"
		return str
	default:
		return fmt.Sprintf("?(unknown type: %d)", t)
	}
}

type BasicType int

// For a texture, a name Texture2D is used instead of Sampler2D. A sampler is a combination of a texture and how to
// get its texel (a fitlter or an address), while a texture is just a 2D byte array. In Ebiten, a sampler in GLSL
// always uses the same filter (nearest) and address (clamp-to-zero), then the name Texture2D is adopted.

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
	Texture2D
	Array
	Struct
)

func (t BasicType) Glsl() string {
	switch t {
	case None:
		return "?(none)"
	case Bool:
		return "bool"
	case Int:
		return "Int"
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
	case Texture2D:
		return "sampler2D"
	case Array:
		// First-class array is not available on GLSL ES 2.
		return "?(array)"
	case Struct:
		return "?(struct)"
	default:
		return fmt.Sprintf("?(unknown type: %d)", t)
	}
}
