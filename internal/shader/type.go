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
	"strings"
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
	basicTypeStruct
)

type structMember struct {
	name string
	typ  typ
	tag  string
}

type typ struct {
	basic         basicType
	name          string
	structMembers []structMember
}

func (t *typ) isNone() bool {
	return t.basic == basicTypeNone
}

func (sh *Shader) parseType(expr ast.Expr) typ {
	switch t := expr.(type) {
	case *ast.Ident:
		switch t.Name {
		case "float":
			return typ{
				basic: basicTypeFloat,
			}
		case "vec2":
			return typ{
				basic: basicTypeVec2,
			}
		case "vec3":
			return typ{
				basic: basicTypeVec3,
			}
		case "vec4":
			return typ{
				basic: basicTypeVec4,
			}
		case "mat2":
			return typ{
				basic: basicTypeMat2,
			}
		case "mat3":
			return typ{
				basic: basicTypeMat3,
			}
		case "mat4":
			return typ{
				basic: basicTypeMat4,
			}
		case "sampler2d":
			return typ{
				basic: basicTypeSampler2d,
			}
		default:
			sh.addError(t.Pos(), fmt.Sprintf("unexpected type: %s", t.Name))
			return typ{}
		}
	case *ast.StructType:
		str := typ{
			basic: basicTypeStruct,
		}
		for _, f := range t.Fields.List {
			typ := sh.parseType(f.Type)
			var tag string
			if f.Tag != nil {
				tag = f.Tag.Value
			}
			for _, n := range f.Names {
				str.structMembers = append(str.structMembers, structMember{
					name: n.Name,
					typ:  typ,
					tag:  tag,
				})
			}
		}
		return str
	default:
		sh.addError(t.Pos(), fmt.Sprintf("unepxected type: %v", t))
		return typ{}
	}
}

func (t typ) dump(indent int) []string {
	idt := strings.Repeat("\t", indent)

	switch t.basic {
	case basicTypeStruct:
		ls := []string{
			fmt.Sprintf("%sstruct {", idt),
		}
		for _, m := range t.structMembers {
			ls = append(ls, fmt.Sprintf("%s\t%s %s", idt, m.name, m.typ))
		}
		ls = append(ls, fmt.Sprintf("%s}", idt))
		return ls
	default:
		return []string{t.basic.String()}
	}
}

func (t typ) String() string {
	if t.name != "" {
		return t.name
	}
	return t.basic.String()
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
	case basicTypeStruct:
		return "(struct)"
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
