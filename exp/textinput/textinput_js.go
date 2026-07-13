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

package textinput

import (
	"fmt"
	"image"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2/internal/hook"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

var (
	document = js.Global().Get("document")
	body     = document.Get("body")
)

func init() {
	if !document.Truthy() {
		return
	}
	theTextInputImpl.init()
}

type textInputImpl struct {
	events *textInputEvents

	textareaElement js.Value

	// dummyTextareaElement parks the DOM focus and handles no input.
	// See discardIMEState.
	dummyTextareaElement js.Value

	// lastSentValue/lastSentCommitted: the last state sent in the session path,
	// used to drop trailing duplicate events (e.g. the input after compositionend).
	lastSentValue     string
	lastSentCommitted bool

	// committedThisSession reports whether this session already committed; a
	// second commit before the next session starts falls back to a caret insert
	// (see trySend).
	committedThisSession bool

	// composing reports whether the IME holds a composition for the textarea. iOS
	// Safari reports one through the input event's inputType only.
	composing bool

	// closedTicks counts consecutive ticks without text inputting.
	// See dismissVirtualKeyboardIfNeeded.
	closedTicks int

	// imeDiscard tracks the IME's composition for an abandoned target.
	imeDiscard imeDiscardState
}

// imeDiscardState is the state of the IME's composition for an abandoned target.
type imeDiscardState int

const (
	// imeDiscardNone marks that the IME holds no composition for an abandoned target.
	imeDiscardNone imeDiscardState = iota

	// imeDiscardNeeded marks that the IME can still hold one. Start discards it, as
	// the next session can start before the abandoned one ends (#3463).
	imeDiscardNeeded

	// imeDiscarding marks that discardIMEState is moving the DOM focus. The events
	// the move fires are the IME's teardown, not the user's input.
	imeDiscarding
)

// markIMEDiscardNeeded records that the IME can still hold a composition for an
// abandoned target.
func (t *textInputImpl) markIMEDiscardNeeded() {
	t.imeDiscard = imeDiscardNeeded
}

// needsIMEDiscard reports whether the IME can still hold a composition for an
// abandoned target.
func (t *textInputImpl) needsIMEDiscard() bool {
	return t.imeDiscard == imeDiscardNeeded
}

// markIMEDiscarded records that the IME no longer holds one.
func (t *textInputImpl) markIMEDiscarded() {
	t.imeDiscard = imeDiscardNone
}

func newTextareaElement(id string) js.Value {
	e := document.Call("createElement", "textarea")
	e.Set("id", id)
	e.Set("autocapitalize", "off")
	e.Set("spellcheck", false)
	e.Set("translate", "no")
	e.Set("wrap", "off")

	style := e.Get("style")
	style.Set("position", "absolute")
	style.Set("left", "0")
	style.Set("top", "0")
	style.Set("opacity", "0")
	style.Set("resize", "none")
	style.Set("cursor", "normal")
	style.Set("pointerEvents", "none")
	style.Set("overflow", "hidden")
	style.Set("tabindex", "-1")
	style.Set("width", "1px")
	style.Set("height", "1px")
	return e
}

func (t *textInputImpl) init() {
	hook.AppendHookOnBeforeUpdate(func() error {
		t.dismissVirtualKeyboardIfNeeded()
		return nil
	})

	t.textareaElement = newTextareaElement("ebitengine-textinput")
	// The dummy must be editable too, or focusing it would dismiss a virtual keyboard.
	t.dummyTextareaElement = newTextareaElement("ebitengine-textinput-dummy")

	t.textareaElement.Call("addEventListener", "compositionstart", js.FuncOf(func(this js.Value, args []js.Value) any {
		t.composing = true
		return nil
	}))
	t.textareaElement.Call("addEventListener", "compositionend", js.FuncOf(func(this js.Value, args []js.Value) any {
		t.composing = false
		t.trySend(commitRegular)
		return nil
	}))
	t.textareaElement.Call("addEventListener", "focusout", js.FuncOf(func(this js.Value, args []js.Value) any {
		if t.imeDiscard == imeDiscarding {
			return nil
		}
		t.events.end()
		return nil
	}))
	t.textareaElement.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		if e.Get("code").String() == "Tab" {
			e.Call("preventDefault")
		}
		isEnter := e.Get("code").String() == "Enter" || e.Get("key").String() == "Enter"
		if isVirtualKeyboard() && isEnter {
			// On a virtual keyboard, Return commits the text and is forwarded as
			// a KeyEnter press the game acts on (e.g. a newline). preventDefault
			// suppresses the textarea's own literal newline.
			e.Call("preventDefault")
			ui.Get().UpdateInputFromEvent(e)
			t.trySend(commitWithPassthroughKey)
			return nil
		}
		if isVirtualKeyboard() && (e.Get("code").String() == "Backspace" || e.Get("key").String() == "Backspace") {
			// The textarea's own deletion fires deleteContentBackward, which the
			// IME path applies; forwarding a raw KeyBackspace too would delete
			// twice, as iOS Safari splits keydown and input across a tick. At the
			// textarea head nothing is deleted and no input fires, so forward the
			// raw key to delete across the line boundary.
			start := t.textareaElement.Get("selectionStart").Int()
			end := t.textareaElement.Get("selectionEnd").Int()
			if start != 0 || end != 0 {
				return nil
			}
		}
		if !e.Get("isComposing").Bool() {
			if isEnter {
				// preventDefault suppresses the textarea's own literal newline; the
				// game acts on the forwarded KeyEnter press instead.
				e.Call("preventDefault")
			}
			ui.Get().UpdateInputFromEvent(e)
		}
		return nil
	}))
	t.textareaElement.Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		// Call UpdateInputFromEvent even if isComposing is true, in order to make sure the key is released (#3328).
		// When an IME starts, the events can be fired in this order:
		//
		//   1. `keydown` code=KeyA isComposing=false
		//   2. `compositionstart`
		//   3. `keyup` code=KeyA isComposing=true
		//
		// and if `keyup` is ignored, the key A is considered to be pressed forever.
		ui.Get().UpdateInputFromEvent(e)
		return nil
	}))
	t.textareaElement.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		if e.Get("inputType").String() == "insertCompositionText" {
			t.composing = true
		}
		// On iOS Safari, `isComposing` can be undefined.
		if e.Get("isComposing").IsUndefined() {
			t.trySend(commitNone)
			return nil
		}
		if e.Get("isComposing").Bool() {
			t.trySend(commitNone)
			return nil
		}
		if e.Get("inputType").String() == "insertLineBreak" {
			t.trySend(commitRegular)
			return nil
		}
		if e.Get("inputType").String() == "insertText" && e.Get("data").Equal(js.Null()) {
			// When a new line is inserted, the 'data' property might be null.
			t.trySend(commitRegular)
			return nil
		}
		// Though `isComposing` is false, send the text as being not committed for text completion with a virtual keyboard.
		if isVirtualKeyboard() {
			t.trySend(commitNone)
			return nil
		}
		t.trySend(commitRegular)
		return nil
	}))
	t.textareaElement.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) any {
		t.trySend(commitRegular)
		return nil
	}))
	body.Call("appendChild", t.textareaElement)
	body.Call("appendChild", t.dummyTextareaElement)
	ui.Get().SetTextInputFocusedFunc(t.textInputElementFocused)

	js.Global().Call("eval", `
// Process the textarea element under user-interaction events.
// This is due to an iOS Safari restriction (#2898).
let handler = (e) => {
	if (window._ebitengine_textinput_x === undefined || window._ebitengine_textinput_y === undefined) {
		return;
	}
	let textarea = document.getElementById("ebitengine-textinput");
	textarea.value = '';
	textarea.focus();
	textarea.style.left = _ebitengine_textinput_x + 'px';
	textarea.style.top = _ebitengine_textinput_y + 'px';
	window._ebitengine_textinput_x = undefined;
	window._ebitengine_textinput_y = undefined;
	window._ebitengine_textinput_ready = true;
};

let body = window.document.body;
body.addEventListener("mouseup", handler);
body.addEventListener("touchend", handler);
body.addEventListener("keyup", handler);`)

	// TODO: What about other events like wheel?
}

func (t *textInputImpl) Start(bounds image.Rectangle, textBeforeCaret, textAfterCaret string) (<-chan textInputState, func()) {
	if !t.textareaElement.Truthy() {
		return nil, nil
	}

	bounds = caretBoundsInClientNativePixels(bounds)

	// A start deferred by deferStart has completed: the user-interaction handler
	// installed in init focused the textarea and reset its value.
	if js.Global().Get("_ebitengine_textinput_ready").Truthy() {
		t.events.end()
		// An IME carries a composition over to the refocused textarea and finishes it
		// there. Those events describe the abandoned target, and start replays them.
		t.events.clearQueue()
		ch, end := t.events.start()
		js.Global().Get("window").Set("_ebitengine_textinput_ready", js.Undefined())
		// Focusing the textarea has restarted the IME.
		t.markIMEDiscarded()
		t.setSurroundingTextToTextarea(textBeforeCaret, textAfterCaret)
		return ch, end
	}

	// If a textarea is focused, create a session immediately.
	// A virtual keyboard should already be shown on mobile browsers.
	if t.needsIMEDiscard() {
		// A composition is the only IME state that outlives its target, and discarding
		// it costs the textarea its input session.
		if t.composing && document.Get("activeElement").Equal(t.textareaElement) {
			// The focus lands on the dummy, so the deferred branch below runs.
			t.discardIMEState()
		} else {
			t.markIMEDiscarded()
		}
	}

	if document.Get("activeElement").Equal(t.textareaElement) {
		t.setSurroundingTextToTextarea(textBeforeCaret, textAfterCaret)
		t.textareaElement.Call("focus")
		style := t.textareaElement.Get("style")
		style.Set("left", fmt.Sprintf("%dpx", bounds.Min.X))
		style.Set("top", fmt.Sprintf("%dpx", bounds.Min.Y))
		style.Set("width", fmt.Sprintf("%dpx", bounds.Dx()))
		style.Set("height", fmt.Sprintf("%dpx", bounds.Dy()))
		style.Set("font-size", fmt.Sprintf("%dpx", bounds.Dy()))
		style.Set("line-height", fmt.Sprintf("%dpx", bounds.Dy()))

		t.events.start()
		return t.events.ch, func() {
			t.events.end()
			// Reset the session explictly, or a new session cannot be created above.
		}
	}

	t.deferStart(bounds)
	return nil, nil
}

// deferStart puts off starting a session until the next user-interaction event,
// where the textarea can take the focus and get an input session (#2898).
func (t *textInputImpl) deferStart(bounds image.Rectangle) {
	t.events.end()
	js.Global().Get("window").Set("_ebitengine_textinput_x", bounds.Min.X)
	js.Global().Get("window").Set("_ebitengine_textinput_y", bounds.Max.Y)
}

// discardIMEState makes the IME let go of the composition it holds for the
// textarea, by parking the DOM focus on the dummy textarea (#3463). The focus is
// left there, and the caller must defer starting a session (#2898).
//
// The textarea must have the DOM focus. The IME discard state is none on return.
func (t *textInputImpl) discardIMEState() {
	t.imeDiscard = imeDiscarding
	defer func() {
		t.imeDiscard = imeDiscardNone
		t.composing = false
	}()
	t.dummyTextareaElement.Call("focus")
}

// textInputElementFocused reports whether the DOM focus is on either textarea.
// It rests on the dummy while a start is deferred (see discardIMEState).
func (t *textInputImpl) textInputElementFocused() bool {
	active := document.Get("activeElement")
	return active.Equal(t.textareaElement) || active.Equal(t.dummyTextareaElement)
}

// dismissVirtualKeyboardIfNeeded moves the DOM focus from the textarea to the
// canvas once the caller stops inputting text, dismissing a virtual keyboard.
func (t *textInputImpl) dismissVirtualKeyboardIfNeeded() {
	if !t.textInputElementFocused() {
		t.closedTicks = 0
		return
	}

	// A start deferred to a user-interaction event is pending (#2898). The caret
	// position can be 0, so test for the property's presence.
	if !js.Global().Get("_ebitengine_textinput_x").IsUndefined() || js.Global().Get("_ebitengine_textinput_ready").Truthy() {
		t.closedTicks = 0
		return
	}

	// An active session outlives the event channel, which a commit closes from a DOM
	// event. The deprecated Field keeps the channel open, registering no session.
	if t.events.isOpen() || t.events.getActiveSession() != nil {
		t.closedTicks = 0
		return
	}

	t.closedTicks++
	// This runs before Update, where text inputting reopens. Finishing text inputting
	// in one tick and reopening it in the next leaves one tick closed, so two mean
	// the caller stopped.
	if t.closedTicks < 2 {
		return
	}
	t.closedTicks = 0
	// The game's key listeners are on the canvas element, so it must take the focus.
	ui.Get().FocusCanvas()

	// Blurring the textarea fires events carrying the text it still holds. With no
	// session to receive them, they are queued for the next one.
	t.events.clearQueue()
}

// setSurroundingTextToTextarea writes the text around the caret so later edits
// can be diffed against it, and puts the textarea's caret where the caller's is.
func (t *textInputImpl) setSurroundingTextToTextarea(textBeforeCaret, textAfterCaret string) {
	value := textBeforeCaret + textAfterCaret
	// On a virtual keyboard, iOS Safari keeps committed IME text marked as a
	// composition; assigning value drops that state so a backspace deletes one
	// character, not the whole run. Force the write even when unchanged. Desktop
	// keeps the skip to avoid dismissing an in-flight accent popup (#3236).
	if t.textareaElement.Get("value").String() != value || isVirtualKeyboard() {
		t.textareaElement.Set("value", "")
		t.textareaElement.Set("value", value)
	}
	// The caller can move its caret without editing the text. Skip an unchanged
	// selection, which an in-flight accent popup does not survive (#3236).
	caret := max(convertByteCountToUTF16Count(textBeforeCaret, len(textBeforeCaret)), 0)
	if t.textareaElement.Get("selectionStart").Int() != caret || t.textareaElement.Get("selectionEnd").Int() != caret {
		t.textareaElement.Call("setSelectionRange", caret, caret)
	}
	t.lastSentValue = value
	t.lastSentCommitted = true
	t.committedThisSession = false
}

// compositionSelectionInBytes returns the IME's selection within the preedit,
// as a byte range relative to the preedit's start. The preedit occupies
// preeditLen bytes of the textarea's value from preeditStart.
func (t *textInputImpl) compositionSelectionInBytes(value string, preeditStart, preeditLen int) (start, end int) {
	if isVirtualKeyboard() {
		// A virtual keyboard offers no way to move the caret inside a preedit, so the
		// caret is at its end. iOS Safari reports the selection from before the preedit
		// was inserted, which would put the caret at the preedit's head.
		return preeditLen, preeditLen
	}
	startInBytes := convertUTF16CountToByteCount(value, t.textareaElement.Get("selectionStart").Int())
	endInBytes := convertUTF16CountToByteCount(value, t.textareaElement.Get("selectionEnd").Int())
	return min(max(startInBytes-preeditStart, 0), preeditLen), min(max(endInBytes-preeditStart, 0), preeditLen)
}

func (t *textInputImpl) trySend(kind commitKind) {

	if t.imeDiscard == imeDiscarding {
		// Moving the focus off the textarea ends a composition, firing events for the
		// target the caller has already taken over.
		return
	}

	s := t.events.getActiveSession()
	if s == nil {
		// No session means the deprecated Field is inputting; it uses the legacy
		// whole-value path.
		// TODO: Remove trySendLegacy and this branch once Field is gone; a
		// Composer session is always active otherwise.
		t.trySendLegacy(kind)
		return
	}

	// Diff the textarea against the session's surrounding text and send the edit
	// as a replacement range (needed for the accent popup, #3236). The textarea
	// is not cleared on commit, so the popup can replace the just-typed character.
	value := t.textareaElement.Get("value").String()
	if value == t.lastSentValue && kind.committed() == t.lastSentCommitted {
		return
	}

	if t.committedThisSession {
		// Already committed this session; the buffer has moved past our baseline.
		// Send just the new delta as a caret insert (rapid double-commit case).
		text, _, _ := computeReplacement(t.lastSentValue, value, -1)
		t.events.send(textInputState{
			Text:                    text,
			ReplacementStartInBytes: noReplacement,
			ReplacementEndInBytes:   noReplacement,
			CommitKind:              kind,
		})
		t.lastSentValue = value
		t.lastSentCommitted = kind.committed()
		if kind.committed() {
			t.events.end()
		}
		return
	}

	baseline := s.textBeforeCaret + s.textAfterCaret

	if !kind.committed() {
		// Composition: the preedit ends where the unchanged after-caret text
		// begins. Anchor there rather than at the caret, which may sit inside
		// the preedit during conversion, so the preedit is located correctly
		// even when the surrounding text repeats. Report the selection relative
		// to the preedit start.
		preeditEnd := len(value) - len(s.textAfterCaret)
		text, replStartInBytes, replEndInBytes := computeReplacement(baseline, value, preeditEnd)
		// A composition can only insert a preedit at the caret. When the edit
		// removes committed bytes (replStartInBytes < replEndInBytes) — e.g. a
		// backspace on a virtual keyboard — a preedit cannot express it, so
		// commit the replacement instead.
		if replStartInBytes >= replEndInBytes {
			selStartInBytes, selEndInBytes := t.compositionSelectionInBytes(value, replStartInBytes, len(text))
			t.events.send(textInputState{
				Text:                             text,
				CompositionSelectionStartInBytes: selStartInBytes,
				CompositionSelectionEndInBytes:   selEndInBytes,
				ReplacementStartInBytes:          noReplacement,
				ReplacementEndInBytes:            noReplacement,
				CommitKind:                       kind,
			})
			t.lastSentValue = value
			t.lastSentCommitted = false
			return
		}
		// A virtual-keyboard deletion removes committed bytes; deliver it as a
		// commit whose key does not pass through to the game.
		kind = commitRegular
	}

	// The caret is at the end of the committed text; anchor on it so an
	// insertion into repeated surrounding text is not misplaced at the end.
	caret := t.textareaElement.Get("selectionStart").Int()
	caretInBytes := convertUTF16CountToByteCount(value, caret)
	text, replStartInBytes, replEndInBytes := computeReplacement(baseline, value, caretInBytes)

	t.events.send(textInputState{
		Text:                    text,
		ReplacementStartInBytes: replStartInBytes,
		ReplacementEndInBytes:   replEndInBytes,
		CommitKind:              kind,
	})
	t.lastSentValue = value
	t.lastSentCommitted = true
	t.committedThisSession = true
	t.events.end()
}

func (t *textInputImpl) trySendLegacy(kind commitKind) {
	textareaValue := t.textareaElement.Get("value").String()
	// textareaValue can be an empty value, but this should be sent especially for a compositing text (#3324).

	start := t.textareaElement.Get("selectionStart").Int()
	end := t.textareaElement.Get("selectionEnd").Int()
	startInBytes := convertUTF16CountToByteCount(textareaValue, start)
	endInBytes := convertUTF16CountToByteCount(textareaValue, end)

	t.events.send(textInputState{
		Text:                             textareaValue,
		CompositionSelectionStartInBytes: startInBytes,
		CompositionSelectionEndInBytes:   endInBytes,
		ReplacementStartInBytes:          noReplacement,
		ReplacementEndInBytes:            noReplacement,
		CommitKind:                       kind,
	})

	if kind.committed() {
		t.events.end()
		t.textareaElement.Set("value", "")
	}
}

func isVirtualKeyboard() bool {
	// Assume that a device whose primary pointer is coarse (a touchscreen)
	// uses a software keyboard. This cannot detect an attached hardware
	// keyboard, and reports the device capability rather than whether a
	// virtual keyboard is currently shown.
	//
	// TODO: Use the `navigator.virtualKeyboard` API once it is widely
	// supported to detect the actual virtual keyboard state.
	// https://developer.mozilla.org/en-US/docs/Web/API/Navigator/virtualKeyboard
	return js.Global().Call("matchMedia", "(pointer: coarse)").Get("matches").Bool()
}
