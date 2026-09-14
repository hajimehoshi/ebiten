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

func TestClamp01(t *testing.T) {
	cases := []struct {
		in   float64
		want float64
	}{
		{
			in:   math.NaN(),
			want: 0,
		},
		{
			in:   math.Inf(-1),
			want: 0,
		},
		{
			in:   -1,
			want: 0,
		},
		{
			in:   0,
			want: 0,
		},
		{
			in:   0.5,
			want: 0.5,
		},
		{
			in:   1,
			want: 1,
		},
		{
			in:   2,
			want: 1,
		},
		{
			in:   math.Inf(1),
			want: 1,
		},
	}
	for _, c := range cases {
		if got := mathutil.Clamp01(c.in); got != c.want {
			t.Errorf("mathutil.Clamp01(%v): got: %v, want: %v", c.in, got, c.want)
		}
	}
}

func TestMulDiv(t *testing.T) {
	cases := []struct {
		x   int64
		mul int64
		div int64
	}{
		{
			x:   2,
			mul: math.MaxInt64,
			div: 1_000_000_000,
		},
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
		if got, ok := mathutil.MulDiv(c.x, c.mul, c.div); !ok || got != want.Int64() {
			t.Errorf("MulDiv(%d, %d, %d): got: (%d, %t), want: (%d, true)", c.x, c.mul, c.div, got, ok, want.Int64())
		}
	}
}

func TestMulDivBoundaries(t *testing.T) {
	values := []int64{
		math.MinInt64, math.MinInt64 + 1,
		-1 << 32, -1_000_000_000, -3, -2, -1,
		0, 1, 2, 3, 1_000_000_000, 1 << 32,
		math.MaxInt64 - 1, math.MaxInt64,
	}
	for _, x := range values {
		for _, mul := range values {
			for _, div := range values {
				testMulDiv(t, x, mul, div)
			}
		}
	}
}

func testMulDiv(t *testing.T, x, mul, div int64) {
	t.Helper()
	var want big.Int
	if div != 0 {
		want.Mul(big.NewInt(x), big.NewInt(mul))
		want.Quo(&want, big.NewInt(div))
	}
	wantOK := div != 0 && want.IsInt64()
	var wantValue int64
	if wantOK {
		wantValue = want.Int64()
	}
	if got, ok := mathutil.MulDiv(x, mul, div); ok != wantOK || got != wantValue {
		t.Errorf("MulDiv(%d, %d, %d): got: (%d, %t), want: (%d, %t)", x, mul, div, got, ok, wantValue, wantOK)
	}
}
