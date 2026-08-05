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

//go:build (freebsd || (linux && !android) || netbsd) && !nintendosdk && !playstation5

package textinput_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

// The input method reports text whether or not a session is open, so text typed
// with nothing focused must not be carried into a session started later, while
// text typed between a commit and the next session must be.
func TestWithinQueueCarry(t *testing.T) {
	const carry = textinput.QueueCarryTicks
	for _, tc := range []struct {
		name        string
		lastEndTick int64
		tick        int64
		want        bool
	}{
		{
			// Typing before any text field is ever focused.
			name:        "no session has ended",
			lastEndTick: 0,
			tick:        1000,
			want:        false,
		},
		{
			// A commit and the restart observed within the same tick.
			name:        "restarted in the same tick",
			lastEndTick: 100,
			tick:        100,
			want:        true,
		},
		{
			name:        "restarted on the next tick",
			lastEndTick: 100,
			tick:        101,
			want:        true,
		},
		{
			name:        "restarted at the end of the window",
			lastEndTick: 100,
			tick:        100 + carry,
			want:        true,
		},
		{
			// The application stopped driving text input, so what was typed
			// since belongs to no session.
			name:        "restarted just past the window",
			lastEndTick: 100,
			tick:        100 + carry + 1,
			want:        false,
		},
		{
			name:        "focused again much later",
			lastEndTick: 100,
			tick:        10000,
			want:        false,
		},
		{
			// The gate that keeps input from accumulating: every keystroke
			// taken while the application drives no session at all reaches
			// this and must be dropped rather than queued.
			name:        "typing on and on with no session",
			lastEndTick: 100,
			tick:        1_000_000,
			want:        false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := textinput.WithinQueueCarry(tc.lastEndTick, tc.tick); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// The events of a tick are processed before the game updates, so text typed in
// the tick a session starts in was reported before the session existed. It is
// what the session is being started for, and must survive; text from any
// earlier tick must not, or it would both accumulate and reach an unrelated
// session.
func TestQueuedStatesBelong(t *testing.T) {
	const carry = textinput.QueueCarryTicks
	for _, tc := range []struct {
		name        string
		lastEndTick int64
		queuedTick  int64
		tick        int64
		want        bool
	}{
		{
			// The very first session: nothing has ended, but the text was
			// typed in the tick the session starts in.
			name:        "typed in the tick the first session starts",
			lastEndTick: 0,
			queuedTick:  500,
			tick:        500,
			want:        true,
		},
		{
			name:        "typed a tick before the first session starts",
			lastEndTick: 0,
			queuedTick:  499,
			tick:        500,
			want:        false,
		},
		{
			// The commit-to-restart gap, which spans a tick boundary.
			name:        "left by a session that just ended",
			lastEndTick: 100,
			queuedTick:  100,
			tick:        101,
			want:        true,
		},
		{
			name:        "left at the end of the carry window",
			lastEndTick: 100,
			queuedTick:  100,
			tick:        100 + carry,
			want:        true,
		},
		{
			name:        "left just past the carry window",
			lastEndTick: 100,
			queuedTick:  100,
			tick:        100 + carry + 1,
			want:        false,
		},
		{
			// Typing on and on with nothing focused: each tick's states are
			// dropped as the next tick reports its own.
			name:        "typed long ago with nothing focused",
			lastEndTick: 100,
			queuedTick:  5000,
			tick:        1_000_000,
			want:        false,
		},
		{
			// Still typing right now, but no session has wanted it since the
			// carry window closed: this tick's text is for whoever starts now.
			name:        "typed in this tick long after the last session",
			lastEndTick: 100,
			queuedTick:  1_000_000,
			tick:        1_000_000,
			want:        true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := textinput.QueuedStatesBelong(tc.lastEndTick, tc.queuedTick, tc.tick)
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
