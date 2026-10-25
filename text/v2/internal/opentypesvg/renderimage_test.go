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
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"net/url"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
)

func encodedImage(t *testing.T, size int, c color.RGBA) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	img.SetRGBA(0, 0, c)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func imageElement(data []byte) string {
	return `<image width="48" height="48" href="data:image/png;base64,` + base64.StdEncoding.EncodeToString(data) + `"/>`
}

func TestEmbeddedImageErrors(t *testing.T) {
	data := encodedImage(t, 1, color.RGBA{A: 255})
	oversized := bytes.Clone(data)
	binary.BigEndian.PutUint32(oversized[16:20], 4097)
	binary.BigEndian.PutUint32(oversized[29:33], crc32.ChecksumIEEE(oversized[12:29]))
	for _, tt := range []struct {
		name, body string
		want       error
	}{
		{
			name: "dimensions-before-decode",
			body: imageElement(oversized),
			want: opentypesvg.ErrLimit,
		},
		{
			name: "truncated-pixels",
			body: imageElement(data[:33]),
			want: opentypesvg.ErrImage,
		},
		{
			name: "malformed-base64",
			body: `<image width="1" height="1" href="data:image/png;base64,!"/>`,
			want: opentypesvg.ErrImage,
		},
		{
			name: "external",
			body: `<image width="1" height="1" href="https://example.invalid/a.png"/>`,
			want: opentypesvg.ErrUnsupported,
		},
		{
			name: "unsupported-format",
			body: `<image width="1" height="1" href="data:image/gif;base64,AA=="/>`,
			want: opentypesvg.ErrUnsupported,
		},
		{
			name: "mime-mismatch",
			body: strings.Replace(imageElement(data), "image/png", "image/jpeg", 1),
			want: opentypesvg.ErrImage,
		},
		{
			name: "encoded-limit",
			body: imageElement(make([]byte, 4<<20)),
			want: opentypesvg.ErrLimit,
		},
		{
			name: "inverse-scale",
			body: strings.Replace(imageElement(data), `width="48"`, `width="1e-30"`, 1),
			want: opentypesvg.ErrLimit,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := opentypesvg.PrepareRender([]byte(`<svg><rect width="4" height="4"/>`+tt.body+`</svg>`), opentypesvg.RenderOptions{
				UnitsPerEm: 48,
				Width:      48,
				Height:     48,
				Samples:    4,
			})
			if !errors.Is(err, tt.want) || plan != nil {
				t.Errorf("plan=%v error=%v, want nil and %v", plan, err, tt.want)
			}
		})
	}
}

func TestEmbeddedImageResources(t *testing.T) {
	// Every encoded source is distinct; each must consume the aggregate budget.
	var body strings.Builder
	for i := range 9 {
		body.WriteString(imageElement(encodedImage(t, 1024, color.RGBA{R: byte(i), A: 255})))
	}
	plan, err := opentypesvg.PrepareRender([]byte("<svg>"+body.String()+"</svg>"), opentypesvg.RenderOptions{
		UnitsPerEm: 48,
		Width:      48,
		Height:     48,
		Samples:    1,
	})
	if !errors.Is(err, opentypesvg.ErrLimit) || plan != nil {
		t.Errorf("aggregate images: plan=%v error=%v", plan, err)
	}

	body.Reset()
	for i := range 257 {
		body.WriteString(imageElement(encodedImage(t, 1, color.RGBA{R: byte(i), G: byte(i >> 8), A: 255})))
	}
	plan, err = opentypesvg.PrepareRender([]byte("<svg>"+body.String()+"</svg>"), opentypesvg.RenderOptions{
		UnitsPerEm: 48,
		Width:      48,
		Height:     48,
		Samples:    1,
	})
	if !errors.Is(err, opentypesvg.ErrLimit) || plan != nil {
		t.Errorf("upload count: plan=%v error=%v", plan, err)
	}
	data := encodedImage(t, 1024, color.RGBA{A: 255})
	// Repeated instances share the decoded asset and therefore fit the same budget.
	prepare(t, strings.Repeat(imageElement(data), 12))
	prepare(t, `<image width="8" height="8" href="data:image/png,`+url.PathEscape(string(data))+`"/>`)
	prepare(t, `<defs><clipPath id="c" clipPathUnits="objectBoundingBox"><rect width="1" height="1"/></clipPath></defs><g clip-path="url(#c)"><image width="8" height="8" opacity="0" href="bad"/><rect width="4" height="4"/></g>`)
}

func TestIsolationLimits(t *testing.T) {
	for _, tt := range []struct {
		name                           string
		depth, siblings, size, samples int
	}{
		{
			name:     "depth",
			depth:    17,
			siblings: 1,
			size:     8,
			samples:  1,
		},
		{
			name:     "simultaneous-area",
			depth:    5,
			siblings: 1,
			size:     256,
			samples:  8,
		},
		{
			name:     "cumulative-work",
			depth:    1,
			siblings: 130,
			size:     256,
			samples:  4,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			body := strings.Repeat(`<g opacity=".5">`, tt.depth) + `<rect width="8" height="8"/>` + strings.Repeat("</g>", tt.depth)
			plan, err := opentypesvg.PrepareRender([]byte("<svg>"+strings.Repeat(body, tt.siblings)+"</svg>"), opentypesvg.RenderOptions{
				UnitsPerEm: 48,
				Width:      tt.size,
				Height:     tt.size,
				Samples:    tt.samples,
			})
			if !errors.Is(err, opentypesvg.ErrLimit) || plan != nil {
				t.Errorf("plan=%v error=%v, want limit", plan, err)
			}
		})
	}
}
