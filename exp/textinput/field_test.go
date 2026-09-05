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
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/exp/textinput"
)

func TestFieldConcurrentUpdateAndRead(t *testing.T) {
	texts := []string{
		"",
		"a",
		"hello\nworld",
		strings.Repeat("あ", 32),
	}

	var f textinput.Field
	done := make(chan struct{})

	var wg sync.WaitGroup
	wg.Go(func() {
		for i := 0; ; i++ {
			select {
			case <-done:
				return
			default:
			}
			text := texts[i%len(texts)]
			f.SetTextAndSelection(text, len(text), len(text))
			f.SetSelection(0, len(text))
		}
	})
	defer func() {
		close(done)
		wg.Wait()
	}()

	for range 100000 {
		if got := f.Text(); !slices.Contains(texts, got) {
			t.Fatalf("f.Text(): got %q; want one of %q", got, texts)
		}
		if got := f.TextForRendering(); !slices.Contains(texts, got) {
			t.Fatalf("f.TextForRendering(): got %q; want one of %q", got, texts)
		}
		start, end := f.Selection()
		if start > end {
			t.Fatalf("f.Selection(): got (%d, %d); want start <= end", start, end)
		}
		if end > len(texts[len(texts)-1]) {
			t.Fatalf("f.Selection(): got end %d; want at most %d", end, len(texts[len(texts)-1]))
		}
	}
}
