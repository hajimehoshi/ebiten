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
	errXML            = errors.New("opentypesvg: invalid XML")
	errNamespace      = errors.New("opentypesvg: invalid namespace")
	errLimit          = errors.New("opentypesvg: document limit exceeded")
	errGlyphNotFound  = errors.New("opentypesvg: glyph not found")
	errReference      = errors.New("opentypesvg: invalid or unresolved local reference")
	errReferenceCycle = errors.New("opentypesvg: reference cycle")
)

// element is an immutable SVG element in its source document.
type element struct {
	name       xml.Name
	attributes []xml.Attr
	parent     *element
	children   []*element

	// Only style text is retained, concatenating direct character data
	// in source order. Mixed text/element ordering is not represented,
	// and descendant text is excluded. Prohibited elements such as a
	// and text have their entire subtrees ignored.
	text string
}

// xmlName returns the expanded XML name.
func (e *element) xmlName() xml.Name {
	return e.name
}

// attribute returns the value of an expanded XML attribute name.
func (e *element) attribute(space, local string) (string, bool) {
	idx := slices.IndexFunc(e.attributes, func(a xml.Attr) bool {
		return a.Name.Space == space && a.Name.Local == local
	})
	if idx < 0 {
		return "", false
	}
	return e.attributes[idx].Value, true
}

// attributeCount returns the number of attributes.
func (e *element) attributeCount() int {
	return len(e.attributes)
}

// attributeAt returns an attribute by source order.
func (e *element) attributeAt(index int) xml.Attr {
	return e.attributes[index]
}

// sourceParent returns the parent in the source document.
func (e *element) sourceParent() *element {
	return e.parent
}

// childCount returns the number of retained child elements.
func (e *element) childCount() int {
	return len(e.children)
}

// child returns a child by source order.
func (e *element) child(index int) *element {
	return e.children[index]
}

// styleText returns character data for a style element, or an empty string otherwise.
func (e *element) styleText() string {
	return e.text
}

// document is an immutable, uncompressed SVG document.
type document struct {
	root   *element
	ids    map[string]*element
	styles []*element
}

// rootElement returns the source document's SVG root.
func (d *document) rootElement() *element {
	return d.root
}

// stylesheetCount returns the number of retained style elements.
func (d *document) stylesheetCount() int {
	return len(d.styles)
}

// stylesheet returns a style element by document order.
func (d *document) stylesheet(index int) *element {
	return d.styles[index]
}

// resolve resolves a local URI fragment to an element.
func (d *document) resolve(reference string) (*element, error) {
	reference = strings.Trim(reference, " \t\r\n\f")
	if !strings.HasPrefix(reference, "#") {
		return nil, errReference
	}
	id, err := url.PathUnescape(reference[1:])
	if err != nil || id == "" {
		return nil, errReference
	}
	if e := d.ids[id]; e != nil {
		return e, nil
	}
	return nil, fmt.Errorf("%w: %q", errReference, id)
}

// selectGlyph returns the requested source entry, or an error for missing glyphs or invalid template references.
func (d *document) selectGlyph(gid uint16) (*element, error) {
	// The interpreter must render the entry as a use instance with its own
	// inheritance context. sourceParent is not an instance parent: original
	// ancestor transforms, opacity and inherited properties do not apply.
	// CSS selector ancestry during glyph extraction remains to be settled in
	// the style interpreter. The source hierarchy is retained independently.
	// https://learn.microsoft.com/en-us/typography/opentype/spec/svg#glyph-identifiers
	// https://www.w3.org/TR/SVG11/struct.html#UseElement
	e := d.ids["glyph"+strconv.FormatUint(uint64(gid), 10)]
	if e == nil {
		return nil, errGlyphNotFound
	}
	if _, err := d.checkReferences(e, make(map[*element]referenceCost), make(map[*element]bool), 0); err != nil {
		return nil, err
	}
	return e, nil
}

type referenceCost struct {
	elements int
	depth    int
}

func (d *document) checkReferences(e *element, costs map[*element]referenceCost, active map[*element]bool, depth int) (referenceCost, error) {
	if depth >= maxReferenceDepth {
		return referenceCost{}, errLimit
	}
	if active[e] {
		return referenceCost{}, errReferenceCycle
	}
	if cost, ok := costs[e]; ok {
		if depth+cost.depth > maxReferenceDepth {
			return referenceCost{}, errLimit
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
	add := func(child *element) error {
		childCost, err := d.checkReferences(child, costs, active, depth+1)
		if err != nil {
			return err
		}
		if childCost.elements > maxExpandedElements-cost.elements {
			return errLimit
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
	// are resolved by the later property interpreter using resolve, which must
	// also bound any paint or CSS reference traversal it introduces.
	switch e.name.Local {
	case "use":
		ref, ok := e.attribute("", "href")
		if !ok {
			ref, ok = e.attribute(xlinkNamespace, "href")
		}
		if ok {
			target, err := d.resolve(ref)
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
	element *element
	text    strings.Builder
}

// parse parses a UTF-8 XML document already decoded by the font library.
// The returned document does not retain source.
func parse(source []byte) (*document, error) {
	// go-text/typesetting decompresses gzip before returning GlyphSVG.Source.
	// Its decompression allocation is unbounded; the byte limit here applies
	// only after that boundary.
	if len(source) > maxBytes {
		return nil, errLimit
	}
	if !utf8.Valid(source) {
		return nil, errXML
	}
	source = bytes.TrimPrefix(source, []byte{0xef, 0xbb, 0xbf})
	decoder := xml.NewDecoder(bytes.NewReader(source))
	// Token manages namespace scopes. Undeclared prefixes remain in Name.Space
	// and are treated as foreign. They cannot match the SVG or XLink namespace
	// URIs, which contain colons, so they cannot introduce SVG elements or hrefs.
	document := &document{
		ids: make(map[string]*element),
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
			return nil, fmt.Errorf("%w: %v", errXML, err)
		}
		switch token := token.(type) {
		case xml.StartElement:
			elements++
			attributes += len(token.Attr)
			if len(stack) >= maxDepth || elements > maxElements || attributes > maxAttributes {
				return nil, errLimit
			}
			if closed {
				return nil, errXML
			}
			var frame parseFrame
			seen := make(map[xml.Name]bool, len(token.Attr))
			for _, a := range token.Attr {
				if seen[a.Name] {
					return nil, errXML
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
					return nil, errNamespace
				}
			}
			// Accept an empty namespace for compatibility with go-text's SVG root-attribute handling.
			isSVG := token.Name.Space == "" || token.Name.Space == svgNamespace
			if len(stack) == 0 && (!isSVG || token.Name.Local != "svg") {
				return nil, errNamespace
			}
			var parent *element
			if len(stack) > 0 {
				parent = stack[len(stack)-1].element
			}
			if (len(stack) == 0 || parent != nil) && isSVG && !ignoredElement(token.Name.Local) {
				e := &element{
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
				if id, ok := e.attribute("", "id"); ok && id != "" && document.ids[id] == nil {
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
					return nil, errXML
				}
			} else if frame := stack[len(stack)-1]; frame.element != nil && frame.element.name.Local == "style" {
				_, _ = frame.text.Write(token)
			}
		case xml.Directive:
			// Ordinary DOCTYPE declarations are inert. External subsets are never
			// fetched, and internal subsets (including entity declarations) are
			// outside the supported profile.
			if document.root != nil || doctype {
				return nil, errXML
			}
			declaration, ok := bytes.CutPrefix(token, []byte("DOCTYPE"))
			if !ok || len(declaration) == 0 || !strings.ContainsRune(" \t\r\n", rune(declaration[0])) {
				return nil, errXML
			}
			if len(bytes.TrimSpace(declaration)) == 0 || bytes.IndexByte(declaration, '[') >= 0 {
				return nil, errXML
			}
			doctype = true
		case xml.ProcInst:
			if strings.EqualFold(token.Target, "xml") && (token.Target != "xml" || offset != 0) {
				return nil, errXML
			}
			// Processing instructions are never executed or used to load stylesheets.
		}
	}
	if document.root == nil || !closed {
		return nil, errXML
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
