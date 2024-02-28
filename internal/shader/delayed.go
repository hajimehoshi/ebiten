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
	"go/ast"
	gconstant "go/constant"
	"go/token"

	"github.com/hajimehoshi/ebiten/v2/internal/shaderir"
)

type resolveTypeStatus int

const (
	resolveUnsure resolveTypeStatus = iota
	resolveOk
	resolveFail
)

type delayedValidator interface {
	Validate(expr ast.Expr) resolveTypeStatus
	Pos() token.Pos
	Error() string
}

func (cs *compileState) tryValidateDelayed(cexpr ast.Expr) (ok bool) {
	valExprs := make([]ast.Expr, 0, len(cs.delayedTypeCheks))
	for k := range cs.delayedTypeCheks {
		valExprs = append(valExprs, k)
	}
	for _, expr := range valExprs {
		if cexpr == expr {
			continue
		}
		// Check if delayed validation can be done by adding current context
		cres := cs.delayedTypeCheks[expr].Validate(cexpr)
		switch cres {
		case resolveFail:
			cs.addError(cs.delayedTypeCheks[expr].Pos(), cs.delayedTypeCheks[expr].Error())
			return false
		case resolveOk:
			delete(cs.delayedTypeCheks, expr)
		}
	}

	return true
}

type delayedShiftValidator struct {
	value gconstant.Value
	pos   token.Pos
	last  ast.Expr
}

func isArgDefaultTypeInt(f shaderir.BuiltinFunc) bool {
	return f == shaderir.IntF || f == shaderir.IVec2F || f == shaderir.IVec3F || f == shaderir.IVec4F
}

func (d *delayedShiftValidator) Validate(cexpr ast.Expr) (rs resolveTypeStatus) {
	switch cexpr.(type) {
	case *ast.Ident:
		ident := cexpr.(*ast.Ident)
		// For BuiltinFunc, only int* are allowed
		if fname, ok := shaderir.ParseBuiltinFunc(ident.Name); ok {
			if isArgDefaultTypeInt(fname) {
				return resolveOk
			}
			return resolveFail
		}
		// Untyped constant must represent int
		if ident.Name == "_" {
			if d.value != nil && d.value.Kind() == gconstant.Int {
				return resolveOk
			}
			return resolveFail
		}
		if ident.Obj != nil {
			if t, ok := ident.Obj.Type.(*ast.Ident); ok {
				return d.Validate(t)
			}
			if decl, ok := ident.Obj.Decl.(*ast.ValueSpec); ok {
				return d.Validate(decl.Type)
			}
			if _, ok := ident.Obj.Decl.(*ast.AssignStmt); ok {
				if d.value != nil && d.value.Kind() == gconstant.Int {
					return resolveOk
				}
				return resolveFail
			}
		}
	case *ast.BinaryExpr:
		bs := cexpr.(*ast.BinaryExpr)
		left, right := bs.X, bs.Y
		if bs.Y == d.last {
			left, right = right, left
		}

		rightCheck := d.Validate(right)
		d.last = cexpr
		return rightCheck
	}
	return resolveUnsure
}

func (d delayedShiftValidator) Pos() token.Pos {
	return d.pos
}

func (d delayedShiftValidator) Error() string {
	return "left shift operand should be int"
}
