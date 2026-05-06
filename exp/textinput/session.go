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

import (
	"fmt"
	"sync"
	"unicode/utf8"
)

// session is one IME composition session.
//
// A session lives from startSession until one of:
//   - the IME commits (IsCommitted becomes true),
//   - the platform tears down the session (e.g. OS focus loss),
//   - Cancel is called.
//
// Update must be called once per tick to drain platform events. Without
// Update, the session never observes commits or platform-side teardown,
// even if the underlying composition state visible to platform IME query
// callbacks is kept up to date.
type session struct {
	ch     <-chan textInputState
	end    func()
	closed bool

	textBeforeCaret string
	textAfterCaret  string

	// Composition state, written by platform IME callbacks (synchronously),
	// read by platform IME query callbacks and by Composition. The mutex
	// guards reads from the user goroutine.
	compositionM sync.Mutex
	composition  Composition

	// composingThisUpdate is true if the most recent Update drained any
	// non-committed state. It captures transient activity that the
	// composition-text snapshot misses — for example, when a backspace
	// clears the preedit to empty within the same tick. Touched only from
	// the user goroutine in Update / IsCompositing.
	composingThisUpdate bool

	// Committed-state fields, populated by Update when a committed state is
	// drained. Valid only after IsCommitted returns true.
	committed bool
	commit    Commit
}

// setComposition updates the composition state read by platform IME callbacks.
// Called from the platform layer when the IME emits a composition update.
func (s *session) setComposition(text string, selStart, selEnd int) {
	s.compositionM.Lock()
	defer s.compositionM.Unlock()
	s.composition.text = text
	s.composition.selStart = selStart
	s.composition.selEnd = selEnd
}

// loadComposition returns the latest composition text seen from the platform IME.
func (s *session) loadComposition() Composition {
	s.compositionM.Lock()
	defer s.compositionM.Unlock()
	return s.composition
}

// markClosed transitions s to closed and clears the active-session pointer if
// it points at s. callPlatformEnd controls whether to invoke the platform end
// callback; pass false when the platform itself has already ended (channel
// closed from below).
func (s *session) markClosed(callPlatformEnd bool) {
	s.closed = true
	if callPlatformEnd {
		s.end()
	}
	theTextInput.events.clearActiveSessionIf(s)
}

// startSession begins a new IME session at the given caret.
//
// If opts is nil, zero-value defaults are used.
//
// startSession returns (nil, nil) when a session cannot be started in the
// current environment. This covers both permanently unsupported platforms
// (e.g. Xbox, JS without a DOM) and transient conditions (e.g. iOS Safari
// requires a user-interaction event to focus, or noime polled with no input
// this tick). Callers should treat a nil session as "try again next tick or
// fall back to plain key handling."
//
// A non-nil error indicates a bug in the caller — currently only
// TextBeforeCaret or TextAfterCaret containing invalid UTF-8.
func startSession(opts *SessionOptions) (*session, error) {
	if opts == nil {
		opts = &SessionOptions{}
	}
	if !utf8.ValidString(opts.TextBeforeCaret) {
		return nil, fmt.Errorf("textinput: TextBeforeCaret is not valid UTF-8")
	}
	if !utf8.ValidString(opts.TextAfterCaret) {
		return nil, fmt.Errorf("textinput: TextAfterCaret is not valid UTF-8")
	}

	ch, end := start(opts.CaretBounds)
	if ch == nil {
		return nil, nil
	}
	s := &session{
		ch:              ch,
		end:             end,
		textBeforeCaret: opts.TextBeforeCaret,
		textAfterCaret:  opts.TextAfterCaret,
	}
	theTextInput.events.setActiveSession(s)
	return s, nil
}

// Update pumps platform IME events queued since the last call. Callers must
// invoke Update once per tick to observe composition updates, commits, and
// platform-side teardown.
func (s *session) Update() error {
	if s.closed {
		return nil
	}
	s.composingThisUpdate = false
	for {
		select {
		case st, ok := <-s.ch:
			if !ok {
				s.markClosed(false)
				return nil
			}
			if st.Error != nil {
				s.markClosed(false)
				return st.Error
			}
			if st.Committed {
				replStart := st.ReplacementStartInBytes
				replEnd := st.ReplacementEndInBytes
				if replStart == noReplacement || replEnd == noReplacement {
					preLen := len(s.textBeforeCaret)
					replStart = preLen
					replEnd = preLen
				}
				s.committed = true
				s.commit = Commit{
					text:            st.Text,
					textBeforeCaret: s.textBeforeCaret,
					textAfterCaret:  s.textAfterCaret,
					replStart:       replStart,
					replEnd:         replEnd,
				}
				s.markClosed(true)
				return nil
			}
			// Non-committed state: mirror the preedit into composition so
			// Composition() reflects it. composingThisUpdate flags the
			// activity so IsCompositing reports true even when the
			// resulting preedit is empty (e.g. backspace cleared it within
			// the same tick).
			s.setComposition(st.Text, st.CompositionSelectionStartInBytes, st.CompositionSelectionEndInBytes)
			s.composingThisUpdate = true
		default:
			return nil
		}
	}
}

// Composition returns the current preedit text and the IME-side selection
// within it. Returns the zero value when no composition is in progress.
func (s *session) Composition() Composition {
	return s.loadComposition()
}

// IsCompositing reports whether the IME currently owns input on this session.
// True when the preedit is non-empty, or when the most recent Update drained
// any non-committed state (covers the tick in which the IME consumes a key
// like backspace that empties the preedit). Returns false once the session
// has closed, regardless of any stale composition state.
func (s *session) IsCompositing() bool {
	if s.closed {
		return false
	}
	if s.composingThisUpdate {
		return true
	}
	return s.loadComposition().text != ""
}

// IsCommitted reports whether the session ended because the IME committed.
// Implies IsClosed. Once IsCommitted returns true, Commit returns the data
// the IME recorded.
func (s *session) IsCommitted() bool {
	return s.committed
}

// Commit returns the IME's committed text and the byte range that the
// text replaces in the joined TextBeforeCaret+TextAfterCaret buffer.
//
// Defined only when IsCommitted returns true; otherwise returns the zero
// value.
func (s *session) Commit() *Commit {
	return &s.commit
}

// IsClosed reports whether the session has ended for any reason.
// IsClosed becomes true after Update observes a commit, after the platform
// unilaterally ends the session, or after Cancel is called.
func (s *session) IsClosed() bool {
	return s.closed
}

// Cancel aborts an in-progress composition and releases the platform IME.
//
// Cancel is a no-op if the session is already closed. Callers must either
// observe a commit via Update or call Cancel; otherwise the platform IME may
// be left in an indeterminate state.
func (s *session) Cancel() {
	if s.closed {
		return
	}
	s.markClosed(true)
}
