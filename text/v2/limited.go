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
	"github.com/hajimehoshi/ebiten/v2/vector"
)

var _ Face = (*LimitedFace)(nil)

// LimitedFace is a Face with glyph limitations.
type LimitedFace struct {
	face          Face
	unicodeRanges unicodeRanges
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
	l.unicodeRanges.add(start, end)
}

// Metrics implements Face.
func (l *LimitedFace) Metrics() Metrics {
	return l.face.Metrics()
}

// advance implements Face.
func (l *LimitedFace) advance(text string) float64 {
	return l.face.advance(l.unicodeRanges.filter(text))
}

// hasGlyph implements Face.
func (l *LimitedFace) hasGlyph(r rune) bool {
	return l.unicodeRanges.contains(r) && l.face.hasGlyph(r)
}

// appendGlyphsForLine implements Face.
func (l *LimitedFace) appendGlyphsForLine(glyphs []Glyph, line string, indexOffset int, originX, originY float64) []Glyph {
	return l.face.appendGlyphsForLine(glyphs, l.unicodeRanges.filter(line), indexOffset, originX, originY)
}

// appendVectorPathForLine implements Face.
func (l *LimitedFace) appendVectorPathForLine(path *vector.Path, line string, originX, originY float64) {
	l.face.appendVectorPathForLine(path, l.unicodeRanges.filter(line), originX, originY)
}

// direction implements Face.
func (l *LimitedFace) direction() Direction {
	return l.face.direction()
}

// private implements Face.
func (l *LimitedFace) private() {
}

type unicodeRange struct {
	start rune
	end   rune
}

type unicodeRanges struct {
	ranges []unicodeRange
}

func (u *unicodeRanges) add(start, end rune) {
	u.ranges = append(u.ranges, unicodeRange{
		start: start,
		end:   end,
	})
}

func (u *unicodeRanges) contains(r rune) bool {
	for _, rg := range u.ranges {
		if rg.start <= r && r <= rg.end {
			return true
		}
	}
	return false
}

func (u *unicodeRanges) filter(str string) string {
	var rs []rune
	for _, r := range str {
		if !u.contains(r) {
			// U+FFFD is "REPLACEMENT CHARACTER".
			r = '\ufffd'
		}
		rs = append(rs, r)
	}
	return string(rs)
}
