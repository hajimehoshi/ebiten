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

func (cs *compileState) forceToInt(node ast.Node, expr *shaderir.Expr) bool {
	if !canTruncateToInteger(expr.Const) {
		cs.addError(node.Pos(), fmt.Sprintf("constant %s truncated to integer", expr.Const.String()))
		return false
	}
	expr.Const = gconstant.ToInt(expr.Const)
	expr.ConstType = shaderir.ConstTypeInt
	return true
}

func (cs *compileState) parseStmt(block *block, fname string, stmt ast.Stmt, inParams, outParams []variable, returnType shaderir.Type) ([]shaderir.Stmt, bool) {
	var stmts []shaderir.Stmt

	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		switch stmt.Tok {
		case token.DEFINE:
			if len(stmt.Lhs) != len(stmt.Rhs) && len(stmt.Rhs) != 1 {
				cs.addError(stmt.Pos(), fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
				return nil, false
			}

			ss, ok := cs.assign(block, fname, stmt.Pos(), stmt.Lhs, stmt.Rhs, inParams, true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)
		case token.ASSIGN:
			if len(stmt.Lhs) != len(stmt.Rhs) && len(stmt.Rhs) != 1 {
				cs.addError(stmt.Pos(), fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
				return nil, false
			}
			ss, ok := cs.assign(block, fname, stmt.Pos(), stmt.Lhs, stmt.Rhs, inParams, false)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)
		case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN:
			rhs, rts, ss, ok := cs.parseExpr(block, fname, stmt.Rhs[0], true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			lhs, lts, ss, ok := cs.parseExpr(block, fname, stmt.Lhs[0], true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			var op shaderir.Op
			switch stmt.Tok {
			case token.ADD_ASSIGN:
				op = shaderir.Add
			case token.SUB_ASSIGN:
				op = shaderir.Sub
			case token.MUL_ASSIGN:
				if lts[0].IsMatrix() || rts[0].IsMatrix() {
					op = shaderir.MatrixMul
				} else {
					op = shaderir.ComponentWiseMul
				}
			case token.QUO_ASSIGN:
				op = shaderir.Div
			case token.REM_ASSIGN:
				op = shaderir.ModOp
			}

			if lts[0].Main == rts[0].Main {
				if op == shaderir.Div && rts[0].IsMatrix() {
					cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: operator / not defined on %s", rts[0].String()))
					return nil, false
				}
				if lts[0].Main == shaderir.Int && rhs[0].Const != nil {
					if !cs.forceToInt(stmt, &rhs[0]) {
						return nil, false
					}
				}
			} else {
				switch lts[0].Main {
				case shaderir.Int:
					if !cs.forceToInt(stmt, &rhs[0]) {
						return nil, false
					}
				case shaderir.Float:
					if rhs[0].Const != nil &&
						rhs[0].ConstType != shaderir.ConstTypeInt &&
						gconstant.ToFloat(rhs[0].Const).Kind() != gconstant.Unknown {
						rhs[0].Const = gconstant.ToFloat(rhs[0].Const)
						rhs[0].ConstType = shaderir.ConstTypeFloat
					} else {
						cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: mismatched types %s and %s", lts[0].String(), rts[0].String()))
						return nil, false
					}
				case shaderir.Vec2, shaderir.Vec3, shaderir.Vec4, shaderir.Mat2, shaderir.Mat3, shaderir.Mat4:
					if (op == shaderir.MatrixMul || op == shaderir.Div) && rts[0].Main == shaderir.Float {
						// OK
					} else if lts[0].IsVector() && rts[0].Main == shaderir.Float {
						// OK
					} else if op == shaderir.MatrixMul && ((lts[0].Main == shaderir.Vec2 && rts[0].Main == shaderir.Mat2) ||
						(lts[0].Main == shaderir.Vec3 && rts[0].Main == shaderir.Mat3) ||
						(lts[0].Main == shaderir.Vec4 && rts[0].Main == shaderir.Mat4)) {
						// OK
					} else if (op == shaderir.MatrixMul || op == shaderir.ComponentWiseMul || lts[0].IsVector()) &&
						rhs[0].Const != nil &&
						rhs[0].ConstType != shaderir.ConstTypeInt &&
						gconstant.ToFloat(rhs[0].Const).Kind() != gconstant.Unknown {
						rhs[0].Const = gconstant.ToFloat(rhs[0].Const)
						rhs[0].ConstType = shaderir.ConstTypeFloat
					} else {
						cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: mismatched types %s and %s", lts[0].String(), rts[0].String()))
						return nil, false
					}
				default:
					cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: mismatched types %s and %s", lts[0].String(), rts[0].String()))
					return nil, false
				}
			}

			if op == shaderir.ModOp && lts[0].Main != shaderir.Int {
				cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: operator %% not defined on %s", lts[0].String()))
				return nil, false
			}

			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Assign,
				Exprs: []shaderir.Expr{
					lhs[0],
					{
						Type: shaderir.Binary,
						Op:   op,
						Exprs: []shaderir.Expr{
							lhs[0],
							rhs[0],
						},
					},
				},
			})
		default:
			cs.addError(stmt.Pos(), fmt.Sprintf("unexpected token: %s", stmt.Tok))
		}
	case *ast.BlockStmt:
		b, ok := cs.parseBlock(block, fname, stmt.List, inParams, outParams, returnType, true)
		if !ok {
			return nil, false
		}
		stmts = append(stmts, shaderir.Stmt{
			Type: shaderir.BlockStmt,
			Blocks: []*shaderir.Block{
				b.ir,
			},
		})
	case *ast.DeclStmt:
		ss, ok := cs.parseDecl(block, fname, stmt.Decl)
		if !ok {
			return nil, false
		}
		stmts = append(stmts, ss...)

	case *ast.ForStmt:
		msg := "for-statement must follow this format: for (varname) := (constant); (varname) (op) (constant); (varname) (op) (constant) { ..."
		if stmt.Init == nil {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if stmt.Cond == nil {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if stmt.Post == nil {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}

		// Create a new pseudo block for the initial statement, so that the counter variable belongs to the
		// new pseudo block for each for-loop. Without this, the samely named counter variables in different
		// for-loops confuses the parser.
		pseudoBlock, ok := cs.parseBlock(block, fname, []ast.Stmt{stmt.Init}, inParams, outParams, returnType, false)
		if !ok {
			return nil, false
		}
		ss := pseudoBlock.ir.Stmts

		if len(ss) != 1 {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if ss[0].Type != shaderir.Assign {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if ss[0].Exprs[0].Type != shaderir.LocalVariable {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		varidx := ss[0].Exprs[0].Index
		if ss[0].Exprs[1].Const == nil {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}

		vartype := pseudoBlock.vars[0].typ
		init := ss[0].Exprs[1].Const

		exprs, ts, ss, ok := cs.parseExpr(pseudoBlock, fname, stmt.Cond, true)
		if !ok {
			return nil, false
		}
		if len(exprs) != 1 {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if len(ts) != 1 || ts[0].Main != shaderir.Bool {
			cs.addError(stmt.Pos(), "for-statement's condition must be bool")
			return nil, false
		}
		if len(ss) != 0 {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if exprs[0].Type != shaderir.Binary {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		op := exprs[0].Op
		if op != shaderir.LessThanOp && op != shaderir.LessThanEqualOp && op != shaderir.GreaterThanOp && op != shaderir.GreaterThanEqualOp && op != shaderir.EqualOp && op != shaderir.NotEqualOp {
			cs.addError(stmt.Pos(), "for-statement's condition must have one of these operators: <, <=, >, >=, ==, !=")
			return nil, false
		}
		if exprs[0].Exprs[0].Type != shaderir.LocalVariable {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if exprs[0].Exprs[0].Index != varidx {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if exprs[0].Exprs[1].Const == nil {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		end := exprs[0].Exprs[1].Const

		postSs, ok := cs.parseStmt(pseudoBlock, fname, stmt.Post, inParams, outParams, returnType)
		if !ok {
			return nil, false
		}
		if len(postSs) != 1 {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if postSs[0].Type != shaderir.Assign {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if postSs[0].Exprs[0].Type != shaderir.LocalVariable {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if postSs[0].Exprs[0].Index != varidx {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if postSs[0].Exprs[1].Type != shaderir.Binary {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if postSs[0].Exprs[1].Exprs[0].Type != shaderir.LocalVariable {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if postSs[0].Exprs[1].Exprs[0].Index != varidx {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		if postSs[0].Exprs[1].Exprs[1].Const == nil {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		delta := postSs[0].Exprs[1].Exprs[1].Const
		switch postSs[0].Exprs[1].Op {
		case shaderir.Add:
		case shaderir.Sub:
			delta = gconstant.UnaryOp(token.SUB, delta, 0)
		default:
			cs.addError(stmt.Pos(), "for-statement's post statement must have one of these operators: +=, -=, ++, --")
			return nil, false
		}

		b, ok := cs.parseBlock(pseudoBlock, fname, []ast.Stmt{stmt.Body}, inParams, outParams, returnType, true)
		if !ok {
			return nil, false
		}
		bodyir := b.ir
		for len(bodyir.Stmts) == 1 && bodyir.Stmts[0].Type == shaderir.BlockStmt {
			bodyir = bodyir.Stmts[0].Blocks[0]
		}

		// As the pseudo block is not actually used, copy the variable part to the actual block.
		// This must be done after parsing the for-loop is done, or the duplicated variables confuses the
		// parsing.
		v := pseudoBlock.vars[0]
		v.forLoopCounter = true
		block.vars = append(block.vars, v)

		stmts = append(stmts, shaderir.Stmt{
			Type:        shaderir.For,
			Blocks:      []*shaderir.Block{bodyir},
			ForVarType:  vartype,
			ForVarIndex: varidx,
			ForInit:     init,
			ForEnd:      end,
			ForOp:       op,
			ForDelta:    delta,
		})

	case *ast.IfStmt:
		if stmt.Init != nil {
			init := stmt.Init
			stmt.Init = nil
			b, ok := cs.parseBlock(block, fname, []ast.Stmt{init, stmt}, inParams, outParams, returnType, true)
			if !ok {
				return nil, false
			}

			stmts = append(stmts, shaderir.Stmt{
				Type:   shaderir.BlockStmt,
				Blocks: []*shaderir.Block{b.ir},
			})
			return stmts, true
		}

		exprs, ts, ss, ok := cs.parseExpr(block, fname, stmt.Cond, true)
		if !ok {
			return nil, false
		}
		if len(ts) != 1 || ts[0].Main != shaderir.Bool {
			var tss []string
			for _, t := range ts {
				tss = append(tss, t.String())
			}
			cs.addError(stmt.Pos(), fmt.Sprintf("if-condition must be bool but: %s", strings.Join(tss, ", ")))
			return nil, false
		}
		stmts = append(stmts, ss...)

		var bs []*shaderir.Block
		b, ok := cs.parseBlock(block, fname, stmt.Body.List, inParams, outParams, returnType, true)
		if !ok {
			return nil, false
		}
		bs = append(bs, b.ir)

		if stmt.Else != nil {
			switch s := stmt.Else.(type) {
			case *ast.BlockStmt:
				b, ok := cs.parseBlock(block, fname, s.List, inParams, outParams, returnType, true)
				if !ok {
					return nil, false
				}
				bs = append(bs, b.ir)
			default:
				b, ok := cs.parseBlock(block, fname, []ast.Stmt{s}, inParams, outParams, returnType, true)
				if !ok {
					return nil, false
				}
				bs = append(bs, b.ir)
			}
		}

		stmts = append(stmts, shaderir.Stmt{
			Type:   shaderir.If,
			Exprs:  exprs,
			Blocks: bs,
		})

	case *ast.IncDecStmt:
		exprs, _, ss, ok := cs.parseExpr(block, fname, stmt.X, true)
		if !ok {
			return nil, false
		}
		stmts = append(stmts, ss...)
		var op shaderir.Op
		switch stmt.Tok {
		case token.INC:
			op = shaderir.Add
		case token.DEC:
			op = shaderir.Sub
		}
		stmts = append(stmts, shaderir.Stmt{
			Type: shaderir.Assign,
			Exprs: []shaderir.Expr{
				exprs[0],
				{
					Type: shaderir.Binary,
					Op:   op,
					Exprs: []shaderir.Expr{
						exprs[0],
						{
							Type:      shaderir.NumberExpr,
							Const:     gconstant.MakeInt64(1),
							ConstType: shaderir.ConstTypeInt,
						},
					},
				},
			},
		})

	case *ast.ReturnStmt:
		if len(stmt.Results) != len(outParams) && len(stmt.Results) != 1 {
			if !(len(stmt.Results) == 0 && len(outParams) > 0 && outParams[0].name != "") {
				// TODO: Check variable shadowings.
				// https://golang.org/ref/spec#Return_statements
				cs.addError(stmt.Pos(), fmt.Sprintf("the number of returning variables must be %d but %d", len(outParams), len(stmt.Results)))
				return nil, false
			}
		}

		var exprs []shaderir.Expr
		var types []shaderir.Type
		for _, r := range stmt.Results {
			es, ts, ss, ok := cs.parseExpr(block, fname, r, true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if len(es) > 1 && (len(stmt.Results) > 1 || len(outParams) == 1) {
				cs.addError(r.Pos(), "single-value context and multiple-value context cannot be mixed")
				return nil, false
			}

			if len(outParams) > 1 && len(stmt.Results) == 1 {
				if len(es) == 1 {
					cs.addError(stmt.Pos(), fmt.Sprintf("the number of returning variables must be %d but %d", len(outParams), len(stmt.Results)))
					return nil, false
				}
				if len(es) > 1 && len(es) != len(outParams) {
					cs.addError(stmt.Pos(), fmt.Sprintf("the number of returning variables must be %d but %d", len(outParams), len(es)))
					return nil, false
				}
			}

			exprs = append(exprs, es...)
			types = append(types, ts...)
		}

		for i, t := range types {
			expr := exprs[i]
			var outT shaderir.Type
			if len(outParams) == 0 {
				outT = returnType
			} else {
				outT = outParams[i].typ
			}
			if expr.Const != nil {
				switch outT.Main {
				case shaderir.Int:
					if !cs.forceToInt(stmt, &expr) {
						return nil, false
					}
					t = shaderir.Type{Main: shaderir.Int}
				case shaderir.Float:
					t = shaderir.Type{Main: shaderir.Float}
				}
			}

			if !t.Equal(&outT) {
				cs.addError(stmt.Pos(), fmt.Sprintf("cannot use type %s as type %s in return argument", t.String(), &outT))
				return nil, false
			}

			if len(outParams) > 0 {
				stmts = append(stmts, shaderir.Stmt{
					Type: shaderir.Assign,
					Exprs: []shaderir.Expr{
						{
							Type:  shaderir.LocalVariable,
							Index: len(inParams) + i,
						},
						expr,
					},
				})
			} else {
				stmts = append(stmts, shaderir.Stmt{
					Type:  shaderir.Return,
					Exprs: []shaderir.Expr{expr},
				})
				// When a return type is specified, there should be only one expr here.
				break
			}
		}

		if len(outParams) > 0 {
			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Return,
			})
		}

	case *ast.BranchStmt:
		switch stmt.Tok {
		case token.BREAK:
			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Break,
			})
		case token.CONTINUE:
			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Continue,
			})
		default:
			cs.addError(stmt.Pos(), fmt.Sprintf("invalid token: %s", stmt.Tok))
			return nil, false
		}

	case *ast.ExprStmt:
		if _, ok := stmt.X.(*ast.CallExpr); !ok {
			cs.addError(stmt.Pos(), fmt.Sprintf("the statement is evaluated but not used"))
			return nil, false
		}

		exprs, _, ss, ok := cs.parseExpr(block, fname, stmt.X, true)
		if !ok {
			return nil, false
		}
		stmts = append(stmts, ss...)

		for _, expr := range exprs {
			// There can be a non-call expr like LocalVariable expressions.
			// These are necessary to be used as arguments for an outside function callers.
			if expr.Type != shaderir.Call {
				continue
			}
			if expr.Exprs[0].Type == shaderir.BuiltinFuncExpr {
				cs.addError(stmt.Pos(), fmt.Sprintf("the statement is evaluated but not used"))
				return nil, false
			}
			stmts = append(stmts, shaderir.Stmt{
				Type:  shaderir.ExprStmt,
				Exprs: []shaderir.Expr{expr},
			})
		}

	default:
		cs.addError(stmt.Pos(), fmt.Sprintf("unexpected statement: %#v", stmt))
		return nil, false
	}
	return stmts, true
}

func (cs *compileState) assign(block *block, fname string, pos token.Pos, lhs, rhs []ast.Expr, inParams []variable, define bool) ([]shaderir.Stmt, bool) {
	var stmts []shaderir.Stmt
	var rhsExprs []shaderir.Expr
	var rhsTypes []shaderir.Type
	allblank := true

	for i, e := range lhs {
		if len(lhs) == len(rhs) {
			// Prase RHS first for the order of the statements.
			r, rts, ss, ok := cs.parseExpr(block, fname, rhs[i], true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if define {
				name := e.(*ast.Ident).Name
				if name != "_" {
					for _, v := range block.vars {
						if v.name == name {
							cs.addError(pos, fmt.Sprintf("duplicated local variable name: %s", name))
							return nil, false
						}
					}
				}
				ts, ok := cs.functionReturnTypes(block, rhs[i])
				if !ok {
					ts = rts
				}
				if len(ts) > 1 {
					cs.addError(pos, fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
					return nil, false
				}

				t := ts[0]
				if t.Main == shaderir.None {
					t = toDefaultType(r[0].Const)
				}
				block.addNamedLocalVariable(name, t, e.Pos())
			}

			if len(r) > 1 {
				cs.addError(pos, fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
				return nil, false
			}

			l, lts, ss, ok := cs.parseExpr(block, fname, lhs[i], false)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if l[0].Type == shaderir.Blank {
				continue
			}

			var isAssignmentForbidden func(e *shaderir.Expr) bool
			isAssignmentForbidden = func(e *shaderir.Expr) bool {
				switch e.Type {
				case shaderir.UniformVariable:
					return true
				case shaderir.LocalVariable:
					if fname == cs.vertexEntry || fname == cs.fragmentEntry {
						return e.Index < len(inParams)
					}
				case shaderir.FieldSelector:
					return isAssignmentForbidden(&e.Exprs[0])
				case shaderir.Index:
					return isAssignmentForbidden(&e.Exprs[0])
				}
				return false
			}

			if isAssignmentForbidden(&l[0]) {
				cs.addError(pos, fmt.Sprintf("a uniform variable cannot be assigned"))
				return nil, false
			}
			allblank = false

			if r[0].Const != nil {
				t, ok := block.findLocalVariableByIndex(l[0].Index)
				if !ok {
					cs.addError(pos, fmt.Sprintf("unexpected local variable index: %d", l[0].Index))
					return nil, false
				}
				switch t.Main {
				case shaderir.Int:
					r[0].ConstType = shaderir.ConstTypeInt
				case shaderir.Float:
					r[0].ConstType = shaderir.ConstTypeFloat
				}
			}

			for i := range lts {
				if !canAssign(&r[i], &lts[i], &rts[i]) {
					cs.addError(pos, fmt.Sprintf("cannot use type %s as type %s in variable declaration", rts[i].String(), lts[i].String()))
					return nil, false
				}
			}

			if len(lhs) == 1 {
				stmts = append(stmts, shaderir.Stmt{
					Type:  shaderir.Assign,
					Exprs: []shaderir.Expr{l[0], r[0]},
				})
			} else {
				// For variable swapping, use temporary variables.
				t := rts[0]
				if t.Main == shaderir.None {
					t = toDefaultType(r[0].Const)
				}
				block.vars = append(block.vars, variable{
					typ: t,
				})
				idx := len(block.vars) - 1
				stmts = append(stmts,
					shaderir.Stmt{
						Type: shaderir.Assign,
						Exprs: []shaderir.Expr{
							{
								Type:  shaderir.LocalVariable,
								Index: idx,
							},
							r[0],
						},
					},
					shaderir.Stmt{
						Type: shaderir.Assign,
						Exprs: []shaderir.Expr{
							l[0],
							{
								Type:  shaderir.LocalVariable,
								Index: idx,
							},
						},
					})
			}
		} else {
			if i == 0 {
				var ss []shaderir.Stmt
				var ok bool
				rhsExprs, rhsTypes, ss, ok = cs.parseExpr(block, fname, rhs[0], true)
				if !ok {
					return nil, false
				}
				if len(rhsExprs) != len(lhs) {
					cs.addError(pos, fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
				}
				stmts = append(stmts, ss...)
			}

			if define {
				name := e.(*ast.Ident).Name
				if name != "_" {
					for _, v := range block.vars {
						if v.name == name {
							cs.addError(pos, fmt.Sprintf("duplicated local variable name: %s", name))
							return nil, false
						}
					}
				}
				t := rhsTypes[i]
				if t.Main == shaderir.None {
					// TODO: This is to determine a type when the rhs values are constants (not literals),
					// but there are no actual cases when len(lhs) != len(rhs). Is this correct?
					t = toDefaultType(rhsExprs[i].Const)
				}
				block.addNamedLocalVariable(name, t, e.Pos())
			}

			l, lts, ss, ok := cs.parseExpr(block, fname, lhs[i], false)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if l[0].Type == shaderir.Blank {
				continue
			}
			allblank = false

			for i, lt := range lts {
				if !canAssign(&rhsExprs[i], &lt, &rhsTypes[i]) {
					cs.addError(pos, fmt.Sprintf("cannot use type %s as type %s in variable declaration", rhsTypes[i].String(), lt.String()))
					return nil, false
				}
			}

			stmts = append(stmts, shaderir.Stmt{
				Type:  shaderir.Assign,
				Exprs: []shaderir.Expr{l[0], rhsExprs[i]},
			})
		}
	}

	if define && allblank {
		cs.addError(pos, fmt.Sprintf("no new variables on left side of :="))
		return nil, false
	}

	return stmts, true
}

func toDefaultType(v gconstant.Value) shaderir.Type {
	switch v.Kind() {
	case gconstant.Bool:
		return shaderir.Type{Main: shaderir.Bool}
	case gconstant.Int:
		return shaderir.Type{Main: shaderir.Int}
	case gconstant.Float:
		return shaderir.Type{Main: shaderir.Float}
	}
	// TODO: Should this be an error?
	return shaderir.Type{}
}

func canAssign(re *shaderir.Expr, lt *shaderir.Type, rt *shaderir.Type) bool {
	if lt.Equal(rt) {
		return true
	}
	if re.Const != nil {
		switch lt.Main {
		case shaderir.Bool:
			return re.Const.Kind() == gconstant.Bool
		case shaderir.Int:
			return canTruncateToInteger(re.Const)
		case shaderir.Float:
			return gconstant.ToFloat(re.Const).Kind() != gconstant.Unknown
		}
	}
	return false
}
