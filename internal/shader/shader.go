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
	name string
	typ  typ
	init ast.Expr
}

type constant struct {
	name string
	typ  typ
	init ast.Expr
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

	global block

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

	// TODO: Make a call graph and reorder the elements.
	return s, nil
}

func (s *Shader) addError(pos token.Pos, str string) {
	p := s.fs.Position(pos)
	s.errs = append(s.errs, fmt.Sprintf("%s: %s", p, str))
}

func (sh *Shader) parse(f *ast.File) {
	for _, d := range f.Decls {
		sh.parseDecl(&sh.global, d)
	}

	vars := make([]variable, len(sh.global.vars))
	copy(vars, sh.global.vars)
	sh.global.vars = nil
	for _, v := range vars {
		if 'A' <= v.name[0] && v.name[0] <= 'Z' {
			sh.uniforms = append(sh.uniforms, v)
		} else {
			sh.global.vars = append(sh.global.vars, v)
		}
	}

	// TODO: This is duplicated with parseBlock.
	sort.Slice(sh.global.consts, func(a, b int) bool {
		return sh.global.consts[a].name < sh.global.consts[b].name
	})
	sort.Slice(sh.global.funcs, func(a, b int) bool {
		return sh.global.funcs[a].name < sh.global.funcs[b].name
	})
	sort.Slice(sh.varyings, func(a, b int) bool {
		return sh.varyings[a].name < sh.varyings[b].name
	})
	sort.Slice(sh.uniforms, func(a, b int) bool {
		return sh.uniforms[a].name < sh.uniforms[b].name
	})
}

func (sh *Shader) parseDecl(b *block, d ast.Decl) {
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
				cs := sh.parseConstant(s)
				b.consts = append(b.consts, cs...)
			}
		case token.VAR:
			for _, s := range d.Specs {
				s := s.(*ast.ValueSpec)
				vs := sh.parseVariable(b, s)
				b.vars = append(b.vars, vs...)
			}
		case token.IMPORT:
			sh.addError(d.Pos(), "import is forbidden")
		default:
			sh.addError(d.Pos(), "unexpected token")
		}
	case *ast.FuncDecl:
		b.funcs = append(b.funcs, sh.parseFunc(d, b))
	default:
		sh.addError(d.Pos(), "unexpected decl")
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
			t := parseType(f.Type)
			if t == typNone {
				sh.addError(f.Type.Pos(), fmt.Sprintf("unexpected type: %s", f.Type))
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
		t := parseType(f.Type)
		if t == typNone {
			sh.addError(f.Type.Pos(), fmt.Sprintf("unexpected type: %s", f.Type))
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

func (s *Shader) parseVariable(block *block, vs *ast.ValueSpec) []variable {
	var t typ
	if vs.Type != nil {
		t = parseType(vs.Type)
		if t == typNone {
			s.addError(vs.Type.Pos(), fmt.Sprintf("unexpected type: %s", vs.Type))
			return nil
		}
	}

	var vars []variable
	for i, n := range vs.Names {
		var init ast.Expr
		if len(vs.Values) > 0 {
			init = vs.Values[i]
			if t == typNone {
				t = s.detectType(block, init)
			}
		}
		name := n.Name
		vars = append(vars, variable{
			name: name,
			typ:  t,
			init: init,
		})
	}
	return vars
}

func (s *Shader) parseConstant(vs *ast.ValueSpec) []constant {
	var t typ
	if vs.Type != nil {
		t = parseType(vs.Type)
		if t == typNone {
			s.addError(vs.Type.Pos(), fmt.Sprintf("unexpected type: %s", vs.Type))
			return nil
		}
	}

	var cs []constant
	for i, n := range vs.Names {
		cs = append(cs, constant{
			name: n.Name,
			typ:  t,
			init: vs.Values[i],
		})
	}
	return cs
}

func (sh *Shader) parseFunc(d *ast.FuncDecl, block *block) function {
	if d.Name == nil {
		sh.addError(d.Pos(), "function must have a name")
		return function{}
	}
	if d.Body == nil {
		sh.addError(d.Pos(), "function must have a body")
		return function{}
	}

	var args []variable
	for _, f := range d.Type.Params.List {
		t := parseType(f.Type)
		if t == typNone {
			sh.addError(f.Type.Pos(), fmt.Sprintf("unexpected type: %s", f.Type))
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
	if d.Type.Results != nil {
		for _, f := range d.Type.Results.List {
			t := parseType(f.Type)
			if t == typNone {
				sh.addError(f.Type.Pos(), fmt.Sprintf("unexpected type: %s", f.Type))
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
	}

	return function{
		name: d.Name.Name,
		args: args,
		rets: rets,
		body: sh.parseBlock(block, d.Body),
	}
}

func (sh *Shader) parseBlock(outer *block, b *ast.BlockStmt) *block {
	block := &block{
		outer: outer,
	}

	for _, l := range b.List {
		switch l := l.(type) {
		case *ast.AssignStmt:
			switch l.Tok {
			case token.DEFINE:
				for i, s := range l.Lhs {
					v := variable{
						name: s.(*ast.Ident).Name,
					}
					if len(l.Rhs) > 0 {
						v.typ = sh.detectType(block, l.Rhs[i])
					}
					block.vars = append(block.vars, v)
				}
				for i := range l.Rhs {
					block.stmts = append(block.stmts, stmt{
						stmtType: stmtAssign,
						exprs:    []ast.Expr{l.Lhs[i], l.Rhs[i]},
					})
				}
			case token.ASSIGN:
				// TODO: What about the statement `a,b = b,a?`
				for i := range l.Rhs {
					block.stmts = append(block.stmts, stmt{
						stmtType: stmtAssign,
						exprs:    []ast.Expr{l.Lhs[i], l.Rhs[i]},
					})
				}
			}
		case *ast.BlockStmt:
			block.stmts = append(block.stmts, stmt{
				stmtType: stmtBlock,
				block:    sh.parseBlock(block, l),
			})
		case *ast.DeclStmt:
			sh.parseDecl(block, l.Decl)
		case *ast.ReturnStmt:
			var exprs []ast.Expr
			for _, r := range l.Results {
				exprs = append(exprs, r)
			}
			block.stmts = append(block.stmts, stmt{
				stmtType: stmtReturn,
				exprs:    exprs,
			})
		}
	}

	sort.Slice(block.consts, func(a, b int) bool {
		return block.consts[a].name < block.consts[b].name
	})
	sort.Slice(block.funcs, func(a, b int) bool {
		return block.funcs[a].name < block.funcs[b].name
	})

	return block
}

func (s *Shader) detectType(b *block, expr ast.Expr) typ {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.FLOAT {
			return typFloat
		}
		if e.Kind == token.INT {
			s.addError(expr.Pos(), fmt.Sprintf("integer literal is not implemented yet: %s", e.Value))
		} else {
			s.addError(expr.Pos(), fmt.Sprintf("unexpected literal: %s", e.Value))
		}
		return typNone
	case *ast.CompositeLit:
		return parseType(e.Type)
	case *ast.Ident:
		n := e.Name
		for _, v := range b.vars {
			if v.name == n {
				return v.typ
			}
		}
		if b.outer != nil {
			return s.detectType(b.outer, e)
		}
		s.addError(expr.Pos(), fmt.Sprintf("unexpected identity: %s", n))
		return typNone
	//case *ast.SelectorExpr:
	//return fmt.Sprintf("%s.%s", dumpExpr(e.X), dumpExpr(e.Sel))
	default:
		s.addError(expr.Pos(), fmt.Sprintf("detecting type not implemented: %#v", expr))
		return typNone
	}
}

// Dump dumps the shader state in an intermediate language.
func (s *Shader) Dump() string {
	var lines []string

	if s.position.name != "" {
		lines = append(lines, fmt.Sprintf("var %s varying %s // position", s.position.name, s.position.typ))
	}
	for _, v := range s.varyings {
		lines = append(lines, fmt.Sprintf("var %s varying %s", v.name, v.typ))
	}

	for _, u := range s.uniforms {
		lines = append(lines, fmt.Sprintf("var %s uniform %s", u.name, u.typ))
	}

	lines = append(lines, s.global.dump(0)...)

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
