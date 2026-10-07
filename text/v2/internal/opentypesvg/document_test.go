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
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-text/typesetting/font"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/opentypesvg"
)

const svgStart = `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">`
const xlink = "http://www.w3.org/1999/xlink"

func parse(t *testing.T, source string) *opentypesvg.Document {
	t.Helper()
	document, err := opentypesvg.Parse([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func read(t *testing.T, path string) []byte {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return source
}

func attribute(t *testing.T, element *opentypesvg.Element, space, local, want string) {
	t.Helper()
	got, ok := element.Attribute(space, local)
	if !ok || got != want {
		t.Errorf("attribute {%s}%s = %q (%v), want %q", space, local, got, ok, want)
	}
}

func TestGlyphSelection(t *testing.T) {
	document := parse(t, string(read(t, "testdata/focused/shared-context.svg")))
	for _, gid := range []uint16{1, 2} {
		entry, err := document.SelectGlyph(gid)
		if err != nil {
			t.Fatal(err)
		}
		attribute(t, entry, "", "id", fmt.Sprintf("glyph%d", gid))
		if entry.ChildCount() != 1 {
			t.Fatalf("entry children = %d, want 1", entry.ChildCount())
		}
		use := entry.Child(0)
		attribute(t, use, xlink, "href", "#tile")
		attribute(t, entry.SourceParent(), "", "transform", "translate(8 8)")
	}
	tile, err := document.Resolve("#tile")
	if err != nil {
		t.Fatal(err)
	}
	attribute(t, tile, "", "d", "M0 0H8V8H0Z")
	if entry, err := document.SelectGlyph(3); entry != nil || !errors.Is(err, opentypesvg.ErrGlyphNotFound) {
		t.Errorf("missing glyph = %v, %v", entry, err)
	}
	rootGlyph := parse(t, "\xef\xbb\xbf"+`<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" id="glyph7"><path d="M0 0Z"/></svg>`)
	if got, err := rootGlyph.SelectGlyph(7); err != nil || got != rootGlyph.Root() {
		t.Errorf("root glyph = %v, %v", got, err)
	}
	if _, err := parse(t, svgStart+`<g id="glyph01"/></svg>`).SelectGlyph(1); !errors.Is(err, opentypesvg.ErrGlyphNotFound) {
		t.Errorf("padded ID: %v", err)
	}
}

func TestDuplicateIDs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "siblings",
			content: `<path id="a"/><g id="a"/>`,
			want:    "path",
		},
		{
			name:    "nested",
			content: `<g id="a"><path id="a"/></g>`,
			want:    "g",
		},
		{
			name:    "ignored",
			content: `<switch><g id="a"/></switch><path id="a"/>`,
			want:    "path",
		},
		{
			name:    "foreign",
			content: `<g xmlns="urn:foreign" id="a"/><path id="a"/>`,
			want:    "path",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document := parse(t, svgStart+tc.content+`</svg>`)
			element, err := document.Resolve("#a")
			if err != nil {
				t.Fatal(err)
			}
			if got := element.Name().Local; got != tc.want {
				t.Errorf("first ID resolved to %s, want %s", got, tc.want)
			}
		})
	}
	document := parse(t, svgStart+`<path id="glyph1"/><g id="glyph1"><use href="#missing"/></g></svg>`)
	entry, err := document.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	if got := entry.Name().Local; got != "path" {
		t.Errorf("first glyph = %s, want path", got)
	}
}

func TestRetainedContext(t *testing.T) {
	document := parse(t, svgStart+`<defs><style><![CDATA[g > .tile {fill:url(#later)}]]></style></defs><g transform="scale(2)" opacity=".5"><g id="glyph1" style="fill:currentColor"><path class="tile" d="M1 2q3 4 5 6z"/><g opacity=".25"><rect width="8"/></g></g></g><linearGradient id="later"/></svg>`)
	entry, err := document.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	if entry.ChildCount() != 2 {
		t.Fatalf("children = %d", entry.ChildCount())
	}
	attribute(t, entry, "", "style", "fill:currentColor")
	attribute(t, entry.Child(0), "", "d", "M1 2q3 4 5 6z")
	attribute(t, entry.Child(1), "", "opacity", ".25")
	if entry.Child(1).ChildCount() != 1 {
		t.Error("group boundary was lost")
	}
	if document.StylesheetCount() != 1 {
		t.Fatalf("styles = %d", document.StylesheetCount())
	}
	if got := document.Stylesheet(0).Text(); got != "g > .tile {fill:url(#later)}" {
		t.Errorf("CSS = %q", got)
	}
	gradient, err := document.Resolve("#later")
	if err != nil || gradient == nil {
		t.Fatalf("forward paint reference: %v", err)
	}
	if gradient.Name().Local != "linearGradient" {
		t.Errorf("gradient name = %v", gradient.Name())
	}
}

func TestReferences(t *testing.T) {
	for _, definitions := range []string{
		`<g id="shape"><path d="M0 0Z"/></g><g id="alias"><use xlink:href="#shape"/></g>`,
		`<g id="alias"><use xlink:href="#shape"/></g><g id="shape"><path d="M0 0Z"/></g>`,
	} {
		document := parse(t, svgStart+`<g id="glyph1"><use xlink:href="#alias"/></g><defs>`+definitions+`</defs></svg>`)
		if _, err := document.SelectGlyph(1); err != nil {
			t.Error(err)
		}
		target, err := document.Resolve("#%73hape")
		if err != nil {
			t.Fatal(err)
		}
		attribute(t, target, "", "id", "shape")
	}
	document := parse(t, svgStart+`</svg>`)
	for _, ref := range []string{"", "#", "#missing", "#%zz", "https://example.invalid/#base", "file.svg#base", "data:image/svg+xml,<svg/>"} {
		if _, err := document.Resolve(ref); !errors.Is(err, opentypesvg.ErrReference) {
			t.Errorf("Resolve(%q): %v", ref, err)
		}
	}
	for _, tc := range []struct {
		name    string
		content string
		want    error
	}{
		{
			name:    "missing",
			content: `<g id="glyph1"><use xlink:href="#missing"/></g>`,
			want:    opentypesvg.ErrReference,
		},
		{
			name:    "external",
			content: `<g id="glyph1"><use xlink:href="https://example.invalid/a"/></g>`,
			want:    opentypesvg.ErrReference,
		},
		{
			name:    "self",
			content: `<g id="glyph1"><use xlink:href="#glyph1"/></g>`,
			want:    opentypesvg.ErrReferenceCycle,
		},
		{
			name:    "indirect",
			content: `<g id="glyph1"><use xlink:href="#a"/></g><g id="a"><use xlink:href="#b"/></g><g id="b"><use xlink:href="#a"/></g>`,
			want:    opentypesvg.ErrReferenceCycle,
		},
		{
			name:    "bad sibling",
			content: `<g id="glyph1"><path/></g><g id="glyph2"><use xlink:href="#glyph2"/></g>`,
		},
		{
			name:    "href precedence",
			content: `<g id="glyph1"><use href="#shape" xlink:href="#missing"/></g><path id="shape"/>`,
		},
		{
			name:    "empty href precedence",
			content: `<g id="glyph1"><use href="" xlink:href="#shape"/></g><path id="shape"/>`,
			want:    opentypesvg.ErrReference,
		},
		{
			name:    "href whitespace",
			content: `<g id="glyph1"><use href=" &#x9;#shape&#xA; "/></g><path id="shape"/>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document := parse(t, svgStart+tc.content+`</svg>`)
			_, err := document.SelectGlyph(1)
			if !errors.Is(err, tc.want) {
				t.Errorf("selection: %v, want %v", err, tc.want)
			}
		})
	}
}

func TestUnusedDefinitions(t *testing.T) {
	for _, name := range []string{"defs", "style", "clipPath", "mask", "pattern", "marker", "symbol", "linearGradient", "radialGradient", "filter"} {
		t.Run(name, func(t *testing.T) {
			document := parse(t, svgStart+`<g id="glyph1"><`+name+` id="unused"><use href="#missing"/></`+name+`><path/></g></svg>`)
			if _, err := document.SelectGlyph(1); err != nil {
				t.Error(err)
			}
			if _, err := document.Resolve("#unused"); err != nil {
				t.Error(err)
			}
		})
	}
	document := parse(t, svgStart+`<g id="glyph1"><use href="#shape"/></g><symbol id="shape"><use href="#missing"/></symbol></svg>`)
	if _, err := document.SelectGlyph(1); !errors.Is(err, opentypesvg.ErrReference) {
		t.Errorf("referenced symbol: %v", err)
	}
}

func TestDOCTYPE(t *testing.T) {
	for _, declaration := range []string{
		`<!DOCTYPE svg>`,
		`<!DOCTYPE svg SYSTEM "https://example.invalid/svg.dtd">`,
		`<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "https://example.invalid/svg.dtd">`,
		"<!DOCTYPE\nsvg>",
	} {
		t.Run(declaration, func(t *testing.T) {
			document := parse(t, declaration+svgStart+`<g id="glyph1"/></svg>`)
			if _, err := document.SelectGlyph(1); err != nil {
				t.Error(err)
			}
		})
	}
	for _, source := range []string{
		`<!DOCTYPE svg []>` + svgStart + `</svg>`,
		`<!DOCTYPE svg><!DOCTYPE svg>` + svgStart + `</svg>`,
		svgStart + `<!DOCTYPE svg></svg>`,
		`<!DOCTYPE>` + svgStart + `</svg>`,
		`<!DOCTYPE >` + svgStart + `</svg>`,
		`<!OTHER svg>` + svgStart + `</svg>`,
	} {
		if _, err := opentypesvg.Parse([]byte(source)); !errors.Is(err, opentypesvg.ErrXML) {
			t.Errorf("Parse(%q): %v", source, err)
		}
	}
}

func TestNamespaces(t *testing.T) {
	for _, root := range []string{`<svg>`, `<svg xmlns="">`, svgStart} {
		t.Run(root, func(t *testing.T) {
			document := parse(t, root+`<g xmlns="" id="glyph1"><use href="#shape"/></g><path id="shape"/></svg>`)
			if _, err := document.SelectGlyph(1); err != nil {
				t.Error(err)
			}
		})
	}
	document := parse(t, `<s:svg xmlns:s="http://www.w3.org/2000/svg" xmlns:l="http://www.w3.org/1999/xlink" xmlns:f="urn:foreign">
<s:g id="glyph1">
<s:g xmlns:l="urn:foreign"><s:use l:href="#missing"/></s:g>
<s:use l:href="#shape"/>
<f:g id="foreign"><s:path id="hidden"/></f:g>
<p:g id="unbound"><s:path id="alsoHidden"/></p:g>
<s:use p:href="#missing"/><s:path p:id="notAnID"/>
<s:g xmlns:p="q"><q:path id="collision"/></s:g>
<g id="noNamespace"/>
</s:g><s:path id="shape"/></s:svg>`)
	entry, err := document.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	if entry.Name().Space != "http://www.w3.org/2000/svg" {
		t.Errorf("glyph namespace = %q", entry.Name().Space)
	}
	if entry.ChildCount() < 2 {
		t.Fatal("missing retained glyph children")
	}
	attribute(t, entry.Child(1), xlink, "href", "#shape")
	if _, err := document.Resolve("#noNamespace"); err != nil {
		t.Error(err)
	}
	for _, id := range []string{"foreign", "hidden", "unbound", "alsoHidden", "notAnID", "collision"} {
		if _, err := document.Resolve("#" + id); !errors.Is(err, opentypesvg.ErrReference) {
			t.Errorf("foreign ID %q: %v", id, err)
		}
	}
}

func TestRestrictedContent(t *testing.T) {
	document := parse(t, `<?xml-stylesheet href="https://example.invalid/a.css"?>`+string(read(t, "testdata/focused/restricted.svg")))
	entry, err := document.SelectGlyph(1)
	if err != nil {
		t.Fatal(err)
	}
	if entry.ChildCount() != 1 || entry.Child(0).Name().Local != "rect" {
		t.Error("prohibited subtrees survived")
	}
}

func TestMalformedDocuments(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		want   error
	}{
		{
			name: "empty",
			want: opentypesvg.ErrXML,
		},
		{
			name:   "truncated",
			source: svgStart + `<g>`,
			want:   opentypesvg.ErrXML,
		},
		{
			name:   "mismatch",
			source: svgStart + `<g></path></svg>`,
			want:   opentypesvg.ErrXML,
		},
		{
			name:   "multiple roots",
			source: svgStart + `</svg>` + svgStart + `</svg>`,
			want:   opentypesvg.ErrXML,
		},
		{
			name:   "trailing text",
			source: svgStart + `</svg>x`,
			want:   opentypesvg.ErrXML,
		},
		{
			name:   "duplicate attribute",
			source: svgStart + `<path id="a" id="b"/></svg>`,
			want:   opentypesvg.ErrXML,
		},
		{
			name:   "duplicate expanded attribute",
			source: svgStart + `<path xmlns:p="urn:a" xmlns:q="urn:a" p:x="a" q:x="b"/></svg>`,
			want:   opentypesvg.ErrXML,
		},
		{
			name:   "foreign root namespace",
			source: `<svg xmlns="urn:foreign"/>`,
			want:   opentypesvg.ErrNamespace,
		},
		{
			name:   "wrong root",
			source: `<g xmlns="http://www.w3.org/2000/svg"/>`,
			want:   opentypesvg.ErrNamespace,
		},
		{
			name:   "multiple colons",
			source: svgStart + `<g xmlns:p:q="urn:a"/></svg>`,
			want:   opentypesvg.ErrXML,
		},
		{
			name:   "reserved binding",
			source: `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xml="urn:wrong"/>`,
			want:   opentypesvg.ErrNamespace,
		},
		{
			name:   "entity",
			source: svgStart + `<style>&unknown;</style></svg>`,
			want:   opentypesvg.ErrXML,
		},
		{
			name:   "DTD",
			source: `<!DOCTYPE svg [<!ENTITY x "foo">]>` + svgStart + `</svg>`,
			want:   opentypesvg.ErrXML,
		},
		{
			name:   "UTF8",
			source: svgStart + "\xff</svg>",
			want:   opentypesvg.ErrXML,
		},
		{
			name:   "misplaced declaration",
			source: svgStart + `<?xml version="1.0"?></svg>`,
			want:   opentypesvg.ErrXML,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, err := opentypesvg.Parse([]byte(tc.source))
			if document != nil || !errors.Is(err, tc.want) {
				t.Errorf("Parse = %v, %v; want %v", document, err, tc.want)
			}
		})
	}
}

func TestLimits(t *testing.T) {
	var attributeSource strings.Builder
	attributeSource.WriteString("<g")
	for i := range 256 {
		fmt.Fprintf(&attributeSource, ` a%d=""`, i)
	}
	attributeSource.WriteString("/>")
	for _, tc := range []struct {
		name   string
		source string
	}{
		{
			name:   "attributes",
			source: svgStart + strings.Repeat(attributeSource.String(), 4096) + `</svg>`,
		},
		{
			name:   "bytes",
			source: strings.Repeat(" ", (32<<20)+1),
		},
		{
			name:   "nesting",
			source: svgStart + strings.Repeat("<g>", 256) + strings.Repeat("</g>", 256) + `</svg>`,
		},
		{
			name:   "elements including ignored",
			source: svgStart + `<script>` + strings.Repeat("<g/>", 1<<18) + `</script></svg>`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := opentypesvg.Parse([]byte(tc.source)); !errors.Is(err, opentypesvg.ErrLimit) {
				t.Errorf("Parse: %v", err)
			}
		})
	}
	nested := `<svg id="glyph1">` + strings.Repeat(`<g>`, 255) + strings.Repeat(`</g>`, 255) + `</svg>`
	if _, err := parse(t, nested).SelectGlyph(1); err != nil {
		t.Errorf("256 XML levels: %v", err)
	}
	large := svgStart + `<g id="glyph1"><path d="` + strings.Repeat("M0 0 ", (14<<20)/5) + `"/></g></svg>`
	if _, err := parse(t, large).SelectGlyph(1); err != nil {
		t.Errorf("14 MiB document: %v", err)
	}
	for _, tc := range []struct {
		name   string
		levels int
		copies int
		want   error
	}{
		{
			name:   "shared DAG",
			levels: 10,
			copies: 2,
		},
		{
			name:   "exponential DAG",
			levels: 20,
			copies: 2,
			want:   opentypesvg.ErrLimit,
		},
		{
			name:   "reference depth",
			levels: 256,
			copies: 1,
			want:   opentypesvg.ErrLimit,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var source strings.Builder
			source.WriteString(svgStart + `<g id="glyph1"><use xlink:href="#n0"/></g><defs>`)
			for i := 0; i < tc.levels; i++ {
				fmt.Fprintf(&source, `<g id="n%d">`, i)
				for j := 0; j < tc.copies; j++ {
					fmt.Fprintf(&source, `<use xlink:href="#n%d"/>`, i+1)
				}
				source.WriteString(`</g>`)
			}
			fmt.Fprintf(&source, `<path id="n%d"/></defs></svg>`, tc.levels)
			document := parse(t, source.String())
			if _, err := document.SelectGlyph(1); !errors.Is(err, tc.want) {
				t.Errorf("selection: %v, want %v", err, tc.want)
			}
		})
	}
}

func fontSVG(t *testing.T, record []byte) []byte {
	t.Helper()
	source := read(t, "../../testdata/chromacheck-svg.ttf")
	for len(source)%4 != 0 {
		source = append(source, 0)
	}
	offset := len(source)
	table := make([]byte, 24)
	binary.BigEndian.PutUint32(table[2:], 10)
	binary.BigEndian.PutUint16(table[10:], 1)
	binary.BigEndian.PutUint16(table[12:], 1)
	binary.BigEndian.PutUint16(table[14:], 2)
	binary.BigEndian.PutUint32(table[16:], 14)
	binary.BigEndian.PutUint32(table[20:], uint32(len(record)))
	source = append(source, table...)
	source = append(source, record...)
	for i := 0; i < int(binary.BigEndian.Uint16(source[4:])); i++ {
		entry := source[12+16*i : 12+16*(i+1)]
		if string(entry[:4]) != "SVG " {
			continue
		}
		binary.BigEndian.PutUint32(entry[8:], uint32(offset))
		binary.BigEndian.PutUint32(entry[12:], uint32(len(table)+len(record)))
		face, err := font.ParseTTF(bytes.NewReader(source))
		if err != nil {
			t.Fatal(err)
		}
		glyph, ok := face.GlyphDataSVG(1)
		if !ok {
			t.Fatal("font has no SVG glyph 1")
		}
		return glyph.Source
	}
	t.Fatal("test font has no SVG table")
	return nil
}

func TestFontDecodingBoundary(t *testing.T) {
	plain := read(t, "testdata/focused/shared-context.svg")
	for _, name := range []string{"shared-context.svg", "shared-context.svg.gz"} {
		source := fontSVG(t, read(t, "testdata/focused/"+name))
		if !bytes.Equal(source, plain) {
			t.Errorf("%s: decoded bytes differ", name)
		}
		document, err := opentypesvg.Parse(source)
		if err != nil {
			t.Fatal(err)
		}
		clear(source)
		entry, err := document.SelectGlyph(1)
		if err != nil {
			t.Fatal(err)
		}
		attribute(t, entry, "", "id", "glyph1")
	}
	if _, err := opentypesvg.Parse(read(t, "testdata/focused/shared-context.svg.gz")); !errors.Is(err, opentypesvg.ErrXML) {
		t.Errorf("encoded input: %v", err)
	}
}

func TestCorpus(t *testing.T) {
	files, err := filepath.Glob("testdata/*/*.svg")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range files {
		t.Run(filepath.Base(path), func(t *testing.T) {
			document := parse(t, string(read(t, path)))
			gid := uint16(1)
			if filepath.Base(filepath.Dir(path)) == "real" {
				var first, last uint16
				name := strings.TrimSuffix(filepath.Base(path), ".svg")
				if _, err := fmt.Sscanf(name[strings.IndexByte(name, '-')+1:], "%d-%d", &first, &last); err != nil {
					t.Fatal(err)
				}
				for gid = first; gid <= last; gid++ {
					if _, err := document.SelectGlyph(gid); err != nil {
						t.Errorf("glyph %d: %v", gid, err)
					}
				}
			} else if _, err := document.SelectGlyph(gid); err != nil {
				t.Error(err)
			}
		})
	}
}

func FuzzDocument(f *testing.F) {
	f.Add([]byte(svgStart+`<g id="glyph1"><use xlink:href="#p"/></g><path id="p"/></svg>`), uint16(1))
	f.Add([]byte(svgStart+`<g id="glyph1"><use xlink:href="#glyph1"/></g></svg>`), uint16(1))
	f.Add([]byte(`<svg/>`), uint16(0))
	files, err := filepath.Glob("testdata/*/*.svg")
	if err != nil {
		f.Fatal(err)
	}
	for _, path := range files {
		source, err := os.ReadFile(path)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(source, uint16(1))
	}
	f.Fuzz(func(t *testing.T, source []byte, gid uint16) {
		document, err := opentypesvg.Parse(source)
		if err != nil {
			return
		}
		entry, err := document.SelectGlyph(gid)
		if err != nil {
			return
		}
		attribute(t, entry, "", "id", fmt.Sprintf("glyph%d", gid))
		resolved, err := document.Resolve(fmt.Sprintf("#glyph%d", gid))
		if err != nil || resolved != entry {
			t.Errorf("selected entry lost its document binding: %v", err)
		}
	})
}
