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
	"unicode/utf8"

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

// advanceAt implements Face.
func (l *LimitedFace) advanceAt(text string, indexInBytes int) float64 {
	firstLineLen := textutil.FirstLineLen(text)
	indexInBytes = min(indexInBytes, firstLineLen)
	firstLine := text[:firstLineLen]
	filtered := l.unicodeRanges.Filter(firstLine)
	if filtered == firstLine {
		return l.face.advanceAt(filtered, indexInBytes)
	}
	// Filter substitutes unsupported runes with U+FFFD (3 bytes), so byte
	// offsets in text don't match those in filtered. Translate indexInBytes
	// into the filtered string's byte space.
	const fffdLen = 3
	var filteredIdx int
	for i, r := range firstLine {
		_, runeLen := utf8.DecodeRuneInString(firstLine[i:])
		if runeLen < 0 {
			runeLen = 1
		}
		if i+runeLen > indexInBytes {
			break
		}
		if l.unicodeRanges.Contains(r) {
			filteredIdx += runeLen
		} else {
			filteredIdx += fffdLen
		}
	}
	return l.face.advanceAt(filtered, filteredIdx)
}

// hasGlyph implements Face.
func (l *LimitedFace) hasGlyph(r rune) bool {
	return l.unicodeRanges.Contains(r) && l.face.hasGlyph(r)
}

// appendLazyGlyphsForLine implements Face.
func (l *LimitedFace) appendLazyGlyphsForLine(glyphs []LazyGlyph, line string, indexOffset int, originX, originY float64, keepGlyph func(originX, originY float64) bool) []LazyGlyph {
	filtered := l.unicodeRanges.Filter(line)
	if filtered == line {
		return l.face.appendLazyGlyphsForLine(glyphs, filtered, indexOffset, originX, originY, keepGlyph)
	}

	mapping := limitedFilterMapping(line, filtered)

	before := len(glyphs)
	glyphs = l.face.appendLazyGlyphsForLine(glyphs, filtered, indexOffset, originX, originY, keepGlyph)
	for i := before; i < len(glyphs); i++ {
		start := glyphs[i].StartIndexInBytes - indexOffset
		end := glyphs[i].EndIndexInBytes - indexOffset
		if start < 0 || start >= len(mapping) || end < 0 || end >= len(mapping) {
			continue
		}
		glyphs[i].StartIndexInBytes = mapping[start] + indexOffset
		glyphs[i].EndIndexInBytes = mapping[end] + indexOffset
	}
	return glyphs
}

func limitedFilterMapping(line, filtered string) []int {
	mapping := make([]int, len(filtered)+1)
	var oi, fi int
	for oi < len(line) && fi < len(filtered) {
		r, size := utf8.DecodeRuneInString(line[oi:])
		if size <= 0 {
			size = 1
		}
		fr, fsize := utf8.DecodeRuneInString(filtered[fi:])
		if fsize <= 0 {
			fsize = 1
		}
		if r == fr {
			for k := range size {
				mapping[fi+k] = oi + k
			}
			oi += size
			fi += fsize
			continue
		}
		const fffdLen = 3
		mapping[fi] = oi
		mapping[fi+1] = oi
		mapping[fi+2] = oi
		mapping[fi+fffdLen] = oi + size
		oi += size
		fi += fffdLen
	}
	for fi <= len(filtered) {
		mapping[fi] = oi
		fi++
	}
	return mapping
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
