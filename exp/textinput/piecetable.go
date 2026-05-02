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
	"unicode/utf8"
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

// utf16CountToByteCount returns the byte offset in the piece table content
// that corresponds to c UTF-16 code units from the start, or -1 if c is past
// the end. The content is assumed to be valid UTF-8.
//
// This could be implemented by materializing the content into a string and
// calling convertUTF16CountToByteCount, but walking the pieces directly is
// faster.
func (p *pieceTable) utf16CountToByteCount(c int) int {
	if c == 0 {
		return 0
	}
	if c < 0 {
		return -1
	}
	var (
		byteOff  int
		utf16Off int
	)
	items := p.items()
	for i := range items {
		item := &items[i]
		chunk := p.table[item.start:item.end]
		if isASCIIBytes(chunk) {
			// In ASCII the UTF-16 unit count equals the byte count, so we
			// can skip past whole chunks (and target into one) by
			// arithmetic.
			remaining := c - utf16Off
			if remaining <= len(chunk) {
				return byteOff + remaining
			}
			byteOff += len(chunk)
			utf16Off += len(chunk)
			continue
		}
		var j int
		for j < len(chunk) {
			r, size := utf8.DecodeRune(chunk[j:])
			if r == utf8.RuneError && size <= 1 {
				return -1
			}
			l16 := 1
			if r >= 0x10000 {
				l16 = 2
			}
			utf16Off += l16
			if utf16Off >= c {
				return byteOff + j + size
			}
			j += size
		}
		byteOff += len(chunk)
	}
	return -1
}

// byteCountToUTF16Count returns the UTF-16 code-unit count of the first c
// bytes of the piece table content, or -1 if c is past the end.
func (p *pieceTable) byteCountToUTF16Count(c int) int {
	if c == 0 {
		return 0
	}
	if c < 0 {
		return -1
	}
	var (
		byteOff  int
		utf16Off int
	)
	items := p.items()
	for i := range items {
		item := &items[i]
		chunk := p.table[item.start:item.end]
		chunkLen := len(chunk)
		if isASCIIBytes(chunk) {
			if c <= byteOff+chunkLen {
				return utf16Off + (c - byteOff)
			}
			byteOff += chunkLen
			utf16Off += chunkLen
			continue
		}
		var j int
		for j < chunkLen {
			r, size := utf8.DecodeRune(chunk[j:])
			if r == utf8.RuneError && size <= 1 {
				return -1
			}
			l16 := 1
			if r >= 0x10000 {
				l16 = 2
			}
			utf16Off += l16
			j += size
			if byteOff+j >= c {
				return utf16Off
			}
		}
		byteOff += chunkLen
	}
	return -1
}

func isASCIIBytes(b []byte) bool {
	for i := range b {
		if b[i] >= 0x80 {
			return false
		}
	}
	return true
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

func (p *pieceTable) updateByIME(state textInputState, start, end int) int {
	if delta := state.DeleteEndInBytes - state.DeleteStartInBytes; delta > 0 {
		if start > state.DeleteStartInBytes {
			start -= delta
		}
		if end > state.DeleteStartInBytes {
			end -= delta
		}
		p.maybeAppendHistory(state.Text, state.DeleteStartInBytes, state.DeleteEndInBytes, start, end, 2, true)
	} else {
		p.maybeAppendHistory(state.Text, start, end, 0, 0, 1, true)
	}

	if state.DeleteEndInBytes-state.DeleteStartInBytes > 0 {
		p.doReplace("", state.DeleteStartInBytes, state.DeleteEndInBytes)
	}
	p.doReplace(state.Text, start, end)
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
