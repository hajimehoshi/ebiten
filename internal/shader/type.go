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

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func (cs *compileState) parseType(block *block, expr ast.Expr) (shaderir.Type, bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "bool":
			return shaderir.Type{Main: shaderir.Bool}, true
		case "int":
			return shaderir.Type{Main: shaderir.Int}, true
		case "float":
			return shaderir.Type{Main: shaderir.Float}, true
		case "vec2":
			return shaderir.Type{Main: shaderir.Vec2}, true
		case "vec3":
			return shaderir.Type{Main: shaderir.Vec3}, true
		case "vec4":
			return shaderir.Type{Main: shaderir.Vec4}, true
		case "mat2":
			return shaderir.Type{Main: shaderir.Mat2}, true
		case "mat3":
			return shaderir.Type{Main: shaderir.Mat3}, true
		case "mat4":
			return shaderir.Type{Main: shaderir.Mat4}, true
		default:
			cs.addError(t.Pos(), fmt.Sprintf("unexpected type: %s", t.Name))
			return shaderir.Type{}, false
		}
	case *ast.ArrayType:
		if t.Len == nil {
			cs.addError(t.Pos(), fmt.Sprintf("array length must be specified"))
			return shaderir.Type{}, false
		}
		var length int
		if _, ok := t.Len.(*ast.Ellipsis); ok {
			length = -1 // Determine the length later.
		} else {
			exprs, _, _, ok := cs.parseExpr(block, t.Len, true)
			if !ok {
				return shaderir.Type{}, false
			}
			if len(exprs) != 1 {
				cs.addError(t.Pos(), fmt.Sprintf("invalid length of array"))
				return shaderir.Type{}, false
			}
			if exprs[0].Type != shaderir.NumberExpr {
				cs.addError(t.Pos(), fmt.Sprintf("length of array must be a constant number"))
				return shaderir.Type{}, false
			}
			l, ok := gconstant.Int64Val(exprs[0].Const)
			if !ok {
				cs.addError(t.Pos(), fmt.Sprintf("length of array must be an integer"))
				return shaderir.Type{}, false
			}
			length = int(l)
		}

		elm, ok := cs.parseType(block, t.Elt)
		if !ok {
			return shaderir.Type{}, false
		}
		if elm.Main == shaderir.Array {
			cs.addError(t.Pos(), fmt.Sprintf("array of array is forbidden"))
			return shaderir.Type{}, false
		}
		return shaderir.Type{
			Main:   shaderir.Array,
			Sub:    []shaderir.Type{elm},
			Length: length,
		}, true
	case *ast.StructType:
		cs.addError(t.Pos(), "struct is not implemented")
		return shaderir.Type{}, false
	default:
		cs.addError(t.Pos(), fmt.Sprintf("unepxected type: %v", t))
		return shaderir.Type{}, false
	}
}

func canBeFloatImplicitly(expr shaderir.Expr, t shaderir.Type) bool {
	// TODO: For integers, should only constants be allowed?
	if t.Main == shaderir.Int {
		return true
	}
	if t.Main == shaderir.Float {
		return true
	}
	if expr.Const != nil {
		if expr.Const.Kind() == gconstant.Int {
			return true
		}
		if expr.Const.Kind() == gconstant.Float {
			return true
		}
	}
	return false
}

func checkArgsForVec2BuiltinFunc(args []shaderir.Expr, argts []shaderir.Type) error {
	if len(args) != len(argts) {
		return fmt.Errorf("the number of arguments and types doesn't match: %d vs %d", len(args), len(argts))
	}

	switch len(args) {
	case 1:
		if canBeFloatImplicitly(args[0], argts[0]) {
			return nil
		}
		if argts[0].Main == shaderir.Vec2 {
			return nil
		}
		return fmt.Errorf("invalid arguments for vec2: (%s)", argts[0].String())
	case 2:
		if canBeFloatImplicitly(args[0], argts[0]) && canBeFloatImplicitly(args[1], argts[1]) {
			return nil
		}
		return fmt.Errorf("invalid arguments for vec2: (%s, %s)", argts[0].String(), argts[1].String())
	default:
		return fmt.Errorf("too many arguments for vec2")
	}
}

func checkArgsForVec3BuiltinFunc(args []shaderir.Expr, argts []shaderir.Type) error {
	if len(args) != len(argts) {
		return fmt.Errorf("the number of arguments and types doesn't match: %d vs %d", len(args), len(argts))
	}

	switch len(args) {
	case 1:
		if canBeFloatImplicitly(args[0], argts[0]) {
			return nil
		}
		if argts[0].Main == shaderir.Vec3 {
			return nil
		}
		return fmt.Errorf("invalid arguments for vec3: (%s)", argts[0].String())
	case 2:
		if canBeFloatImplicitly(args[0], argts[0]) && argts[1].Main == shaderir.Vec2 {
			return nil
		}
		if argts[0].Main == shaderir.Vec2 && canBeFloatImplicitly(args[1], argts[1]) {
			return nil
		}
		return fmt.Errorf("invalid arguments for vec3: (%s, %s)", argts[0].String(), argts[1].String())
	case 3:
		if canBeFloatImplicitly(args[0], argts[0]) && canBeFloatImplicitly(args[1], argts[1]) && canBeFloatImplicitly(args[2], argts[2]) {
			return nil
		}
		return fmt.Errorf("invalid arguments for vec3: (%s, %s, %s)", argts[0].String(), argts[1].String(), argts[2].String())
	default:
		return fmt.Errorf("too many arguments for vec3")
	}
}

func checkArgsForVec4BuiltinFunc(args []shaderir.Expr, argts []shaderir.Type) error {
	if len(args) != len(argts) {
		return fmt.Errorf("the number of arguments and types doesn't match: %d vs %d", len(args), len(argts))
	}

	switch len(args) {
	case 1:
		if canBeFloatImplicitly(args[0], argts[0]) {
			return nil
		}
		if argts[0].Main == shaderir.Vec4 {
			return nil
		}
		return fmt.Errorf("invalid arguments for vec4: (%s)", argts[0].String())
	case 2:
		if canBeFloatImplicitly(args[0], argts[0]) && argts[1].Main == shaderir.Vec3 {
			return nil
		}
		if argts[0].Main == shaderir.Vec2 && argts[1].Main == shaderir.Vec2 {
			return nil
		}
		if argts[0].Main == shaderir.Vec3 && canBeFloatImplicitly(args[1], argts[1]) {
			return nil
		}
		return fmt.Errorf("invalid arguments for vec4: (%s, %s)", argts[0].String(), argts[1].String())
	case 3:
		if canBeFloatImplicitly(args[0], argts[0]) && canBeFloatImplicitly(args[1], argts[1]) && argts[2].Main == shaderir.Vec2 {
			return nil
		}
		if canBeFloatImplicitly(args[0], argts[0]) && argts[1].Main == shaderir.Vec2 && canBeFloatImplicitly(args[2], argts[2]) {
			return nil
		}
		if argts[0].Main == shaderir.Vec2 && canBeFloatImplicitly(args[1], argts[1]) && canBeFloatImplicitly(args[2], argts[2]) {
			return nil
		}
		return fmt.Errorf("invalid arguments for vec4: (%s, %s, %s)", argts[0].String(), argts[1].String(), argts[2].String())
	case 4:
		if canBeFloatImplicitly(args[0], argts[0]) && canBeFloatImplicitly(args[1], argts[1]) && canBeFloatImplicitly(args[2], argts[2]) && canBeFloatImplicitly(args[3], argts[3]) {
			return nil
		}
		return fmt.Errorf("invalid arguments for vec4: (%s, %s, %s, %s)", argts[0].String(), argts[1].String(), argts[2].String(), argts[3].String())
	default:
		return fmt.Errorf("too many arguments for vec4")
	}
}
