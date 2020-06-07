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
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

type variable struct {
	name string
	typ  shaderir.Type
}

type constant struct {
	name string
	typ  shaderir.Type
	init ast.Expr
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

	// uniforms is a collection of uniform variable names.
	uniforms []string

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
	for i, u := range cs.uniforms {
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
	types  []typ
	vars   []variable
	consts []constant
	pos    token.Pos
	outer  *block

	ir shaderir.Block
}

func (b *block) findLocalVariable(name string) (int, bool) {
	idx := 0
	for outer := b.outer; outer != nil; outer = outer.outer {
		idx += len(outer.vars)
	}
	for i, v := range b.vars {
		if v.name == name {
			return idx + i, true
		}
	}
	if b.outer != nil {
		return b.outer.findLocalVariable(name)
	}
	return 0, false
}

type ParseError struct {
	errs []string
}

func (p *ParseError) Error() string {
	return strings.Join(p.errs, "\n")
}

func Compile(fs *token.FileSet, f *ast.File, vertexEntry, fragmentEntry string) (*shaderir.Program, error) {
	s := &compileState{
		fs:            fs,
		vertexEntry:   vertexEntry,
		fragmentEntry: fragmentEntry,
	}
	s.parse(f)

	if len(s.errs) > 0 {
		return nil, &ParseError{s.errs}
	}

	// TODO: Resolve identifiers?
	// TODO: Resolve constants

	// TODO: Make a call graph and reorder the elements.
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
			cs.parseDecl(&cs.global, d)
		}
	}

	// Sort the uniform variable so that special variable starting with __ should come first.
	var unames []string
	var utypes []shaderir.Type
	for i, u := range cs.uniforms {
		if strings.HasPrefix(u, "__") {
			unames = append(unames, u)
			utypes = append(utypes, cs.ir.Uniforms[i])
		}
	}
	for i, u := range cs.uniforms {
		if !strings.HasPrefix(u, "__") {
			unames = append(unames, u)
			utypes = append(utypes, cs.ir.Uniforms[i])
		}
	}
	cs.uniforms = unames
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

		inParams, outParams := cs.parseFuncParams(fd)
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
			},
		})
	}

	// Parse functions.
	for _, d := range f.Decls {
		if _, ok := d.(*ast.FuncDecl); ok {
			cs.parseDecl(&cs.global, d)
		}
	}

	if len(cs.errs) > 0 {
		return
	}

	for _, f := range cs.funcs {
		cs.ir.Funcs = append(cs.ir.Funcs, f.ir)
	}
}

func (cs *compileState) parseDecl(b *block, d ast.Decl) {
	switch d := d.(type) {
	case *ast.GenDecl:
		switch d.Tok {
		case token.TYPE:
			// TODO: Parse other types
			for _, s := range d.Specs {
				s := s.(*ast.TypeSpec)
				t := cs.parseType(s.Type)
				b.types = append(b.types, typ{
					name: s.Name.Name,
					ir:   t,
				})
			}
		case token.CONST:
			for _, s := range d.Specs {
				s := s.(*ast.ValueSpec)
				cs := cs.parseConstant(s)
				b.consts = append(b.consts, cs...)
			}
		case token.VAR:
			for _, s := range d.Specs {
				s := s.(*ast.ValueSpec)
				vs, inits, stmts := cs.parseVariable(b, s)
				b.ir.Stmts = append(b.ir.Stmts, stmts...)
				if b == &cs.global {
					// TODO: Should rhs be ignored?
					for i, v := range vs {
						if !strings.HasPrefix(v.name, "__") {
							if v.name[0] < 'A' || 'Z' < v.name[0] {
								cs.addError(s.Names[i].Pos(), fmt.Sprintf("global variables must be exposed: %s", v.name))
							}
						}
						cs.uniforms = append(cs.uniforms, v.name)
						cs.ir.Uniforms = append(cs.ir.Uniforms, v.typ)
					}
					continue
				}
				for i, v := range vs {
					b.vars = append(b.vars, v)
					b.ir.LocalVars = append(b.ir.LocalVars, v.typ)
					if inits[i] != nil {
						b.ir.Stmts = append(b.ir.Stmts, shaderir.Stmt{
							Type: shaderir.Assign,
							Exprs: []shaderir.Expr{
								{
									Type:  shaderir.LocalVariable,
									Index: len(b.vars) - 1,
								},
								*inits[i],
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
		f := cs.parseFunc(b, d)
		if b != &cs.global {
			cs.addError(d.Pos(), "non-global function is not implemented")
			return
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
	}
}

func (s *compileState) parseVariable(block *block, vs *ast.ValueSpec) ([]variable, []*shaderir.Expr, []shaderir.Stmt) {
	var t shaderir.Type
	if vs.Type != nil {
		t = s.parseType(vs.Type)
	}

	var vars []variable
	var inits []*shaderir.Expr
	var stmts []shaderir.Stmt
	for i, n := range vs.Names {
		var init ast.Expr
		if len(vs.Values) > 0 {
			init = vs.Values[i]
			if t.Main == shaderir.None {
				ts := s.detectType(block, init)
				if len(ts) > 1 {
					s.addError(vs.Pos(), fmt.Sprintf("the numbers of lhs and rhs don't match"))
				}
				t = ts[0]
			}
		}
		name := n.Name
		vars = append(vars, variable{
			name: name,
			typ:  t,
		})

		var expr *shaderir.Expr
		if init != nil {
			e, ss := s.parseExpr(block, init)
			expr = &e
			stmts = append(stmts, ss...)
		}
		inits = append(inits, expr)
	}
	return vars, inits, stmts
}

func (s *compileState) parseConstant(vs *ast.ValueSpec) []constant {
	var t shaderir.Type
	if vs.Type != nil {
		t = s.parseType(vs.Type)
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

func (cs *compileState) parseFuncParams(d *ast.FuncDecl) (in, out []variable) {
	for _, f := range d.Type.Params.List {
		t := cs.parseType(f.Type)
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
		t := cs.parseType(f.Type)
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

func (cs *compileState) parseFunc(block *block, d *ast.FuncDecl) function {
	if d.Name == nil {
		cs.addError(d.Pos(), "function must have a name")
		return function{}
	}
	if d.Body == nil {
		cs.addError(d.Pos(), "function must have a body")
		return function{}
	}

	inParams, outParams := cs.parseFuncParams(d)

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
				return function{}
			}
			if outParams[0].typ.Main != shaderir.Vec4 {
				cs.addError(d.Pos(), fmt.Sprintf("vertex entry point must have at least one returning vec4 value for a position"))
				return function{}
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
				return function{}
			}
			if inParams[0].typ.Main != shaderir.Vec4 {
				cs.addError(d.Pos(), fmt.Sprintf("fragment entry point must have at least one vec4 parameter for a position"))
				return function{}
			}

			if len(outParams) != 1 {
				cs.addError(d.Pos(), fmt.Sprintf("fragment entry point must have one returning vec4 value for a color"))
				return function{}
			}
			if outParams[0].typ.Main != shaderir.Vec4 {
				cs.addError(d.Pos(), fmt.Sprintf("fragment entry point must have one returning vec4 value for a color"))
				return function{}
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

	b := cs.parseBlock(block, d.Body, inParams, outParams)

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
	}
}

func (cs *compileState) parseBlock(outer *block, b *ast.BlockStmt, inParams, outParams []variable) *block {
	vars := make([]variable, 0, len(inParams)+len(outParams))
	vars = append(vars, inParams...)
	vars = append(vars, outParams...)
	block := &block{
		vars:  vars,
		outer: outer,
	}

	for _, l := range b.List {
		switch l := l.(type) {
		case *ast.AssignStmt:
			switch l.Tok {
			case token.DEFINE:
				for i, e := range l.Lhs {
					v := variable{
						name: e.(*ast.Ident).Name,
					}
					ts := cs.detectType(block, l.Rhs[i])
					if len(ts) > 1 {
						cs.addError(l.Pos(), fmt.Sprintf("the numbers of lhs and rhs don't match"))
					}
					v.typ = ts[0]
					block.vars = append(block.vars, v)
					block.ir.LocalVars = append(block.ir.LocalVars, v.typ)

					// Prase RHS first for the order of the statements.
					rhs, stmts := cs.parseExpr(block, l.Rhs[i])
					block.ir.Stmts = append(block.ir.Stmts, stmts...)
					lhs, stmts := cs.parseExpr(block, l.Lhs[i])
					block.ir.Stmts = append(block.ir.Stmts, stmts...)

					block.ir.Stmts = append(block.ir.Stmts, shaderir.Stmt{
						Type:  shaderir.Assign,
						Exprs: []shaderir.Expr{lhs, rhs},
					})
				}
			case token.ASSIGN:
				// TODO: What about the statement `a,b = b,a?`
				for i := range l.Rhs {
					// Prase RHS first for the order of the statements.
					rhs, stmts := cs.parseExpr(block, l.Rhs[i])
					block.ir.Stmts = append(block.ir.Stmts, stmts...)
					lhs, stmts := cs.parseExpr(block, l.Lhs[i])
					block.ir.Stmts = append(block.ir.Stmts, stmts...)

					block.ir.Stmts = append(block.ir.Stmts, shaderir.Stmt{
						Type:  shaderir.Assign,
						Exprs: []shaderir.Expr{lhs, rhs},
					})
				}
			}
		case *ast.BlockStmt:
			b := cs.parseBlock(block, l, nil, nil)
			block.ir.Stmts = append(block.ir.Stmts, shaderir.Stmt{
				Type: shaderir.BlockStmt,
				Blocks: []shaderir.Block{
					b.ir,
				},
			})
		case *ast.DeclStmt:
			cs.parseDecl(block, l.Decl)
		case *ast.ReturnStmt:
			for i, r := range l.Results {
				e, stmts := cs.parseExpr(block, r)
				block.ir.Stmts = append(block.ir.Stmts, stmts...)
				block.ir.Stmts = append(block.ir.Stmts, shaderir.Stmt{
					Type: shaderir.Assign,
					Exprs: []shaderir.Expr{
						{
							Type:  shaderir.LocalVariable,
							Index: len(inParams) + i,
						},
						e,
					},
				})
			}
			block.ir.Stmts = append(block.ir.Stmts, shaderir.Stmt{
				Type: shaderir.Return,
			})
		}
	}

	return block
}

func (cs *compileState) parseExpr(block *block, expr ast.Expr) (shaderir.Expr, []shaderir.Stmt) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			v, err := strconv.ParseInt(e.Value, 10, 32)
			if err != nil {
				cs.addError(e.Pos(), fmt.Sprintf("unexpected literal: %s", e.Value))
				return shaderir.Expr{}, nil
			}
			return shaderir.Expr{
				Type: shaderir.IntExpr,
				Int:  int32(v),
			}, nil
		case token.FLOAT:
			v, err := strconv.ParseFloat(e.Value, 32)
			if err != nil {
				cs.addError(e.Pos(), fmt.Sprintf("unexpected literal: %s", e.Value))
				return shaderir.Expr{}, nil
			}
			return shaderir.Expr{
				Type:  shaderir.FloatExpr,
				Float: float32(v),
			}, nil
		default:
			cs.addError(e.Pos(), fmt.Sprintf("literal not implemented: %#v", e))
		}
	case *ast.BinaryExpr:
		var op shaderir.Op
		switch e.Op {
		case token.ADD:
			op = shaderir.Add
		case token.SUB:
			op = shaderir.Sub
		case token.NOT:
			op = shaderir.NotOp
		case token.MUL:
			op = shaderir.Mul
		case token.QUO:
			op = shaderir.Div
		case token.REM:
			op = shaderir.ModOp
		case token.SHL:
			op = shaderir.LeftShift
		case token.SHR:
			op = shaderir.RightShift
		case token.LSS:
			op = shaderir.LessThanOp
		case token.LEQ:
			op = shaderir.LessThanEqualOp
		case token.GTR:
			op = shaderir.GreaterThanOp
		case token.GEQ:
			op = shaderir.GreaterThanEqualOp
		case token.EQL:
			op = shaderir.EqualOp
		case token.NEQ:
			op = shaderir.NotEqualOp
		case token.AND:
			op = shaderir.And
		case token.XOR:
			op = shaderir.Xor
		case token.OR:
			op = shaderir.Or
		case token.LAND:
			op = shaderir.AndAnd
		case token.LOR:
			op = shaderir.OrOr
		default:
			cs.addError(e.Pos(), fmt.Sprintf("unexpected operator: %s", e.Op))
			return shaderir.Expr{}, nil
		}

		var stmts []shaderir.Stmt

		// Prase RHS first for the order of the statements.
		rhs, ss := cs.parseExpr(block, e.Y)
		stmts = append(stmts, ss...)
		lhs, ss := cs.parseExpr(block, e.X)
		stmts = append(stmts, ss...)

		return shaderir.Expr{
			Type:  shaderir.Binary,
			Op:    op,
			Exprs: []shaderir.Expr{lhs, rhs},
		}, stmts
	case *ast.CallExpr:
		var (
			callee shaderir.Expr
			args   []shaderir.Expr
			stmts  []shaderir.Stmt
		)

		// Parse the argument first for the order of the statements.
		for _, a := range e.Args {
			e, ss := cs.parseExpr(block, a)
			// TODO: Convert integer literals to float literals if necessary.
			args = append(args, e)
			stmts = append(stmts, ss...)
		}

		// TODO: When len(ss) is not 0?
		expr, ss := cs.parseExpr(block, e.Fun)
		callee = expr
		stmts = append(stmts, ss...)

		// For built-in functions, we can call this in this position. Return an expression for the function
		// call.
		if expr.Type == shaderir.BuiltinFuncExpr {
			return shaderir.Expr{
				Type:  shaderir.Call,
				Exprs: append([]shaderir.Expr{callee}, args...),
			}, stmts
		}

		if expr.Type != shaderir.FunctionExpr {
			cs.addError(e.Pos(), fmt.Sprintf("function callee must be a funciton name but %s", e.Fun))
		}
		f := cs.funcs[expr.Index]

		var outParams []int
		for _, p := range f.ir.OutParams {
			idx := len(block.vars)
			block.vars = append(block.vars, variable{
				typ: p,
			})
			block.ir.LocalVars = append(block.ir.LocalVars, p)
			args = append(args, shaderir.Expr{
				Type:  shaderir.LocalVariable,
				Index: idx,
			})
			outParams = append(outParams, idx)
		}

		if t := f.ir.Return; t.Main != shaderir.None {
			idx := len(block.vars)
			block.vars = append(block.vars, variable{
				typ: t,
			})

			// Calling the function should be done eariler to treat out-params correctly.
			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Assign,
				Exprs: []shaderir.Expr{
					{
						Type:  shaderir.LocalVariable,
						Index: idx,
					},
					{
						Type:  shaderir.Call,
						Exprs: append([]shaderir.Expr{callee}, args...),
					},
				},
			})

			// The actual expression here is just a local variable that includes the result of the
			// function call.
			return shaderir.Expr{
				Type:  shaderir.LocalVariable,
				Index: idx,
			}, stmts
		}

		// Even if the function doesn't return anything, calling the function should be done eariler to keep
		// the evaluation order.
		stmts = append(stmts, shaderir.Stmt{
			Type: shaderir.ExprStmt,
			Exprs: []shaderir.Expr{
				{
					Type:  shaderir.Call,
					Exprs: append([]shaderir.Expr{callee}, args...),
				},
			},
		})

		// TODO: What about the other params?
		if len(outParams) > 0 {
			return shaderir.Expr{
				Type:  shaderir.LocalVariable,
				Index: outParams[0],
			}, stmts
		}

		// TODO: Is an empty expression work?
		return shaderir.Expr{}, stmts
	case *ast.Ident:
		if i, ok := block.findLocalVariable(e.Name); ok {
			return shaderir.Expr{
				Type:  shaderir.LocalVariable,
				Index: i,
			}, nil
		}
		if i, ok := cs.findFunction(e.Name); ok {
			return shaderir.Expr{
				Type:  shaderir.FunctionExpr,
				Index: i,
			}, nil
		}
		if i, ok := cs.findUniformVariable(e.Name); ok {
			return shaderir.Expr{
				Type:  shaderir.UniformVariable,
				Index: i,
			}, nil
		}
		if f, ok := shaderir.ParseBuiltinFunc(e.Name); ok {
			return shaderir.Expr{
				Type:        shaderir.BuiltinFuncExpr,
				BuiltinFunc: f,
			}, nil
		}
		cs.addError(e.Pos(), fmt.Sprintf("unexpected identifier: %s", e.Name))
	case *ast.SelectorExpr:
		expr, stmts := cs.parseExpr(block, e.X)
		return shaderir.Expr{
			Type: shaderir.FieldSelector,
			Exprs: []shaderir.Expr{
				expr,
				{
					Type:      shaderir.SwizzlingExpr,
					Swizzling: e.Sel.Name,
				},
			},
		}, stmts
	case *ast.UnaryExpr:
		var op shaderir.Op
		switch e.Op {
		case token.ADD:
			op = shaderir.Add
		case token.SUB:
			op = shaderir.Sub
		case token.NOT:
			op = shaderir.NotOp
		default:
			cs.addError(e.Pos(), fmt.Sprintf("unexpected operator: %s", e.Op))
			return shaderir.Expr{}, nil
		}
		expr, stmts := cs.parseExpr(block, e.X)
		return shaderir.Expr{
			Type:  shaderir.Unary,
			Op:    op,
			Exprs: []shaderir.Expr{expr},
		}, stmts
	default:
		cs.addError(e.Pos(), fmt.Sprintf("expression not implemented: %#v", e))
	}
	return shaderir.Expr{}, nil
}
