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

package vmprotocol

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// maxBulkByteCount is the maximum size of one bulk payload in bytes.
const maxBulkByteCount = (1 << 30) << (^uint(0) >> 62)

// bulkScratchByteCount is the size of the buffer that 32-bit elements are converted through. It is a
// multiple of 4.
const bulkScratchByteCount = 1 << 16

func newBulkWriter(w *bufio.Writer) bulkWriter {
	return bulkWriter{
		w:       w,
		scratch: make([]byte, bulkScratchByteCount),
	}
}

func newBulkReader(r *bufio.Reader) bulkReader {
	return bulkReader{
		r:       r,
		scratch: make([]byte, bulkScratchByteCount),
	}
}

// bulkWriter writes bulk payloads: each one is its element count as a uvarint followed by its elements,
// which are little-endian when they are wider than a byte.
type bulkWriter struct {
	w       *bufio.Writer
	scratch []byte
}

func (b *bulkWriter) writeCount(n int) error {
	var buf [binary.MaxVarintLen64]byte
	_, err := b.w.Write(buf[:binary.PutUvarint(buf[:], uint64(n))])
	return err
}

func (b *bulkWriter) writeBytes(p []byte) error {
	if err := b.writeCount(len(p)); err != nil {
		return err
	}
	_, err := b.w.Write(p)
	return err
}

func (b *bulkWriter) writeBytesList(ps [][]byte) error {
	if err := b.writeCount(len(ps)); err != nil {
		return err
	}
	for _, p := range ps {
		if err := b.writeBytes(p); err != nil {
			return err
		}
	}
	return nil
}

func (b *bulkWriter) writeUint32s(vs []uint32) error {
	if err := b.writeCount(len(vs)); err != nil {
		return err
	}
	for len(vs) > 0 {
		k := min(len(vs), len(b.scratch)/4)
		for i, v := range vs[:k] {
			binary.LittleEndian.PutUint32(b.scratch[4*i:], v)
		}
		if _, err := b.w.Write(b.scratch[:4*k]); err != nil {
			return err
		}
		vs = vs[k:]
	}
	return nil
}

func (b *bulkWriter) writeFloat32s(vs []float32) error {
	if err := b.writeCount(len(vs)); err != nil {
		return err
	}
	for len(vs) > 0 {
		k := min(len(vs), len(b.scratch)/4)
		for i, v := range vs[:k] {
			binary.LittleEndian.PutUint32(b.scratch[4*i:], math.Float32bits(v))
		}
		if _, err := b.w.Write(b.scratch[:4*k]); err != nil {
			return err
		}
		vs = vs[k:]
	}
	return nil
}

// bulkReader reads the payloads a bulkWriter writes. A payload is read into a slice of exactly its
// size, and an empty payload is read as nil.
type bulkReader struct {
	r       *bufio.Reader
	scratch []byte
}

// readCount reads an element count for elements of the given size in bytes.
func (b *bulkReader) readCount(elemByteCount int) (int, error) {
	n, err := binary.ReadUvarint(b.r)
	if err != nil {
		return 0, unexpectedEOF(err)
	}
	if n > maxBulkByteCount/uint64(elemByteCount) {
		return 0, fmt.Errorf("vmprotocol: a bulk payload of %d elements is too large", n)
	}
	return int(n), nil
}

func (b *bulkReader) readBytes() ([]byte, error) {
	n, err := b.readCount(1)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	p := make([]byte, n)
	if _, err := io.ReadFull(b.r, p); err != nil {
		return nil, unexpectedEOF(err)
	}
	return p, nil
}

func (b *bulkReader) readBytesList() ([][]byte, error) {
	// Each element takes at least one byte on the wire, so the count is bounded like a byte count.
	n, err := b.readCount(1)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	// The list grows as its elements arrive, so a count that the stream does not back allocates nothing.
	var ps [][]byte
	for range n {
		p, err := b.readBytes()
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

func (b *bulkReader) readUint32s() ([]uint32, error) {
	n, err := b.readCount(4)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	vs := make([]uint32, n)
	for rest := vs; len(rest) > 0; {
		k := min(len(rest), len(b.scratch)/4)
		if _, err := io.ReadFull(b.r, b.scratch[:4*k]); err != nil {
			return nil, unexpectedEOF(err)
		}
		for i := range rest[:k] {
			rest[i] = binary.LittleEndian.Uint32(b.scratch[4*i:])
		}
		rest = rest[k:]
	}
	return vs, nil
}

func (b *bulkReader) readFloat32s() ([]float32, error) {
	n, err := b.readCount(4)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	vs := make([]float32, n)
	for rest := vs; len(rest) > 0; {
		k := min(len(rest), len(b.scratch)/4)
		if _, err := io.ReadFull(b.r, b.scratch[:4*k]); err != nil {
			return nil, unexpectedEOF(err)
		}
		for i := range rest[:k] {
			rest[i] = math.Float32frombits(binary.LittleEndian.Uint32(b.scratch[4*i:]))
		}
		rest = rest[k:]
	}
	return vs, nil
}

// unexpectedEOF maps io.EOF to io.ErrUnexpectedEOF: a stream ending inside a message is not a clean
// end.
func unexpectedEOF(err error) error {
	if errors.Is(err, io.EOF) {
		return io.ErrUnexpectedEOF
	}
	return err
}
