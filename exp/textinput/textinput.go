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
	"unicode/utf16"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// State represents the current state of text inputting.
//
// State is the low-level API. For most use cases, Field is easier to use.
type State struct {
	// Text represents the current inputting text.
	Text string

	// CompositionSelectionStartInBytes represents the start position of the selection in bytes.
	CompositionSelectionStartInBytes int

	// CompositionSelectionStartInBytes represents the end position of the selection in bytes.
	CompositionSelectionEndInBytes int

	// Committed reports whether the current Text is the settled text.
	Committed bool

	// Error is an error that happens during text inputting.
	Error error
}

// Start starts text inputting.
// Start returns a channel to send the state repeatedly, and a function to end the text inputting.
//
// Start is the low-level API. For most use cases, Field is easier to use.
//
// Start returns nil and nil if the current environment doesn't support this package.
func Start(x, y int) (states <-chan State, close func()) {
	cx, cy := ui.Get().LogicalPositionToClientPositionInNativePixels(float64(x), float64(y))
	return theTextInput.Start(int(cx), int(cy))
}

func convertUTF16CountToByteCount(text string, c int) int {
	return len(string(utf16.Decode(utf16.Encode([]rune(text))[:c])))
}

type session struct {
	ch   chan State
	done chan struct{}
}

func newSession() *session {
	return &session{
		ch:   make(chan State, 1),
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

func (s *session) trySend(state State) {
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
