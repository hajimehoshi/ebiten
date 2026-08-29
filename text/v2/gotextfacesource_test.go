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
	"sync"
	"testing"

	"golang.org/x/image/font/gofont/goregular"

	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestGlyphImageCacheConcurrent(t *testing.T) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	face := &text.GoTextFace{Source: src, Size: 16}
	glyphs := text.AppendLazyGlyphs(nil, "Hello, 世界!", face, nil)

	const goroutines = 8
	var wg sync.WaitGroup
	for range goroutines {
		wg.Go(func() {
			for _, gl := range glyphs {
				// A space or a control character has no image.
				if gl.ImageBounds.Empty() {
					continue
				}
				if gl.Image() == nil {
					t.Error("Image() returned nil for a glyph which should have an image")
				}
			}
		})
	}
	wg.Wait()
}
