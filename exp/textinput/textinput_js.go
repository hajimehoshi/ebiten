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
	theTextInput.init()
}

type textInput struct {
	textareaElement js.Value

	session *session
}

var theTextInput textInput

func (t *textInput) init() {
	t.textareaElement = document.Call("createElement", "textarea")
	t.textareaElement.Set("id", "ebitengine-textinput")
	t.textareaElement.Set("autocapitalize", "off")
	t.textareaElement.Set("spellcheck", false)
	t.textareaElement.Set("translate", "no")
	t.textareaElement.Set("wrap", "off")

	style := t.textareaElement.Get("style")
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

	t.textareaElement.Call("addEventListener", "compositionend", js.FuncOf(func(this js.Value, args []js.Value) any {
		t.trySend(true)
		return nil
	}))
	t.textareaElement.Call("addEventListener", "focusout", js.FuncOf(func(this js.Value, args []js.Value) any {
		if t.session != nil {
			t.session.end()
			t.session = nil
		}
		return nil
	}))
	t.textareaElement.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		if e.Get("code").String() == "Tab" {
			e.Call("preventDefault")
		}
		if ui.IsVirtualKeyboard() && (e.Get("code").String() == "Enter" || e.Get("key").String() == "Enter") {
			// Ignore Enter key to avoid ebiten.IsKeyPressed(ebiten.KeyEnter) unexpectedly becomes true.
			e.Call("preventDefault")
			ui.Get().UpdateInputFromEvent(e)
			t.trySend(true)
			return nil
		}
		if !e.Get("isComposing").Bool() {
			ui.Get().UpdateInputFromEvent(e)
		}
		return nil
	}))
	t.textareaElement.Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		if !e.Get("isComposing").Bool() {
			ui.Get().UpdateInputFromEvent(e)
		}
		return nil
	}))
	t.textareaElement.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		// On iOS Safari, `isComposing` can be undefined.
		if e.Get("isComposing").IsUndefined() {
			t.trySend(false)
			return nil
		}
		if e.Get("isComposing").Bool() {
			t.trySend(false)
			return nil
		}
		if e.Get("inputType").String() == "insertLineBreak" {
			t.trySend(true)
			return nil
		}
		if e.Get("inputType").String() == "insertText" && e.Get("data").Equal(js.Null()) {
			// When a new line is inserted, the 'data' property might be null.
			t.trySend(true)
			return nil
		}
		// Though `isComposing` is false, send the text as being not committed for text completion with a virtual keyboard.
		if ui.IsVirtualKeyboard() {
			t.trySend(false)
			return nil
		}
		t.trySend(true)
		return nil
	}))
	t.textareaElement.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) any {
		t.trySend(true)
		return nil
	}))
	body.Call("appendChild", t.textareaElement)

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

func (t *textInput) Start(bounds image.Rectangle) (<-chan textInputState, func()) {
	if !t.textareaElement.Truthy() {
		return nil, nil
	}

	if js.Global().Get("_ebitengine_textinput_ready").Truthy() {
		if t.session != nil {
			t.session.end()
		}
		s := newSession()
		t.session = s
		js.Global().Get("window").Set("_ebitengine_textinput_ready", js.Undefined())
		return s.ch, s.end
	}

	// If a textarea is focused, create a session immediately.
	// A virtual keyboard should already be shown on mobile browsers.
	if document.Get("activeElement").Equal(t.textareaElement) {
		t.textareaElement.Set("value", "")
		t.textareaElement.Call("focus")
		style := t.textareaElement.Get("style")
		style.Set("left", fmt.Sprintf("%dpx", bounds.Min.X))
		style.Set("top", fmt.Sprintf("%dpx", bounds.Min.Y))
		style.Set("width", fmt.Sprintf("%dpx", bounds.Dx()))
		style.Set("height", fmt.Sprintf("%dpx", bounds.Dy()))
		style.Set("font-size", fmt.Sprintf("%dpx", bounds.Dy()))
		style.Set("line-height", fmt.Sprintf("%dpx", bounds.Dy()))

		if t.session == nil {
			s := newSession()
			t.session = s
		}
		return t.session.ch, func() {
			if t.session != nil {
				t.session.end()
				// Reset the session explictly, or a new session cannot be created above.
				t.session = nil
			}
		}
	}

	if t.session != nil {
		t.session.end()
		t.session = nil
	}

	// On iOS Safari, `focus` works only in user-interaction events (#2898).
	// Assuming Start is called every tick, defer the starting process to the next user-interaction event.
	js.Global().Get("window").Set("_ebitengine_textinput_x", bounds.Min.X)
	js.Global().Get("window").Set("_ebitengine_textinput_y", bounds.Max.Y)
	return nil, nil
}

func (t *textInput) trySend(committed bool) {
	if t.session == nil {
		return
	}

	textareaValue := t.textareaElement.Get("value").String()
	if textareaValue == "" {
		return
	}

	start := t.textareaElement.Get("selectionStart").Int()
	end := t.textareaElement.Get("selectionEnd").Int()
	startInBytes := convertUTF16CountToByteCount(textareaValue, start)
	endInBytes := convertUTF16CountToByteCount(textareaValue, end)

	t.session.trySend(textInputState{
		Text:                             textareaValue,
		CompositionSelectionStartInBytes: startInBytes,
		CompositionSelectionEndInBytes:   endInBytes,
		Committed:                        committed,
	})

	if committed {
		if t.session != nil {
			t.session.end()
			t.session = nil
		}
		t.textareaElement.Set("value", "")
	}
}
