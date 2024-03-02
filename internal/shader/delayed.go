// Copyright 2024 The Ebiten Authors
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
	gconstant "go/constant"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type delayedTypeValidator interface {
	Validate(t shaderir.Type) (shaderir.Type, bool)
	IsValidated() (shaderir.Type, bool)
	Error() string
}

func isArgDefaultTypeInt(f shaderir.BuiltinFunc) bool {
	return f == shaderir.IntF || f == shaderir.IVec2F || f == shaderir.IVec3F || f == shaderir.IVec4F
}

func isIntType(t shaderir.Type) bool {
	return t.Main == shaderir.Int || t.IsIntVector()
}

func (cs *compileState) ValidateDefaultTypesForExpr(block *block, expr shaderir.Expr, t shaderir.Type) shaderir.Type {
	if check, ok := cs.delayedTypeCheks[expr.Ast]; ok {
		if resT, ok := check.IsValidated(); ok {
			return resT
		}
		resT, ok := check.Validate(t)
		if !ok {
			return shaderir.Type{Main: shaderir.None}
		}
		return resT
	}

	switch expr.Type {
	case shaderir.LocalVariable:
		return block.vars[expr.Index].typ

	case shaderir.Binary:
		left := expr.Exprs[0]
		right := expr.Exprs[1]

		leftType := cs.ValidateDefaultTypesForExpr(block, left, t)
		rightType := cs.ValidateDefaultTypesForExpr(block, right, t)

		// Usure about top-level type, try to validate by neighbour type
		// The same work is done twice. Can it be optimized?
		if t.Main == shaderir.None {
			cs.ValidateDefaultTypesForExpr(block, left, rightType)
			cs.ValidateDefaultTypesForExpr(block, right, leftType)
		}
	case shaderir.Call:
		fun := expr.Exprs[0]
		if fun.Type == shaderir.BuiltinFuncExpr {
			if isArgDefaultTypeInt(fun.BuiltinFunc) {
				for _, e := range expr.Exprs[1:] {
					cs.ValidateDefaultTypesForExpr(block, e, shaderir.Type{Main: shaderir.Int})
				}
				return shaderir.Type{Main: shaderir.Int}
			}

			for _, e := range expr.Exprs[1:] {
				cs.ValidateDefaultTypesForExpr(block, e, shaderir.Type{Main: shaderir.Float})
			}
			return shaderir.Type{Main: shaderir.Float}
		}

		if fun.Type == shaderir.FunctionExpr {
			args := cs.funcs[fun.Index].ir.InParams

			for i, e := range expr.Exprs[1:] {
				cs.ValidateDefaultTypesForExpr(block, e, args[i])
			}

			retT := cs.funcs[fun.Index].ir.Return

			return retT
		}
	}

	return shaderir.Type{Main: shaderir.None}
}

func (cs *compileState) ValidateDefaultTypes(block *block, stmt shaderir.Stmt) {
	switch stmt.Type {
	case shaderir.Assign:
		left := stmt.Exprs[0]
		right := stmt.Exprs[1]
		if left.Type == shaderir.LocalVariable {
			varType := block.vars[left.Index].typ
			// Type is not explicitly specified
			if stmt.IsTypeGuessed {
				varType = shaderir.Type{Main: shaderir.None}
			}
			cs.ValidateDefaultTypesForExpr(block, right, varType)
		}
	case shaderir.ExprStmt:
		for _, e := range stmt.Exprs {
			cs.ValidateDefaultTypesForExpr(block, e, shaderir.Type{Main: shaderir.None})
		}
	}
}

type delayedShiftValidator struct {
	shiftType      shaderir.Op
	value          gconstant.Value
	validated      bool
	closestUnknown bool
	failed         bool
}

func (d *delayedShiftValidator) IsValidated() (shaderir.Type, bool) {
	if d.failed {
		return shaderir.Type{}, false
	}
	if d.validated {
		return shaderir.Type{Main: shaderir.Int}, true
	}
	// If only matched with None
	if d.closestUnknown {
		// Was it originally represented by an int constant?
		if d.value.Kind() == gconstant.Int {
			return shaderir.Type{Main: shaderir.Int}, true
		}
	}
	return shaderir.Type{}, false
}

func (d *delayedShiftValidator) Validate(t shaderir.Type) (shaderir.Type, bool) {
	if d.validated {
		return shaderir.Type{Main: shaderir.Int}, true
	}
	if isIntType(t) {
		d.validated = true
		return shaderir.Type{Main: shaderir.Int}, true
	}
	if t.Main == shaderir.None {
		d.closestUnknown = true
		return t, true
	}
	return shaderir.Type{Main: shaderir.None}, false
}

func (d *delayedShiftValidator) Error() string {
	st := "left shift"
	if d.shiftType == shaderir.RightShift {
		st = "right shift"
	}
	return fmt.Sprintf("left operand for %s should be int", st)
}
