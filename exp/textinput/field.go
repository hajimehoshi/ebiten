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

// withFocusedField runs fn under the focus lock with the focused field, and
// reports whether a field was focused. Callers must not retain the *Field
// past the call.
func withFocusedField(fn func(f *Field)) bool {
	theFocusedFieldM.Lock()
	defer theFocusedFieldM.Unlock()
	if theFocusedField == nil {
		return false
	}
	fn(theFocusedField)
	return true
}

// Field is a region accepting text inputting with IME.
//
// Field is not focused by default. You have to call Focus when you start text inputting.
//
// Deprecated: use [Composer] instead.
type Field struct {
	// mu guards text, selectionStartInBytes, selectionEndInBytes and state,
	// which the platform's IME callbacks read on the platform thread while the
	// game goroutine updates them.
	//
	// The lock order is theFocusedFieldM then mu: a goroutine holding mu must
	// not take theFocusedFieldM.
	mu                    sync.Mutex
	text                  string
	selectionStartInBytes int
	selectionEndInBytes   int
	state                 textInputState

	ch  <-chan textInputState
	end func()
	err error
}

func (f *Field) loadState() textInputState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state
}

func (f *Field) storeState(state textInputState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state = state
}

func (f *Field) setSelection(startInBytes, endInBytes int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.selectionStartInBytes = min(max(startInBytes, 0), len(f.text))
	f.selectionEndInBytes = min(max(endInBytes, 0), len(f.text))
}

func (f *Field) setTextAndSelection(text string, selectionStartInBytes, selectionEndInBytes int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.text = text
	f.selectionStartInBytes = min(max(selectionStartInBytes, 0), len(f.text))
	f.selectionEndInBytes = min(max(selectionEndInBytes, 0), len(f.text))
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
// Deprecated: use [Field.HandleInputWithBounds] instead.
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

	reportVirtualKeyboardToUI(bounds)

	// Text inputting can happen multiple times in one tick (1/60[s] by default).
	// Handle all of them.
	var endedByUser bool
	for {
		if f.ch == nil {
			// TODO: On iOS Safari, Start doesn't work as expected (#2898).
			// Handle a click event and focus the textarea there.
			f.ch, f.end = startTextInput(bounds, "", "")
			// startTextInput returns nil for non-supported environments, or when unable to start text inputting for some reasons.
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
					if theTextInput.events.takeEndedByUser() {
						// Keep f.state: the pending composition is committed
						// below.
						endedByUser = true
					} else {
						f.storeState(textInputState{})
					}
					break readchar
				}
				if state.CommitKind.committed() && state.Text == "\x7f" {
					// DEL should not modify the text (#3212).
					f.storeState(textInputState{})
					continue
				}
				handled = true
				if state.CommitKind.committed() {
					f.commit(state)
					continue
				}
				f.storeState(state)
			default:
				break readchar
			}
		}

		if endedByUser {
			// The user ended text inputting, e.g. by dismissing the virtual
			// keyboard. Commit what the user last saw, and unfocus instead of
			// restarting: another session would show the virtual keyboard
			// again.
			if state := f.loadState(); state.Text != "" {
				f.commit(textInputState{
					Text:                    state.Text,
					ReplacementStartInBytes: noReplacement,
					ReplacementEndInBytes:   noReplacement,
					CommitKind:              commitRegular,
				})
			}
			f.storeState(textInputState{})
			f.Blur()
			return handled, nil
		}

		if f.ch == nil {
			continue
		}

		break
	}

	return
}

func (f *Field) commit(state textInputState) {
	if !state.CommitKind.committed() {
		panic("textinput: commit must be called with committed state")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	replStart, replEnd := state.ReplacementStartInBytes, state.ReplacementEndInBytes
	if state.ReplacementRelativeToCaret {
		// A state queued for a session this field took over instead carries
		// offsets from that session's caret, which is this field's selection.
		replStart = min(max(f.selectionStartInBytes+replStart, 0), len(f.text))
		replEnd = min(max(f.selectionStartInBytes+replEnd, replStart), len(f.text))
	}
	if replEnd-replStart > 0 {
		if f.selectionStartInBytes > replStart {
			f.selectionStartInBytes -= replEnd - replStart
		}
		if f.selectionEndInBytes > replStart {
			f.selectionEndInBytes -= replEnd - replStart
		}
		f.text = f.text[:replStart] + f.text[replEnd:]
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
	clearVirtualKeyboardFromUI()

	// If the text field still has a session without a recorded error, read
	// the last state and process it just in case.
	if f.err == nil && f.ch != nil {
		select {
		case state, ok := <-f.ch:
			if state.Error != nil {
				f.err = state.Error
			} else if ok && state.CommitKind.committed() {
				f.commit(state)
			} else {
				f.storeState(state)
			}
		default:
			break
		}
	}

	// The platform holds the session until its end callback runs, even after
	// the session reported an error, so end the session here.
	if f.end != nil {
		f.end()
		f.ch = nil
		f.end = nil
		f.storeState(textInputState{})
	}

	theTextInput.events.clearQueue()
}

// Selection returns the current selection range in bytes.
func (f *Field) Selection() (startInBytes, endInBytes int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.selectionStartInBytes, f.selectionEndInBytes
}

// CompositionSelection returns the current composition selection in bytes if a text is composited.
// If a text is not composited, this returns 0s and false.
// The returned values indicate relative positions in bytes where the current composition text's start is 0.
func (f *Field) CompositionSelection() (startInBytes, endInBytes int, ok bool) {
	if !f.IsFocused() {
		return 0, 0, false
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.state.Text == "" {
		return 0, 0, false
	}
	return f.state.CompositionSelectionStartInBytes, f.state.CompositionSelectionEndInBytes, true
}

// SetSelection sets the selection range.
func (f *Field) SetSelection(startInBytes, endInBytes int) {
	f.cleanUp()
	f.setSelection(startInBytes, endInBytes)
}

// Text returns the current text.
// The returned value doesn't include compositing texts.
func (f *Field) Text() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.text
}

// TextForRendering returns the text for rendering.
// The returned value includes compositing texts.
func (f *Field) TextForRendering() string {
	focused := f.IsFocused()

	f.mu.Lock()
	defer f.mu.Unlock()
	if focused && f.state.Text != "" {
		return f.text[:f.selectionStartInBytes] + f.state.Text + f.text[f.selectionEndInBytes:]
	}
	return f.text
}

// UncommittedTextLengthInBytes returns the compositing text length in bytes when the field is focused and the text is editing.
// The uncommitted text range is from the selection start to the selection start + the uncommitted text length.
// UncommittedTextLengthInBytes returns 0 otherwise.
func (f *Field) UncommittedTextLengthInBytes() int {
	if !f.IsFocused() {
		return 0
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.state.Text)
}

// SetTextAndSelection sets the text and the selection range.
func (f *Field) SetTextAndSelection(text string, selectionStartInBytes, selectionEndInBytes int) {
	f.cleanUp()
	f.setTextAndSelection(text, selectionStartInBytes, selectionEndInBytes)
}
