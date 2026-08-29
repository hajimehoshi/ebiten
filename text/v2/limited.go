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
	"slices"
	"sync"
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

	// The appended glyphs' indices are indices in filtered, not in line, so
	// they have to be translated back (see limitedFilterMapping).
	// The buffer is pooled for this is in the text rendering hot path.
	mappingP := theLimitedFilterMappingPool.Get().(*[]int)
	var mapping []int
	defer func() {
		*mappingP = mapping[:0]
		theLimitedFilterMappingPool.Put(mappingP)
	}()
	mapping = limitedFilterMapping((*mappingP)[:0], line, filtered)

	before := len(glyphs)
	glyphs = l.face.appendLazyGlyphsForLine(glyphs, filtered, indexOffset, originX, originY, keepGlyph)
	// The indices of glyphs are always in the range [0, len(filtered)], where
	// mapping has len(filtered)+1 entries.
	for i := before; i < len(glyphs); i++ {
		glyphs[i].StartIndexInBytes = mapping[glyphs[i].StartIndexInBytes-indexOffset] + indexOffset
		glyphs[i].EndIndexInBytes = mapping[glyphs[i].EndIndexInBytes-indexOffset] + indexOffset
	}
	return glyphs
}

var theLimitedFilterMappingPool = sync.Pool{
	New: func() any {
		// 64 is an arbitrary number for the initial capacity.
		s := make([]int, 0, 64)
		// Return a pointer instead of a slice, or go-vet warns at Put.
		return &s
	},
}

// limitedFilterMapping returns a mapping from a byte index in filtered to a
// byte index in line, where filtered is UnicodeRanges.Filter(line). buf is
// reused as the returned slice's backing array, and its contents are
// discarded.
//
// Filter replaces each unsupported rune with U+FFFD, whose byte length can
// differ from the replaced rune's length, so the indices are translated rune
// by rune. The mapping has len(filtered)+1 entries so that the end index of
// the last rune can be mapped, and the last entry is len(line). The values at
// indices in the middle of a rune in filtered are arbitrary, but a glyph
// index is always at a rune boundary.
func limitedFilterMapping(buf []int, line, filtered string) []int {
	mapping := slices.Grow(buf, len(filtered)+1)[:len(filtered)+1]
	var oi, fi int
	for oi < len(line) && fi < len(filtered) {
		_, size := utf8.DecodeRuneInString(line[oi:])
		_, fsize := utf8.DecodeRuneInString(filtered[fi:])
		for k := range fsize {
			if k < size {
				mapping[fi+k] = oi + k
			} else {
				mapping[fi+k] = oi
			}
		}
		oi += size
		fi += fsize
	}
	for ; fi <= len(filtered); fi++ {
		mapping[fi] = oi
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
