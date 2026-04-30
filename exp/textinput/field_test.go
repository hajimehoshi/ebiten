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
	"time"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

func TestFieldChangedAtZeroValue(t *testing.T) {
	var f textinput.Field
	if got := f.ChangedAt(); !got.IsZero() {
		t.Errorf("fresh Field: ChangedAt() = %v, want zero value", got)
	}
}

func TestFieldChangedAtAdvancesOnMutations(t *testing.T) {
	testCases := []struct {
		name   string
		setup  func(*textinput.Field)
		mutate func(*textinput.Field)
	}{
		{
			name:  "ResetText",
			setup: func(f *textinput.Field) {},
			mutate: func(f *textinput.Field) {
				f.ResetText("hello")
			},
		},
		{
			name:  "SetTextAndSelection",
			setup: func(f *textinput.Field) {},
			mutate: func(f *textinput.Field) {
				f.SetTextAndSelection("abc", 1, 2)
			},
		},
		{
			name: "ReplaceText",
			setup: func(f *textinput.Field) {
				f.ResetText("hello")
			},
			mutate: func(f *textinput.Field) {
				f.ReplaceText("X", 0, 5)
			},
		},
		{
			name: "ReplaceTextAtSelection",
			setup: func(f *textinput.Field) {
				f.ResetText("hello")
				f.SetSelection(0, 5)
			},
			mutate: func(f *textinput.Field) {
				f.ReplaceTextAtSelection("Y")
			},
		},
		{
			name: "SetSelection",
			setup: func(f *textinput.Field) {
				f.ResetText("hello")
			},
			mutate: func(f *textinput.Field) {
				f.SetSelection(1, 3)
			},
		},
		{
			name:  "Focus",
			setup: func(f *textinput.Field) {},
			mutate: func(f *textinput.Field) {
				f.Focus()
			},
		},
		{
			name: "Blur",
			setup: func(f *textinput.Field) {
				f.Focus()
			},
			mutate: func(f *textinput.Field) {
				f.Blur()
			},
		},
		{
			name: "Undo",
			setup: func(f *textinput.Field) {
				f.ResetText("hello")
				f.ReplaceText("X", 0, 5)
			},
			mutate: func(f *textinput.Field) {
				f.Undo()
			},
		},
		{
			name: "Redo",
			setup: func(f *textinput.Field) {
				f.ResetText("hello")
				f.ReplaceText("X", 0, 5)
				f.Undo()
			},
			mutate: func(f *textinput.Field) {
				f.Redo()
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var f textinput.Field
			t.Cleanup(func() { f.Blur() })
			tc.setup(&f)
			before := f.ChangedAt()
			tc.mutate(&f)
			after := f.ChangedAt()
			if !after.After(before) {
				t.Errorf("%s: ChangedAt did not advance: before=%v after=%v", tc.name, before, after)
			}
		})
	}
}

func TestFieldChangedAtNoOpDoesNotAdvance(t *testing.T) {
	testCases := []struct {
		name   string
		setup  func(*textinput.Field)
		mutate func(*textinput.Field)
	}{
		{
			name: "SetSelectionWithCurrentSelection",
			setup: func(f *textinput.Field) {
				f.ResetText("hello")
				f.SetSelection(2, 4)
			},
			mutate: func(f *textinput.Field) {
				f.SetSelection(2, 4)
			},
		},
		{
			name: "SetSelectionClampingToCurrent",
			setup: func(f *textinput.Field) {
				f.ResetText("hello")
			},
			mutate: func(f *textinput.Field) {
				// (0,0) is already the selection; negatives clamp to (0,0).
				f.SetSelection(-5, -5)
			},
		},
		{
			name: "ReplaceTextEmptyZeroWidth",
			setup: func(f *textinput.Field) {
				f.ResetText("hello")
				f.SetSelection(3, 3)
			},
			mutate: func(f *textinput.Field) {
				f.ReplaceText("", 3, 3)
			},
		},
		{
			name: "FocusWhenAlreadyFocused",
			setup: func(f *textinput.Field) {
				f.Focus()
			},
			mutate: func(f *textinput.Field) {
				f.Focus()
			},
		},
		{
			name:  "BlurWhenNotFocused",
			setup: func(f *textinput.Field) {},
			mutate: func(f *textinput.Field) {
				f.Blur()
			},
		},
		{
			name:  "UndoWithEmptyHistory",
			setup: func(f *textinput.Field) {},
			mutate: func(f *textinput.Field) {
				f.Undo()
			},
		},
		{
			name: "RedoWithNothingToRedo",
			setup: func(f *textinput.Field) {
				f.ResetText("hello")
			},
			mutate: func(f *textinput.Field) {
				f.Redo()
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var f textinput.Field
			t.Cleanup(func() { f.Blur() })
			tc.setup(&f)
			before := f.ChangedAt()
			tc.mutate(&f)
			after := f.ChangedAt()
			if !after.Equal(before) {
				t.Errorf("%s: ChangedAt advanced on no-op: before=%v after=%v", tc.name, before, after)
			}
		})
	}
}

func TestFieldChangedAtReadOnlyMethodsDoNotAdvance(t *testing.T) {
	var f textinput.Field
	t.Cleanup(func() { f.Blur() })
	f.ResetText("hello")
	f.SetSelection(1, 3)
	f.Focus()

	prior := f.ChangedAt()

	_ = f.Text()
	_ = f.TextForRendering()
	_ = f.HasText()
	_ = f.TextLengthInBytes()
	_ = f.IsFocused()
	_, _ = f.Selection()
	_, _, _ = f.CompositionSelection()
	_ = f.CanUndo()
	_ = f.CanRedo()
	_ = f.UncommittedTextLengthInBytes()
	_ = f.Handled()
	var b strings.Builder
	_ = f.WriteText(&b)
	b.Reset()
	_ = f.WriteTextForRendering(&b)
	b.Reset()
	_ = f.WriteTextRange(&b, 0, f.TextLengthInBytes())

	if got := f.ChangedAt(); !got.Equal(prior) {
		t.Errorf("read-only methods advanced ChangedAt: before=%v after=%v", prior, got)
	}
}

func TestFieldWriteTextRange(t *testing.T) {
	var f textinput.Field
	t.Cleanup(func() { f.Blur() })
	f.Focus()
	f.ResetText("Hello, World!")
	// Force the underlying piece table to fragment.
	f.ReplaceText("Gopher", 7, 12) // "Hello, Gopher!"

	full := f.Text()
	if full != "Hello, Gopher!" {
		t.Fatalf("setup: got %q, want %q", full, "Hello, Gopher!")
	}

	cases := []struct {
		name  string
		start int
		end   int
		want  string
	}{
		{
			name:  "prefix",
			start: 0,
			end:   5,
			want:  "Hello",
		},
		{
			name:  "middle (post-fragmentation)",
			start: 7,
			end:   13,
			want:  "Gopher",
		},
		{
			name:  "whole",
			start: 0,
			end:   len(full),
			want:  full,
		},
		{
			name:  "empty",
			start: 5,
			end:   5,
			want:  "",
		},
		{
			name:  "clamp negative start",
			start: -5,
			end:   5,
			want:  "Hello",
		},
		{
			name:  "clamp end past length",
			start: 7,
			end:   1000,
			want:  "Gopher!",
		},
		{
			name:  "start > end",
			start: 8,
			end:   3,
			want:  "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var b strings.Builder
			if err := f.WriteTextRange(&b, c.start, c.end); err != nil {
				t.Fatalf("WriteTextRange failed: %v", err)
			}
			if got := b.String(); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}

	// Round-trip: every (start, end) in the buffer matches Text()[start:end].
	for start := 0; start <= len(full); start++ {
		for end := start; end <= len(full); end++ {
			var b strings.Builder
			if err := f.WriteTextRange(&b, start, end); err != nil {
				t.Fatalf("WriteTextRange(%d, %d) failed: %v", start, end, err)
			}
			if got, want := b.String(), full[start:end]; got != want {
				t.Errorf("range [%d, %d): got %q, want %q", start, end, got, want)
			}
		}
	}

	// After Undo, the range must reflect the prior text, not stale data.
	f.Undo()
	prior := f.Text()
	if prior != "Hello, World!" {
		t.Fatalf("after Undo: got %q, want %q", prior, "Hello, World!")
	}
	var b strings.Builder
	if err := f.WriteTextRange(&b, 7, 12); err != nil {
		t.Fatalf("WriteTextRange after Undo: %v", err)
	}
	if got := b.String(); got != "World" {
		t.Errorf("after Undo, range [7, 12): got %q, want %q", got, "World")
	}

	// Empty buffer.
	f.ResetText("")
	b.Reset()
	if err := f.WriteTextRange(&b, 0, 100); err != nil {
		t.Fatalf("WriteTextRange on empty: %v", err)
	}
	if got := b.String(); got != "" {
		t.Errorf("empty buffer: got %q, want %q", got, "")
	}
}

func TestFieldChangedAtStolenFocusAdvancesPreviousField(t *testing.T) {
	var a, b textinput.Field
	t.Cleanup(func() {
		a.Blur()
		b.Blur()
	})
	a.Focus()
	before := a.ChangedAt()
	b.Focus() // steals focus from a
	if !a.ChangedAt().After(before) {
		t.Errorf("a.ChangedAt did not advance when b stole focus: before=%v after=%v", before, a.ChangedAt())
	}
	if a.IsFocused() {
		t.Errorf("a should no longer be focused")
	}
	if !b.IsFocused() {
		t.Errorf("b should be focused")
	}
}

func TestFieldChangedAtStrictlyMonotonicUnderTightLoop(t *testing.T) {
	var f textinput.Field
	t.Cleanup(func() { f.Blur() })

	const n = 1000
	times := make([]time.Time, n)
	for i := range n {
		// Insert a character at the current end so each call is a real mutation.
		f.ReplaceText("a", i, i)
		times[i] = f.ChangedAt()
	}
	// Strict .After() across the whole sequence implies both ordering and uniqueness.
	for i := range n - 1 {
		if !times[i+1].After(times[i]) {
			t.Errorf("index %d: timestamps not strictly increasing: %v <= %v", i+1, times[i+1], times[i])
		}
	}
}
