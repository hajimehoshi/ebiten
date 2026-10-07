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

// Package opentypesvg retains OpenType SVG documents for glyph interpretation.
package opentypesvg

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	svgNamespace   = "http://www.w3.org/2000/svg"
	xlinkNamespace = "http://www.w3.org/1999/xlink"
	xmlNamespace   = "http://www.w3.org/XML/1998/namespace"
	xmlnsNamespace = "http://www.w3.org/2000/xmlns/"

	// The decoded byte budget leaves more than twofold headroom for Noto's
	// roughly 14 MB shared document.
	maxBytes = 32 << 20
	maxDepth = 256

	// This allows over twice the 109,577 elements in the largest document
	// of Noto Color Emoji 2.051.
	maxElements   = 1 << 18
	maxAttributes = 1 << 20

	// Reference depth includes child edges. The expansion budget counts
	// conceptual instances, including repeated visits to shared definitions.
	maxReferenceDepth   = maxDepth
	maxExpandedElements = 1 << 20
)

var (
	ErrXML            = errors.New("opentypesvg: invalid XML")
	ErrNamespace      = errors.New("opentypesvg: invalid namespace")
	ErrLimit          = errors.New("opentypesvg: document limit exceeded")
	ErrGlyphNotFound  = errors.New("opentypesvg: glyph not found")
	ErrReference      = errors.New("opentypesvg: invalid or unresolved local reference")
	ErrReferenceCycle = errors.New("opentypesvg: reference cycle")
)

// Element is an immutable SVG element in its source document.
type Element struct {
	name       xml.Name
	attributes []xml.Attr
	parent     *Element
	children   []*Element

	// Only style text is retained, concatenating direct character data
	// in source order. Mixed text/element ordering is not represented,
	// and descendant text is excluded. Prohibited elements such as a
	// and text have their entire subtrees ignored.
	text string
}

// Name returns the expanded XML name.
func (e *Element) Name() xml.Name {
	return e.name
}

// Attribute returns the value of an expanded XML attribute name.
func (e *Element) Attribute(space, local string) (string, bool) {
	idx := slices.IndexFunc(e.attributes, func(a xml.Attr) bool {
		return a.Name.Space == space && a.Name.Local == local
	})
	if idx < 0 {
		return "", false
	}
	return e.attributes[idx].Value, true
}

// AttributeCount returns the number of attributes.
func (e *Element) AttributeCount() int {
	return len(e.attributes)
}

// AttributeAt returns an attribute by source order.
func (e *Element) AttributeAt(index int) xml.Attr {
	return e.attributes[index]
}

// SourceParent returns the parent in the source document.
func (e *Element) SourceParent() *Element {
	return e.parent
}

// ChildCount returns the number of retained child elements.
func (e *Element) ChildCount() int {
	return len(e.children)
}

// Child returns a child by source order.
func (e *Element) Child(index int) *Element {
	return e.children[index]
}

// Text returns character data for a style element, or an empty string otherwise.
func (e *Element) Text() string {
	return e.text
}

// Document is an immutable, uncompressed SVG document.
type Document struct {
	root   *Element
	ids    map[string]*Element
	styles []*Element
}

// Root returns the source document's SVG root.
func (d *Document) Root() *Element {
	return d.root
}

// StylesheetCount returns the number of retained style elements.
func (d *Document) StylesheetCount() int {
	return len(d.styles)
}

// Stylesheet returns a style element by document order.
func (d *Document) Stylesheet(index int) *Element {
	return d.styles[index]
}

// Resolve resolves a local URI fragment to an element.
func (d *Document) Resolve(reference string) (*Element, error) {
	reference = strings.Trim(reference, " \t\r\n\f")
	if !strings.HasPrefix(reference, "#") {
		return nil, ErrReference
	}
	id, err := url.PathUnescape(reference[1:])
	if err != nil || id == "" {
		return nil, ErrReference
	}
	if e := d.ids[id]; e != nil {
		return e, nil
	}
	return nil, fmt.Errorf("%w: %q", ErrReference, id)
}

// SelectGlyph returns the requested source entry, or an error for missing glyphs or invalid template references.
func (d *Document) SelectGlyph(gid uint16) (*Element, error) {
	// The interpreter must render the entry as a use instance with its own
	// inheritance context. SourceParent is not an instance parent: original
	// ancestor transforms, opacity and inherited properties do not apply.
	// CSS selector ancestry during glyph extraction remains to be settled in
	// the style interpreter. The source hierarchy is retained independently.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/svg#glyph-identifiers
	// https://www.w3.org/TR/SVG11/struct.html#UseElement
	e := d.ids["glyph"+strconv.FormatUint(uint64(gid), 10)]
	if e == nil {
		return nil, ErrGlyphNotFound
	}
	if _, err := d.checkReferences(e, make(map[*Element]referenceCost), make(map[*Element]bool), 0); err != nil {
		return nil, err
	}
	return e, nil
}

type referenceCost struct {
	elements int
	depth    int
}

func (d *Document) checkReferences(e *Element, costs map[*Element]referenceCost, active map[*Element]bool, depth int) (referenceCost, error) {
	if depth >= maxReferenceDepth {
		return referenceCost{}, ErrLimit
	}
	if active[e] {
		return referenceCost{}, ErrReferenceCycle
	}
	if cost, ok := costs[e]; ok {
		if depth+cost.depth > maxReferenceDepth {
			return referenceCost{}, ErrLimit
		}
		return cost, nil
	}
	active[e] = true
	defer delete(active, e)
	cost := referenceCost{
		elements: 1,
		depth:    1,
	}
	// Memoized subtree costs are added for every instance without cloning the
	// subtree. The sum detects excessive expansion even in acyclic use graphs.
	add := func(child *Element) error {
		childCost, err := d.checkReferences(child, costs, active, depth+1)
		if err != nil {
			return err
		}
		if childCost.elements > maxExpandedElements-cost.elements {
			return ErrLimit
		}
		cost.elements += childCost.elements
		cost.depth = max(cost.depth, childCost.depth+1)
		return nil
	}
	for _, child := range e.children {
		switch child.name.Local {
		case "clipPath", "defs", "filter", "linearGradient", "marker", "mask", "pattern", "radialGradient", "style", "symbol":
			continue
		}
		if err := add(child); err != nil {
			return referenceCost{}, err
		}
	}
	// Only template references contribute to conceptual expansion. Paint URLs
	// are resolved by the later property interpreter using Resolve, which must
	// also bound any paint or CSS reference traversal it introduces.
	switch e.name.Local {
	case "use":
		ref, ok := e.Attribute("", "href")
		if !ok {
			ref, ok = e.Attribute(xlinkNamespace, "href")
		}
		if ok {
			target, err := d.Resolve(ref)
			if err != nil {
				return referenceCost{}, err
			}
			if err := add(target); err != nil {
				return referenceCost{}, err
			}
		}
	}
	costs[e] = cost
	return cost, nil
}

type parseFrame struct {
	element *Element
	text    strings.Builder
}

// Parse parses a UTF-8 XML document already decoded by the font library.
// The returned document does not retain source.
func Parse(source []byte) (*Document, error) {
	// go-text/typesetting decompresses gzip before returning GlyphSVG.Source.
	// Its decompression allocation is unbounded; the byte limit here applies
	// only after that boundary.
	if len(source) > maxBytes {
		return nil, ErrLimit
	}
	if !utf8.Valid(source) {
		return nil, ErrXML
	}
	source = bytes.TrimPrefix(source, []byte{0xef, 0xbb, 0xbf})
	decoder := xml.NewDecoder(bytes.NewReader(source))
	// Token manages namespace scopes. Undeclared prefixes remain in Name.Space
	// and are treated as foreign. They cannot match the SVG or XLink namespace
	// URIs, which contain colons, so they cannot introduce SVG elements or hrefs.
	document := &Document{
		ids: make(map[string]*Element),
	}
	var stack []*parseFrame
	var elements, attributes int
	var closed, doctype bool
	for {
		offset := decoder.InputOffset()
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrXML, err)
		}
		switch token := token.(type) {
		case xml.StartElement:
			elements++
			attributes += len(token.Attr)
			if len(stack) >= maxDepth || elements > maxElements || attributes > maxAttributes {
				return nil, ErrLimit
			}
			if closed {
				return nil, ErrXML
			}
			var frame parseFrame
			seen := make(map[xml.Name]bool, len(token.Attr))
			for _, a := range token.Attr {
				if seen[a.Name] {
					return nil, ErrXML
				}
				seen[a.Name] = true
			}
			for _, a := range token.Attr {
				if a.Name.Space != "xmlns" && !(a.Name.Space == "" && a.Name.Local == "xmlns") {
					continue
				}
				prefix := a.Name.Local
				if a.Name.Space == "" {
					prefix = ""
				}
				if strings.Contains(prefix, ":") || prefix == "xmlns" || a.Value == xmlnsNamespace || (prefix == "xml") != (a.Value == xmlNamespace) || (prefix != "" && a.Value == "") {
					return nil, ErrNamespace
				}
			}
			// Accept an empty namespace for compatibility with go-text's SVG root-attribute handling.
			isSVG := token.Name.Space == "" || token.Name.Space == svgNamespace
			if len(stack) == 0 && (!isSVG || token.Name.Local != "svg") {
				return nil, ErrNamespace
			}
			var parent *Element
			if len(stack) > 0 {
				parent = stack[len(stack)-1].element
			}
			if (len(stack) == 0 || parent != nil) && isSVG && !ignoredElement(token.Name.Local) {
				e := &Element{
					name:       token.Name,
					attributes: append([]xml.Attr(nil), token.Attr...),
					parent:     parent,
				}
				frame.element = e
				if parent != nil {
					parent.children = append(parent.children, e)
				} else {
					document.root = e
				}
				if id, ok := e.Attribute("", "id"); ok && id != "" && document.ids[id] == nil {
					document.ids[id] = e
				}
				if e.name.Local == "style" {
					document.styles = append(document.styles, e)
				}
			}
			stack = append(stack, &frame)
		case xml.EndElement:
			frame := stack[len(stack)-1]
			if frame.element != nil {
				frame.element.text = frame.text.String()
			}
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				closed = true
			}
		case xml.CharData:
			if len(stack) == 0 {
				if len(bytes.TrimSpace(token)) != 0 {
					return nil, ErrXML
				}
			} else if frame := stack[len(stack)-1]; frame.element != nil && frame.element.name.Local == "style" {
				_, _ = frame.text.Write(token)
			}
		case xml.Directive:
			// Ordinary DOCTYPE declarations are inert. External subsets are never
			// fetched, and internal subsets (including entity declarations) are
			// outside the supported profile.
			if document.root != nil || doctype {
				return nil, ErrXML
			}
			declaration, ok := bytes.CutPrefix(token, []byte("DOCTYPE"))
			if !ok || len(declaration) == 0 || !strings.ContainsRune(" \t\r\n", rune(declaration[0])) {
				return nil, ErrXML
			}
			if len(bytes.TrimSpace(declaration)) == 0 || bytes.IndexByte(declaration, '[') >= 0 {
				return nil, ErrXML
			}
			doctype = true
		case xml.ProcInst:
			if strings.EqualFold(token.Target, "xml") && (token.Target != "xml" || offset != 0) {
				return nil, ErrXML
			}
			// Processing instructions are never executed or used to load stylesheets.
		}
	}
	if document.root == nil || !closed {
		return nil, ErrXML
	}
	return document, nil
}

func ignoredElement(name string) bool {
	switch name {
	case "a", "altGlyph", "altGlyphDef", "altGlyphItem", "color-profile", "desc", "font", "font-face",
		"font-face-format", "font-face-name", "font-face-src", "font-face-uri", "foreignObject", "glyph",
		"glyphRef", "hkern", "metadata", "missing-glyph", "script", "switch", "text", "textPath", "title", "tref",
		"tspan", "view", "vkern":
		return true
	}
	return false
}
