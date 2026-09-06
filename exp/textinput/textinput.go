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
// This package is supported on Windows, macOS, Linux, iOS, Android, other
// UNIX-like systems with X11, and Web browsers so far.
// It also works in a virtualization guest regardless of the operating system,
// with the host serving text inputting (see exp/vmhost).
//
// # Virtual keyboards
//
// On iOS and Android, a virtual keyboard appears during text inputting.
// While the keyboard would cover the caret reported by
// [SessionOptions.CaretBounds], the rendering is shifted so that the caret
// stays visible, and the shift is undone when the keyboard disappears.
// Web browsers scroll the page by themselves instead.
//
// When the user dismisses the virtual keyboard, e.g. with the Back gesture
// on Android, text inputting ends: [Composer.OnEndByUser] is called, and a
// focused [Field] loses its focus.
//
// # Android
//
// The soft-input mode (android:windowSoftInputMode) is left to the app,
// since EbitenView can be embedded in any activity. Every mode works: the
// keyboard's occlusion is measured from the actual geometry, not assumed
// from the mode. adjustNothing is the most predictable, as the caret is
// kept visible by the shift described above without the window being
// resized or panned.
//
// EbitenView contains a hidden text-editing view that takes the focus
// during text inputting. When embedding EbitenView in a custom view
// hierarchy, no ancestor view must prevent its descendants from taking
// the focus.
package textinput

import (
	"fmt"
	"image"
	"slices"
	"sync"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2"
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

	// commitRegular marks a final committed edit whose triggering key press, if
	// any, is consumed by the IME, so no key passes through to the game (a
	// desktop IME commit, a suggestion tap).
	commitRegular

	// commitWithPassthroughKey marks a final committed edit whose triggering key
	// press also passes through to the game. The Composer leaves handled false
	// for it so the caller still acts on that key.
	commitWithPassthroughKey
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

	// ReplacementRelativeToCaret reports whether ReplacementStartInBytes and
	// ReplacementEndInBytes are offsets from the receiving session's caret,
	// which may be negative, rather than positions in its joined buffer.
	// [noReplacement] does not apply then, an offset pair of 0 and 0 meaning
	// the same. Used for an edit diffed against a buffer another session's
	// commit left behind, where only the caret is a common reference.
	ReplacementRelativeToCaret bool

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
	return theTextInput.backend().Start(bounds, textBeforeCaret, textAfterCaret)
}

// caretBoundsInClientNativePixels converts logical caret bounds to the client area's native pixels,
// the coordinates the platform backends hand to the OS IME.
func caretBoundsInClientNativePixels(bounds image.Rectangle) image.Rectangle {
	cMinX, cMinY := ui.Get().LogicalPositionToClientPositionInNativePixels(float64(bounds.Min.X), float64(bounds.Min.Y))
	cMaxX, cMaxY := ui.Get().LogicalPositionToClientPositionInNativePixels(float64(bounds.Max.X), float64(bounds.Max.Y))
	return image.Rect(int(cMinX), int(cMinY), int(cMaxX), int(cMaxY))
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
			panic(fmt.Sprintf("textinput: invalid rune: %c", r))
		}
		utf16Len += l16
		l8 := utf8.RuneLen(r)
		if l8 < 0 {
			panic(fmt.Sprintf("textinput: invalid rune: %c", r))
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
	// Start starts text inputting, with the same contract as [startTextInput].
	// bounds is in logical pixels; a backend converts it to the coordinates it
	// needs (e.g. [caretBoundsInClientNativePixels] for the OS IME).
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

// queueCarryTicks is how long after a session ends the states queued since
// still count as belonging to the next session.
const queueCarryTicks = 2

// withinQueueCarry reports whether a session starting at tick follows one that
// ended at lastEndTick closely enough to take over its queued states.
// lastEndTick is 0 when no session has ended yet.
//
// The platform reports text whether or not a session is open, and a commit ends
// the session, so the text typed while the application starts the next one is
// queued. That text belongs to the next session: typing faster than one
// character per tick would otherwise lose everything after the first. The two
// are told apart by when the last session ended, as an application driving text
// input starts the next session as soon as it observes the commit.
func withinQueueCarry(lastEndTick, tick int64) bool {
	if lastEndTick == 0 {
		return false
	}
	return tick-lastEndTick <= queueCarryTicks
}

// queuedStatesBelong reports whether states reported at queuedTick are for a
// session starting at tick, which is so in two cases: the previous session
// ended within the carry window, or they were reported in the tick the session
// starts in.
//
// The events of a tick are processed before the game updates, so text typed in
// the same tick a session starts in was reported before the session existed.
// It is what the application is starting the session for.
func queuedStatesBelong(lastEndTick, queuedTick, tick int64) bool {
	return withinQueueCarry(lastEndTick, tick) || queuedTick == tick
}

type textInputEvents struct {
	ch   chan textInputState
	done chan struct{}

	queuedStates []textInputState

	// sessionCommitted reports whether a commit has already been delivered to
	// the open session. A session ends at its first commit, so whatever is
	// queued behind that commit is for the next one.
	sessionCommitted bool

	// lastEndTick is the tick the last session ended at, or 0 before the first
	// one ends.
	lastEndTick int64

	// queuedTick is the tick the queued states were reported in.
	queuedTick int64

	// endedByUser reports whether the last ending was the user ending text
	// inputting from the platform side, e.g. by dismissing the virtual
	// keyboard. Taken by the session observing the closed channel.
	endedByUser bool

	// tick overrides the tick source in tests. A nil tick means [ebiten.Tick].
	tick func() int64

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

// currentTick reports the current tick.
func (s *textInputEvents) currentTick() int64 {
	if s.tick != nil {
		return s.tick()
	}
	return ebiten.Tick()
}

func (s *textInputEvents) start() (ch chan textInputState, endFunc func()) {
	s.m.Lock()
	defer s.m.Unlock()

	// States belonging to no session describe a target this one knows nothing
	// about.
	if !queuedStatesBelong(s.lastEndTick, s.queuedTick, s.currentTick()) {
		s.queuedStates = s.queuedStates[:0]
	}

	if s.ch == nil {
		// 10 should be enough for most cases.
		// Typical keyboards can send less than 10 events at the same time.
		s.ch = make(chan textInputState, 10)
		s.done = make(chan struct{})
	}
	s.sessionCommitted = false
	s.endedByUser = false
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
	s.doEnd()
}

// endByUser ends like end, recording that the ending is the user ending text
// inputting from the platform side, e.g. by dismissing the virtual keyboard.
// The record is kept even when the channel is already closed: a platform can
// commit, which ends, and then report the user's ending, and the ending
// belongs to the session that has not drained that commit yet. start clears a
// record no session consumed.
func (s *textInputEvents) endByUser() {
	s.m.Lock()
	defer s.m.Unlock()
	s.endedByUser = true
	s.doEnd()
}

// takeEndedByUser reports whether the last ending was the user's, and clears
// the record.
func (s *textInputEvents) takeEndedByUser() bool {
	s.m.Lock()
	defer s.m.Unlock()
	ended := s.endedByUser
	s.endedByUser = false
	return ended
}

func (s *textInputEvents) doEnd() {
	if s.ch == nil {
		// There is no session to end. A platform that ends unconditionally
		// after every commit reaches this whenever a commit is queued or
		// dropped, and treating those as endings would push the carry deadline
		// forward for as long as the application takes keyboard input.
		return
	}
	s.lastEndTick = s.currentTick()
	close(s.ch)
	s.ch = nil
	close(s.done)
	s.done = nil
}

// send hands state to the open session, or queues it for the next one, and
// reports whether the session took it. A state no session can claim is
// dropped: queueing it would grow the queue for as long as the application
// takes keyboard input without opening a session, and would reach whichever
// session eventually starts.
func (s *textInputEvents) send(state textInputState) bool {
	s.m.Lock()
	defer s.m.Unlock()

	if s.ch == nil {
		// States left from an earlier tick are for a session that never
		// started. Dropping them as the tick turns over bounds what is held to
		// a single tick of typing, however long the application goes without
		// opening a session.
		tick := s.currentTick()
		if !queuedStatesBelong(s.lastEndTick, s.queuedTick, tick) {
			s.queuedStates = s.queuedStates[:0]
		}
		s.queuedTick = tick
	}

	// Queueing first keeps states in the order they were reported, as an
	// earlier one may still be queued behind a commit.
	s.queuedStates = append(s.queuedStates, state)
	s.flushStateQueue()
	return len(s.queuedStates) == 0
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

// flushStateQueue delivers queued states to the open session, stopping at the
// commit that ends it. A session reports at most one commit, so anything
// queued behind that commit stays for the next session rather than being
// delivered to a channel that is about to close.
func (s *textInputEvents) flushStateQueue() {
	var sent int
	for _, st := range s.queuedStates {
		if s.ch == nil || s.sessionCommitted {
			break
		}
		s.doSend(st)
		sent++
		if st.CommitKind.committed() {
			s.sessionCommitted = true
		}
	}
	s.queuedStates = slices.Delete(s.queuedStates, 0, sent)
}
