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
	"testing"

	"github.com/go-text/typesetting/bidi"
)

// Regression test for https://github.com/hajimehoshi/ebiten/issues/3833:
// text after bidi paragraph separators must not be dropped from chunks.
func TestChunkPreservesTextAfterSeparators(t *testing.T) {
	for _, text := range []string{"abc\x1cdef", "abc\x1ddef", "abc\x1edef"} {
		got := AppendChunks(nil, text, 0)
		covered := 0
		for _, ch := range got {
			covered += ch.End - ch.Start
		}
		if covered != len(text) {
			t.Errorf("%q: chunks %v cover %d of %d bytes", text, got, covered, len(text))
		}
	}
	// Bidi-aware path (Hebrew forces the non-fast path).
	got := AppendChunks(nil, "אbc\x1cdef", 0)
	covered := 0
	for _, ch := range got {
		covered += ch.End - ch.Start
	}
	t.Logf("rtl chunks: %v", got)
	if covered != len("אbc\x1cdef") {
		t.Errorf("rtl: chunks %v cover %d of %d bytes", got, covered, len("אbc\x1cdef"))
	}
	var _ = bidi.LeftToRight
}
