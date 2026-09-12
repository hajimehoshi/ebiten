// Copyright 2026 The Ebitengine Authors
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

// Package mathutil provides arithmetic helpers.
package mathutil

import "math/bits"

// Clamp01 returns x clamped to the range 0 to 1. NaN is treated as 0.
func Clamp01(x float64) float64 {
	// A NaN value fails both of the comparisons below and results in 0.
	if !(x > 0) {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// MulDiv returns x * mul / div truncated toward zero.
// It returns (0, false) if div is zero or the result does not fit in int64.
func MulDiv(x, mul, div int64) (int64, bool) {
	if div == 0 {
		return 0, false
	}

	// Unsigned magnitudes can represent the absolute value of MinInt64.
	abs := func(v int64) uint64 {
		if v < 0 {
			return -uint64(v)
		}
		return uint64(v)
	}
	hi, lo := bits.Mul64(abs(x), abs(mul))
	d := abs(div)
	if hi >= d {
		return 0, false
	}
	q, _ := bits.Div64(hi, lo, d)
	negative := (x < 0) != (mul < 0) != (div < 0)
	if negative {
		if q > 1<<63 {
			return 0, false
		}
		return -int64(q), true
	}
	if q > 1<<63-1 {
		return 0, false
	}
	return int64(q), true
}
