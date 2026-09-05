// Copyright 2024 The Ebitengine Authors
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

package convert

import (
	"errors"
	"fmt"
	"io"
	"math"
)

func NewFloat32BytesReaderFromInt16BytesReader(r io.Reader) io.Reader {
	return &float32BytesReader{r: r}
}

func NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(r io.ReadSeeker) io.ReadSeeker {
	return &float32BytesReader{r: r}
}

type float32BytesReader struct {
	r      io.Reader
	eof    bool
	i16Buf []byte
}

func (r *float32BytesReader) Read(buf []byte) (int, error) {
	if r.eof && len(r.i16Buf) < 2 {
		return 0, io.EOF
	}
	if len(buf) == 0 {
		return 0, nil
	}
	if len(buf) < 4 {
		return 0, io.ErrShortBuffer
	}

	// Read int16 bytes. Keep reading until one sample is available so that a source returning
	// less than one sample at a time doesn't make Read return (0, nil).
	i16LenToFill := len(buf) / 4 * 2
	for len(r.i16Buf) < i16LenToFill && !r.eof {
		origLen := len(r.i16Buf)
		if cap(r.i16Buf) < i16LenToFill {
			r.i16Buf = append(r.i16Buf, make([]byte, i16LenToFill-origLen)...)
		}

		n, err := r.r.Read(r.i16Buf[origLen:i16LenToFill])
		if err != nil && err != io.EOF {
			return 0, err
		}
		if err == io.EOF {
			r.eof = true
		}
		r.i16Buf = r.i16Buf[:origLen+n]
		if len(r.i16Buf) >= 2 || n == 0 {
			break
		}
	}

	// Convert int16 bytes to float32 bytes and fill buf.
	samplesToFill := min(len(r.i16Buf)/2, len(buf)/4)
	for i := range samplesToFill {
		vi16l := r.i16Buf[2*i]
		vi16h := r.i16Buf[2*i+1]
		v := float32(int16(vi16l)|int16(vi16h)<<8) / (1 << 15)
		vf32 := math.Float32bits(v)
		buf[4*i] = byte(vf32)
		buf[4*i+1] = byte(vf32 >> 8)
		buf[4*i+2] = byte(vf32 >> 16)
		buf[4*i+3] = byte(vf32 >> 24)
	}

	// Copy the remaining part for the next read.
	copy(r.i16Buf, r.i16Buf[samplesToFill*2:])
	r.i16Buf = r.i16Buf[:len(r.i16Buf)-samplesToFill*2]

	n := samplesToFill * 4
	if r.eof {
		return n, io.EOF
	}
	return n, nil
}

func (r *float32BytesReader) Seek(offset int64, whence int) (int64, error) {
	s, ok := r.r.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("float32: the source must be io.Seeker to seek: %w", errors.ErrUnsupported)
	}
	// Resolve the requested position before rounding the offset toward the sample boundary
	// below, as the rounding truncates toward zero and would turn a small negative position
	// into 0.
	var pos int64
	// alignedEnd is the source position just past the last whole sample. It is resolved only
	// for io.SeekEnd.
	var alignedEnd int64
	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		// The position this reader presents is never negative, so only a negative offset can
		// resolve before the start.
		if offset < 0 {
			cur, err := s.Seek(0, io.SeekCurrent)
			if err != nil {
				return 0, err
			}
			// The source is ahead of the position this reader presents by the buffered bytes.
			pos = (cur-int64(len(r.i16Buf)))/2*4 + offset
		}
	case io.SeekEnd:
		// The source length is not necessarily a multiple of the sample size. Resolve the offset
		// from the last whole sample so that the source is never seeked to the middle of a sample.
		cur, err := s.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		end, err := s.Seek(0, io.SeekEnd)
		if err != nil {
			return 0, err
		}
		// Undo the probe so that a rejected seek leaves the source where it was.
		if _, err := s.Seek(cur, io.SeekStart); err != nil {
			return 0, err
		}
		alignedEnd = end / 2 * 2
		pos = alignedEnd/2*4 + offset
	default:
		return 0, fmt.Errorf("convert: whence must be io.SeekStart, io.SeekCurrent, or io.SeekEnd but was %d", whence)
	}
	if pos < 0 {
		return 0, fmt.Errorf("convert: position must be >= 0 but was %d", pos)
	}

	offset = offset / 4 * 2

	switch whence {
	case io.SeekCurrent:
		offset -= int64(len(r.i16Buf))
	case io.SeekEnd:
		offset += alignedEnd
		whence = io.SeekStart
	}

	n, err := s.Seek(offset, whence)
	if err != nil {
		return 0, err
	}

	// Drop the buffered bytes only after the seek has succeeded, as the position this
	// reader presents is behind the source by their length.
	r.i16Buf = r.i16Buf[:0]
	r.eof = false
	return n / 2 * 4, nil
}
