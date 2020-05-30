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

	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

// TODO: What about array types?

type typ struct {
	ir   shaderir.Type
	name string
}

func (cs *compileState) parseType(expr ast.Expr) typ {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "bool":
			return typ{
				ir: shaderir.Type{Main: shaderir.Bool},
			}
		case "int":
			return typ{
				ir: shaderir.Type{Main: shaderir.Int},
			}
		case "float":
			return typ{
				ir: shaderir.Type{Main: shaderir.Float},
			}
		case "vec2":
			return typ{
				ir: shaderir.Type{Main: shaderir.Vec2},
			}
		case "vec3":
			return typ{
				ir: shaderir.Type{Main: shaderir.Vec3},
			}
		case "vec4":
			return typ{
				ir: shaderir.Type{Main: shaderir.Vec4},
			}
		case "mat2":
			return typ{
				ir: shaderir.Type{Main: shaderir.Mat2},
			}
		case "mat3":
			return typ{
				ir: shaderir.Type{Main: shaderir.Mat3},
			}
		case "mat4":
			return typ{
				ir: shaderir.Type{Main: shaderir.Mat4},
			}
		case "texture2d":
			return typ{
				ir: shaderir.Type{Main: shaderir.Texture2D},
			}
		default:
			cs.addError(t.Pos(), fmt.Sprintf("unexpected type: %s", t.Name))
			return typ{}
		}
	case *ast.StructType:
		cs.addError(t.Pos(), "struct is not implemented")
		return typ{}
	default:
		cs.addError(t.Pos(), fmt.Sprintf("unepxected type: %v", t))
		return typ{}
	}
}
