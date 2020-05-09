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
	typNone typ = iota
	typFloat
	typVec2
	typVec3
	typVec4
	typMat2
	typMat3
	typMat4
	typSampler2d
)

func parseType(expr ast.Expr) typ {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "float":
			return typFloat
		case "vec2":
			return typVec2
		case "vec3":
			return typVec3
		case "vec4":
			return typVec4
		case "mat2":
			return typMat2
		case "mat3":
			return typMat3
		case "mat4":
			return typMat4
		case "sampler2d":
			return typSampler2d
		}
		// TODO: Parse array types
	}
	return typNone
}

func (t typ) String() string {
	switch t {
	case typNone:
		return "(none)"
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
	return t != typNone && t != typSampler2d
}

func (t typ) glslString() string {
	switch t {
	case typNone:
		return "?(none)"
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
