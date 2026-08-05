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

package glfw_test

import (
	"runtime"
	"slices"
	"testing"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

func TestReplaceRunes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		s      string
		first  int
		length int
		src    string
		want   string
	}{
		{"insert at head", "bc", 0, 0, "a", "abc"},
		{"insert at tail", "ab", 2, 0, "c", "abc"},
		{"insert in middle", "ac", 1, 0, "b", "abc"},
		{"delete", "abc", 1, 1, "", "ac"},
		{"delete all", "abc", 0, 3, "", ""},
		{"replace same length", "abc", 1, 1, "x", "axc"},
		{"replace longer", "abc", 1, 1, "xyz", "axyzc"},
		{"replace shorter", "abcde", 1, 3, "x", "axe"},
		{"replace all", "abc", 0, 3, "xyz", "xyz"},
		{"into empty", "", 0, 0, "abc", "abc"},
		{"multibyte", "あう", 1, 0, "い", "あいう"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := string(glfw.ReplaceRunes([]rune(tc.s), tc.first, tc.length, []rune(tc.src)))
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPreeditSelection(t *testing.T) {
	const (
		none  = glfw.XIMFeedback(0)
		under = glfw.XIMFeedback(glfw.XIMUnderline)
		rev   = glfw.XIMFeedback(glfw.XIMReverse)
		high  = glfw.XIMFeedback(glfw.XIMHighlight)
	)
	for _, tc := range []struct {
		name      string
		text      string
		feedback  []glfw.XIMFeedback
		caret     int
		wantStart int
		wantEnd   int
	}{
		{
			name:      "no feedback falls back to the caret",
			text:      "abc",
			feedback:  []glfw.XIMFeedback{none, none, none},
			caret:     2,
			wantStart: 2, wantEnd: 2,
		},
		{
			name:      "underline alone is not a selection",
			text:      "abc",
			feedback:  []glfw.XIMFeedback{under, under, under},
			caret:     1,
			wantStart: 1, wantEnd: 1,
		},
		{
			name:      "reverse run",
			text:      "abcd",
			feedback:  []glfw.XIMFeedback{under, rev, rev, under},
			caret:     0,
			wantStart: 1, wantEnd: 3,
		},
		{
			name:      "highlight run",
			text:      "abcd",
			feedback:  []glfw.XIMFeedback{none, none, high, high},
			caret:     0,
			wantStart: 2, wantEnd: 4,
		},
		{
			name:      "run at the head",
			text:      "abc",
			feedback:  []glfw.XIMFeedback{rev, none, none},
			caret:     0,
			wantStart: 0, wantEnd: 1,
		},
		{
			// Byte offsets, not character offsets: each of these is 3 bytes.
			name:      "multibyte run",
			text:      "あいう",
			feedback:  []glfw.XIMFeedback{none, rev, none},
			caret:     0,
			wantStart: 3, wantEnd: 6,
		},
		{
			name:      "multibyte caret",
			text:      "あいう",
			feedback:  []glfw.XIMFeedback{none, none, none},
			caret:     2,
			wantStart: 6, wantEnd: 6,
		},
		{
			name:      "caret past the end is clamped",
			text:      "abc",
			feedback:  []glfw.XIMFeedback{none, none, none},
			caret:     99,
			wantStart: 3, wantEnd: 3,
		},
		{
			name:      "negative caret is clamped",
			text:      "abc",
			feedback:  []glfw.XIMFeedback{none, none, none},
			caret:     -1,
			wantStart: 0, wantEnd: 0,
		},
		{
			name:      "short feedback is tolerated",
			text:      "abc",
			feedback:  nil,
			caret:     1,
			wantStart: 1, wantEnd: 1,
		},
		{
			name:      "empty text",
			text:      "",
			feedback:  nil,
			caret:     0,
			wantStart: 0, wantEnd: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			start, end := glfw.PreeditSelection([]rune(tc.text), tc.feedback, tc.caret)
			if start != tc.wantStart || end != tc.wantEnd {
				t.Errorf("got (%d, %d), want (%d, %d)", start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestDecodeXIMTextWideChar(t *testing.T) {
	wcs := []int32{'あ', 'b', 'c'}
	fb := []glfw.XIMFeedback{glfw.XIMReverse, 0, glfw.XIMHighlight}
	text := glfw.XIMText{
		Length:          uint16(len(wcs)),
		Feedback:        uintptr(unsafe.Pointer(&fb[0])),
		EncodingIsWChar: 1,
		String:          uintptr(unsafe.Pointer(&wcs[0])),
	}

	rs, gotFb := glfw.DecodeXIMText(&text)
	runtime.KeepAlive(wcs)
	runtime.KeepAlive(fb)

	if got, want := string(rs), "あbc"; got != want {
		t.Errorf("text: got %q, want %q", got, want)
	}
	if !slices.Equal(gotFb, fb) {
		t.Errorf("feedback: got %v, want %v", gotFb, fb)
	}
}

func TestDecodeXIMTextMultiByte(t *testing.T) {
	// The multi-byte string is NUL-terminated and, under a UTF-8 locale,
	// UTF-8 encoded.
	buf := append([]byte("あbc"), 0)
	fb := []glfw.XIMFeedback{glfw.XIMReverse, 0, glfw.XIMHighlight}
	text := glfw.XIMText{
		Length:          3,
		Feedback:        uintptr(unsafe.Pointer(&fb[0])),
		EncodingIsWChar: 0,
		String:          uintptr(unsafe.Pointer(&buf[0])),
	}

	rs, gotFb := glfw.DecodeXIMText(&text)
	runtime.KeepAlive(buf)
	runtime.KeepAlive(fb)

	if got, want := string(rs), "あbc"; got != want {
		t.Errorf("text: got %q, want %q", got, want)
	}
	if !slices.Equal(gotFb, fb) {
		t.Errorf("feedback: got %v, want %v", gotFb, fb)
	}
}

func TestDecodeXIMTextEmpty(t *testing.T) {
	var text glfw.XIMText
	rs, fb := glfw.DecodeXIMText(&text)
	if len(rs) != 0 || len(fb) != 0 {
		t.Errorf("got (%v, %v), want empty", rs, fb)
	}
}

// A feedback-only update carries no string: the input method is restyling
// characters it has already drawn, not replacing them.
func TestDecodeXIMFeedbackWithoutString(t *testing.T) {
	fb := []glfw.XIMFeedback{glfw.XIMReverse, glfw.XIMHighlight}
	text := glfw.XIMText{
		Length:   uint16(len(fb)),
		Feedback: uintptr(unsafe.Pointer(&fb[0])),
	}

	got := glfw.DecodeXIMFeedback(&text)
	runtime.KeepAlive(fb)

	if !slices.Equal(got, fb) {
		t.Errorf("got %v, want %v", got, fb)
	}
	// The text itself must not be reconstructed from a nil string.
	rs, _ := glfw.DecodeXIMText(&text)
	if len(rs) != 0 {
		t.Errorf("text: got %q, want empty", string(rs))
	}
}

func TestDecodeXIMFeedbackAbsent(t *testing.T) {
	text := glfw.XIMText{Length: 3}
	if got := glfw.DecodeXIMFeedback(&text); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

func TestApplyFeedback(t *testing.T) {
	const (
		none = glfw.XIMFeedback(0)
		rev  = glfw.XIMFeedback(glfw.XIMReverse)
		high = glfw.XIMFeedback(glfw.XIMHighlight)
	)
	for _, tc := range []struct {
		name     string
		feedback []glfw.XIMFeedback
		first    int
		src      []glfw.XIMFeedback
		want     []glfw.XIMFeedback
	}{
		{
			name:     "restyle in the middle",
			feedback: []glfw.XIMFeedback{none, none, none, none},
			first:    1,
			src:      []glfw.XIMFeedback{rev, rev},
			want:     []glfw.XIMFeedback{none, rev, rev, none},
		},
		{
			name:     "restyle from the head",
			feedback: []glfw.XIMFeedback{rev, rev, rev},
			first:    0,
			src:      []glfw.XIMFeedback{high},
			want:     []glfw.XIMFeedback{high, rev, rev},
		},
		{
			// A run reaching past the buffered composition is truncated rather
			// than growing it: there are no characters there to restyle.
			name:     "run past the end is truncated",
			feedback: []glfw.XIMFeedback{none, none},
			first:    1,
			src:      []glfw.XIMFeedback{rev, rev, rev},
			want:     []glfw.XIMFeedback{none, rev},
		},
		{
			name:     "start past the end changes nothing",
			feedback: []glfw.XIMFeedback{none, none},
			first:    5,
			src:      []glfw.XIMFeedback{rev},
			want:     []glfw.XIMFeedback{none, none},
		},
		{
			name:     "no feedback to apply changes nothing",
			feedback: []glfw.XIMFeedback{rev, high},
			first:    0,
			src:      nil,
			want:     []glfw.XIMFeedback{rev, high},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := glfw.ApplyFeedback(slices.Clone(tc.feedback), tc.first, tc.src)
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolvePreeditCaret(t *testing.T) {
	// A composition of 5 characters with its caret at 2.
	text := []rune("abcde")
	const (
		n     = 5
		caret = 2
	)
	for _, tc := range []struct {
		name      string
		direction int32
		position  int
		want      int
	}{
		{"absolute", glfw.XIMAbsolutePosition, 4, 4},
		{"absolute past the end is clamped", glfw.XIMAbsolutePosition, 99, n},
		{"absolute before the start is clamped", glfw.XIMAbsolutePosition, -1, 0},
		{"forward", glfw.XIMForwardChar, 0, caret + 1},
		{"backward", glfw.XIMBackwardChar, 0, caret - 1},
		// A composition drawn as one run is a single line, so its start and
		// end are the ends of the composition.
		{"line start", glfw.XIMLineStart, 0, 0},
		{"line end", glfw.XIMLineEnd, 0, n},
		{"no change", glfw.XIMDontChange, 0, caret},
		// A composition with no spaces is a single word, so its bounds are
		// the bounds of the composition.
		{"forward word", glfw.XIMForwardWord, 0, n},
		{"backward word", glfw.XIMBackwardWord, 0, 0},
		// Vertical movements resolve nowhere, and must report the caret
		// unmoved rather than leaving the input method to guess.
		{"caret up is unresolved", 4, 0, caret},
		{"next line is unresolved", 6, 0, caret},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := glfw.ResolvePreeditCaret(tc.direction, tc.position, caret, text); got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

func TestResolvePreeditCaretAtBoundaries(t *testing.T) {
	text := []rune("abc")
	if got := glfw.ResolvePreeditCaret(glfw.XIMForwardChar, 0, 3, text); got != 3 {
		t.Errorf("forward at the end: got %d, want 3", got)
	}
	if got := glfw.ResolvePreeditCaret(glfw.XIMBackwardChar, 0, 0, text); got != 0 {
		t.Errorf("backward at the start: got %d, want 0", got)
	}
	// An empty composition has both ends at 0.
	if got := glfw.ResolvePreeditCaret(glfw.XIMLineEnd, 0, 0, nil); got != 0 {
		t.Errorf("line end of an empty composition: got %d, want 0", got)
	}
}

func TestDecodeXIMTextWithoutFeedback(t *testing.T) {
	wcs := []int32{'a', 'b'}
	text := glfw.XIMText{
		Length:          uint16(len(wcs)),
		EncodingIsWChar: 1,
		String:          uintptr(unsafe.Pointer(&wcs[0])),
	}

	rs, fb := glfw.DecodeXIMText(&text)
	runtime.KeepAlive(wcs)

	if got, want := string(rs), "ab"; got != want {
		t.Errorf("text: got %q, want %q", got, want)
	}
	// Nil rather than zeroed, so the caller can tell "no feedback given" from
	// "no feedback set" and inherit the surrounding appearance instead.
	if fb != nil {
		t.Errorf("feedback: got %v, want nil", fb)
	}
}

func TestInheritedFeedback(t *testing.T) {
	const (
		none = glfw.XIMFeedback(0)
		rev  = glfw.XIMFeedback(glfw.XIMReverse)
		high = glfw.XIMFeedback(glfw.XIMHighlight)
	)
	for _, tc := range []struct {
		name     string
		feedback []glfw.XIMFeedback
		first    int
		length   int
		n        int
		want     []glfw.XIMFeedback
	}{
		{
			// Inserting inside a highlighted run keeps the run whole, so the
			// composition selection stays one range.
			name:     "insert inside a run takes the preceding character",
			feedback: []glfw.XIMFeedback{rev, rev, rev},
			first:    2,
			length:   0,
			n:        2,
			want:     []glfw.XIMFeedback{rev, rev},
		},
		{
			name:     "insert at the head takes the following character",
			feedback: []glfw.XIMFeedback{high, high},
			first:    0,
			length:   0,
			n:        1,
			want:     []glfw.XIMFeedback{high},
		},
		{
			// The preceding character wins when both sides exist, so a run
			// does not spread across the boundary that ends it.
			name:     "preceding wins over following",
			feedback: []glfw.XIMFeedback{none, rev},
			first:    1,
			length:   0,
			n:        1,
			want:     []glfw.XIMFeedback{none},
		},
		{
			name:     "replacement takes the character before the range",
			feedback: []glfw.XIMFeedback{rev, none, none},
			first:    1,
			length:   2,
			n:        1,
			want:     []glfw.XIMFeedback{rev},
		},
		{
			name:     "replacing everything has nothing to inherit",
			feedback: []glfw.XIMFeedback{rev, rev},
			first:    0,
			length:   2,
			n:        2,
			want:     []glfw.XIMFeedback{none, none},
		},
		{
			name:     "into an empty composition",
			feedback: nil,
			first:    0,
			length:   0,
			n:        2,
			want:     []glfw.XIMFeedback{none, none},
		},
		{
			name:     "no characters to give feedback to",
			feedback: []glfw.XIMFeedback{rev},
			first:    0,
			length:   0,
			n:        0,
			want:     nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := glfw.InheritedFeedback(tc.feedback, tc.first, tc.length, tc.n)
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

// The multi-byte string is in the locale's encoding. Under a UTF-8 locale, and
// under the fallback taken when libc cannot decode it, the result is the same.
func TestDecodeMultiByte(t *testing.T) {
	buf := append([]byte("あbc"), 0)
	got := glfw.DecodeMultiByte(uintptr(unsafe.Pointer(&buf[0])), 3)
	runtime.KeepAlive(buf)

	if want := "あbc"; string(got) != want {
		t.Errorf("got %q, want %q", string(got), want)
	}
}

// Word movement is defined over spaces, so it is meaningful for a composition
// that has them and collapses to the composition bounds for one that does not,
// as a composition of Japanese or Chinese characters is written without any.
func TestResolvePreeditCaretWord(t *testing.T) {
	for _, tc := range []struct {
		name      string
		text      string
		direction int32
		caret     int
		want      int
	}{
		{
			name:      "forward over the rest of a word",
			text:      "one two three",
			direction: glfw.XIMForwardWord,
			caret:     5, // inside "two"
			want:      7,
		},
		{
			name:      "forward skips the spaces before the next word",
			text:      "one two",
			direction: glfw.XIMForwardWord,
			caret:     3, // on the space
			want:      7,
		},
		{
			name:      "forward stops at the end",
			text:      "one two",
			direction: glfw.XIMForwardWord,
			caret:     7,
			want:      7,
		},
		{
			name:      "backward to the start of the current word",
			text:      "one two three",
			direction: glfw.XIMBackwardWord,
			caret:     6, // inside "two"
			want:      4,
		},
		{
			name:      "backward skips the spaces after the previous word",
			text:      "one two",
			direction: glfw.XIMBackwardWord,
			caret:     4, // at the head of "two"
			want:      0,
		},
		{
			name:      "backward stops at the start",
			text:      "one two",
			direction: glfw.XIMBackwardWord,
			caret:     0,
			want:      0,
		},
		{
			name:      "a composition without spaces is one word",
			text:      "あいうえお",
			direction: glfw.XIMForwardWord,
			caret:     2,
			want:      5,
		},
		{
			name:      "a composition without spaces is one word, backwards",
			text:      "あいうえお",
			direction: glfw.XIMBackwardWord,
			caret:     2,
			want:      0,
		},
		{
			name:      "an empty composition has nowhere to go",
			text:      "",
			direction: glfw.XIMForwardWord,
			caret:     0,
			want:      0,
		},
		{
			// The offsets are in characters, not bytes, so a caret past the
			// composition is clamped before the movement runs.
			name:      "a caret past the end is clamped",
			text:      "あいうえお",
			direction: glfw.XIMBackwardWord,
			caret:     99,
			want:      0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := glfw.ResolvePreeditCaret(tc.direction, 0, tc.caret, []rune(tc.text))
			if got != tc.want {
				t.Errorf("got %d, want %d", got, tc.want)
			}
		})
	}
}

// Xlib assigns no meaning to any feedback value, so which ones single out part
// of the composition is a matter of what input methods do with them. Reverse
// and highlight mark the part being worked on; the rest say how to draw a
// character, and a composition drawn entirely in one of them has no selection
// to report.
func TestPreeditSelectionFeedbackStyles(t *testing.T) {
	for _, tc := range []struct {
		name      string
		feedback  glfw.XIMFeedback
		wantStart int
		wantEnd   int
	}{
		{"reverse selects", glfw.XIMReverse, 1, 2},
		{"highlight selects", glfw.XIMHighlight, 1, 2},
		{"reverse with underline selects", glfw.XIMReverse | glfw.XIMUnderline, 1, 2},
		{"primary with reverse selects", glfw.XIMPrimary | glfw.XIMReverse, 1, 2},
		// Caret at 0, so an unselected composition reports the empty range there.
		{"underline alone does not select", glfw.XIMUnderline, 0, 0},
		{"primary alone does not select", glfw.XIMPrimary, 0, 0},
		{"secondary alone does not select", glfw.XIMSecondary, 0, 0},
		{"tertiary alone does not select", glfw.XIMTertiary, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Only the middle character carries the style under test.
			feedback := []glfw.XIMFeedback{0, tc.feedback, 0}
			start, end := glfw.PreeditSelection([]rune("abc"), feedback, 0)
			if start != tc.wantStart || end != tc.wantEnd {
				t.Errorf("got (%d, %d), want (%d, %d)", start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

// A composition every character of which carries a drawing style is not
// entirely selected: an input method that underlines the whole composition and
// reverses one clause is marking that clause, not everything.
func TestPreeditSelectionWholeCompositionStyled(t *testing.T) {
	const (
		under = glfw.XIMFeedback(glfw.XIMUnderline)
		rev   = glfw.XIMFeedback(glfw.XIMReverse | glfw.XIMUnderline)
	)
	feedback := []glfw.XIMFeedback{under, rev, rev, under}
	start, end := glfw.PreeditSelection([]rune("abcd"), feedback, 0)
	if start != 1 || end != 3 {
		t.Errorf("got (%d, %d), want (1, 3)", start, end)
	}
}

// The input method path and the ordinary character path must accept exactly the
// same characters, or a composition could commit what a key press cannot
// produce.
func TestIsTextCodepoint(t *testing.T) {
	for _, tc := range []struct {
		name      string
		codepoint rune
		want      bool
	}{
		{"nul", 0x00, false},
		{"tab", 0x09, false},
		{"newline", 0x0a, false},
		{"escape", 0x1b, false},
		{"space", 0x20, true},
		{"ascii", 'a', true},
		{"tilde", 0x7e, true},
		{"delete", 0x7f, false},
		// The C1 controls, which the input method path used to let through.
		{"first C1 control", 0x80, false},
		{"middle C1 control", 0x90, false},
		{"last C1 control", 0x9f, false},
		{"no-break space", 0xa0, true},
		{"latin letter", 0xe9, true},
		{"hiragana", 'あ', true},
		// Format characters are not controls, and stay accepted.
		{"zero width joiner", 0x200d, true},
		{"byte order mark", 0xfeff, true},
		{"emoji", 0x1f600, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := glfw.IsTextCodepoint(tc.codepoint); got != tc.want {
				t.Errorf("IsTextCodepoint(%#x) = %v, want %v", tc.codepoint, got, tc.want)
			}
		})
	}
}
