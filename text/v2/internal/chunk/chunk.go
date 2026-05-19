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

// Package chunk places cache-friendly boundaries within a single line
// of text. Callers use the returned ranges as keys for per-chunk shape
// caches: an edit confined to one chunk reshapes only that chunk,
// rather than the whole line. Each returned range carries its bidi
// embedding level so callers can apply UAX #9 L2 to walk chunks in
// visual order during composition.
//
// Chunk boundaries fall at bidi-level transitions, sentence
// terminators, or whitespace within long no-terminator spans — none
// of which sit inside a typical ligature. Ligatures are therefore
// unlikely to be split across two chunks. This is a property of the
// boundary placement rather than a hard guarantee: exotic
// constructions could in principle straddle a cut, but for ordinary
// text the per-chunk shape matches whole-text shape closely enough
// to be a workable trade-off.
package chunk

import (
	"unicode"
	"unicode/utf8"

	"github.com/go-text/typesetting/bidi"

	"github.com/hajimehoshi/ebiten/v2/text/v2/internal/textutil"
)

// Chunk is a byte range [Start, End) within the original text, plus
// the bidi embedding level of every codepoint in the range. The level
// is constant across the range — chunks never span a bidi-level
// transition, so visual composition can apply UAX #9 L2 by treating
// each chunk as a single character at the given level.
type Chunk struct {
	Start int
	End   int
	Level bidi.Level
}

// fallbackBytes is the soft upper bound on chunk size. When a chunk
// has grown past this many bytes without hitting a sentence terminator,
// the chunker breaks at the next safe boundary instead of letting the
// chunk grow unbounded, so a single edit in a long no-terminator span
// (HTML, JSON, code blobs) reshapes only a portion of that span rather
// than the whole thing.
const fallbackBytes = 4 * 1024

// AppendChunks appends the chunks of text's first line to dst and
// returns the extended slice. paragraphLevel is the UAX #9 paragraph
// embedding level (0 for an LTR-base paragraph, 1 for RTL); it picks
// the default direction for the bidi pass and determines which level
// runs are "same-direction" runs that can be sentence-chunked. The
// appended chunks are in logical (source) order; each carries the bidi
// embedding level so composition can apply UAX #9 L2 to walk them in
// visual order.
func AppendChunks(dst []Chunk, text string, paragraphLevel bidi.Level) []Chunk {
	// Pure-LTR fast path: only when the paragraph base is LTR
	// (paragraphLevel even), where text without strong-RTL bytes
	// resolves to level 0 uniformly. Under an RTL base the same
	// bytes would resolve to level 2, so the bidi pass has to run
	// to get the level right. appendChunksForRun stops on the first
	// line break, so no first-line pre-trim is needed here.
	if paragraphLevel%2 == 0 && !mayContainStrongRTLInFirstLine(text) {
		return appendChunksForRun(dst, text, paragraphLevel, paragraphLevel, 0)
	}

	// Bidi-aware path. The bidi package limits its input at class-B
	// paragraph separators only, which is a narrower set than the
	// chunker's line-break codepoints (LS/VT/FF aren't B), so the
	// first-line slice has to be computed here before SegmentString.
	firstLine := text[:textutil.FirstLineLen(text)]
	if len(firstLine) == 0 {
		return append(dst, Chunk{0, 0, paragraphLevel})
	}
	defaultDir := bidi.LeftToRight
	if paragraphLevel%2 == 1 {
		defaultDir = bidi.RightToLeft
	}
	var p bidi.Paragraph
	runs := p.SegmentString(firstLine, defaultDir)
	if runs.NumRuns() == 0 {
		return append(dst, Chunk{0, len(firstLine), paragraphLevel})
	}

	var byteIdx int
	var runeIdx int
	for ri := range runs.NumRuns() {
		run := runs.Run(ri)
		// Walk runes in firstLine until reaching the rune-indexed
		// run boundary. The bidi package returns rune-indexed runs;
		// chunker output is byte-indexed.
		for runeIdx < run.Start {
			_, w := utf8.DecodeRuneInString(firstLine[byteIdx:])
			byteIdx += w
			runeIdx++
		}
		runStart := byteIdx
		for runeIdx < run.End {
			_, w := utf8.DecodeRuneInString(firstLine[byteIdx:])
			byteIdx += w
			runeIdx++
		}
		runEnd := byteIdx
		dst = appendChunksForRun(dst, firstLine[runStart:runEnd], run.Level, paragraphLevel, runStart)
	}
	return dst
}

// appendChunksForRun appends sentence-terminator chunks for one
// single-level bidi run to dst and returns the extended slice. text is
// the run's bytes, level is the bidi level stamped on every appended
// chunk, paragraphLevel is the paragraph base level (used to decide
// whether this run agrees with the paragraph direction), and
// byteOffset is added to each chunk's Start/End so the result is in
// source-text coordinates.
//
// The same-direction branch stops walking at the first line break
// in text, so callers that pass a full first-line slice and callers
// that pass an entire multi-line string both produce only first-line
// chunks. The disagreeing-direction branch (level%2 != paragraphLevel%2)
// emits text as a single chunk and assumes the caller has trimmed line
// breaks (the bidi-aware path in [AppendChunks] does this).
func appendChunksForRun(dst []Chunk, text string, level, paragraphLevel bidi.Level, byteOffset int) []Chunk {
	if len(text) == 0 {
		return append(dst, Chunk{byteOffset, byteOffset, level})
	}

	// Runs whose direction disagrees with the paragraph base are
	// emitted whole. A cut inside such a run would put the
	// terminator and trailing whitespace at a chunk boundary;
	// reshaping that chunk in isolation lets its bidi pass re-resolve
	// those NIs against sot/eot (the paragraph direction), flipping
	// their level and visually misplacing the whitespace.
	//
	// TODO: lift this by threading per-character bidi levels from the
	// whole-text pass through to shaping, so a chunk can be shaped at
	// its resolved level instead of re-running bidi in isolation.
	if level%2 != paragraphLevel%2 {
		return append(dst, Chunk{byteOffset, byteOffset + len(text), level})
	}

	var chunkStart int
	var i int
walk:
	for i < len(text) {
		// ".", "!", "?" are the only ASCII members of ATerm ∪ STerm;
		// any other ASCII byte cannot start a terminator cluster.
		if b := text[i]; b < 0x80 {
			switch b {
			case '\n', '\v', '\f', '\r':
				break walk
			}
			// Whitespace fallback: when the current chunk has
			// outgrown [fallbackBytes] without hitting a sentence
			// terminator, the chunker breaks at the next ASCII
			// space or tab so long no-terminator spans (HTML,
			// JSON, code without punctuation) don't collapse into
			// one giant chunk.
			if (b == ' ' || b == '\t') && i-chunkStart >= fallbackBytes {
				end := absorbExtendFormat(text, i+1)
				dst = append(dst, Chunk{byteOffset + chunkStart, byteOffset + end, level})
				chunkStart = end
				i = end
				continue
			}
			if b != '.' && b != '!' && b != '?' {
				i++
				continue
			}
		} else {
			r, w := utf8.DecodeRuneInString(text[i:])
			if textutil.IsLineBreak(r) {
				break walk
			}
			if !isATerm(r) && !isSTerm(r) {
				i += w
				continue
			}
		}

		// Found the start of a sentence-terminator cluster. Extend
		// across any consecutive ATerm/STerm runes so "?!" or
		// "..." stays one cluster, not many.
		clusterEnd := i
		var hasSTerm bool
		for clusterEnd < len(text) {
			if b := text[clusterEnd]; b < 0x80 {
				switch b {
				case '.':
					clusterEnd++
					continue
				case '!', '?':
					hasSTerm = true
					clusterEnd++
					continue
				}
				break
			}
			r, w := utf8.DecodeRuneInString(text[clusterEnd:])
			if isSTerm(r) {
				hasSTerm = true
				clusterEnd += w
				continue
			}
			if isATerm(r) {
				clusterEnd += w
				continue
			}
			break
		}

		if shouldSplitAfterTerminator(text, clusterEnd, hasSTerm) {
			splitEnd := absorbExtendFormat(text, clusterEnd)
			dst = append(dst, Chunk{byteOffset + chunkStart, byteOffset + splitEnd, level})
			chunkStart = splitEnd
			clusterEnd = splitEnd
		}

		i = clusterEnd
	}

	if chunkStart < i {
		dst = append(dst, Chunk{byteOffset + chunkStart, byteOffset + i, level})
	} else if i == 0 {
		// text started with a line break: emit a zero-width chunk
		// for the empty first line.
		dst = append(dst, Chunk{byteOffset, byteOffset, level})
	}
	return dst
}

// shouldSplitAfterTerminator decides whether the terminator cluster
// ending at clusterEnd is a sentence boundary. STerm clusters always
// are (e.g. "Hello!" or "What?!" → break). ATerm-only clusters break
// only when followed by whitespace and a non-lowercase next character
// (mirroring UAX #29 SB8): "Hello. World" breaks, "etc. Apples"
// breaks, but "Hello. and so on" / "etc. and so on" do not — the
// lowercase continuation suppresses the break for any ATerm,
// abbreviation or not.
func shouldSplitAfterTerminator(text string, clusterEnd int, hasSTerm bool) bool {
	if hasSTerm {
		return true
	}

	// ATerm-only: require whitespace, and the next non-whitespace
	// character (if any) must not be lowercase.
	var sawWhitespace bool
	j := clusterEnd
	for j < len(text) {
		r, w := utf8.DecodeRuneInString(text[j:])
		if !unicode.IsSpace(r) {
			break
		}
		sawWhitespace = true
		j += w
	}
	if !sawWhitespace {
		return false
	}
	if j == len(text) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(text[j:])
	return !unicode.IsLower(r)
}

// absorbExtendFormat returns the smallest offset >= pos at which the
// next codepoint is neither a combining mark nor a format character.
// The skip keeps trailing combining marks, variation selectors,
// joiners, and bidi-format characters bound to the base they attach
// to instead of orphaning them at a chunk boundary.
//
// Example: at a fallback cut between "abc " and a following U+200D
// (ZWJ) the next chunk would otherwise start with the U+200D,
// detached from any neighbor it could meaningfully join. The skip
// pulls it into the preceding chunk so the boundary lands at a base
// codepoint.
func absorbExtendFormat(text string, pos int) int {
	// Corresponds to UAX #29 WB4's X (Extend | Format | ZWJ)* tail,
	// approximated via general categories Mark and Cf rather than the
	// exact Word_Break property. The approximation is over-broad on Mc
	// (Spacing combining marks) and may miss the handful of
	// Word_Break = Extend characters that lie outside Mark and Cf;
	// over-broad is safe because the worst case is keeping a
	// non-attaching character with the preceding chunk.
	for pos < len(text) {
		if text[pos] < 0x80 {
			return pos
		}
		r, w := utf8.DecodeRuneInString(text[pos:])
		if !unicode.IsMark(r) && !unicode.Is(unicode.Cf, r) {
			return pos
		}
		pos += w
	}
	return pos
}

// mayContainStrongRTLInFirstLine reports whether text's first line
// contains any byte that could be the UTF-8 lead byte of a strong-RTL
// rune. It is an upper bound: every first line containing strong-RTL
// returns true, but a true result does not guarantee strong-RTL is
// present.
func mayContainStrongRTLInFirstLine(text string) bool {
	// Lead bytes that may begin a strong-RTL rune:
	//   - 0xD6..0xDF — 2-byte UTF-8 covering U+0590..U+07FF
	//     (Hebrew, Arabic, Syriac, Arabic Supplement, Thaana, NKo).
	//   - 0xE0       — 3-byte UTF-8 covering U+0800..U+08FF
	//     (Samaritan, Mandaic, Syriac Supplement, Arabic Extended-A/B).
	//   - 0xF0       — 4-byte UTF-8 covering plane 1
	//     (Mende Kikakui, Adlam, and other SMP RTL scripts).
	//
	// 0xF0 admits false positives for non-RTL SMP content (emoji,
	// mathematical alphanumerics); those texts go through the bidi
	// pass and chunk correctly anyway.
	//
	// Line-break bytes are detected inline so the scan stops at the
	// first line break instead of walking past it.
	//   ASCII: '\n' '\v' '\f' '\r'
	//   NEL  U+0085 → 0xC2 0x85
	//   LS   U+2028 → 0xE2 0x80 0xA8
	//   PS   U+2029 → 0xE2 0x80 0xA9
	for i := 0; i < len(text); i++ {
		b := text[i]
		if b < 0x80 {
			switch b {
			case '\n', '\v', '\f', '\r':
				return false
			}
			continue
		}
		if (b >= 0xD6 && b <= 0xE0) || b == 0xF0 {
			return true
		}
		if b == 0xC2 && i+1 < len(text) && text[i+1] == 0x85 {
			return false
		}
		if b == 0xE2 && i+2 < len(text) && text[i+1] == 0x80 && (text[i+2] == 0xA8 || text[i+2] == 0xA9) {
			return false
		}
	}
	return false
}
