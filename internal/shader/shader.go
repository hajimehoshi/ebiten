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
	"bufio"
	"bytes"
	"fmt"
	"go/ast"
	gconstant "go/constant"
	"go/parser"
	"go/token"
	"regexp"
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
	value gconstant.Value
}

type function struct {
	name string

	ir shaderir.Func
}

type compileState struct {
	fs *token.FileSet

	vertexEntry   string
	fragmentEntry string
	unit          shaderir.Unit

	ir shaderir.Program

	funcs []function

	global block

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

func (b *block) totalLocalVariableCount() int {
	c := len(b.vars)
	if b.outer != nil {
		c += b.outer.totalLocalVariableCount()
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

func Compile(src []byte, vertexEntry, fragmentEntry string, textureCount int) (*shaderir.Program, error) {
	unit, err := ParseCompilerDirectives(src)
	if err != nil {
		return nil, err
	}

	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", src, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	s := &compileState{
		fs:            fs,
		vertexEntry:   vertexEntry,
		fragmentEntry: fragmentEntry,
		unit:          unit,
	}
	s.ir.SourceHash = shaderir.CalcSourceHash(src)
	s.global.ir = &shaderir.Block{}
	s.parse(f)

	if len(s.errs) > 0 {
		return nil, &ParseError{s.errs}
	}

	// TODO: Resolve identifiers?
	// TODO: Resolve constants

	// TODO: Make a call graph and reorder the elements.

	s.ir.TextureCount = textureCount
	return &s.ir, nil
}

func ParseCompilerDirectives(src []byte) (shaderir.Unit, error) {
	// TODO: Change the unit to pixels in v3 (#2645).
	unit := shaderir.Texels

	// Go's whitespace is U+0020 (SP), U+0009 (\t), U+000d (\r), and U+000A (\n).
	// See https://go.dev/ref/spec#Tokens
	reUnit := regexp.MustCompile(`^[ \t\r\n]*//kage:unit\s+([^ \t\r\n]+)[ \t\r\n]*$`)
	var unitParsed bool

	buf := bytes.NewBuffer(src)
	s := bufio.NewScanner(buf)
	for s.Scan() {
		m := reUnit.FindStringSubmatch(s.Text())
		if m == nil {
			continue
		}
		if unitParsed {
			return 0, fmt.Errorf("shader: at most one //kage:unit can exist in a shader")
		}
		switch m[1] {
		case "pixels":
			unit = shaderir.Pixels
		case "texels":
			unit = shaderir.Texels
		default:
			return 0, fmt.Errorf("shader: invalid value for //kage:unit: %s", m[1])
		}
		unitParsed = true
	}

	return unit, nil
}

func (s *compileState) addError(pos token.Pos, str string) {
	p := s.fs.Position(pos)
	s.errs = append(s.errs, fmt.Sprintf("%s: %s", p, str))
}

func (cs *compileState) parse(f *ast.File) {
	cs.ir.Unit = cs.unit

	// Parse GenDecl for global variables, and then parse functions.
	for _, d := range f.Decls {
		if _, ok := d.(*ast.FuncDecl); !ok {
			ss, ok := cs.parseDecl(&cs.global, "", d)
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
	var vertexInParams []variable
	var vertexOutParams []variable
	var fragmentInParams []variable
	var fragmentOutParams []variable
	var fragmentReturnType shaderir.Type
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		n := fd.Name.Name

		for _, f := range cs.funcs {
			if f.name == n {
				cs.addError(d.Pos(), fmt.Sprintf("redeclared function: %s", n))
				return
			}
		}

		inParams, outParams, ret := cs.parseFuncParams(&cs.global, n, fd)

		if n == cs.vertexEntry {
			vertexInParams = inParams
			vertexOutParams = outParams
			continue
		}
		if n == cs.fragmentEntry {
			fragmentInParams = inParams
			fragmentOutParams = outParams
			fragmentReturnType = ret
			continue
		}

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
				Return:    ret,
				Block:     &shaderir.Block{},
			},
		})
	}

	// Check varying variables.
	// In testings, there might not be vertex and fragment entry points.
	if len(vertexOutParams) > 0 && len(fragmentInParams) > 0 {
		for i, p := range vertexOutParams {
			if len(fragmentInParams) <= i {
				break
			}
			t := fragmentInParams[i].typ
			if !p.typ.Equal(&t) {
				name := fragmentInParams[i].name
				cs.addError(0, fmt.Sprintf("fragment argument %s must be %s but was %s", name, p.typ.String(), t.String()))
			}
		}

		// The first out-param is treated as gl_Position in GLSL.
		if vertexOutParams[0].typ.Main != shaderir.Vec4 {
			cs.addError(0, "vertex entry point must have at least one returning vec4 value for a position")
		}
		if len(fragmentOutParams) != 0 || fragmentReturnType.Main != shaderir.Vec4 {
			cs.addError(0, "fragment entry point must have one returning vec4 value for a color")
		}
	}

	if len(cs.errs) > 0 {
		return
	}

	// Set attribute and varying veraibles.
	for _, p := range vertexInParams {
		cs.ir.Attributes = append(cs.ir.Attributes, p.typ)
	}
	if len(vertexOutParams) > 0 {
		// TODO: Check that these params are not arrays or structs
		// The 0th argument is a special variable for position and is not included in varying variables.
		for _, p := range vertexOutParams[1:] {
			cs.ir.Varyings = append(cs.ir.Varyings, p.typ)
		}
	}

	// Parse functions.
	for _, d := range f.Decls {
		if f, ok := d.(*ast.FuncDecl); ok {
			ss, ok := cs.parseDecl(&cs.global, f.Name.Name, d)
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

func (cs *compileState) parseDecl(b *block, fname string, d ast.Decl) ([]shaderir.Stmt, bool) {
	var stmts []shaderir.Stmt

	switch d := d.(type) {
	case *ast.GenDecl:
		switch d.Tok {
		case token.TYPE:
			// TODO: Parse other types
			for _, s := range d.Specs {
				s := s.(*ast.TypeSpec)
				t, ok := cs.parseType(b, fname, s.Type)
				if !ok {
					return nil, false
				}
				n := s.Name.Name
				for _, t := range b.types {
					if t.name == n {
						cs.addError(s.Pos(), fmt.Sprintf("%s redeclared in this block", n))
						return nil, false
					}
				}
				b.types = append(b.types, typ{
					name: n,
					ir:   t,
				})
			}
		case token.CONST:
			for _, s := range d.Specs {
				s := s.(*ast.ValueSpec)
				cs, ok := cs.parseConstant(b, fname, s)
				if !ok {
					return nil, false
				}
				b.consts = append(b.consts, cs...)
			}
		case token.VAR:
			for _, s := range d.Specs {
				s := s.(*ast.ValueSpec)
				vs, inits, ss, ok := cs.parseVariable(b, fname, s)
				if !ok {
					return nil, false
				}

				stmts = append(stmts, ss...)
				if b == &cs.global {
					if len(inits) > 0 {
						cs.addError(s.Pos(), "a uniform variable cannot have initial values")
						return nil, false
					}

					// TODO: Should rhs be ignored?
					for i, v := range vs {
						if !strings.HasPrefix(v.name, "__") {
							if v.name[0] < 'A' || 'Z' < v.name[0] {
								cs.addError(s.Names[i].Pos(), fmt.Sprintf("global variables must be exposed: %s", v.name))
							}
						}
						for _, name := range cs.ir.UniformNames {
							if name == v.name {
								cs.addError(s.Pos(), fmt.Sprintf("%s redeclared in this block", name))
								return nil, false
							}
						}
						cs.ir.UniformNames = append(cs.ir.UniformNames, v.name)
						cs.ir.Uniforms = append(cs.ir.Uniforms, v.typ)
					}
					continue
				}

				// base must be obtained before adding the variables.
				base := b.totalLocalVariableCount()
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

func (s *compileState) parseVariable(block *block, fname string, vs *ast.ValueSpec) ([]variable, []shaderir.Expr, []shaderir.Stmt, bool) {
	if len(vs.Names) != len(vs.Values) && len(vs.Values) != 1 && len(vs.Values) != 0 {
		s.addError(vs.Pos(), "the numbers of lhs and rhs don't match")
		return nil, nil, nil, false
	}

	var declt shaderir.Type
	if vs.Type != nil {
		var ok bool
		declt, ok = s.parseType(block, fname, vs.Type)
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

			es, rts, ss, ok := s.parseExpr(block, fname, init, true)
			if !ok {
				return nil, nil, nil, false
			}

			if t.Main == shaderir.None {
				ts, ok := s.functionReturnTypes(block, init)
				if !ok {
					ts = rts
				}
				if len(ts) > 1 {
					s.addError(vs.Pos(), "the numbers of lhs and rhs don't match")
				}
				t = ts[0]
				if t.Main == shaderir.None {
					t = toDefaultType(es[0].Const)
				}
			}

			for i, rt := range rts {
				if !canAssign(&t, &rt, es[i].Const) {
					s.addError(vs.Pos(), fmt.Sprintf("cannot use type %s as type %s in variable declaration", rt.String(), t.String()))
				}
			}

			inits = append(inits, es...)
			stmts = append(stmts, ss...)

		default:
			// Multiple-value context
			// See testcase/var_multiple.go for an actual case.

			if i == 0 {
				init := vs.Values[0]

				var ss []shaderir.Stmt
				var ok bool
				initexprs, inittypes, ss, ok = s.parseExpr(block, fname, init, true)
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
						s.addError(vs.Pos(), "the numbers of lhs and rhs don't match")
						continue
					}
				}
			}

			if t.Main == shaderir.None && len(inittypes) > 0 {
				t = inittypes[i]
				// TODO: Is it possible to reach this?
				if t.Main == shaderir.None {
					t = toDefaultType(initexprs[i].Const)
				}
			}

			if !canAssign(&t, &inittypes[i], initexprs[i].Const) {
				s.addError(vs.Pos(), fmt.Sprintf("cannot use type %s as type %s in variable declaration", inittypes[i].String(), t.String()))
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

func (s *compileState) parseConstant(block *block, fname string, vs *ast.ValueSpec) ([]constant, bool) {
	var t shaderir.Type
	if vs.Type != nil {
		var ok bool
		t, ok = s.parseType(block, fname, vs.Type)
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

		es, ts, ss, ok := s.parseExpr(block, fname, vs.Values[i], false)
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
			s.addError(vs.Pos(), fmt.Sprintf("constant expression must be a number but not: %s", n))
			return nil, false
		}

		if !t.Equal(&shaderir.Type{}) && !canAssign(&t, &ts[0], es[0].Const) {
			s.addError(vs.Pos(), fmt.Sprintf("cannot use %v as %s value in constant declaration", es[0].Const, t.String()))
			return nil, false
		}

		c := es[0].Const
		switch t.Main {
		case shaderir.Bool:
		case shaderir.Int:
			c = gconstant.ToInt(c)
		case shaderir.Float:
			c = gconstant.ToFloat(c)
		}

		cs = append(cs, constant{
			name:  name,
			typ:   t,
			value: c,
		})
	}
	return cs, true
}

func (cs *compileState) parseFuncParams(block *block, fname string, d *ast.FuncDecl) (in, out []variable, ret shaderir.Type) {
	for _, f := range d.Type.Params.List {
		t, ok := cs.parseType(block, fname, f.Type)
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
		t, ok := cs.parseType(block, fname, f.Type)
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

	// If there is only one returning value, it is treated as a returning value.
	// An array cannot be a returning value, especially for HLSL (#2923).
	//
	// For the vertex entry, a parameter (variable) is used as a returning value.
	// For example, GLSL doesn't treat gl_Position as a returning value.
	// Thus, the returning value is not set for the vertex entry.
	// TODO: This can be resolved by having an indirect function like what the fragment entry already does.
	// See internal/shaderir/glsl.adjustProgram.
	if len(out) == 1 && out[0].name == "" && out[0].typ.Main != shaderir.Array && fname != cs.vertexEntry {
		ret = out[0].typ
		out = nil
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

	inParams, outParams, returnType := cs.parseFuncParams(block, d.Name.Name, d)
	if d.Name.Name == cs.fragmentEntry {
		if len(inParams) == 0 {
			inParams = append(inParams, variable{
				name: "_",
				typ:  shaderir.Type{Main: shaderir.Vec4},
			})
		}
		// The 0th inParams is a special variable for position and is not included in varying variables.
		if diff := len(cs.ir.Varyings) - (len(inParams) - 1); diff > 0 {
			// inParams is not enough when the vertex shader has more returning values than the fragment shader's arguments.
			orig := len(inParams) - 1
			for i := 0; i < diff; i++ {
				inParams = append(inParams, variable{
					name: "_",
					typ:  cs.ir.Varyings[orig+i],
				})
			}
		}
	}
	b, ok := cs.parseBlock(block, d.Name.Name, d.Body.List, inParams, outParams, returnType, true)
	if !ok {
		return function{}, false
	}

	if len(outParams) > 0 || returnType.Main != shaderir.None {
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
		name: d.Name.Name,
		ir: shaderir.Func{
			InParams:  inT,
			OutParams: outT,
			Return:    returnType,
			Block:     b.ir,
		},
	}, true
}

func (cs *compileState) parseBlock(outer *block, fname string, stmts []ast.Stmt, inParams, outParams []variable, returnType shaderir.Type, checkLocalVariableUsage bool) (*block, bool) {
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
		ss, ok := cs.parseStmt(block, fname, stmt, inParams, outParams, returnType)
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
