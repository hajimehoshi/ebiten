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

//go:build freebsd || linux || netbsd

package glfw

import (
	"slices"
	"sync"
	"unicode"
	"unsafe"

	"github.com/ebitengine/purego"
)

// PreeditCallback is called when the composition the input method shows
// changes. selStartInBytes and selEndInBytes delimit the highlighted part of
// text, and are both the caret position when nothing is highlighted. An empty
// text means the composition has ended.
type PreeditCallback func(w *Window, text string, selStartInBytes, selEndInBytes int)

// TextInputCallback is called with text produced by the keyboard input path,
// which is either committed by the input method or typed directly.
type TextInputCallback func(w *Window, text string)

// SetPreeditCallback sets the callback reporting composition updates, and
// returns the previously set one.
//
// Compositions are reported only when the input method supports the
// on-the-spot input style. Otherwise the input method draws the composition
// itself and this callback is never called.
func (w *Window) SetPreeditCallback(cbfun PreeditCallback) (PreeditCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.platform.preeditCallback
	w.platform.preeditCallback = cbfun
	return old, nil
}

// SetTextInputCallback sets the callback reporting committed text, and returns
// the previously set one.
func (w *Window) SetTextInputCallback(cbfun TextInputCallback) (TextInputCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.platform.textInputCallback
	w.platform.textInputCallback = cbfun
	return old, nil
}

// inputText reports text produced by the keyboard input path. plain reports
// whether the text was produced without a modifier combination the platform
// treats as a shortcut. Text from a shortcut chord, and control characters,
// are dropped: neither is text the application can insert.
func (w *Window) inputText(text string, plain bool) {
	if !plain {
		return
	}
	if w.consumeDiscardedText(text) {
		return
	}
	if w.platform.textInputCallback == nil {
		return
	}
	// The same characters the character callback drops, so that what the input
	// method commits and what an ordinary key press produces agree.
	filtered := make([]rune, 0, len(text))
	for _, r := range text {
		if !isTextCodepoint(r) {
			continue
		}
		filtered = append(filtered, r)
	}
	if len(filtered) == 0 {
		return
	}
	w.platform.textInputCallback(w, string(filtered))
}

// The XIM callbacks registered for the on-the-spot input style. The nested
// list handed to XCreateIC refers to the names and the values, so both live at
// package level to stay valid for as long as the input contexts do. They are
// used on the main thread only.
var (
	preeditStartCallbackName = []byte("preeditStartCallback\x00")
	preeditDoneCallbackName  = []byte("preeditDoneCallback\x00")
	preeditDrawCallbackName  = []byte("preeditDrawCallback\x00")
	preeditCaretCallbackName = []byte("preeditCaretCallback\x00")

	preeditStartCallbackValue _XIMCallback
	preeditDoneCallbackValue  _XIMCallback
	preeditDrawCallbackValue  _XIMCallback
	preeditCaretCallbackValue _XIMCallback
)

// createInputContext creates the window's X input context. The context is 0
// when no input method is available.
//
// createInputContext must be called from the main thread.
func (w *Window) createInputContext() {
	if _glfw.platformWindow.im == 0 {
		return
	}

	if _glfw.platformWindow.imStyle&_XIMPreeditCallbacks != 0 {
		w.platform.ic = w.createOnTheSpotInputContext()
		if w.platform.ic != 0 {
			return
		}
		// The input method advertised the on-the-spot style but would not
		// create a context in it, so fall back to over-the-spot, where the
		// input method draws the composition itself.
		_glfw.platformWindow.imStyle = _XIMPreeditNothing | _XIMStatusNothing
	}

	w.platform.ic = xCreateIC(_glfw.platformWindow.im,
		"inputStyle",
		_glfw.platformWindow.imStyle,
		"clientWindow",
		w.platform.handle,
		"focusWindow",
		w.platform.handle,
		0)
}

// createOnTheSpotInputContext creates an input context that reports its
// composition through the preedit callbacks, and returns 0 on failure.
//
// createOnTheSpotInputContext must be called from the main thread.
func (w *Window) createOnTheSpotInputContext() uintptr {
	// The window is identified to the callbacks by its XID rather than by a Go
	// pointer, which must not be stored in memory handed to a C library.
	clientData := uintptr(w.platform.handle)
	preeditStartCallbackValue = _XIMCallback{ClientData: clientData, Callback: preeditStartCallbackPtr()}
	preeditDoneCallbackValue = _XIMCallback{ClientData: clientData, Callback: preeditDoneCallbackPtr()}
	preeditDrawCallbackValue = _XIMCallback{ClientData: clientData, Callback: preeditDrawCallbackPtr()}
	preeditCaretCallbackValue = _XIMCallback{ClientData: clientData, Callback: preeditCaretCallbackPtr()}

	list := xVaCreateNestedList(0,
		&preeditStartCallbackName[0], &preeditStartCallbackValue,
		&preeditDoneCallbackName[0], &preeditDoneCallbackValue,
		&preeditDrawCallbackName[0], &preeditDrawCallbackValue,
		&preeditCaretCallbackName[0], &preeditCaretCallbackValue,
		0)
	if list == 0 {
		return 0
	}
	defer xFree(list)

	return xCreateICPreedit(_glfw.platformWindow.im,
		"inputStyle",
		_glfw.platformWindow.imStyle,
		"clientWindow",
		w.platform.handle,
		"focusWindow",
		w.platform.handle,
		"preeditAttributes",
		list,
		0)
}

// preeditWindow returns the window the XIM callback client data identifies, or
// nil when the window has already been destroyed.
func preeditWindow(clientData uintptr) *Window {
	return _glfw.platformWindow.windowsByXID[_XID(clientData)]
}

// The XIM callback trampolines. Registering a Go function as a C callback is
// permanent, so each is created once.
var (
	preeditStartCallbackPtr = sync.OnceValue(func() uintptr {
		return purego.NewCallback(preeditStartCallback)
	})
	preeditDoneCallbackPtr = sync.OnceValue(func() uintptr {
		return purego.NewCallback(preeditDoneCallback)
	})
	preeditDrawCallbackPtr = sync.OnceValue(func() uintptr {
		return purego.NewCallback(preeditDrawCallback)
	})
	preeditCaretCallbackPtr = sync.OnceValue(func() uintptr {
		return purego.NewCallback(preeditCaretCallback)
	})
)

// preeditStartCallback returns the maximum length of the composition, which is
// -1 for unlimited.
func preeditStartCallback(ic uintptr, clientData uintptr, callData uintptr) uintptr {
	w := preeditWindow(clientData)
	if w == nil {
		return ^uintptr(0)
	}
	// A new composition means the input method dropped the discarded one.
	w.platform.discardedText = ""
	w.clearPreedit()
	return ^uintptr(0)
}

func preeditDoneCallback(ic uintptr, clientData uintptr, callData uintptr) uintptr {
	w := preeditWindow(clientData)
	if w == nil {
		return 0
	}
	w.clearPreedit()
	w.inputPreedit()
	return 0
}

func (w *Window) clearPreedit() {
	w.platform.preeditText = w.platform.preeditText[:0]
	w.platform.preeditFeedback = w.platform.preeditFeedback[:0]
	w.platform.preeditCaret = 0
}

// ResetInputContext discards the composition the input method holds. The
// discarded composition is not reported.
//
// ResetInputContext must be called from the main thread.
func (w *Window) ResetInputContext() error {
	if !_glfw.initialized {
		return NotInitialized
	}
	if w.platform.ic == 0 {
		return nil
	}
	// Resetting deletes the pending input, so the buffered composition has to
	// go with it. A reset returns the composition it deleted, and must do so
	// whenever one was visible, leaving the input method with nothing further
	// to report. One that returns nothing may instead report the composition
	// as a commit, so remember it to recognize it if it comes back.
	if s := xmbResetIC(w.platform.ic); s != 0 {
		xFree(s)
		w.platform.discardedText = ""
	} else {
		w.platform.discardedText = string(w.platform.preeditText)
	}
	w.clearPreedit()
	return nil
}

// consumeDiscardedText reports whether text is the composition a reset
// discarded, reported one last time by an input method that commits what it
// was holding instead of dropping it. Such a commit belongs to the target that
// was abandoned, which no longer exists, so it is nobody's input.
//
// The composition is only ever reported once, so the first text after a reset
// settles it either way: it is the discarded composition, or the input method
// dropped it as asked and this is new input.
func (w *Window) consumeDiscardedText(text string) bool {
	if w.platform.discardedText == "" {
		return false
	}
	discarded := w.platform.discardedText
	w.platform.discardedText = ""
	return text == discarded
}

func preeditDrawCallback(ic uintptr, clientData uintptr, callData uintptr) uintptr {
	w := preeditWindow(clientData)
	if w == nil || callData == 0 {
		return 0
	}
	draw := (*_XIMPreeditDrawCallbackStruct)(unsafe.Pointer(callData))

	// The input method addresses the composition it has drawn so far, so clamp
	// the change to what is actually buffered.
	first := min(max(int(draw.ChgFirst), 0), len(w.platform.preeditText))
	length := min(max(int(draw.ChgLength), 0), len(w.platform.preeditText)-first)

	if draw.Text == 0 {
		// The change is a deletion.
		w.platform.preeditText = replaceSlice(w.platform.preeditText, first, length, nil)
		w.platform.preeditFeedback = replaceSlice(w.platform.preeditFeedback, first, length, nil)
	} else if text := (*_XIMText)(unsafe.Pointer(draw.Text)); text.String == 0 && text.Length > 0 {
		// The change restyles characters that are already drawn, leaving them
		// in place. It is a no-op when there is no feedback to apply either.
		w.platform.preeditFeedback = applyFeedback(w.platform.preeditFeedback, first, decodeXIMFeedback(text))
	} else {
		rs, feedback := decodeXIMText(text)
		if feedback == nil {
			feedback = inheritedFeedback(w.platform.preeditFeedback, first, length, len(rs))
		}
		w.platform.preeditText = replaceSlice(w.platform.preeditText, first, length, rs)
		w.platform.preeditFeedback = replaceSlice(w.platform.preeditFeedback, first, length, feedback)
	}

	w.platform.preeditCaret = min(max(int(draw.Caret), 0), len(w.platform.preeditText))
	if len(w.platform.preeditText) > 0 {
		// Drawing a composition means the input method dropped the discarded
		// one. Clearing the composition is part of discarding it, so an empty
		// one says nothing.
		w.platform.discardedText = ""
	}
	w.inputPreedit()
	return 0
}

func preeditCaretCallback(ic uintptr, clientData uintptr, callData uintptr) uintptr {
	w := preeditWindow(clientData)
	if w == nil || callData == 0 {
		return 0
	}
	caret := (*_XIMPreeditCaretCallbackStruct)(unsafe.Pointer(callData))
	position := resolvePreeditCaret(caret.Direction, int(caret.Position),
		w.platform.preeditCaret, w.platform.preeditText)
	// The input method reads the resolved position back, whether or not the
	// movement got anywhere.
	caret.Position = int32(position)
	if position == w.platform.preeditCaret {
		return 0
	}
	w.platform.preeditCaret = position
	w.inputPreedit()
	return 0
}

// resolvePreeditCaret returns the character offset a caret movement resolves
// to within text, whose caret is at caret. position is the offset the input
// method supplied, which only an absolute movement uses. A movement that
// resolves nowhere leaves the caret where it is.
func resolvePreeditCaret(direction int32, position, caret int, text []rune) int {
	n := len(text)
	caret = min(max(caret, 0), n)
	switch direction {
	case _XIMAbsolutePosition:
		return min(max(position, 0), n)
	case _XIMForwardChar:
		return min(caret+1, n)
	case _XIMBackwardChar:
		return max(caret-1, 0)
	case _XIMForwardWord:
		return forwardWord(text, caret)
	case _XIMBackwardWord:
		return backwardWord(text, caret)
	case _XIMLineStart:
		return 0
	case _XIMLineEnd:
		return n
	case _XIMDontChange:
		return caret
	default:
		// Vertical movements have nowhere to go in a composition the input
		// method draws as a single run of characters.
		return caret
	}
}

// forwardWord returns the offset past the word that follows caret: the spaces
// before the word, then the word itself. A composition written without spaces
// is one word, so the offset is its end.
func forwardWord(text []rune, caret int) int {
	i := caret
	for i < len(text) && unicode.IsSpace(text[i]) {
		i++
	}
	for i < len(text) && !unicode.IsSpace(text[i]) {
		i++
	}
	return i
}

// backwardWord returns the offset at the start of the word that precedes
// caret. A composition written without spaces is one word, so the offset is
// its start.
func backwardWord(text []rune, caret int) int {
	i := caret
	for i > 0 && unicode.IsSpace(text[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(text[i-1]) {
		i--
	}
	return i
}

// inputPreedit reports the buffered composition.
func (w *Window) inputPreedit() {
	if w.platform.preeditCallback == nil {
		return
	}
	start, end := preeditSelection(w.platform.preeditText, w.platform.preeditFeedback, w.platform.preeditCaret)
	w.platform.preeditCallback(w, string(w.platform.preeditText), start, end)
}

// inheritedFeedback returns the feedback for n characters replacing length
// characters at first, for text the input method sent none with: it keeps the
// appearance of the text around it. The character before the replaced range
// gives it, or the one after when the range starts the composition.
func inheritedFeedback(feedback []_XIMFeedback, first, length, n int) []_XIMFeedback {
	if n == 0 {
		return nil
	}
	var f _XIMFeedback
	switch {
	case first > 0 && first-1 < len(feedback):
		f = feedback[first-1]
	case first+length < len(feedback):
		f = feedback[first+length]
	}
	return slices.Repeat([]_XIMFeedback{f}, n)
}

// applyFeedback overwrites the feedback of len(src) characters at first,
// leaving the characters themselves and any feedback beyond them untouched.
func applyFeedback(feedback []_XIMFeedback, first int, src []_XIMFeedback) []_XIMFeedback {
	for i, f := range src {
		j := first + i
		if j >= len(feedback) {
			break
		}
		feedback[j] = f
	}
	return feedback
}

// selectionFeedback is the feedback an input method marks the part of the
// composition it is working on with. Xlib assigns no meaning to any feedback
// value, so this is what input methods use in practice: the rest describe how
// to draw a character rather than which characters are singled out, and a
// composition whose every character carries one of those has no selection.
const selectionFeedback = _XIMReverse | _XIMHighlight

// preeditSelection returns the byte range of the run the input method has
// highlighted, or the empty range at the caret when nothing is highlighted.
func preeditSelection(text []rune, feedback []_XIMFeedback, caret int) (startInBytes, endInBytes int) {
	first, last := -1, -1
	for i := range text {
		if i < len(feedback) && feedback[i]&selectionFeedback != 0 {
			if first < 0 {
				first = i
			}
			last = i + 1
			continue
		}
		if first >= 0 {
			break
		}
	}
	if first < 0 {
		c := min(max(caret, 0), len(text))
		b := len(string(text[:c]))
		return b, b
	}
	return len(string(text[:first])), len(string(text[:last]))
}

// decodeXIMText returns the characters of an XIMText and their feedback. The
// feedback is as long as the returned text.
func decodeXIMText(text *_XIMText) ([]rune, []_XIMFeedback) {
	n := int(text.Length)
	if n <= 0 || text.String == 0 {
		return nil, nil
	}

	var rs []rune
	if text.EncodingIsWChar != 0 {
		wcs := unsafe.Slice((*int32)(unsafe.Pointer(text.String)), n)
		rs = make([]rune, 0, n)
		for _, wc := range wcs {
			rs = append(rs, rune(wc))
		}
	} else {
		rs = decodeMultiByte(text.String, n)
	}

	// No feedback means the text keeps whatever the text around it has, which
	// only the caller knows.
	src := decodeXIMFeedback(text)
	if src == nil {
		return rs, nil
	}
	fb := make([]_XIMFeedback, len(rs))
	copy(fb, src)
	return rs, fb
}

// decodeMultiByte returns the characters of a NUL-terminated multi-byte string
// of n characters. The string is in the encoding of the locale bound to the
// input method, which is the one initIME set. Decoding reads the process-wide
// LC_CTYPE, so the two agree only as long as nothing changes the locale after
// initIME.
func decodeMultiByte(s uintptr, n int) []rune {
	if mbstowcs != nil {
		// One more than the characters to convert, so that a string of exactly
		// n characters is still terminated.
		wcs := make([]int32, n+1)
		if got := mbstowcs(&wcs[0], s, uintptr(len(wcs))); got != ^uintptr(0) {
			rs := make([]rune, 0, min(int(got), n))
			for _, wc := range wcs[:min(int(got), n)] {
				rs = append(rs, rune(wc))
			}
			return rs
		}
	}
	// The locale does not decode the string, or libc offers no way to. Read it
	// as UTF-8, which is what it is in any locale in ordinary use.
	return []rune(goString(s))
}

// decodeXIMFeedback returns the feedback of an XIMText, which is nil when the
// input method sends none.
func decodeXIMFeedback(text *_XIMText) []_XIMFeedback {
	n := int(text.Length)
	if n <= 0 || text.Feedback == 0 {
		return nil
	}
	return unsafe.Slice((*_XIMFeedback)(unsafe.Pointer(text.Feedback)), n)
}

// replaceSlice replaces length elements of s at first with src.
func replaceSlice[T any](s []T, first, length int, src []T) []T {
	s = slices.Delete(s, first, first+length)
	return slices.Insert(s, first, src...)
}
