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
	"image"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/hook"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// The Android text input integration mirrors the browser backend: a hidden
// EditText in EbitenView, seeded with the session's surrounding text, takes
// the focus, and every state it reports is diffed against the session's
// baseline by diffSender.
//
// The EditText is driven through [ui.TextInputDriver], whose calls the view
// posts to the UI thread. States reported for a superseded seeding are told
// apart by the generation number they carry.

func init() {
	t := &theTextInputImpl
	t.sender.events = t.events
	ui.Get().SetTextInputStateCallback(t.onState)
	hook.AppendHookOnBeforeUpdate(func() error {
		t.dismissVirtualKeyboardIfNeeded()
		return nil
	})
}

type textInputImpl struct {
	events *textInputEvents
	sender diffSender

	// gate arbitrates the asynchronous reseeding: the states the platform
	// reports carry back the generation of the seed its buffer was on.
	gate seedGate

	// active reports whether text inputting has been requested and not
	// dismissed. closedTicks counts consecutive ticks without text inputting;
	// see dismissVirtualKeyboardIfNeeded.
	active      bool
	closedTicks int

	// legacyCleared drops the states fired by the legacy path clearing the
	// text buffer.
	legacyCleared bool

	mu sync.Mutex
}

func (t *textInputImpl) markIMEDiscardNeeded() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.gate.abandon()
}

func (t *textInputImpl) Start(bounds image.Rectangle, textBeforeCaret, textAfterCaret string) (<-chan textInputState, func()) {
	bounds = caretBoundsInClientNativePixels(bounds)
	value := textBeforeCaret + textAfterCaret
	caret := max(convertByteCountToUTF16Count(textBeforeCaret, len(textBeforeCaret)), 0)
	generation := t.recordStart(value)

	ch, end := t.events.start()

	if !ui.Get().StartPlatformTextInput(value, caret, caret, bounds, generation) {
		// No driver is registered: the view predates text input support.
		end()
		return nil, nil
	}
	return ch, end
}

// recordStart records value as the next seed and returns its generation. The
// diff baseline is not switched here: until the platform applies the seed, the
// buffer still holds the previous target, whose states keep flowing through
// the sender (see seedGate).
func (t *textInputImpl) recordStart(value string) int {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.active = true
	t.closedTicks = 0
	return t.gate.start(value)
}

// onState handles a state reported by the platform text buffer.
func (t *textInputImpl) onState(text string, selectionStartInUTF16, selectionEndInUTF16, composingStartInUTF16, composingEndInUTF16, generation int) {
	kind := commitRegular
	if composingStartInUTF16 >= 0 && composingEndInUTF16 >= 0 {
		kind = commitNone
	}

	// Evaluated outside t.mu: the focus lock is taken with no other lock held.
	fieldFocused := withFocusedField(func(*Field) {})

	if t.handleState(text, selectionStartInUTF16, selectionEndInUTF16, kind, generation, fieldFocused) {
		ui.Get().StartPlatformTextInput("", 0, 0, image.Rectangle{}, t.currentGeneration())
	}
}

// handleState reports the state and returns whether the platform text buffer
// must be cleared.
func (t *textInputImpl) handleState(text string, selectionStartInUTF16, selectionEndInUTF16 int, kind commitKind, generation int, fieldFocused bool) (clearBuffer bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	resetBaseline, ok := t.gate.admit(generation)
	if !ok {
		return false
	}
	if resetBaseline {
		t.sender.reset(t.gate.pendingValue)
	}

	// The selection does not track the preedit on a virtual keyboard; see
	// compositionSelectionInBytes.
	return handlePlatformState(t.events, &t.sender, &t.legacyCleared, text, selectionStartInUTF16, selectionEndInUTF16, true, kind, fieldFocused)
}

// currentGeneration returns the current seeding generation.
func (t *textInputImpl) currentGeneration() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.gate.generation
}

// shouldDismiss reports whether the caller stopped inputting text and the
// virtual keyboard is to be dismissed.
func (t *textInputImpl) shouldDismiss() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.active {
		t.closedTicks = 0
		return false
	}
	if t.events.isOpen() || t.events.getActiveSession() != nil {
		t.closedTicks = 0
		return false
	}
	t.closedTicks++
	// This runs before Update, where text inputting reopens. Finishing text
	// inputting in one tick and reopening it in the next leaves one tick
	// closed, so two mean the caller stopped.
	if t.closedTicks < 2 {
		return false
	}
	t.closedTicks = 0
	t.active = false
	return true
}

// dismissVirtualKeyboardIfNeeded dismisses the virtual keyboard once the
// caller stops inputting text.
func (t *textInputImpl) dismissVirtualKeyboardIfNeeded() {
	if !t.shouldDismiss() {
		return
	}
	ui.Get().DismissPlatformTextInput()
	// Dismissing fires states carrying the text the buffer still holds. With
	// no session to receive them, they are queued for the next one.
	t.events.clearQueue()
}

// readVirtualKeyboard reports whether a virtual keyboard is shown and the
// client-area region it leaves visible, in native pixels. ok is false when the
// region is unknown.
func readVirtualKeyboard() (visible bool, visibleClientRegion image.Rectangle, ok bool) {
	return ui.Get().VirtualKeyboardState()
}
