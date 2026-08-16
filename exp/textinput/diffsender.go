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

package textinput

// diffSender turns snapshots of a platform text buffer, seeded with the active
// session's surrounding text, into textInputState updates, by diffing each
// snapshot against a baseline.
//
// A commit ends the session, while the platform keeps reporting edits until the
// application starts the next one. Those edits belong to that next session, so
// they are diffed against the buffer the commit left behind, which the
// application reproduces by applying the commit, and are expressed relative to
// its caret: the next session brings its own surrounding text, and applying the
// commit leaves its caret at the end of the committed text.
type diffSender struct {
	events *textInputEvents

	// lastSentValue/lastSentCommitted/lastSentSelStart/lastSentSelEnd: the last
	// state sent, used to drop trailing duplicate events (e.g. the input after
	// compositionend). The selection is the one reported within the preedit,
	// which can move while the buffer stays the same.
	lastSentValue     string
	lastSentCommitted bool
	lastSentSelStart  int
	lastSentSelEnd    int

	// committedValue is the buffer as of the last commit sent and
	// committedCaretInBytes the caret within it, at the end of the committed
	// text. Meaningful only while committedThisSession is true.
	//
	// A composition leaves them alone: its finalization must commit the whole
	// composition, not the difference from its last snapshot.
	committedValue        string
	committedCaretInBytes int

	// committedThisSession reports whether the session the buffer was seeded for
	// has already committed.
	committedThisSession bool
}

// reset records that the platform buffer was reseeded with value, the session's
// surrounding text, making it the diff baseline.
func (d *diffSender) reset(value string) {
	d.lastSentValue = value
	d.lastSentCommitted = true
	d.lastSentSelStart = 0
	d.lastSentSelEnd = 0
	d.committedValue = ""
	d.committedCaretInBytes = 0
	d.committedThisSession = false
}

// baseline returns the text the buffer's snapshots are diffed against and the
// caret within it: session s's surrounding text until s commits, and the buffer
// that commit left behind afterwards.
func (d *diffSender) baseline(s *session) (value string, caretInBytes int) {
	if d.committedThisSession {
		return d.committedValue, d.committedCaretInBytes
	}
	return s.textBeforeCaret + s.textAfterCaret, len(s.textBeforeCaret)
}

// compositionSelectionInBytes returns the selection within the preedit, as a
// byte range relative to the preedit's start. The preedit occupies preeditLen
// bytes of value from preeditStart, and selStartInUTF16 and selEndInUTF16 are
// the buffer's selection.
//
// caretAtPreeditEnd pins the selection to the preedit's end, for platforms
// whose reported selection does not track the preedit (a virtual keyboard, or
// iOS reporting the selection from before the preedit was inserted).
func compositionSelectionInBytes(value string, preeditStart, preeditLen, selStartInUTF16, selEndInUTF16 int, caretAtPreeditEnd bool) (start, end int) {
	if caretAtPreeditEnd {
		return preeditLen, preeditLen
	}
	startInBytes := convertUTF16CountToByteCount(value, selStartInUTF16)
	endInBytes := convertUTF16CountToByteCount(value, selEndInUTF16)
	return min(max(startInBytes-preeditStart, 0), preeditLen), min(max(endInBytes-preeditStart, 0), preeditLen)
}

// handlePlatformState reports a platform text-buffer state: diffed by sender
// for the active session, or delivered whole to the deprecated focused Field.
// fieldFocused reports whether a Field is focused, evaluated by the caller
// outside its own locks. handlePlatformState reports whether the caller must
// clear the platform text buffer.
//
// A state with neither an active session nor a focused Field is dropped: it is
// either the echo of a seeding whose session is not yet registered
// (startSession installs the session only after starting the platform text
// input), or the teardown noise of a dismissal.
func handlePlatformState(events *textInputEvents, sender *diffSender, legacyCleared *bool, value string, selStartInUTF16, selEndInUTF16 int, caretAtPreeditEnd bool, kind commitKind, fieldFocused bool) (clearBuffer bool) {
	if s := events.getActiveSession(); s != nil {
		sender.trySend(s, value, selStartInUTF16, selEndInUTF16, caretAtPreeditEnd, kind)
		return false
	}

	// The legacy whole-value path serves only the deprecated Field.
	// TODO: Remove this path once Field is gone; a Composer session is always
	// active otherwise.
	if !fieldFocused {
		return false
	}
	if !events.isOpen() {
		return false
	}
	if *legacyCleared {
		*legacyCleared = false
		if value == "" {
			return false
		}
	}
	events.send(textInputState{
		Text:                             value,
		CompositionSelectionStartInBytes: convertUTF16CountToByteCount(value, selStartInUTF16),
		CompositionSelectionEndInBytes:   convertUTF16CountToByteCount(value, selEndInUTF16),
		ReplacementStartInBytes:          noReplacement,
		ReplacementEndInBytes:            noReplacement,
		CommitKind:                       kind,
	})
	if kind.committed() {
		events.end()
		*legacyCleared = true
		return true
	}
	return false
}

// trySend diffs the buffer against the baseline and sends the edit as a
// replacement range. value is the buffer's current content and selStartInUTF16
// and selEndInUTF16 its selection; caretAtPreeditEnd is documented at
// [compositionSelectionInBytes].
func (d *diffSender) trySend(s *session, value string, selStartInUTF16, selEndInUTF16 int, caretAtPreeditEnd bool, kind commitKind) {
	// A composition is compared against the last state sent in
	// trySendComposition, which knows the selection it would report.
	if kind.committed() && d.lastSentCommitted && value == d.lastSentValue {
		return
	}

	if !kind.committed() {
		if d.trySendComposition(s, value, selStartInUTF16, selEndInUTF16, caretAtPreeditEnd) {
			return
		}
		// A virtual-keyboard deletion removes committed bytes; deliver it as a
		// commit whose key does not pass through to the game.
		kind = commitRegular
	}

	baseline, baselineCaretInBytes := d.baseline(s)

	// The caret is at the end of the committed text; anchor on it so an
	// insertion into repeated surrounding text is not misplaced at the end.
	caretInBytes := convertUTF16CountToByteCount(value, selStartInUTF16)
	text, replStartInBytes, replEndInBytes := computeReplacement(baseline, value, caretInBytes)

	// Applying the commit leaves the application's caret at the end of the
	// committed text: the baseline for the edits reported until the next session
	// starts.
	committedCaretInBytes := replStartInBytes + len(text)

	// The session the baseline came from has ended, so the range is not in the
	// coordinates of the session that receives this commit.
	relative := d.committedThisSession
	if relative {
		replStartInBytes -= baselineCaretInBytes
		replEndInBytes -= baselineCaretInBytes
	}

	d.events.send(textInputState{
		Text:                       text,
		ReplacementStartInBytes:    replStartInBytes,
		ReplacementEndInBytes:      replEndInBytes,
		ReplacementRelativeToCaret: relative,
		CommitKind:                 kind,
	})
	d.lastSentValue = value
	d.lastSentCommitted = true
	d.committedValue = value
	d.committedCaretInBytes = committedCaretInBytes
	d.committedThisSession = true
	d.events.end()
}

// trySendComposition sends the buffer as a preedit inserted at the baseline's
// caret, and reports whether it could: a preedit cannot express an edit that
// removes text from the baseline. The arguments are documented at
// [diffSender.trySend].
func (d *diffSender) trySendComposition(s *session, value string, selStartInUTF16, selEndInUTF16 int, caretAtPreeditEnd bool) bool {
	// The preedit ends where the unchanged after-caret text begins. Anchor there
	// rather than at the caret, which may sit inside the preedit during
	// conversion, so the preedit is located correctly even when the surrounding
	// text repeats.
	baseline, baselineCaretInBytes := d.baseline(s)
	preeditEnd := len(value) - (len(baseline) - baselineCaretInBytes)
	text, replStartInBytes, replEndInBytes := computeReplacement(baseline, value, preeditEnd)
	if replStartInBytes < replEndInBytes {
		return false
	}
	if text == "" && d.lastSentCommitted {
		// Nothing is composed, and no preedit was reported that needs clearing.
		return true
	}

	// The selection is relative to the preedit's start.
	selStartInBytes, selEndInBytes := compositionSelectionInBytes(value, replStartInBytes, len(text), selStartInUTF16, selEndInUTF16, caretAtPreeditEnd)
	if value == d.lastSentValue && !d.lastSentCommitted &&
		selStartInBytes == d.lastSentSelStart && selEndInBytes == d.lastSentSelEnd {
		// Neither the preedit nor the selection within it moved.
		return true
	}

	d.events.send(textInputState{
		Text:                             text,
		CompositionSelectionStartInBytes: selStartInBytes,
		CompositionSelectionEndInBytes:   selEndInBytes,
		ReplacementStartInBytes:          noReplacement,
		ReplacementEndInBytes:            noReplacement,
		CommitKind:                       commitNone,
	})
	d.lastSentValue = value
	d.lastSentCommitted = false
	d.lastSentSelStart = selStartInBytes
	d.lastSentSelEnd = selEndInBytes
	return true
}
