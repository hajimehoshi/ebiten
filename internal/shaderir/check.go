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
		// For %, both operands must be integers if both are constants. Truncatable to an integer is not enough.
		if op == ModOp {
			return lhs.Const.Kind() == constant.Int && rhs.Const.Kind() == constant.Int
		}
		if lhs.Const.Kind() == rhs.Const.Kind() {
			return true
		}
		if lhs.Const.Kind() == constant.Float && constant.ToFloat(rhs.Const).Kind() != constant.Unknown {
			return true
		}
		if rhs.Const.Kind() == constant.Float && constant.ToFloat(lhs.Const).Kind() != constant.Unknown {
			return true
		}
		if lhs.Const.Kind() == constant.Int && constant.ToInt(rhs.Const).Kind() != constant.Unknown {
			return true
		}
		if rhs.Const.Kind() == constant.Int && constant.ToInt(lhs.Const).Kind() != constant.Unknown {
			return true
		}
		return false
	}

	// If the types match, that's fine.
	if lhst.Equal(&rhst) {
		return true
	}

	// If lhs is untyped and rhs is not, compare the constant and the type and try to truncate the constant if necessary.
	if lhst.Main == None {
		// For %, if only one of the operands is a constant, try to truncate it.
		if op == ModOp {
			return constant.ToInt(lhs.Const).Kind() != constant.Unknown && rhst.Main == Int
		}
		if rhst.Main == Float {
			return constant.ToFloat(lhs.Const).Kind() != constant.Unknown
		}
		if rhst.Main == Int {
			return constant.ToInt(lhs.Const).Kind() != constant.Unknown
		}
		if rhst.Main == Bool {
			return lhs.Const.Kind() == constant.Bool
		}
		return false
	}

	// Ditto.
	if rhst.Main == None {
		if op == ModOp {
			return constant.ToInt(rhs.Const).Kind() != constant.Unknown && lhst.Main == Int
		}
		if lhst.Main == Float {
			return constant.ToFloat(rhs.Const).Kind() != constant.Unknown
		}
		if lhst.Main == Int {
			return constant.ToInt(rhs.Const).Kind() != constant.Unknown
		}
		if lhst.Main == Bool {
			return rhs.Const.Kind() == constant.Bool
		}
		return false
	}

	return false
}
