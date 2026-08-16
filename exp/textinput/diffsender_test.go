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

package textinput_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

// The buffer in these tests is a text field holding "ab|cd", into which an IME
// inserts the preedit "あい". In UTF-16 the buffer "abあいcd" indexes as
// a=0, b=1, あ=2, い=3, c=4, d=5.

func TestDiffSenderCompositionSelectionMoves(t *testing.T) {
	d := textinput.NewDiffSender("ab", "cd")

	// The IME selects the first character of the preedit.
	d.TrySend("abあいcd", 2, 3, false, textinput.CommitNone)
	states := d.Drain()
	if got, want := len(states), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}
	if got, want := states[0].Text, "あい"; got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
	if got, want := states[0].CompositionSelectionStartInBytes, 0; got != want {
		t.Errorf("CompositionSelectionStartInBytes = %d, want %d", got, want)
	}
	if got, want := states[0].CompositionSelectionEndInBytes, len("あ"); got != want {
		t.Errorf("CompositionSelectionEndInBytes = %d, want %d", got, want)
	}

	// The selection moves to the second character without the buffer changing.
	d.TrySend("abあいcd", 3, 4, false, textinput.CommitNone)
	states = d.Drain()
	if got, want := len(states), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}
	if got, want := states[0].Text, "あい"; got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
	if got, want := states[0].CompositionSelectionStartInBytes, len("あ"); got != want {
		t.Errorf("CompositionSelectionStartInBytes = %d, want %d", got, want)
	}
	if got, want := states[0].CompositionSelectionEndInBytes, len("あい"); got != want {
		t.Errorf("CompositionSelectionEndInBytes = %d, want %d", got, want)
	}
}

func TestDiffSenderCompositionUnchanged(t *testing.T) {
	d := textinput.NewDiffSender("ab", "cd")

	d.TrySend("abあいcd", 2, 3, false, textinput.CommitNone)
	if got, want := len(d.Drain()), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}

	// Neither the preedit nor the selection within it moved.
	d.TrySend("abあいcd", 2, 3, false, textinput.CommitNone)
	if got, want := len(d.Drain()), 0; got != want {
		t.Errorf("len(states) = %d, want %d", got, want)
	}
}

// A platform pinning the caret to the preedit's end reports the same selection
// whatever the buffer's own selection is, so the repeats stay duplicates.
func TestDiffSenderCompositionCaretAtPreeditEnd(t *testing.T) {
	d := textinput.NewDiffSender("ab", "cd")

	d.TrySend("abあいcd", 2, 3, true, textinput.CommitNone)
	states := d.Drain()
	if got, want := len(states), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}
	if got, want := states[0].CompositionSelectionStartInBytes, len("あい"); got != want {
		t.Errorf("CompositionSelectionStartInBytes = %d, want %d", got, want)
	}
	if got, want := states[0].CompositionSelectionEndInBytes, len("あい"); got != want {
		t.Errorf("CompositionSelectionEndInBytes = %d, want %d", got, want)
	}

	d.TrySend("abあいcd", 3, 4, true, textinput.CommitNone)
	if got, want := len(d.Drain()), 0; got != want {
		t.Errorf("len(states) = %d, want %d", got, want)
	}
}

func TestDiffSenderCommitUnchanged(t *testing.T) {
	d := textinput.NewDiffSender("ab", "cd")

	d.TrySend("abあいcd", 4, 4, false, textinput.CommitRegular)
	states := d.Drain()
	if got, want := len(states), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}
	if got, want := states[0].Text, "あい"; got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}

	// The platform reports the committed buffer once more. The commit ended the
	// session, so a state that is not dropped would be held for the next one.
	d.TrySend("abあいcd", 4, 4, false, textinput.CommitRegular)
	if got, want := d.PendingStateCount(), 0; got != want {
		t.Errorf("PendingStateCount() = %d, want %d", got, want)
	}
}

// applyCommit applies c as [textinput.Commit] documents, and returns the text
// around the caret afterwards with the caret's position within it.
func applyCommit(c *textinput.Commit) (text string, caretInBytes int) {
	before, after := c.SurroundingText()
	return before + c.Text() + after, len(before) + len(c.Text())
}

// commitForNextSession hands the states queued behind a commit to a session
// seeded with the surrounding text the application holds after applying it, and
// returns that session's commit.
func commitForNextSession(t *testing.T, d *textinput.DiffSender, textBeforeCaret, textAfterCaret string) *textinput.Commit {
	t.Helper()
	d.StartNextSession(textBeforeCaret, textAfterCaret)
	if err := d.Update(); err != nil {
		t.Fatalf("Update: %v", err)
	}
	c := d.Commit()
	if c == nil {
		t.Fatal("the next session got no commit")
	}
	return c
}

// The platform reports a deletion of the just-committed text before the
// application starts the next session. The next session must delete, not insert
// nothing at its caret.
func TestDiffSenderDeletionAfterCommit(t *testing.T) {
	d := textinput.NewDiffSender("ab", "cd")

	// The IME commits "x" at the caret, ending the session.
	d.TrySend("abxcd", 3, 3, false, textinput.CommitRegular)
	if got, want := len(d.Drain()), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}

	// A backspace removes it before the application opens the next session.
	d.TrySend("abcd", 2, 2, false, textinput.CommitRegular)

	// The application applied the commit, so its buffer is "abx|cd".
	c := commitForNextSession(t, d, "abx", "cd")
	if before, after := c.IsSurroundingTextReplaced(); !before || after {
		t.Errorf("IsSurroundingTextReplaced() = %v, %v, want true, false", before, after)
	}
	text, caret := applyCommit(c)
	if got, want := text, "abcd"; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
	if got, want := caret, len("ab"); got != want {
		t.Errorf("caret = %d, want %d", got, want)
	}
}

// A virtual keyboard reports its backspace as a composition, which cannot
// express a deletion, so the same handoff applies (see [diffSender.trySend]).
func TestDiffSenderCompositionDeletionAfterCommit(t *testing.T) {
	d := textinput.NewDiffSender("ab", "cd")

	d.TrySend("abxcd", 3, 3, true, textinput.CommitRegular)
	if got, want := len(d.Drain()), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}

	d.TrySend("abcd", 2, 2, true, textinput.CommitNone)

	c := commitForNextSession(t, d, "abx", "cd")
	if got, want := c.Text(), ""; got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
	text, caret := applyCommit(c)
	if got, want := text, "abcd"; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
	if got, want := caret, len("ab"); got != want {
		t.Errorf("caret = %d, want %d", got, want)
	}
}

// A deletion carried into the next session is reported in bytes, though the
// platform reports its selection in UTF-16 code units.
func TestDiffSenderDeletionAfterCommitInMultibyteText(t *testing.T) {
	d := textinput.NewDiffSender("あ", "")

	d.TrySend("あい", 2, 2, false, textinput.CommitRegular)
	if got, want := len(d.Drain()), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}

	d.TrySend("あ", 1, 1, false, textinput.CommitRegular)

	c := commitForNextSession(t, d, "あい", "")
	text, caret := applyCommit(c)
	if got, want := text, "あ"; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
	if got, want := caret, len("あ"); got != want {
		t.Errorf("caret = %d, want %d", got, want)
	}
}

// A composition started before the next session ends with the same buffer it
// last reported. The commit must carry the whole composition, not the difference
// from that report.
func TestDiffSenderCompositionFinalizedAfterCommit(t *testing.T) {
	d := textinput.NewDiffSender("ab", "cd")

	d.TrySend("abxcd", 3, 3, false, textinput.CommitRegular)
	if got, want := len(d.Drain()), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}

	// The IME composes "あ" and finalizes it without changing the buffer.
	d.TrySend("abxあcd", 4, 4, false, textinput.CommitNone)
	d.TrySend("abxあcd", 4, 4, false, textinput.CommitRegular)

	c := commitForNextSession(t, d, "abx", "cd")
	if got, want := c.Text(), "あ"; got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
	if before, after := c.IsSurroundingTextReplaced(); before || after {
		t.Errorf("IsSurroundingTextReplaced() = %v, %v, want false, false", before, after)
	}
	text, caret := applyCommit(c)
	if got, want := text, "abxあcd"; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
	if got, want := caret, len("abxあ"); got != want {
		t.Errorf("caret = %d, want %d", got, want)
	}
}

// The composition reported before the next session starts reaches it as a
// preedit, not as a commit.
func TestDiffSenderCompositionAfterCommitReachesNextSession(t *testing.T) {
	d := textinput.NewDiffSender("ab", "cd")

	d.TrySend("abxcd", 3, 3, false, textinput.CommitRegular)
	if got, want := len(d.Drain()), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}

	d.TrySend("abxあcd", 4, 4, false, textinput.CommitNone)

	d.StartNextSession("abx", "cd")
	if err := d.Update(); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if c := d.Commit(); c != nil {
		t.Fatalf("the session committed %q, want a composition", c.Text())
	}
	if got, want := d.Composition().Text(), "あ"; got != want {
		t.Errorf("Composition().Text() = %q, want %q", got, want)
	}
}

// Typing faster than the application opens sessions commits every keystroke at
// the caret of the session that takes it.
func TestDiffSenderInsertionsAfterCommit(t *testing.T) {
	d := textinput.NewDiffSender("ab", "cd")

	d.TrySend("abxcd", 3, 3, false, textinput.CommitRegular)
	d.TrySend("abxycd", 4, 4, false, textinput.CommitRegular)
	d.TrySend("abxyzcd", 5, 5, false, textinput.CommitRegular)
	if got, want := d.PendingStateCount(), 2; got != want {
		t.Fatalf("PendingStateCount() = %d, want %d", got, want)
	}

	textBeforeCaret, textAfterCaret := "abx", "cd"
	for _, want := range []string{"y", "z"} {
		c := commitForNextSession(t, d, textBeforeCaret, textAfterCaret)
		if got := c.Text(); got != want {
			t.Errorf("Text = %q, want %q", got, want)
		}
		if before, after := c.IsSurroundingTextReplaced(); before || after {
			t.Errorf("IsSurroundingTextReplaced() = %v, %v, want false, false", before, after)
		}
		text, caret := applyCommit(c)
		textBeforeCaret, textAfterCaret = text[:caret], text[caret:]
	}

	if got, want := textBeforeCaret+textAfterCaret, "abxyzcd"; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}

// A backspace on a virtual keyboard removes committed text, which a preedit
// cannot express, so it is committed instead (see [diffSender.trySend]).
func TestDiffSenderCompositionRemovingCommittedText(t *testing.T) {
	d := textinput.NewDiffSender("ab", "cd")

	d.TrySend("acd", 1, 1, true, textinput.CommitNone)
	states := d.Drain()
	if got, want := len(states), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}
	if got, want := states[0].CommitKind, textinput.CommitRegular; got != want {
		t.Errorf("CommitKind = %d, want %d", got, want)
	}
	if got, want := states[0].Text, ""; got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
	if got, want := states[0].ReplacementStartInBytes, 1; got != want {
		t.Errorf("ReplacementStartInBytes = %d, want %d", got, want)
	}
	if got, want := states[0].ReplacementEndInBytes, 2; got != want {
		t.Errorf("ReplacementEndInBytes = %d, want %d", got, want)
	}
}

// TestHandlePlatformStateBeforeSessionRegistration simulates the platform
// reporting a state after the platform text input started but before the
// session is registered (startSession installs the session only after starting
// the platform text input). The state must not be taken for the deprecated
// Field's input.
func TestHandlePlatformStateBeforeSessionRegistration(t *testing.T) {
	h := textinput.NewPlatformStateHandler("abc")

	// The platform echoes the seeded value while no session is registered.
	if got, want := h.Handle("abc", 3, 3, textinput.CommitRegular, false), false; got != want {
		t.Errorf("Handle = %t, want %t", got, want)
	}
	if states := h.Drain(); len(states) != 0 {
		t.Errorf("states delivered before session registration: %+v", states)
	}
	if !h.IsOpen() {
		t.Errorf("events closed before session registration")
	}

	// Once the session is registered, states are diffed against it.
	h.RegisterSession("abc", "")
	h.Handle("abcd", 4, 4, textinput.CommitRegular, false)
	states := h.Drain()
	if got, want := len(states), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}
	if got, want := states[0].Text, "d"; got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
	if got, want := states[0].CommitKind, textinput.CommitRegular; got != want {
		t.Errorf("CommitKind = %d, want %d", got, want)
	}
}

// TestHandlePlatformStateLegacyField verifies that the legacy whole-value path
// still serves a focused Field.
func TestHandlePlatformStateLegacyField(t *testing.T) {
	h := textinput.NewPlatformStateHandler("")

	if got, want := h.Handle("abc", 3, 3, textinput.CommitRegular, true), true; got != want {
		t.Errorf("Handle = %t, want %t", got, want)
	}
	states := h.Drain()
	if got, want := len(states), 1; got != want {
		t.Fatalf("len(states) = %d, want %d", got, want)
	}
	if got, want := states[0].Text, "abc"; got != want {
		t.Errorf("Text = %q, want %q", got, want)
	}
	if h.IsOpen() {
		t.Errorf("events still open after a legacy commit")
	}
	if !h.LegacyCleared() {
		t.Errorf("LegacyCleared() = false, want true")
	}
}
