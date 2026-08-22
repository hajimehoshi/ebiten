// Copyright 2026 The Ebiten Authors
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

package clock_test

import (
	"math"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/clock"
)

// frameTimes returns timestamps of frames at the given frame rate.
//
// jitter is the amplitude of the deviation from the exact frame interval. The deviation cancels
// itself out over several frames, like a real frame timing where a late frame is followed by an
// early one.
func frameTimes(fps float64, jitter time.Duration, frames int) []int64 {
	interval := float64(time.Second) / fps
	ts := make([]int64, frames)
	var now int64
	for i := range ts {
		now += int64(interval + float64(jitter)*math.Sin(2*math.Pi*float64(i)/7))
		ts[i] = now
	}
	return ts
}

// countTicks runs the given frames and returns the tick count of each frame.
func countTicks(c *clock.Clock, ts []int64) []int {
	counts := make([]int, len(ts))
	for i, t := range ts {
		counts[i] = c.UpdateFrame(t)
	}
	return counts
}

// tps returns the actual TPS of the given tick counts over the given frames.
func tps(counts []int, ts []int64) float64 {
	var total int
	for _, c := range counts {
		total += c
	}
	return float64(total) * float64(time.Second) / float64(ts[len(ts)-1])
}

// distinctCounts returns the set of the tick counts, as a map from a count to the number of frames.
func distinctCounts(counts []int) map[int]int {
	m := map[int]int{}
	for _, c := range counts {
		m[c]++
	}
	return m
}

func TestUpdateFrameSyncWithFPS(t *testing.T) {
	c := clock.NewClock(0)
	c.SetTPS(clock.SyncWithFPS)
	for i, got := range countTicks(c, frameTimes(60, time.Millisecond, 100)) {
		if got != 1 {
			t.Errorf("frame %d: got: %d, want: 1", i, got)
		}
	}
}

func TestUpdateFrameZeroTPS(t *testing.T) {
	c := clock.NewClock(0)
	c.SetTPS(0)
	for i, got := range countTicks(c, frameTimes(60, time.Millisecond, 100)) {
		if got != 0 {
			t.Errorf("frame %d: got: %d, want: 0", i, got)
		}
	}
}

// TestTickRate tests that ticks run at the specified TPS at various frame rates.
//
// When the frame rate is so low that one frame cannot run enough ticks, the game time gets behind
// the real time and the actual TPS is limited to the number of ticks one frame can catch up with.
func TestTickRate(t *testing.T) {
	const catchUpTicks = 5

	tests := []struct {
		fps float64
		tps int
	}{
		{fps: 60, tps: 60},
		{fps: 59.94, tps: 60},
		{fps: 50, tps: 60},
		{fps: 30, tps: 60},
		{fps: 120, tps: 60},
		{fps: 144, tps: 60},
		{fps: 1000, tps: 60},
		{fps: 60, tps: 30},
		{fps: 60, tps: 120},
		{fps: 60, tps: 300},
		{fps: 60, tps: 1},
		{fps: 15, tps: 60},
		{fps: 11, tps: 60},
		{fps: 5, tps: 60},
		// A window occluded by another window can get frames at about 1 Hz.
		{fps: 1, tps: 60},
	}

	for _, tc := range tests {
		c := clock.NewClock(0)
		c.SetTPS(tc.tps)
		ts := frameTimes(tc.fps, time.Millisecond, int(tc.fps*10))
		got := tps(countTicks(c, ts), ts)

		// One frame can run more ticks than catchUpTicks when the ticks are shorter than
		// minCatchUpTime (5/60 sec).
		maxTicksPerFrame := math.Max(catchUpTicks, 5.0/60.0*float64(tc.tps))
		want := math.Min(float64(tc.tps), maxTicksPerFrame*tc.fps)
		if math.Abs(got-want)/want > 0.05 {
			t.Errorf("fps: %v, tps: %d: got: %0.2f, want: %0.2f", tc.fps, tc.tps, got, want)
		}
	}
}

// TestTickCountIsStable tests that the tick count doesn't fluctuate with a jitter in the frame
// timing. An unstable count like 0, 2, 0, 2, ... makes the game stutter even though the average
// TPS is correct.
func TestTickCountIsStable(t *testing.T) {
	tests := []struct {
		fps    float64
		tps    int
		jitter time.Duration
		want   []int
	}{
		{
			fps:    60,
			tps:    60,
			jitter: 2 * time.Millisecond,
			want:   []int{1},
		},
		{
			fps:    30,
			tps:    60,
			jitter: 2 * time.Millisecond,
			want:   []int{2},
		},
		{
			fps:    20,
			tps:    60,
			jitter: 2 * time.Millisecond,
			want:   []int{3},
		},
		{
			fps:    15,
			tps:    60,
			jitter: 2 * time.Millisecond,
			want:   []int{4},
		},
		{
			fps:    60,
			tps:    30,
			jitter: 2 * time.Millisecond,
			want:   []int{0, 1},
		},
		{
			fps:    120,
			tps:    60,
			jitter: 2 * time.Millisecond,
			want:   []int{0, 1},
		},
		// A jitter bigger than a half of a tick inevitably changes the count, as a tick is not
		// divisible. Use a smaller jitter for a frame rate close to the TPS and for a big TPS.
		{
			fps:    59.94,
			tps:    60,
			jitter: 200 * time.Microsecond,
			want:   []int{1},
		},
		{
			fps:    60,
			tps:    300,
			jitter: 200 * time.Microsecond,
			want:   []int{5},
		},
	}

	for _, tc := range tests {
		c := clock.NewClock(0)
		c.SetTPS(tc.tps)
		frames := int(tc.fps * 10)
		counts := distinctCounts(countTicks(c, frameTimes(tc.fps, tc.jitter, frames)))

		// A count other than the expected ones is acceptable only if it is off by one and rare.
		// The game time cannot follow the real time exactly when the frame interval is not a
		// multiple of a tick.
		var rare int
		for got, n := range counts {
			if slices.Contains(tc.want, got) {
				continue
			}
			if !slices.Contains(tc.want, got-1) && !slices.Contains(tc.want, got+1) {
				t.Errorf("fps: %v, tps: %d: unexpected tick count %d (counts: %v, want: %v)", tc.fps, tc.tps, got, counts, tc.want)
				continue
			}
			rare += n
		}
		if got := float64(rare) / float64(frames); got > 0.05 {
			t.Errorf("fps: %v, tps: %d: the tick count is unstable in %0.1f%% of the frames (counts: %v, want: %v)", tc.fps, tc.tps, got*100, counts, tc.want)
		}
	}
}

// TestSlowFrameRateDegradesGradually tests that a lower frame rate never results in a higher TPS.
func TestSlowFrameRateDegradesGradually(t *testing.T) {
	var lastTPS float64
	for fps := 3.0; fps <= 20.0; fps += 0.5 {
		c := clock.NewClock(0)
		c.SetTPS(60)
		ts := frameTimes(fps, time.Millisecond, int(fps*20))
		got := tps(countTicks(c, ts), ts)
		// Allow a small error, as the measured TPS is not exact.
		if got < lastTPS*0.99 {
			t.Errorf("fps: %v: TPS decreased as the frame rate increased: got: %0.2f, previous: %0.2f", fps, got, lastTPS)
		}
		lastTPS = got
	}
}

// TestLongGapDoesNotBurst tests that a long gap between frames, like a suspended application,
// doesn't run a burst of ticks.
func TestLongGapDoesNotBurst(t *testing.T) {
	c := clock.NewClock(0)
	c.SetTPS(60)

	var now int64
	for range 60 {
		now += int64(time.Second) / 60
		c.UpdateFrame(now)
	}

	now += int64(10 * time.Second)
	if got, want := c.UpdateFrame(now), 5; got > want {
		t.Errorf("got: %d, want: <= %d", got, want)
	}

	// The dropped time must not be kept for the following frames.
	for i := range 60 {
		now += int64(time.Second) / 60
		if got, want := c.UpdateFrame(now), 1; got != want {
			t.Errorf("frame %d after a long gap: got: %d, want: %d", i, got, want)
		}
	}
}

// TestNoDriftFromRealTime tests that the game time doesn't drift from the real time even when the
// frame interval is not a multiple of a tick.
func TestNoDriftFromRealTime(t *testing.T) {
	for _, fps := range []float64{59.94, 61, 75, 100} {
		c := clock.NewClock(0)
		c.SetTPS(60)
		ts := frameTimes(fps, time.Millisecond, int(fps*600))
		var total int
		for _, n := range countTicks(c, ts) {
			total += n
		}
		want := 60 * float64(ts[len(ts)-1]) / float64(time.Second)
		if math.Abs(float64(total)-want) > 2 {
			t.Errorf("fps: %v: got: %d ticks, want: %0.2f ticks", fps, total, want)
		}
	}
}

func TestSetTPSInTheMiddle(t *testing.T) {
	c := clock.NewClock(0)
	c.SetTPS(60)

	var now int64
	for range 60 {
		now += int64(time.Second) / 60
		c.UpdateFrame(now)
	}

	for _, tps := range []int{300, 1, 30} {
		c.SetTPS(tps)
		var total int
		start := now
		for range 60 {
			now += int64(time.Second) / 60
			total += c.UpdateFrame(now)
		}
		got := float64(total) * float64(time.Second) / float64(now-start)
		if math.Abs(got-float64(tps))/float64(tps) > 0.05 {
			t.Errorf("tps: %d: got: %0.2f", tps, got)
		}
	}
}

// TestSinkTick tests that a tick run outside of UpdateFrame is subtracted from the following frames.
func TestSinkTick(t *testing.T) {
	const tick = int64(time.Second) / 60

	c := clock.NewClock(0)
	c.SetTPS(60)

	// A tick forced in the middle of a tick interval makes the following frame run no tick.
	c.UpdateFrame(tick)
	c.SinkTick(tick)
	if got, want := c.UpdateFrame(tick+tick/4), 0; got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}

	// Sinking ticks never banks future ticks, so that the game doesn't stall after a lot of
	// forced ticks.
	for range 10 {
		c.SinkTick(tick + tick/4)
	}
	if got, want := c.UpdateFrame(tick+tick/4+tick), 1; got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}
}

// TestSinkTickWithoutUpdateFrame tests that a sunk tick consumes the real time that has passed even
// when UpdateFrame is not called in between.
func TestSinkTickWithoutUpdateFrame(t *testing.T) {
	const tick = int64(time.Second) / 60

	c := clock.NewClock(0)
	c.SetTPS(60)
	c.UpdateFrame(0)

	// Force a tick a tick-time later without calling UpdateFrame. The forced tick covers the
	// elapsed time, so the following frame must run no tick.
	c.SinkTick(tick)
	if got, want := c.UpdateFrame(tick), 0; got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}

	// Forcing ticks at the target TPS keeps the total number of ticks at the target TPS.
	var now int64
	for range 60 {
		now += tick
		c.SinkTick(now)
	}
	if got, want := c.UpdateFrame(now), 0; got != want {
		t.Errorf("got: %d, want: %d", got, want)
	}
}

func TestActualFPSAndTPS(t *testing.T) {
	c := clock.NewClock(0)
	c.SetTPS(30)
	countTicks(c, frameTimes(60, time.Millisecond, 180))

	if got := c.ActualFPS(); math.Abs(got-60) > 2 {
		t.Errorf("ActualFPS: got: %0.2f, want: 60", got)
	}
	if got := c.ActualTPS(); math.Abs(got-30) > 2 {
		t.Errorf("ActualTPS: got: %0.2f, want: 30", got)
	}
}
