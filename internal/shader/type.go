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

type typ int

// TODO: What about array types?

const (
	typBool typ = iota
	typInt
	typFloat
	typVec2
	typVec3
	typVec4
	typMat2
	typMat3
	typMat4
	typSampler2d
)

func parseType(expr ast.Expr) (typ, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "bool":
			return typBool, nil
		case "int":
			return typInt, nil
		case "float":
			return typFloat, nil
		case "vec2":
			return typVec2, nil
		case "vec3":
			return typVec3, nil
		case "vec4":
			return typVec4, nil
		case "mat2":
			return typMat2, nil
		case "mat3":
			return typMat3, nil
		case "mat4":
			return typMat4, nil
		case "sampler2d":
			return typSampler2d, nil
		}
		// TODO: Parse array types
	}
	return 0, fmt.Errorf("invalid type: %s", expr)
}

func (t typ) String() string {
	switch t {
	case typBool:
		return "bool"
	case typInt:
		return "int"
	case typFloat:
		return "float"
	case typVec2:
		return "vec2"
	case typVec3:
		return "vec3"
	case typVec4:
		return "vec4"
	case typMat2:
		return "mat2"
	case typMat3:
		return "mat3"
	case typMat4:
		return "mat4"
	case typSampler2d:
		return "sampler2d"
	default:
		return fmt.Sprintf("unknown(%d)", t)
	}
}

func (t typ) numeric() bool {
	return t != typSampler2d
}

func (t typ) glslString() string {
	switch t {
	case typBool:
		return "bool"
	case typInt:
		return "int"
	case typFloat:
		return "float"
	case typVec2:
		return "vec2"
	case typVec3:
		return "vec3"
	case typVec4:
		return "vec4"
	case typMat2:
		return "mat2"
	case typMat3:
		return "mat3"
	case typMat4:
		return "mat4"
	case typSampler2d:
		return "?(sampler2d)"
	default:
		return fmt.Sprintf("?(%d)", t)
	}
}
