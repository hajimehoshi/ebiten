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

package ui_test

import (
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/graphicscommand"
	"github.com/hajimehoshi/ebiten/v2/internal/graphicsdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestVsyncIgnored(t *testing.T) {
	const refreshInterval = time.Second / 60

	repeat := func(frameTime time.Duration, count int) []time.Duration {
		ts := make([]time.Duration, count)
		for i := range ts {
			ts[i] = frameTime
		}
		return ts
	}

	concat := func(tss ...[]time.Duration) []time.Duration {
		var ts []time.Duration
		for _, t := range tss {
			ts = append(ts, t...)
		}
		return ts
	}

	cases := []struct {
		name       string
		frameTimes []time.Duration
		want       bool
	}{
		{
			name:       "no frame",
			frameTimes: nil,
			want:       false,
		},
		{
			name:       "waiting for the vsync",
			frameTimes: repeat(refreshInterval, 1000),
			want:       false,
		},
		{
			name:       "waiting for a slightly shorter time than the refresh interval",
			frameTimes: repeat(refreshInterval*9/10, 1000),
			want:       false,
		},
		{
			name:       "not waiting at all",
			frameTimes: repeat(time.Millisecond, 1000),
			want:       true,
		},
		{
			name:       "a burst of early frames",
			frameTimes: concat(repeat(time.Millisecond, 29), repeat(refreshInterval, 1000)),
			want:       false,
		},
		{
			name:       "early frames interrupted by a frame waiting for the vsync",
			frameTimes: concat(repeat(time.Millisecond, 29), repeat(refreshInterval, 1), repeat(time.Millisecond, 29)),
			want:       false,
		},
		{
			name:       "early frames just reaching the threshold",
			frameTimes: repeat(time.Millisecond, 30),
			want:       true,
		},
		{
			name:       "frames longer than the refresh interval",
			frameTimes: repeat(refreshInterval*2, 1000),
			want:       false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, want := ui.VsyncIgnoredForTest(c.frameTimes, refreshInterval), c.want; got != want {
				t.Errorf("ui.VsyncIgnoredForTest(%d frames): got: %t, want: %t", len(c.frameTimes), got, want)
			}
		})
	}
}

type frameDriver struct {
	graphicsdriver.Graphics
	images   int
	frames   int
	presents int
}

func (*frameDriver) Begin() error {
	return nil
}

func (*frameDriver) SetVsyncEnabled(bool) {
}

func (g *frameDriver) NewImage(int, int) (graphicsdriver.Image, error) {
	g.images++
	return nil, nil
}

func (g *frameDriver) End(mode graphicsdriver.FlushMode) error {
	if mode != graphicsdriver.FlushModeIntermediate {
		g.frames++
	}
	if mode == graphicsdriver.FlushModePresent {
		g.presents++
	}
	return nil
}

func TestSkippedPresentationFlushesCommands(t *testing.T) {
	var driver frameDriver
	const frames = 10
	for frame := range frames {
		graphicscommand.NewImage(1, 1, false, "")
		if err := ui.FlushCommandsAndWaitForTesting(&driver, false); err != nil {
			t.Fatal(err)
		}
		if driver.images != frame+1 || driver.frames != frame+1 || driver.presents != 0 {
			t.Errorf("frame %d without presentation: images = %d, frames = %d, presents = %d; want %d, %d, 0", frame, driver.images, driver.frames, driver.presents, frame+1, frame+1)
		}
	}
	graphicscommand.NewImage(1, 1, false, "")
	if err := ui.FlushCommandsAndWaitForTesting(&driver, true); err != nil {
		t.Fatal(err)
	}
	if driver.images != frames+1 || driver.frames != frames+1 || driver.presents != 1 {
		t.Errorf("after presentation resumes: images = %d, frames = %d, presents = %d; want %d, %d, 1", driver.images, driver.frames, driver.presents, frames+1, frames+1)
	}
}
