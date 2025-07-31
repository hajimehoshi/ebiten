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

package text

import (
	"errors"
	"iter"
	"unicode/utf8"

	"github.com/hajimehoshi/ebiten/v2/vector"
)

var _ Face = (*MultiFace)(nil)

// MultiFace is a Face that consists of multiple Face objects.
// The face in the first index is used in the highest priority, and the last the lowest priority.
//
// There is a known issue: if the writing directions of the faces don't agree, the rendering result might be messed up.
type MultiFace struct {
	faces []Face
}

// NewMultiFace creates a new MultiFace from the given faces.
//
// NewMultiFace returns an error when no faces are given, or the faces' directions don't agree.
func NewMultiFace(faces ...Face) (*MultiFace, error) {
	if len(faces) == 0 {
		return nil, errors.New("text: no faces are given at NewMultiFace")
	}

	d := faces[0].direction()
	for _, f := range faces[1:] {
		if f.direction() != d {
			return nil, errors.New("text: all the faces' directions must agree at NewMultiFace")
		}
	}

	m := &MultiFace{}
	m.faces = make([]Face, len(faces))
	copy(m.faces, faces)
	return m, nil
}

// Metrics implements Face.
func (m *MultiFace) Metrics() Metrics {
	var mt Metrics
	for _, f := range m.faces {
		mt1 := f.Metrics()
		if mt1.HLineGap > mt.HLineGap {
			mt.HLineGap = mt1.HLineGap
		}
		if mt1.HAscent > mt.HAscent {
			mt.HAscent = mt1.HAscent
		}
		if mt1.HDescent > mt.HDescent {
			mt.HDescent = mt1.HDescent
		}
		if mt1.VLineGap > mt.VLineGap {
			mt.VLineGap = mt1.VLineGap
		}
		if mt1.VAscent > mt.VAscent {
			mt.VAscent = mt1.VAscent
		}
		if mt1.VDescent > mt.VDescent {
			mt.VDescent = mt1.VDescent
		}
		if mt1.XHeight > mt.XHeight {
			mt.XHeight = mt1.XHeight
		}
		if mt1.CapHeight > mt.CapHeight {
			mt.CapHeight = mt1.CapHeight
		}
	}
	return mt
}

// advance implements Face.
func (m *MultiFace) advance(text string) float64 {
	var a float64
	for c := range m.splitText(text) {
		if c.faceIndex == -1 {
			continue
		}
		f := m.faces[c.faceIndex]
		a += f.advance(text[c.textStartIndex:c.textEndIndex])
	}
	return a
}

// hasGlyph implements Face.
func (m *MultiFace) hasGlyph(r rune) bool {
	for _, f := range m.faces {
		if f.hasGlyph(r) {
			return true
		}
	}
	return false
}

// appendGlyphsForLine implements Face.
func (m *MultiFace) appendGlyphsForLine(glyphs []Glyph, line string, indexOffset int, originX, originY float64) []Glyph {
	for c := range m.splitText(line) {
		if c.faceIndex == -1 {
			continue
		}
		f := m.faces[c.faceIndex]
		t := line[c.textStartIndex:c.textEndIndex]
		glyphs = f.appendGlyphsForLine(glyphs, t, indexOffset, originX, originY)
		if a := f.advance(t); f.direction().isHorizontal() {
			originX += a
		} else {
			originY += a
		}
		indexOffset += len(t)
	}
	return glyphs
}

// appendVectorPathForLine implements Face.
func (m *MultiFace) appendVectorPathForLine(path *vector.Path, line string, originX, originY float64) {
	for c := range m.splitText(line) {
		if c.faceIndex == -1 {
			continue
		}
		f := m.faces[c.faceIndex]
		t := line[c.textStartIndex:c.textEndIndex]
		f.appendVectorPathForLine(path, t, originX, originY)
		if a := f.advance(t); f.direction().isHorizontal() {
			originX += a
		} else {
			originY += a
		}
	}
}

// direction implements Face.
func (m *MultiFace) direction() Direction {
	if len(m.faces) == 0 {
		return DirectionLeftToRight
	}
	return m.faces[0].direction()
}

// private implements Face.
func (m *MultiFace) private() {
}

type textChunk struct {
	textStartIndex int
	textEndIndex   int
	faceIndex      int
}

func (m *MultiFace) splitText(text string) iter.Seq[textChunk] {
	return func(yield func(textChunk) bool) {
		var chunk textChunk
		for i, r := range text {
			fi := -1
			for i, f := range m.faces {
				if !f.hasGlyph(r) && i < len(m.faces)-1 {
					continue
				}
				fi = i
				break
			}
			if fi == -1 {
				panic("text: a face was not selected correctly")
			}

			// Do not use utf8.RuneLen here, as r may be U+FFFD (replacement character)
			// when the line contains invalid UTF-8 sequences (#3284).
			_, l := utf8.DecodeRuneInString(text[i:])
			if l < 0 {
				// A string for-loop iterator advances by 1 byte when it encounters an invalid UTF-8 sequence.
				l = 1
			}

			var s int
			if chunk != (textChunk{}) {
				if chunk.faceIndex == fi {
					chunk.textEndIndex += l
					continue
				}
				if !yield(chunk) {
					return
				}
				s = chunk.textEndIndex
			}
			chunk = textChunk{
				textStartIndex: s,
				textEndIndex:   s + l,
				faceIndex:      fi,
			}
		}
		if chunk != (textChunk{}) {
			if !yield(chunk) {
				return
			}
		}
	}
}
