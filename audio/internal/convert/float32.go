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

	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
)

func NewFloat32BytesReaderFromInt16BytesReader(r io.Reader) io.Reader {
	return &float32BytesReader{r: r}
}

func NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(r io.ReadSeeker) io.ReadSeeker {
	return &float32BytesReader{r: r}
}

const (
	channelCount = 2
	// int16SampleSize is the size in bytes of one sample of the source.
	int16SampleSize = 2 * channelCount
	// float32SampleSize is the size in bytes of one sample of the output.
	float32SampleSize = 4 * channelCount
)

// float32BytesReader converts a signed 16bit integer stereo stream into a 32bit float stereo stream.
type float32BytesReader struct {
	r      io.Reader
	eof    bool
	i16Buf []byte
}

func (r *float32BytesReader) Read(buf []byte) (int, error) {
	if r.eof && len(r.i16Buf) < int16SampleSize {
		return 0, io.EOF
	}
	if len(buf) == 0 {
		return 0, nil
	}
	// Read int16 bytes. Keep reading until one sample is available so that a source returning
	// less than one sample at a time doesn't make Read return (0, nil).
	// Buffer at least one sample to distinguish EOF from a short destination buffer.
	i16LenToFill := max(len(buf)/float32SampleSize, 1) * int16SampleSize
	var readErr error
	for len(r.i16Buf) < i16LenToFill && !r.eof {
		origLen := len(r.i16Buf)
		if cap(r.i16Buf) < i16LenToFill {
			r.i16Buf = append(r.i16Buf, make([]byte, i16LenToFill-origLen)...)
		}

		n, err := r.r.Read(r.i16Buf[origLen:i16LenToFill])
		r.i16Buf = r.i16Buf[:origLen+n]
		if err != nil && err != io.EOF {
			readErr = err
			break
		}
		if err == io.EOF {
			r.eof = true
		}
		if len(r.i16Buf) >= int16SampleSize || n == 0 {
			break
		}
	}

	if len(buf) < float32SampleSize {
		if readErr != nil {
			return 0, readErr
		}
		if r.eof && len(r.i16Buf) < int16SampleSize {
			return 0, io.EOF
		}
		return 0, io.ErrShortBuffer
	}

	// Convert the whole samples and fill buf. An incomplete sample is left for the next read.
	samples := min(len(r.i16Buf)/int16SampleSize, len(buf)/float32SampleSize)
	for i := range samples * channelCount {
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
	copy(r.i16Buf, r.i16Buf[samples*int16SampleSize:])
	r.i16Buf = r.i16Buf[:len(r.i16Buf)-samples*int16SampleSize]

	n := samples * float32SampleSize
	if readErr != nil {
		return n, readErr
	}
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
	// Resolve the requested position before rounding the offset down to a sample boundary.
	var base int64
	// alignedEnd is the source position just past the last whole sample. It is resolved only
	// for io.SeekEnd.
	var alignedEnd int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		cur, err := s.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		samples := (cur - int64(len(r.i16Buf))) / int16SampleSize
		base, ok = mathutil.Mul(samples, float32SampleSize)
		if !ok {
			return 0, fmt.Errorf("convert: position overflows int64")
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
		alignedEnd = end / int16SampleSize * int16SampleSize
		base, ok = mathutil.Mul(alignedEnd/int16SampleSize, float32SampleSize)
		if !ok {
			return 0, fmt.Errorf("convert: position overflows int64")
		}
	default:
		return 0, fmt.Errorf("convert: whence must be io.SeekStart, io.SeekCurrent, or io.SeekEnd but was %d", whence)
	}
	if _, ok := mathutil.AddForSeek(base, offset); !ok {
		return 0, fmt.Errorf("convert: invalid seek position")
	}

	offset = mathutil.FloorDiv(offset, float32SampleSize) * int16SampleSize

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
	return n / int16SampleSize * float32SampleSize, nil
}
