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

package text_test

import (
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestSVGGlyphDocument(t *testing.T) {
	// A document shared by two glyphs, in the style of Noto Color
	// Emoji: glyph elements referencing shared defs via use, including
	// a transitive reference (grad -> grad2).
	const shared = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
	<defs>
		<path id="a" d="M0,0h10v10h-10z"/>
		<path id="b" d="M0,0h20v20h-20z"/>
		<linearGradient id="grad" xlink:href="#grad2"/>
		<linearGradient id="grad2"/>
	</defs>
	<g id="glyph1"><use xlink:href="#a"/><path fill="url(#grad)" d="M0,0z"/></g>
	<g id="glyph2"><use xlink:href="#b"/></g>
</svg>`

	for _, tc := range []struct {
		name     string
		source   string
		gid      uint32
		want     []string // substrings the result must contain
		wantNot  []string // substrings the result must not contain
		wantNil  bool
		wantSame bool // result must be the whole source
	}{
		{
			name:    "shared doc glyph1",
			source:  shared,
			gid:     1,
			want:    []string{`id="glyph1"`, `id="a"`, `id="grad"`, `id="grad2"`},
			wantNot: []string{`id="glyph2"`, `id="b"`},
		},
		{
			name:    "shared doc glyph2",
			source:  shared,
			gid:     2,
			want:    []string{`id="glyph2"`, `id="b"`},
			wantNot: []string{`id="glyph1"`, `id="a"`, `id="grad"`},
		},
		{
			name:    "shared doc missing glyph",
			source:  shared,
			gid:     3,
			wantNil: true,
		},
		{
			// References use XML entity references; they must resolve
			// against the decoded ids.
			name: "entity references in ids",
			source: `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
	<defs>
		<path id="a&amp;b" d="M0,0z"/>
		<linearGradient id="g&lt;1"/>
	</defs>
	<g id="glyph1"><use xlink:href="#a&amp;b"/><path fill="url(#g&lt;1)" d="M0,0z"/></g>
	<g id="glyph2"><path d="M0,0z"/></g>
</svg>`,
			gid:     1,
			want:    []string{`id="a&amp;b"`, `id="g&lt;1"`},
			wantNot: []string{`id="glyph2"`},
		},
		{
			name:     "id on the root element",
			source:   `<svg xmlns="http://www.w3.org/2000/svg" id="glyph7"><path d="M0,0z"/></svg>`,
			gid:      7,
			wantSame: true,
		},
		{
			name:     "no per-glyph structure",
			source:   `<svg xmlns="http://www.w3.org/2000/svg"><path d="M0,0z"/></svg>`,
			gid:      1,
			wantSame: true,
		},
		{
			// The root element never closes, but the glyph subtree is
			// complete, so extraction still works.
			name:   "unclosed root",
			source: `<svg><g id="glyph1"><path d="M0,0z"/></g>`,
			gid:    1,
			want:   []string{`id="glyph1"`},
		},
		{
			name:    "unclosed glyph subtree",
			source:  `<svg><g id="glyph1"><path d="M0,0z"/>`,
			gid:     1,
			wantNil: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := text.SVGGlyphDocument([]byte(tc.source), tc.gid)
			if tc.wantNil {
				if got != nil {
					t.Fatalf("got: %q, want: nil", got)
				}
				return
			}
			if got == nil {
				t.Fatal("got: nil")
			}
			if tc.wantSame {
				if string(got) != tc.source {
					t.Fatalf("got: %q, want the whole source", got)
				}
				return
			}
			for _, w := range tc.want {
				if !strings.Contains(string(got), w) {
					t.Errorf("result must contain %q but doesn't: %q", w, got)
				}
			}
			for _, w := range tc.wantNot {
				if strings.Contains(string(got), w) {
					t.Errorf("result must not contain %q but does: %q", w, got)
				}
			}
		})
	}
}
