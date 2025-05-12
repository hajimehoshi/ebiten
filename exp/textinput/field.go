// Copyright 2024 The Ebitengine Authors
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
	"image"
	"sync"
)

var (
	theFocusedField  *Field
	theFocusedFieldM sync.Mutex
)

func focusField(f *Field) {
	var origField *Field
	defer func() {
		if origField != nil {
			origField.cleanUp()
		}
	}()

	theFocusedFieldM.Lock()
	defer theFocusedFieldM.Unlock()
	if theFocusedField == f {
		return
	}
	origField = theFocusedField
	theFocusedField = f
}

func blurField(f *Field) {
	var origField *Field
	defer func() {
		if origField != nil {
			origField.cleanUp()
		}
	}()

	theFocusedFieldM.Lock()
	defer theFocusedFieldM.Unlock()
	if theFocusedField != f {
		return
	}
	origField = theFocusedField
	theFocusedField = nil
}

func isFieldFocused(f *Field) bool {
	theFocusedFieldM.Lock()
	defer theFocusedFieldM.Unlock()
	return theFocusedField == f
}

func currentState() (string, int, int, textInputState, bool) {
	theFocusedFieldM.Lock()
	defer theFocusedFieldM.Unlock()
	if theFocusedField == nil {
		return "", 0, 0, textInputState{}, false
	}
	f := theFocusedField
	return f.text, f.selectionStartInBytes, f.selectionEndInBytes, f.state, true
}

// Field is a region accepting text inputting with IME.
//
// Field is not focused by default. You have to call Focus when you start text inputting.
//
// Field is a wrapper of the low-level API like Start.
//
// For an actual usage, see the examples "textinput".
type Field struct {
	text                  string
	selectionStartInBytes int
	selectionEndInBytes   int

	ch    <-chan textInputState
	end   func()
	state textInputState
	err   error
}

// HandleInput updates the field state.
// HandleInput must be called every tick, i.e., every Update, when Field is focused.
// HandleInput takes a position where an IME window is shown if needed.
//
// HandleInput returns whether the text inputting is handled or not.
// If HandleInput returns true, a Field user should not handle further input events.
//
// HandleInput returns an error when handling input causes an error.
//
// Deprecated: use HandleInputWithBounds instead.
func (f *Field) HandleInput(x, y int) (handled bool, err error) {
	return f.HandleInputWithBounds(image.Rect(x, y, x+1, y+1))
}

// HandleInputWithBounds updates the field state.
// HandleInputWithBounds must be called every tick, i.e., every Update, when Field is focused.
// HandleInputWithBounds takes a character bounds, which decides the position where an IME window is shown if needed.
// The bounds width doesn't matter very much as long as it is greater than 0.
// The bounds height should be the text height like a cursor height.
//
// HandleInputWithBounds returns whether the text inputting is handled or not.
// If HandleInputWithBounds returns true, a Field user should not handle further input events.
//
// HandleInputWithBounds returns an error when handling input causes an error.
func (f *Field) HandleInputWithBounds(bounds image.Rectangle) (handled bool, err error) {
	if f.err != nil {
		return false, f.err
	}
	if !f.IsFocused() {
		return false, nil
	}

	// Text inputting can happen multiple times in one tick (1/60[s] by default).
	// Handle all of them.
	for {
		if f.ch == nil {
			// TODO: On iOS Safari, Start doesn't work as expected (#2898).
			// Handle a click event and focus the textarea there.
			f.ch, f.end = start(bounds)
			// Start returns nil for non-supported envrionments, or when unable to start text inputting for some reasons.
			if f.ch == nil {
				return handled, nil
			}
		}

	readchar:
		for {
			select {
			case state, ok := <-f.ch:
				if state.Error != nil {
					f.err = state.Error
					return false, f.err
				}
				if !ok {
					f.ch = nil
					f.end = nil
					f.state = textInputState{}
					break readchar
				}
				if state.Committed && state.Text == "\x7f" {
					// DEL should not modify the text (#3212).
					f.state = textInputState{}
					continue
				}
				handled = true
				if state.Committed {
					f.commit(state)
					continue
				}
				f.state = state
			default:
				break readchar
			}
		}

		if f.ch == nil {
			continue
		}

		break
	}

	return
}

func (f *Field) commit(state textInputState) {
	if !state.Committed {
		panic("textinput: commit must be called with committed state")
	}
	if state.DeleteEndInBytes-state.DeleteStartInBytes > 0 {
		if f.selectionStartInBytes > state.DeleteStartInBytes {
			f.selectionStartInBytes -= state.DeleteEndInBytes - state.DeleteStartInBytes
		}
		if f.selectionEndInBytes > state.DeleteStartInBytes {
			f.selectionEndInBytes -= state.DeleteEndInBytes - state.DeleteStartInBytes
		}
		f.text = f.text[:state.DeleteStartInBytes] + f.text[state.DeleteEndInBytes:]
	}
	f.text = f.text[:f.selectionStartInBytes] + state.Text + f.text[f.selectionEndInBytes:]
	f.selectionStartInBytes += len(state.Text)
	f.selectionEndInBytes = f.selectionStartInBytes
	f.state = textInputState{}
}

// Focus focuses the field.
// A Field has to be focused to start text inputting.
//
// There can be only one Field that is focused at the same time.
// When Focus is called and there is already a focused field, Focus removes the focus of that.
func (f *Field) Focus() {
	focusField(f)
}

// Blur removes the focus from the field.
func (f *Field) Blur() {
	blurField(f)
}

// IsFocused reports whether the field is focused or not.
func (f *Field) IsFocused() bool {
	return isFieldFocused(f)
}

func (f *Field) cleanUp() {
	if f.err != nil {
		return
	}

	// If the text field still has a session, read the last state and process it just in case.
	if f.ch != nil {
		select {
		case state, ok := <-f.ch:
			if state.Error != nil {
				f.err = state.Error
				return
			}
			if ok && state.Committed {
				f.commit(state)
			} else {
				f.state = state
			}
		default:
			break
		}
	}

	if f.end != nil {
		f.end()
		f.ch = nil
		f.end = nil
		f.state = textInputState{}
	}
}

// Selection returns the current selection range in bytes.
func (f *Field) Selection() (startInBytes, endInBytes int) {
	return f.selectionStartInBytes, f.selectionEndInBytes
}

// CompositionSelection returns the current composition selection in bytes if a text is composited.
// If a text is not composited, this returns 0s and false.
// The returned values indicate relative positions in bytes where the current composition text's start is 0.
func (f *Field) CompositionSelection() (startInBytes, endInBytes int, ok bool) {
	if f.IsFocused() && f.state.Text != "" {
		return f.state.CompositionSelectionStartInBytes, f.state.CompositionSelectionEndInBytes, true
	}
	return 0, 0, false
}

// SetSelection sets the selection range.
func (f *Field) SetSelection(startInBytes, endInBytes int) {
	f.cleanUp()
	f.selectionStartInBytes = startInBytes
	f.selectionEndInBytes = endInBytes
}

// Text returns the current text.
// The returned value doesn't include compositing texts.
func (f *Field) Text() string {
	return f.text
}

// TextForRendering returns the text for rendering.
// The returned value includes compositing texts.
func (f *Field) TextForRendering() string {
	if f.IsFocused() && f.state.Text != "" {
		return f.text[:f.selectionStartInBytes] + f.state.Text + f.text[f.selectionEndInBytes:]
	}
	return f.text
}

// UncommittedTextLengthInBytes returns the compositing text length in bytes when the field is focused and the text is editing.
// The uncommitted text range is from the selection start to the selection start + the uncommitted text length.
// UncommittedTextLengthInBytes returns 0 otherwise.
func (f *Field) UncommittedTextLengthInBytes() int {
	if f.IsFocused() {
		return len(f.state.Text)
	}
	return 0
}

// SetTextAndSelection sets the text and the selection range.
func (f *Field) SetTextAndSelection(text string, selectionStartInBytes, selectionEndInBytes int) {
	f.cleanUp()
	f.text = text
	f.selectionStartInBytes = selectionStartInBytes
	f.selectionEndInBytes = selectionEndInBytes
}
