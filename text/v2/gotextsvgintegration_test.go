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
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func svgTestFont(t testing.TB, document string, compressed bool, extraTables ...opentype.Table) *text.GoTextFaceSource {
	t.Helper()
	loader, err := opentype.NewLoader(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	f, err := font.NewFont(loader)
	if err != nil {
		t.Fatal(err)
	}
	var gids []font.GID
	for _, r := range "ABCDEF" {
		gid, _ := f.Cmap.Lookup(r)
		gids = append(gids, gid)
		document = strings.ReplaceAll(document, "GLYPH"+string(r), fmt.Sprintf("glyph%d", gid))
	}
	source := []byte(document)
	if compressed {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(source); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		source = buf.Bytes()
	}
	// Multiple records reference the same byte range, exercising document identity.
	svg := make([]byte, 12+12*len(gids))
	binary.BigEndian.PutUint32(svg[2:], 10)
	binary.BigEndian.PutUint16(svg[10:], uint16(len(gids)))
	for i, gid := range gids {
		r := svg[12+12*i:]
		binary.BigEndian.PutUint16(r, uint16(gid))
		binary.BigEndian.PutUint16(r[2:], uint16(gid))
		binary.BigEndian.PutUint32(r[4:], uint32(2+12*len(gids)))
		binary.BigEndian.PutUint32(r[8:], uint32(len(source)))
	}
	svg = append(svg, source...)
	tables := []opentype.Table{{
		Tag:     opentype.MustNewTag("SVG "),
		Content: svg,
	}}
	tables = append(tables, extraTables...)
	for _, tag := range loader.Tables() {
		raw, err := loader.RawTable(tag)
		if err != nil {
			t.Fatal(err)
		}
		tables = append(tables, opentype.Table{
			Tag:     tag,
			Content: raw,
		})
	}
	src, err := text.NewGoTextFaceSource(bytes.NewReader(opentype.WriteTTF(tables)))
	if err != nil {
		t.Fatal(err)
	}
	return src
}

func TestSVGFaceColorsAndLazySnapshot(t *testing.T) {
	src := svgTestFont(t, `<svg viewBox="0 0 100 100"><defs><rect id="tile" width="20" height="20"/></defs><g id="GLYPHA"><use href="#tile" fill="currentColor"/><use href="#tile" x="20" fill="var(--color0)"/><use href="#tile" x="40" fill="blue"/></g><g id="GLYPHB"><use href="#tile" fill="currentColor"/></g></svg>`, true)
	palette := []color.NRGBA{{
		G: 255,
		A: 128,
	}}
	face := &text.GoTextFace{
		Source:        src,
		Size:          100,
		SVGForeground: color.NRGBA{R: 255, A: 255},
		SVGPalette:    palette,
	}
	old := text.AppendLazyGlyphs(nil, "AB", face, nil)
	face.SVGForeground = color.NRGBA{B: 255, A: 255}
	palette[0] = color.NRGBA{R: 255, A: 255}
	changed := text.AppendGlyphs(nil, "A", face, nil)
	if got := changed[0].Image.At(5, 5); got != (color.RGBA{B: 255, A: 255}) {
		t.Errorf("changed foreground: %v", got)
	}
	if got := changed[0].Image.At(25, 5); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("changed palette: %v", got)
	}
	face.Size = 50
	smaller := text.AppendGlyphs(nil, "A", face, nil)[0]
	if smaller.Image.Bounds().Dx() >= changed[0].Image.Bounds().Dx() {
		t.Error("size reused the larger glyph")
	}
	img := old[0].Image()
	for _, tc := range []struct {
		x    int
		want color.RGBA
	}{
		{
			x:    5,
			want: color.RGBA{R: 255, A: 255},
		},
		{
			x:    25,
			want: color.RGBA{G: 128, A: 128},
		},
		{
			x:    45,
			want: color.RGBA{B: 255, A: 255},
		},
	} {
		if got := img.At(tc.x, 5); got != tc.want {
			t.Errorf("snapshot at %d: %v, want %v", tc.x, got, tc.want)
		}
	}
	if got := old[1].Image().At(5, 5); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("shared document glyph B: %v", got)
	}
	face.Size = 100
	face.SVGForeground = color.NRGBA{R: 255, A: 255}
	face.SVGPalette = []color.NRGBA{{
		G: 255,
		A: 128,
	}}
	dst := ebiten.NewImage(80, 30)
	defer dst.Deallocate()
	op := &text.DrawOptions{}
	op.GeoM.Translate(0, -face.Metrics().HAscent)
	op.ColorScale.ScaleAlpha(0.5)
	text.Draw(dst, "A", face, op)
	for _, tc := range []struct {
		x    int
		want color.RGBA
	}{
		{
			x:    5,
			want: color.RGBA{R: 128, A: 128},
		},
		{
			x:    25,
			want: color.RGBA{G: 64, A: 64},
		},
		{
			x:    45,
			want: color.RGBA{B: 128, A: 128},
		},
	} {
		if got := dst.At(tc.x, 5).(color.RGBA); colorDiff(got, tc.want) > 1 {
			t.Errorf("Draw alpha at %d: %v, want %v", tc.x, got, tc.want)
		}
	}
}

func TestSVGInkBoundsAndCulling(t *testing.T) {
	src := svgTestFont(t, `<svg viewBox="0 0 100 100"><g id="GLYPHA" transform="translate(-200 300)"><rect width="20" height="10" fill="red"/></g></svg>`, false)
	face := &text.GoTextFace{
		Source: src,
		Size:   100,
	}
	glyph := text.AppendLazyGlyphs(nil, "A", face, nil)[0]
	baseline := int(face.Metrics().HAscent)
	want := image.Rect(-200, baseline+300, -180, baseline+310)
	if glyph.ImageBounds != want {
		t.Errorf("bounds: %v, want %v", glyph.ImageBounds, want)
	}
	dst := ebiten.NewImage(30, 20)
	defer dst.Deallocate()
	op := &text.DrawOptions{}
	op.GeoM.Translate(205, -float64(baseline)-300)
	text.Draw(dst, "A", face, op)
	if got := dst.At(10, 5); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("visible artwork with far-offscreen origin was culled: %v", got)
	}
}

func TestSVGEmptyAndFallback(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		empty      bool
	}{
		{
			name:  "empty",
			body:  `<g id="GLYPHA"/>`,
			empty: true,
		},
		{
			name: "missing",
			body: `<g id="glyph65535"><rect width="10" height="10"/></g>`,
		},
		{
			name: "malformed",
			body: `<g id="GLYPHA"><path d="M0 0L"/></g>`,
		},
		{
			name: "unsupported",
			body: `<g id="GLYPHA" filter="url(#effect)"><rect width="10" height="10"/></g>`,
		},
		{
			name: "image cache budget",
			body: `<g id="GLYPHA"><rect width="30000" height="30000"/></g>`,
		},
		{
			name: "limit",
			body: `<g id="GLYPHA"><rect width="1000000" height="1000000"/></g>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			face := &text.GoTextFace{
				Source: svgTestFont(t, "<svg>"+tc.body+"</svg>", false),
				Size:   40,
			}
			glyph := text.AppendLazyGlyphs(nil, "A", face, nil)[0]
			if tc.empty {
				if !glyph.ImageBounds.Empty() || glyph.Image() != nil {
					t.Error("empty artwork used fallback")
				}
				return
			}
			if glyph.Image() == nil || glyph.Colored() {
				t.Error("failed SVG did not use a monochrome outline")
			}
		})
	}
}

func TestSVGRealRecordThroughText(t *testing.T) {
	data, err := os.ReadFile("internal/opentypesvg/testdata/real/twemoji-839-839.svg")
	if err != nil {
		t.Fatal(err)
	}
	src := svgTestFont(t, strings.ReplaceAll(string(data), "glyph839", "GLYPHA"), true)
	face := &text.GoTextFace{
		Source: src,
		Size:   64,
	}
	g := text.AppendGlyphs(nil, "A", face, nil)[0]
	if !g.Colored || g.Image == nil {
		t.Fatal("real record did not produce an SVG image")
	}
	var occupied int
	b := g.Image.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := g.Image.At(x, y).(color.RGBA)
			if c.A == 255 {
				occupied++
				if c != (color.RGBA{R: 0x31, G: 0x37, B: 0x3d, A: 255}) {
					t.Errorf("opaque sunglasses pixel: %v", c)
					return
				}
			}
		}
	}
	if occupied == 0 {
		t.Error("real record image is empty")
	}
}

func TestSVGSubpixelCache(t *testing.T) {
	face := &text.GoTextFace{
		Source: svgTestFont(t, `<svg viewBox="0 0 100 100"><g id="GLYPHA"><rect x="-1" width="40" height="40"/></g></svg>`, false),
		Size:   16,
	}
	glyphs := text.AppendLazyGlyphs(nil, "AA", face, nil)
	var pixels [2][]byte
	for i, glyph := range glyphs {
		img := glyph.Image()
		if img == nil {
			t.Fatal("missing SVG image")
		}
		pixels[i] = make([]byte, 4*img.Bounds().Dx()*img.Bounds().Dy())
		img.ReadPixels(pixels[i])
	}
	if bytes.Equal(pixels[0], pixels[1]) {
		t.Error("fractional positioning did not change glyph coverage")
	}
}

func TestSVGDefaultCPAL(t *testing.T) {
	// One straight RGBA palette entry, encoded in CPAL's BGRA order.
	cpal := []byte{0, 0, 0, 1, 0, 1, 0, 1, 0, 0, 0, 14, 0, 0, 0, 255, 0, 128}
	face := &text.GoTextFace{
		Source: svgTestFont(t, `<svg viewBox="0 0 100 100"><rect width="20" height="20" fill="var(--color0, red)"/></svg>`, false, opentype.Table{
			Tag:     opentype.MustNewTag("CPAL"),
			Content: cpal,
		}),
		Size: 100,
	}
	glyph := text.AppendGlyphs(nil, "A", face, nil)[0]
	if got := glyph.Image.At(5, 5); got != (color.RGBA{G: 128, A: 128}) {
		t.Errorf("default CPAL entry: %v", got)
	}
	face.SVGPalette = []color.NRGBA{}
	glyph = text.AppendGlyphs(nil, "A", face, nil)[0]
	if got := glyph.Image.At(5, 5); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("explicit empty palette fallback: %v", got)
	}
}

func TestSVGZeroSize(t *testing.T) {
	face := &text.GoTextFace{
		Source: svgTestFont(t, `<svg><rect width="10" height="10"/></svg>`, false),
	}
	glyph := text.AppendLazyGlyphs(nil, "A", face, nil)[0]
	if !glyph.ImageBounds.Empty() || glyph.Image() != nil {
		t.Error("zero-size SVG produced artwork")
	}
}

func TestSVGLayoutBeyondPlanCache(t *testing.T) {
	var doc strings.Builder
	doc.WriteString(`<svg viewBox="0 0 100 100">`)
	for _, r := range "ABCDEF" {
		fmt.Fprintf(&doc, `<rect id="GLYPH%c" width="20" height="20" fill="red"/>`, r)
	}
	doc.WriteString(`</svg>`)
	face := &text.GoTextFace{
		Source: svgTestFont(t, doc.String(), true),
		Size:   40,
	}
	lazy := text.AppendLazyGlyphs(nil, "ABCDEF", face, nil)
	dst := ebiten.NewImage(256, 80)
	defer dst.Deallocate()
	text.Draw(dst, "ABCDEF", face, nil)
	expected := make([]byte, 256*80*4)
	dst.ReadPixels(expected)
	dst.Clear()
	text.Draw(dst, "ABCDEF", face, nil)
	got := make([]byte, len(expected))
	dst.ReadPixels(got)
	if !bytes.Equal(got, expected) {
		t.Error("repeated SVG drawing changed after heavy-plan eviction")
	}
	for _, g := range lazy {
		if img := g.Image(); img == nil {
			t.Error("captured glyph lost its image after plan eviction")
		} else if got := img.At(1, 1); got != (color.RGBA{R: 255, A: 255}) {
			t.Errorf("captured glyph after plan eviction: %v", got)
		}
	}
}

func TestSVGSidewaysOutline(t *testing.T) {
	plain, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		t.Fatal(err)
	}
	face := &text.GoTextFace{
		Source:    plain,
		Size:      40,
		Direction: text.DirectionTopToBottomAndLeftToRight,
	}
	want := text.AppendGlyphs(nil, "A", face, nil)[0]
	face.Source = svgTestFont(t, `<svg><rect width="1000" height="1000" fill="red"/></svg>`, false)
	got := text.AppendGlyphs(nil, "A", face, nil)[0]
	if got.Image == nil || want.Image == nil {
		t.Fatal("missing outline image")
	}
	if got.Colored {
		t.Error("sideways SVG used colored artwork")
	}
	if got.X != want.X || got.Y != want.Y || got.Image.Bounds() != want.Image.Bounds() {
		t.Errorf("sideways position/bounds differ: got %v %v %v, want %v %v %v", got.X, got.Y, got.Image.Bounds(), want.X, want.Y, want.Image.Bounds())
		return
	}
	gp, wp := make([]byte, 4*got.Image.Bounds().Dx()*got.Image.Bounds().Dy()), make([]byte, 4*want.Image.Bounds().Dx()*want.Image.Bounds().Dy())
	got.Image.ReadPixels(gp)
	want.Image.ReadPixels(wp)
	if !bytes.Equal(gp, wp) {
		t.Error("sideways SVG did not use the rotated outline")
	}
}

func TestSVGColorFormatPriority(t *testing.T) {
	for _, tc := range []struct {
		file, str string
		want      color.RGBA
	}{
		{
			file: "chromacheck-colr.ttf",
			str:  "\uE900",
			want: color.RGBA{R: 0xc8, A: 255},
		},
		{
			file: "chromacheck-cbdt.ttf",
			str:  "\uE903",
			want: color.RGBA{R: 0x64, A: 255},
		},
	} {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + tc.file)
			if err != nil {
				t.Fatal(err)
			}
			loader, err := opentype.NewLoader(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			doc := []byte(`<svg viewBox="0 0 100 100"><rect width="100" height="100" fill="blue"/></svg>`)
			svg := make([]byte, 24)
			binary.BigEndian.PutUint32(svg[2:], 10)
			binary.BigEndian.PutUint16(svg[10:], 1)
			binary.BigEndian.PutUint16(svg[14:], 65535)
			binary.BigEndian.PutUint32(svg[16:], 14)
			binary.BigEndian.PutUint32(svg[20:], uint32(len(doc)))
			var tables []opentype.Table
			for _, tag := range loader.Tables() {
				if tag == opentype.MustNewTag("SVG ") {
					continue
				}
				raw, err := loader.RawTable(tag)
				if err != nil {
					t.Fatal(err)
				}
				tables = append(tables, opentype.Table{
					Tag:     tag,
					Content: raw,
				})
			}
			tables = append(tables, opentype.Table{
				Tag:     opentype.MustNewTag("SVG "),
				Content: append(svg, doc...),
			})
			src, err := text.NewGoTextFaceSource(bytes.NewReader(opentype.WriteTTF(tables)))
			if err != nil {
				t.Fatal(err)
			}
			face := &text.GoTextFace{
				Source: src,
				Size:   40,
			}
			glyph := text.AppendGlyphs(nil, tc.str, face, nil)[0]
			if glyph.Image == nil {
				t.Fatal("missing color glyph")
			}
			b := glyph.Image.Bounds()
			if got := glyph.Image.At(b.Dx()/2, b.Dy()/2); got != tc.want {
				t.Errorf("SVG displaced preferred format: %v, want %v", got, tc.want)
			}
		})
	}
}
