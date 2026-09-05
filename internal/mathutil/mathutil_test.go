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

package mathutil_test

import (
	"math"
	"math/big"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
)

func TestMulDiv(t *testing.T) {
	cases := []struct {
		x   int64
		mul int64
		div int64
	}{
		{
			x:   0,
			mul: 1e9,
			div: 48000,
		},
		{
			x:   1,
			mul: 1e9,
			div: 48000,
		},
		{
			x:   47999,
			mul: 1e9,
			div: 48000,
		},
		{
			x:   48000,
			mul: 1e9,
			div: 48000,
		},
		{
			x:   48001,
			mul: 1e9,
			div: 48000,
		},
		{
			x:   -48001,
			mul: 1e9,
			div: 48000,
		},
		// 54 hours of samples at 48000Hz, where x * mul overflows int64.
		{
			x:   54 * 3600 * 48000,
			mul: 1e9,
			div: 48000,
		},
		{
			x:   54*3600*48000 + 1,
			mul: 1e9,
			div: 48000,
		},
		{
			x:   -(54*3600*48000 + 1),
			mul: 1e9,
			div: 48000,
		},
		// The inverse conversion: a duration to a byte position of 32bit float stereo at 48000Hz.
		{
			x:   54 * 3600 * 1e9,
			mul: 8 * 48000,
			div: 1e9,
		},
		{
			x:   54*3600*1e9 + 1,
			mul: 8 * 48000,
			div: 1e9,
		},
		{
			x:   -(54*3600*1e9 + 1),
			mul: 8 * 48000,
			div: 1e9,
		},
		{
			x:   math.MaxInt64,
			mul: 1e9,
			div: 1e9,
		},
		{
			x:   math.MinInt64,
			mul: 1e9,
			div: 1e9,
		},
	}
	for _, c := range cases {
		// big.Int gives the exact x * mul / div truncated toward zero, which is what MulDiv must
		// return without the intermediate overflow.
		want := new(big.Int).Quo(new(big.Int).Mul(big.NewInt(c.x), big.NewInt(c.mul)), big.NewInt(c.div))
		if !want.IsInt64() {
			t.Fatalf("the test case MulDiv(%d, %d, %d) does not fit in int64", c.x, c.mul, c.div)
		}
		if got := mathutil.MulDiv(c.x, c.mul, c.div); got != want.Int64() {
			t.Errorf("MulDiv(%d, %d, %d): got: %d, want: %d", c.x, c.mul, c.div, got, want.Int64())
		}
	}
}
