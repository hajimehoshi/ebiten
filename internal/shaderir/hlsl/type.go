// Copyright 2022 The Ebiten Authors
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

package hlsl

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func opString(op shaderir.Op) string {
	switch op {
	case shaderir.Add:
		return "+"
	case shaderir.Sub:
		return "-"
	case shaderir.NotOp:
		return "!"
	case shaderir.ComponentWiseMul:
		return "*"
	case shaderir.Div:
		return "/"
	case shaderir.ModOp:
		return "%"
	case shaderir.LeftShift:
		return "<<"
	case shaderir.RightShift:
		return ">>"
	case shaderir.LessThanOp:
		return "<"
	case shaderir.LessThanEqualOp:
		return "<="
	case shaderir.GreaterThanOp:
		return ">"
	case shaderir.GreaterThanEqualOp:
		return ">="
	case shaderir.EqualOp:
		return "=="
	case shaderir.NotEqualOp:
		return "!="
	case shaderir.And:
		return "&"
	case shaderir.Xor:
		return "^"
	case shaderir.Or:
		return "|"
	case shaderir.AndAnd:
		return "&&"
	case shaderir.OrOr:
		return "||"
	}
	return fmt.Sprintf("?(unexpected operator: %d)", op)
}

func typeString(t *shaderir.Type) (string, string) {
	switch t.Main {
	case shaderir.Array:
		t0, t1 := typeString(&t.Sub[0])
		return t0 + t1, fmt.Sprintf("[%d]", t.Length)
	case shaderir.Struct:
		panic("hlsl: a struct is not implemented")
	default:
		return basicTypeString(t.Main), ""
	}
}

func basicTypeString(t shaderir.BasicType) string {
	switch t {
	case shaderir.None:
		return "?(none)"
	case shaderir.Bool:
		return "bool"
	case shaderir.Int:
		return "int"
	case shaderir.Float:
		return "float"
	case shaderir.Vec2:
		return "float2"
	case shaderir.Vec3:
		return "float3"
	case shaderir.Vec4:
		return "float4"
	case shaderir.IVec2:
		return "int2"
	case shaderir.IVec3:
		return "int3"
	case shaderir.IVec4:
		return "int4"
	case shaderir.Mat2:
		return "float2x2"
	case shaderir.Mat3:
		return "float3x3"
	case shaderir.Mat4:
		return "float4x4"
	case shaderir.Array:
		return "?(array)"
	case shaderir.Struct:
		return "?(struct)"
	default:
		return fmt.Sprintf("?(unknown type: %d)", t)
	}
}

func (c *compileContext) builtinFuncString(f shaderir.BuiltinFunc) string {
	switch f {
	case shaderir.Vec2F:
		return "float2"
	case shaderir.Vec3F:
		return "float3"
	case shaderir.Vec4F:
		return "float4"
	case shaderir.IVec2F:
		return "int2"
	case shaderir.IVec3F:
		return "int3"
	case shaderir.IVec4F:
		return "int4"
	case shaderir.Mat2F:
		return "float2x2"
	case shaderir.Mat3F:
		return "float3x3"
	case shaderir.Mat4F:
		return "float4x4"
	case shaderir.Inversesqrt:
		return "rsqrt"
	case shaderir.Fract:
		return "frac"
	case shaderir.Mix:
		return "lerp"
	case shaderir.Dfdx:
		return "ddx"
	case shaderir.Dfdy:
		return "ddy"
	case shaderir.TexelAt:
		return "?(__texelAt)"
	default:
		return string(f)
	}
}
