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
	"strings"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

func canTruncateToInteger(v gconstant.Value) bool {
	return gconstant.ToInt(v).Kind() != gconstant.Unknown
}

func canTruncateToFloat(v gconstant.Value) bool {
	return gconstant.ToFloat(v).Kind() != gconstant.Unknown
}

var textureVariableRe = regexp.MustCompile(`\A__t(\d+)\z`)

func (cs *compileState) parseExpr(block *block, fname string, expr ast.Expr, markLocalVariableUsed bool) ([]shaderir.Expr, []shaderir.Type, []shaderir.Stmt, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		switch e.Kind {
		case token.INT:
			// The type is not determined yet.
			return []shaderir.Expr{
				{
					Type:  shaderir.NumberExpr,
					Const: gconstant.MakeFromLiteral(e.Value, e.Kind, 0),
				},
			}, []shaderir.Type{{}}, nil, true
		case token.FLOAT:
			// The type is not determined yet.
			return []shaderir.Expr{
				{
					Type:  shaderir.NumberExpr,
					Const: gconstant.MakeFromLiteral(e.Value, e.Kind, 0),
				},
			}, []shaderir.Type{{}}, nil, true
		default:
			cs.addError(e.Pos(), fmt.Sprintf("literal not implemented: %#v", e))
		}

	case *ast.BinaryExpr:
		var stmts []shaderir.Stmt

		// Prase LHS first for the order of the statements.
		lhs, ts, ss, ok := cs.parseExpr(block, fname, e.X, markLocalVariableUsed)
		if !ok {
			return nil, nil, nil, false
		}
		if len(lhs) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at a binary operator: %s", e.X))
			return nil, nil, nil, false
		}
		stmts = append(stmts, ss...)
		if len(ts) == 0 {
			cs.addError(e.Pos(), fmt.Sprintf("unexpected binary operator: %s", e.X))
			return nil, nil, nil, false
		}
		lhst := ts[0]

		rhs, ts, ss, ok := cs.parseExpr(block, fname, e.Y, markLocalVariableUsed)
		if !ok {
			return nil, nil, nil, false
		}
		if len(rhs) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at a binary operator: %s", e.Y))
			return nil, nil, nil, false
		}
		stmts = append(stmts, ss...)
		if len(ts) == 0 {
			cs.addError(e.Pos(), fmt.Sprintf("unexpected binary operator: %s", e.Y))
			return nil, nil, nil, false
		}
		rhst := ts[0]

		op := e.Op
		// https://pkg.go.dev/go/constant/#BinaryOp
		// "To force integer division of Int operands, use op == token.QUO_ASSIGN instead of
		// token.QUO; the result is guaranteed to be Int in this case."
		if op == token.QUO && lhs[0].Const != nil && lhs[0].Const.Kind() == gconstant.Int && rhs[0].Const != nil && rhs[0].Const.Kind() == gconstant.Int {
			op = token.QUO_ASSIGN
		}

		op2, ok := shaderir.OpFromToken(e.Op, lhst, rhst)
		if !ok {
			cs.addError(e.Pos(), fmt.Sprintf("unexpected operator: %s", e.Op))
			return nil, nil, nil, false
		}

		// Resolve untyped constants.
		l, r, ok := shaderir.ResolveUntypedConstsForBinaryOp(op2, lhs[0].Const, rhs[0].Const, lhst, rhst)
		if !ok {
			// TODO: Show a better type name for untyped constants.
			cs.addError(e.Pos(), fmt.Sprintf("types don't match: %s %s %s", lhst.String(), op, rhst.String()))
			return nil, nil, nil, false
		}
		lhs[0].Const, rhs[0].Const = l, r

		// If either is typed, resolve the other type.
		// If both are untyped, keep them untyped.
		if lhst.Main != shaderir.None || rhst.Main != shaderir.None {
			if lhs[0].Const != nil {
				switch lhs[0].Const.Kind() {
				case gconstant.Float:
					lhst = shaderir.Type{Main: shaderir.Float}
				case gconstant.Int:
					lhst = shaderir.Type{Main: shaderir.Int}
				case gconstant.Bool:
					lhst = shaderir.Type{Main: shaderir.Bool}
				}
			}
			if rhs[0].Const != nil {
				switch rhs[0].Const.Kind() {
				case gconstant.Float:
					rhst = shaderir.Type{Main: shaderir.Float}
				case gconstant.Int:
					rhst = shaderir.Type{Main: shaderir.Int}
				case gconstant.Bool:
					rhst = shaderir.Type{Main: shaderir.Bool}
				}
			}
		}

		t, ok := shaderir.TypeFromBinaryOp(op2, lhst, rhst, lhs[0].Const, rhs[0].Const)
		if !ok {
			// TODO: Show a better type name for untyped constants.
			cs.addError(e.Pos(), fmt.Sprintf("types don't match: %s %s %s", lhst.String(), op, rhst.String()))
			return nil, nil, nil, false
		}

		if lhs[0].Const != nil && rhs[0].Const != nil {
			var v gconstant.Value
			switch op {
			case token.LAND, token.LOR:
				b := gconstant.BoolVal(gconstant.BinaryOp(lhs[0].Const, op, rhs[0].Const))
				v = gconstant.MakeBool(b)
			case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
				v = gconstant.MakeBool(gconstant.Compare(lhs[0].Const, op, rhs[0].Const))
			case token.SHL, token.SHR:
				shift, ok := gconstant.Int64Val(rhs[0].Const)
				if !ok {
					cs.addError(e.Pos(), fmt.Sprintf("unexpected %s type for: %s", rhs[0].Const.String(), e.Op))
					return nil, nil, nil, false
				}
				v = gconstant.Shift(lhs[0].Const, op, uint(shift))
			default:
				v = gconstant.BinaryOp(lhs[0].Const, op, rhs[0].Const)
			}

			return []shaderir.Expr{
				{
					Type:  shaderir.NumberExpr,
					Const: v,
				},
			}, []shaderir.Type{t}, stmts, true
		}

		return []shaderir.Expr{
			{
				Type:  shaderir.Binary,
				Op:    op2,
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
			es, ts, ss, ok := cs.parseExpr(block, fname, a, markLocalVariableUsed)
			if !ok {
				return nil, nil, nil, false
			}
			if len(es) > 1 && len(e.Args) > 1 {
				cs.addError(e.Pos(), fmt.Sprintf("single-value context and multiple-value context cannot be mixed: %s", e.Fun))
				return nil, nil, nil, false
			}
			for _, expr := range es {
				if expr.Type == shaderir.FunctionExpr || expr.Type == shaderir.BuiltinFuncExpr {
					cs.addError(e.Pos(), fmt.Sprintf("function name cannot be an argument: %s", e.Fun))
					return nil, nil, nil, false
				}
			}
			args = append(args, es...)
			argts = append(argts, ts...)
			stmts = append(stmts, ss...)
		}

		// TODO: When len(ss) is not 0?
		es, _, ss, ok := cs.parseExpr(block, fname, e.Fun, markLocalVariableUsed)
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
			// Process compile-time evaluations.
			switch callee.BuiltinFunc {
			case shaderir.Len, shaderir.Cap:
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
						Type:  shaderir.NumberExpr,
						Const: gconstant.MakeInt64(int64(argts[0].Length)),
					},
				}, []shaderir.Type{{Main: shaderir.Int}}, stmts, true
			case shaderir.BoolF:
				if len(args) == 1 && args[0].Const != nil {
					if args[0].Const.Kind() != gconstant.Bool {
						cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type bool", args[0].Const.String()))
						return nil, nil, nil, false
					}
					return []shaderir.Expr{
						{
							Type:  shaderir.NumberExpr,
							Const: args[0].Const,
						},
					}, []shaderir.Type{{Main: shaderir.Bool}}, stmts, true
				}
			case shaderir.IntF:
				if len(args) == 1 && args[0].Const != nil {
					// For constants, a cast-like function doesn't work as a cast.
					// For example, `int(1.1)` is invalid.
					v := gconstant.ToInt(args[0].Const)
					if v.Kind() == gconstant.Unknown {
						cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type int", args[0].Const.String()))
						return nil, nil, nil, false
					}
					return []shaderir.Expr{
						{
							Type:  shaderir.NumberExpr,
							Const: v,
						},
					}, []shaderir.Type{{Main: shaderir.Int}}, stmts, true
				}
			case shaderir.FloatF:
				if len(args) == 1 && args[0].Const != nil {
					v := gconstant.ToFloat(args[0].Const)
					if v.Kind() == gconstant.Unknown {
						cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type float", args[0].Const.String()))
						return nil, nil, nil, false
					}
					return []shaderir.Expr{
						{
							Type:  shaderir.NumberExpr,
							Const: v,
						},
					}, []shaderir.Type{{Main: shaderir.Float}}, stmts, true
				}
			}

			// Process the expression as a regular function call.
			var finalType shaderir.Type
			switch callee.BuiltinFunc {
			case shaderir.BoolF:
				if err := checkArgsForBoolBuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				finalType = shaderir.Type{Main: shaderir.Bool}
			case shaderir.IntF:
				if err := checkArgsForIntBuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				finalType = shaderir.Type{Main: shaderir.Int}
			case shaderir.FloatF:
				if err := checkArgsForFloatBuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				finalType = shaderir.Type{Main: shaderir.Float}
			case shaderir.Vec2F:
				if err := checkArgsForVec2BuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				for i := range args {
					if args[i].Const == nil {
						continue
					}
					args[i].Const = gconstant.ToFloat(args[i].Const)
					argts[i] = shaderir.Type{Main: shaderir.Float}
				}
				finalType = shaderir.Type{Main: shaderir.Vec2}
			case shaderir.Vec3F:
				if err := checkArgsForVec3BuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				for i := range args {
					if args[i].Const == nil {
						continue
					}
					args[i].Const = gconstant.ToFloat(args[i].Const)
					argts[i] = shaderir.Type{Main: shaderir.Float}
				}
				finalType = shaderir.Type{Main: shaderir.Vec3}
			case shaderir.Vec4F:
				if err := checkArgsForVec4BuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				for i := range args {
					if args[i].Const == nil {
						continue
					}
					args[i].Const = gconstant.ToFloat(args[i].Const)
					argts[i] = shaderir.Type{Main: shaderir.Float}
				}
				finalType = shaderir.Type{Main: shaderir.Vec4}
			case shaderir.IVec2F:
				if err := checkArgsForIVec2BuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				finalType = shaderir.Type{Main: shaderir.IVec2}
			case shaderir.IVec3F:
				if err := checkArgsForIVec3BuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				finalType = shaderir.Type{Main: shaderir.IVec3}
			case shaderir.IVec4F:
				if err := checkArgsForIVec4BuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				finalType = shaderir.Type{Main: shaderir.IVec4}
			case shaderir.Mat2F:
				if err := checkArgsForMat2BuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				for i := range args {
					if args[i].Const == nil {
						continue
					}
					args[i].Const = gconstant.ToFloat(args[i].Const)
					argts[i] = shaderir.Type{Main: shaderir.Float}
				}
				finalType = shaderir.Type{Main: shaderir.Mat2}
			case shaderir.Mat3F:
				if err := checkArgsForMat3BuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				for i := range args {
					if args[i].Const == nil {
						continue
					}
					args[i].Const = gconstant.ToFloat(args[i].Const)
					argts[i] = shaderir.Type{Main: shaderir.Float}
				}
				finalType = shaderir.Type{Main: shaderir.Mat3}
			case shaderir.Mat4F:
				if err := checkArgsForMat4BuiltinFunc(args, argts); err != nil {
					cs.addError(e.Pos(), err.Error())
					return nil, nil, nil, false
				}
				for i := range args {
					if args[i].Const == nil {
						continue
					}
					args[i].Const = gconstant.ToFloat(args[i].Const)
					argts[i] = shaderir.Type{Main: shaderir.Float}
				}
				finalType = shaderir.Type{Main: shaderir.Mat4}
			case shaderir.TexelAt:
				if len(args) != 2 {
					cs.addError(e.Pos(), fmt.Sprintf("number of %s's arguments must be 2 but %d", callee.BuiltinFunc, len(args)))
					return nil, nil, nil, false
				}
				if argts[0].Main != shaderir.Texture {
					cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as texture value in argument to %s", argts[0].String(), callee.BuiltinFunc))
					return nil, nil, nil, false
				}
				if argts[1].Main != shaderir.Vec2 {
					cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as vec2 value in argument to %s", argts[1].String(), callee.BuiltinFunc))
					return nil, nil, nil, false
				}
				finalType = shaderir.Type{Main: shaderir.Vec4}
			case shaderir.DiscardF:
				if len(args) != 0 {
					cs.addError(e.Pos(), fmt.Sprintf("number of %s's arguments must be 0 but %d", callee.BuiltinFunc, len(args)))
					return nil, nil, nil, false
				}
				if fname != cs.fragmentEntry {
					cs.addError(e.Pos(), fmt.Sprintf("discard is available only in %s", cs.fragmentEntry))
					return nil, nil, nil, false
				}
				stmts = append(stmts, shaderir.Stmt{
					Type: shaderir.Discard,
				})
				return nil, nil, stmts, true

			case shaderir.Clamp, shaderir.Mix, shaderir.Smoothstep, shaderir.Faceforward, shaderir.Refract:
				// 3 arguments
				if len(args) != 3 {
					cs.addError(e.Pos(), fmt.Sprintf("number of %s's arguments must be 3 but %d", callee.BuiltinFunc, len(args)))
					return nil, nil, nil, false
				}
				switch callee.BuiltinFunc {
				case shaderir.Clamp:
					if kind, _ := resolveConstKind(args, argts); kind != gconstant.Unknown {
						switch kind {
						case gconstant.Int:
							for i, arg := range args {
								if arg.Const == nil {
									if argts[i].Main != shaderir.Int {
										cs.addError(e.Pos(), fmt.Sprintf("%s's arguments don't match: %s, %s, and %s", callee.BuiltinFunc, argts[0].String(), argts[1].String(), argts[2].String()))
										return nil, nil, nil, false
									}
									continue
								}
								v := gconstant.ToInt(arg.Const)
								if v.Kind() == gconstant.Unknown {
									cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type int", arg.Const.String()))
									return nil, nil, nil, false
								}
								args[i].Const = gconstant.ToInt(args[i].Const)
								argts[i] = shaderir.Type{Main: shaderir.Int}
							}
						case gconstant.Float:
							for i, arg := range args {
								if arg.Const == nil {
									if argts[i].Main != shaderir.Float {
										cs.addError(e.Pos(), fmt.Sprintf("%s's arguments don't match: %s, %s, and %s", callee.BuiltinFunc, argts[0].String(), argts[1].String(), argts[2].String()))
										return nil, nil, nil, false
									}
									continue
								}
								v := gconstant.ToFloat(arg.Const)
								if v.Kind() == gconstant.Unknown {
									cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type float", arg.Const.String()))
									return nil, nil, nil, false
								}
								args[i].Const = gconstant.ToFloat(args[i].Const)
								argts[i] = shaderir.Type{Main: shaderir.Float}
							}
						}
					}
					if argts[0].IsIntVector() {
						for i := 1; i < 3; i++ {
							if args[i].Const != nil {
								v := gconstant.ToInt(args[i].Const)
								if v.Kind() == gconstant.Unknown {
									cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type int", args[i].Const.String()))
									return nil, nil, nil, false
								}
								args[i].Const = v
								argts[i] = shaderir.Type{Main: shaderir.Int}
							}
						}
					}
					if argts[0].IsFloatVector() {
						for i := 1; i < 3; i++ {
							if args[i].Const != nil {
								v := gconstant.ToFloat(args[i].Const)
								if v.Kind() == gconstant.Unknown {
									cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type float", args[i].Const.String()))
									return nil, nil, nil, false
								}
								args[i].Const = v
								argts[i] = shaderir.Type{Main: shaderir.Float}
							}
						}
					}
				default:
					for i := range args {
						// If the argument is a non-typed constant value, treat this as a float value (#1874).
						if args[i].Const != nil && argts[i].Main == shaderir.None && gconstant.ToFloat(args[i].Const).Kind() != gconstant.Unknown {
							args[i].Const = gconstant.ToFloat(args[i].Const)
							argts[i] = shaderir.Type{Main: shaderir.Float}
						}
					}
				}
				for i := range args {
					switch callee.BuiltinFunc {
					case shaderir.Clamp:
						if argts[i].Main != shaderir.Float && !argts[i].IsFloatVector() && argts[i].Main != shaderir.Int && !argts[i].IsIntVector() {
							cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as float, vecN, int, or ivecN value in argument to %s", argts[i].String(), callee.BuiltinFunc))
							return nil, nil, nil, false
						}
					default:
						if argts[i].Main != shaderir.Float && !argts[i].IsFloatVector() {
							cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as float, vec2, vec3, or vec4 value in argument to %s", argts[i].String(), callee.BuiltinFunc))
							return nil, nil, nil, false
						}
					}
				}
				switch callee.BuiltinFunc {
				case shaderir.Clamp:
					if !((argts[0].Equal(&argts[1]) && argts[0].Equal(&argts[2])) ||
						(argts[0].IsFloatVector() && argts[1].Main == shaderir.Float && argts[2].Main == shaderir.Float) ||
						(argts[0].IsIntVector() && argts[1].Main == shaderir.Int && argts[2].Main == shaderir.Int)) {
						cs.addError(e.Pos(), fmt.Sprintf("the second and the third arguments for %s must equal to the first argument %s or float or int but %s and %s", callee.BuiltinFunc, argts[0].String(), argts[1].String(), argts[2].String()))
						return nil, nil, nil, false
					}
				case shaderir.Mix:
					if !argts[0].Equal(&argts[1]) {
						cs.addError(e.Pos(), fmt.Sprintf("%s and %s don't match in argument to %s", argts[0].String(), argts[1].String(), callee.BuiltinFunc))
						return nil, nil, nil, false
					}
					if !argts[0].Equal(&argts[2]) && argts[2].Main != shaderir.Float {
						cs.addError(e.Pos(), fmt.Sprintf("the third arguments for %s must equal to the first/second argument %s or float but %s", callee.BuiltinFunc, argts[0].String(), argts[2].String()))
						return nil, nil, nil, false
					}
				case shaderir.Smoothstep:
					if (!argts[0].Equal(&argts[1]) || !argts[0].Equal(&argts[2])) && (argts[0].Main != shaderir.Float || argts[1].Main != shaderir.Float) {
						cs.addError(e.Pos(), fmt.Sprintf("the first and the second arguments for %s must equal to the third argument %s or float but %s and %s", callee.BuiltinFunc, argts[2].String(), argts[0].String(), argts[1].String()))
						return nil, nil, nil, false
					}
				case shaderir.Refract:
					if !argts[0].Equal(&argts[1]) {
						cs.addError(e.Pos(), fmt.Sprintf("%s and %s don't match in argument to %s", argts[0].String(), argts[1].String(), callee.BuiltinFunc))
						return nil, nil, nil, false
					}
					if argts[2].Main != shaderir.Float {
						cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as float value in argument to %s", argts[2].String(), callee.BuiltinFunc))
						return nil, nil, nil, false
					}
				default:
					if !argts[0].Equal(&argts[1]) || !argts[0].Equal(&argts[2]) {
						cs.addError(e.Pos(), fmt.Sprintf("all the argument types for %s must be the same but %s, %s, and %s", callee.BuiltinFunc, argts[0].String(), argts[1].String(), argts[2].String()))
						return nil, nil, nil, false
					}
				}

				switch callee.BuiltinFunc {
				case shaderir.Smoothstep:
					finalType = argts[2]
				default:
					finalType = argts[0]
				}

			case shaderir.Atan2, shaderir.Pow, shaderir.Mod, shaderir.Min, shaderir.Max, shaderir.Step, shaderir.Distance, shaderir.Dot, shaderir.Cross, shaderir.Reflect:
				// 2 arguments
				if len(args) != 2 {
					cs.addError(e.Pos(), fmt.Sprintf("number of %s's arguments must be 2 but %d", callee.BuiltinFunc, len(args)))
					return nil, nil, nil, false
				}

				switch callee.BuiltinFunc {
				case shaderir.Min, shaderir.Max:
					if kind, _ := resolveConstKind(args, argts); kind != gconstant.Unknown {
						switch kind {
						case gconstant.Int:
							for i, arg := range args {
								if arg.Const == nil {
									if argts[i].Main != shaderir.Int {
										cs.addError(e.Pos(), fmt.Sprintf("%s's arguments don't match: %s and %s", callee.BuiltinFunc, argts[0].String(), argts[1].String()))
										return nil, nil, nil, false
									}
									continue
								}
								v := gconstant.ToInt(arg.Const)
								if v.Kind() == gconstant.Unknown {
									cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type int", arg.Const.String()))
									return nil, nil, nil, false
								}
								args[i].Const = gconstant.ToInt(args[i].Const)
								argts[i] = shaderir.Type{Main: shaderir.Int}
							}
						case gconstant.Float:
							for i, arg := range args {
								if arg.Const == nil {
									if argts[i].Main != shaderir.Float {
										cs.addError(e.Pos(), fmt.Sprintf("%s's arguments don't match: %s and %s", callee.BuiltinFunc, argts[0].String(), argts[1].String()))
										return nil, nil, nil, false
									}
									continue
								}
								v := gconstant.ToFloat(arg.Const)
								if v.Kind() == gconstant.Unknown {
									cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type float", arg.Const.String()))
									return nil, nil, nil, false
								}
								args[i].Const = gconstant.ToFloat(args[i].Const)
								argts[i] = shaderir.Type{Main: shaderir.Float}
							}
						}
					}

					if argts[0].IsIntVector() && args[1].Const != nil {
						v := gconstant.ToInt(args[1].Const)
						if v.Kind() == gconstant.Unknown {
							cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type int", args[1].Const.String()))
							return nil, nil, nil, false
						}
						args[1].Const = v
						argts[1] = shaderir.Type{Main: shaderir.Int}
					}
					if argts[0].IsFloatVector() && args[1].Const != nil {
						v := gconstant.ToFloat(args[1].Const)
						if v.Kind() == gconstant.Unknown {
							cs.addError(e.Pos(), fmt.Sprintf("cannot convert %s to type float", args[1].Const.String()))
							return nil, nil, nil, false
						}
						args[1].Const = v
						argts[1] = shaderir.Type{Main: shaderir.Float}
					}
				default:
					for i := range args {
						if args[i].Const != nil && argts[i].Main == shaderir.None {
							// If the argument is a non-typed constant value, treat this as a float value (#1874).
							if gconstant.ToFloat(args[i].Const).Kind() != gconstant.Unknown {
								args[i].Const = gconstant.ToFloat(args[i].Const)
								argts[i] = shaderir.Type{Main: shaderir.Float}
							}
						}
					}
				}

				for i := range args {
					switch callee.BuiltinFunc {
					case shaderir.Min, shaderir.Max:
						if argts[i].Main != shaderir.Float && !argts[i].IsFloatVector() && argts[i].Main != shaderir.Int && !argts[i].IsIntVector() {
							cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as float, vecN, int, or ivecN value in argument to %s", argts[i].String(), callee.BuiltinFunc))
							return nil, nil, nil, false
						}
					default:
						if argts[i].Main != shaderir.Float && !argts[i].IsFloatVector() {
							cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as float, vec2, vec3, or vec4 value in argument to %s", argts[i].String(), callee.BuiltinFunc))
							return nil, nil, nil, false
						}
					}
				}

				switch callee.BuiltinFunc {
				case shaderir.Mod:
					if !argts[0].Equal(&argts[1]) && argts[1].Main != shaderir.Float {
						cs.addError(e.Pos(), fmt.Sprintf("the second argument for %s must equal to the first argument %s or float but %s", callee.BuiltinFunc, argts[0].String(), argts[1].String()))
						return nil, nil, nil, false
					}
				case shaderir.Min, shaderir.Max:
					if !(argts[0].Equal(&argts[1]) || (argts[0].IsFloatVector() && argts[1].Main == shaderir.Float) || (argts[0].IsIntVector() && argts[1].Main == shaderir.Int)) {
						cs.addError(e.Pos(), fmt.Sprintf("the second argument for %s must equal to the first argument %s or float or int but %s", callee.BuiltinFunc, argts[0].String(), argts[1].String()))
						return nil, nil, nil, false
					}
				case shaderir.Step:
					if !argts[0].Equal(&argts[1]) && argts[0].Main != shaderir.Float {
						cs.addError(e.Pos(), fmt.Sprintf("the first argument for %s must equal to the second argument %s or float but %s", callee.BuiltinFunc, argts[1].String(), argts[0].String()))
						return nil, nil, nil, false
					}
				case shaderir.Cross:
					for i := range argts {
						if argts[i].Main != shaderir.Vec3 {
							cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as vec3 value in argument to %s", argts[i].String(), callee.BuiltinFunc))
							return nil, nil, nil, false
						}
					}
				default:
					if !argts[0].Equal(&argts[1]) {
						cs.addError(e.Pos(), fmt.Sprintf("%s and %s don't match in argument to %s", argts[0].String(), argts[1].String(), callee.BuiltinFunc))
						return nil, nil, nil, false
					}
				}
				switch callee.BuiltinFunc {
				case shaderir.Distance, shaderir.Dot:
					finalType = shaderir.Type{Main: shaderir.Float}
				case shaderir.Step:
					finalType = argts[1]
				default:
					finalType = argts[0]
				}

			default:
				// 1 argument
				if len(args) != 1 {
					cs.addError(e.Pos(), fmt.Sprintf("number of %s's arguments must be 1 but %d", callee.BuiltinFunc, len(args)))
					return nil, nil, nil, false
				}

				if args[0].Const != nil && argts[0].Main == shaderir.None {
					switch callee.BuiltinFunc {
					case shaderir.Abs, shaderir.Sign:
						if args[0].Const.Kind() == gconstant.Int {
							argts[0] = shaderir.Type{Main: shaderir.Int}
						}
						if args[0].Const.Kind() == gconstant.Float {
							argts[0] = shaderir.Type{Main: shaderir.Float}
						}
					default:
						// If the argument is a non-typed constant value, treat this as a float value (#1874).
						if gconstant.ToFloat(args[0].Const).Kind() != gconstant.Unknown {
							args[0].Const = gconstant.ToFloat(args[0].Const)
							argts[0] = shaderir.Type{Main: shaderir.Float}
						}
					}
				}
				switch callee.BuiltinFunc {
				case shaderir.Transpose:
					if !argts[0].IsMatrix() {
						cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as mat2, mat3, or mat4 value in argument to %s", argts[0].String(), callee.BuiltinFunc))
						return nil, nil, nil, false
					}
				case shaderir.Abs, shaderir.Sign:
					if argts[0].Main != shaderir.Float && !argts[0].IsFloatVector() && argts[0].Main != shaderir.Int && !argts[0].IsIntVector() {
						cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as float, vecN, int, or ivenN value in argument to %s", argts[0].String(), callee.BuiltinFunc))
						return nil, nil, nil, false
					}
				default:
					if argts[0].Main != shaderir.Float && !argts[0].IsFloatVector() {
						cs.addError(e.Pos(), fmt.Sprintf("cannot use %s as float, vec2, vec3, or vec4 value in argument to %s", argts[0].String(), callee.BuiltinFunc))
						return nil, nil, nil, false
					}
				}
				if callee.BuiltinFunc == shaderir.Length {
					finalType = shaderir.Type{Main: shaderir.Float}
				} else {
					finalType = argts[0]
				}
			}
			return []shaderir.Expr{
				{
					Type:  shaderir.Call,
					Exprs: append([]shaderir.Expr{callee}, args...),
				},
			}, []shaderir.Type{finalType}, stmts, true
		}

		if callee.Type != shaderir.FunctionExpr {
			cs.addError(e.Pos(), fmt.Sprintf("function callee must be a function name but %s", e.Fun))
			return nil, nil, nil, false
		}

		f := cs.funcs[callee.Index]

		if len(f.ir.InParams) < len(args) {
			cs.addError(e.Pos(), fmt.Sprintf("too many arguments in call to %s", e.Fun))
			return nil, nil, nil, false
		}
		if len(f.ir.InParams) > len(args) {
			cs.addError(e.Pos(), fmt.Sprintf("not enough arguments in call to %s", e.Fun))
			return nil, nil, nil, false
		}

		for i, p := range f.ir.InParams {
			if !canAssign(&p, &argts[i], args[i].Const) {
				cs.addError(e.Pos(), fmt.Sprintf("cannot use type %s as type %s in argument", argts[i].String(), p.String()))
				return nil, nil, nil, false
			}

			if args[i].Const != nil {
				switch p.Main {
				case shaderir.Int:
					args[i].Const = gconstant.ToInt(args[i].Const)
					argts[i] = shaderir.Type{Main: shaderir.Int}
				case shaderir.Float:
					args[i].Const = gconstant.ToFloat(args[i].Const)
					argts[i] = shaderir.Type{Main: shaderir.Float}
				}
			}
		}

		var outParams []int
		for _, p := range f.ir.OutParams {
			idx := block.totalLocalVariableCount()
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

			// The actual expression here is just a local variable that includes the result of the
			// function call.
			return []shaderir.Expr{
				{
					Type:  shaderir.Call,
					Exprs: append([]shaderir.Expr{callee}, args...),
				},
			}, []shaderir.Type{t}, stmts, true
		}

		// Even if the function doesn't return anything, calling the function should be done earlier to keep
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

		// These local-variable expressions are used for an outside function callers.
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
				cs.addError(e.Pos(), "cannot use _ as value")
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
					Type:  shaderir.NumberExpr,
					Const: c.value,
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
			}, []shaderir.Type{{Main: shaderir.Texture}}, nil, true
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
		return cs.parseExpr(block, fname, e.X, markLocalVariableUsed)

	case *ast.SelectorExpr:
		exprs, types, stmts, ok := cs.parseExpr(block, fname, e.X, true)
		if !ok {
			return nil, nil, nil, false
		}
		if len(exprs) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at a selector: %s", e.X))
			return nil, nil, nil, false
		}

		if len(types) == 0 || !isValidSwizzling(e.Sel.Name, types[0]) {
			cs.addError(e.Pos(), fmt.Sprintf("unexpected swizzling: %s", e.Sel.Name))
			return nil, nil, nil, false
		}

		var t shaderir.Type
		switch types[0].Main {
		case shaderir.Vec2, shaderir.Vec3, shaderir.Vec4:
			switch len(e.Sel.Name) {
			case 1:
				t.Main = shaderir.Float
			case 2:
				t.Main = shaderir.Vec2
			case 3:
				t.Main = shaderir.Vec3
			case 4:
				t.Main = shaderir.Vec4
			}
		case shaderir.IVec2, shaderir.IVec3, shaderir.IVec4:
			switch len(e.Sel.Name) {
			case 1:
				t.Main = shaderir.Int
			case 2:
				t.Main = shaderir.IVec2
			case 3:
				t.Main = shaderir.IVec3
			case 4:
				t.Main = shaderir.IVec4
			}
		}
		if t.Equal(&shaderir.Type{}) {
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
		exprs, ts, stmts, ok := cs.parseExpr(block, fname, e.X, markLocalVariableUsed)
		if !ok {
			return nil, nil, nil, false
		}
		if len(exprs) != 1 {
			cs.addError(e.Pos(), fmt.Sprintf("multiple-value context is not available at a unary operator: %s", e.X))
			return nil, nil, nil, false
		}
		if len(ts) == 0 {
			cs.addError(e.Pos(), fmt.Sprintf("unexpected unary operator: %s", e.X))
			return nil, nil, nil, false
		}

		if exprs[0].Const != nil {
			v := gconstant.UnaryOp(e.Op, exprs[0].Const, 0)
			// Use the original type as it is.
			// Keep the type untyped if the original expression is untyped (#2705).
			return []shaderir.Expr{
				{
					Type:  shaderir.NumberExpr,
					Const: v,
				},
			}, ts[:1], stmts, true
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
		}, ts[:1], stmts, true

	case *ast.CompositeLit:
		t, ok := cs.parseType(block, fname, e.Type)
		if !ok {
			return nil, nil, nil, false
		}
		if t.Main != shaderir.Array {
			cs.addError(e.Pos(), fmt.Sprintf("invalid composite literal type %s", t.String()))
			return nil, nil, nil, false
		}
		if t.Main == shaderir.Array {
			if t.Length == -1 {
				t.Length = len(e.Elts)
			} else if t.Length < len(e.Elts) {
				// KeyValueExpr is not supported yet. Just compare the length.
				cs.addError(e.Pos(), fmt.Sprintf("too many values in %s literal", t.String()))
				return nil, nil, nil, false
			}
		}

		idx := block.totalLocalVariableCount()
		block.vars = append(block.vars, variable{
			typ: t,
		})

		var stmts []shaderir.Stmt
		for i, e := range e.Elts {
			exprs, _, ss, ok := cs.parseExpr(block, fname, e, markLocalVariableUsed)
			if !ok {
				return nil, nil, nil, false
			}
			if len(exprs) != 1 {
				cs.addError(e.Pos(), "multiple-value context is not available at a composite literal")
				return nil, nil, nil, false
			}

			expr := exprs[0]
			if expr.Const != nil {
				switch t.Sub[0].Main {
				case shaderir.Bool:
					if expr.Const.Kind() != gconstant.Bool {
						cs.addError(e.Pos(), fmt.Sprintf("cannot %s to type bool", expr.Const.String()))
					}
				case shaderir.Int:
					if !canTruncateToInteger(expr.Const) {
						cs.addError(e.Pos(), fmt.Sprintf("constant %s truncated to integer", expr.Const.String()))
						return nil, nil, nil, false
					}
					expr.Const = gconstant.ToInt(expr.Const)
				case shaderir.Float:
					if !canTruncateToFloat(expr.Const) {
						cs.addError(e.Pos(), fmt.Sprintf("constant %s truncated to float", expr.Const.String()))
						return nil, nil, nil, false
					}
					expr.Const = gconstant.ToFloat(expr.Const)
				default:
					cs.addError(e.Pos(), fmt.Sprintf("constant %s cannot be used for the array type %s", expr.Const.String(), t.String()))
					return nil, nil, nil, false
				}
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
								Type:  shaderir.NumberExpr,
								Const: gconstant.MakeInt64(int64(i)),
							},
						},
					},
					expr,
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
		exprs, _, ss, ok := cs.parseExpr(block, fname, e.Index, true)
		if !ok {
			return nil, nil, nil, false
		}
		stmts = append(stmts, ss...)

		if len(exprs) != 1 {
			cs.addError(e.Pos(), "multiple-value context is not available at an index expression")
			return nil, nil, nil, false
		}
		idx := exprs[0]
		if idx.Const != nil {
			if !canTruncateToInteger(idx.Const) {
				cs.addError(e.Pos(), fmt.Sprintf("constant %s truncated to integer", idx.Const.String()))
				return nil, nil, nil, false
			}
		}

		exprs, ts, ss, ok := cs.parseExpr(block, fname, e.X, markLocalVariableUsed)
		if !ok {
			return nil, nil, nil, false
		}
		stmts = append(stmts, ss...)
		if len(exprs) != 1 {
			cs.addError(e.Pos(), "multiple-value context is not available at an index expression")
			return nil, nil, nil, false
		}
		if len(ts) == 0 {
			cs.addError(e.Pos(), "unexpected index expression")
			return nil, nil, nil, false
		}
		x := exprs[0]
		t := ts[0]

		// Check the length only when the index is a constant.
		if idx.Const != nil {
			var length int
			switch {
			case t.Main == shaderir.Array:
				length = t.Length
			case t.IsFloatVector() || t.IsIntVector():
				length = t.VectorElementCount()
			case t.IsMatrix():
				length = t.MatrixSize()
			default:
				cs.addError(e.Pos(), fmt.Sprintf("index operator cannot be applied to the type %s", t.String()))
				return nil, nil, nil, false
			}
			v, ok := gconstant.Int64Val(gconstant.ToInt(idx.Const))
			if !ok {
				cs.addError(e.Pos(), fmt.Sprintf("constant %s cannot be used as an index", idx.Const.String()))
				return nil, nil, nil, false
			}
			if v < 0 || int(v) >= length {
				cs.addError(e.Pos(), fmt.Sprintf("index out of range: %d", v))
				return nil, nil, nil, false
			}
		}

		var typ shaderir.Type
		switch t.Main {
		case shaderir.Vec2, shaderir.Vec3, shaderir.Vec4:
			typ = shaderir.Type{Main: shaderir.Float}
		case shaderir.IVec2, shaderir.IVec3, shaderir.IVec4:
			typ = shaderir.Type{Main: shaderir.Int}
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

func isValidSwizzling(swizzling string, t shaderir.Type) bool {
	if !shaderir.IsValidSwizzling(swizzling) {
		return false
	}

	switch t.Main {
	case shaderir.Vec2, shaderir.IVec2:
		return !strings.ContainsAny(swizzling, "zwbapq")
	case shaderir.Vec3, shaderir.IVec3:
		return !strings.ContainsAny(swizzling, "waq")
	case shaderir.Vec4, shaderir.IVec4:
		return true
	default:
		return false
	}
}

func resolveConstKind(exprs []shaderir.Expr, ts []shaderir.Type) (kind gconstant.Kind, allConsts bool) {
	if len(exprs) != len(ts) {
		panic("not reached")
	}

	allConsts = true
	for _, expr := range exprs {
		if expr.Const == nil {
			allConsts = false
		}
	}

	if !allConsts {
		for _, t := range ts {
			if t.Main == shaderir.None {
				continue
			}
			if t.Main == shaderir.Float {
				return gconstant.Float, false
			}
			if t.Main == shaderir.Int {
				return gconstant.Int, false
			}
			return gconstant.Unknown, false
		}
	}

	kind = gconstant.Unknown
	for _, t := range ts {
		switch t.Main {
		case shaderir.None:
			continue
		case shaderir.Int:
			switch kind {
			case gconstant.Unknown:
				kind = gconstant.Int
			case gconstant.Int:
			case gconstant.Float:
				return gconstant.Unknown, true
			}
		case shaderir.Float:
			switch kind {
			case gconstant.Unknown:
				kind = gconstant.Float
			case gconstant.Int:
				return gconstant.Unknown, true
			case gconstant.Float:
			}
		}
	}

	if kind != gconstant.Unknown {
		return kind, true
	}

	// Prefer floats over integers for non-typed constant values.
	// For example, max(1.0, 1) should return a float value.
	for _, expr := range exprs {
		if expr.Const.Kind() == gconstant.Float {
			return gconstant.Float, true
		}
	}

	return gconstant.Int, true
}
