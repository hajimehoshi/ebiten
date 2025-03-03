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

//go:build !ios

package textinput

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Cocoa
import "C"

import (
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

type textInput struct {
	// session must be accessed from the main thread.
	session *session
}

var theTextInput textInput

func (t *textInput) Start(x, y int) (<-chan State, func()) {
	var session *session
	ui.Get().RunOnMainThread(func() {
		t.end()
		start(x, y)
		session = newSession()
		t.session = session
	})
	return session.ch, session.end
}

//export ebitengine_textinput_update
func ebitengine_textinput_update(text *C.char, start, end C.int, committed C.int) {
	theTextInput.update(C.GoString(text), int(start), int(end), committed != 0)
}

func (t *textInput) update(text string, start, end int, committed bool) {
	if t.session != nil {
		startInBytes := convertUTF16CountToByteCount(text, start)
		endInBytes := convertUTF16CountToByteCount(text, end)
		t.session.trySend(State{
			Text:                             text,
			CompositionSelectionStartInBytes: startInBytes,
			CompositionSelectionEndInBytes:   endInBytes,
			Committed:                        committed,
		})
	}
	if committed {
		t.end()
	}
}

//export ebitengine_textinput_end
func ebitengine_textinput_end() {
	theTextInput.end()
}

func (t *textInput) end() {
	if t.session != nil {
		t.session.end()
		t.session = nil
	}
}

var (
	selAddSubview         = objc.RegisterName("addSubview:")
	selAlloc              = objc.RegisterName("alloc")
	selContentView        = objc.RegisterName("contentView")
	selFrame              = objc.RegisterName("frame")
	selInit               = objc.RegisterName("init")
	selMainWindow         = objc.RegisterName("mainWindow")
	selMakeFirstResponder = objc.RegisterName("makeFirstResponder:")
	selSetFrame           = objc.RegisterName("setFrame:")
	selSharedApplication  = objc.RegisterName("sharedApplication")

	idNSApplication = objc.ID(objc.GetClass("NSApplication"))
)

var theTextInputClient objc.ID

func getTextInputClient() objc.ID {
	if theTextInputClient == 0 {
		class := objc.ID(objc.GetClass("TextInputClient"))
		theTextInputClient = class.Send(selAlloc).Send(selInit)
	}
	return theTextInputClient
}

type nsPoint struct {
	x float64
	y float64
}

type nsSize struct {
	width  float64
	height float64
}

type nsRect struct {
	origin nsPoint
	size   nsSize
}

func start(x, y int) {
	t := getTextInputClient()
	window := idNSApplication.Send(selSharedApplication).Send(selMainWindow)
	contentView := window.Send(selContentView)
	contentView.Send(selAddSubview, t)
	window.Send(selMakeFirstResponder, t)

	r := objc.Send[nsRect](contentView, selFrame)
	y = int(r.size.height) - y - 4
	t.Send(selSetFrame, nsRect{
		origin: nsPoint{float64(x), float64(y)},
		size:   nsSize{1, 1},
	})
}
