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

// These wrappers re-export the preedit helpers for x11_ime_linbsd_test.go,
// which is an external (glfw_test) test.

type XIMFeedback = _XIMFeedback

const (
	XIMReverse   = _XIMReverse
	XIMUnderline = _XIMUnderline
	XIMHighlight = _XIMHighlight
	XIMPrimary   = _XIMPrimary
	XIMSecondary = _XIMSecondary
	XIMTertiary  = _XIMTertiary

	XIMForwardChar      = _XIMForwardChar
	XIMBackwardChar     = _XIMBackwardChar
	XIMForwardWord      = _XIMForwardWord
	XIMBackwardWord     = _XIMBackwardWord
	XIMLineStart        = _XIMLineStart
	XIMLineEnd          = _XIMLineEnd
	XIMAbsolutePosition = _XIMAbsolutePosition
	XIMDontChange       = _XIMDontChange
)

func PreeditSelection(text []rune, feedback []XIMFeedback, caret int) (int, int) {
	return preeditSelection(text, feedback, caret)
}

func ReplaceRunes(s []rune, first, length int, src []rune) []rune {
	return replaceSlice(s, first, length, src)
}

func DecodeXIMText(text *XIMText) ([]rune, []XIMFeedback) {
	return decodeXIMText(text)
}

func DecodeXIMFeedback(text *XIMText) []XIMFeedback {
	return decodeXIMFeedback(text)
}

func ApplyFeedback(feedback []XIMFeedback, first int, src []XIMFeedback) []XIMFeedback {
	return applyFeedback(feedback, first, src)
}

func InheritedFeedback(feedback []XIMFeedback, first, length, n int) []XIMFeedback {
	return inheritedFeedback(feedback, first, length, n)
}

func DecodeMultiByte(s uintptr, n int) []rune {
	return decodeMultiByte(s, n)
}

func IsTextCodepoint(codepoint rune) bool {
	return isTextCodepoint(codepoint)
}

func ResolvePreeditCaret(direction int32, position, caret int, text []rune) int {
	return resolvePreeditCaret(direction, position, caret, text)
}
