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
	"strings"

	"github.com/hajimehoshi/ebiten/internal/shaderir"
)

func (cs *compileState) parseStmt(block *block, stmt ast.Stmt, inParams []variable) ([]shaderir.Stmt, bool) {
	var stmts []shaderir.Stmt

	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		switch stmt.Tok {
		case token.DEFINE:
			if len(stmt.Lhs) != len(stmt.Rhs) && len(stmt.Rhs) != 1 {
				cs.addError(stmt.Pos(), fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
				return nil, false
			}

			ss, ok := cs.assign(block, stmt.Pos(), stmt.Lhs, stmt.Rhs, true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)
		case token.ASSIGN:
			// TODO: What about the statement `a,b = b,a?`
			if len(stmt.Lhs) != len(stmt.Rhs) && len(stmt.Rhs) != 1 {
				cs.addError(stmt.Pos(), fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
				return nil, false
			}
			ss, ok := cs.assign(block, stmt.Pos(), stmt.Lhs, stmt.Rhs, false)
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

			rhs, _, ss, ok := cs.parseExpr(block, stmt.Rhs[0])
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			lhs, _, ss, ok := cs.parseExpr(block, stmt.Lhs[0])
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

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
		b, ok := cs.parseBlock(block, stmt.List, inParams, nil)
		if !ok {
			return nil, false
		}
		stmts = append(stmts, shaderir.Stmt{
			Type: shaderir.BlockStmt,
			Blocks: []shaderir.Block{
				b.ir,
			},
		})
	case *ast.DeclStmt:
		if !cs.parseDecl(block, stmt.Decl) {
			return nil, false
		}
	case *ast.IfStmt:
		if stmt.Init != nil {
			init := stmt.Init
			stmt.Init = nil
			b, ok := cs.parseBlock(block, []ast.Stmt{init, stmt}, inParams, nil)
			if !ok {
				return nil, false
			}

			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.BlockStmt,
				Blocks: []shaderir.Block{
					b.ir,
				},
			})
			return stmts, true
		}

		exprs, ts, ss, ok := cs.parseExpr(block, stmt.Cond)
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

		var bs []shaderir.Block
		b, ok := cs.parseBlock(block, stmt.Body.List, inParams, nil)
		if !ok {
			return nil, false
		}
		bs = append(bs, b.ir)

		if stmt.Else != nil {
			switch s := stmt.Else.(type) {
			case *ast.BlockStmt:
				b, ok := cs.parseBlock(block, s.List, inParams, nil)
				if !ok {
					return nil, false
				}
				bs = append(bs, b.ir)
			default:
				b, ok := cs.parseBlock(block, []ast.Stmt{s}, inParams, nil)
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
	case *ast.ReturnStmt:
		for i, r := range stmt.Results {
			exprs, _, ss, ok := cs.parseExpr(block, r)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)
			if len(exprs) == 0 {
				continue
			}
			if len(exprs) > 1 {
				cs.addError(r.Pos(), "multiple-context with return is not implemented yet")
				continue
			}
			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Assign,
				Exprs: []shaderir.Expr{
					{
						Type:  shaderir.LocalVariable,
						Index: len(inParams) + i,
					},
					exprs[0],
				},
			})
		}
		stmts = append(stmts, shaderir.Stmt{
			Type: shaderir.Return,
		})
	case *ast.ExprStmt:
		exprs, _, ss, ok := cs.parseExpr(block, stmt.X)
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

func (cs *compileState) assign(block *block, pos token.Pos, lhs, rhs []ast.Expr, define bool) ([]shaderir.Stmt, bool) {
	var stmts []shaderir.Stmt
	var rhsExprs []shaderir.Expr
	var rhsTypes []shaderir.Type

	for i, e := range lhs {
		if len(lhs) == len(rhs) {
			// Prase RHS first for the order of the statements.
			r, origts, ss, ok := cs.parseExpr(block, rhs[i])
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if define {
				v := variable{
					name: e.(*ast.Ident).Name,
				}
				ts, ok := cs.functionReturnTypes(block, rhs[i])
				if !ok {
					ts = origts
				}
				if len(ts) > 1 {
					cs.addError(pos, fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
					return nil, false
				}
				if len(ts) == 1 {
					v.typ = ts[0]
				}
				block.vars = append(block.vars, v)
			}

			if len(r) > 1 {
				cs.addError(pos, fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
				return nil, false
			}

			l, _, ss, ok := cs.parseExpr(block, lhs[i])
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if r[0].Type == shaderir.NumberExpr {
				switch block.vars[l[0].Index].typ.Main {
				case shaderir.Int:
					r[0].ConstType = shaderir.ConstTypeInt
				case shaderir.Float:
					r[0].ConstType = shaderir.ConstTypeFloat
				}
			}

			stmts = append(stmts, shaderir.Stmt{
				Type:  shaderir.Assign,
				Exprs: []shaderir.Expr{l[0], r[0]},
			})
		} else {
			if i == 0 {
				var ss []shaderir.Stmt
				var ok bool
				rhsExprs, rhsTypes, ss, ok = cs.parseExpr(block, rhs[0])
				if !ok {
					return nil, false
				}
				if len(rhsExprs) != len(lhs) {
					cs.addError(pos, fmt.Sprintf("single-value context and multiple-value context cannot be mixed"))
				}
				stmts = append(stmts, ss...)
			}

			if define {
				v := variable{
					name: e.(*ast.Ident).Name,
				}
				v.typ = rhsTypes[i]
				block.vars = append(block.vars, v)
			}

			l, _, ss, ok := cs.parseExpr(block, lhs[i])
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			stmts = append(stmts, shaderir.Stmt{
				Type:  shaderir.Assign,
				Exprs: []shaderir.Expr{l[0], rhsExprs[i]},
			})
		}
	}
	return stmts, true
}
