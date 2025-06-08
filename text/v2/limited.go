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
	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/textutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var _ Face = (*LimitedFace)(nil)

// LimitedFace is a Face with glyph limitations.
type LimitedFace struct {
	face          Face
	unicodeRanges textutil.UnicodeRanges
}

// NewLimitedFace creates a new LimitedFace from the given face.
// In the default state, glyphs for any runes are limited and not rendered.
// You have to call AddUnicodeRange to add allowed glyphs.
func NewLimitedFace(face Face) *LimitedFace {
	return &LimitedFace{
		face: face,
	}
}

// AddUnicodeRange adds a rune range for rendered glyphs.
// A range is inclusive, which means that a range contains the specified rune end.
func (l *LimitedFace) AddUnicodeRange(start, end rune) {
	l.unicodeRanges.Add(start, end)
}

// Metrics implements Face.
func (l *LimitedFace) Metrics() Metrics {
	return l.face.Metrics()
}

// advance implements Face.
func (l *LimitedFace) advance(text string) float64 {
	return l.face.advance(l.unicodeRanges.Filter(text))
}

// hasGlyph implements Face.
func (l *LimitedFace) hasGlyph(r rune) bool {
	return l.unicodeRanges.Contains(r) && l.face.hasGlyph(r)
}

// appendGlyphsForLine implements Face.
func (l *LimitedFace) appendGlyphsForLine(glyphs []Glyph, line string, indexOffset int, originX, originY float64) []Glyph {
	return l.face.appendGlyphsForLine(glyphs, l.unicodeRanges.Filter(line), indexOffset, originX, originY)
}

// appendVectorPathForLine implements Face.
func (l *LimitedFace) appendVectorPathForLine(path *vector.Path, line string, originX, originY float64) {
	l.face.appendVectorPathForLine(path, l.unicodeRanges.Filter(line), originX, originY)
}

// direction implements Face.
func (l *LimitedFace) direction() Direction {
	return l.face.direction()
}

// private implements Face.
func (l *LimitedFace) private() {
}
