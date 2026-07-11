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

package text

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// svgDocElement is an indexed element: the bytes of its subtree and
// the ids the subtree references, directly or via descendants.
type svgDocElement struct {
	content []byte
	refs    []string
}

// svgDocIndex indexes an OpenType SVG document. A document may describe
// several glyphs: each glyph is the element with id="glyph<GID>", and
// glyph elements typically share definitions via <defs> and <use>.
type svgDocIndex struct {
	source []byte

	// rootTag is the document's root <svg ...> start tag, verbatim,
	// and rootName is its element name for the matching closing tag.
	rootTag  []byte
	rootName string

	// rootGID is the glyph ID on the root element's id attribute, or -1.
	// A root-level glyph id means the whole document is that one glyph.
	rootGID int64

	// glyphs maps a glyph ID to its subtree.
	glyphs map[uint32]svgDocElement

	// ids maps an id attribute to its subtree, for resolving references
	// from glyph subtrees.
	ids map[string]svgDocElement

	// parseErr indicates the document could not be fully indexed.
	parseErr bool

	// extracted memoizes glyphDocument results.
	extracted map[uint32][]byte
}

// newSVGDocIndex scans source once and records the root tag plus, for
// every glyph element and every element with an id, the bytes of its
// subtree and the ids the subtree references: href="#id" attribute
// values, and url(#id) occurrences in attribute values and character
// data. The XML tokenizer decodes entity and character references, so
// the recorded references match the recorded ids.
func newSVGDocIndex(source []byte) *svgDocIndex {
	idx := &svgDocIndex{
		source:  source,
		rootGID: -1,
		glyphs:  map[uint32]svgDocElement{},
		ids:     map[string]svgDocElement{},
	}

	type openElement struct {
		start int
		id    string
		gid   int64 // -1 unless id is glyph<GID>
		refs  []string
	}

	d := xml.NewDecoder(bytes.NewReader(source))
	var stack []openElement
	for {
		tokStart := int(d.InputOffset())
		tok, err := d.RawToken()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				idx.parseErr = true
			}
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			e := openElement{start: tokStart, gid: -1}
			for _, attr := range t.Attr {
				switch {
				case attr.Name.Local == "id" && attr.Name.Space == "":
					if e.id != "" {
						break
					}
					e.id = attr.Value
					if rest, ok := strings.CutPrefix(attr.Value, "glyph"); ok {
						if gid, err := strconv.ParseUint(rest, 10, 32); err == nil {
							e.gid = int64(gid)
						}
					}
				case attr.Name.Local == "href":
					if id, ok := strings.CutPrefix(attr.Value, "#"); ok {
						e.refs = append(e.refs, id)
					}
				default:
					e.refs = appendCSSURLReferences(e.refs, attr.Value)
				}
			}
			if len(stack) == 0 {
				idx.rootTag = source[tokStart:int(d.InputOffset())]
				idx.rootName = t.Name.Local
				if t.Name.Space != "" {
					idx.rootName = t.Name.Space + ":" + t.Name.Local
				}
				idx.rootGID = e.gid
				// The root element is the whole document; only its
				// descendants are registered.
				e.id = ""
				e.gid = -1
			}
			stack = append(stack, e)
		case xml.CharData:
			// url(#id) can appear in character data via style sheets.
			if len(stack) > 0 {
				top := &stack[len(stack)-1]
				top.refs = appendCSSURLReferences(top.refs, string(t))
			}
		case xml.EndElement:
			if len(stack) == 0 {
				idx.parseErr = true
				break
			}
			e := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			entry := svgDocElement{
				content: source[e.start:int(d.InputOffset())],
				refs:    e.refs,
			}
			if e.gid >= 0 {
				if _, ok := idx.glyphs[uint32(e.gid)]; !ok {
					idx.glyphs[uint32(e.gid)] = entry
				}
			}
			if e.id != "" {
				if _, ok := idx.ids[e.id]; !ok {
					idx.ids[e.id] = entry
				}
			}
			// An element's references count for its ancestors too.
			if len(stack) > 0 && len(e.refs) > 0 {
				parent := &stack[len(stack)-1]
				parent.refs = append(parent.refs, e.refs...)
			}
		}
	}
	if len(stack) > 0 {
		idx.parseErr = true
	}
	return idx
}

// svgCSSURLReferenceRegexp matches url(#id) references in CSS values,
// such as fill attribute values and style sheets.
var svgCSSURLReferenceRegexp = regexp.MustCompile(`url\(\s*['"]?#([^'")\s]+)['"]?\s*\)`)

// appendCSSURLReferences appends the ids of url(#id) references in a
// CSS value.
func appendCSSURLReferences(dst []string, value string) []string {
	if !strings.Contains(value, "url(") {
		return dst
	}
	for _, m := range svgCSSURLReferenceRegexp.FindAllStringSubmatch(value, -1) {
		dst = append(dst, m[1])
	}
	return dst
}

// glyphDocument returns an SVG document describing only the glyph gid.
//
// When the document has no per-glyph structure for gid — it is a
// single-glyph document, possibly with the glyph id on the root
// element — the whole source is returned. When gid has its own subtree,
// a document containing the root tag, the definitions the subtree
// references, and the subtree itself is assembled. nil is returned
// when the glyph cannot be extracted reliably.
func (i *svgDocIndex) glyphDocument(gid uint32) []byte {
	if doc, ok := i.extracted[gid]; ok {
		return doc
	}
	doc := i.buildGlyphDocument(gid)
	if i.extracted == nil {
		i.extracted = map[uint32][]byte{}
	}
	i.extracted[gid] = doc
	return doc
}

func (i *svgDocIndex) buildGlyphDocument(gid uint32) []byte {
	glyph, ok := i.glyphs[gid]
	if !ok {
		if i.rootGID == int64(gid) {
			return i.source
		}
		// The document mentions the glyph's id but the subtree was not
		// indexed (malformed document): refuse rather than render the
		// whole document, which may contain other glyphs.
		if i.parseErr && bytes.Contains(i.source, fmt.Appendf(nil, `id="glyph%d"`, gid)) {
			return nil
		}
		// A document without any per-glyph elements describes a single
		// glyph as a whole. A document with per-glyph elements but
		// without this glyph has nothing to render for it.
		if len(i.glyphs) == 0 && i.rootGID < 0 && !i.parseErr {
			return i.source
		}
		return nil
	}

	// Collect the definitions referenced from the glyph subtree,
	// transitively. The relative order of the definitions doesn't
	// matter: references are resolved against ids, not positions, and
	// duplicated content of nested definitions is inert inside <defs>.
	var defs [][]byte
	visited := map[string]bool{}
	refs := slices.Clone(glyph.refs)
	for len(refs) > 0 {
		id := refs[len(refs)-1]
		refs = refs[:len(refs)-1]
		if visited[id] {
			continue
		}
		visited[id] = true
		def, ok := i.ids[id]
		if !ok {
			continue
		}
		defs = append(defs, def.content)
		refs = append(refs, def.refs...)
	}

	// Every written fragment is a balanced element copied verbatim from
	// the already-encoded source, so the concatenation stays well-formed
	// XML without re-escaping.
	var buf bytes.Buffer
	buf.Write(i.rootTag)
	if len(defs) > 0 {
		buf.WriteString("<defs>")
		for _, d := range defs {
			buf.Write(d)
		}
		buf.WriteString("</defs>")
	}
	buf.Write(glyph.content)
	buf.WriteString("</" + i.rootName + ">")
	return buf.Bytes()
}
