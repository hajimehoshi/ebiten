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
	"go/token"
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type variable struct {
	name           string
	typ            shaderir.Type
	forLoopCounter bool
}

type constant struct {
	name  string
	typ   shaderir.Type
	ctyp  shaderir.ConstType
	value gconstant.Value
}

type function struct {
	name  string
	block *block

	ir shaderir.Func
}

type compileState struct {
	fs *token.FileSet

	vertexEntry   string
	fragmentEntry string

	ir shaderir.Program

	funcs []function

	global block

	varyingParsed bool

	errs []string
}

func (cs *compileState) findFunction(name string) (int, bool) {
	for i, f := range cs.funcs {
		if f.name == name {
			return i, true
		}
	}
	return 0, false
}

func (cs *compileState) findUniformVariable(name string) (int, bool) {
	for i, u := range cs.ir.UniformNames {
		if u == name {
			return i, true
		}
	}
	return 0, false
}

type typ struct {
	name string
	ir   shaderir.Type
}

type block struct {
	types      []typ
	vars       []variable
	unusedVars map[int]token.Pos
	consts     []constant
	pos        token.Pos
	outer      *block

	ir *shaderir.Block
}

func (b *block) totalLocalVariableNum() int {
	c := len(b.vars)
	if b.outer != nil {
		c += b.outer.totalLocalVariableNum()
	}
	return c
}

func (b *block) addNamedLocalVariable(name string, typ shaderir.Type, pos token.Pos) {
	b.vars = append(b.vars, variable{
		name: name,
		typ:  typ,
	})
	if name == "_" {
		return
	}
	idx := len(b.vars) - 1
	if b.unusedVars == nil {
		b.unusedVars = map[int]token.Pos{}
	}
	b.unusedVars[idx] = pos
}

func (b *block) findLocalVariable(name string, markLocalVariableUsed bool) (int, shaderir.Type, bool) {
	if name == "" || name == "_" {
		panic("shader: variable name must be non-empty and non-underscore")
	}

	idx := 0
	for outer := b.outer; outer != nil; outer = outer.outer {
		idx += len(outer.vars)
	}
	for i, v := range b.vars {
		if v.name == name {
			if markLocalVariableUsed {
				delete(b.unusedVars, i)
			}
			return idx + i, v.typ, true
		}
	}
	if b.outer != nil {
		return b.outer.findLocalVariable(name, markLocalVariableUsed)
	}
	return 0, shaderir.Type{}, false
}

func (b *block) findLocalVariableByIndex(idx int) (shaderir.Type, bool) {
	bs := []*block{b}
	for outer := b.outer; outer != nil; outer = outer.outer {
		bs = append(bs, outer)
	}
	for i := len(bs) - 1; i >= 0; i-- {
		if len(bs[i].vars) <= idx {
			idx -= len(bs[i].vars)
			continue
		}
		return bs[i].vars[idx].typ, true
	}
	return shaderir.Type{}, false
}

func (b *block) findConstant(name string) (constant, bool) {
	if name == "" || name == "_" {
		panic("shader: constant name must be non-empty and non-underscore")
	}

	for _, c := range b.consts {
		if c.name == name {
			return c, true
		}
	}
	if b.outer != nil {
		return b.outer.findConstant(name)
	}

	return constant{}, false
}

type ParseError struct {
	errs []string
}

func (p *ParseError) Error() string {
	return strings.Join(p.errs, "\n")
}

func Compile(fs *token.FileSet, f *ast.File, vertexEntry, fragmentEntry string, textureNum int) (*shaderir.Program, error) {
	s := &compileState{
		fs:            fs,
		vertexEntry:   vertexEntry,
		fragmentEntry: fragmentEntry,
	}
	s.global.ir = &shaderir.Block{}
	s.parse(f)

	if len(s.errs) > 0 {
		return nil, &ParseError{s.errs}
	}

	// TODO: Resolve identifiers?
	// TODO: Resolve constants

	// TODO: Make a call graph and reorder the elements.

	s.ir.TextureNum = textureNum
	return &s.ir, nil
}

func (s *compileState) addError(pos token.Pos, str string) {
	p := s.fs.Position(pos)
	s.errs = append(s.errs, fmt.Sprintf("%s: %s", p, str))
}

func (cs *compileState) parse(f *ast.File) {
	// Parse GenDecl for global variables, and then parse functions.
	for _, d := range f.Decls {
		if _, ok := d.(*ast.FuncDecl); !ok {
			ss, ok := cs.parseDecl(&cs.global, d)
			if !ok {
				return
			}
			cs.global.ir.Stmts = append(cs.global.ir.Stmts, ss...)
		}
	}

	// Sort the uniform variable so that special variable starting with __ should come first.
	var unames []string
	var utypes []shaderir.Type
	for i, u := range cs.ir.UniformNames {
		if strings.HasPrefix(u, "__") {
			unames = append(unames, u)
			utypes = append(utypes, cs.ir.Uniforms[i])
		}
	}
	// TODO: Check len(unames) == graphics.PreservedUniformVariablesNum. Unfortunately this is not true on tests.
	for i, u := range cs.ir.UniformNames {
		if !strings.HasPrefix(u, "__") {
			unames = append(unames, u)
			utypes = append(utypes, cs.ir.Uniforms[i])
		}
	}
	cs.ir.UniformNames = unames
	cs.ir.Uniforms = utypes

	// Parse function names so that any other function call the others.
	// The function data is provisional and will be updated soon.
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		n := fd.Name.Name
		if n == cs.vertexEntry {
			continue
		}
		if n == cs.fragmentEntry {
			continue
		}

		for _, f := range cs.funcs {
			if f.name == n {
				cs.addError(d.Pos(), fmt.Sprintf("redeclared function: %s", n))
				return
			}
		}

		inParams, outParams := cs.parseFuncParams(&cs.global, fd)
		var inT, outT []shaderir.Type
		for _, v := range inParams {
			inT = append(inT, v.typ)
		}
		for _, v := range outParams {
			outT = append(outT, v.typ)
		}

		cs.funcs = append(cs.funcs, function{
			name: n,
			ir: shaderir.Func{
				Index:     len(cs.funcs),
				InParams:  inT,
				OutParams: outT,
				Block:     &shaderir.Block{},
			},
		})
	}

	// Parse functions.
	for _, d := range f.Decls {
		if _, ok := d.(*ast.FuncDecl); ok {
			ss, ok := cs.parseDecl(&cs.global, d)
			if !ok {
				return
			}
			cs.global.ir.Stmts = append(cs.global.ir.Stmts, ss...)
		}
	}

	if len(cs.errs) > 0 {
		return
	}

	for _, f := range cs.funcs {
		cs.ir.Funcs = append(cs.ir.Funcs, f.ir)
	}
}

func (cs *compileState) parseDecl(b *block, d ast.Decl) ([]shaderir.Stmt, bool) {
	var stmts []shaderir.Stmt

	switch d := d.(type) {
	case *ast.GenDecl:
		switch d.Tok {
		case token.TYPE:
			// TODO: Parse other types
			for _, s := range d.Specs {
				s := s.(*ast.TypeSpec)
				t, ok := cs.parseType(b, s.Type)
				if !ok {
					return nil, false
				}
				b.types = append(b.types, typ{
					name: s.Name.Name,
					ir:   t,
				})
			}
		case token.CONST:
			for _, s := range d.Specs {
				s := s.(*ast.ValueSpec)
				cs, ok := cs.parseConstant(b, s)
				if !ok {
					return nil, false
				}
				b.consts = append(b.consts, cs...)
			}
		case token.VAR:
			for _, s := range d.Specs {
				s := s.(*ast.ValueSpec)
				vs, inits, ss, ok := cs.parseVariable(b, s)
				if !ok {
					return nil, false
				}
				stmts = append(stmts, ss...)
				if b == &cs.global {
					// TODO: Should rhs be ignored?
					for i, v := range vs {
						if !strings.HasPrefix(v.name, "__") {
							if v.name[0] < 'A' || 'Z' < v.name[0] {
								cs.addError(s.Names[i].Pos(), fmt.Sprintf("global variables must be exposed: %s", v.name))
							}
						}
						cs.ir.UniformNames = append(cs.ir.UniformNames, v.name)
						cs.ir.Uniforms = append(cs.ir.Uniforms, v.typ)
					}
					continue
				}

				// base must be obtained before adding the variables.
				base := b.totalLocalVariableNum()
				for _, v := range vs {
					b.addNamedLocalVariable(v.name, v.typ, d.Pos())
				}

				if len(inits) > 0 {
					for i := range vs {
						stmts = append(stmts, shaderir.Stmt{
							Type: shaderir.Assign,
							Exprs: []shaderir.Expr{
								{
									Type:  shaderir.LocalVariable,
									Index: base + i,
								},
								inits[i],
							},
						})
					}
				}
			}
		case token.IMPORT:
			cs.addError(d.Pos(), "import is forbidden")
		default:
			cs.addError(d.Pos(), "unexpected token")
		}
	case *ast.FuncDecl:
		f, ok := cs.parseFunc(b, d)
		if !ok {
			return nil, false
		}
		if b != &cs.global {
			cs.addError(d.Pos(), "non-global function is not implemented")
			return nil, false
		}
		switch d.Name.Name {
		case cs.vertexEntry:
			cs.ir.VertexFunc.Block = f.ir.Block
		case cs.fragmentEntry:
			cs.ir.FragmentFunc.Block = f.ir.Block
		default:
			// The function is already registered for their names.
			for i := range cs.funcs {
				if cs.funcs[i].name == d.Name.Name {
					// Index is already determined by the provisional parsing.
					f.ir.Index = cs.funcs[i].ir.Index
					cs.funcs[i] = f
					break
				}
			}
		}
	default:
		cs.addError(d.Pos(), "unexpected decl")
		return nil, false
	}

	return stmts, true
}

// functionReturnTypes returns the original returning value types, if the given expression is call.
//
// Note that parseExpr returns the returning types for IR, not the original function.
func (cs *compileState) functionReturnTypes(block *block, expr ast.Expr) ([]shaderir.Type, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}

	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return nil, false
	}

	for _, f := range cs.funcs {
		if f.name == ident.Name {
			// TODO: Is it correct to combine out-params and return param?
			ts := f.ir.OutParams
			if f.ir.Return.Main != shaderir.None {
				ts = append(ts, f.ir.Return)
			}
			return ts, true
		}
	}
	return nil, false
}

func (s *compileState) parseVariable(block *block, vs *ast.ValueSpec) ([]variable, []shaderir.Expr, []shaderir.Stmt, bool) {
	if len(vs.Names) != len(vs.Values) && len(vs.Values) != 1 && len(vs.Values) != 0 {
		s.addError(vs.Pos(), fmt.Sprintf("the numbers of lhs and rhs don't match"))
		return nil, nil, nil, false
	}

	var declt shaderir.Type
	if vs.Type != nil {
		var ok bool
		declt, ok = s.parseType(block, vs.Type)
		if !ok {
			return nil, nil, nil, false
		}
	}

	var (
		vars  []variable
		inits []shaderir.Expr
		stmts []shaderir.Stmt
	)

	// These variables are used only in multiple-value context.
	var inittypes []shaderir.Type
	var initexprs []shaderir.Expr

	for i, n := range vs.Names {
		t := declt
		switch {
		case len(vs.Values) == 0:
			// No initialization

		case len(vs.Names) == len(vs.Values):
			// Single-value context

			init := vs.Values[i]

			es, origts, ss, ok := s.parseExpr(block, init, true)
			if !ok {
				return nil, nil, nil, false
			}

			if t.Main == shaderir.None {
				ts, ok := s.functionReturnTypes(block, init)
				if !ok {
					ts = origts
				}
				if len(ts) > 1 {
					s.addError(vs.Pos(), fmt.Sprintf("the numbers of lhs and rhs don't match"))
				}
				t = ts[0]
			}

			if es[0].Type == shaderir.NumberExpr {
				switch t.Main {
				case shaderir.Int:
					es[0].ConstType = shaderir.ConstTypeInt
				case shaderir.Float:
					es[0].ConstType = shaderir.ConstTypeFloat
				}
			}

			inits = append(inits, es...)
			stmts = append(stmts, ss...)

		default:
			// Multiple-value context

			if i == 0 {
				init := vs.Values[0]

				var ss []shaderir.Stmt
				var ok bool
				initexprs, inittypes, ss, ok = s.parseExpr(block, init, true)
				if !ok {
					return nil, nil, nil, false
				}
				stmts = append(stmts, ss...)

				if t.Main == shaderir.None {
					ts, ok := s.functionReturnTypes(block, init)
					if ok {
						inittypes = ts
					}
					if len(ts) != len(vs.Names) {
						s.addError(vs.Pos(), fmt.Sprintf("the numbers of lhs and rhs don't match"))
						continue
					}
				}
			}
			if len(inittypes) > 0 {
				t = inittypes[i]
			}

			// Add the same initexprs for each variable.
			inits = append(inits, initexprs...)
		}

		name := n.Name
		for _, v := range append(block.vars, vars...) {
			if v.name == name {
				s.addError(vs.Pos(), fmt.Sprintf("duplicated local variable name: %s", name))
				return nil, nil, nil, false
			}
		}
		for _, c := range block.consts {
			if c.name == name {
				s.addError(vs.Pos(), fmt.Sprintf("duplicated local constant/variable name: %s", name))
				return nil, nil, nil, false
			}
		}
		vars = append(vars, variable{
			name: name,
			typ:  t,
		})
	}

	return vars, inits, stmts, true
}

func (s *compileState) parseConstant(block *block, vs *ast.ValueSpec) ([]constant, bool) {
	var t shaderir.Type
	if vs.Type != nil {
		var ok bool
		t, ok = s.parseType(block, vs.Type)
		if !ok {
			return nil, false
		}
	}

	var cs []constant
	for i, n := range vs.Names {
		name := n.Name
		for _, c := range block.consts {
			if c.name == name {
				s.addError(vs.Pos(), fmt.Sprintf("duplicated local constant name: %s", name))
				return nil, false
			}
		}
		for _, v := range block.vars {
			if v.name == name {
				s.addError(vs.Pos(), fmt.Sprintf("duplicated local constant/variable name: %s", name))
				return nil, false
			}
		}

		es, ts, ss, ok := s.parseExpr(block, vs.Values[i], false)
		if !ok {
			return nil, false
		}
		if len(ss) > 0 {
			s.addError(vs.Pos(), fmt.Sprintf("invalid constant expression: %s", name))
			return nil, false
		}
		if len(ts) != 1 || len(es) != 1 {
			s.addError(vs.Pos(), fmt.Sprintf("invalid constant expression: %s", n))
			return nil, false
		}
		if es[0].Type != shaderir.NumberExpr {
			s.addError(vs.Pos(), fmt.Sprintf("constant expresion must be a number but not: %s", n))
			return nil, false
		}
		cs = append(cs, constant{
			name:  name,
			typ:   t,
			ctyp:  es[0].ConstType,
			value: es[0].Const,
		})
	}
	return cs, true
}

func (cs *compileState) parseFuncParams(block *block, d *ast.FuncDecl) (in, out []variable) {
	for _, f := range d.Type.Params.List {
		t, ok := cs.parseType(block, f.Type)
		if !ok {
			return
		}
		for _, n := range f.Names {
			in = append(in, variable{
				name: n.Name,
				typ:  t,
			})
		}
	}

	if d.Type.Results == nil {
		return
	}

	for _, f := range d.Type.Results.List {
		t, ok := cs.parseType(block, f.Type)
		if !ok {
			return
		}
		if len(f.Names) == 0 {
			out = append(out, variable{
				name: "",
				typ:  t,
			})
		} else {
			for _, n := range f.Names {
				out = append(out, variable{
					name: n.Name,
					typ:  t,
				})
			}
		}
	}
	return
}

func (cs *compileState) parseFunc(block *block, d *ast.FuncDecl) (function, bool) {
	if d.Name == nil {
		cs.addError(d.Pos(), "function must have a name")
		return function{}, false
	}
	if d.Name.Name == "init" {
		cs.addError(d.Pos(), "init function is not implemented")
		return function{}, false
	}
	if d.Body == nil {
		cs.addError(d.Pos(), "function must have a body")
		return function{}, false
	}

	inParams, outParams := cs.parseFuncParams(block, d)

	checkVaryings := func(vs []variable) {
		if len(cs.ir.Varyings) != len(vs) {
			cs.addError(d.Pos(), fmt.Sprintf("the number of vertex entry point's returning values and the number of framgent entry point's params must be the same"))
			return
		}
		for i, t := range cs.ir.Varyings {
			if t.Main != vs[i].typ.Main {
				cs.addError(d.Pos(), fmt.Sprintf("vertex entry point's returning value types and framgent entry point's param types must match"))
			}
		}
	}

	if block == &cs.global {
		switch d.Name.Name {
		case cs.vertexEntry:
			for _, v := range inParams {
				cs.ir.Attributes = append(cs.ir.Attributes, v.typ)
			}

			// The first out-param is treated as gl_Position in GLSL.
			if len(outParams) == 0 {
				cs.addError(d.Pos(), fmt.Sprintf("vertex entry point must have at least one returning vec4 value for a position"))
				return function{}, false
			}
			if outParams[0].typ.Main != shaderir.Vec4 {
				cs.addError(d.Pos(), fmt.Sprintf("vertex entry point must have at least one returning vec4 value for a position"))
				return function{}, false
			}

			if cs.varyingParsed {
				checkVaryings(outParams[1:])
			} else {
				for _, v := range outParams[1:] {
					// TODO: Check that these params are not arrays or structs
					cs.ir.Varyings = append(cs.ir.Varyings, v.typ)
				}
			}
			cs.varyingParsed = true
		case cs.fragmentEntry:
			if len(inParams) == 0 {
				cs.addError(d.Pos(), fmt.Sprintf("fragment entry point must have at least one vec4 parameter for a position"))
				return function{}, false
			}
			if inParams[0].typ.Main != shaderir.Vec4 {
				cs.addError(d.Pos(), fmt.Sprintf("fragment entry point must have at least one vec4 parameter for a position"))
				return function{}, false
			}

			if len(outParams) != 1 {
				cs.addError(d.Pos(), fmt.Sprintf("fragment entry point must have one returning vec4 value for a color"))
				return function{}, false
			}
			if outParams[0].typ.Main != shaderir.Vec4 {
				cs.addError(d.Pos(), fmt.Sprintf("fragment entry point must have one returning vec4 value for a color"))
				return function{}, false
			}

			if cs.varyingParsed {
				checkVaryings(inParams[1:])
			} else {
				for _, v := range inParams[1:] {
					cs.ir.Varyings = append(cs.ir.Varyings, v.typ)
				}
			}
			cs.varyingParsed = true
		}
	}

	b, ok := cs.parseBlock(block, d.Name.Name, d.Body.List, inParams, outParams, true)
	if !ok {
		return function{}, false
	}

	if len(outParams) > 0 {
		var hasReturn func(stmts []shaderir.Stmt) bool
		hasReturn = func(stmts []shaderir.Stmt) bool {
			for _, stmt := range stmts {
				if stmt.Type == shaderir.Return {
					return true
				}
				for _, b := range stmt.Blocks {
					if hasReturn(b.Stmts) {
						return true
					}
				}
			}
			return false
		}

		if !hasReturn(b.ir.Stmts) {
			cs.addError(d.Pos(), fmt.Sprintf("function %s must have a return statement but not", d.Name))
			return function{}, false
		}
	}

	var inT, outT []shaderir.Type
	for _, v := range inParams {
		inT = append(inT, v.typ)
	}
	for _, v := range outParams {
		outT = append(outT, v.typ)
	}

	return function{
		name:  d.Name.Name,
		block: b,
		ir: shaderir.Func{
			InParams:  inT,
			OutParams: outT,
			Block:     b.ir,
		},
	}, true
}

func (cs *compileState) parseBlock(outer *block, fname string, stmts []ast.Stmt, inParams, outParams []variable, checkLocalVariableUsage bool) (*block, bool) {
	var vars []variable
	if outer == &cs.global {
		vars = make([]variable, 0, len(inParams)+len(outParams))
		vars = append(vars, inParams...)
		vars = append(vars, outParams...)
	}

	var offset int
	for b := outer; b != nil; b = b.outer {
		offset += len(b.vars)
	}
	if outer == &cs.global {
		offset += len(inParams) + len(outParams)
	}

	block := &block{
		vars:  vars,
		outer: outer,
		ir: &shaderir.Block{
			LocalVarIndexOffset: offset,
		},
	}

	defer func() {
		var offset int
		if outer == &cs.global {
			offset = len(inParams) + len(outParams)
		}
		for _, v := range block.vars[offset:] {
			if v.forLoopCounter {
				block.ir.LocalVars = append(block.ir.LocalVars, shaderir.Type{})
				continue
			}
			block.ir.LocalVars = append(block.ir.LocalVars, v.typ)
		}
	}()

	if outer.outer == nil && len(outParams) > 0 && outParams[0].name != "" {
		for i := range outParams {
			block.ir.Stmts = append(block.ir.Stmts, shaderir.Stmt{
				Type:      shaderir.Init,
				InitIndex: len(inParams) + i,
			})
		}
	}

	for _, stmt := range stmts {
		ss, ok := cs.parseStmt(block, fname, stmt, inParams, outParams)
		if !ok {
			return nil, false
		}
		block.ir.Stmts = append(block.ir.Stmts, ss...)
	}

	if checkLocalVariableUsage && len(block.unusedVars) > 0 {
		for idx, pos := range block.unusedVars {
			cs.addError(pos, fmt.Sprintf("local variable %s is not used", block.vars[idx].name))
		}
		return nil, false
	}

	return block, true
}
