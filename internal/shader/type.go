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

package shader

import (
	"fmt"
	"go/ast"
)

type basicType int

// TODO: What about array types?

const (
	basicTypeNone basicType = iota
	basicTypeFloat
	basicTypeVec2
	basicTypeVec3
	basicTypeVec4
	basicTypeMat2
	basicTypeMat3
	basicTypeMat4
	basicTypeSampler2d
)

func (s *Shader) parseType(expr ast.Expr) basicType {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "float":
			return basicTypeFloat
		case "vec2":
			return basicTypeVec2
		case "vec3":
			return basicTypeVec3
		case "vec4":
			return basicTypeVec4
		case "mat2":
			return basicTypeMat2
		case "mat3":
			return basicTypeMat3
		case "mat4":
			return basicTypeMat4
		case "sampler2d":
			return basicTypeSampler2d
		default:
			s.addError(t.Pos(), fmt.Sprintf("unexpected type: %s", t.Name))
		}
	}
	return basicTypeNone
}

func (t basicType) String() string {
	switch t {
	case basicTypeNone:
		return "(none)"
	case basicTypeFloat:
		return "float"
	case basicTypeVec2:
		return "vec2"
	case basicTypeVec3:
		return "vec3"
	case basicTypeVec4:
		return "vec4"
	case basicTypeMat2:
		return "mat2"
	case basicTypeMat3:
		return "mat3"
	case basicTypeMat4:
		return "mat4"
	case basicTypeSampler2d:
		return "sampler2d"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

func (t basicType) numeric() bool {
	return t != basicTypeNone && t != basicTypeSampler2d
}

func (t basicType) glslString() string {
	switch t {
	case basicTypeNone:
		return "?(none)"
	case basicTypeFloat:
		return "float"
	case basicTypeVec2:
		return "vec2"
	case basicTypeVec3:
		return "vec3"
	case basicTypeVec4:
		return "vec4"
	case basicTypeMat2:
		return "mat2"
	case basicTypeMat3:
		return "mat3"
	case basicTypeMat4:
		return "mat4"
	case basicTypeSampler2d:
		return "?(sampler2d)"
	default:
		return fmt.Sprintf("?(%d)", t)
	}
}
