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

package chunk

import (
	"strings"
	"testing"

	"github.com/go-text/typesetting/bidi"
)

func TestChunks(t *testing.T) {
	cases := []struct {
		name string
		text string
		want []string
	}{
		{
			name: "empty",
			text: "",
			want: []string{""},
		},
		{
			name: "no_boundary",
			text: "hello world",
			want: []string{"hello world"},
		},
		{
			name: "single_sentence_period_no_trailing_space",
			text: "Hello.",
			want: []string{"Hello."},
		},
		{
			name: "two_sentences_period",
			text: "Hello. World.",
			want: []string{"Hello.", " World."},
		},
		{
			name: "two_sentences_bang",
			text: "Hello! World.",
			want: []string{"Hello!", " World."},
		},
		{
			name: "lowercase_after_period_suppresses",
			text: "etc. and so on",
			want: []string{"etc. and so on"},
		},
		{
			// The chunker doesn't know "etc." is an abbreviation; the
			// suppression is purely about the case of the next
			// non-whitespace character. Uppercase after the period
			// breaks even when the preceding word looks like an
			// abbreviation.
			name: "abbreviation_then_uppercase_breaks",
			text: "etc. Apples",
			want: []string{"etc.", " Apples"},
		},
		{
			name: "bang_breaks_even_before_lowercase",
			text: "Hello! and so on",
			want: []string{"Hello!", " and so on"},
		},
		{
			name: "cluster_question_bang",
			text: "What?! Hello.",
			want: []string{"What?!", " Hello."},
		},
		{
			name: "ellipsis_lowercase_follows",
			text: "... and so on",
			want: []string{"... and so on"},
		},
		{
			name: "ellipsis_uppercase_follows",
			text: "... Hello.",
			want: []string{"...", " Hello."},
		},
		{
			name: "initials_no_internal_split",
			text: "I went to U.S. Army.",
			want: []string{"I went to U.S.", " Army."},
		},
		{
			name: "cjk_sentence_terminator_no_space",
			text: "今日は雨だ。明日も雨だ。",
			want: []string{"今日は雨だ。", "明日も雨だ。"},
		},
		{
			// LTR paragraph with one embedded RTL word: bidi
			// segmentation produces an LTR run, an RTL run for the
			// Arabic word, then another LTR run that the sentence
			// chunker further splits at the period.
			name: "ltr_paragraph_with_embedded_rtl_run",
			text: "I read كتاب yesterday. Today too.",
			want: []string{"I read ", "كتاب", " yesterday.", " Today too."},
		},
		{
			// UAX #29 WB4: a combining mark immediately after a
			// sentence terminator attaches to the terminator and
			// must not be split off. The chunk boundary slides
			// past the mark so the next chunk begins with " ".
			name: "combining_mark_after_sterm_absorbed",
			text: "Hello?\u0301 World.",
			want: []string{"Hello?\u0301", " World."},
		},
		{
			// ZWJ after a sentence terminator: same WB4 protection.
			name: "zwj_after_sterm_absorbed",
			text: "Hello!\u200d World.",
			want: []string{"Hello!\u200d", " World."},
		},
		{
			// The walk terminates at the first line break: text past
			// the LF is not covered by any chunk.
			name: "lf_terminates_walk",
			text: "Hello.\nWorld.",
			want: []string{"Hello."},
		},
		{
			// Same behaviour for CR and the Unicode line separators.
			name: "cr_terminates_walk",
			text: "Hello. World.\rExtra.",
			want: []string{"Hello.", " World."},
		},
		{
			// U+2028 LINE SEPARATOR also terminates the walk.
			name: "ls_terminates_walk",
			text: "Hello. World.\u2028Extra.",
			want: []string{"Hello.", " World."},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AppendChunks(nil, tc.text, 0)
			gotStrs := make([]string, len(got))
			for i, r := range got {
				gotStrs[i] = tc.text[r.Start:r.End]
			}
			if len(gotStrs) != len(tc.want) {
				t.Fatalf("chunk count = %d, want %d; got %q", len(gotStrs), len(tc.want), gotStrs)
			}
			for i := range gotStrs {
				if gotStrs[i] != tc.want[i] {
					t.Errorf("chunk %d = %q, want %q", i, gotStrs[i], tc.want[i])
				}
			}
			// Chunks must cover the want-bytes contiguously without
			// gaps. Bytes past the first line break (if any) are
			// intentionally not covered.
			var pos, wantLen int
			for _, s := range tc.want {
				wantLen += len(s)
			}
			for _, r := range got {
				if r.Start != pos {
					t.Errorf("chunk gap at %d (chunk starts at %d)", pos, r.Start)
				}
				pos = r.End
			}
			if pos != wantLen {
				t.Errorf("chunks cover %d bytes, want %d", pos, wantLen)
			}
		})
	}
}

// TestChunks_WhitespaceFallback verifies that a long no-terminator
// span is split at whitespace once the cumulative chunk size crosses
// [fallbackBytes].
func TestChunks_WhitespaceFallback(t *testing.T) {
	// Build "word word word ..." with no sentence terminators, long
	// enough that the fallback fires several times.
	word := strings.Repeat("a", 31) + " " // 32 bytes per word
	text := strings.Repeat(word, 3000)    // ~96 KB
	got := AppendChunks(nil, text, 0)
	if len(got) < 4 {
		t.Fatalf("expected fallback to fire multiple times; got %d chunks for %d bytes", len(got), len(text))
	}
	for i, r := range got {
		size := r.End - r.Start
		// Every chunk except possibly the last must end at a space
		// (the fallback trigger) and not exceed the threshold by
		// more than one word.
		if i < len(got)-1 {
			if text[r.End-1] != ' ' {
				t.Errorf("chunk %d does not end at a space (ends with %q)", i, text[r.End-1])
			}
			if size > fallbackBytes+len(word) {
				t.Errorf("chunk %d size %d exceeds threshold %d by more than one word", i, size, fallbackBytes)
			}
		}
	}
}

// TestChunks_Levels verifies that each chunk carries the correct
// bidi embedding level so the composition layer can apply UAX #9 L2.
func TestChunks_Levels(t *testing.T) {
	cases := []struct {
		name           string
		text           string
		paragraphLevel bidi.Level
		wantTexts      []string
		wantLevels     []bidi.Level
	}{
		{
			// Pure LTR under LTR base: every chunk is level 0.
			name:       "pure_ltr_under_ltr",
			text:       "Hello. World.",
			wantTexts:  []string{"Hello.", " World."},
			wantLevels: []bidi.Level{0, 0},
		},
		{
			// RTL text on an LTR-paragraph treatment. Internal
			// neutrals between Arabic letters become R by UAX #9 N1,
			// so the bidi pass yields a single L1 run covering
			// "اب. كد"; the chunker emits that whole run as one chunk
			// because it disagrees with the LTR base. The trailing
			// period — with no strong character to its right — takes
			// the paragraph's embedding direction (L) by N2, splitting
			// off as a final level-0 chunk.
			name:       "rtl_text_under_ltr_base",
			text:       "اب. كد.",
			wantTexts:  []string{"اب. كد", "."},
			wantLevels: []bidi.Level{1, 0},
		},
		{
			// LTR paragraph with embedded RTL word: levels alternate
			// 0 / 1 / 0 across the runs.
			name:       "ltr_with_embedded_rtl",
			text:       "I read كتاب yesterday.",
			wantTexts:  []string{"I read ", "كتاب", " yesterday."},
			wantLevels: []bidi.Level{0, 1, 0},
		},
		{
			// Pure RTL under RTL base: agrees with the paragraph, so
			// the chunker sentence-chunks the line just like LTR text
			// under an LTR base. The Arabic full stop U+06D4 is a
			// recognized STerm.
			name:           "pure_rtl_under_rtl",
			text:           "اب۔ كد۔",
			paragraphLevel: 1,
			wantTexts:      []string{"اب۔", " كد۔"},
			wantLevels:     []bidi.Level{1, 1},
		},
		{
			// LTR text declared as an RTL paragraph (mirror of
			// rtl_text_under_ltr_base). The body resolves to level 2
			// (LTR embedded in RTL base) and is emitted whole
			// because it disagrees with paragraph level 1; the
			// trailing period — with no strong character to its
			// right — takes the paragraph's embedding direction R by
			// N2, splitting off as a final level-1 chunk.
			name:           "ltr_text_under_rtl_base",
			text:           "Hello. World.",
			paragraphLevel: 1,
			wantTexts:      []string{"Hello. World", "."},
			wantLevels:     []bidi.Level{2, 1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := AppendChunks(nil, tc.text, tc.paragraphLevel)
			if len(got) != len(tc.wantTexts) {
				t.Fatalf("chunk count = %d, want %d", len(got), len(tc.wantTexts))
			}
			for i, r := range got {
				if s := tc.text[r.Start:r.End]; s != tc.wantTexts[i] {
					t.Errorf("chunk %d text = %q, want %q", i, s, tc.wantTexts[i])
				}
				if r.Level != tc.wantLevels[i] {
					t.Errorf("chunk %d level = %d, want %d", i, r.Level, tc.wantLevels[i])
				}
			}
		})
	}
}

// TestAppendChunks_PreservesDst verifies the append-style contract:
// existing content in dst is left untouched and the new chunks are
// appended to it.
func TestAppendChunks_PreservesDst(t *testing.T) {
	sentinel := Chunk{Start: 99, End: 99, Level: 99}
	dst := []Chunk{sentinel}
	got := AppendChunks(dst, "Hello. World.", 0)
	if len(got) < 2 {
		t.Fatalf("got %d chunks, want at least 2 (sentinel + appended)", len(got))
	}
	if got[0] != sentinel {
		t.Errorf("got[0] = %+v, want sentinel %+v", got[0], sentinel)
	}
}

// TestChunks_FallbackAbsorbsExtend verifies that UAX #29 WB4
// protection applies at the whitespace-fallback split: a combining
// mark immediately after the fallback space must be absorbed into the
// preceding chunk, not orphaned at the start of the next chunk.
func TestChunks_FallbackAbsorbsExtend(t *testing.T) {
	// The run of 'a' is long enough that i-chunkStart >= fallbackBytes
	// holds at the space that follows. The next byte is the start of
	// a combining acute (U+0301), which the fallback must absorb into
	// the first chunk.
	first := strings.Repeat("a", fallbackBytes) + " "
	text := first + "\u0301" + "bcd"
	got := AppendChunks(nil, text, 0)
	if len(got) != 2 {
		t.Fatalf("got %d chunks, want 2", len(got))
	}
	// The first chunk must include the 'a' run, the space, and the
	// absorbed combining mark.
	if want := fallbackBytes + 1 + len("\u0301"); got[0].End != want {
		t.Errorf("first chunk ends at %d, want %d", got[0].End, want)
	}
}
