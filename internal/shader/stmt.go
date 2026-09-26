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
	return true
}

func (cs *compileState) parseStmt(block *block, fname string, stmt ast.Stmt, inParams, outParams []variable, returnType shaderir.Type) ([]shaderir.Stmt, bool) {
	var stmts []shaderir.Stmt

	switch stmt := stmt.(type) {
	case *ast.AssignStmt:
		switch stmt.Tok {
		case token.DEFINE:
			if len(stmt.Lhs) != len(stmt.Rhs) && len(stmt.Rhs) != 1 {
				cs.addError(stmt.Pos(), "single-value context and multiple-value context cannot be mixed")
				return nil, false
			}

			ss, ok := cs.assign(block, fname, stmt.Pos(), stmt.Lhs, stmt.Rhs, true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)
		case token.ASSIGN:
			if len(stmt.Lhs) != len(stmt.Rhs) && len(stmt.Rhs) != 1 {
				cs.addError(stmt.Pos(), "single-value context and multiple-value context cannot be mixed")
				return nil, false
			}
			ss, ok := cs.assign(block, fname, stmt.Pos(), stmt.Lhs, stmt.Rhs, false)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)
		case token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN, token.AND_ASSIGN, token.OR_ASSIGN, token.XOR_ASSIGN, token.AND_NOT_ASSIGN, token.SHL_ASSIGN, token.SHR_ASSIGN:
			if len(stmt.Lhs) != 1 || len(stmt.Rhs) != 1 {
				cs.addError(stmt.Pos(), "single-value context and multiple-value context cannot be mixed")
				return nil, false
			}

			rhs, rts, ss, ok := cs.parseExpr(block, fname, stmt.Rhs[0], true)
			if !ok {
				return nil, false
			}
			if len(rhs) != 1 || len(rts) != 1 {
				cs.addError(stmt.Rhs[0].Pos(), fmt.Sprintf("the right-hand side of %s must be a single value", stmt.Tok))
				return nil, false
			}
			stmts = append(stmts, ss...)

			lhs, lts, ss, ok := cs.parseExpr(block, fname, stmt.Lhs[0], true)
			if !ok {
				return nil, false
			}
			if len(lhs) != 1 || len(lts) != 1 {
				cs.addError(stmt.Pos(), fmt.Sprintf("the left-hand side of %s must be a single value", stmt.Tok))
				return nil, false
			}
			stmts = append(stmts, ss...)

			if !cs.checkAssignmentTarget(stmt.Pos(), &lhs[0]) {
				return nil, false
			}

			if (stmt.Tok == token.QUO_ASSIGN || stmt.Tok == token.REM_ASSIGN) && isConstZero(rhs[0].Const) {
				cs.addError(stmt.Rhs[0].Pos(), "division by zero")
				return nil, false
			}

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
			case token.AND_ASSIGN:
				op = shaderir.And
			case token.AND_NOT_ASSIGN:
				op = shaderir.AndNot
			case token.OR_ASSIGN:
				op = shaderir.Or
			case token.XOR_ASSIGN:
				op = shaderir.Xor
			case token.SHL_ASSIGN:
				op = shaderir.LeftShift
			case token.SHR_ASSIGN:
				op = shaderir.RightShift
			default:
				cs.addError(stmt.Pos(), fmt.Sprintf("unexpected token: %s", stmt.Tok))
				return nil, false
			}

			if lts[0].Main == rts[0].Main {
				if lts[0].Main != shaderir.Int && lts[0].Main != shaderir.Float && !lts[0].IsIntVector() && !lts[0].IsFloatVector() && !lts[0].IsMatrix() {
					cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: operator %s not defined on %s", stmt.Tok, lts[0].String()))
					return nil, false
				}
				if op == shaderir.Div && rts[0].IsMatrix() {
					cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: operator / not defined on %s", rts[0].String()))
					return nil, false
				}
				if op == shaderir.And || op == shaderir.AndNot || op == shaderir.Or || op == shaderir.Xor || op == shaderir.LeftShift || op == shaderir.RightShift {
					if lts[0].Main != shaderir.Int && !lts[0].IsIntVector() {
						cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: operator %s not defined on %s", stmt.Tok, lts[0].String()))
						return nil, false
					}
					if rts[0].Main != shaderir.Int && !rts[0].IsIntVector() {
						cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: operator %s not defined on %s", stmt.Tok, rts[0].String()))
						return nil, false
					}
				}
				if lts[0].Main == shaderir.Int && rhs[0].Const != nil {
					if !cs.forceToInt(stmt, &rhs[0]) {
						return nil, false
					}
				}
			} else {
				switch lts[0].Main {
				case shaderir.Int, shaderir.IVec2, shaderir.IVec3, shaderir.IVec4:
					if rts[0].Main != shaderir.Int {
						if !rts[0].Equal(&shaderir.Type{}) {
							cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: mismatched types %s and %s", lts[0].String(), typeString(rts[0], rhs[0].Const)))
							return nil, false
						}
						if !cs.forceToInt(stmt, &rhs[0]) {
							return nil, false
						}
					}
				case shaderir.Float:
					if op == shaderir.And || op == shaderir.AndNot || op == shaderir.Or || op == shaderir.Xor || op == shaderir.LeftShift || op == shaderir.RightShift {
						cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: operator %s not defined on %s", stmt.Tok, lts[0].String()))
					} else if rhs[0].Const != nil &&
						(rts[0].Main == shaderir.None || rts[0].Main == shaderir.Float) &&
						gconstant.ToFloat(rhs[0].Const).Kind() != gconstant.Unknown {
						rhs[0].Const = gconstant.ToFloat(rhs[0].Const)
					} else {
						cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: mismatched types %s and %s", lts[0].String(), typeString(rts[0], rhs[0].Const)))
						return nil, false
					}
				case shaderir.Vec2, shaderir.Vec3, shaderir.Vec4, shaderir.Mat2, shaderir.Mat3, shaderir.Mat4:
					if op == shaderir.And || op == shaderir.AndNot || op == shaderir.Or || op == shaderir.Xor || op == shaderir.LeftShift || op == shaderir.RightShift {
						cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: operator %s not defined on %s", stmt.Tok, lts[0].String()))
					} else if (op == shaderir.MatrixMul || op == shaderir.Div) &&
						(rts[0].Main == shaderir.Float ||
							(rhs[0].Const != nil &&
								(rts[0].Main == shaderir.None || rts[0].Main == shaderir.Float) &&
								gconstant.ToFloat(rhs[0].Const).Kind() != gconstant.Unknown)) {
						if rhs[0].Const != nil {
							rhs[0].Const = gconstant.ToFloat(rhs[0].Const)
						}
					} else if op == shaderir.MatrixMul && ((lts[0].Main == shaderir.Vec2 && rts[0].Main == shaderir.Mat2) ||
						(lts[0].Main == shaderir.Vec3 && rts[0].Main == shaderir.Mat3) ||
						(lts[0].Main == shaderir.Vec4 && rts[0].Main == shaderir.Mat4)) {
						// OK
					} else if (op == shaderir.MatrixMul || op == shaderir.ComponentWiseMul || lts[0].IsFloatVector()) &&
						(rts[0].Main == shaderir.Float ||
							(rhs[0].Const != nil &&
								(rts[0].Main == shaderir.None || rts[0].Main == shaderir.Float) &&
								gconstant.ToFloat(rhs[0].Const).Kind() != gconstant.Unknown)) {
						if rhs[0].Const != nil {
							rhs[0].Const = gconstant.ToFloat(rhs[0].Const)
						}
					} else {
						cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: mismatched types %s and %s", lts[0].String(), typeString(rts[0], rhs[0].Const)))
						return nil, false
					}
				default:
					cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: mismatched types %s and %s", lts[0].String(), typeString(rts[0], rhs[0].Const)))
					return nil, false
				}
			}

			if op == shaderir.ModOp && lts[0].Main != shaderir.Int && lts[0].Main != shaderir.IVec2 && lts[0].Main != shaderir.IVec3 && lts[0].Main != shaderir.IVec4 {
				cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation: operator %% not defined on %s", lts[0].String()))
				return nil, false
			}

			if (op == shaderir.LeftShift || op == shaderir.RightShift) && rhs[0].Const != nil {
				if _, ok := cs.shiftCount(stmt.Pos(), stmt.Tok, rhs[0].Const); !ok {
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
		ss, ok := cs.parseFor(block, fname, stmt, inParams, outParams, returnType, true)
		if !ok {
			return nil, false
		}
		stmts = append(stmts, ss...)

	case *ast.RangeStmt:
		ss, ok := cs.parseForRange(block, fname, stmt, inParams, outParams, returnType)
		if !ok {
			return nil, false
		}
		stmts = append(stmts, ss...)

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
		if len(ts) != 1 {
			var tss []string
			for _, t := range ts {
				tss = append(tss, t.String())
			}
			cs.addError(stmt.Cond.Pos(), fmt.Sprintf("if-condition must be bool but: %s", strings.Join(tss, ", ")))
			return nil, false
		}
		if !(ts[0].Main == shaderir.Bool || (ts[0].Main == shaderir.None && exprs[0].Const != nil && exprs[0].Const.Kind() == gconstant.Bool)) {
			cs.addError(stmt.Cond.Pos(), fmt.Sprintf("if-condition must be bool but: %s", typeString(ts[0], exprs[0].Const)))
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
		exprs, ts, ss, ok := cs.parseExpr(block, fname, stmt.X, true)
		if !ok {
			return nil, false
		}
		if len(exprs) != 1 || len(ts) != 1 {
			cs.addError(stmt.Pos(), fmt.Sprintf("the operand of %s must be a single value", stmt.Tok))
			return nil, false
		}
		if !cs.checkAssignmentTarget(stmt.Pos(), &exprs[0]) {
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
		var c gconstant.Value
		switch {
		case ts[0].Main == shaderir.Int, ts[0].IsIntVector():
			c = gconstant.MakeInt64(1)
		case ts[0].Main == shaderir.Float, ts[0].IsFloatVector():
			c = gconstant.MakeFloat64(1)
		default:
			cs.addError(stmt.Pos(), fmt.Sprintf("invalid operation %s (non-numeric type %s)", stmt.Tok.String(), ts[0].String()))
			return nil, false
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
							Type:  shaderir.NumberExpr,
							Const: c,
						},
					},
				},
			},
		})

	case *ast.ReturnStmt:
		want := len(outParams)
		if want == 0 && returnType.Main != shaderir.None {
			want = 1
		}
		if len(stmt.Results) != len(outParams) && len(stmt.Results) != 1 {
			if !(len(stmt.Results) == 0 && len(outParams) > 0 && outParams[0].name != "") {
				// TODO: Check variable shadowings.
				// https://go.dev/ref/spec#Return_statements
				cs.addError(returnCountErrorPos(stmt, len(stmt.Results), want), fmt.Sprintf("the number of returning variables must be %d but %d", want, len(stmt.Results)))
				return nil, false
			}
		}

		var exprs []shaderir.Expr
		var types []shaderir.Type
		var positions []token.Pos
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
					cs.addError(returnCountErrorPos(stmt, len(stmt.Results), want), fmt.Sprintf("the number of returning variables must be %d but %d", want, len(stmt.Results)))
					return nil, false
				}
				if len(es) > 1 && len(es) != len(outParams) {
					cs.addError(returnCountErrorPos(stmt, len(es), want), fmt.Sprintf("the number of returning variables must be %d but %d", want, len(es)))
					return nil, false
				}
			}

			exprs = append(exprs, es...)
			types = append(types, ts...)
			for range es {
				positions = append(positions, r.Pos())
			}
		}

		if len(stmt.Results) > 0 {
			got := len(exprs)
			if want == 0 {
				// A function returning nothing must not return even a call yielding no values.
				got = len(stmt.Results)
			}
			if got != want {
				cs.addError(returnCountErrorPos(stmt, got, want), fmt.Sprintf("the number of returning variables must be %d but %d", want, got))
				return nil, false
			}
		}

		for i, t := range types {
			expr := exprs[i]
			var outT shaderir.Type
			if len(outParams) == 0 {
				outT = returnType
			} else {
				outT = outParams[i].typ
			}
			result := "return argument"
			if len(types) > 1 {
				result = fmt.Sprintf("the %s return argument", ordinal(i+1))
			}
			if expr.Const != nil {
				switch outT.Main {
				case shaderir.Bool:
					if expr.Const.Kind() != gconstant.Bool {
						cs.addError(positions[i], fmt.Sprintf("cannot use type %s as type %s in %s", typeString(t, expr.Const), &outT, result))
						return nil, false
					}
					t = shaderir.Type{Main: shaderir.Bool}
				case shaderir.Int:
					if gconstant.ToInt(expr.Const).Kind() == gconstant.Unknown {
						cs.addError(positions[i], fmt.Sprintf("cannot use type %s as type %s in %s", typeString(t, expr.Const), &outT, result))
						return nil, false
					}
					expr.Const = gconstant.ToInt(expr.Const)
					t = shaderir.Type{Main: shaderir.Int}
				case shaderir.Float:
					if gconstant.ToFloat(expr.Const).Kind() == gconstant.Unknown {
						cs.addError(positions[i], fmt.Sprintf("cannot use type %s as type %s in %s", typeString(t, expr.Const), &outT, result))
						return nil, false
					}
					expr.Const = gconstant.ToFloat(expr.Const)
					t = shaderir.Type{Main: shaderir.Float}
				}
			}

			if !t.Equal(&outT) {
				cs.addError(positions[i], fmt.Sprintf("cannot use type %s as type %s in %s", typeString(t, expr.Const), &outT, result))
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
		} else if len(stmt.Results) == 0 {
			if returnType.Main != shaderir.None {
				cs.addError(stmt.Pos(), "cannot use a bare return in a function that returns a value")
				return nil, false
			}
			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Return,
			})
		}

	case *ast.BranchStmt:
		switch stmt.Tok {
		case token.BREAK:
			if stmt.Label != nil {
				cs.addError(stmt.Pos(), "break with a label is not supported")
				return nil, false
			}
			if !block.inLoop() {
				cs.addError(stmt.Pos(), "break is not in a loop")
				return nil, false
			}
			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Break,
			})
		case token.CONTINUE:
			if stmt.Label != nil {
				cs.addError(stmt.Pos(), "continue with a label is not supported")
				return nil, false
			}
			if !block.inLoop() {
				cs.addError(stmt.Pos(), "continue is not in a loop")
				return nil, false
			}
			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Continue,
			})
		default:
			cs.addError(stmt.Pos(), fmt.Sprintf("invalid token: %s", stmt.Tok))
			return nil, false
		}

	case *ast.LabeledStmt:
		cs.addError(stmt.Pos(), "labeled statement is not supported")
		return nil, false

	case *ast.ExprStmt:
		if _, ok := stmt.X.(*ast.CallExpr); !ok {
			cs.addError(stmt.Pos(), "the statement is evaluated but not used")
			return nil, false
		}

		exprs, _, ss, ok := cs.parseExpr(block, fname, stmt.X, true)
		if !ok {
			return nil, false
		}
		stmts = append(stmts, ss...)

		for _, expr := range exprs {
			// There can be a non-call expr like LocalVariable expressions.
			// These are necessary to be used as arguments for callers of an outside function.
			if expr.Type != shaderir.Call {
				continue
			}
			if expr.Exprs[0].Type == shaderir.BuiltinFuncExpr {
				cs.addError(stmt.Pos(), "the statement is evaluated but not used")
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

// returnCountErrorPos returns where to report stmt returning got values instead of want: the first extra
// result expression if stmt has more than want of them, the last result expression otherwise, or stmt itself
// if it has none.
func returnCountErrorPos(stmt *ast.ReturnStmt, got, want int) token.Pos {
	n := len(stmt.Results)
	switch {
	case n == 0:
		return stmt.Pos()
	case got > want && n > want:
		return stmt.Results[want].Pos()
	default:
		return stmt.Results[n-1].Pos()
	}
}

func (cs *compileState) assign(block *block, fname string, pos token.Pos, lhs, rhs []ast.Expr, define bool) ([]shaderir.Stmt, bool) {
	var stmts []shaderir.Stmt
	var rhsExprs []shaderir.Expr
	var rhsTypes []shaderir.Type
	var newVariable bool
	lhsNames := map[string]struct{}{}

	if len(lhs) == len(rhs) {
		var localVariablIndicesToAssignLater []int
		var leftExprsToAssignLater []shaderir.Expr
		for i, e := range lhs {
			// Parse RHS first for the order of the statements.
			r, rts, ss, ok := cs.parseExpr(block, fname, rhs[i], true)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if define {
				if _, ok := e.(*ast.Ident); !ok {
					cs.addError(e.Pos(), "non-name on the left side of :=")
					return nil, false
				}
				name := e.(*ast.Ident).Name
				if name != "_" {
					if _, ok := lhsNames[name]; ok {
						cs.addError(e.Pos(), fmt.Sprintf("%s repeated on left side of :=", name))
						return nil, false
					}
					lhsNames[name] = struct{}{}
				}
				// A name already declared in this block is assigned instead of declared.
				if !declared(name, block.vars, block.consts) {
					ts, ok := cs.functionReturnTypes(block, rhs[i])
					if !ok {
						ts = rts
					}
					if len(ts) > 1 {
						cs.addError(rhs[i].Pos(), "single-value context and multiple-value context cannot be mixed")
						return nil, false
					}
					if len(ts) == 0 {
						cs.addError(rhs[i].Pos(), "the right-hand side of := has no value")
						return nil, false
					}
					t := ts[0]
					if t.Main == shaderir.None {
						t = toDefaultType(r[0].Const)
					}
					block.addNamedLocalVariable(name, t, e.Pos())
					if name != "_" {
						newVariable = true
					}
				}
			}

			if len(r) > 1 {
				cs.addError(rhs[i].Pos(), "single-value context and multiple-value context cannot be mixed")
				return nil, false
			}

			l, lts, ss, ok := cs.parseExpr(block, fname, lhs[i], false)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if len(l) != len(r) {
				if len(r) == 0 {
					cs.addError(rhs[i].Pos(), "right-hand side (no value) used as value")
				} else {
					cs.addError(rhs[i].Pos(), fmt.Sprintf("assignment mismatch: %d variables but the right-hand side has %d values", len(l), len(r)))
				}
				return nil, false
			}

			if l[0].Type == shaderir.Blank {
				continue
			}

			if !cs.checkAssignmentTarget(e.Pos(), &l[0]) {
				return nil, false
			}

			for i := range lts {
				if !canAssign(&lts[i], &rts[i], r[i].Const) {
					cs.addError(e.Pos(), fmt.Sprintf("cannot use type %s as type %s in assignment", typeString(rts[i], r[i].Const), lts[i].String()))
					return nil, false
				}
				if r[i].Const != nil {
					switch lts[0].Main {
					case shaderir.Int:
						r[i].Const = gconstant.ToInt(r[i].Const)
					case shaderir.Float:
						r[i].Const = gconstant.ToFloat(r[i].Const)
					}
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
				idx := block.totalLocalVariableCount() - 1
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
					})
				localVariablIndicesToAssignLater = append(localVariablIndicesToAssignLater, idx)
				leftExprsToAssignLater = append(leftExprsToAssignLater, l[0])
			}
		}
		for i, idx := range localVariablIndicesToAssignLater {
			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Assign,
				Exprs: []shaderir.Expr{
					leftExprsToAssignLater[i],
					{
						Type:  shaderir.LocalVariable,
						Index: idx,
					},
				},
			})
		}
	} else {
		var ss []shaderir.Stmt
		var ok bool
		rhsExprs, rhsTypes, ss, ok = cs.parseExpr(block, fname, rhs[0], true)
		if !ok {
			return nil, false
		}
		if len(lhs) != len(rhsExprs) {
			cs.addError(pos, fmt.Sprintf("assignment mismatch: %d variables but %d", len(lhs), len(rhsExprs)))
			return nil, false
		}
		stmts = append(stmts, ss...)

		for i, e := range lhs {
			if define {
				if _, ok := e.(*ast.Ident); !ok {
					cs.addError(e.Pos(), "non-name on the left side of :=")
					return nil, false
				}
				name := e.(*ast.Ident).Name
				if name != "_" {
					if _, ok := lhsNames[name]; ok {
						cs.addError(e.Pos(), fmt.Sprintf("%s repeated on left side of :=", name))
						return nil, false
					}
					lhsNames[name] = struct{}{}
				}
				// A name already declared in this block is assigned instead of declared.
				if !declared(name, block.vars, block.consts) {
					t := rhsTypes[i]
					if t.Main == shaderir.None {
						// TODO: This is to determine a type when the rhs values are constants (not literals),
						// but there are no actual cases when len(lhs) != len(rhs). Is this correct?
						t = toDefaultType(rhsExprs[i].Const)
					}
					block.addNamedLocalVariable(name, t, e.Pos())
					if name != "_" {
						newVariable = true
					}
				}
			}

			l, lts, ss, ok := cs.parseExpr(block, fname, lhs[i], false)
			if !ok {
				return nil, false
			}
			stmts = append(stmts, ss...)

			if len(l) != 1 {
				cs.addError(pos, fmt.Sprintf("unexpected count of expressions in lhs: %d", len(l)))
				return nil, false
			}
			if len(lts) != 1 {
				cs.addError(pos, fmt.Sprintf("unexpected count of types in lhs: %d", len(lts)))
				return nil, false
			}

			if l[0].Type == shaderir.Blank {
				continue
			}

			if !cs.checkAssignmentTarget(e.Pos(), &l[0]) {
				return nil, false
			}

			if !canAssign(&lts[0], &rhsTypes[i], rhsExprs[i].Const) {
				cs.addError(e.Pos(), fmt.Sprintf("cannot use type %s as type %s in assignment", rhsTypes[i].String(), lts[0].String()))
				return nil, false
			}

			stmts = append(stmts, shaderir.Stmt{
				Type:  shaderir.Assign,
				Exprs: []shaderir.Expr{l[0], rhsExprs[i]},
			})
		}
	}

	if define && !newVariable {
		cs.addError(pos, "no new variables on left side of :=")
		return nil, false
	}

	return stmts, true
}

// checkAssignmentTarget reports whether e can be assigned, and adds an error if not.
func (cs *compileState) checkAssignmentTarget(pos token.Pos, e *shaderir.Expr) bool {
	switch e.Type {
	case shaderir.LocalVariable:
		return true
	case shaderir.UniformVariable:
		cs.addError(pos, "a uniform variable cannot be assigned")
		return false
	case shaderir.TextureVariable:
		cs.addError(pos, "a texture variable cannot be assigned")
		return false
	case shaderir.FieldSelector:
		if s := e.Exprs[1]; s.Type == shaderir.SwizzlingExpr && hasDuplicatedSwizzlingComponent(s.Swizzling) {
			cs.addError(pos, fmt.Sprintf("cannot assign to a swizzling with a duplicated component: %s", s.SourceSwizzling()))
			return false
		}
		return cs.checkAssignmentTarget(pos, &e.Exprs[0])
	case shaderir.Index:
		return cs.checkAssignmentTarget(pos, &e.Exprs[0])
	case shaderir.FunctionExpr, shaderir.BuiltinFuncExpr:
		cs.addError(pos, "a function cannot be assigned")
		return false
	case shaderir.Call:
		cs.addError(pos, "a function call cannot be assigned")
		return false
	default:
		cs.addError(pos, "a non-variable expression cannot be assigned")
		return false
	}
}

func hasDuplicatedSwizzlingComponent(swizzling string) bool {
	for i := range len(swizzling) {
		if strings.IndexByte(swizzling[i+1:], swizzling[i]) >= 0 {
			return true
		}
	}
	return false
}

// isConstZero reports whether v is a numeric constant equal to zero.
func isConstZero(v gconstant.Value) bool {
	if v == nil {
		return false
	}
	switch v.Kind() {
	case gconstant.Int, gconstant.Float:
		return gconstant.Sign(v) == 0
	}
	return false
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

// typeString returns the name of t, the type of an expression whose constant value is c.
// For an untyped constant, the name is "untyped " followed by the name of the constant's default type.
func typeString(t shaderir.Type, c gconstant.Value) string {
	if t.Main == shaderir.None && c != nil {
		if d := toDefaultType(c); d.Main != shaderir.None {
			return "untyped " + d.String()
		}
	}
	return t.String()
}

func canAssign(lt *shaderir.Type, rt *shaderir.Type, rc gconstant.Value) bool {
	if lt.Equal(rt) {
		return true
	}

	if rc == nil {
		return false
	}

	if !rt.Equal(&shaderir.Type{}) {
		return false
	}

	switch lt.Main {
	case shaderir.Bool:
		return rc.Kind() == gconstant.Bool
	case shaderir.Int:
		return gconstant.ToInt(rc).Kind() != gconstant.Unknown
	case shaderir.Float:
		return gconstant.ToFloat(rc).Kind() != gconstant.Unknown
	}

	return false
}

func (cs *compileState) parseFor(block *block, fname string, stmt *ast.ForStmt, inParams, outParams []variable, returnType shaderir.Type, checkLocalVariableUsage bool) ([]shaderir.Stmt, bool) {
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
	// new pseudo block for each for-loop. Without this, the same-named counter variables in different
	// for-loops confuse the parser.
	pseudoBlock, ok := cs.parseBlock(block, fname, []ast.Stmt{stmt.Init}, inParams, outParams, returnType, false)
	if !ok {
		return nil, false
	}
	pseudoBlock.loop = true
	ss := pseudoBlock.ir.Stmts

	if len(ss) != 1 {
		cs.addError(stmt.Init.Pos(), msg)
		return nil, false
	}
	if ss[0].Type != shaderir.Assign {
		cs.addError(stmt.Init.Pos(), msg)
		return nil, false
	}
	if ss[0].Exprs[0].Type != shaderir.LocalVariable {
		cs.addError(stmt.Init.Pos(), msg)
		return nil, false
	}
	varidx := ss[0].Exprs[0].Index
	if ss[0].Exprs[1].Const == nil {
		cs.addError(stmt.Init.Pos(), msg)
		return nil, false
	}

	if len(pseudoBlock.vars) != 1 {
		cs.addError(stmt.Init.Pos(), msg)
		return nil, false
	}

	vartype := pseudoBlock.vars[0].typ
	init := ss[0].Exprs[1].Const

	exprs, ts, ss, ok := cs.parseExpr(pseudoBlock, fname, stmt.Cond, true)
	if !ok {
		return nil, false
	}
	if len(exprs) != 1 {
		cs.addError(stmt.Cond.Pos(), msg)
		return nil, false
	}
	if len(ts) != 1 || ts[0].Main != shaderir.Bool {
		cs.addError(stmt.Cond.Pos(), "for-statement's condition must be bool")
		return nil, false
	}
	if len(ss) != 0 {
		cs.addError(stmt.Cond.Pos(), msg)
		return nil, false
	}
	if exprs[0].Type != shaderir.Binary {
		cs.addError(stmt.Cond.Pos(), msg)
		return nil, false
	}
	op := exprs[0].Op
	if op != shaderir.LessThanOp && op != shaderir.LessThanEqualOp && op != shaderir.GreaterThanOp && op != shaderir.GreaterThanEqualOp && op != shaderir.EqualOp && op != shaderir.NotEqualOp {
		cs.addError(stmt.Cond.Pos(), "for-statement's condition must have one of these operators: <, <=, >, >=, ==, !=")
		return nil, false
	}
	if exprs[0].Exprs[0].Type != shaderir.LocalVariable {
		cs.addError(stmt.Cond.Pos(), msg)
		return nil, false
	}
	if exprs[0].Exprs[0].Index != varidx {
		cs.addError(stmt.Cond.Pos(), msg)
		return nil, false
	}
	if exprs[0].Exprs[1].Const == nil {
		cs.addError(stmt.Cond.Pos(), msg)
		return nil, false
	}
	end := exprs[0].Exprs[1].Const

	postSs, ok := cs.parseStmt(pseudoBlock, fname, stmt.Post, inParams, outParams, returnType)
	if !ok {
		return nil, false
	}
	if len(postSs) != 1 {
		cs.addError(stmt.Post.Pos(), msg)
		return nil, false
	}
	if postSs[0].Type != shaderir.Assign {
		cs.addError(stmt.Post.Pos(), msg)
		return nil, false
	}
	if postSs[0].Exprs[0].Type != shaderir.LocalVariable {
		cs.addError(stmt.Post.Pos(), msg)
		return nil, false
	}
	if postSs[0].Exprs[0].Index != varidx {
		cs.addError(stmt.Post.Pos(), msg)
		return nil, false
	}
	if postSs[0].Exprs[1].Type != shaderir.Binary {
		cs.addError(stmt.Post.Pos(), msg)
		return nil, false
	}
	if postSs[0].Exprs[1].Exprs[0].Type != shaderir.LocalVariable {
		cs.addError(stmt.Post.Pos(), msg)
		return nil, false
	}
	if postSs[0].Exprs[1].Exprs[0].Index != varidx {
		cs.addError(stmt.Post.Pos(), msg)
		return nil, false
	}
	if postSs[0].Exprs[1].Exprs[1].Const == nil {
		cs.addError(stmt.Post.Pos(), msg)
		return nil, false
	}
	delta := postSs[0].Exprs[1].Exprs[1].Const
	switch postSs[0].Exprs[1].Op {
	case shaderir.Add:
	case shaderir.Sub:
		delta = gconstant.UnaryOp(token.SUB, delta, 0)
	default:
		cs.addError(stmt.Post.Pos(), "for-statement's post statement must have one of these operators: +=, -=, ++, --")
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
	// This must be done after parsing the for-loop is done, or the duplicated variables confuse the
	// parsing.
	// The scope of the counter variable ends with this for-loop. Clear its name so that the
	// variable is neither found nor checked by its name anymore. The variable itself is still kept
	// for the local-variable indices.
	v := pseudoBlock.vars[0]
	v.forLoopCounter = true
	v.name = ""
	block.vars = append(block.vars, v)

	return []shaderir.Stmt{
		{
			Type:        shaderir.For,
			Blocks:      []*shaderir.Block{bodyir},
			ForVarType:  vartype,
			ForVarIndex: varidx,
			ForInit:     init,
			ForEnd:      end,
			ForOp:       op,
			ForDelta:    delta,
		},
	}, true
}

// parseRangeVar returns the name and the position of an iteration variable of a range-statement.
// A missing variable is treated as a blank identifier.
func (cs *compileState) parseRangeVar(e ast.Expr, defaultPos token.Pos, msg string) (string, token.Pos, bool) {
	if e == nil {
		return "_", defaultPos, true
	}
	ident, ok := e.(*ast.Ident)
	if !ok {
		cs.addError(e.Pos(), msg)
		return "", 0, false
	}
	return ident.Name, ident.Pos(), true
}

func (cs *compileState) parseForRange(block *block, fname string, stmt *ast.RangeStmt, inParams, outParams []variable, returnType shaderir.Type) ([]shaderir.Stmt, bool) {
	msg := "range-statement must follow this format: for (varname) := range (constant integer or array) { ..."

	exprs, ts, ss, ok := cs.parseExpr(block, fname, stmt.X, true)
	if !ok {
		return nil, false
	}
	if len(exprs) != 1 || len(ts) != 1 {
		cs.addError(stmt.X.Pos(), msg)
		return nil, false
	}
	if len(ss) != 0 {
		cs.addError(stmt.X.Pos(), "a range expression must be a constant or a variable")
		return nil, false
	}
	t := ts[0]
	if t.Main == shaderir.None && exprs[0].Const != nil {
		t = toDefaultType(exprs[0].Const)
	}

	var end gconstant.Value
	var elmType shaderir.Type
	switch t.Main {
	case shaderir.Int:
		if stmt.Value != nil {
			cs.addError(stmt.Value.Pos(), "range over an integer permits only one iteration variable")
			return nil, false
		}
		if exprs[0].Const == nil {
			cs.addError(stmt.X.Pos(), "a range expression must be a constant")
			return nil, false
		}
		end = gconstant.ToInt(exprs[0].Const)
	case shaderir.Array:
		// The array is indexed at each iteration, so the range expression must be one that can be
		// evaluated repeatedly.
		if stmt.Value != nil && exprs[0].Type != shaderir.LocalVariable && exprs[0].Type != shaderir.UniformVariable {
			cs.addError(stmt.X.Pos(), "a range expression must be a variable to use the second iteration variable")
			return nil, false
		}
		end = gconstant.MakeInt64(int64(t.Length))
		elmType = t.Sub[0]
	default:
		cs.addError(stmt.X.Pos(), fmt.Sprintf("cannot range over a value of type %s", t.String()))
		return nil, false
	}

	keyname, keypos, ok := cs.parseRangeVar(stmt.Key, stmt.Pos(), msg)
	if !ok {
		return nil, false
	}
	valname, valpos, ok := cs.parseRangeVar(stmt.Value, stmt.Pos(), msg)
	if !ok {
		return nil, false
	}
	switch stmt.Tok {
	case token.DEFINE:
		if keyname == "_" && valname == "_" {
			cs.addError(keypos, "no new variables on left side of :=")
			return nil, false
		}
	case token.ASSIGN:
		// A range-statement introduces its own iteration variables, so an existing variable cannot be
		// one of them.
		if keyname != "_" || valname != "_" {
			cs.addError(keypos, msg)
			return nil, false
		}
	}

	// Create a new pseudo block for the iteration variables, so that the variables belong to the new
	// pseudo block for each for-loop. Without this, the same-named variables in different for-loops
	// confuse the parser.
	pseudoBlock, ok := cs.parseBlock(block, fname, nil, inParams, outParams, returnType, false)
	if !ok {
		return nil, false
	}
	pseudoBlock.loop = true
	vartype := shaderir.Type{Main: shaderir.Int}
	varidx := pseudoBlock.totalLocalVariableCount()
	pseudoBlock.addNamedLocalVariable(keyname, vartype, keypos)
	validx := pseudoBlock.totalLocalVariableCount()
	if stmt.Value != nil {
		pseudoBlock.addNamedLocalVariable(valname, elmType, valpos)
	}

	b, ok := cs.parseBlock(pseudoBlock, fname, []ast.Stmt{stmt.Body}, inParams, outParams, returnType, true)
	if !ok {
		return nil, false
	}
	for i, v := range pseudoBlock.vars {
		if pos, ok := pseudoBlock.unusedVars[i]; ok {
			cs.addError(pos, fmt.Sprintf("local variable %s is not used", v.name))
			return nil, false
		}
	}

	bodyir := b.ir
	for len(bodyir.Stmts) == 1 && bodyir.Stmts[0].Type == shaderir.BlockStmt {
		bodyir = bodyir.Stmts[0].Blocks[0]
	}
	if stmt.Value != nil {
		bodyir.Stmts = append([]shaderir.Stmt{
			{
				Type: shaderir.Assign,
				Exprs: []shaderir.Expr{
					{
						Type:  shaderir.LocalVariable,
						Index: validx,
					},
					{
						Type: shaderir.Index,
						Exprs: []shaderir.Expr{
							exprs[0],
							{
								Type:  shaderir.LocalVariable,
								Index: varidx,
							},
						},
					},
				},
			},
		}, bodyir.Stmts...)
	}

	// As the pseudo block is not actually used, copy the variable part to the actual block.
	// This must be done after parsing the for-loop is done, or the duplicated variables confuse the
	// parsing.
	// The scopes of the iteration variables end with this for-loop. Clear their names so that the
	// variables are neither found nor checked by their names anymore. The variables themselves are
	// still kept for the local-variable indices.
	counter := pseudoBlock.vars[0]
	counter.forLoopCounter = true
	counter.name = ""
	block.vars = append(block.vars, counter)
	if stmt.Value != nil {
		value := pseudoBlock.vars[1]
		value.name = ""
		block.vars = append(block.vars, value)
	}

	return []shaderir.Stmt{
		{
			Type:        shaderir.For,
			Blocks:      []*shaderir.Block{bodyir},
			ForVarType:  vartype,
			ForVarIndex: varidx,
			ForInit:     gconstant.MakeInt64(0),
			ForEnd:      end,
			ForOp:       shaderir.LessThanOp,
			ForDelta:    gconstant.MakeInt64(1),
		},
	}, true
}
