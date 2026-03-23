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

import (
	"image"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2"
)

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

func (t *textInput) endIfNeeded() {
	t.session.end()
}

var (
	sel_addSubview                 = objc.RegisterName("addSubview:")
	sel_alloc                      = objc.RegisterName("alloc")
	sel_contentView                = objc.RegisterName("contentView")
	sel_convertRectToScreen        = objc.RegisterName("convertRectToScreen:")
	sel_frame                      = objc.RegisterName("frame")
	sel_init                       = objc.RegisterName("init")
	sel_mainWindow                 = objc.RegisterName("mainWindow")
	sel_makeFirstResponder         = objc.RegisterName("makeFirstResponder:")
	sel_setFrame                   = objc.RegisterName("setFrame:")
	sel_sharedApplication          = objc.RegisterName("sharedApplication")
	sel_window                     = objc.RegisterName("window")
	sel_string                     = objc.RegisterName("string")
	sel_UTF8String                 = objc.RegisterName("UTF8String")
	sel_length                     = objc.RegisterName("length")
	sel_characterAtIndex           = objc.RegisterName("characterAtIndex:")
	sel_resignFirstResponder       = objc.RegisterName("resignFirstResponder")
	sel_isKindOfClass              = objc.RegisterName("isKindOfClass:")
	sel_array                      = objc.RegisterName("array")
	sel_lengthOfBytesUsingEncoding = objc.RegisterName("lengthOfBytesUsingEncoding:")

	class_NSArray            = objc.GetClass("NSArray")
	class_NSView             = objc.GetClass("NSView")
	class_NSAttributedString = objc.GetClass("NSAttributedString")
	idNSApplication          = objc.ID(objc.GetClass("NSApplication"))
)

var theTextInputClient objc.ID

func getTextInputClient() objc.ID {
	if theTextInputClient == 0 {
		class := objc.ID(class_TextInputClient)
		theTextInputClient = class.Send(sel_alloc).Send(sel_init)
	}
	return theTextInputClient
}

const nsUTF8StringEncoding = 4

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

type nsRange struct {
	location uint
	length   uint
}

func (t *textInput) start(bounds image.Rectangle) (<-chan textInputState, func()) {
	t.endIfNeeded()

	tc := getTextInputClient()
	window := idNSApplication.Send(sel_sharedApplication).Send(sel_mainWindow)
	contentView := window.Send(sel_contentView)
	contentView.Send(sel_addSubview, tc)
	window.Send(sel_makeFirstResponder, tc)

	r := objc.Send[nsRect](contentView, sel_frame)
	// The Y dirction is upward in the Cocoa coordinate system.
	y := int(r.size.height) - bounds.Max.Y
	// X is shifted a little bit, especially for the accent popup.
	bounds = bounds.Add(image.Pt(6, 0))
	tc.Send(sel_setFrame, nsRect{
		origin: nsPoint{float64(bounds.Min.X), float64(y)},
		size:   nsSize{float64(bounds.Dx()), float64(bounds.Dy())},
	})

	return t.session.start()
}

var class_TextInputClient objc.Class

func init() {
	var err error
	class_TextInputClient, err = objc.RegisterClass(
		"TextInputClient",
		class_NSView,
		[]*objc.Protocol{objc.GetProtocol("NSTextInputClient")},
		[]objc.FieldDef{},
		[]objc.MethodDef{
			{
				Cmd: objc.RegisterName("hasMarkedText"),
				Fn:  hasMarkedText,
			},
			{
				Cmd: objc.RegisterName("markedRange"),
				Fn:  markedRange,
			},
			{
				Cmd: objc.RegisterName("selectedRange"),
				Fn:  selectedRange,
			},
			{
				Cmd: objc.RegisterName("setMarkedText:selectedRange:replacementRange:"),
				Fn:  setMarkedText,
			},
			{
				Cmd: objc.RegisterName("unmarkText"),
				Fn:  unmarkText,
			},
			{
				Cmd: objc.RegisterName("validAttributesForMarkedText"),
				Fn:  validAttributesForMarkedText,
			},
			{
				Cmd: objc.RegisterName("attributedSubstringForProposedRange:actualRange:"),
				Fn:  attributedSubstringForProposedRange,
			},
			{
				Cmd: objc.RegisterName("insertText:replacementRange:"),
				Fn:  insertText,
			},
			{
				Cmd: objc.RegisterName("characterIndexForPoint:"),
				Fn:  characterIndexForPoint,
			},
			{
				Cmd: objc.RegisterName("firstRectForCharacterRange:actualRange:"),
				Fn:  firstRectForCharacterRange,
			},
			{
				Cmd: objc.RegisterName("doCommandBySelector:"),
				Fn:  doCommandBySelector,
			},
			{
				Cmd: sel_resignFirstResponder,
				Fn:  resignFirstResponder,
			},
		},
	)
	if err != nil {
		panic(err)
	}
}

func hasMarkedText(_ objc.ID, _ objc.SEL) bool {
	_, _, _, state, ok := currentState()
	if !ok {
		return false
	}
	return len(state.Text) > 0
}

func markedRange(_ objc.ID, _ objc.SEL) nsRange {
	text, startInBytes, _, state, ok := currentState()
	if !ok {
		return nsRange{location: ^uint(0), length: 0} // NSNotFound
	}
	if len(state.Text) == 0 {
		return nsRange{location: ^uint(0), length: 0} // NSNotFound
	}
	startInUTF16 := convertByteCountToUTF16Count(text, startInBytes)
	markedTextLenInUTF16 := convertByteCountToUTF16Count(state.Text, len(state.Text))
	return nsRange{location: uint(startInUTF16), length: uint(markedTextLenInUTF16)}
}

func selectedRange(_ objc.ID, _ objc.SEL) nsRange {
	text, startInBytes, endInBytes, _, ok := currentState()
	if !ok {
		return nsRange{location: ^uint(0), length: 0} // NSNotFound
	}
	startInUTF16 := convertByteCountToUTF16Count(text, startInBytes)
	endInUTF16 := convertByteCountToUTF16Count(text, endInBytes)
	return nsRange{location: uint(startInUTF16), length: uint(endInUTF16 - startInUTF16)}
}

func setMarkedText(_ objc.ID, _ objc.SEL, str objc.ID, selectedRange nsRange, replacementRange nsRange) {
	// selectionStart's origin is the beginning of the inserted text.
	// replaceStart's origin is also the beginning of the inserted text (= the marked text in the current implementation).
	// As the text argument already represents the complete marked text, it seems fine to ignore the replaceStart and replaceLen arguments.
	//
	// https://developer.apple.com/documentation/appkit/nstextinputclient/setmarkedtext(_:selectedrange:replacementrange:)?language=objc

	if str.Send(sel_isKindOfClass, objc.ID(class_NSAttributedString)) != 0 {
		str = str.Send(sel_string)
	}

	utf8Len := str.Send(sel_lengthOfBytesUsingEncoding, nsUTF8StringEncoding)
	charPtr := str.Send(sel_UTF8String)
	t := string(unsafe.Slice(*(**byte)(unsafe.Pointer(&charPtr)), utf8Len))

	startInBytes := convertUTF16CountToByteCount(t, int(selectedRange.location))
	endInBytes := convertUTF16CountToByteCount(t, int(selectedRange.location+selectedRange.length))
	theTextInput.update(t, startInBytes, endInBytes, 0, 0, false)
}

func unmarkText(_ objc.ID, _ objc.SEL) {
	// Do nothing
}

func validAttributesForMarkedText(_ objc.ID, _ objc.SEL) objc.ID {
	return objc.ID(class_NSArray).Send(sel_array)
}

func attributedSubstringForProposedRange(_ objc.ID, _ objc.SEL, _ nsRange, _ unsafe.Pointer) objc.ID {
	return 0 // nil
}

func insertText(_ objc.ID, _ objc.SEL, str objc.ID, replacementRange nsRange) {
	if str.Send(sel_isKindOfClass, objc.ID(class_NSAttributedString)) != 0 {
		str = str.Send(sel_string)
	}

	if str.Send(sel_length) == 1 && str.Send(sel_characterAtIndex, 0) < 0x20 {
		return
	}

	// replaceStart's origin is the beginning of the current text.
	//
	// https://developer.apple.com/documentation/appkit/nstextinputclient/inserttext(_:replacementrange:)?language=objc

	utf8Len := str.Send(sel_lengthOfBytesUsingEncoding, nsUTF8StringEncoding)
	charPtr := str.Send(sel_UTF8String)
	t := string(unsafe.Slice(*(**byte)(unsafe.Pointer(&charPtr)), utf8Len))

	var delStartInBytes, delEndInBytes int
	if int64(replacementRange.location) >= 0 {
		if text, _, _, _, ok := currentState(); ok {
			delStartInBytes = convertUTF16CountToByteCount(text, int(replacementRange.location))
			delEndInBytes = convertUTF16CountToByteCount(text, int(replacementRange.location+replacementRange.length))
		}
	}
	theTextInput.update(t, 0, len(t), delStartInBytes, delEndInBytes, true)
}

func characterIndexForPoint(_ objc.ID, _ objc.SEL, _ nsPoint) uint64 {
	return 0
}

func firstRectForCharacterRange(self objc.ID, _ objc.SEL, rang nsRange, actualRange *nsRange) nsRect {
	if actualRange != nil {
		if text, startInBytes, _, _, ok := currentState(); ok {
			s := uint(convertByteCountToUTF16Count(text, startInBytes))
			actualRange.location = s
			// 0 seems to work correctly.
			// See https://developer.apple.com/documentation/appkit/nstextinputclient/firstrect(forcharacterrange:actualrange:)?language=objc
			// > If the length of aRange is 0 (as it would be if there is nothing selected at the insertion point),
			// > the rectangle coincides with the insertion point, and its width is 0.
			actualRange.length = 0
		}
	}

	window := self.Send(sel_window)
	frame := objc.Send[nsRect](self, sel_frame)
	return objc.Send[nsRect](window, sel_convertRectToScreen, frame)
}

func doCommandBySelector(_ objc.ID, _ objc.SEL, _ objc.SEL) {
	// Do nothing
}

func resignFirstResponder(self objc.ID, cmd objc.SEL) bool {
	theTextInput.endIfNeeded()
	return objc.SendSuper[bool](self, cmd)
}
