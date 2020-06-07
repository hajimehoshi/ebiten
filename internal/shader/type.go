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
	"go/token"

	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

// TODO: What about array types?

func (cs *compileState) parseType(expr ast.Expr) shaderir.Type {
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
		case "texture2d":
			return shaderir.Type{Main: shaderir.Texture2D}
		default:
			cs.addError(t.Pos(), fmt.Sprintf("unexpected type: %s", t.Name))
			return shaderir.Type{}
		}
	case *ast.StructType:
		cs.addError(t.Pos(), "struct is not implemented")
		return shaderir.Type{}
	default:
		cs.addError(t.Pos(), fmt.Sprintf("unepxected type: %v", t))
		return shaderir.Type{}
	}
}

func (cs *compileState) detectType(b *block, expr ast.Expr) []shaderir.Type {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.FLOAT:
			return []shaderir.Type{{Main: shaderir.Float}}
		case token.INT:
			return []shaderir.Type{{Main: shaderir.Int}}
		}
		cs.addError(expr.Pos(), fmt.Sprintf("unexpected literal: %s", e.Value))
		return nil
	case *ast.CallExpr:
		n := e.Fun.(*ast.Ident).Name
		f, ok := shaderir.ParseBuiltinFunc(n)
		if ok {
			switch f {
			case shaderir.Vec2F:
				return []shaderir.Type{{Main: shaderir.Vec2}}
			case shaderir.Vec3F:
				return []shaderir.Type{{Main: shaderir.Vec3}}
			case shaderir.Vec4F:
				return []shaderir.Type{{Main: shaderir.Vec4}}
			case shaderir.Mat2F:
				return []shaderir.Type{{Main: shaderir.Mat2}}
			case shaderir.Mat3F:
				return []shaderir.Type{{Main: shaderir.Mat3}}
			case shaderir.Mat4F:
				return []shaderir.Type{{Main: shaderir.Mat4}}
				// TODO: Add more functions
			}
		}
		cs.addError(expr.Pos(), fmt.Sprintf("unexpected call: %s", n))
		return nil
	case *ast.CompositeLit:
		return []shaderir.Type{cs.parseType(e.Type)}
	case *ast.Ident:
		n := e.Name
		for _, v := range b.vars {
			if v.name == n {
				return []shaderir.Type{v.typ}
			}
		}
		if b == &cs.global {
			for i, v := range cs.uniforms {
				if v == n {
					return []shaderir.Type{cs.ir.Uniforms[i]}
				}
			}
		}
		if b.outer != nil {
			return cs.detectType(b.outer, e)
		}
		cs.addError(expr.Pos(), fmt.Sprintf("unexpected identifier: %s", n))
		return nil
	//case *ast.SelectorExpr:
	//return fmt.Sprintf("%cs.%s", dumpExpr(e.X), dumpExpr(e.Sel))
	default:
		cs.addError(expr.Pos(), fmt.Sprintf("detecting type not implemented: %#v", expr))
		return nil
	}
}
