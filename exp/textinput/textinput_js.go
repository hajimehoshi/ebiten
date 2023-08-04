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
		if !e.Get("isComposing").Bool() {
			ui.UpdateInputFromEvent(e)
		}
		return nil
	}))
	t.textareaElement.Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		if !e.Get("isComposing").Bool() {
			ui.UpdateInputFromEvent(e)
		}
		return nil
	}))
	t.textareaElement.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) any {
		e := args[0]
		t.trySend(!e.Get("isComposing").Bool())
		return nil
	}))
	// TODO: What about other events like wheel?
}

func (t *textInput) Start(x, y int) (chan State, func()) {
	if !t.textareaElement.Truthy() {
		return nil, nil
	}

	if t.session != nil {
		t.session.end()
		t.session = nil
	}

	if !body.Call("contains", t.textareaElement).Bool() {
		body.Call("appendChild", t.textareaElement)
	}
	t.textareaElement.Set("value", "")
	t.textareaElement.Call("focus")

	xf, yf := ui.LogicalPositionToClientPosition(float64(x), float64(y))
	style := t.textareaElement.Get("style")
	style.Set("left", fmt.Sprintf("%0.2fpx", xf))
	style.Set("top", fmt.Sprintf("%0.2fpx", yf))

	s := newSession()
	t.session = s
	return s.ch, s.end
}

func (t *textInput) trySend(committed bool) {
	if t.session == nil {
		return
	}

	textareaValue := t.textareaElement.Get("value").String()
	start := t.textareaElement.Get("selectionStart").Int()
	end := t.textareaElement.Get("selectionEnd").Int()
	startInBytes := convertUTF16CountToByteCount(textareaValue, start)
	endInBytes := convertUTF16CountToByteCount(textareaValue, end)

	t.session.trySend(State{
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
