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

package textinput

import (
	"io"
	"slices"
)

type opType int

const (
	opTypeIME opType = iota
	opTypeOneNewLine
	opTypeDelete
	opTypeOther
)

type lastOp struct {
	valid bool
	typ   opType
}

type pieceTable struct {
	table []byte

	history      []historyItem
	historyIndex int
	lastOp       lastOp
}

type historyItem struct {
	items []pieceTableItem

	undoSelectionStart int
	undoSelectionEnd   int
	redoSelectionStart int
	redoSelectionEnd   int
}

type pieceTableItem struct {
	start int
	end   int
}

func (p *pieceTable) items() []pieceTableItem {
	if len(p.history) == 0 {
		return nil
	}
	return p.history[p.historyIndex].items
}

func (p *pieceTable) WriteTo(w io.Writer) (int64, error) {
	var n int64
	items := p.items()
	for i := range items {
		item := &items[i]
		nn, err := w.Write(p.table[item.start:item.end])
		n += int64(nn)
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// writeRangeTo writes the bytes of the current text in [start, end) to w.
// start and end are clamped to [0, Len()]; if start >= end after clamping, nothing is written.
func (p *pieceTable) writeRangeTo(w io.Writer, start, end int) (int64, error) {
	l := p.Len()
	start = max(start, 0)
	end = min(end, l)
	if start >= end {
		return 0, nil
	}

	var n int64
	var offset int
	items := p.items()
	for i := range items {
		item := &items[i]
		itemLen := item.end - item.start
		itemEnd := offset + itemLen

		if itemEnd <= start {
			offset = itemEnd
			continue
		}
		if offset >= end {
			break
		}

		readStart := item.start + max(start-offset, 0)
		readEnd := item.start + min(end-offset, itemLen)

		nn, err := w.Write(p.table[readStart:readEnd])
		n += int64(nn)
		if err != nil {
			return n, err
		}

		offset = itemEnd
	}
	return n, nil
}

// writeRangeToWithInsertion writes the bytes of the rendering text in [rangeStart, rangeEnd) to w.
// The rendering text is the conceptual stream
//
//	committed[:insertStart] ++ text ++ committed[insertEnd:]
//
// where committed is the current piece-table content. rangeStart and rangeEnd are clamped to
// [0, renderingLength] where renderingLength = Len() - (insertEnd - insertStart) + len(text).
// If the clamped start is not less than the clamped end, nothing is written.
func (p *pieceTable) writeRangeToWithInsertion(w io.Writer, text string, insertStart, insertEnd, rangeStart, rangeEnd int) (int64, error) {
	pl := p.Len()
	insertLen := len(text)
	selLen := insertEnd - insertStart
	totalLen := pl - selLen + insertLen

	rangeStart = max(rangeStart, 0)
	rangeEnd = min(rangeEnd, totalLen)
	if rangeStart >= rangeEnd {
		return 0, nil
	}

	var n int64

	// 1. Committed prefix overlap with [rangeStart, rangeEnd).
	if rangeStart < insertStart {
		prefixEnd := min(rangeEnd, insertStart)
		nn, err := p.writeRangeTo(w, rangeStart, prefixEnd)
		n += nn
		if err != nil {
			return n, err
		}
	}

	// 2. Composition overlap with [rangeStart, rangeEnd).
	if ts, te := max(0, rangeStart-insertStart), min(insertLen, rangeEnd-insertStart); ts < te {
		nn, err := io.WriteString(w, text[ts:te])
		n += int64(nn)
		if err != nil {
			return n, err
		}
	}

	// 3. Committed suffix overlap with [rangeStart, rangeEnd).
	if rangeEnd > insertStart+insertLen {
		suffixRangeStart := max(rangeStart, insertStart+insertLen)
		ptStart := suffixRangeStart - insertLen + selLen
		ptEnd := rangeEnd - insertLen + selLen
		nn, err := p.writeRangeTo(w, ptStart, ptEnd)
		n += nn
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func (p *pieceTable) writeToWithInsertion(w io.Writer, text string, start, end int) (int64, error) {
	var n int64
	var offset int
	var insertedTextWritten bool

	if start == 0 {
		nn, err := io.WriteString(w, text)
		n += int64(nn)
		if err != nil {
			return n, err
		}
		insertedTextWritten = true
	}

	items := p.items()
	for i := range items {
		item := &items[i]
		itemLen := item.end - item.start

		// Part before the replaced range
		if offset < start {
			if toWrite := min(itemLen, start-offset); toWrite > 0 {
				nn, err := w.Write(p.table[item.start : item.start+toWrite])
				n += int64(nn)
				if err != nil {
					return n, err
				}
			}
		}

		if !insertedTextWritten && offset+itemLen >= start {
			nn, err := io.WriteString(w, text)
			n += int64(nn)
			if err != nil {
				return n, err
			}
			insertedTextWritten = true
		}

		// Part after the replaced range
		if offset+itemLen > end {
			if startRead := max(0, end-offset); startRead < itemLen {
				nn, err := w.Write(p.table[item.start+startRead : item.end])
				n += int64(nn)
				if err != nil {
					return n, err
				}
			}
		}

		offset += itemLen
	}

	if !insertedTextWritten {
		nn, err := io.WriteString(w, text)
		n += int64(nn)
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

func (p *pieceTable) hasText() bool {
	for _, item := range p.items() {
		if item.start < item.end {
			return true
		}
	}
	return false
}

func (p *pieceTable) Len() int {
	var n int
	items := p.items()
	for i := range items {
		item := &items[i]
		n += item.end - item.start
	}
	return n
}

// findLineBounds returns the byte offsets bounding the line that contains the
// selection [selStart, selEnd]. lineStart is the position right after the
// previous line break (or 0 if none), and lineEnd is the position of the next
// line break (or Len() if none). The line break bytes themselves are excluded
// from both ends.
//
// Line breaks that fall within [selStart, selEnd) are ignored, so a selection
// crossing line breaks yields a single combined line view.
func (p *pieceTable) findLineBounds(selStart, selEnd int) (lineStart, lineEnd int) {
	l := p.Len()
	selStart = min(max(selStart, 0), l)
	selEnd = min(max(selEnd, selStart), l)

	lineEnd = l

	items := p.items()

	// peekByte returns the byte at offset bytes after (pi, bi).
	peekByte := func(pi, bi, offset int) (byte, bool) {
		bi += offset
		for pi < len(items) {
			chunkLen := items[pi].end - items[pi].start
			if bi < chunkLen {
				return p.table[items[pi].start+bi], true
			}
			bi -= chunkLen
			pi++
		}
		return 0, false
	}

	// peekByteBack returns the byte at offset bytes before (pi, bi).
	peekByteBack := func(pi, bi, offset int) (byte, bool) {
		bi -= offset
		for bi < 0 {
			pi--
			if pi < 0 {
				return 0, false
			}
			bi += items[pi].end - items[pi].start
		}
		return p.table[items[pi].start+bi], true
	}

	// findPiece returns (pieceIdx, byteIdxWithinPiece) for absolute byte
	// position pos. Returns (len(items), 0) if pos is past the end.
	findPiece := func(pos int) (int, int) {
		var offset int
		for i, item := range items {
			chunkLen := item.end - item.start
			if pos < offset+chunkLen {
				return i, pos - offset
			}
			offset += chunkLen
		}
		return len(items), 0
	}

	// Scan backward from selStart-1 for the latest line break ending at or
	// before selStart. The first line break encountered going backward is by
	// definition the latest.
	if selStart > 0 {
		pi, bi := findPiece(selStart - 1)
		absPos := selStart - 1
		for pi >= 0 {
			b := p.table[items[pi].start+bi]
			var isLB bool
			switch b {
			case 0x0A, 0x0B, 0x0C, 0x0D: // LF, VT, FF, CR
				isLB = true
			case 0x85: // possible NEL last byte (0xC2 0x85)
				if prev, ok := peekByteBack(pi, bi, 1); ok && prev == 0xC2 {
					isLB = true
				}
			case 0xA8, 0xA9: // possible LS/PS last byte (0xE2 0x80 0xA8/0xA9)
				if b1, ok := peekByteBack(pi, bi, 1); ok && b1 == 0x80 {
					if b2, ok := peekByteBack(pi, bi, 2); ok && b2 == 0xE2 {
						isLB = true
					}
				}
			}
			if isLB {
				lineStart = absPos + 1
				break
			}
			// Step backward.
			if bi > 0 {
				bi--
			} else {
				pi--
				if pi < 0 {
					break
				}
				bi = items[pi].end - items[pi].start - 1
			}
			absPos--
		}
	}

	// Scan forward from selEnd for the earliest line break starting at or
	// after selEnd.
	pi, bi := findPiece(selEnd)
	absPos := selEnd
	for pi < len(items) {
		b := p.table[items[pi].start+bi]
		var isLB bool
		switch b {
		case 0x0A, 0x0B, 0x0C, 0x0D: // LF, VT, FF, CR (alone or as part of CRLF)
			isLB = true
		case 0xC2: // possible NEL first byte
			if next, ok := peekByte(pi, bi, 1); ok && next == 0x85 {
				isLB = true
			}
		case 0xE2: // possible LS/PS first byte
			if next1, ok := peekByte(pi, bi, 1); ok && next1 == 0x80 {
				if next2, ok := peekByte(pi, bi, 2); ok && (next2 == 0xA8 || next2 == 0xA9) {
					isLB = true
				}
			}
		}
		if isLB {
			lineEnd = absPos
			return
		}
		// Step forward.
		bi++
		if bi >= items[pi].end-items[pi].start {
			pi++
			bi = 0
		}
		absPos++
	}
	return
}

func (p *pieceTable) reset(text string) {
	p.table = p.table[:0]
	p.table = append(p.table, text...)
	p.resetHistory()
}

// readFrom resets the piece table by reading bytes from r until EOF.
// Unlike [bytes.Buffer.ReadFrom], readFrom does not append: any prior content is discarded.
//
// The return value is the number of bytes read.
// On non-EOF error, the piece table is left in an empty state and the error is returned.
func (p *pieceTable) readFrom(r io.Reader) (int64, error) {
	p.table = p.table[:0]
	var total int64
	const minRead = 512
	for {
		p.table = slices.Grow(p.table, minRead)
		n, err := r.Read(p.table[len(p.table):cap(p.table)])
		p.table = p.table[:len(p.table)+n]
		total += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			p.table = p.table[:0]
			p.resetHistory()
			return total, err
		}
	}
	p.resetHistory()
	return total, nil
}

func (p *pieceTable) resetHistory() {
	p.history = p.history[:0]
	p.history = append(p.history, historyItem{
		items: []pieceTableItem{
			{
				start: 0,
				end:   len(p.table),
			},
		},
		undoSelectionStart: 0,
		undoSelectionEnd:   len(p.table),
		redoSelectionStart: 0,
		redoSelectionEnd:   len(p.table),
	})
	p.historyIndex = 0
	p.lastOp = lastOp{}
}

func (p *pieceTable) replace(text string, start, end int) {
	p.maybeAppendHistory(text, start, end, 0, 0, 1, false)
	p.doReplace(text, start, end)
}

func (p *pieceTable) doReplace(text string, start, end int) {
	items := p.history[p.historyIndex].items

	// Append the new text to the table.
	newTextStart := len(p.table)
	p.table = append(p.table, text...)
	newTextEnd := len(p.table)

	// Calculate the range of items to replace.
	var startItemIndex, endItemIndex int

	// Find the first intersecting item.
	var offset int
	for startItemIndex < len(items) {
		item := &items[startItemIndex]
		itemLen := item.end - item.start
		if offset+itemLen > start {
			break
		}
		offset += itemLen
		startItemIndex++
	}
	startItemOffset := offset

	// Find the last intersecting item.
	endItemIndex = startItemIndex
	for endItemIndex < len(items) {
		item := items[endItemIndex]
		itemLen := item.end - item.start
		if offset+itemLen >= end {
			break
		}
		offset += itemLen
		endItemIndex++
	}
	endItemOffset := offset

	// Prepare new items.
	var newItems [3]pieceTableItem
	var newItemsCount int

	// 1. Prefix of the first affected item.
	if startItemIndex < len(items) {
		if s := start - startItemOffset; s > 0 {
			item := &items[startItemIndex]
			newItems[newItemsCount] = pieceTableItem{
				start: item.start,
				end:   item.start + s,
			}
			newItemsCount++
		}
	}

	// 2. The new text.
	if newTextEnd > newTextStart {
		newItems[newItemsCount] = pieceTableItem{
			start: newTextStart,
			end:   newTextEnd,
		}
		newItemsCount++
	}

	// 3. Suffix of the last affected item.
	if endItemIndex < len(items) {
		item := &items[endItemIndex]
		if e := end - endItemOffset; e < item.end-item.start {
			newItems[newItemsCount] = pieceTableItem{
				start: item.start + e,
				end:   item.end,
			}
			newItemsCount++
		}
	}

	// Determine the number of items currently occupying the range to be replaced.
	var oldItemsCount int
	if endItemIndex < len(items) {
		oldItemsCount = endItemIndex - startItemIndex + 1
	} else {
		oldItemsCount = len(items) - startItemIndex
	}

	// Adjust the slice.
	newLen := len(items) - oldItemsCount + newItemsCount
	if newLen > cap(items) {
		newSlice := make([]pieceTableItem, newLen)
		copy(newSlice, items[:startItemIndex])
		copy(newSlice[startItemIndex+newItemsCount:], items[startItemIndex+oldItemsCount:])
		items = newSlice
	} else {
		if newLen > len(items) {
			items = items[:newLen]
		}
		copy(items[startItemIndex+newItemsCount:], items[startItemIndex+oldItemsCount:])
		if newLen < len(items) {
			items = items[:newLen]
		}
	}

	copy(items[startItemIndex:], newItems[:newItemsCount])
	p.history[p.historyIndex].items = items
}

func (p *pieceTable) updateByIME(text string, replacementStart, replacementEnd, start, end int) int {
	if delta := replacementEnd - replacementStart; delta > 0 {
		if start > replacementStart {
			start -= delta
		}
		if end > replacementStart {
			end -= delta
		}
		p.maybeAppendHistory(text, replacementStart, replacementEnd, start, end, 2, true)
	} else {
		p.maybeAppendHistory(text, start, end, 0, 0, 1, true)
	}

	if replacementEnd-replacementStart > 0 {
		p.doReplace("", replacementStart, replacementEnd)
	}
	p.doReplace(text, start, end)
	return start
}

func (p *pieceTable) canUndo() bool {
	return p.historyIndex > 0
}

func (p *pieceTable) canRedo() bool {
	return p.historyIndex < len(p.history)-1
}

func (p *pieceTable) undo() (int, int, bool) {
	if !p.canUndo() {
		return 0, 0, false
	}
	item := p.history[p.historyIndex]
	p.historyIndex--
	p.lastOp.valid = false
	return item.undoSelectionStart, item.undoSelectionEnd, true
}

func (p *pieceTable) redo() (int, int, bool) {
	if !p.canRedo() {
		return 0, 0, false
	}
	p.historyIndex++
	p.lastOp.valid = false
	item := p.history[p.historyIndex]
	return item.redoSelectionStart, item.redoSelectionEnd, true
}

func (p *pieceTable) maybeAppendHistory(text string, start1, end1 int, start2, end2 int, rangeCount int, fromIME bool) {
	// If the history is empty, initialize it.
	if p.history == nil {
		p.history = []historyItem{{}}
	}

	var opType opType
	switch {
	case text == "\n":
		opType = opTypeOneNewLine
	case fromIME:
		opType = opTypeIME
	case text == "":
		opType = opTypeDelete
	default:
		opType = opTypeOther
	}

	// Check if the piece table can merge this operation with the last one.
	var merge bool
	if len(p.history) > 0 &&
		p.lastOp.valid &&
		((p.lastOp.typ == opTypeIME && (opType == opTypeIME || opType == opTypeOneNewLine)) ||
			(p.lastOp.typ == opTypeDelete && opType == opTypeDelete)) {
		item := &p.history[p.historyIndex]
		if start1 == item.redoSelectionStart || start1 == item.redoSelectionEnd ||
			end1 == item.redoSelectionStart || end1 == item.redoSelectionEnd {
			merge = true
		}
	}

	p.lastOp.valid = true
	p.lastOp.typ = opType

	if !merge {
		if start2 != end2 {
			p.appendHistory(start1, end1, start2, start2+len(text))
		} else {
			p.appendHistory(start1, end1, start1, start1+len(text))
		}
		return
	}

	item := &p.history[p.historyIndex]
	if opType == opTypeDelete {
		if end1 == item.redoSelectionStart {
			item.undoSelectionStart = start1
		} else if start1 == item.redoSelectionStart {
			item.undoSelectionEnd += end1 - start1
		}
	}

	if rangeCount == 2 {
		item.redoSelectionStart = min(item.redoSelectionStart, start2)
		if opType != opTypeDelete {
			item.redoSelectionEnd = max(item.redoSelectionEnd, start2+len(text))
		} else {
			item.redoSelectionEnd = item.redoSelectionStart
		}
	} else {
		item.redoSelectionStart = min(item.redoSelectionStart, start1)
		if opType != opTypeDelete {
			item.redoSelectionEnd = max(item.redoSelectionEnd, start1+len(text))
		} else {
			item.redoSelectionEnd = item.redoSelectionStart
		}
	}
}

func (p *pieceTable) appendHistory(undoStart, undoEnd, redoStart, redoEnd int) {
	// Truncate the history.
	if p.historyIndex < len(p.history)-1 {
		p.history = p.history[:p.historyIndex+1]
	}

	// Append the current items (cloned) to the history.
	// As doReplace modifies the underlying array, duplicate the items here.
	newItems := append([]pieceTableItem(nil), p.history[p.historyIndex].items...)
	p.history = append(p.history, historyItem{
		items:              newItems,
		undoSelectionStart: undoStart,
		undoSelectionEnd:   undoEnd,
		redoSelectionStart: redoStart,
		redoSelectionEnd:   redoEnd,
	})
	p.historyIndex++
}
