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
	args []variable
	rets []variable
	body *block
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

	// TODO: Resolve identifiers?
	// TODO: Resolve constants

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
					vs := sh.parseVariable(s)
					for _, v := range vs {
						if 'A' <= v.name[0] && v.name[0] <= 'Z' {
							sh.uniforms = append(sh.uniforms, v)
						} else {
							sh.globals = append(sh.globals, v)
						}
					}
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

func (s *Shader) parseVariable(vs *ast.ValueSpec) []variable {
	var t typ
	if vs.Type != nil {
		var err error
		t, err = parseType(vs.Type)
		if err != nil {
			s.addError(vs.Type.Pos(), err.Error())
			return nil
		}
	}

	var vars []variable
	for _, n := range vs.Names {
		name := n.Name
		vars = append(vars, variable{
			name: name,
			typ:  t,
		})
	}
	return vars
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
			init = v.Value // TODO: This should be go/constant.Value
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

	var args []variable
	for _, f := range d.Type.Params.List {
		t, err := parseType(f.Type)
		if err != nil {
			sh.addError(f.Type.Pos(), err.Error())
			continue
		}
		for _, n := range f.Names {
			args = append(args, variable{
				name: n.Name,
				typ:  t,
			})
		}
	}

	var rets []variable
	for _, f := range d.Type.Results.List {
		t, err := parseType(f.Type)
		if err != nil {
			sh.addError(f.Type.Pos(), err.Error())
			continue
		}
		if len(f.Names) == 0 {
			rets = append(rets, variable{
				name: "",
				typ:  t,
			})
		} else {
			for _, n := range f.Names {
				rets = append(rets, variable{
					name: n.Name,
					typ:  t,
				})
			}
		}
	}

	f := function{
		name: d.Name.Name,
		args: args,
		rets: rets,
		body: sh.parseBlock(d.Body),
	}
	sh.funcs = append(sh.funcs, f)
}

func (sh *Shader) parseBlock(b *ast.BlockStmt) *block {
	block := &block{}

	for _, l := range b.List {
		switch l := l.(type) {
		case *ast.AssignStmt:
			if l.Tok == token.DEFINE {
				for _, s := range l.Lhs {
					ident := s.(*ast.Ident)
					block.vars = append(block.vars, variable{
						name: ident.Name,
					})
				}
			} else {
				// TODO
			}
		case *ast.DeclStmt:
			switch d := l.Decl.(type) {
			case *ast.GenDecl:
				switch d.Tok {
				case token.VAR:
					for _, s := range d.Specs {
						s := s.(*ast.ValueSpec)
						vs := sh.parseVariable(s)
						block.vars = append(block.vars, vs...)
					}
				}
			}
		case *ast.ReturnStmt:
			var exprs []expr
			for _, r := range l.Results {
				exprs = append(exprs, sh.parseExpr(r))
			}
			block.stmts = append(block.stmts, stmt{
				stmtType: stmtReturn,
				exprs:    exprs,
			})
		default:
		}
	}

	sort.Slice(block.vars, func(a, b int) bool {
		return block.vars[a].name < block.vars[b].name
	})

	return block
}

func (sh *Shader) parseExpr(e ast.Expr) expr {
	switch e := e.(type) {
	case *ast.Ident:
		return expr{
			exprType: exprIdent,
			value:    e.Name,
		}
	}
	return expr{}
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
		var args []string
		for _, a := range f.args {
			args = append(args, fmt.Sprintf("%s %s", a.name, a.typ))
		}
		var rets []string
		for _, r := range f.rets {
			name := r.name
			if name == "" {
				name = "_"
			}
			rets = append(rets, fmt.Sprintf("%s %s", name, r.typ))
		}
		l := fmt.Sprintf("func %s(%s)", f.name, strings.Join(args, ", "))
		if len(rets) > 0 {
			l += " (" + strings.Join(rets, ", ") + ")"
		}
		l += " {"
		lines = append(lines, l)
		lines = append(lines, f.body.dump(1)...)
		lines = append(lines, "}")
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
