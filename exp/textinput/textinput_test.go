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

// TestClearQueueDropsDiscardedMarkedText verifies the macOS fix: a preedit
// queued after a commit closed the channel would be replayed as a live
// composition by the next session's start(). Clearing the queue first (as macOS
// start() does after discarding the OS marked text) drops the stale preedit.
func TestClearQueueDropsDiscardedMarkedText(t *testing.T) {
	for _, tc := range []struct {
		name       string
		clearQueue bool
		want       bool
	}{
		{
			// The queued preedit is replayed as a live composition: the bug.
			name:       "without clear",
			clearQueue: false,
			want:       true,
		},
		{
			// Clearing after the discard drops the stale preedit.
			name:       "with clear",
			clearQueue: true,
			want:       false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var ev textinput.TextInputEvents

			// Session 1: the IME commits, then closes the channel.
			ev.Start()
			ev.SendCommit("committed")
			ev.End()

			// A keystroke between sessions begins a preedit; with the channel
			// closed, it is queued.
			ev.SendComposition("l")

			if tc.clearQueue {
				ev.ClearQueue()
			}

			// Session 2: the queued preedit is replayed only if not cleared.
			if got := ev.StartSessionCompositing(); got != tc.want {
				t.Errorf("StartSessionCompositing() = %v, want %v", got, tc.want)
			}
		})
	}
}
