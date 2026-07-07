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
			ev.Send(textinput.TextInputState{Text: "committed", CommitKind: textinput.CommitWithoutKeyPress})
			ev.End()

			// A keystroke between sessions begins a preedit; with the channel
			// closed, it is queued.
			ev.Send(textinput.TextInputState{Text: "l"})

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

func TestComputeReplacement(t *testing.T) {
	tests := []struct {
		name      string
		baseline  string
		newText   string
		wantText  string
		wantStart int
		wantEnd   int
	}{
		{
			name:      "insert into empty",
			baseline:  "",
			newText:   "a",
			wantText:  "a",
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "append",
			baseline:  "a",
			newText:   "ab",
			wantText:  "b",
			wantStart: 1,
			wantEnd:   1,
		},
		{
			name:      "no change",
			baseline:  "abc",
			newText:   "abc",
			wantText:  "",
			wantStart: 3,
			wantEnd:   3,
		},
		{
			name:      "prepend",
			baseline:  "bc",
			newText:   "abc",
			wantText:  "a",
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "middle insert",
			baseline:  "helloworld",
			newText:   "helloXworld",
			wantText:  "X",
			wantStart: 5,
			wantEnd:   5,
		},
		{
			name:      "middle replace",
			baseline:  "hello",
			newText:   "hEllo",
			wantText:  "E",
			wantStart: 1,
			wantEnd:   2,
		},

		// Accent popup: the trailing base character is replaced.
		{
			name:      "accent replaces last ASCII",
			baseline:  "a",
			newText:   "à",
			wantText:  "à",
			wantStart: 0,
			wantEnd:   1,
		},
		{
			name:      "accent replaces last in word",
			baseline:  "cafe",
			newText:   "café",
			wantText:  "é",
			wantStart: 3,
			wantEnd:   4,
		},
		{
			name:      "accent adds combining mark",
			baseline:  "e",
			newText:   "e\u0301",
			wantText:  "\u0301",
			wantStart: 1,
			wantEnd:   1,
		},

		// Two precomposed accents sharing a leading UTF-8 byte must not be
		// split mid-rune (à and è both start with 0xC3).
		{
			name:      "precomposed to precomposed",
			baseline:  "à",
			newText:   "è",
			wantText:  "è",
			wantStart: 0,
			wantEnd:   2,
		},

		// Multibyte alignment on the suffix side.
		{
			name:      "replace before multibyte suffix",
			baseline:  "aé",
			newText:   "bé",
			wantText:  "b",
			wantStart: 0,
			wantEnd:   1,
		},
		{
			name:      "replace multibyte middle",
			baseline:  "海老天",
			newText:   "海X天",
			wantText:  "X",
			wantStart: 3,
			wantEnd:   6,
		},
		{
			name:      "cjk preedit",
			baseline:  "",
			newText:   "日本",
			wantText:  "日本",
			wantStart: 0,
			wantEnd:   0,
		},

		// Emoji (surrogate pair in UTF-16, 4 bytes in UTF-8).
		{
			name:      "replace emoji",
			baseline:  "\U0001f363",
			newText:   "\U0001f371",
			wantText:  "\U0001f371",
			wantStart: 0,
			wantEnd:   4,
		},

		// Deletions yield empty text.
		{
			name:      "delete last",
			baseline:  "ab",
			newText:   "a",
			wantText:  "",
			wantStart: 1,
			wantEnd:   2,
		},
		{
			name:      "delete first",
			baseline:  "ab",
			newText:   "b",
			wantText:  "",
			wantStart: 0,
			wantEnd:   1,
		},
		{
			name:      "delete all",
			baseline:  "abc",
			newText:   "",
			wantText:  "",
			wantStart: 0,
			wantEnd:   3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotStart, gotEnd := textinput.ComputeReplacement(tt.baseline, tt.newText)
			if gotText != tt.wantText || gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Errorf("ComputeReplacement(%q, %q) = (%q, %d, %d), want (%q, %d, %d)",
					tt.baseline, tt.newText, gotText, gotStart, gotEnd, tt.wantText, tt.wantStart, tt.wantEnd)
			}
			// The replaced range must be valid, and applying the edit must
			// reproduce newText.
			if gotStart < 0 || gotEnd < gotStart || gotEnd > len(tt.baseline) {
				t.Fatalf("invalid range [%d, %d) for baseline len %d", gotStart, gotEnd, len(tt.baseline))
			}
			if got := tt.baseline[:gotStart] + gotText + tt.baseline[gotEnd:]; got != tt.newText {
				t.Errorf("applying edit gave %q, want %q", got, tt.newText)
			}
		})
	}
}

func TestFindLineBounds(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		selStart      int
		selEnd        int
		wantLineStart int
		wantLineEnd   int
	}{
		{
			name:          "empty",
			text:          "",
			selStart:      0,
			selEnd:        0,
			wantLineStart: 0,
			wantLineEnd:   0,
		},
		{
			name:          "no line break",
			text:          "Hello, World",
			selStart:      5,
			selEnd:        5,
			wantLineStart: 0,
			wantLineEnd:   12,
		},
		{
			name:          "LF before and after",
			text:          "abc\ndef\nghi",
			selStart:      5, // cursor inside "def"
			selEnd:        5,
			wantLineStart: 4,
			wantLineEnd:   7,
		},
		{
			name:          "cursor right after LF",
			text:          "abc\ndef",
			selStart:      4,
			selEnd:        4,
			wantLineStart: 4,
			wantLineEnd:   7,
		},
		{
			name:          "cursor at LF position",
			text:          "abc\ndef",
			selStart:      3,
			selEnd:        3,
			wantLineStart: 0,
			wantLineEnd:   3,
		},
		{
			name:          "VT",
			text:          "abc\vdef",
			selStart:      5,
			selEnd:        5,
			wantLineStart: 4,
			wantLineEnd:   7,
		},
		{
			name:          "FF",
			text:          "abc\fdef",
			selStart:      5,
			selEnd:        5,
			wantLineStart: 4,
			wantLineEnd:   7,
		},
		{
			name:          "CR alone",
			text:          "abc\rdef",
			selStart:      5,
			selEnd:        5,
			wantLineStart: 4,
			wantLineEnd:   7,
		},
		{
			name:          "CRLF treated as one break",
			text:          "abc\r\ndef",
			selStart:      6, // cursor inside "def"
			selEnd:        6,
			wantLineStart: 5,
			wantLineEnd:   8,
		},
		{
			name:          "CRLF with cursor at end of break",
			text:          "abc\r\ndef",
			selStart:      5,
			selEnd:        5,
			wantLineStart: 5,
			wantLineEnd:   8,
		},
		{
			name:          "NEL (U+0085)",
			text:          "abc\u0085def",
			selStart:      7, // 3 + 2 + 2 = 7 (within "def")
			selEnd:        7,
			wantLineStart: 5, // "def" at bytes [5, 8)
			wantLineEnd:   8,
		},
		{
			name:          "LS (U+2028)",
			text:          "abc\u2028def",
			selStart:      7,
			selEnd:        7,
			wantLineStart: 6,
			wantLineEnd:   9,
		},
		{
			name:          "PS (U+2029)",
			text:          "abc\u2029def",
			selStart:      7,
			selEnd:        7,
			wantLineStart: 6,
			wantLineEnd:   9,
		},
		{
			name:          "selection crossing LF",
			text:          "abc\ndef\nghi",
			selStart:      2, // spans the first LF at byte 3
			selEnd:        6,
			wantLineStart: 0,
			wantLineEnd:   7, // expands past the LF; next LF is at 7
		},
		{
			name:          "selection crossing CRLF",
			text:          "abc\r\ndef\r\nghi",
			selStart:      2, // spans CRLF (3..5); 7 is inside "def"
			selEnd:        7,
			wantLineStart: 0,
			wantLineEnd:   8, // next break is the CR at 8
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd := textinput.FindLineBounds(tt.text, tt.selStart, tt.selEnd)
			if gotStart != tt.wantLineStart || gotEnd != tt.wantLineEnd {
				t.Errorf("FindLineBounds(%q, %d, %d) = (%d, %d), want (%d, %d)",
					tt.text, tt.selStart, tt.selEnd, gotStart, gotEnd, tt.wantLineStart, tt.wantLineEnd)
			}
		})
	}
}
