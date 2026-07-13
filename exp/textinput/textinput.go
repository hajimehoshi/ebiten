// Copyright 2023 The Ebitengine Authors
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

// Package textinput provides a text-inputting controller.
// This package is experimental and the API might be changed in the future.
//
// This package is supported on Windows, macOS, and Web browsers so far.
package textinput

import (
	"fmt"
	"image"
	"slices"
	"sync"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
	"github.com/hajimehoshi/ebiten/v2/internal/vmguest"
)

// noReplacement is the sentinel value for [textInputState.ReplacementStartInBytes]
// and [textInputState.ReplacementEndInBytes] meaning "no replacement, at
// the caret of the receiving session." [session.Update] resolves it to
// len(textBeforeCaret) on commit.
const noReplacement = -1

// commitKind describes whether a textInputState is a final commit and, if so,
// how it was produced.
type commitKind int

const (
	// commitNone marks a composition (preedit) update, not a commit.
	commitNone commitKind = iota

	// commitWithoutKeyPress marks a final committed edit delivered without a key
	// press the game receives (a desktop IME commit, a suggestion tap).
	commitWithoutKeyPress

	// commitWithKeyPress marks a final committed edit delivered with a key press
	// the game also receives (Return on a virtual keyboard). The Composer leaves
	// handled false for it so the caller still acts on that key.
	commitWithKeyPress
)

// committed reports whether k is a final commit.
func (k commitKind) committed() bool {
	return k != commitNone
}

// textInputState is the internal record of an IME event flowing from the
// platform layer to the Go side via a channel.
type textInputState struct {
	// Text is the current composition text, or the committed text for a final
	// commit (see CommitKind).
	Text string

	// CompositionSelectionStartInBytes is the start position of the selection
	// within Text, in bytes. Meaningful only for a composition (CommitKind is
	// commitNone).
	CompositionSelectionStartInBytes int

	// CompositionSelectionEndInBytes is the end position of the selection
	// within Text, in bytes. Meaningful only for a composition (CommitKind is
	// commitNone).
	CompositionSelectionEndInBytes int

	// ReplacementStartInBytes is the start position of the byte range that
	// Text replaces, in the joined TextBeforeCaret+TextAfterCaret buffer.
	// Meaningful only for a final commit. Use [noReplacement] for the
	// "no replacement, at the caret" case.
	ReplacementStartInBytes int

	// ReplacementEndInBytes is the end position of the byte range that Text
	// replaces, in the joined TextBeforeCaret+TextAfterCaret buffer.
	// Meaningful only for a final commit. Use [noReplacement] for the
	// "no replacement, at the caret" case.
	ReplacementEndInBytes int

	// CommitKind reports whether Text is a final commit and, if so, how it was
	// produced.
	CommitKind commitKind

	// Error is an error that happens during text inputting.
	Error error
}

// startTextInput starts text inputting.
// startTextInput returns a channel to send the state repeatedly, and a function to end the text inputting.
//
// A platform may mirror textBeforeCaret and textAfterCaret, the surrounding
// text, into its input buffer so an edit reported by the OS can be expressed as
// a replacement range within it.
//
// startTextInput returns nil and nil if the current environment doesn't support this package.
func startTextInput(bounds image.Rectangle, textBeforeCaret, textAfterCaret string) (states <-chan textInputState, close func()) {
	cMinX, cMinY := ui.Get().LogicalPositionToClientPositionInNativePixels(float64(bounds.Min.X), float64(bounds.Min.Y))
	cMaxX, cMaxY := ui.Get().LogicalPositionToClientPositionInNativePixels(float64(bounds.Max.X), float64(bounds.Max.Y))
	return theTextInput.backend().Start(image.Rect(int(cMinX), int(cMinY), int(cMaxX), int(cMaxY)), textBeforeCaret, textAfterCaret)
}

func convertUTF16CountToByteCount(text string, c int) int {
	if !utf8.ValidString(text) {
		return -1
	}
	if c == 0 {
		return 0
	}
	var utf16Len int
	for idx, r := range text {
		l16 := utf16.RuneLen(r)
		if l16 < 0 {
			panic(fmt.Sprintf("textinput: invalid rune: %c", r))
		}
		utf16Len += l16
		if utf16Len >= c {
			l8 := utf8.RuneLen(r)
			if l8 < 0 {
				panic(fmt.Sprintf("textinput: invalid rune: %c", r))
			}
			return idx + l8
		}
	}
	return -1
}

func convertByteCountToUTF16Count(text string, c int) int {
	if !utf8.ValidString(text) {
		return -1
	}
	if c == 0 {
		return 0
	}
	var utf16Len int
	for idx, r := range text {
		l16 := utf16.RuneLen(r)
		if l16 < 0 {
			panic(fmt.Sprintf("textinput: invalid rune length for rune %c", r))
		}
		utf16Len += l16
		l8 := utf8.RuneLen(r)
		if l8 < 0 {
			panic(fmt.Sprintf("textinput: invalid rune length for rune %c", r))
		}
		if idx+l8 >= c {
			return utf16Len
		}
	}
	return -1
}

// computeReplacement returns the single contiguous edit turning baseline into
// newText: the replacement text and the rune-aligned byte range
// [startInBytes, endInBytes) it replaces in baseline. The range is what remains
// after stripping the longest common prefix and suffix, so it is not
// necessarily the minimal edit.
//
// A non-negative caretInBytes is taken as the end of the edited region in
// newText and anchors the replacement, disambiguating an edit into repeated
// surrounding text — e.g. committing "na" at "ba|na" — where the prefix/suffix
// span alone would wrongly land at the end. Pass a negative value when the
// caret does not mark the edit's end (a composition preedit) or is unknown.
func computeReplacement(baseline, newText string, caretInBytes int) (replacement string, startInBytes, endInBytes int) {
	// Common prefix, rune by rune.
	var prefix int
	for prefix < len(baseline) && prefix < len(newText) {
		rb, size := utf8.DecodeRuneInString(baseline[prefix:])
		rn, _ := utf8.DecodeRuneInString(newText[prefix:])
		if rb != rn {
			break
		}
		prefix += size
	}

	// Caret-anchored: the text after the caret is unchanged, so it must be a
	// suffix of baseline. This holds only when the caret sits at the end of the
	// edited region (a commit); otherwise fall through to the common-suffix
	// scan below.
	if 0 <= caretInBytes && caretInBytes <= len(newText) {
		if end := len(baseline) - (len(newText) - caretInBytes); end >= 0 && newText[caretInBytes:] == baseline[end:] {
			start := min(prefix, caretInBytes, end)
			return newText[start:caretInBytes], start, end
		}
	}

	// Common suffix, rune by rune, without crossing the prefix.
	sufBaseline, sufNew := len(baseline), len(newText)
	for sufBaseline > prefix && sufNew > prefix {
		rb, size := utf8.DecodeLastRuneInString(baseline[:sufBaseline])
		rn, _ := utf8.DecodeLastRuneInString(newText[:sufNew])
		if rb != rn {
			break
		}
		sufBaseline -= size
		sufNew -= size
	}

	return newText[prefix:sufNew], prefix, sufBaseline
}

// findLineBounds returns the byte offsets bounding the line of text that
// contains the selection [selStart, selEnd]. lineStart is the position right
// after the previous line break (or 0 if none), and lineEnd is the position of
// the next line break (or len(text) if none). The line break bytes themselves
// are excluded from both ends.
//
// Line breaks that fall within [selStart, selEnd) are ignored, so a selection
// crossing line breaks yields a single combined line.
func findLineBounds(text string, selStart, selEnd int) (lineStart, lineEnd int) {
	selStart = min(max(selStart, 0), len(text))
	selEnd = min(max(selEnd, selStart), len(text))

	for i := selStart; i > 0; {
		r, size := utf8.DecodeLastRuneInString(text[:i])
		if isLineBreak(r) {
			lineStart = i
			break
		}
		i -= size
	}

	lineEnd = len(text)
	for i := selEnd; i < len(text); {
		r, size := utf8.DecodeRuneInString(text[i:])
		if isLineBreak(r) {
			lineEnd = i
			break
		}
		i += size
	}
	return
}

// isLineBreak reports whether r is a line-break codepoint.
func isLineBreak(r rune) bool {
	switch r {
	case '\n', '\v', '\f', '\r':
		return true
	case '\u0085', // NEL
		'\u2028', // LS
		'\u2029': // PS
		return true
	}
	return false
}

// textInputBackend produces the raw text-input state stream for sessions.
type textInputBackend interface {
	// Start starts text inputting, with the same contract as [startTextInput]
	// except that bounds is in client-side native pixels.
	Start(bounds image.Rectangle, textBeforeCaret, textAfterCaret string) (states <-chan textInputState, close func())

	// markIMEDiscardNeeded records that the IME can still hold a composition
	// for an abandoned target. The composition is not discarded immediately:
	// the backend discards it when the next session starts.
	markIMEDiscardNeeded()
}

type textInput struct {
	events textInputEvents
}

var theTextInput textInput

// theTextInputImpl is the platform text-input backend, chosen at build time.
var theTextInputImpl = textInputImpl{events: &theTextInput.events}

// backend returns the text-input backend serving this process: the VM guest
// backend when running as a VM guest, and the platform implementation
// otherwise.
func (t *textInput) backend() textInputBackend {
	if vmguest.IsGuest() {
		return &theVMGuestTextInput
	}
	return &theTextInputImpl
}

// cancelSessionIfNeeded cancels the session owning the platform IME, if any.
func (t *textInput) cancelSessionIfNeeded() {
	if s := t.events.getActiveSession(); s != nil {
		s.Cancel()
	}
}

// abandonTarget drops what the platform still holds for a cancelled session's
// target: the IME's composition, and the queued states. Either would otherwise
// reach the next session.
func (t *textInput) abandonTarget() {
	t.backend().markIMEDiscardNeeded()
	t.events.clearQueue()
}

type textInputEvents struct {
	ch   chan textInputState
	done chan struct{}

	queuedStates []textInputState

	m sync.Mutex

	// activeSession is the public-facing session currently driving these
	// events, or nil. At most one session is active at a time because the
	// OS IME context is global per app. Guarded by activeSessionM (kept
	// separate from m so that platform-side queries can read the active
	// session without contending on the channel-buffer lock).
	activeSession  *session
	activeSessionM sync.Mutex
}

// getActiveSession returns the active session pointer.
func (s *textInputEvents) getActiveSession() *session {
	s.activeSessionM.Lock()
	defer s.activeSessionM.Unlock()
	return s.activeSession
}

// setActiveSession sets the active session pointer.
func (s *textInputEvents) setActiveSession(active *session) {
	s.activeSessionM.Lock()
	defer s.activeSessionM.Unlock()
	s.activeSession = active
}

// clearActiveSessionIf clears the active session pointer only if it currently
// equals active. Used during session teardown to avoid clobbering a pointer
// that has already been replaced by a newer startSession call.
func (s *textInputEvents) clearActiveSessionIf(active *session) {
	s.activeSessionM.Lock()
	defer s.activeSessionM.Unlock()
	if s.activeSession == active {
		s.activeSession = nil
	}
}

func (s *textInputEvents) start() (ch chan textInputState, endFunc func()) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.ch == nil {
		// 10 should be enough for most cases.
		// Typical keyboards can send less than 10 events at the same time.
		s.ch = make(chan textInputState, 10)
		s.done = make(chan struct{})
	}
	s.flushStateQueue()
	return s.ch, s.end
}

// isOpen reports whether text inputting is in progress, including by the
// deprecated Field, which registers no session.
func (s *textInputEvents) isOpen() bool {
	s.m.Lock()
	defer s.m.Unlock()
	return s.ch != nil
}

func (s *textInputEvents) end() {
	s.m.Lock()
	defer s.m.Unlock()

	if s.ch == nil {
		return
	}
	close(s.ch)
	s.ch = nil
	close(s.done)
	s.done = nil
}

func (s *textInputEvents) send(state textInputState) {
	s.m.Lock()
	defer s.m.Unlock()

	if s.ch != nil {
		s.flushStateQueue()
		s.doSend(state)
	} else {
		s.queuedStates = append(s.queuedStates, state)
	}
}

func (s *textInputEvents) doSend(state textInputState) {
	if s.ch == nil {
		panic("textinput: session is not started")
	}
	for {
		select {
		case s.ch <- state:
			return
		default:
			// Ignore the first value.
			select {
			case <-s.ch:
			case <-s.done:
				return
			}
		}
	}
}

// clearQueue clears queued states.
// This should be called when the text field is unfocused
// so that the queued states are not flushed when the next session starts (#3429).
func (s *textInputEvents) clearQueue() {
	s.m.Lock()
	defer s.m.Unlock()
	s.queuedStates = s.queuedStates[:0]
}

func (s *textInputEvents) flushStateQueue() {
	for _, st := range s.queuedStates {
		s.doSend(st)
	}
	s.queuedStates = slices.Delete(s.queuedStates, 0, len(s.queuedStates))
}
