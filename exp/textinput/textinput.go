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
)

// noReplacement is the sentinel value for [textInputState.ReplacementStartInBytes]
// and [textInputState.ReplacementEndInBytes] meaning "no replacement, at
// the caret of the receiving session." [session.Update] resolves it to
// len(textBeforeCaret) on commit.
const noReplacement = -1

// textInputState is the internal record of an IME event flowing from the
// platform layer to the Go side via a channel.
type textInputState struct {
	// Text is the current composition text, or the committed text when Committed is true.
	Text string

	// CompositionSelectionStartInBytes is the start position of the selection
	// within Text, in bytes. Meaningful only when Committed is false.
	CompositionSelectionStartInBytes int

	// CompositionSelectionEndInBytes is the end position of the selection
	// within Text, in bytes. Meaningful only when Committed is false.
	CompositionSelectionEndInBytes int

	// ReplacementStartInBytes is the start position of the byte range that
	// Text replaces, in the joined TextBeforeCaret+TextAfterCaret buffer.
	// Meaningful only when Committed is true. Use [noReplacement] for the
	// "no replacement, at the caret" case.
	ReplacementStartInBytes int

	// ReplacementEndInBytes is the end position of the byte range that Text
	// replaces, in the joined TextBeforeCaret+TextAfterCaret buffer.
	// Meaningful only when Committed is true. Use [noReplacement] for the
	// "no replacement, at the caret" case.
	ReplacementEndInBytes int

	// Committed reports whether Text is the final committed text.
	Committed bool

	// Error is an error that happens during text inputting.
	Error error
}

// start starts text inputting.
// start returns a channel to send the state repeatedly, and a function to end the text inputting.
//
// start returns nil and nil if the current environment doesn't support this package.
func start(bounds image.Rectangle) (states <-chan textInputState, close func()) {
	cMinX, cMinY := ui.Get().LogicalPositionToClientPositionInNativePixels(float64(bounds.Min.X), float64(bounds.Min.Y))
	cMaxX, cMaxY := ui.Get().LogicalPositionToClientPositionInNativePixels(float64(bounds.Max.X), float64(bounds.Max.Y))
	return theTextInput.Start(image.Rect(int(cMinX), int(cMinY), int(cMaxX), int(cMaxY)))
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

type textInput struct {
	textInputImpl
	events textInputEvents
}

var theTextInput textInput

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
