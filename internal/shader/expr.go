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
	"regexp"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func canTruncateToInteger(v gconstant.Value) bool {
	return gconstant.ToInt(v).Kind() != gconstant.Unknown
}

var textureVariableRe = regexp.MustCompile(`\A__t(\d+)\z`)

func (cs *compileState) parseExpr(block *block, expr ast.Expr, markLocalVariableUsed bool) ([]shaderir.Expr, []shaderir.Type, []shaderir.Stmt, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			return []shaderir.Expr{
				{
					Type:  shaderir.NumberExpr,
					Const: gconstant.MakeFromLiteral(e.Value, e.Kind, 0),
				},
			}, []shaderir.Type{{Main: shaderir.Int}}, nil, true
		case token.FLOAT:
			return []shaderir.Expr{
				{
					Type:  shaderir.NumberExpr,
					Const: gconstant.MakeFromLiteral(e.Value, e.Kind, 0),
				},
			}, []shaderir.Type{{Main: shaderir.Float}}, nil, true
		default:
			cs.addError(e.Pos(), fmt.Sprintf("literal not implemented: %#v", e))
		}

	case *ast.BinaryExpr:
		var stmts []shaderir.Stmt

		// Prase LHS first for the order of the statements.
		lhs, ts, ss, ok := cs.parseExpr(block, e.X, markLocalVariableUsed)
		if !ok {
			return nil, nil, nil, false
		}
		if len(lhs) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at a binary operator: %s", e.X))
			return nil, nil, nil, false
		}
		stmts = append(stmts, ss...)
		lhst := ts[0]

		rhs, ts, ss, ok := cs.parseExpr(block, e.Y, markLocalVariableUsed)
		if !ok {
			return nil, nil, nil, false
		}
		if len(rhs) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at a binary operator: %s", e.Y))
			return nil, nil, nil, false
		}
		stmts = append(stmts, ss...)
		rhst := ts[0]

		if lhs[0].Type == shaderir.NumberExpr && rhs[0].Type == shaderir.NumberExpr {
			op := e.Op
			// https://golang.org/pkg/go/constant/#BinaryOp
			// "To force integer division of Int operands, use op == token.QUO_ASSIGN instead of
			// token.QUO; the result is guaranteed to be Int in this case."
			if op == token.QUO && lhs[0].Const.Kind() == gconstant.Int && rhs[0].Const.Kind() == gconstant.Int {
				op = token.QUO_ASSIGN
			}
			var v gconstant.Value
			var t shaderir.Type
			switch op {
			case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
				v = gconstant.MakeBool(gconstant.Compare(lhs[0].Const, op, rhs[0].Const))
				t = shaderir.Type{Main: shaderir.Bool}
			default:
				v = gconstant.BinaryOp(lhs[0].Const, op, rhs[0].Const)
				if v.Kind() == gconstant.Float {
					t = shaderir.Type{Main: shaderir.Float}
				} else {
					t = shaderir.Type{Main: shaderir.Int}
				}
			}

			return []shaderir.Expr{
				{
					Type:  shaderir.NumberExpr,
					Const: v,
				},
			}, []shaderir.Type{t}, stmts, true
		}

		op, ok := shaderir.OpFromToken(e.Op)
		if !ok {
			cs.addError(e.Pos(), fmt.Sprintf("unexpected operator: %s", e.Op))
			return nil, nil, nil, false
		}

		var t shaderir.Type
		switch {
		case op == shaderir.LessThanOp || op == shaderir.LessThanEqualOp || op == shaderir.GreaterThanOp || op == shaderir.GreaterThanEqualOp || op == shaderir.EqualOp || op == shaderir.NotEqualOp || op == shaderir.AndAnd || op == shaderir.OrOr:
			t = shaderir.Type{Main: shaderir.Bool}
		case lhs[0].Type == shaderir.NumberExpr && rhs[0].Type != shaderir.NumberExpr:
			if rhst.Main == shaderir.Int {
				if !canTruncateToInteger(lhs[0].Const) {
					cs.addError(e.Pos(), fmt.Sprintf("constant %s truncated to integer", lhs[0].Const.String()))
					return nil, nil, nil, false
				}
				lhs[0].ConstType = shaderir.ConstTypeInt
			}
			t = rhst
		case lhs[0].Type != shaderir.NumberExpr && rhs[0].Type == shaderir.NumberExpr:
			if lhst.Main == shaderir.Int {
				if !canTruncateToInteger(rhs[0].Const) {
					cs.addError(e.Pos(), fmt.Sprintf("constant %s truncated to integer", rhs[0].Const.String()))
					return nil, nil, nil, false
				}
				rhs[0].ConstType = shaderir.ConstTypeInt
			}
			t = lhst
		case lhst.Equal(&rhst):
			t = lhst
		case lhst.Main == shaderir.Float || lhst.Main == shaderir.Int:
			switch rhst.Main {
			case shaderir.Int:
				t = lhst
			case shaderir.Float, shaderir.Vec2, shaderir.Vec3, shaderir.Vec4, shaderir.Mat2, shaderir.Mat3, shaderir.Mat4:
				t = rhst
			default:
				cs.addError(e.Pos(), fmt.Sprintf("types don't match: %s %s %s", lhst.String(), e.Op, rhst.String()))
				return nil, nil, nil, false
			}
		case rhst.Main == shaderir.Float || rhst.Main == shaderir.Int:
			switch lhst.Main {
			case shaderir.Int:
				t = rhst
			case shaderir.Float, shaderir.Vec2, shaderir.Vec3, shaderir.Vec4, shaderir.Mat2, shaderir.Mat3, shaderir.Mat4:
				t = lhst
			default:
				cs.addError(e.Pos(), fmt.Sprintf("types don't match: %s %s %s", lhst.String(), e.Op, rhst.String()))
				return nil, nil, nil, false
			}
		case lhst.Main == shaderir.Vec2 && rhst.Main == shaderir.Mat2 ||
			lhst.Main == shaderir.Mat2 && rhst.Main == shaderir.Vec2:
			t = shaderir.Type{Main: shaderir.Vec2}
		case lhst.Main == shaderir.Vec3 && rhst.Main == shaderir.Mat3 ||
			lhst.Main == shaderir.Mat3 && rhst.Main == shaderir.Vec3:
			t = shaderir.Type{Main: shaderir.Vec3}
		case lhst.Main == shaderir.Vec4 && rhst.Main == shaderir.Mat4 ||
			lhst.Main == shaderir.Mat4 && rhst.Main == shaderir.Vec4:
			t = shaderir.Type{Main: shaderir.Vec4}
		default:
			cs.addError(e.Pos(), fmt.Sprintf("invalid expression: %s %s %s", lhst.String(), e.Op, rhst.String()))
			return nil, nil, nil, false
		}

		return []shaderir.Expr{
			{
				Type:  shaderir.Binary,
				Op:    op,
				Exprs: []shaderir.Expr{lhs[0], rhs[0]},
			},
		}, []shaderir.Type{t}, stmts, true

	case *ast.CallExpr:
		var (
			callee shaderir.Expr
			args   []shaderir.Expr
			argts  []shaderir.Type
			stmts  []shaderir.Stmt
		)

		// Parse the argument first for the order of the statements.
		for _, a := range e.Args {
			es, ts, ss, ok := cs.parseExpr(block, a, markLocalVariableUsed)
			if !ok {
				return nil, nil, nil, false
			}
			if len(es) > 1 && len(e.Args) > 1 {
				cs.addError(e.Pos(), fmt.Sprintf("single-value context and multiple-value context cannot be mixed: %s", e.Fun))
				return nil, nil, nil, false
			}
			args = append(args, es...)
			argts = append(argts, ts...)
			stmts = append(stmts, ss...)
		}

		// TODO: When len(ss) is not 0?
		es, _, ss, ok := cs.parseExpr(block, e.Fun, markLocalVariableUsed)
		if !ok {
			return nil, nil, nil, false
		}
		if len(es) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at a callee: %s", e.Fun))
			return nil, nil, nil, false
		}
		callee = es[0]
		stmts = append(stmts, ss...)

		// For built-in functions, we can call this in this position. Return an expression for the function
		// call.
		if callee.Type == shaderir.BuiltinFuncExpr {
			if callee.BuiltinFunc == shaderir.Len || callee.BuiltinFunc == shaderir.Cap {
				if len(args) != 1 {
					cs.addError(e.Pos(), fmt.Sprintf("number of %s's arguments must be 1 but %d", callee.BuiltinFunc, len(args)))
					return nil, nil, nil, false
				}
				if argts[0].Main != shaderir.Array {
					cs.addError(e.Pos(), fmt.Sprintf("%s takes an array but %s", callee.BuiltinFunc, argts[0].String()))
					return nil, nil, nil, false
				}
				return []shaderir.Expr{
					{
						Type:      shaderir.NumberExpr,
						Const:     gconstant.MakeInt64(int64(argts[0].Length)),
						ConstType: shaderir.ConstTypeInt,
					},
				}, []shaderir.Type{{Main: shaderir.Int}}, stmts, true
			}

			var t shaderir.Type
			switch callee.BuiltinFunc {
			case shaderir.BoolF:
				t = shaderir.Type{Main: shaderir.Bool}
			case shaderir.IntF:
				t = shaderir.Type{Main: shaderir.Int}
			case shaderir.FloatF:
				t = shaderir.Type{Main: shaderir.Float}
			case shaderir.Vec2F:
				t = shaderir.Type{Main: shaderir.Vec2}
			case shaderir.Vec3F:
				t = shaderir.Type{Main: shaderir.Vec3}
			case shaderir.Vec4F:
				t = shaderir.Type{Main: shaderir.Vec4}
			case shaderir.Mat2F:
				t = shaderir.Type{Main: shaderir.Mat2}
			case shaderir.Mat3F:
				t = shaderir.Type{Main: shaderir.Mat3}
			case shaderir.Mat4F:
				t = shaderir.Type{Main: shaderir.Mat4}
			case shaderir.Step:
				t = argts[1]
			case shaderir.Smoothstep:
				t = argts[2]
			case shaderir.Length, shaderir.Distance, shaderir.Dot:
				t = shaderir.Type{Main: shaderir.Float}
			case shaderir.Cross:
				t = shaderir.Type{Main: shaderir.Vec3}
			case shaderir.Texture2DF:
				t = shaderir.Type{Main: shaderir.Vec4}
			default:
				t = argts[0]
			}
			return []shaderir.Expr{
				{
					Type:  shaderir.Call,
					Exprs: append([]shaderir.Expr{callee}, args...),
				},
			}, []shaderir.Type{t}, stmts, true
		}

		if callee.Type != shaderir.FunctionExpr {
			cs.addError(e.Pos(), fmt.Sprintf("function callee must be a funciton name but %s", e.Fun))
			return nil, nil, nil, false
		}

		f := cs.funcs[callee.Index]

		for i, p := range f.ir.InParams {
			if args[i].Type == shaderir.NumberExpr && p.Main == shaderir.Int {
				if !cs.forceToInt(e, &args[i]) {
					return nil, nil, nil, false
				}
			}
		}

		var outParams []int
		for _, p := range f.ir.OutParams {
			idx := block.totalLocalVariableNum()
			block.vars = append(block.vars, variable{
				typ: p,
			})
			args = append(args, shaderir.Expr{
				Type:  shaderir.LocalVariable,
				Index: idx,
			})
			outParams = append(outParams, idx)
		}

		if t := f.ir.Return; t.Main != shaderir.None {
			if len(outParams) != 0 {
				cs.addError(e.Pos(), fmt.Sprintf("a function returning value cannot have out-params so far: %s", e.Fun))
				return nil, nil, nil, false
			}

			idx := block.totalLocalVariableNum()
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
			return []shaderir.Expr{
				{
					Type:  shaderir.LocalVariable,
					Index: idx,
				},
			}, []shaderir.Type{t}, stmts, true
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

		if len(outParams) == 0 {
			// TODO: Is this an error?
		}

		var exprs []shaderir.Expr
		for _, p := range outParams {
			exprs = append(exprs, shaderir.Expr{
				Type:  shaderir.LocalVariable,
				Index: p,
			})
		}
		return exprs, f.ir.OutParams, stmts, true

	case *ast.Ident:
		if e.Name == "_" {
			// In the context where a local variable is marked as used, any expressions must have its
			// meaning. Then, a blank identifier is not available there.
			if markLocalVariableUsed {
				cs.addError(e.Pos(), fmt.Sprintf("cannot use _ as value"))
				return nil, nil, nil, false
			}
			return []shaderir.Expr{
				{
					Type: shaderir.Blank,
				},
			}, []shaderir.Type{{}}, nil, true
		}
		if i, t, ok := block.findLocalVariable(e.Name, markLocalVariableUsed); ok {
			return []shaderir.Expr{
				{
					Type:  shaderir.LocalVariable,
					Index: i,
				},
			}, []shaderir.Type{t}, nil, true
		}
		if c, ok := block.findConstant(e.Name); ok {
			return []shaderir.Expr{
				{
					Type:      shaderir.NumberExpr,
					Const:     c.value,
					ConstType: c.ctyp,
				},
			}, []shaderir.Type{c.typ}, nil, true
		}
		if i, ok := cs.findFunction(e.Name); ok {
			return []shaderir.Expr{
				{
					Type:  shaderir.FunctionExpr,
					Index: i,
				},
			}, nil, nil, true
		}
		if i, ok := cs.findUniformVariable(e.Name); ok {
			return []shaderir.Expr{
				{
					Type:  shaderir.UniformVariable,
					Index: i,
				},
			}, []shaderir.Type{cs.ir.Uniforms[i]}, nil, true
		}
		if f, ok := shaderir.ParseBuiltinFunc(e.Name); ok {
			return []shaderir.Expr{
				{
					Type:        shaderir.BuiltinFuncExpr,
					BuiltinFunc: f,
				},
			}, nil, nil, true
		}
		if m := textureVariableRe.FindStringSubmatch(e.Name); m != nil {
			i, _ := strconv.Atoi(m[1])
			return []shaderir.Expr{
				{
					Type:  shaderir.TextureVariable,
					Index: i,
				},
			}, nil, nil, true
		}
		if e.Name == "true" || e.Name == "false" {
			return []shaderir.Expr{
				{
					Type:  shaderir.NumberExpr,
					Const: gconstant.MakeBool(e.Name == "true"),
				},
			}, []shaderir.Type{{Main: shaderir.Bool}}, nil, true
		}
		cs.addError(e.Pos(), fmt.Sprintf("unexpected identifier: %s", e.Name))

	case *ast.ParenExpr:
		return cs.parseExpr(block, e.X, markLocalVariableUsed)

	case *ast.SelectorExpr:
		exprs, _, stmts, ok := cs.parseExpr(block, e.X, true)
		if !ok {
			return nil, nil, nil, false
		}
		if len(exprs) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at a selector: %s", e.X))
			return nil, nil, nil, false
		}
		var t shaderir.Type
		switch len(e.Sel.Name) {
		case 1:
			t.Main = shaderir.Float
		case 2:
			t.Main = shaderir.Vec2
		case 3:
			t.Main = shaderir.Vec3
		case 4:
			t.Main = shaderir.Vec4
		default:
			cs.addError(e.Pos(), fmt.Sprintf("unexpected swizzling: %s", e.Sel.Name))
			return nil, nil, nil, false
		}
		return []shaderir.Expr{
			{
				Type: shaderir.FieldSelector,
				Exprs: []shaderir.Expr{
					exprs[0],
					{
						Type:      shaderir.SwizzlingExpr,
						Swizzling: e.Sel.Name,
					},
				},
			},
		}, []shaderir.Type{t}, stmts, true

	case *ast.UnaryExpr:
		exprs, t, stmts, ok := cs.parseExpr(block, e.X, markLocalVariableUsed)
		if !ok {
			return nil, nil, nil, false
		}
		if len(exprs) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at a unary operator: %s", e.X))
			return nil, nil, nil, false
		}

		if exprs[0].Type == shaderir.NumberExpr {
			v := gconstant.UnaryOp(e.Op, exprs[0].Const, 0)
			t := shaderir.Type{Main: shaderir.Int}
			if v.Kind() == gconstant.Float {
				t = shaderir.Type{Main: shaderir.Float}
			}
			return []shaderir.Expr{
				{
					Type:  shaderir.NumberExpr,
					Const: v,
				},
			}, []shaderir.Type{t}, stmts, true
		}

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
			return nil, nil, nil, false
		}
		return []shaderir.Expr{
			{
				Type:  shaderir.Unary,
				Op:    op,
				Exprs: exprs,
			},
		}, t, stmts, true

	case *ast.CompositeLit:
		t, ok := cs.parseType(block, e.Type)
		if !ok {
			return nil, nil, nil, false
		}
		if t.Main == shaderir.Array && t.Length == -1 {
			t.Length = len(e.Elts)
		}

		idx := block.totalLocalVariableNum()
		block.vars = append(block.vars, variable{
			typ: t,
		})

		var stmts []shaderir.Stmt
		for i, e := range e.Elts {
			exprs, _, ss, ok := cs.parseExpr(block, e, markLocalVariableUsed)
			if !ok {
				return nil, nil, nil, false
			}
			if len(exprs) != 1 {
				cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at a composite literal"))
				return nil, nil, nil, false
			}
			stmts = append(stmts, ss...)
			stmts = append(stmts, shaderir.Stmt{
				Type: shaderir.Assign,
				Exprs: []shaderir.Expr{
					{
						Type: shaderir.Index,
						Exprs: []shaderir.Expr{
							{
								Type:  shaderir.LocalVariable,
								Index: idx,
							},
							{
								Type:      shaderir.NumberExpr,
								Const:     gconstant.MakeInt64(int64(i)),
								ConstType: shaderir.ConstTypeInt,
							},
						},
					},
					exprs[0],
				},
			})
		}

		return []shaderir.Expr{
			{
				Type:  shaderir.LocalVariable,
				Index: idx,
			},
		}, []shaderir.Type{t}, stmts, true

	case *ast.IndexExpr:
		var stmts []shaderir.Stmt

		// Parse the index first
		exprs, _, ss, ok := cs.parseExpr(block, e.Index, markLocalVariableUsed)
		if !ok {
			return nil, nil, nil, false
		}
		stmts = append(stmts, ss...)

		if len(exprs) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at an index expression"))
			return nil, nil, nil, false
		}
		idx := exprs[0]
		if idx.Type == shaderir.NumberExpr {
			if !canTruncateToInteger(idx.Const) {
				cs.addError(e.Pos(), fmt.Sprintf("constant %s truncated to integer", idx.Const.String()))
				return nil, nil, nil, false
			}
			idx.ConstType = shaderir.ConstTypeInt
		}

		exprs, ts, ss, ok := cs.parseExpr(block, e.X, markLocalVariableUsed)
		if !ok {
			return nil, nil, nil, false
		}
		stmts = append(stmts, ss...)
		if len(exprs) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at an index expression"))
			return nil, nil, nil, false
		}
		x := exprs[0]
		t := ts[0]

		var typ shaderir.Type
		switch t.Main {
		case shaderir.Vec2, shaderir.Vec3, shaderir.Vec4:
			typ = shaderir.Type{Main: shaderir.Float}
		case shaderir.Mat2:
			typ = shaderir.Type{Main: shaderir.Vec2}
		case shaderir.Mat3:
			typ = shaderir.Type{Main: shaderir.Vec3}
		case shaderir.Mat4:
			typ = shaderir.Type{Main: shaderir.Vec4}
		case shaderir.Array:
			typ = t.Sub[0]
		default:
			cs.addError(e.Pos(), fmt.Sprintf("index operator cannot be applied to the type %s", t.String()))
			return nil, nil, nil, false
		}

		return []shaderir.Expr{
			{
				Type: shaderir.Index,
				Exprs: []shaderir.Expr{
					x,
					idx,
				},
			},
		}, []shaderir.Type{typ}, stmts, true

	default:
		cs.addError(e.Pos(), fmt.Sprintf("expression not implemented: %#v", e))
	}
	return nil, nil, nil, false
}
