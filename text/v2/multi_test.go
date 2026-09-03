// Copyright 2023 The Ebitengine Authors
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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/bitmapfont/v4"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestMultiFace(t *testing.T) {
	faces := []text.Face{text.NewGoXFace(bitmapfont.Face)}
	f, err := text.NewMultiFace(faces...)
	if err != nil {
		t.Fatal(err)
	}
	img := ebiten.NewImage(30, 30)
	text.Draw(img, "Hello", f, nil)

	// Confirm that the given slice doesn't cause crash.
	faces[0] = nil
	text.Draw(img, "World", f, nil)
}

func TestMultiFaceFallback(t *testing.T) {
	enFaceSource, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	enFace := &text.GoTextFace{
		Source: enFaceSource,
		Size:   10,
	}
	multiFace, err := text.NewMultiFace(enFace)
	if err != nil {
		t.Fatal(err)
	}

	// If all the faces in a MultiFace doesn't have a glyph, the last face should be used.
	str := "あ"
	got := text.AppendGlyphs(nil, str, multiFace, nil)
	want := text.AppendGlyphs(nil, str, enFace, nil)
	if len(got) != len(want) {
		t.Errorf("got: %d, want: %d", len(got), len(want))
	}
}

// Issue #3284
func TestMultiFaceAdvance(t *testing.T) {
	f := text.NewGoXFace(bitmapfont.Face)
	f1 := text.NewLimitedFace(f)
	f1.AddUnicodeRange(0x0000, 0x007F)
	f2 := text.NewLimitedFace(f)
	f2.AddUnicodeRange(0x0080, 0xFFFF)
	m, err := text.NewMultiFace(f1, f2)
	if err != nil {
		t.Fatal(err)
	}
	for _, str := range []string{
		"",
		"abc",
		"aあb",
		"\x80",
		"a\x80b",
		"a\x80\x80b",
		"a\x80b\x80",
		"a\x80b\x80c",
	} {
		t.Run(fmt.Sprintf("str=%q", str), func(t *testing.T) {
			got := text.Advance(str, m)
			want := text.Advance(str, f)
			if got != want {
				t.Errorf("got: %f, want: %f", got, want)
			}
		})
	}
}

func TestMultiFaceAfterFaceUpdate(t *testing.T) {
	enFaceSource, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}

	high := text.NewLimitedFace(text.NewGoXFace(bitmapfont.Face))
	high.AddUnicodeRange('a', 'a')
	low := text.NewLimitedFace(&text.GoTextFace{
		Source: enFaceSource,
		Size:   10,
	})
	low.AddUnicodeRange('a', 'z')

	m, err := text.NewMultiFace(high, low)
	if err != nil {
		t.Fatal(err)
	}

	const str = "ab"
	// 'a' is rendered by high and 'b' is rendered by low here.
	before := text.Advance(str, m)

	// A face's glyphs can change even after a MultiFace creation, and the change must be reflected.
	high.AddUnicodeRange('b', 'b')
	got := text.Advance(str, m)
	if want := text.Advance(str, high); got != want {
		t.Errorf("got: %f, want: %f", got, want)
	}
	if before == got {
		t.Errorf("the advance before and after a face update must differ: %f", got)
	}
}

func TestMultiFaceAfterGoTextFaceSourceUpdate(t *testing.T) {
	goregularSource, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	mplusData, err := os.ReadFile(filepath.Join("testdata", "MPLUS1p-Regular.ttf"))
	if err != nil {
		t.Fatal(err)
	}
	mplusSource, err := text.NewGoTextFaceSource(bytes.NewReader(mplusData))
	if err != nil {
		t.Fatal(err)
	}

	// goTextFace has a glyph for 'a' but not for 'あ', so the bitmap font is used for 'あ'.
	goTextFace := &text.GoTextFace{
		Source: goregularSource,
		Size:   10,
	}
	m, err := text.NewMultiFace(goTextFace, text.NewGoXFace(bitmapfont.Face))
	if err != nil {
		t.Fatal(err)
	}

	const str = "aあ"
	before := text.Advance(str, m)

	// The source of a GoTextFace can be changed even after a MultiFace creation,
	// and the change must be reflected.
	goTextFace.Source = mplusSource
	got := text.Advance(str, m)
	if want := text.Advance(str, goTextFace); got != want {
		t.Errorf("got: %f, want: %f", got, want)
	}
	if before == got {
		t.Errorf("the advance before and after a source update must differ: %f", got)
	}
}
