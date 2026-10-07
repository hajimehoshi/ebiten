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

package opentypesvg

import "encoding/xml"

type Document = document
type Element = element

var Parse = parse

var (
	ErrXML            = errXML
	ErrNamespace      = errNamespace
	ErrLimit          = errLimit
	ErrGlyphNotFound  = errGlyphNotFound
	ErrReference      = errReference
	ErrReferenceCycle = errReferenceCycle
)

func (e *element) Name() xml.Name {
	return e.xmlName()
}

func (e *element) Attribute(space, local string) (string, bool) {
	return e.attribute(space, local)
}

func (e *element) AttributeCount() int {
	return e.attributeCount()
}

func (e *element) AttributeAt(index int) xml.Attr {
	return e.attributeAt(index)
}

func (e *element) SourceParent() *element {
	return e.sourceParent()
}

func (e *element) ChildCount() int {
	return e.childCount()
}

func (e *element) Child(index int) *element {
	return e.child(index)
}

func (e *element) Text() string {
	return e.styleText()
}

func (d *document) Root() *element {
	return d.rootElement()
}

func (d *document) StylesheetCount() int {
	return d.stylesheetCount()
}

func (d *document) Stylesheet(index int) *element {
	return d.stylesheet(index)
}

func (d *document) Resolve(reference string) (*element, error) {
	return d.resolve(reference)
}

func (d *document) SelectGlyph(gid uint16) (*element, error) {
	return d.selectGlyph(gid)
}
