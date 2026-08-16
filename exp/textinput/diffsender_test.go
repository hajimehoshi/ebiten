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
