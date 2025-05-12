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
	"image"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// textInputState represents the current state of text inputting.
type textInputState struct {
	// Text represents the current inputting text.
	Text string

	// CompositionSelectionStartInBytes represents the start position of the selection in bytes.
	CompositionSelectionStartInBytes int

	// CompositionSelectionStartInBytes represents the end position of the selection in bytes.
	CompositionSelectionEndInBytes int

	// DeleteStartInBytes represents the start position of the range to be removed in bytes.
	//
	// DeleteStartInBytes is valid only when Committed is true.
	DeleteStartInBytes int

	// DeleteEndInBytes represents the end position of the range to be removed in bytes.
	//
	// DeleteEndInBytes is valid only when Committed is true.
	DeleteEndInBytes int

	// Committed reports whether the current Text is the settled text.
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
	if c == 0 {
		return 0
	}
	var utf16Len int
	for idx, r := range text {
		utf16Len += utf16.RuneLen(r)
		if utf16Len >= c {
			return idx + utf8.RuneLen(r)
		}
	}
	return -1
}

func convertByteCountToUTF16Count(text string, c int) int {
	if c == 0 {
		return 0
	}
	var utf16Len int
	for idx, r := range text {
		utf16Len += utf16.RuneLen(r)
		if idx+utf8.RuneLen(r) >= c {
			return utf16Len
		}
	}
	return -1
}

type session struct {
	ch   chan textInputState
	done chan struct{}
}

func newSession() *session {
	return &session{
		ch:   make(chan textInputState, 1),
		done: make(chan struct{}),
	}
}

func (s *session) end() {
	if s.ch == nil {
		return
	}
	close(s.ch)
	s.ch = nil
	close(s.done)
}

func (s *session) trySend(state textInputState) {
	for {
		select {
		case s.ch <- state:
			return
		default:
			// Only the last value matters.
			select {
			case <-s.ch:
			case <-s.done:
				return
			}
		}
	}
}
