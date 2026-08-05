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
			ev.Send(textinput.TextInputState{Text: "committed", CommitKind: textinput.CommitRegular})
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
	// caret is the byte offset of the caret in newText; a negative caret
	// exercises the plain caret-agnostic path, a non-negative one the
	// caret-anchored path.
	tests := []struct {
		name      string
		baseline  string
		newText   string
		caret     int
		wantText  string
		wantStart int
		wantEnd   int
	}{
		{
			name:      "insert into empty",
			baseline:  "",
			newText:   "a",
			caret:     -1,
			wantText:  "a",
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "append",
			baseline:  "a",
			newText:   "ab",
			caret:     -1,
			wantText:  "b",
			wantStart: 1,
			wantEnd:   1,
		},
		{
			name:      "no change",
			baseline:  "abc",
			newText:   "abc",
			caret:     -1,
			wantText:  "",
			wantStart: 3,
			wantEnd:   3,
		},
		{
			name:      "prepend",
			baseline:  "bc",
			newText:   "abc",
			caret:     -1,
			wantText:  "a",
			wantStart: 0,
			wantEnd:   0,
		},
		{
			name:      "middle insert",
			baseline:  "helloworld",
			newText:   "helloXworld",
			caret:     -1,
			wantText:  "X",
			wantStart: 5,
			wantEnd:   5,
		},
		{
			name:      "middle replace",
			baseline:  "hello",
			newText:   "hEllo",
			caret:     -1,
			wantText:  "E",
			wantStart: 1,
			wantEnd:   2,
		},

		// Accent popup: the trailing base character is replaced.
		{
			name:      "accent replaces last ASCII",
			baseline:  "a",
			newText:   "à",
			caret:     -1,
			wantText:  "à",
			wantStart: 0,
			wantEnd:   1,
		},
		{
			name:      "accent replaces last in word",
			baseline:  "cafe",
			newText:   "café",
			caret:     -1,
			wantText:  "é",
			wantStart: 3,
			wantEnd:   4,
		},
		{
			name:      "accent adds combining mark",
			baseline:  "e",
			newText:   "e\u0301",
			caret:     -1,
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
			caret:     -1,
			wantText:  "è",
			wantStart: 0,
			wantEnd:   2,
		},

		// Multibyte alignment on the suffix side.
		{
			name:      "replace before multibyte suffix",
			baseline:  "aé",
			newText:   "bé",
			caret:     -1,
			wantText:  "b",
			wantStart: 0,
			wantEnd:   1,
		},
		{
			name:      "replace multibyte middle",
			baseline:  "海老天",
			newText:   "海X天",
			caret:     -1,
			wantText:  "X",
			wantStart: 3,
			wantEnd:   6,
		},
		{
			name:      "cjk preedit",
			baseline:  "",
			newText:   "日本",
			caret:     -1,
			wantText:  "日本",
			wantStart: 0,
			wantEnd:   0,
		},

		// Emoji (surrogate pair in UTF-16, 4 bytes in UTF-8).
		{
			name:      "replace emoji",
			baseline:  "\U0001f363",
			newText:   "\U0001f371",
			caret:     -1,
			wantText:  "\U0001f371",
			wantStart: 0,
			wantEnd:   4,
		},

		// Deletions yield empty text.
		{
			name:      "delete last",
			baseline:  "ab",
			newText:   "a",
			caret:     -1,
			wantText:  "",
			wantStart: 1,
			wantEnd:   2,
		},
		{
			name:      "delete first",
			baseline:  "ab",
			newText:   "b",
			caret:     -1,
			wantText:  "",
			wantStart: 0,
			wantEnd:   1,
		},
		{
			name:      "delete all",
			baseline:  "abc",
			newText:   "",
			caret:     -1,
			wantText:  "",
			wantStart: 0,
			wantEnd:   3,
		},

		// Caret-anchored cases.

		// Inserting "na" at "ba|na" to build "banana". The caret after the
		// committed text anchors the edit in the middle; the prefix/suffix span
		// alone would append at the end.
		{
			name:      "anchored insert into repeated text at caret",
			baseline:  "bana",
			newText:   "banana",
			caret:     len("bana"),
			wantText:  "na",
			wantStart: len("ba"),
			wantEnd:   len("ba"),
		},
		// The same repeated text, but the caret is at the very end: this really
		// is an append.
		{
			name:      "anchored append to repeated text at end",
			baseline:  "bana",
			newText:   "banana",
			caret:     len("banana"),
			wantText:  "na",
			wantStart: len("bana"),
			wantEnd:   len("bana"),
		},
		// An unambiguous middle insert: the anchored path agrees with the plain
		// "middle insert" case above.
		{
			name:      "anchored middle insert",
			baseline:  "helloworld",
			newText:   "helloXworld",
			caret:     6,
			wantText:  "X",
			wantStart: 5,
			wantEnd:   5,
		},
		// Accent popup with the caret after the replaced character.
		{
			name:      "anchored accent replaces last ASCII",
			baseline:  "a",
			newText:   "à",
			caret:     len("à"),
			wantText:  "à",
			wantStart: 0,
			wantEnd:   1,
		},
		// A composition preedit ("abXY") whose leading text repeats the
		// surrounding text, anchored at the preedit end. The prefix/suffix span
		// alone would grab the wrong run ("XYab").
		{
			name:      "anchored preedit repeats surrounding text",
			baseline:  "abab",
			newText:   "ababXYab",
			caret:     len("ababXY"),
			wantText:  "abXY",
			wantStart: 2,
			wantEnd:   2,
		},
		// A caret that does not sit at the end of the edited region (the text
		// after it is not a suffix of baseline) falls back to the prefix/suffix
		// span.
		{
			name:      "anchored caret not at edit end falls back",
			baseline:  "abc",
			newText:   "aXYc",
			caret:     2,
			wantText:  "XY",
			wantStart: 1,
			wantEnd:   2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotStart, gotEnd := textinput.ComputeReplacement(tt.baseline, tt.newText, tt.caret)
			if gotText != tt.wantText || gotStart != tt.wantStart || gotEnd != tt.wantEnd {
				t.Errorf("ComputeReplacement(%q, %q, %d) = (%q, %d, %d), want (%q, %d, %d)",
					tt.baseline, tt.newText, tt.caret, gotText, gotStart, gotEnd, tt.wantText, tt.wantStart, tt.wantEnd)
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

// commitState returns a state representing a committed text.
func commitState(text string) textinput.TextInputState {
	return textinput.TextInputState{Text: text, CommitKind: textinput.CommitRegular}
}

// TestQueuedCommitsReachSuccessiveSessions verifies that several commits
// arriving before the next tick are all delivered, in order. A session ends at
// its first commit, so flushing every queued commit into one session's channel
// would close that channel over the ones it never read.
func TestQueuedCommitsReachSuccessiveSessions(t *testing.T) {
	var ev textinput.TextInputEvents

	// Three keystrokes land between ticks, with no session open to take them.
	for _, text := range []string{"a", "b", "c"} {
		if ev.Send(commitState(text)) {
			t.Fatalf("Send(%q) reported delivery with no session open", text)
		}
	}

	// The application starts a session per commit it observes, as Composer
	// does within a single Update.
	for _, want := range []string{"a", "b", "c"} {
		got, ok := ev.StartSessionCommit()
		if !ok {
			t.Fatalf("no commit delivered, want %q", want)
		}
		if got != want {
			t.Errorf("commit = %q, want %q", got, want)
		}
	}

	if got, ok := ev.StartSessionCommit(); ok {
		t.Errorf("a fourth session got commit %q, want none", got)
	}
}

// TestQueuedCompositionsPrecedeTheirCommit verifies that the states before a
// commit reach the same session as the commit, and that what follows it does
// not.
func TestQueuedCompositionsPrecedeTheirCommit(t *testing.T) {
	var ev textinput.TextInputEvents

	ev.Send(textinput.TextInputState{Text: "a"})
	ev.Send(commitState("A"))
	ev.Send(textinput.TextInputState{Text: "b"})
	ev.Send(commitState("B"))

	for _, want := range []string{"A", "B"} {
		got, ok := ev.StartSessionCommit()
		if !ok {
			t.Fatalf("no commit delivered, want %q", want)
		}
		if got != want {
			t.Errorf("commit = %q, want %q", got, want)
		}
	}
}

// TestSendReportsDelivery verifies that Send distinguishes a state an open
// session took from one merely queued. The X11 backend ends a session only on
// the former, so that a commit no session read cannot push the deadline that
// decides whether queued states still belong to a session about to start.
func TestSendReportsDelivery(t *testing.T) {
	var ev textinput.TextInputEvents

	// No session: the state is queued, not delivered.
	if ev.Send(commitState("a")) {
		t.Error("Send with no session open reported delivery")
	}

	// A session takes the queued commit, and ends at it, so a further commit
	// belongs to the next session.
	ev.Start()
	if ev.Send(commitState("b")) {
		t.Error("Send after the session took its commit reported delivery")
	}
	ev.End()

	// A session with nothing queued takes a commit directly.
	var fresh textinput.TextInputEvents
	fresh.Start()
	if !fresh.Send(commitState("c")) {
		t.Error("Send to an open session did not report delivery")
	}
	fresh.End()
}

// The platform reports text whether or not a session is open, so text typed
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

// newEventsAtTick returns events reading the tick the returned pointer holds,
// so a test can advance ticks as a running game would.
func newEventsAtTick(tick *int64) *textinput.TextInputEvents {
	var ev textinput.TextInputEvents
	ev.SetTick(func() int64 {
		return *tick
	})
	return &ev
}

// TestQueueDoesNotGrowWithoutSession verifies that states reported with no
// session open do not accumulate. The platforms keep reporting after a session
// ends — Windows never restores its window-procedure subclass, and macOS leaves
// its text input client the first responder — so without this every keystroke
// taken outside a text field would be held forever.
func TestQueueDoesNotGrowWithoutSession(t *testing.T) {
	var tick int64 = 1
	ev := newEventsAtTick(&tick)

	// A session runs and ends, as focusing a text field and committing would.
	ev.Start()
	ev.Send(commitState("a"))
	ev.End()

	// The platform keeps reporting for the rest of the application's life.
	// Only the states of the carry window, and then of the current tick, are
	// held: what is queued never grows with the number of ticks.
	const perTick = 5
	const wantAtMost = perTick * (textinput.QueueCarryTicks + 1)
	for tick = 2; tick <= 1000; tick++ {
		for range perTick {
			ev.Send(commitState("x"))
		}
		got := ev.QueuedStateCount()
		if got > wantAtMost {
			t.Fatalf("at tick %d: %d states queued, want at most %d", tick, got, wantAtMost)
		}
		// Past the carry window nothing but this tick's states survives.
		if tick > 1+textinput.QueueCarryTicks && got != perTick {
			t.Fatalf("at tick %d: %d states queued, want %d", tick, got, perTick)
		}
	}
}

// TestStaleQueuedStatesDoNotReachNextSession verifies that text typed with
// nothing focused does not land in a text field focused later.
func TestStaleQueuedStatesDoNotReachNextSession(t *testing.T) {
	var tick int64 = 100
	ev := newEventsAtTick(&tick)

	// Typing with no text field focused.
	ev.Send(commitState("stale"))

	// The application focuses a text field much later.
	tick = 500
	if got, ok := ev.StartSessionCommit(); ok {
		t.Errorf("the session got commit %q, want none", got)
	}
}

// TestQueuedStatesOfStartTickReachSession verifies that text typed in the tick
// a session starts in still reaches it. The events of a tick are processed
// before the game updates, so a keystroke opening a text field reports its text
// before the session exists.
func TestQueuedStatesOfStartTickReachSession(t *testing.T) {
	var tick int64 = 100
	ev := newEventsAtTick(&tick)

	ev.Send(commitState("a"))

	got, ok := ev.StartSessionCommit()
	if !ok {
		t.Fatal("no commit delivered, want \"a\"")
	}
	if got != "a" {
		t.Errorf("commit = %q, want %q", got, "a")
	}
}

// TestQueuedStatesWithinCarryReachNextSession verifies that text typed between
// a commit and the next session start is not lost. A commit ends the session,
// so typing faster than one character per tick reports the rest with no session
// open.
func TestQueuedStatesWithinCarryReachNextSession(t *testing.T) {
	for _, tc := range []struct {
		name      string
		startTick int64
		want      bool
	}{
		{
			name:      "restarted within the carry window",
			startTick: 100 + textinput.QueueCarryTicks,
			want:      true,
		},
		{
			name:      "restarted past the carry window",
			startTick: 100 + textinput.QueueCarryTicks + 1,
			want:      false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var tick int64 = 100
			ev := newEventsAtTick(&tick)

			// A session takes a commit and ends at it.
			ev.Start()
			ev.Send(commitState("a"))
			ev.End()

			// The next character arrives before the application reopens.
			tick = 101
			ev.Send(commitState("b"))

			tick = tc.startTick
			got, ok := ev.StartSessionCommit()
			if ok != tc.want {
				t.Fatalf("commit delivered = %v (%q), want %v", ok, got, tc.want)
			}
			if ok && got != "b" {
				t.Errorf("commit = %q, want %q", got, "b")
			}
		})
	}
}

// TestEndWithoutSessionDoesNotExtendCarry verifies that ending when no session
// is open leaves the carry deadline alone. Windows and macOS end
// unconditionally after every commit, so a commit that no session took would
// otherwise keep the window open for as long as the application takes keyboard
// input.
func TestEndWithoutSessionDoesNotExtendCarry(t *testing.T) {
	var tick int64 = 100
	ev := newEventsAtTick(&tick)

	// Typing with no text field focused, as a platform reports it: a commit
	// no session takes, followed by an unconditional end.
	ev.Send(commitState("stale"))
	ev.End()

	// Within what would be the carry window had the end counted.
	tick = 100 + textinput.QueueCarryTicks
	if got, ok := ev.StartSessionCommit(); ok {
		t.Errorf("the session got commit %q, want none", got)
	}
}
