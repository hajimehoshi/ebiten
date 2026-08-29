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
	"bytes"
	"testing"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func limitedFaceTestFace(t *testing.T) *text.GoTextFace {
	t.Helper()
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	return &text.GoTextFace{Source: src, Size: 16}
}

func lineBreaks() []string {
	return []string{"\n", "\v", "\f", "\r", "\u0085", "\u2028", "\u2029", "\r\n"}
}

func TestLimitedFaceAdvanceAtNewline(t *testing.T) {
	base := limitedFaceTestFace(t)
	l := text.NewLimitedFace(base)
	l.AddUnicodeRange('A', 'D')

	for _, lineBreak := range lineBreaks() {
		str := "AB" + lineBreak + "CD"
		for idx := 2; idx <= len(str); idx++ {
			got := text.AdvanceAt(str, idx, l)
			want := text.AdvanceAt(str, idx, base)
			if got != want {
				t.Errorf("%q index %d: LimitedFace.AdvanceAt got: %v, want: %v", str, idx, got, want)
			}
		}
	}

	// 'Z' is not in the allowed range, so the first line is filtered into
	// "a\uFFFDb" and the index translation is exercised.
	l2 := text.NewLimitedFace(base)
	l2.AddUnicodeRange('a', 'd')
	for _, lineBreak := range lineBreaks() {
		str := "aZb" + lineBreak + "cd"
		for idx := 0; idx <= len(str); idx++ {
			got := text.AdvanceAt(str, idx, l2)
			want := text.AdvanceAt("aZb", min(idx, 3), l2)
			if got != want {
				t.Errorf("%q index %d: LimitedFace.AdvanceAt got: %v, want: %v", str, idx, got, want)
			}
		}
	}
}

func TestMultiFaceAdvanceAtNewline(t *testing.T) {
	base := limitedFaceTestFace(t)
	l := text.NewLimitedFace(base)
	l.AddUnicodeRange('A', 'A')
	m, err := text.NewMultiFace(l, base)
	if err != nil {
		t.Fatal(err)
	}

	want := text.AdvanceAt("A", 1, base)
	for _, lineBreak := range lineBreaks() {
		str := "A" + lineBreak + "A"
		for idx := 1; idx <= len(str); idx++ {
			got := text.AdvanceAt(str, idx, m)
			if got != want {
				t.Errorf("%q index %d: MultiFace.AdvanceAt got: %v, want: %v", str, idx, got, want)
			}
		}
	}
}
