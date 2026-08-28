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

func TestLimitedFaceAdvanceAtNewline(t *testing.T) {
	base := limitedFaceTestFace(t)
	l := text.NewLimitedFace(base)
	l.AddUnicodeRange('A', 'D')

	for _, idx := range []int{2, 3, 5} {
		got := text.AdvanceAt("AB\nCD", idx, l)
		want := text.AdvanceAt("AB\nCD", idx, base)
		if got != want {
			t.Errorf("index %d: LimitedFace.AdvanceAt got: %v, want: %v", idx, got, want)
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

	got := text.AdvanceAt("A\nA", 3, m)
	want := text.AdvanceAt("A", 1, base)
	if got != want {
		t.Errorf("AdvanceAt: got: %v, want: %v", got, want)
	}
}
