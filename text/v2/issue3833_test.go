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

// Regression test for https://github.com/hajimehoshi/ebiten/issues/3833.
func TestAdvancePreservesTextAfterSeparators(t *testing.T) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	face := &text.GoTextFace{Source: src, Size: 24}
	for _, tc := range []struct{ name, s string }{
		{"FS", "abc\x1cdef"},
		{"GS", "abc\x1ddef"},
		{"RS", "abc\x1edef"},
	} {
		got := text.Advance(tc.s, face)
		want := text.Advance("abc", face) + text.Advance("\x1c", face) + text.Advance("def", face)
		t.Logf("%s: got=%.4f want=%.4f", tc.name, got, want)
		if got != want {
			t.Errorf("%s: got %.4f, want %.4f (text after separator lost)", tc.name, got, want)
		}
	}
}
