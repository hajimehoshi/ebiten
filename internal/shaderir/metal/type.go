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

package metal

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

func typeString(t *shaderir.Type, packed bool, ref bool) string {
	switch t.Main {
	case shaderir.Array:
		st := typeString(&t.Sub[0], packed, false)
		t := fmt.Sprintf("array<%s, %d>", st, t.Length)
		if ref {
			t += "&"
		}
		return t
	case shaderir.Struct:
		panic("metal: a struct is not implemented")
	default:
		t := basicTypeString(t.Main, packed)
		if ref {
			t += "&"
		}
		return t
	}
}

func basicTypeString(t shaderir.BasicType, packed bool) string {
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
		if packed {
			return "packed_float2"
		}
		return "float2"
	case shaderir.Vec3:
		if packed {
			return "packed_float3"
		}
		return "float3"
	case shaderir.Vec4:
		if packed {
			return "packed_float4"
		}
		return "float4"
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

func builtinFuncString(f shaderir.BuiltinFunc) string {
	switch f {
	case shaderir.BoolF:
		return "static_cast<bool>"
	case shaderir.IntF:
		return "static_cast<int>"
	case shaderir.FloatF:
		return "static_cast<float>"
	case shaderir.Vec2F:
		return "float2"
	case shaderir.Vec3F:
		return "float3"
	case shaderir.Vec4F:
		return "float4"
	case shaderir.Mat2F:
		return "float2x2"
	case shaderir.Mat3F:
		return "float3x3"
	case shaderir.Mat4F:
		return "float4x4"
	case shaderir.Inversesqrt:
		return "rsqrt"
	case shaderir.Mod:
		return "fmod"
	case shaderir.Texture2DF:
		return "?(texture2D)"
	}
	return string(f)
}
