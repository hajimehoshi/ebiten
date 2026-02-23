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
//
// #include <stdint.h>
// #include <Cocoa/Cocoa.h>
import "C"

import (
	"image"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2"
)

//export ebitengine_textinput_hasMarkedText
func ebitengine_textinput_hasMarkedText() C.int64_t {
	_, _, _, state, ok := currentState()
	if !ok {
		return 0
	}
	if len(state.Text) > 0 {
		return 1
	}
	return 0
}

//export ebitengine_textinput_markedRange
func ebitengine_textinput_markedRange(start, length *C.int64_t) {
	*start = -1
	*length = 0

	text, startInBytes, _, state, ok := currentState()
	if !ok {
		return
	}

	if len(state.Text) == 0 {
		return
	}

	startInUTF16 := convertByteCountToUTF16Count(text, startInBytes)
	markedTextLenInUTF16 := convertByteCountToUTF16Count(state.Text, len(state.Text))
	*start = C.int64_t(startInUTF16)
	*length = C.int64_t(markedTextLenInUTF16)
}

//export ebitengine_textinput_selectedRange
func ebitengine_textinput_selectedRange(start, length *C.int64_t) {
	*start = -1
	*length = 0

	text, startInBytes, endInBytes, _, ok := currentState()
	if !ok {
		return
	}

	startInUTF16 := convertByteCountToUTF16Count(text, startInBytes)
	endInUTF16 := convertByteCountToUTF16Count(text, endInBytes)
	*start = C.int64_t(startInUTF16)
	*length = C.int64_t(endInUTF16 - startInUTF16)
}

//export ebitengine_textinput_unmarkText
func ebitengine_textinput_unmarkText() {
}

//export ebitengine_textinput_setMarkedText
func ebitengine_textinput_setMarkedText(text *C.char, selectionStart, selectionLen, replaceStart, replaceLen C.int64_t) {
	// selectionStart's origin is the beginning of the inserted text.
	// replaceStart's origin is also the beginning of the inserted text (= the marked text in the current implementation).
	// As the text argument already represents the complete marked text, it seems fine to ignore the replaceStart and replaceLen arguments.
	//
	// https://developer.apple.com/documentation/appkit/nstextinputclient/setmarkedtext(_:selectedrange:replacementrange:)?language=objc

	t := C.GoString(text)
	startInBytes := convertUTF16CountToByteCount(t, int(selectionStart))
	endInBytes := convertUTF16CountToByteCount(t, int(selectionStart+selectionLen))
	theTextInput.update(t, startInBytes, endInBytes, 0, 0, false)
}

//export ebitengine_textinput_insertText
func ebitengine_textinput_insertText(text *C.char, replaceStart, replaceLen C.int64_t) {
	// replaceStart's origin is the beginning of the current text.
	//
	// https://developer.apple.com/documentation/appkit/nstextinputclient/inserttext(_:replacementrange:)?language=objc

	t := C.GoString(text)
	var delStartInBytes, delEndInBytes int
	if replaceStart >= 0 {
		if text, _, _, _, ok := currentState(); ok {
			delStartInBytes = convertUTF16CountToByteCount(text, int(replaceStart))
			delEndInBytes = convertUTF16CountToByteCount(text, int(replaceStart+replaceLen))
		}
	}
	theTextInput.update(t, 0, len(t), delStartInBytes, delEndInBytes, true)
}

//export ebitengine_textinput_firstRectForCharacterRange
func ebitengine_textinput_firstRectForCharacterRange(self C.uintptr_t, crange C.NSRange, actualRange C.NSRangePointer) C.NSRect {
	if actualRange != nil {
		if text, startInBytes, _, _, ok := currentState(); ok {
			s := C.NSUInteger(convertUTF16CountToByteCount(text, startInBytes))
			actualRange.location = s
			// 0 seems to work correctly.
			// See https://developer.apple.com/documentation/appkit/nstextinputclient/firstrect(forcharacterrange:actualrange:)?language=objc
			// > If the length of aRange is 0 (as it would be if there is nothing selected at the insertion point),
			// > the rectangle coincides with the insertion point, and its width is 0.
			actualRange.length = 0
		}
	}

	window := objc.ID(self).Send(selWindow)
	frame := objc.Send[C.NSRect](objc.ID(self), selFrame)
	return objc.Send[C.NSRect](window, selConvertRectToScreen, frame)
}

type textInput struct {
	// session must be accessed from the main thread.
	session session
}

var theTextInput textInput

func (t *textInput) Start(bounds image.Rectangle) (<-chan textInputState, func()) {
	var ch <-chan textInputState
	var end func()
	ebiten.RunOnMainThread(func() {
		ch, end = t.start(bounds)
	})
	return ch, end
}

func (t *textInput) update(text string, startInBytes, endInBytes int, deleteStartInBytes, deleteEndInBytes int, committed bool) {
	t.session.send(textInputState{
		Text:                             text,
		CompositionSelectionStartInBytes: startInBytes,
		CompositionSelectionEndInBytes:   endInBytes,
		DeleteStartInBytes:               deleteStartInBytes,
		DeleteEndInBytes:                 deleteEndInBytes,
		Committed:                        committed,
	})
	if committed {
		t.endIfNeeded()
	}
}

//export ebitengine_textinput_end
func ebitengine_textinput_end() {
	theTextInput.endIfNeeded()
}

func (t *textInput) endIfNeeded() {
	t.session.end()
}

var (
	selAddSubview          = objc.RegisterName("addSubview:")
	selAlloc               = objc.RegisterName("alloc")
	selContentView         = objc.RegisterName("contentView")
	selConvertRectToScreen = objc.RegisterName("convertRectToScreen:")
	selFrame               = objc.RegisterName("frame")
	selInit                = objc.RegisterName("init")
	selMainWindow          = objc.RegisterName("mainWindow")
	selMakeFirstResponder  = objc.RegisterName("makeFirstResponder:")
	selSetFrame            = objc.RegisterName("setFrame:")
	selSharedApplication   = objc.RegisterName("sharedApplication")
	selWindow              = objc.RegisterName("window")

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

func (t *textInput) start(bounds image.Rectangle) (<-chan textInputState, func()) {
	t.endIfNeeded()

	tc := getTextInputClient()
	window := idNSApplication.Send(selSharedApplication).Send(selMainWindow)
	contentView := window.Send(selContentView)
	contentView.Send(selAddSubview, tc)
	window.Send(selMakeFirstResponder, tc)

	r := objc.Send[nsRect](contentView, selFrame)
	// The Y dirction is upward in the Cocoa coordinate system.
	y := int(r.size.height) - bounds.Max.Y
	// X is shifted a little bit, especially for the accent popup.
	bounds = bounds.Add(image.Pt(6, 0))
	tc.Send(selSetFrame, nsRect{
		origin: nsPoint{float64(bounds.Min.X), float64(y)},
		size:   nsSize{float64(bounds.Dx()), float64(bounds.Dy())},
	})

	return t.session.start()
}
