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
// snapshot against the session's baseline.
type diffSender struct {
	events *textInputEvents

	// lastSentValue/lastSentCommitted: the last state sent, used to drop
	// trailing duplicate events (e.g. the input after compositionend).
	lastSentValue     string
	lastSentCommitted bool

	// committedThisSession reports whether this session already committed; a
	// second commit before the next session starts falls back to a caret insert
	// (see trySend).
	committedThisSession bool
}

// reset records that the platform buffer was reseeded with value, the session's
// surrounding text, making it the diff baseline.
func (d *diffSender) reset(value string) {
	d.lastSentValue = value
	d.lastSentCommitted = true
	d.committedThisSession = false
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

// trySend diffs the buffer against session s's surrounding text and sends the
// edit as a replacement range. value is the buffer's current content and
// selStartInUTF16 and selEndInUTF16 its selection; caretAtPreeditEnd is
// documented at [compositionSelectionInBytes].
func (d *diffSender) trySend(s *session, value string, selStartInUTF16, selEndInUTF16 int, caretAtPreeditEnd bool, kind commitKind) {
	if value == d.lastSentValue && kind.committed() == d.lastSentCommitted {
		return
	}

	if d.committedThisSession {
		// Already committed this session; the buffer has moved past our baseline.
		// Send just the new delta as a caret insert (rapid double-commit case).
		text, _, _ := computeReplacement(d.lastSentValue, value, -1)
		d.events.send(textInputState{
			Text:                    text,
			ReplacementStartInBytes: noReplacement,
			ReplacementEndInBytes:   noReplacement,
			CommitKind:              kind,
		})
		d.lastSentValue = value
		d.lastSentCommitted = kind.committed()
		if kind.committed() {
			d.events.end()
		}
		return
	}

	baseline := s.textBeforeCaret + s.textAfterCaret

	if !kind.committed() {
		// Composition: the preedit ends where the unchanged after-caret text
		// begins. Anchor there rather than at the caret, which may sit inside
		// the preedit during conversion, so the preedit is located correctly
		// even when the surrounding text repeats. Report the selection relative
		// to the preedit start.
		preeditEnd := len(value) - len(s.textAfterCaret)
		text, replStartInBytes, replEndInBytes := computeReplacement(baseline, value, preeditEnd)
		// A composition can only insert a preedit at the caret. When the edit
		// removes committed bytes (replStartInBytes < replEndInBytes) — e.g. a
		// backspace on a virtual keyboard — a preedit cannot express it, so
		// commit the replacement instead.
		if replStartInBytes >= replEndInBytes {
			selStartInBytes, selEndInBytes := compositionSelectionInBytes(value, replStartInBytes, len(text), selStartInUTF16, selEndInUTF16, caretAtPreeditEnd)
			d.events.send(textInputState{
				Text:                             text,
				CompositionSelectionStartInBytes: selStartInBytes,
				CompositionSelectionEndInBytes:   selEndInBytes,
				ReplacementStartInBytes:          noReplacement,
				ReplacementEndInBytes:            noReplacement,
				CommitKind:                       kind,
			})
			d.lastSentValue = value
			d.lastSentCommitted = false
			return
		}
		// A virtual-keyboard deletion removes committed bytes; deliver it as a
		// commit whose key does not pass through to the game.
		kind = commitRegular
	}

	// The caret is at the end of the committed text; anchor on it so an
	// insertion into repeated surrounding text is not misplaced at the end.
	caretInBytes := convertUTF16CountToByteCount(value, selStartInUTF16)
	text, replStartInBytes, replEndInBytes := computeReplacement(baseline, value, caretInBytes)

	d.events.send(textInputState{
		Text:                    text,
		ReplacementStartInBytes: replStartInBytes,
		ReplacementEndInBytes:   replEndInBytes,
		CommitKind:              kind,
	})
	d.lastSentValue = value
	d.lastSentCommitted = true
	d.committedThisSession = true
	d.events.end()
}
