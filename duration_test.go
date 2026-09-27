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

package ebiten_test

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestDurationConversion(t *testing.T) {
	for _, d := range []time.Duration{0, 1, -1, 1388, -1388, 1389, -1389, time.Millisecond, -time.Millisecond, time.Second / 60, time.Hour, math.MaxInt64, math.MinInt64} {
		want := new(big.Int).Mul(big.NewInt(int64(d)), big.NewInt(720_000))
		want.Quo(want, big.NewInt(int64(time.Second)))
		if got := ebiten.DurationFromTime(d); int64(got) != want.Int64() {
			t.Errorf("DurationFromTime(%d): got %d, want %s", d, got, want)
		}
	}
	max := ebiten.DurationFromTime(time.Duration(math.MaxInt64))
	min := ebiten.DurationFromTime(time.Duration(math.MinInt64))
	for _, d := range []ebiten.Duration{0, 1, -1, ebiten.DurationMillisecond, -ebiten.DurationMillisecond, ebiten.DurationSecond, ebiten.DurationHour, max - 1, max, max + 1, min - 1, min, min + 1, math.MaxInt64, math.MinInt64} {
		want := new(big.Int).Mul(big.NewInt(int64(d)), big.NewInt(int64(time.Second)))
		want.Quo(want, big.NewInt(720_000))
		if want.Cmp(big.NewInt(math.MaxInt64)) > 0 {
			want.SetInt64(math.MaxInt64)
		}
		if want.Cmp(big.NewInt(math.MinInt64)) < 0 {
			want.SetInt64(math.MinInt64)
		}
		if got := d.TimeDuration(); int64(got) != want.Int64() {
			t.Errorf("Duration(%d).TimeDuration(): got %d, want %s", d, got, want)
		}
	}
}

func TestDurationSeconds(t *testing.T) {
	for _, tt := range []struct {
		duration ebiten.Duration
		seconds  float64
	}{
		{
			duration: ebiten.DurationMillisecond,
			seconds:  0.001,
		},
		{
			duration: ebiten.DurationSecond,
			seconds:  1,
		},
		{
			duration: -ebiten.DurationMinute,
			seconds:  -60,
		},
		{
			duration: ebiten.DurationHour,
			seconds:  3600,
		},
	} {
		if got := tt.duration.Seconds(); got != tt.seconds {
			t.Errorf("Duration(%d).Seconds(): got %v, want %v", tt.duration, got, tt.seconds)
		}
	}
}
