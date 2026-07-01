// Copyright 2025 The Ebitengine Authors
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

package textinput_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

func TestConvertUTF16CountToByteCount(t *testing.T) {
	testCases := []struct {
		text string
		c    int
		want int
	}{
		{"", 0, 0},
		{"a", 0, 0},
		{"a", 1, 1},
		{"a", 2, -1},
		{"abc", 1, 1},
		{"abc", 2, 2},
		{"àbc", 1, 2},
		{"àbc", 2, 3},
		{"海老天", 1, 3},
		{"海老天", 2, 6},
		{"海老天", 3, 9},
		{"寿司🍣食べたい", 1, 3},
		{"寿司🍣食べたい", 2, 6},
		{"寿司🍣食べたい", 4, 10},
		{"寿司🍣食べたい", 5, 13},
		{"寿司🍣食べたい", 100, -1},
		{"\xff\xff\xff\xff", 0, -1},
		{"\xff\xff\xff\xff", 1, -1},
		{"\xff\xff\xff\xff", 2, -1},
		{"\xff\xff\xff\xff", 100, -1},
	}
	for _, tc := range testCases {
		if got := textinput.ConvertUTF16CountToByteCount(tc.text, tc.c); got != tc.want {
			t.Errorf("ConvertUTF16CountToByteCount(%q, %d) = %v, want %v", tc.text, tc.c, got, tc.want)
		}
	}
}

func TestConvertByteCountToUTF16Count(t *testing.T) {
	testCases := []struct {
		text string
		c    int
		want int
	}{
		{"", 0, 0},
		{"a", 0, 0},
		{"a", 1, 1},
		{"a", 2, -1},
		{"abc", 1, 1},
		{"abc", 2, 2},
		{"àbc", 2, 1},
		{"àbc", 3, 2},
		{"海老天", 3, 1},
		{"海老天", 6, 2},
		{"海老天", 9, 3},
		{"寿司🍣食べたい", 3, 1},
		{"寿司🍣食べたい", 6, 2},
		{"寿司🍣食べたい", 10, 4},
		{"寿司🍣食べたい", 13, 5},
		{"寿司🍣食べたい", 100, -1},
		{"\xff\xff\xff\xff", 0, -1},
		{"\xff\xff\xff\xff", 3, -1},
		{"\xff\xff\xff\xff", 6, -1},
		{"\xff\xff\xff\xff", 100, -1},
	}
	for _, tc := range testCases {
		if got := textinput.ConvertByteCountToUTF16Count(tc.text, tc.c); got != tc.want {
			t.Errorf("ConvertByteCountToUTF16Count(%q, %d) = %v, want %v", tc.text, tc.c, got, tc.want)
		}
	}
}

// TestQueuedStateReplayedWithoutClear documents the desync mechanism: a preedit
// that arrives after a commit closed the channel is queued, and the next
// session's start() replays it as a live composition.
func TestQueuedStateReplayedWithoutClear(t *testing.T) {
	var ev textinput.TextInputEvents

	// Session 1: the IME commits, which delivers the committed state and then
	// closes the channel.
	ev.Start()
	ev.SendCommit("committed")
	ev.End()

	// A keystroke landing in the window between the commit and the next session
	// begins a new preedit. With the channel closed, the state is queued.
	ev.SendComposition("l")

	// Session 2 starts without clearing the queue: the queued preedit is
	// flushed into the new session and reported as a live composition.
	if got := ev.StartSessionCompositing(); !got {
		t.Errorf("StartSessionCompositing() = %v, want true (queued preedit replayed)", got)
	}
}

// TestClearQueueDropsDiscardedMarkedText verifies the macOS fix: when a session
// starts after the OS marked text is discarded, clearing the queue first
// prevents the stale preedit from being replayed as a live composition, so the
// session does not get stuck compositing.
func TestClearQueueDropsDiscardedMarkedText(t *testing.T) {
	var ev textinput.TextInputEvents

	ev.Start()
	ev.SendCommit("committed")
	ev.End()

	ev.SendComposition("l")

	// macOS start() discards the OS marked text and then clears the queue, so
	// the stale preedit is not replayed.
	ev.ClearQueue()
	if got := ev.StartSessionCompositing(); got {
		t.Errorf("StartSessionCompositing() = %v, want false (queue cleared after discard)", got)
	}
}
