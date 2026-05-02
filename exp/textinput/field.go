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
	"io"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/hook"
)

var (
	theFocusedField  *Field
	theFocusedFieldM sync.Mutex
)

// focusField makes f the focused field. Any previously focused field is cleaned up
// and its ChangedAt is bumped to reflect the focus loss.
func focusField(f *Field) {
	var origField *Field
	var focused bool
	// All ChangedAt bumps happen here, outside the mutex.
	defer func() {
		if focused {
			f.bumpChangedAt()
		}
		if origField != nil {
			origField.cleanUp()
			// The previously focused field just lost focus, which is observable via IsFocused.
			origField.bumpChangedAt()
		}
	}()

	theFocusedFieldM.Lock()
	defer theFocusedFieldM.Unlock()
	if theFocusedField == f {
		return
	}
	origField = theFocusedField
	theFocusedField = f
	focused = true
}

// blurField removes the focus from f. If f was focused, its ChangedAt is bumped.
func blurField(f *Field) {
	var origField *Field
	// All ChangedAt bumps happen here, outside the mutex. origField is always f when set.
	defer func() {
		if origField != nil {
			origField.cleanUp()
			origField.bumpChangedAt()
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
// past the call. fn can read from f.pieceTable directly without
// materializing the committed text into a Go string.
func withFocusedField(fn func(f *Field)) bool {
	theFocusedFieldM.Lock()
	defer theFocusedFieldM.Unlock()
	if theFocusedField == nil {
		return false
	}
	fn(theFocusedField)
	return true
}

func init() {
	hook.AppendHookOnBeforeUpdate(func() error {
		theFocusedFieldM.Lock()
		f := theFocusedField
		theFocusedFieldM.Unlock()

		if f == nil {
			return nil
		}

		handled, err := f.handleInput()
		f.handled = handled
		return err
	})
}

// Field is a region accepting text inputting with IME.
//
// Field is not focused by default. You have to call Focus when you start text inputting.
//
// Field is a wrapper of the low-level API like Start.
//
// For an actual usage, see the examples "textinput".
type Field struct {
	pieceTable            pieceTable
	selectionStartInBytes int
	selectionEndInBytes   int

	bounds  image.Rectangle
	handled bool

	ch    <-chan textInputState
	end   func()
	state textInputState
	err   error

	generation int64
	changedAt  time.Time
}

// Generation returns a counter that advances when the field's renderable content changes.
//
// "Renderable content" is the text returned by [Field.WriteTextForRenderingTo]: the
// committed text plus any active IME composition. Selection-only changes and focus
// changes do not advance Generation.
//
// Generation is monotonically non-decreasing and is suitable as a cache key for
// derivations of the renderable content. The zero value indicates that no
// content-changing mutation has occurred.
func (f *Field) Generation() int64 {
	return f.generation
}

// ChangedAt returns the time of the most recent state-changing mutation.
//
// ChangedAt is monotonically non-decreasing and advances on any observable state
// change, including selection-only changes and focus changes. The zero value
// indicates that no mutation has occurred.
//
// Deprecated: Use [Field.Generation] instead. Generation has stricter content-only
// semantics (better suited for cache invalidation) and does not depend on the
// system clock.
func (f *Field) ChangedAt() time.Time {
	return f.changedAt
}

// bumpChangedAt advances changedAt only. Used for state changes that are not content
// changes (focus, selection). changedAt is forced strictly after its previous value
// to keep equality checks meaningful even on platforms with a coarse clock.
func (f *Field) bumpChangedAt() {
	now := time.Now()
	if !now.After(f.changedAt) {
		now = f.changedAt.Add(time.Nanosecond)
	}
	f.changedAt = now
}

// bumpGeneration advances generation, and also bumps changedAt for the deprecated
// [Field.ChangedAt] API. Used for content changes.
func (f *Field) bumpGeneration() {
	f.generation++
	f.bumpChangedAt()
}

// setState assigns s to f.state, bumping generation when the visible composition state changes.
//
// Only the fields observable through the public API (Text and the composition selection)
// are compared. The remaining fields (Committed/Delete*/Error) are transient IME signaling,
// not part of the observable state this function is meant to track.
func (f *Field) setState(s textInputState) {
	changed := f.state.Text != s.Text ||
		f.state.CompositionSelectionStartInBytes != s.CompositionSelectionStartInBytes ||
		f.state.CompositionSelectionEndInBytes != s.CompositionSelectionEndInBytes
	f.state = s
	if changed {
		f.bumpGeneration()
	}
}

// SetBounds sets the bounds used for IME window positioning.
// The bounds indicate the character position (e.g., cursor bounds) where the IME window should appear.
// The bounds width doesn't matter very much as long as it is greater than 0.
// The bounds height should be the text height like a cursor height.
//
// Call SetBounds when the bounds change, such as when the cursor position updates.
// Unlike the deprecated [Field.HandleInputWithBounds], SetBounds does not need to be called every tick.
func (f *Field) SetBounds(bounds image.Rectangle) {
	f.bounds = bounds
}

// Handled reports whether the text inputting was handled in the current tick.
// If Handled returns true, a Field user should not handle further input events.
func (f *Field) Handled() bool {
	return f.handled
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
// Deprecated: use [Field.SetBounds] and [Field.Handled] instead.
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
//
// Deprecated: use [Field.SetBounds] and [Field.Handled] instead.
func (f *Field) HandleInputWithBounds(bounds image.Rectangle) (handled bool, err error) {
	f.bounds = bounds
	f.handled = false
	handled, err = f.handleInput()
	if handled {
		f.handled = true
	}
	return
}

func (f *Field) handleInput() (handled bool, err error) {
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
			f.ch, f.end = start(f.bounds)
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
					f.setState(textInputState{})
					break readchar
				}
				if state.Committed && state.Text == "\x7f" {
					// DEL should not modify the text (#3212).
					f.setState(textInputState{})
					continue
				}
				handled = true
				if state.Committed {
					f.commit(state)
					continue
				}
				f.setState(state)
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
	start := f.pieceTable.updateByIME(state, f.selectionStartInBytes, f.selectionEndInBytes)
	f.selectionStartInBytes = start + len(state.Text)
	f.selectionEndInBytes = f.selectionStartInBytes
	f.state = textInputState{}
	f.bumpGeneration()
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
				f.setState(state)
			}
		default:
			break
		}
	}

	if f.end != nil {
		f.end()
		f.ch = nil
		f.end = nil
		f.setState(textInputState{})
	}

	theTextInput.session.clearQueue()
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
	l := f.pieceTable.Len()
	newStart := min(max(startInBytes, 0), l)
	newEnd := min(max(endInBytes, 0), l)
	if newStart == f.selectionStartInBytes && newEnd == f.selectionEndInBytes {
		return
	}
	f.selectionStartInBytes = newStart
	f.selectionEndInBytes = newEnd
	f.bumpChangedAt()
}

// Text returns the current text.
// The returned value doesn't include compositing texts.
func (f *Field) Text() string {
	var b strings.Builder
	_, _ = f.WriteTextTo(&b)
	return b.String()
}

// TextForRendering returns the text for rendering.
// The returned value includes compositing texts.
func (f *Field) TextForRendering() string {
	var b strings.Builder
	_, _ = f.WriteTextForRenderingTo(&b)
	return b.String()
}

// HasText reports whether the field has any text.
func (f *Field) HasText() bool {
	return f.pieceTable.hasText()
}

// TextLengthInBytes returns the length of the current text in bytes.
func (f *Field) TextLengthInBytes() int {
	return f.pieceTable.Len()
}

// WriteTextTo writes the current text to w.
// The written text doesn't include compositing texts.
//
// The return value n is the number of bytes written.
// Any error encountered during the write is also returned.
func (f *Field) WriteTextTo(w io.Writer) (int64, error) {
	return f.pieceTable.WriteTo(w)
}

// WriteText writes the current text to w.
//
// Deprecated: use [Field.WriteTextTo] instead.
func (f *Field) WriteText(w io.Writer) error {
	_, err := f.WriteTextTo(w)
	return err
}

// WriteTextRangeTo writes the bytes of the current text in [startInBytes, endInBytes) to w.
// startInBytes and endInBytes are clamped to [0, TextLengthInBytes()].
// If the clamped start is not less than the clamped end, nothing is written.
// The written text doesn't include compositing texts.
//
// The return value n is the number of bytes written.
// Any error encountered during the write is also returned.
func (f *Field) WriteTextRangeTo(w io.Writer, startInBytes, endInBytes int) (int64, error) {
	return f.pieceTable.writeRangeTo(w, startInBytes, endInBytes)
}

// WriteTextRange writes the bytes of the current text in [startInBytes, endInBytes) to w.
//
// Deprecated: use [Field.WriteTextRangeTo] instead.
func (f *Field) WriteTextRange(w io.Writer, startInBytes, endInBytes int) error {
	_, err := f.WriteTextRangeTo(w, startInBytes, endInBytes)
	return err
}

// WriteTextForRenderingTo writes the text for rendering to w.
// The written text includes compositing texts.
//
// The return value n is the number of bytes written.
// Any error encountered during the write is also returned.
func (f *Field) WriteTextForRenderingTo(w io.Writer) (int64, error) {
	if f.IsFocused() && f.state.Text != "" {
		return f.pieceTable.writeToWithInsertion(w, f.state.Text, f.selectionStartInBytes, f.selectionEndInBytes)
	}
	return f.pieceTable.WriteTo(w)
}

// WriteTextForRendering writes the text for rendering to w.
//
// Deprecated: use [Field.WriteTextForRenderingTo] instead.
func (f *Field) WriteTextForRendering(w io.Writer) error {
	_, err := f.WriteTextForRenderingTo(w)
	return err
}

// WriteTextForRenderingRangeTo writes the bytes of the rendering text in [startInBytes, endInBytes) to w.
// Coordinates are in rendering space: the rendering text is the committed text with the active
// composition (if any) spliced in at the selection.
//
// startInBytes and endInBytes are clamped to [0, renderingLength], where renderingLength is
// TextLengthInBytes() - (selectionEnd - selectionStart) + UncommittedTextLengthInBytes().
// If the clamped start is not less than the clamped end, nothing is written.
//
// The return value n is the number of bytes written.
// Any error encountered during the write is also returned.
func (f *Field) WriteTextForRenderingRangeTo(w io.Writer, startInBytes, endInBytes int) (int64, error) {
	if f.IsFocused() && f.state.Text != "" {
		return f.pieceTable.writeRangeToWithInsertion(w, f.state.Text, f.selectionStartInBytes, f.selectionEndInBytes, startInBytes, endInBytes)
	}
	return f.pieceTable.writeRangeTo(w, startInBytes, endInBytes)
}

// WriteTextForRenderingRange writes the bytes of the rendering text in [startInBytes, endInBytes) to w.
//
// Deprecated: use [Field.WriteTextForRenderingRangeTo] instead.
func (f *Field) WriteTextForRenderingRange(w io.Writer, startInBytes, endInBytes int) error {
	_, err := f.WriteTextForRenderingRangeTo(w, startInBytes, endInBytes)
	return err
}

// ResetText resets the text.
// ResetText clears the undo history and initializes it with the specified text.
func (f *Field) ResetText(text string) {
	f.cleanUp()
	f.pieceTable.reset(text)
	f.selectionStartInBytes = 0
	f.selectionEndInBytes = 0
	f.bumpGeneration()
}

// ReadTextFrom resets the text by reading bytes from r until EOF.
// ReadTextFrom clears the undo history and initializes it with the read text.
//
// The return value n is the number of bytes read.
// If r returns a non-EOF error, the field's text is reset to empty and the error is returned.
func (f *Field) ReadTextFrom(r io.Reader) (int64, error) {
	f.cleanUp()
	n, err := f.pieceTable.readFrom(r)
	f.selectionStartInBytes = 0
	f.selectionEndInBytes = 0
	f.bumpGeneration()
	return n, err
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
// This operation is added to the undo history.
func (f *Field) SetTextAndSelection(text string, selectionStartInBytes, selectionEndInBytes int) {
	f.cleanUp()
	l := f.pieceTable.Len()
	f.pieceTable.replace(text, 0, l)
	f.selectionStartInBytes = min(max(selectionStartInBytes, 0), l)
	f.selectionEndInBytes = min(max(selectionEndInBytes, 0), l)
	f.bumpGeneration()
}

// ReplaceText replaces the text at the specified range and updates the selection range.
// This operation is added to the undo history.
func (f *Field) ReplaceText(text string, startInBytes, endInBytes int) {
	f.cleanUp()
	if text == "" && startInBytes == endInBytes {
		// Empty replacement over a zero-width range is observably a no-op.
		return
	}
	f.pieceTable.replace(text, startInBytes, endInBytes)
	f.selectionStartInBytes = startInBytes + len(text)
	f.selectionEndInBytes = f.selectionStartInBytes
	f.bumpGeneration()
}

// ReplaceTextAtSelection replaces the text at the selection range and updates the selection range.
// This operation is added to the undo history.
func (f *Field) ReplaceTextAtSelection(text string) {
	f.ReplaceText(text, f.selectionStartInBytes, f.selectionEndInBytes)
}

// CanUndo reports whether the field can undo or not.
func (f *Field) CanUndo() bool {
	return f.pieceTable.canUndo()
}

// CanRedo reports whether the field can redo or not.
func (f *Field) CanRedo() bool {
	return f.pieceTable.canRedo()
}

// Undo undoes the last operation.
//
// History granularity may vary depending on the internal implementation. Do not write code that depends on the granularity.
func (f *Field) Undo() {
	start, end, ok := f.pieceTable.undo()
	if !ok {
		return
	}
	f.selectionStartInBytes = start
	f.selectionEndInBytes = end
	f.bumpGeneration()
}

// Redo redoes the last undone operation.
//
// History granularity may vary depending on the internal implementation. Do not write code that depends on the granularity.
func (f *Field) Redo() {
	start, end, ok := f.pieceTable.redo()
	if !ok {
		return
	}
	f.selectionStartInBytes = start
	f.selectionEndInBytes = end
	f.bumpGeneration()
}
