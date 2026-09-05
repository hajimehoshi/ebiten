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
	"image"
	"sync"
	"unicode/utf8"
)

// sessionState describes whether a session has closed and how.
type sessionState int

const (
	sessionStateOpen sessionState = iota

	// sessionStateClosed is any closing but the user's: a commit, a
	// cancellation, or a platform-side teardown.
	sessionStateClosed

	// sessionStateClosedByUser is the user ending text inputting from the
	// platform side, e.g. by dismissing the virtual keyboard.
	sessionStateClosedByUser
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
	events *textInputEvents
	state  sessionState

	textBeforeCaret string
	textAfterCaret  string

	// caretBounds is where the caret was when the session started. The caret cannot move
	// vertically within a session: a line break commits instead of joining the preedit, and a
	// caller-driven move must confirm the session first.
	caretBounds image.Rectangle

	// Composition state, written by the user goroutine in Update and read
	// by the platform's IME query callbacks on the platform thread. The
	// mutex guards those cross-thread reads.
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
	commitKind commitKind
	commit     Commit
}

// setComposition updates the composition state. It must be called from the
// user goroutine.
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
// callback, which releases what the platform holds for the session; pass
// false only when the platform tore the session down by itself.
func (s *session) markClosed(callPlatformEnd bool) {
	if s.events.takeEndedByUser() {
		s.state = sessionStateClosedByUser
	} else {
		s.state = sessionStateClosed
	}
	clearVirtualKeyboardFromUI()
	if callPlatformEnd {
		s.end()
	}
	s.events.clearActiveSessionIf(s)
}

// startSession begins a new IME session at the given caret.
//
// A session already owning the platform IME is cancelled first.
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

	// Only one session can own the platform IME, and the previous one's teardown
	// closes the event channel this session is about to be handed.
	theTextInput.cancelSessionIfNeeded()

	ch, end := startTextInput(opts.CaretBounds, opts.TextBeforeCaret, opts.TextAfterCaret)
	if ch == nil {
		return nil, nil
	}
	s := &session{
		ch:              ch,
		end:             end,
		events:          &theTextInput.events,
		textBeforeCaret: opts.TextBeforeCaret,
		textAfterCaret:  opts.TextAfterCaret,
		caretBounds:     opts.CaretBounds,
	}
	theTextInput.events.setActiveSession(s)
	return s, nil
}

// Update pumps platform IME events queued since the last call. Callers must
// invoke Update once per tick to observe composition updates, commits, and
// platform-side teardown.
func (s *session) Update() error {
	if s.IsClosed() {
		return nil
	}
	reportVirtualKeyboardToUI(s.caretBounds)
	s.composingThisUpdate = false
	for {
		select {
		case st, ok := <-s.ch:
			if !ok {
				s.markClosed(false)
				return nil
			}
			if st.Error != nil {
				// The platform holds the session even after it reported an
				// error, so release it here as the commit path does.
				s.markClosed(true)
				return st.Error
			}
			if st.CommitKind.committed() {
				replStart := st.ReplacementStartInBytes
				replEnd := st.ReplacementEndInBytes
				preLen := len(s.textBeforeCaret)
				switch {
				case st.ReplacementRelativeToCaret:
					// The edit was diffed against a buffer whose caret sat where
					// this session's does, but whose surrounding text was the
					// previous session's. Clamp: this session decides how much
					// text around the caret it exposes.
					total := preLen + len(s.textAfterCaret)
					replStart = min(max(preLen+replStart, 0), total)
					replEnd = min(max(preLen+replEnd, replStart), total)
				case replStart == noReplacement || replEnd == noReplacement:
					replStart = preLen
					replEnd = preLen
				}
				s.commitKind = st.CommitKind
				s.commit = Commit{
					text:            st.Text,
					textBeforeCaret: s.textBeforeCaret,
					textAfterCaret:  s.textAfterCaret,
					replStart:       replStart,
					replEnd:         replEnd,
					passthroughKey:  st.CommitKind == commitWithPassthroughKey,
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
	if s.IsClosed() {
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
	return s.commitKind.committed()
}

// IsCommittedWithPassthroughKey reports whether the commit arrived with a key
// press that also passes through to the game. Defined only when IsCommitted
// returns true.
func (s *session) IsCommittedWithPassthroughKey() bool {
	return s.commitKind == commitWithPassthroughKey
}

// Commit returns the IME's committed text and the byte range that the
// text replaces in the joined TextBeforeCaret+TextAfterCaret buffer.
//
// Defined only when IsCommitted returns true; otherwise returns the zero
// value.
func (s *session) Commit() *Commit {
	return &s.commit
}

// compositionAsCommit returns a Commit that inserts the current composition at
// the caret with no surrounding-text replacement, as if the IME had committed
// the preedit.
func (s *session) compositionAsCommit() *Commit {
	preLen := len(s.textBeforeCaret)
	return &Commit{
		text:            s.loadComposition().text,
		textBeforeCaret: s.textBeforeCaret,
		textAfterCaret:  s.textAfterCaret,
		replStart:       preLen,
		replEnd:         preLen,
	}
}

// IsClosed reports whether the session has ended for any reason.
// IsClosed becomes true after Update observes a commit, after the platform
// unilaterally ends the session, or after Cancel is called.
func (s *session) IsClosed() bool {
	return s.state != sessionStateOpen
}

// IsClosedByUser reports whether the session closed because the user ended
// text inputting from the platform side, e.g. by dismissing the virtual
// keyboard. Implies IsClosed.
func (s *session) IsClosedByUser() bool {
	return s.state == sessionStateClosedByUser
}

// Cancel aborts an in-progress composition and releases the platform IME.
//
// Cancel is a no-op if the session is already closed. Callers must either
// observe a commit via Update or call Cancel; otherwise the platform IME may
// be left in an indeterminate state.
func (s *session) Cancel() {
	if s.IsClosed() {
		return
	}
	s.markClosed(true)
	theTextInput.abandonTarget()
}
