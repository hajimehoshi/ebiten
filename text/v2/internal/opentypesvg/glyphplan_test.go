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

package opentypesvg_test

import (
	"bytes"
	"compress/gzip"
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
)

func TestPreparedGlyphBounds(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       image.Rectangle
	}{
		{
			name: "transformed curve",
			body: `<path transform="translate(-10 20) scale(2)" d="M0 0Q20 -20 40 0Z"/>`,
			want: image.Rect(-10, 0, 70, 20),
		},
		{
			name: "clip union and intersection",
			body: `<defs><clipPath id="a"><rect x="-5" width="10" height="20"/><rect x="10" width="10" height="20"/></clipPath><clipPath id="b"><rect x="-20" y="5" width="50" height="10"/></clipPath></defs><g clip-path="url(#b)"><rect x="-50" y="-50" width="100" height="100" clip-path="url(#a)"/></g>`,
			want: image.Rect(-5, 5, 20, 15),
		},
		{
			name: "empty opacity",
			body: `<g opacity="0"><rect width="1000" height="1000"/></g>`,
		},
		{
			name: "empty clip",
			body: `<defs><clipPath id="c"/></defs><rect width="20" height="20" clip-path="url(#c)"/>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := opentypesvg.ParseRenderDocument([]byte(`<svg viewBox="0 0 100 100">` + tc.body + `</svg>`))
			if err != nil {
				t.Fatal(err)
			}
			_, got, err := doc.PrepareGlyph(opentypesvg.RenderOptions{
				UnitsPerEm: 100,
				Samples:    4,
				Foreground: color.NRGBA{A: 255},
			})
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("bounds: %v, want %v", got, tc.want)
			}
		})
	}
}

func TestSVGDecodeBudgets(t *testing.T) {
	compress := func(data []byte) []byte {
		t.Helper()
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		return buf.Bytes()
	}
	// A large legitimate shared document fits, including gzip trailer validation.
	legitimate := bytes.Repeat([]byte(" "), 14<<20)
	decoded, err := opentypesvg.Decode(compress(legitimate))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, legitimate) {
		t.Error("large document changed")
	}
	// A tiny gzip input must not escape the decoded budget. No allocator or
	// scheduler state is involved in this assertion.
	oversized := compress(bytes.Repeat([]byte(" "), (32<<20)+1))
	if _, err := opentypesvg.Decode(oversized); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Errorf("decoded limit: %v", err)
	}
	compressed := make([]byte, (8<<20)+1)
	compressed[0], compressed[1] = 0x1f, 0x8b
	if _, err := opentypesvg.Decode(compressed); !errors.Is(err, opentypesvg.ErrLimit) {
		t.Errorf("compressed limit: %v", err)
	}
	broken := compress([]byte("<svg/>"))
	broken[len(broken)-8] ^= 1
	if _, err := opentypesvg.Decode(broken); !errors.Is(err, opentypesvg.ErrCompression) {
		t.Errorf("corrupt trailer: %v", err)
	}
}
