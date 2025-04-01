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

type float32BytesReaderVariable struct {
	r        io.Reader
	numBytes int
	signed   bool
	eof      bool
	iBuf     []byte
}

func NewFloat32BytesReadSeekerFromVariableIntBytesReader(r io.Reader, numBytes int) io.Reader {
	return &float32BytesReaderVariable{
		r:        r,
		numBytes: numBytes,
		// The spec has 1-8 bits as unsigned integers, 9 or more as signed
		signed: numBytes > 1,
	}
}

func NewFloat32BytesReadSeekerFromVariableIntBytesReadSeeker(r io.ReadSeeker, numBytes int) io.ReadSeeker {
	return &float32BytesReaderVariable{
		r:        r,
		numBytes: numBytes,
		// The spec has 1-8 bits as unsigned integers, 9 or more as signed
		signed: numBytes > 1,
	}
}

func (r *float32BytesReaderVariable) asFloat32(buf []byte) float32 {
	if r.signed {

		var rBig int32
		for s := 0; s < r.numBytes; s++ {
			b := buf[s]
			rBig |= int32(b) << (8 * (s + (4 - r.numBytes)))

		}
		rBig = rBig / 1 >> (8 * (4 - r.numBytes))
		v := float32(rBig) / float32((int32(1) << (r.numBytes*8 - 1)))
		return v
	}
	if r.numBytes == 1 {
		// This converts the byte into a int16, and then into a float32
		// This gives slightly different floats than converting from byte to float32 directly
		// but is kept this way as that is what the package used to do
		rBig := int16(int(buf[0])*0x101 - (1 << 15))
		v := float32(rBig) / float32((int32(1) << (2*8 - 1)))
		return v
	}

	var rBig int32
	for s := 0; s < r.numBytes; s++ {
		b := buf[s]
		rBig |= int32(b) << (8 * s)

	}
	rBig = rBig - 1<<(8*r.numBytes-1)
	v := float32(rBig) / float32((int32(1) << (r.numBytes*8 - 1)))
	return v
}

func (r *float32BytesReaderVariable) Read(buf []byte) (int, error) {
	if r.eof && len(r.iBuf) == 0 {
		return 0, io.EOF
	}

	if lenToFill := len(buf) / 4 * r.numBytes; len(r.iBuf) < lenToFill && !r.eof {
		origLen := len(r.iBuf)
		if cap(r.iBuf) < lenToFill {
			r.iBuf = append(r.iBuf, make([]byte, lenToFill-origLen)...)
		}

		// Read bytes.
		n, err := r.r.Read(r.iBuf[origLen:lenToFill])
		if err != nil && err != io.EOF {
			return 0, err
		}
		if err == io.EOF {
			r.eof = true
		}
		r.iBuf = r.iBuf[:origLen+n]
	}

	// Convert bytes to float32 bytes and fill buf.
	samplesToFill := min(len(r.iBuf)/r.numBytes, len(buf)/4)
	for i := 0; i < samplesToFill; i++ {
		v := r.asFloat32(r.iBuf[r.numBytes*i : r.numBytes*i+r.numBytes])

		vf32 := math.Float32bits(v)
		buf[4*i] = byte(vf32)
		buf[4*i+1] = byte(vf32 >> 8)
		buf[4*i+2] = byte(vf32 >> 16)
		buf[4*i+3] = byte(vf32 >> 24)
	}

	// Copy the remaining part for the next read.
	copy(r.iBuf, r.iBuf[samplesToFill*r.numBytes:])
	r.iBuf = r.iBuf[:len(r.iBuf)-samplesToFill*r.numBytes]

	n := samplesToFill * 4
	if r.eof {
		return n, io.EOF
	}
	return n, nil
}

func (r *float32BytesReaderVariable) Seek(offset int64, whence int) (int64, error) {
	s, ok := r.r.(io.Seeker)
	if !ok {
		return 0, fmt.Errorf("float32: the source must be io.Seeker when seeking but not: %w", errors.ErrUnsupported)
	}
	r.iBuf = r.iBuf[:0]
	r.eof = false
	n, err := s.Seek(offset/4*int64(r.numBytes), whence)
	if err != nil {
		return 0, err
	}
	return n / int64(r.numBytes) * 4, nil
}
