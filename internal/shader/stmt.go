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

	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

func (cs *compileState) forceToInt(node ast.Node, expr *shaderir.Expr) bool {
	if !canTruncateToInteger(expr.Const) {
		cs.addError(node.Pos(), fmt.Sprintf("constant %s truncated to integer", expr.Const.String()))
		return false
	}
	expr.ConstType = shaderir.ConstTypeInt
	return true
}

func (cs *compileState) parseStmt(block *block, fname string, stmt ast.Stmt, inParams, outParams []variable) ([]shaderir.Stmt, bool) {
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
			var op shaderir.Op
			switch stmt.Tok {
			case token.ADD_ASSIGN:
				op = shaderir.Add
			case token.SUB_ASSIGN:
				op = shaderir.Sub
			case token.MUL_ASSIGN:
				op = shaderir.Mul
			case token.QUO_ASSIGN:
				op = shaderir.Div
			case token.REM_ASSIGN:
				op = shaderir.ModOp
			}

			rhs, _, ss, ok := cs.parseExpr(block, stmt.Rhs[0], true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			lhs, ts, ss, ok := cs.parseExpr(block, stmt.Lhs[0], true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if rhs[0].Type == shaderir.NumberExpr && ts[0].Main == shaderir.Int {
				if !cs.forceToInt(stmt, &rhs[0]) {
					return nil, false
				}
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
		b, ok := cs.parseBlock(block, fname, stmt.List, inParams, outParams, true)
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
		ss, ok := cs.parseDecl(block, stmt.Decl)
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
		pseudoBlock, ok := cs.parseBlock(block, fname, []ast.Stmt{stmt.Init}, inParams, outParams, false)
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
		if ss[0].Exprs[1].Type != shaderir.NumberExpr {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}

		vartype := pseudoBlock.vars[0].typ
		init := ss[0].Exprs[1].Const

		exprs, ts, ss, ok := cs.parseExpr(pseudoBlock, stmt.Cond, true)
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
		if exprs[0].Exprs[1].Type != shaderir.NumberExpr {
			cs.addError(stmt.Pos(), msg)
			return nil, false
		}
		end := exprs[0].Exprs[1].Const

		postSs, ok := cs.parseStmt(pseudoBlock, fname, stmt.Post, inParams, outParams)
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
		if postSs[0].Exprs[1].Exprs[1].Type != shaderir.NumberExpr {
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

		b, ok := cs.parseBlock(pseudoBlock, fname, []ast.Stmt{stmt.Body}, inParams, outParams, true)
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
			b, ok := cs.parseBlock(block, fname, []ast.Stmt{init, stmt}, inParams, outParams, true)
			if !ok {
				return nil, false
			}

			stmts = append(stmts, shaderir.Stmt{
				Type:   shaderir.BlockStmt,
				Blocks: []*shaderir.Block{b.ir},
			})
			return stmts, true
		}

		exprs, ts, ss, ok := cs.parseExpr(block, stmt.Cond, true)
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
		b, ok := cs.parseBlock(block, fname, stmt.Body.List, inParams, outParams, true)
		if !ok {
			return nil, false
		}
		bs = append(bs, b.ir)

		if stmt.Else != nil {
			switch s := stmt.Else.(type) {
			case *ast.BlockStmt:
				b, ok := cs.parseBlock(block, fname, s.List, inParams, outParams, true)
				if !ok {
					return nil, false
				}
				bs = append(bs, b.ir)
			default:
				b, ok := cs.parseBlock(block, fname, []ast.Stmt{s}, inParams, outParams, true)
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
		exprs, _, ss, ok := cs.parseExpr(block, stmt.X, true)
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

		for i, r := range stmt.Results {
			exprs, ts, ss, ok := cs.parseExpr(block, r, true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if len(exprs) > 1 {
				if len(stmt.Results) > 1 || len(outParams) == 1 {
					cs.addError(r.Pos(), "single-value context and multiple-value context cannot be mixed")
					return nil, false
				}
			}

			if len(outParams) > 1 && len(stmt.Results) == 1 {
				if len(exprs) == 1 {
					cs.addError(stmt.Pos(), fmt.Sprintf("the number of returning variables must be %d but %d", len(outParams), len(stmt.Results)))
					return nil, false
				}
				if len(exprs) > 1 && len(exprs) != len(outParams) {
					cs.addError(stmt.Pos(), fmt.Sprintf("the number of returning variables must be %d but %d", len(outParams), len(exprs)))
					return nil, false
				}
			}

			for j, t := range ts {
				expr := exprs[j]
				if expr.Type == shaderir.NumberExpr {
					switch outParams[i+j].typ.Main {
					case shaderir.Int:
						if !cs.forceToInt(stmt, &expr) {
							return nil, false
						}
						t = shaderir.Type{Main: shaderir.Int}
					case shaderir.Float:
						t = shaderir.Type{Main: shaderir.Float}
					}
				}

				if !t.Equal(&outParams[i+j].typ) {
					cs.addError(stmt.Pos(), fmt.Sprintf("cannot use type %s as type %s in return argument", &t, &outParams[i].typ))
					return nil, false
				}

				stmts = append(stmts, shaderir.Stmt{
					Type: shaderir.Assign,
					Exprs: []shaderir.Expr{
						{
							Type:  shaderir.LocalVariable,
							Index: len(inParams) + i + j,
						},
						expr,
					},
				})
			}
		}
		stmts = append(stmts, shaderir.Stmt{
			Type: shaderir.Return,
		})

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
		exprs, _, ss, ok := cs.parseExpr(block, stmt.X, true)
		if !ok {
			return nil, false
		}
		stmts = append(stmts, ss...)

		for _, expr := range exprs {
			if expr.Type != shaderir.Call {
				continue
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
			r, origts, ss, ok := cs.parseExpr(block, rhs[i], true)
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
					ts = origts
				}
				if len(ts) > 1 {
					cs.addError(pos, fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
					return nil, false
				}

				block.addNamedLocalVariable(name, ts[0], e.Pos())
			}

			if len(r) > 1 {
				cs.addError(pos, fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
				return nil, false
			}

			l, _, ss, ok := cs.parseExpr(block, lhs[i], false)
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

			if r[0].Type == shaderir.NumberExpr {
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

			if len(lhs) == 1 {
				stmts = append(stmts, shaderir.Stmt{
					Type:  shaderir.Assign,
					Exprs: []shaderir.Expr{l[0], r[0]},
				})
			} else {
				// For variable swapping, use temporary variables.
				block.vars = append(block.vars, variable{
					typ: origts[0],
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
				rhsExprs, rhsTypes, ss, ok = cs.parseExpr(block, rhs[0], true)
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
				block.addNamedLocalVariable(name, rhsTypes[i], e.Pos())
			}

			l, _, ss, ok := cs.parseExpr(block, lhs[i], false)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if l[0].Type == shaderir.Blank {
				continue
			}
			allblank = false

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
