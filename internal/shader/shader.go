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
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strings"
)

const (
	varyingStructName = "VertexOut"
)

var (
	kageTagRe = regexp.MustCompile("^`" + `kage:\"(.+)\"` + "`$")
)

type variable struct {
	name     string
	typ      typ
	constant bool
	init     string
}

type function struct {
	name string
}

type Shader struct {
	fs *token.FileSet

	// position is the field name of VertexOut that represents a vertex position (gl_Position in GLSL).
	position variable

	// varyings is a collection of varying variables.
	varyings []variable

	// uniforms is a collection of uniform variables.
	uniforms []variable

	// globals is a collection of global variables.
	globals []variable

	funcs []function

	errs []string
}

type ParseError struct {
	errs []string
}

func (p *ParseError) Error() string {
	return strings.Join(p.errs, "\n")
}

func NewShader(src []byte) (*Shader, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", src, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	s := &Shader{
		fs: fs,
	}
	s.parse(f)

	if len(s.errs) > 0 {
		return nil, &ParseError{s.errs}
	}

	sort.Slice(s.varyings, func(a, b int) bool {
		return s.varyings[a].name < s.varyings[b].name
	})
	sort.Slice(s.uniforms, func(a, b int) bool {
		return s.uniforms[a].name < s.uniforms[b].name
	})
	sort.Slice(s.globals, func(a, b int) bool {
		return s.globals[a].name < s.globals[b].name
	})

	// TODO: Make a call graph and reorder the elements.
	return s, nil
}

func (s *Shader) addError(pos token.Pos, str string) {
	p := s.fs.Position(pos)
	s.errs = append(s.errs, fmt.Sprintf("%s: %s", p, str))
}

func (sh *Shader) parse(f *ast.File) {
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				// TODO: Parse regular structs or other types
				for _, s := range d.Specs {
					s := s.(*ast.TypeSpec)
					if s.Name.Name == varyingStructName {
						sh.parseVaryingStruct(s)
					}
				}
			case token.CONST:
				for _, s := range d.Specs {
					s := s.(*ast.ValueSpec)
					sh.parseTopLevelConstant(s)
				}
			case token.VAR:
				for _, s := range d.Specs {
					s := s.(*ast.ValueSpec)
					sh.parseTopLevelVariable(s)
				}
			case token.IMPORT:
				sh.addError(d.Pos(), "import is forbidden")
			default:
				sh.addError(d.Pos(), "unexpected token")
			}
		case *ast.FuncDecl:
			sh.parseFunc(d)
		default:
			sh.addError(d.Pos(), "unexpected decl")
		}
	}
}

func (sh *Shader) parseVaryingStruct(t *ast.TypeSpec) {
	s, ok := t.Type.(*ast.StructType)
	if !ok {
		sh.addError(t.Type.Pos(), fmt.Sprintf("%s must be a struct but not", t.Name))
		return
	}

	for _, f := range s.Fields.List {
		if f.Tag != nil {
			tag := f.Tag.Value
			m := kageTagRe.FindStringSubmatch(tag)
			if m == nil {
				sh.addError(f.Tag.Pos(), fmt.Sprintf("invalid struct tag: %s", tag))
				continue
			}
			if m[1] != "position" {
				sh.addError(f.Tag.Pos(), fmt.Sprintf("struct tag value must be position in %s but %s", varyingStructName, m[1]))
				continue
			}
			if len(f.Names) != 1 {
				sh.addError(f.Pos(), fmt.Sprintf("position members must be one"))
				continue
			}
			t, err := parseType(f.Type)
			if err != nil {
				sh.addError(f.Type.Pos(), err.Error())
				continue
			}
			if t != typVec4 {
				sh.addError(f.Type.Pos(), fmt.Sprintf("position must be vec4 but %s", t))
				continue
			}
			sh.position = variable{
				name: f.Names[0].Name,
				typ:  t,
			}
			continue
		}
		t, err := parseType(f.Type)
		if err != nil {
			sh.addError(f.Type.Pos(), err.Error())
			continue
		}
		if !t.numeric() {
			sh.addError(f.Type.Pos(), fmt.Sprintf("members in %s must be numeric but %s", varyingStructName, t))
			continue
		}
		for _, n := range f.Names {
			sh.varyings = append(sh.varyings, variable{
				name: n.Name,
				typ:  t,
			})
		}
	}
}

func (s *Shader) parseTopLevelVariable(vs *ast.ValueSpec) {
	t, err := parseType(vs.Type)
	if err != nil {
		s.addError(vs.Type.Pos(), err.Error())
		return
	}
	for _, n := range vs.Names {
		name := n.Name
		val := variable{
			name: name,
			typ:  t,
		}
		// TODO: Parse initial value.
		if 'A' <= name[0] && name[0] <= 'Z' {
			s.uniforms = append(s.uniforms, val)
		} else {
			s.globals = append(s.globals, val)
		}
	}
}

func (s *Shader) parseTopLevelConstant(vs *ast.ValueSpec) {
	t, err := parseType(vs.Type)
	if err != nil {
		s.addError(vs.Type.Pos(), err.Error())
		return
	}
	for i, n := range vs.Names {
		v := vs.Values[i]
		var init string
		switch v := v.(type) {
		case *ast.BasicLit:
			if v.Kind != token.INT && v.Kind != token.FLOAT {
				s.addError(v.Pos(), fmt.Sprintf("literal must be int or float but %s", v.Kind))
				return
			}
			init = v.Value // TODO: This should be math/big.Int or Float.
		default:
			// TODO: Parse the expression.
		}
		name := n.Name
		val := variable{
			name:     name,
			typ:      t, // TODO: Treat consts without types
			constant: true,
			init:     init,
		}
		s.globals = append(s.globals, val)
	}
}

func (sh *Shader) parseFunc(d *ast.FuncDecl) {
	if d.Name == nil {
		sh.addError(d.Pos(), "function must have a name")
		return
	}
	if d.Body == nil {
		sh.addError(d.Pos(), "function must have a body")
		return
	}

	f := function{
		name: d.Name.Name,
	}
	sh.funcs = append(sh.funcs, f)
}

// Dump dumps the shader state in an intermediate language.
func (s *Shader) Dump() string {
	var lines []string

	lines = append(lines, fmt.Sprintf("var %s varying %s // position", s.position.name, s.position.typ))
	for _, v := range s.varyings {
		lines = append(lines, fmt.Sprintf("var %s varying %s", v.name, v.typ))
	}

	for _, u := range s.uniforms {
		lines = append(lines, fmt.Sprintf("var %s uniform %s", u.name, u.typ))
	}

	for _, g := range s.globals {
		prefix := "var"
		if g.constant {
			prefix = "const"
		}
		init := ""
		if g.init != "" {
			init = " = " + g.init
		}
		lines = append(lines, fmt.Sprintf("%s %s %s%s", prefix, g.name, g.typ, init))
	}

	for _, f := range s.funcs {
		lines = append(lines, fmt.Sprintf("func %s", f.name))
	}

	return strings.Join(lines, "\n") + "\n"
}

func (s *Shader) GlslVertex() string {
	var lines []string

	for _, v := range s.varyings {
		// TODO: variable names must be escaped not to conflict with keywords.
		lines = append(lines, fmt.Sprintf("varying %s %s;", v.typ.glslString(), v.name))
	}
	return strings.Join(lines, "\n") + "\n"
}
