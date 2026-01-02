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
