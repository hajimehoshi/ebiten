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
	if r.eof && len(r.i16Buf) == 0 {
		return 0, io.EOF
	}

	if i16LenToFill := len(buf) / 4 * 2; len(r.i16Buf) < i16LenToFill && !r.eof {
		origLen := len(r.i16Buf)
		if cap(r.i16Buf) < i16LenToFill {
			r.i16Buf = append(r.i16Buf, make([]byte, i16LenToFill-origLen)...)
		}

		// Read int16 bytes.
		n, err := r.r.Read(r.i16Buf[origLen:i16LenToFill])
		if err != nil && err != io.EOF {
			return 0, err
		}
		if err == io.EOF {
			r.eof = true
		}
		r.i16Buf = r.i16Buf[:origLen+n]
	}

	// Convert int16 bytes to float32 bytes and fill buf.
	samplesToFill := min(len(r.i16Buf)/2, len(buf)/4)
	for i := 0; i < samplesToFill; i++ {
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
		return 0, fmt.Errorf("float32: the source must be io.Seeker when seeking but not: %w", errors.ErrUnsupported)
	}
	r.i16Buf = r.i16Buf[:0]
	r.eof = false
	n, err := s.Seek(offset/4*2, whence)
	if err != nil {
		return 0, err
	}
	return n / 2 * 4, nil
}
