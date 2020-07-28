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
	gconstant "go/constant"

	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

func (cs *compileState) parseType(block *block, expr ast.Expr) shaderir.Type {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "bool":
			return shaderir.Type{Main: shaderir.Bool}
		case "int":
			return shaderir.Type{Main: shaderir.Int}
		case "float":
			return shaderir.Type{Main: shaderir.Float}
		case "vec2":
			return shaderir.Type{Main: shaderir.Vec2}
		case "vec3":
			return shaderir.Type{Main: shaderir.Vec3}
		case "vec4":
			return shaderir.Type{Main: shaderir.Vec4}
		case "mat2":
			return shaderir.Type{Main: shaderir.Mat2}
		case "mat3":
			return shaderir.Type{Main: shaderir.Mat3}
		case "mat4":
			return shaderir.Type{Main: shaderir.Mat4}
		default:
			cs.addError(t.Pos(), fmt.Sprintf("unexpected type: %s", t.Name))
			return shaderir.Type{}
		}
	case *ast.ArrayType:
		if t.Len == nil {
			cs.addError(t.Pos(), fmt.Sprintf("array length must be specified"))
			return shaderir.Type{}
		}
		// TODO: Parse ellipsis

		exprs, _, _, ok := cs.parseExpr(block, t.Len)
		if !ok {
			return shaderir.Type{}
		}
		if len(exprs) != 1 {
			cs.addError(t.Pos(), fmt.Sprintf("invalid length of array"))
			return shaderir.Type{}
		}
		if exprs[0].Type != shaderir.NumberExpr {
			cs.addError(t.Pos(), fmt.Sprintf("length of array must be a constant number"))
			return shaderir.Type{}
		}
		len, ok := gconstant.Int64Val(exprs[0].Const)
		if !ok {
			cs.addError(t.Pos(), fmt.Sprintf("length of array must be an integer"))
			return shaderir.Type{}
		}

		elm := cs.parseType(block, t.Elt)
		if elm.Main == shaderir.Array {
			cs.addError(t.Pos(), fmt.Sprintf("array of array is forbidden"))
			return shaderir.Type{}
		}
		return shaderir.Type{
			Main:   shaderir.Array,
			Sub:    []shaderir.Type{elm},
			Length: int(len),
		}
	case *ast.StructType:
		cs.addError(t.Pos(), "struct is not implemented")
		return shaderir.Type{}
	default:
		cs.addError(t.Pos(), fmt.Sprintf("unepxected type: %v", t))
		return shaderir.Type{}
	}
}
