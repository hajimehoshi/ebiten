// Copyright 2023 The Ebitengine Authors
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

package shaderir

import (
	"go/constant"
)

func ResolveUntypedConstsForBinaryOp(lhs, rhs constant.Value, lhst, rhst Type) (newLhs, newRhs constant.Value, ok bool) {
	if lhst.Main == None && rhst.Main == None {
		if lhs.Kind() == rhs.Kind() {
			return lhs, rhs, true
		}
		if lhs.Kind() == constant.Float && constant.ToFloat(rhs).Kind() != constant.Unknown {
			return lhs, constant.ToFloat(rhs), true
		}
		if rhs.Kind() == constant.Float && constant.ToFloat(lhs).Kind() != constant.Unknown {
			return constant.ToFloat(lhs), rhs, true
		}
		if lhs.Kind() == constant.Int && constant.ToInt(rhs).Kind() != constant.Unknown {
			return lhs, constant.ToInt(rhs), true
		}
		if rhs.Kind() == constant.Int && constant.ToInt(lhs).Kind() != constant.Unknown {
			return constant.ToInt(lhs), rhs, true
		}
		return nil, nil, false
	}

	if lhst.Main == None {
		if (rhst.Main == Float || rhst.isFloatVector() || rhst.IsMatrix()) && constant.ToFloat(lhs).Kind() != constant.Unknown {
			return constant.ToFloat(lhs), rhs, true
		}
		if (rhst.Main == Int || rhst.isIntVector()) && constant.ToInt(lhs).Kind() != constant.Unknown {
			return constant.ToInt(lhs), rhs, true
		}
		if rhst.Main == Bool && lhs.Kind() == constant.Bool {
			return lhs, rhs, true
		}
		return nil, nil, false
	}

	if rhst.Main == None {
		if (lhst.Main == Float || lhst.isFloatVector() || lhst.IsMatrix()) && constant.ToFloat(rhs).Kind() != constant.Unknown {
			return lhs, constant.ToFloat(rhs), true
		}
		if (lhst.Main == Int || lhst.isIntVector()) && constant.ToInt(rhs).Kind() != constant.Unknown {
			return lhs, constant.ToInt(rhs), true
		}
		if lhst.Main == Bool && rhs.Kind() == constant.Bool {
			return lhs, rhs, true
		}
		return nil, nil, false
	}

	// lhst and rhst might not match, but this has nothing to do with resolving untyped consts.
	return lhs, rhs, true
}

func AreValidTypesForBinaryOp(op Op, lhs, rhs *Expr, lhst, rhst Type) bool {
	if op == AndAnd || op == OrOr {
		return lhst.Main == Bool && rhst.Main == Bool
	}

	if op == VectorEqualOp || op == VectorNotEqualOp {
		return lhst.IsVector() && rhst.IsVector() && lhst.Equal(&rhst)
	}

	// Comparing matrices are forbidden (#2187).
	if op == LessThanOp || op == LessThanEqualOp || op == GreaterThanOp || op == GreaterThanEqualOp || op == EqualOp || op == NotEqualOp {
		if lhst.IsMatrix() || rhst.IsMatrix() {
			return false
		}
	}

	// If both are untyped consts, compare the constants and try to truncate them if necessary.
	if lhst.Main == None && rhst.Main == None {
		// Assume that the constant types are already adjusted.
		if lhs.Const.Kind() != rhs.Const.Kind() {
			panic("shaderir: const types for a binary op must be adjusted")
		}

		// For %, both operands must be integers if both are constants. Truncatable to an integer is not enough.
		if op == ModOp {
			return lhs.Const.Kind() == constant.Int && rhs.Const.Kind() == constant.Int
		}
		return true
	}

	// Both types must not be untyped.
	if lhst.Main == None || rhst.Main == None {
		panic("shaderir: cannot resolve untyped values")
	}

	return lhst.Equal(&rhst)
}
