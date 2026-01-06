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

package textinput_test

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

func TestPieceTableReplace(t *testing.T) {
	type replace struct {
		text  string
		start int
		end   int
	}

	tests := []struct {
		name     string
		init     string
		replaces []replace
		want     string
	}{
		{
			name: "insert at beginning",
			init: "World",
			replaces: []replace{
				{text: "Hello ", start: 0, end: 0},
			},
			want: "Hello World",
		},
		{
			name: "insert at end",
			init: "Hello",
			replaces: []replace{
				{text: " World", start: 5, end: 5},
			},
			want: "Hello World",
		},
		{
			name: "insert in middle",
			init: "Hello World",
			replaces: []replace{
				{text: ",", start: 5, end: 5},
			},
			want: "Hello, World",
		},
		{
			name: "delete at beginning",
			init: "Hello World",
			replaces: []replace{
				{text: "", start: 0, end: 6},
			},
			want: "World",
		},
		{
			name: "delete at end",
			init: "Hello World",
			replaces: []replace{
				{text: "", start: 5, end: 11},
			},
			want: "Hello",
		},
		{
			name: "delete in middle",
			init: "Hello, World",
			replaces: []replace{
				{text: "", start: 5, end: 6},
			},
			want: "Hello World",
		},
		{
			name: "replace",
			init: "Hello World",
			replaces: []replace{
				{text: "Gopher", start: 6, end: 11},
			},
			want: "Hello Gopher",
		},
		{
			name: "multiple operations",
			init: "A",
			replaces: []replace{
				{text: "B", start: 1, end: 1}, // AB
				{text: "C", start: 2, end: 2}, // ABC
				{text: "D", start: 0, end: 0}, // DABC
				{text: "E", start: 2, end: 2}, // DAEBC
				{text: "", start: 1, end: 2},  // DEBC
			},
			want: "DEBC",
		},
		{
			name: "empty init",
			init: "",
			replaces: []replace{
				{text: "Hello", start: 0, end: 0},
			},
			want: "Hello",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var p textinput.PieceTable
			p.Replace(tc.init, 0, 0)
			for _, r := range tc.replaces {
				p.Replace(r.text, r.start, r.end)
			}
			var b strings.Builder
			if _, err := p.WriteTo(&b); err != nil {
				t.Fatalf("WriteTo failed: %v", err)
			}
			if got := b.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPieceTableWriteToWithInsertion(t *testing.T) {
	tests := []struct {
		name  string
		init  string
		text  string
		start int
		end   int
		want  string
	}{
		{
			name:  "insert at beginning",
			init:  "World",
			text:  "Hello ",
			start: 0,
			end:   0,
			want:  "Hello World",
		},
		{
			name:  "insert at end",
			init:  "Hello",
			text:  " World",
			start: 5,
			end:   5,
			want:  "Hello World",
		},
		{
			name:  "insert in middle",
			init:  "Hello World",
			text:  ",",
			start: 5,
			end:   5,
			want:  "Hello, World",
		},
		{
			name:  "replace at beginning",
			init:  "Hello World",
			text:  "Hi",
			start: 0,
			end:   5,
			want:  "Hi World",
		},
		{
			name:  "replace at end",
			init:  "Hello World",
			text:  "Gopher",
			start: 6,
			end:   11,
			want:  "Hello Gopher",
		},
		{
			name:  "replace in middle",
			init:  "Hello World",
			text:  ", ",
			start: 5,
			end:   6,
			want:  "Hello, World",
		},
		{
			name:  "delete (replace with empty)",
			init:  "Hello World",
			text:  "",
			start: 5,
			end:   6,
			want:  "HelloWorld",
		},
		{
			name:  "empty init",
			init:  "",
			text:  "Hello",
			start: 0,
			end:   0,
			want:  "Hello",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var p textinput.PieceTable
			p.Replace(tc.init, 0, 0)
			var b strings.Builder
			if _, err := p.WriteToWithInsertion(&b, tc.text, tc.start, tc.end); err != nil {
				t.Fatalf("WriteToWithInsertion failed: %v", err)
			}
			if got := b.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}

			// Verify piece table itself is not modified
			var b2 strings.Builder
			if _, err := p.WriteTo(&b2); err != nil {
				t.Fatalf("WriteTo failed: %v", err)
			}
			if got := b2.String(); got != tc.init {
				t.Errorf("piece table modified: got %q, want %q", got, tc.init)
			}
		})
	}
}

func TestPieceTableUndoRedo(t *testing.T) {
	var p textinput.PieceTable

	check := func(want string) {
		t.Helper()
		var b strings.Builder
		if _, err := p.WriteTo(&b); err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
		if got := b.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}

	// Initial state
	check("")

	// Op 1: Insert "Hello"
	p.UpdateByIME(textinput.TextInputState{Text: "Hello"}, 0, 0)
	check("Hello")

	// Op 2: Insert "\n"
	p.UpdateByIME(textinput.TextInputState{Text: "\n"}, 5, 5)
	check("Hello\n")

	// Op 3: Insert "World"
	p.UpdateByIME(textinput.TextInputState{Text: "World"}, 6, 6)
	check("Hello\nWorld")

	// Undo Op 3
	start, end, ok := p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 6 || end != 6 {
		t.Errorf("Undo: got (%d, %d), want (6, 6)", start, end)
	}
	check("Hello\n")

	// Undo Op 2 and 1
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 0 {
		t.Errorf("Undo: got (%d, %d), want (0, 0)", start, end)
	}
	check("")

	// Undo (No effect)
	start, end, ok = p.Undo()
	if ok {
		t.Fatalf("Undo should not be possible")
	}
	check("")

	// Redo Op 1 and 2
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 0 || end != 6 {
		t.Errorf("Redo: got (%d, %d), want (0, 6)", start, end)
	}
	check("Hello\n")

	// Redo Op 3
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 6 || end != 11 {
		t.Errorf("Redo: got (%d, %d), want (6, 11)", start, end)
	}
	check("Hello\nWorld")

	// Redo (No effect)
	start, end, ok = p.Redo()
	if ok {
		t.Fatalf("Redo should not be possible")
	}
	check("Hello\nWorld")

	// Undo Op 3
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 6 || end != 6 {
		t.Errorf("Undo: got (%d, %d), want (6, 6)", start, end)
	}
	check("Hello\n")

	// New Op 3: Insert " Gopher" (Should clear redo stack for Op 2)
	p.Replace("Gopher", 6, 6)
	check("Hello\nGopher")

	// Undo Op 3
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 6 || end != 6 {
		t.Errorf("Undo: got (%d, %d), want (6, 6)", start, end)
	}
	check("Hello\n")

	// Redo Op 3
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 6 || end != 12 {
		t.Errorf("Redo: got (%d, %d), want (6, 12)", start, end)
	}
	check("Hello\nGopher")

	// Undo Op 3
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 6 || end != 6 {
		t.Errorf("Undo: got (%d, %d), want (6, 6)", start, end)
	}
	check("Hello\n")

	// Undo Op 2 and 1
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 0 {
		t.Errorf("Undo: got (%d, %d), want (0, 0)", start, end)
	}
	check("")
}

func TestPieceTableHistoryMerging(t *testing.T) {
	var p textinput.PieceTable

	check := func(want string) {
		t.Helper()
		var b strings.Builder
		if _, err := p.WriteTo(&b); err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
		if got := b.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}

	// Op 1: Merge sequential characters
	p.UpdateByIME(textinput.TextInputState{Text: "a"}, 0, 0)
	p.UpdateByIME(textinput.TextInputState{Text: "b"}, 1, 1)
	p.UpdateByIME(textinput.TextInputState{Text: "c"}, 2, 2)
	check("abc")

	// Undo Op 1
	start, end, ok := p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 0 {
		t.Errorf("Undo: got (%d, %d), want (0, 0)", start, end)
	}
	check("")

	// Redo Op 1
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Redo: got (%d, %d), want (0, 3)", start, end)
	}
	check("abc")

	// Op 2: Newline breaks merge.
	p.UpdateByIME(textinput.TextInputState{Text: "\n"}, 3, 3)
	check("abc\n")

	// Op 3: Insert "d" at 4.
	p.UpdateByIME(textinput.TextInputState{Text: "d"}, 4, 4)
	check("abc\nd")

	// Undo Op 3
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 4 || end != 4 {
		t.Errorf("Undo: got (%d, %d), want (4, 4)", start, end)
	}
	check("abc\n")

	// Redo Op 3
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 4 || end != 5 {
		t.Errorf("Redo: got (%d, %d), want (4, 5)", start, end)
	}
	check("abc\nd")

	// Op 4: Adjacency: Non-adjacent
	p.UpdateByIME(textinput.TextInputState{Text: "x"}, 0, 0)
	check("xabc\nd")

	// Undo Op 4
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 0 {
		t.Errorf("Undo: got (%d, %d), want (0, 0)", start, end)
	}
	check("abc\nd")
}

func TestPieceTableHistoryDelete(t *testing.T) {
	var p textinput.PieceTable

	check := func(want string) {
		t.Helper()
		var b strings.Builder
		if _, err := p.WriteTo(&b); err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
		if got := b.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}

	check("")

	p.UpdateByIME(textinput.TextInputState{
		Text: "Hello",
	}, 0, 0)
	check("Hello")

	// Op 1: Delete "o" and "l" like a backspace key
	p.Replace("", 4, 5)
	check("Hell")
	p.Replace("", 3, 4)
	check("Hel")

	// Op 2: Delete "H", "e", and "l" like a delete key
	p.Replace("", 0, 1)
	check("el")
	p.Replace("", 0, 1)
	check("l")
	p.Replace("", 0, 1)
	check("")

	// Undo Op 2
	start, end, ok := p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Undo: got (%d, %d), want (0, 3)", start, end)
	}
	check("Hel")

	// Undo Op 1
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 3 || end != 5 {
		t.Errorf("Undo: got (%d, %d), want (3, 5)", start, end)
	}
	check("Hello")

	// Redo Op 1
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 3 || end != 3 {
		t.Errorf("Redo: got (%d, %d), want (3, 3)", start, end)
	}
	check("Hel")

	// Redo Op 2
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 0 || end != 0 {
		t.Errorf("Redo: got (%d, %d), want (0, 0)", start, end)
	}
	check("")
}

func TestPieceTableHistoryMergingApplePressHold(t *testing.T) {
	var p textinput.PieceTable

	check := func(want string) {
		t.Helper()
		var b strings.Builder
		if _, err := p.WriteTo(&b); err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
		if got := b.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}

	// This test emulates Apple's behavior of press-and-hold.

	check("")

	// Op 1: Start with "foo"
	p.Replace("foo", 0, 0)
	check("foo")

	// Op 2: Add "a"
	p.UpdateByIME(textinput.TextInputState{Text: "a"}, 3, 3)
	check("fooa")

	// Op 3: Delete "a" and add "à"
	p.UpdateByIME(textinput.TextInputState{Text: "à", DeleteStartInBytes: 3, DeleteEndInBytes: 4}, 4, 4)
	check("fooà")

	// Op 4: Add "a"
	p.UpdateByIME(textinput.TextInputState{Text: "a"}, 5, 5)
	check("fooàa")

	// Op 5: Delete "a" and add "à"
	p.UpdateByIME(textinput.TextInputState{Text: "à", DeleteStartInBytes: 5, DeleteEndInBytes: 6}, 6, 6)
	check("fooàà")

	// Undo Op 5, 4, 3, 2
	start, end, ok := p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 3 || end != 3 {
		t.Errorf("Undo: got (%d, %d), want (3, 3)", start, end)
	}
	check("foo")

	// Undo Op 1 fails, as the initial state is determined by Replace.
	start, end, ok = p.Undo()
	if ok {
		t.Fatal("Undo should fail")
	}
	check("foo")

	// Redo Op 2, 3, 4, 5
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 3 || end != 7 {
		t.Errorf("Redo: got (%d, %d), want (3, 7)", start, end)
	}
	check("fooàà")
}

func TestPieceTableInitialStateWithReplace(t *testing.T) {
	var p textinput.PieceTable

	check := func(want string) {
		t.Helper()
		var b strings.Builder
		if _, err := p.WriteTo(&b); err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
		if got := b.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}

	check("")

	// Op 1
	p.Replace("foo", 0, 0)
	check("foo")

	// Op 2
	p.Replace("bar", 0, 3)
	check("bar")

	// Op 3
	p.Replace("baz", 0, 3)
	check("baz")

	// Undo Op 3
	start, end, ok := p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Undo: got (%d, %d), want (0, 3)", start, end)
	}
	check("bar")

	// Undo Op 2
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Undo: got (%d, %d), want (0, 3)", start, end)
	}
	check("foo")

	// Undo Op 1 fails, as the initial state is determined by Replace.
	start, end, ok = p.Undo()
	if ok {
		t.Fatal("Undo should fail")
	}
	check("foo")

	// Redo Op 2
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Redo: got (%d, %d), want (0, 3)", start, end)
	}
	check("bar")

	// Redo Op 3
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Redo: got (%d, %d), want (0, 3)", start, end)
	}
	check("baz")
}

func TestPieceTableInitialStateWithUpdateIME(t *testing.T) {
	var p textinput.PieceTable

	check := func(want string) {
		t.Helper()
		var b strings.Builder
		if _, err := p.WriteTo(&b); err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
		if got := b.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}

	check("")

	// Op 1
	p.UpdateByIME(textinput.TextInputState{Text: "foo"}, 0, 0)
	check("foo")

	// Op 2
	p.Replace("", 0, 3)
	check("")

	// Op 3
	p.Replace("bar", 0, 0)
	check("bar")

	// Op 4
	p.Replace("baz", 0, 3)
	check("baz")

	// Undo Op 4
	start, end, ok := p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Undo: got (%d, %d), want (0, 3)", start, end)
	}
	check("bar")

	// Undo Op 3
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 0 {
		t.Errorf("Undo: got (%d, %d), want (0, 0)", start, end)
	}
	check("")

	// Undo Op 2
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Undo: got (%d, %d), want (0, 3)", start, end)
	}
	check("foo")

	// Undo Op 1 succeeds, as the first operation is UpdateByIME.
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 0 {
		t.Errorf("Undo: got (%d, %d), want (0, 0)", start, end)
	}
	check("")

	// Redo Op 1
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Redo: got (%d, %d), want (0, 3)", start, end)
	}
	check("foo")

	// Redo Op 2
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 0 || end != 0 {
		t.Errorf("Redo: got (%d, %d), want (0, 0)", start, end)
	}
	check("")

	// Redo Op 3
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Redo: got (%d, %d), want (0, 3)", start, end)
	}
	check("bar")

	// Redo Op 4
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Redo: got (%d, %d), want (0, 3)", start, end)
	}
	check("baz")
}
