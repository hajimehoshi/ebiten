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
	"bytes"
	"fmt"
	"go/ast"
	gconstant "go/constant"
	"go/parser"
	"go/token"
	"math"
	"slices"
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
	pos  token.Pos

	ir shaderir.Func
}

type compileState struct {
	fs *token.FileSet

	// internalRegionStart and internalRegionEnd are the byte offsets of the internal region in the source.
	// The region is empty when the source has none.
	internalRegionStart int
	internalRegionEnd   int

	vertexEntry   string
	fragmentEntry string

	vertexEntryPos   token.Pos
	fragmentEntryPos token.Pos

	ir shaderir.Program

	funcs []function

	global block

	errs []compileError
}

type compileError struct {
	position token.Position
	message  string
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
	outer      *block

	// loop is true when the block is the scope of a for-statement.
	loop bool

	ir *shaderir.Block
}

// inLoop reports whether the block or one of its enclosing blocks belongs to a for-statement.
func (b *block) inLoop() bool {
	for ; b != nil; b = b.outer {
		if b.loop {
			return true
		}
	}
	return false
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

	var idx int
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
	for _, b := range slices.Backward(bs) {
		if len(b.vars) <= idx {
			idx -= len(b.vars)
			continue
		}
		return b.vars[idx].typ, true
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
	errs []compileError
}

func (p *ParseError) Error() string {
	msgs := make([]string, 0, len(p.errs))
	for _, e := range p.errs {
		msgs = append(msgs, fmt.Sprintf("%s: %s", e.position, e.message))
	}
	return strings.Join(msgs, "\n")
}

// Positions returns the source positions of the errors, in the same order as [ParseError.Error] reports them.
func (p *ParseError) Positions() []token.Position {
	ps := make([]token.Position, 0, len(p.errs))
	for _, e := range p.errs {
		ps = append(ps, e.position)
	}
	return ps
}

const internalRegionDirective = "//kage:internalregion"

const (
	// InternalRegionBegin is the line that begins the internal region.
	InternalRegionBegin = internalRegionDirective + " begin"

	// InternalRegionEnd is the line that ends the internal region.
	InternalRegionEnd = internalRegionDirective + " end"
)

// Compile compiles a Kage source into an intermediate representation.
//
// A uniform variable whose name starts with __ can be declared only in the internal region, which is the
// lines between [InternalRegionBegin] and [InternalRegionEnd]. A source can have at most one internal region.
func Compile(src []byte, vertexEntry, fragmentEntry string, textureCount int) (*shaderir.Program, error) {
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "", src, parser.AllErrors)
	if err != nil {
		return nil, err
	}

	s := &compileState{
		fs:            fs,
		vertexEntry:   vertexEntry,
		fragmentEntry: fragmentEntry,
	}
	s.ir.SourceID = shaderir.CalcSourceID(src)
	s.ir.TextureCount = textureCount
	s.global.ir = &shaderir.Block{}
	s.parseInternalRegion(fs.File(f.Pos()), src)
	if len(s.errs) > 0 {
		return nil, &ParseError{s.errs}
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
	s.errs = append(s.errs, compileError{
		position: s.fs.Position(pos),
		message:  str,
	})
}

// parseInternalRegion records the byte offsets of the internal region in src, the source of file, and
// reports malformed internal region directives.
func (cs *compileState) parseInternalRegion(file *token.File, src []byte) {
	begin, end := -1, -1
	var offset int
	for line := range bytes.Lines(src) {
		lineOffset := offset
		offset += len(line)

		directive := strings.TrimSpace(string(line))
		arg, ok := strings.CutPrefix(directive, internalRegionDirective)
		if !ok {
			continue
		}
		// Skip another directive whose name merely starts with the same prefix.
		if arg != "" && arg[0] != ' ' && arg[0] != '\t' {
			continue
		}
		switch strings.TrimSpace(arg) {
		case "begin":
			if begin >= 0 {
				cs.addError(file.Pos(lineOffset), "at most one internal region can exist in a shader")
				return
			}
			begin = lineOffset
		case "end":
			if end >= 0 {
				cs.addError(file.Pos(lineOffset), "at most one internal region can exist in a shader")
				return
			}
			end = lineOffset
		default:
			cs.addError(file.Pos(lineOffset), fmt.Sprintf("invalid directive: %s", directive))
			return
		}
	}

	if begin < 0 && end < 0 {
		return
	}
	if begin < 0 || end < 0 || end < begin {
		cs.addError(file.Pos(max(begin, end)), fmt.Sprintf("%s and %s must appear in this order", InternalRegionBegin, InternalRegionEnd))
		return
	}
	cs.internalRegionStart = begin
	cs.internalRegionEnd = end
}

// inInternalRegion reports whether pos is in the internal region.
func (cs *compileState) inInternalRegion(pos token.Pos) bool {
	offset := cs.fs.Position(pos).Offset
	return cs.internalRegionStart <= offset && offset < cs.internalRegionEnd
}

func (cs *compileState) parse(f *ast.File) {
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
	// TODO: Check len(unames) == graphics.PreservedUniformVariablesCount. Unfortunately this is not true in tests.
	for i, u := range cs.ir.UniformNames {
		if !strings.HasPrefix(u, "__") {
			unames = append(unames, u)
			utypes = append(utypes, cs.ir.Uniforms[i])
		}
	}
	cs.ir.UniformNames = unames
	cs.ir.Uniforms = utypes

	// Parse function names so that any function can call the other functions.
	// The function data is provisional and will be updated soon.
	var vertexInParams []variable
	var vertexOutParams []variable
	var fragmentInParams []variable
	var fragmentInParamPositions []token.Pos
	var fragmentOutParams []variable
	var fragmentReturnType shaderir.Type
	funcNames := map[string]struct{}{}
	for _, d := range f.Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok {
			continue
		}
		n := fd.Name.Name

		// The entry points are not registered in cs.funcs, so check the names separately.
		if _, ok := funcNames[n]; ok {
			cs.addError(d.Pos(), fmt.Sprintf("redeclared function: %s", n))
			return
		}
		funcNames[n] = struct{}{}

		inParams, outParams, ret := cs.parseFuncParams(&cs.global, n, fd)

		if n == cs.vertexEntry {
			cs.vertexEntryPos = d.Pos()
			vertexInParams = inParams
			vertexOutParams = outParams
			continue
		}
		if n == cs.fragmentEntry {
			cs.fragmentEntryPos = d.Pos()
			fragmentInParams = inParams
			fragmentOutParams = outParams
			fragmentReturnType = ret
			for _, field := range fd.Type.Params.List {
				if len(field.Names) == 0 {
					fragmentInParamPositions = append(fragmentInParamPositions, field.Type.Pos())
					continue
				}
				for _, name := range field.Names {
					fragmentInParamPositions = append(fragmentInParamPositions, name.Pos())
				}
			}
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
			pos:  d.Pos(),
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
	// In tests, there might not be vertex and fragment entry points.
	if len(vertexOutParams) > 0 && len(fragmentInParams) > 0 {
		for i, p := range vertexOutParams {
			if len(fragmentInParams) <= i {
				break
			}
			t := fragmentInParams[i].typ
			if !p.typ.Equal(&t) {
				arg := "fragment argument " + fragmentInParams[i].name
				// A blank or unnamed argument has no name to identify it.
				if fragmentInParams[i].name == "_" {
					arg = fmt.Sprintf("the %s fragment argument", ordinal(i+1))
				}
				cs.addError(fragmentInParamPositions[i], fmt.Sprintf("%s must be %s but was %s", arg, p.typ.String(), t.String()))
			}
		}
		if len(fragmentInParams) > len(vertexOutParams) {
			cs.addError(cs.fragmentEntryPos, fmt.Sprintf("the number of the fragment arguments (%d) must not be greater than the number of the vertex returning values (%d)", len(fragmentInParams), len(vertexOutParams)))
		}
	}
	if cs.vertexEntryPos.IsValid() {
		// The first out-param is treated as gl_Position in GLSL.
		if len(vertexOutParams) == 0 || vertexOutParams[0].typ.Main != shaderir.Vec4 {
			cs.addError(cs.vertexEntryPos, "vertex entry point must have at least one returning vec4 value for a position")
		}
	}
	if cs.fragmentEntryPos.IsValid() {
		if len(fragmentOutParams) != 0 || fragmentReturnType.Main != shaderir.Vec4 {
			cs.addError(cs.fragmentEntryPos, "fragment entry point must have one returning vec4 value for a color")
		}
	}

	if len(cs.errs) > 0 {
		return
	}

	// Set attribute and varying variables.
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

	cs.checkRecursiveCalls()
	if len(cs.errs) > 0 {
		return
	}

	for _, f := range cs.funcs {
		cs.ir.Funcs = append(cs.ir.Funcs, f.ir)
	}
}

// ordinal returns n written as an English ordinal number.
func ordinal(n int) string {
	if n%100 >= 11 && n%100 <= 13 {
		return fmt.Sprintf("%dth", n)
	}
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%dst", n)
	case 2:
		return fmt.Sprintf("%dnd", n)
	case 3:
		return fmt.Sprintf("%drd", n)
	}
	return fmt.Sprintf("%dth", n)
}

// checkRecursiveCalls adds an error for each function that calls itself directly or indirectly (#3536).
func (cs *compileState) checkRecursiveCalls() {
	callees := make([][]int, len(cs.funcs))
	for i, f := range cs.funcs {
		callees[i] = calledFunctionIndices(f.ir.Block)
	}
	for i, f := range cs.funcs {
		if callsFunction(callees, i, i) {
			cs.addError(f.pos, fmt.Sprintf("function %s must not be called recursively", f.name))
		}
	}
}

// callsFunction reports whether the function at index from calls the function at index to directly or indirectly.
// callees[i] holds the indices of the functions that the function at index i calls.
func callsFunction(callees [][]int, from, to int) bool {
	visited := make([]bool, len(callees))
	stack := slices.Clone(callees[from])
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if i == to {
			return true
		}
		if visited[i] {
			continue
		}
		visited[i] = true
		stack = append(stack, callees[i]...)
	}
	return false
}

// calledFunctionIndices returns the indices of the user-defined functions called in the block, without duplicates.
func calledFunctionIndices(b *shaderir.Block) []int {
	seen := map[int]struct{}{}
	var indices []int
	var walkExprs func(exprs []shaderir.Expr)
	walkExprs = func(exprs []shaderir.Expr) {
		for i := range exprs {
			e := &exprs[i]
			if e.Type == shaderir.FunctionExpr {
				if _, ok := seen[e.Index]; !ok {
					seen[e.Index] = struct{}{}
					indices = append(indices, e.Index)
				}
			}
			walkExprs(e.Exprs)
		}
	}
	var walkBlock func(b *shaderir.Block)
	walkBlock = func(b *shaderir.Block) {
		if b == nil {
			return
		}
		for i := range b.Stmts {
			walkExprs(b.Stmts[i].Exprs)
			for _, bb := range b.Stmts[i].Blocks {
				walkBlock(bb)
			}
		}
	}
	walkBlock(b)
	return indices
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
						// A uniform variable starting with __ is reserved for the internal region.
						reserved := strings.HasPrefix(v.name, "__") && cs.inInternalRegion(s.Names[i].Pos())
						if !reserved {
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

// functionReturnTypes returns the original returning value types, if the given expression is a call.
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
			if len(es) == 0 || len(rts) == 0 {
				s.addError(vs.Pos(), "the right-hand side of the variable declaration has no value")
				return nil, nil, nil, false
			}
			if len(es) > 1 || len(rts) > 1 {
				s.addError(vs.Pos(), "the numbers of lhs and rhs don't match")
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
				if len(ts) == 0 {
					s.addError(vs.Pos(), "the right-hand side of the variable declaration has no value")
					return nil, nil, nil, false
				}
				t = ts[0]
				if t.Main == shaderir.None {
					t = toDefaultType(es[0].Const)
				}
			}

			for i, rt := range rts {
				if !canAssign(&t, &rt, es[i].Const) {
					s.addError(n.Pos(), fmt.Sprintf("cannot use type %s as type %s in variable declaration", rt.String(), t.String()))
				}
				if es[i].Const != nil {
					switch t.Main {
					case shaderir.Int:
						es[i].Const = gconstant.ToInt(es[i].Const)
					case shaderir.Float:
						es[i].Const = gconstant.ToFloat(es[i].Const)
					}
				}
			}

			inits = append(inits, es...)
			stmts = append(stmts, ss...)

		default:
			// Multiple-value context
			// See testdata/var_multiple.go for an actual case.

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
				}

				if len(initexprs) != len(vs.Names) || len(inittypes) != len(vs.Names) {
					s.addError(vs.Pos(), "the numbers of lhs and rhs don't match")
					return nil, nil, nil, false
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
				s.addError(n.Pos(), fmt.Sprintf("cannot use type %s as type %s in variable declaration", inittypes[i].String(), t.String()))
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
	if len(vs.Names) > len(vs.Values) {
		s.addError(vs.Pos(), "missing init expr for const declaration")
		return nil, false
	}
	if len(vs.Names) < len(vs.Values) {
		s.addError(vs.Pos(), "extra init expr for const declaration")
		return nil, false
	}

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
			if !s.checkIntConstRange(vs.Pos(), c) {
				return nil, false
			}
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
	// Parameters and named results share one scope, so a name must not appear twice among them.
	names := map[string]struct{}{}
	checkName := func(n *ast.Ident) bool {
		if n.Name == "_" {
			return true
		}
		if _, ok := names[n.Name]; ok {
			cs.addError(n.Pos(), fmt.Sprintf("duplicate argument %s", n.Name))
			return false
		}
		names[n.Name] = struct{}{}
		return true
	}

	for _, f := range d.Type.Params.List {
		t, ok := cs.parseType(block, fname, f.Type)
		if !ok {
			return
		}
		if len(f.Names) == 0 {
			// An unnamed parameter cannot be referred to, just like a blank identifier.
			in = append(in, variable{
				name: "_",
				typ:  t,
			})
			continue
		}
		for _, n := range f.Names {
			if !checkName(n) {
				return
			}
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
				if !checkName(n) {
					return
				}
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
			for i := range diff {
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
		if !isTerminating(b.ir.Stmts) {
			cs.addError(d.Pos(), fmt.Sprintf("function %s must end with a return statement on every path but does not", d.Name))
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
		pos:  d.Pos(),
		ir: shaderir.Func{
			InParams:  inT,
			OutParams: outT,
			Return:    returnType,
			Block:     b.ir,
		},
	}, true
}

// checkIntConstRange reports an error at pos when v is an integer constant that does not fit in a 32-bit integer.
func (cs *compileState) checkIntConstRange(pos token.Pos, v gconstant.Value) bool {
	if v == nil || v.Kind() != gconstant.Int {
		return true
	}
	if x, exact := gconstant.Int64Val(v); exact && math.MinInt32 <= x && x <= math.MaxInt32 {
		return true
	}
	cs.addError(pos, fmt.Sprintf("constant %s overflows int", v.String()))
	return false
}

// checkIntConstRangeInStmts reports an error at pos for every integer constant in stmts that does not fit in a 32-bit integer.
// Nested blocks are not checked, as parseBlock checks each block it parses.
func (cs *compileState) checkIntConstRangeInStmts(pos token.Pos, stmts []shaderir.Stmt) bool {
	ok := true
	var checkExpr func(e *shaderir.Expr)
	checkExpr = func(e *shaderir.Expr) {
		if e.Type == shaderir.NumberExpr && !cs.checkIntConstRange(pos, e.Const) {
			ok = false
		}
		for i := range e.Exprs {
			checkExpr(&e.Exprs[i])
		}
	}
	for i := range stmts {
		s := &stmts[i]
		for j := range s.Exprs {
			checkExpr(&s.Exprs[j])
		}
		if s.Type != shaderir.For {
			continue
		}
		for _, v := range []gconstant.Value{s.ForInit, s.ForEnd, s.ForDelta} {
			if !cs.checkIntConstRange(pos, v) {
				ok = false
			}
		}
	}
	return ok
}

// isTerminating reports whether stmts ends in a statement that returns or discards on every path.
func isTerminating(stmts []shaderir.Stmt) bool {
	if len(stmts) == 0 {
		return false
	}
	last := stmts[len(stmts)-1]
	switch last.Type {
	case shaderir.Return, shaderir.Discard:
		return true
	case shaderir.BlockStmt:
		return isTerminating(last.Blocks[0].Stmts)
	case shaderir.If:
		// An if-statement without an else branch falls through when its condition is false.
		if len(last.Blocks) != 2 {
			return false
		}
		return isTerminating(last.Blocks[0].Stmts) && isTerminating(last.Blocks[1].Stmts)
	}
	// A for-statement never terminates. Every Kage loop has a condition or a range clause, either of which
	// makes a loop non-terminating in Go.
	return false
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
		if !cs.checkIntConstRangeInStmts(stmt.Pos(), ss) {
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
