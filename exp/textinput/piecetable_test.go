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
	"io"
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

func TestPieceTableUTF16Count(t *testing.T) {
	type tc struct {
		c    int
		want int
	}
	type op struct {
		start, end int
		text       string
	}
	tests := []struct {
		name string
		// ops is a sequence of Replace operations applied in order. The
		// final piece-table layout depends on this sequence, so cases use
		// different sequences to exercise single-piece and multi-piece
		// content with various rune sizes.
		ops  []op
		full string
		u2b  []tc // utf16CountToByteCount cases
		b2u  []tc // byteCountToUTF16Count cases
	}{
		{
			name: "ASCII single piece",
			ops:  []op{{0, 0, "Hello, World"}},
			full: "Hello, World",
			u2b:  []tc{{0, 0}, {5, 5}, {12, 12}, {13, -1}},
			b2u:  []tc{{0, 0}, {5, 5}, {12, 12}, {13, -1}},
		},
		{
			name: "BMP single piece",
			ops:  []op{{0, 0, "海老天"}},
			full: "海老天",
			u2b:  []tc{{0, 0}, {1, 3}, {2, 6}, {3, 9}, {4, -1}},
			b2u:  []tc{{0, 0}, {3, 1}, {6, 2}, {9, 3}, {10, -1}},
		},
		{
			name: "Supplementary single piece",
			ops:  []op{{0, 0, "寿司🍣食べたい"}},
			full: "寿司🍣食べたい",
			// 寿(3B/1U) 司(3/1) 🍣(4/2) 食(3/1) べ(3/1) た(3/1) い(3/1) -> 22 bytes / 8 UTF-16 units
			u2b: []tc{{0, 0}, {1, 3}, {2, 6}, {4, 10}, {5, 13}, {8, 22}, {9, -1}},
			b2u: []tc{{0, 0}, {3, 1}, {6, 2}, {10, 4}, {13, 5}, {22, 8}, {23, -1}},
		},
		{
			name: "Multi-piece ASCII",
			ops: []op{
				{0, 0, "Hello World"},
				{5, 6, ", "},     // "Hello, World"
				{7, 12, "there"}, // "Hello, there"
			},
			full: "Hello, there",
			u2b:  []tc{{0, 0}, {5, 5}, {7, 7}, {12, 12}, {13, -1}},
			b2u:  []tc{{0, 0}, {5, 5}, {7, 7}, {12, 12}, {13, -1}},
		},
		{
			name: "Multi-piece mixed",
			// Final content "Hello,🍣 there!" is laid out across four
			// pieces: "Hello,", "🍣", " there", "!". Exercises the
			// boundary between an ASCII piece and a non-ASCII piece, so
			// both the per-piece ASCII fast path and the rune-walk path
			// run in one call.
			ops: []op{
				{0, 0, "Hello, there"}, // "Hello, there"
				{6, 6, "🍣"},            // "Hello,🍣 there" (16 bytes)
				{16, 16, "!"},          // "Hello,🍣 there!" (17 bytes)
			},
			full: "Hello,🍣 there!",
			// "Hello,"(6B/6U) 🍣(4B/2U) " there"(6B/6U) "!"(1B/1U) -> 17B/15U
			u2b: []tc{{0, 0}, {6, 6}, {7, 10}, {8, 10}, {9, 11}, {15, 17}, {16, -1}},
			b2u: []tc{{0, 0}, {6, 6}, {10, 8}, {11, 9}, {17, 15}, {18, -1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p textinput.PieceTable
			for _, o := range tt.ops {
				p.Replace(o.text, o.start, o.end)
			}
			// Sanity check setup.
			var b strings.Builder
			if _, err := p.WriteTo(&b); err != nil {
				t.Fatalf("WriteTo failed: %v", err)
			}
			if got := b.String(); got != tt.full {
				t.Fatalf("setup: got %q, want %q", got, tt.full)
			}
			for _, c := range tt.u2b {
				if got := p.UTF16CountToByteCount(c.c); got != c.want {
					t.Errorf("UTF16CountToByteCount(%d) = %d, want %d", c.c, got, c.want)
				}
			}
			for _, c := range tt.b2u {
				if got := p.ByteCountToUTF16Count(c.c); got != c.want {
					t.Errorf("ByteCountToUTF16Count(%d) = %d, want %d", c.c, got, c.want)
				}
			}
		})
	}
}

func TestPieceTableWriteRangeTo(t *testing.T) {
	// Build a fragmented piece table by replacing in the middle and via IME insertions
	// at different positions. The piece table should now have multiple items.
	var p textinput.PieceTable
	p.Replace("Hello World", 0, 0)
	p.Replace(", ", 5, 6)                                      // "Hello, World"
	p.Replace("there", 7, 12)                                  // "Hello, there"
	p.UpdateByIME(textinput.TextInputState{Text: "!"}, 12, 12) // "Hello, there!"

	const full = "Hello, there!"

	// Sanity check the setup.
	{
		var b strings.Builder
		if _, err := p.WriteTo(&b); err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
		if got := b.String(); got != full {
			t.Fatalf("setup: got %q, want %q", got, full)
		}
	}

	tests := []struct {
		name  string
		start int
		end   int
		want  string
	}{
		{
			name:  "whole range",
			start: 0,
			end:   len(full),
			want:  full,
		},
		{
			name:  "empty range in middle",
			start: 5,
			end:   5,
			want:  "",
		},
		{
			name:  "empty range at start",
			start: 0,
			end:   0,
			want:  "",
		},
		{
			name:  "empty range at end",
			start: len(full),
			end:   len(full),
			want:  "",
		},
		{
			name:  "start > end",
			start: 8,
			end:   3,
			want:  "",
		},
		{
			name:  "prefix",
			start: 0,
			end:   5,
			want:  "Hello",
		},
		{
			name:  "suffix",
			start: 7,
			end:   len(full),
			want:  "there!",
		},
		{
			name:  "single character",
			start: 0,
			end:   1,
			want:  "H",
		},
		{
			name:  "clamp negative start",
			start: -10,
			end:   5,
			want:  "Hello",
		},
		{
			name:  "clamp end past length",
			start: 7,
			end:   1000,
			want:  "there!",
		},
		{
			name:  "clamp both ends",
			start: -1,
			end:   len(full) + 100,
			want:  full,
		},
		{
			name:  "start past length",
			start: len(full) + 5,
			end:   len(full) + 10,
			want:  "",
		},
		{
			name:  "both past length",
			start: len(full) + 1,
			end:   len(full) + 2,
			want:  "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			n, err := p.WriteRangeTo(&b, tc.start, tc.end)
			if err != nil {
				t.Fatalf("WriteRangeTo failed: %v", err)
			}
			if got := b.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if int(n) != len(tc.want) {
				t.Errorf("n: got %d, want %d", n, len(tc.want))
			}
		})
	}
}

func TestPieceTableWriteRangeRoundTrip(t *testing.T) {
	// Fragment the piece table so the round-trip exercises piece boundaries.
	var p textinput.PieceTable
	p.Replace("Hello World", 0, 0)
	p.Replace(", ", 5, 6)
	p.Replace("there", 7, 12)
	p.UpdateByIME(textinput.TextInputState{Text: "!"}, 12, 12)
	p.UpdateByIME(textinput.TextInputState{Text: "?"}, 13, 13)

	var ref strings.Builder
	if _, err := p.WriteTo(&ref); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}
	full := ref.String()

	for start := 0; start <= len(full); start++ {
		for end := start; end <= len(full); end++ {
			var b strings.Builder
			if _, err := p.WriteRangeTo(&b, start, end); err != nil {
				t.Fatalf("WriteRangeTo(%d, %d) failed: %v", start, end, err)
			}
			if got, want := b.String(), full[start:end]; got != want {
				t.Errorf("range [%d, %d): got %q, want %q", start, end, got, want)
			}
		}
	}
}

func TestPieceTableWriteRangePieceBoundaries(t *testing.T) {
	// Construct a piece table with known piece boundaries.
	// After these operations, the piece table fragments into multiple items at
	// well-defined byte offsets.
	var p textinput.PieceTable
	p.Replace("AAA", 0, 0)
	p.Replace("BBB", 3, 3) // "AAABBB" - new piece starting at byte 3
	p.Replace("CCC", 6, 6) // "AAABBBCCC" - new piece starting at byte 6

	const full = "AAABBBCCC"
	{
		var b strings.Builder
		if _, err := p.WriteTo(&b); err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
		if got := b.String(); got != full {
			t.Fatalf("setup: got %q, want %q", got, full)
		}
	}

	// Read at the known piece boundaries.
	tests := []struct {
		name  string
		start int
		end   int
		want  string
	}{
		{
			name:  "first piece exactly",
			start: 0,
			end:   3,
			want:  "AAA",
		},
		{
			name:  "second piece exactly",
			start: 3,
			end:   6,
			want:  "BBB",
		},
		{
			name:  "third piece exactly",
			start: 6,
			end:   9,
			want:  "CCC",
		},
		{
			name:  "start at boundary, span two pieces",
			start: 3,
			end:   9,
			want:  "BBBCCC",
		},
		{
			name:  "end at boundary, span two pieces",
			start: 0,
			end:   6,
			want:  "AAABBB",
		},
		{
			name:  "cross seam mid-pieces",
			start: 1,
			end:   8,
			want:  "AABBBCC",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			if _, err := p.WriteRangeTo(&b, tc.start, tc.end); err != nil {
				t.Fatalf("WriteRangeTo failed: %v", err)
			}
			if got := b.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPieceTableWriteRangeEmpty(t *testing.T) {
	var p textinput.PieceTable

	// Zero-length text: any range should return empty.
	for _, r := range []struct {
		start int
		end   int
	}{
		{
			start: 0,
			end:   0,
		},
		{
			start: -5,
			end:   5,
		},
		{
			start: 0,
			end:   100,
		},
	} {
		var b strings.Builder
		if _, err := p.WriteRangeTo(&b, r.start, r.end); err != nil {
			t.Fatalf("WriteRangeTo(%d, %d) failed: %v", r.start, r.end, err)
		}
		if got := b.String(); got != "" {
			t.Errorf("range [%d, %d) on empty: got %q, want %q", r.start, r.end, got, "")
		}
	}
}

func TestPieceTableWriteRangeAfterUndoRedo(t *testing.T) {
	var p textinput.PieceTable
	p.Replace("Hello", 0, 0)
	p.Replace(", World", 5, 5) // "Hello, World"

	check := func(start, end int, want string) {
		t.Helper()
		var b strings.Builder
		if _, err := p.WriteRangeTo(&b, start, end); err != nil {
			t.Fatalf("WriteRangeTo failed: %v", err)
		}
		if got := b.String(); got != want {
			t.Errorf("range [%d, %d): got %q, want %q", start, end, got, want)
		}
	}

	// Current state: "Hello, World"
	check(0, 12, "Hello, World")
	check(7, 12, "World")

	// Undo back to "Hello".
	if _, _, ok := p.Undo(); !ok {
		t.Fatal("Undo failed")
	}
	check(0, 5, "Hello")
	// End past current length must clamp to current Len, not stale state.
	check(0, 100, "Hello")
	// A range valid in the prior state must not leak old bytes.
	check(7, 12, "")

	// Redo back to "Hello, World".
	if _, _, ok := p.Redo(); !ok {
		t.Fatal("Redo failed")
	}
	check(0, 12, "Hello, World")
	check(7, 12, "World")
}

type errWriter struct {
	err       error
	failAfter int
	written   int
}

func (e *errWriter) Write(b []byte) (int, error) {
	if e.written >= e.failAfter {
		return 0, e.err
	}
	e.written += len(b)
	return len(b), nil
}

func TestPieceTableWriteRangeWriterError(t *testing.T) {
	var p textinput.PieceTable
	p.Replace("AAA", 0, 0)
	p.Replace("BBB", 3, 3)

	wantErr := io.ErrShortWrite
	w := &errWriter{err: wantErr, failAfter: 2}
	if _, err := p.WriteRangeTo(w, 0, 6); err != wantErr {
		t.Errorf("got err %v, want %v", err, wantErr)
	}
}

func TestPieceTableWriteRangeToWithInsertion(t *testing.T) {
	// "Hello, there!" laid out across multiple pieces.
	newPT := func() *textinput.PieceTable {
		var p textinput.PieceTable
		p.Replace("Hello World", 0, 0)
		p.Replace(", ", 5, 6)
		p.Replace("there", 7, 12)
		p.UpdateByIME(textinput.TextInputState{Text: "!"}, 12, 12)
		return &p
	}

	tests := []struct {
		name        string
		text        string
		insertStart int
		insertEnd   int
		rangeStart  int
		rangeEnd    int
		want        string
	}{
		{
			name:        "no insertion empty text, range matches writeRangeTo",
			text:        "",
			insertStart: 5,
			insertEnd:   5,
			rangeStart:  0,
			rangeEnd:    13,
			want:        "Hello, there!",
		},
		{
			name:        "no insertion empty text, partial range",
			text:        "",
			insertStart: 0,
			insertEnd:   0,
			rangeStart:  7,
			rangeEnd:    12,
			want:        "there",
		},
		{
			name:        "pure insertion, range entirely before",
			text:        "[XYZ]",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  0,
			rangeEnd:    5,
			want:        "Hello",
		},
		{
			// Rendering: "Hello, [XYZ]there!" length 18.
			name:        "pure insertion, range entirely after",
			text:        "[XYZ]",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  12,
			rangeEnd:    18,
			want:        "there!",
		},
		{
			// Rendering: "Hello, [XYZ]there!"; bytes [5, 14) = ", [XYZ]th".
			name:        "pure insertion, range straddles insertion point",
			text:        "[XYZ]",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  5,
			rangeEnd:    14,
			want:        ", [XYZ]th",
		},
		{
			// Composition occupies [7, 12); inside is "XYZ" at [8, 11).
			name:        "pure insertion, range inside composition",
			text:        "[XYZ]",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  8,
			rangeEnd:    11,
			want:        "XYZ",
		},
		{
			// "ABC" replaces "there"; rendering "Hello, ABC!" length 11; bytes [5, 10) = ", ABC".
			name:        "replacement (start != end), range straddles",
			text:        "ABC",
			insertStart: 7,
			insertEnd:   12,
			rangeStart:  5,
			rangeEnd:    10,
			want:        ", ABC",
		},
		{
			// Rendering: "Hello, ABCDEF!" length 14; composition occupies [7, 13).
			name:        "replacement, range entirely inside composition",
			text:        "ABCDEF",
			insertStart: 7,
			insertEnd:   12,
			rangeStart:  8,
			rangeEnd:    12,
			want:        "BCDE",
		},
		{
			// Rendering: "Hello, ABC!" length 11; suffix "!" at [10, 11).
			name:        "replacement, range entirely after composition",
			text:        "ABC",
			insertStart: 7,
			insertEnd:   12,
			rangeStart:  10,
			rangeEnd:    11,
			want:        "!",
		},
		{
			// Rendering: "Hello, <C>there!" length 16; range [0, 7) = "Hello, ".
			name:        "range exactly at insertion start boundary",
			text:        "<C>",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  0,
			rangeEnd:    7,
			want:        "Hello, ",
		},
		{
			// Range [7, 10) = "<C>".
			name:        "range exactly at insertion end boundary",
			text:        "<C>",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  7,
			rangeEnd:    10,
			want:        "<C>",
		},
		{
			name:        "empty range",
			text:        "<C>",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  5,
			rangeEnd:    5,
			want:        "",
		},
		{
			name:        "full range with composition",
			text:        "<C>",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  0,
			rangeEnd:    16,
			want:        "Hello, <C>there!",
		},
		{
			name:        "negative start clamps to 0",
			text:        "<C>",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  -100,
			rangeEnd:    5,
			want:        "Hello",
		},
		{
			name:        "end past renderingLength clamps",
			text:        "<C>",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  13,
			rangeEnd:    1000,
			want:        "re!",
		},
		{
			name:        "start > end yields empty",
			text:        "<C>",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  9,
			rangeEnd:    4,
			want:        "",
		},
		{
			name:        "start past renderingLength yields empty",
			text:        "<C>",
			insertStart: 7,
			insertEnd:   7,
			rangeStart:  1000,
			rangeEnd:    2000,
			want:        "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := newPT()
			var b strings.Builder
			n, err := p.WriteRangeToWithInsertion(&b, tc.text, tc.insertStart, tc.insertEnd, tc.rangeStart, tc.rangeEnd)
			if err != nil {
				t.Fatalf("WriteRangeToWithInsertion failed: %v", err)
			}
			if got := b.String(); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
			if int(n) != len(tc.want) {
				t.Errorf("n: got %d, want %d", n, len(tc.want))
			}
		})
	}
}

func TestPieceTableWriteRangeToWithInsertionEmptyText(t *testing.T) {
	// When text == "" and insertStart == insertEnd, output must equal writeRangeTo for the same range.
	var p textinput.PieceTable
	p.Replace("Hello, World!", 0, 0)
	p.Replace("there", 7, 12) // fragment

	for start := 0; start <= 13; start++ {
		for end := start; end <= 13; end++ {
			var got, want strings.Builder
			if _, err := p.WriteRangeToWithInsertion(&got, "", 5, 5, start, end); err != nil {
				t.Fatalf("WriteRangeToWithInsertion(%d, %d) failed: %v", start, end, err)
			}
			if _, err := p.WriteRangeTo(&want, start, end); err != nil {
				t.Fatalf("WriteRangeTo(%d, %d) failed: %v", start, end, err)
			}
			if got.String() != want.String() {
				t.Errorf("range [%d, %d): got %q, want %q", start, end, got.String(), want.String())
			}
		}
	}
}

func TestPieceTableWriteRangeToWithInsertionRoundTrip(t *testing.T) {
	// Fragment so traversal exercises piece boundaries.
	var p textinput.PieceTable
	p.Replace("Hello World", 0, 0)
	p.Replace(", ", 5, 6)
	p.Replace("there", 7, 12)
	p.UpdateByIME(textinput.TextInputState{Text: "!"}, 12, 12)

	cases := []struct {
		name        string
		text        string
		insertStart int
		insertEnd   int
	}{
		{
			name:        "no-op",
			text:        "",
			insertStart: 0,
			insertEnd:   0,
		},
		{
			name:        "pure insert at start",
			text:        "ABC",
			insertStart: 0,
			insertEnd:   0,
		},
		{
			name:        "pure insert at end",
			text:        "ABC",
			insertStart: 13,
			insertEnd:   13,
		},
		{
			name:        "pure insert in middle",
			text:        "ABC",
			insertStart: 7,
			insertEnd:   7,
		},
		{
			name:        "replacement",
			text:        "ABCDEFG",
			insertStart: 7,
			insertEnd:   12,
		},
		{
			name:        "pure delete via composition (empty text)",
			text:        "",
			insertStart: 7,
			insertEnd:   12,
		},
		{
			name:        "replace whole text",
			text:        "X",
			insertStart: 0,
			insertEnd:   13,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Build the full reference output via writeToWithInsertion.
			var ref strings.Builder
			if _, err := p.WriteToWithInsertion(&ref, c.text, c.insertStart, c.insertEnd); err != nil {
				t.Fatalf("WriteToWithInsertion failed: %v", err)
			}
			full := ref.String()
			for start := 0; start <= len(full); start++ {
				for end := start; end <= len(full); end++ {
					var b strings.Builder
					if _, err := p.WriteRangeToWithInsertion(&b, c.text, c.insertStart, c.insertEnd, start, end); err != nil {
						t.Fatalf("WriteRangeToWithInsertion(%q, %d, %d, %d, %d) failed: %v",
							c.text, c.insertStart, c.insertEnd, start, end, err)
					}
					if got, want := b.String(), full[start:end]; got != want {
						t.Errorf("range [%d, %d): got %q, want %q", start, end, got, want)
					}
				}
			}
		})
	}
}

func TestPieceTableWriteRangeToWithInsertionWriterError(t *testing.T) {
	var p textinput.PieceTable
	p.Replace("AAABBB", 0, 0)

	wantErr := io.ErrShortWrite
	w := &errWriter{err: wantErr, failAfter: 2}
	if _, err := p.WriteRangeToWithInsertion(w, "XYZ", 3, 3, 0, 9); err != wantErr {
		t.Errorf("got err %v, want %v", err, wantErr)
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
	p.Reset("foo")
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

func TestPieceTableReset(t *testing.T) {
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
	p.Reset("foo")
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

func TestPieceTableResetInMiddle(t *testing.T) {
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
	p.Reset("bar")
	check("bar")

	// Op 3
	p.Replace("baz", 0, 3)
	check("baz")

	// Op 4
	p.Replace("qux", 0, 3)
	check("qux")

	// Undo Op 4
	start, end, ok := p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Undo: got (%d, %d), want (0, 3)", start, end)
	}
	check("baz")

	// Undo Op 3
	start, end, ok = p.Undo()
	if !ok {
		t.Fatal("Undo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Undo: got (%d, %d), want (0, 3)", start, end)
	}
	check("bar")

	// Undo Op 2 fails
	start, end, ok = p.Undo()
	if ok {
		t.Fatal("Undo should fail")
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

	// Redo Op 4
	start, end, ok = p.Redo()
	if !ok {
		t.Fatal("Redo failed")
	}
	if start != 0 || end != 3 {
		t.Errorf("Redo: got (%d, %d), want (0, 3)", start, end)
	}
	check("qux")
}

// errReader yields data and then returns err on the next Read.
type errReader struct {
	data []byte
	err  error
}

func (r *errReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, r.err
	}
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, nil
}

func TestPieceTableReadFrom(t *testing.T) {
	check := func(t *testing.T, p *textinput.PieceTable, want string) {
		t.Helper()
		var b strings.Builder
		if _, err := p.WriteTo(&b); err != nil {
			t.Fatalf("WriteTo failed: %v", err)
		}
		if got := b.String(); got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}

	t.Run("empty", func(t *testing.T) {
		var p textinput.PieceTable
		n, err := p.ReadFrom(strings.NewReader(""))
		if err != nil {
			t.Fatalf("ReadFrom failed: %v", err)
		}
		if n != 0 {
			t.Errorf("ReadFrom returned n=%d, want 0", n)
		}
		check(t, &p, "")
	})

	t.Run("short", func(t *testing.T) {
		var p textinput.PieceTable
		n, err := p.ReadFrom(strings.NewReader("hello"))
		if err != nil {
			t.Fatalf("ReadFrom failed: %v", err)
		}
		if int(n) != len("hello") {
			t.Errorf("ReadFrom returned n=%d, want %d", n, len("hello"))
		}
		check(t, &p, "hello")
	})

	t.Run("longerThanMinRead", func(t *testing.T) {
		// Larger than the 512-byte minRead to exercise the grow path.
		want := strings.Repeat("abcdefgh", 200)
		var p textinput.PieceTable
		n, err := p.ReadFrom(strings.NewReader(want))
		if err != nil {
			t.Fatalf("ReadFrom failed: %v", err)
		}
		if int(n) != len(want) {
			t.Errorf("ReadFrom returned n=%d, want %d", n, len(want))
		}
		check(t, &p, want)
	})

	t.Run("replacesExistingContent", func(t *testing.T) {
		var p textinput.PieceTable
		p.Replace("old content here", 0, 0)
		if _, err := p.ReadFrom(strings.NewReader("new")); err != nil {
			t.Fatalf("ReadFrom failed: %v", err)
		}
		check(t, &p, "new")
		// History is reset: undo should not restore the previous text.
		if _, _, ok := p.Undo(); ok {
			t.Errorf("Undo should fail after ReadFrom")
		}
	})

	t.Run("readerError", func(t *testing.T) {
		var p textinput.PieceTable
		p.Reset("kept")
		wantErr := io.ErrUnexpectedEOF
		_, err := p.ReadFrom(&errReader{data: []byte("partial"), err: wantErr})
		if err != wantErr {
			t.Errorf("ReadFrom error: got %v, want %v", err, wantErr)
		}
		check(t, &p, "")
	})
}
