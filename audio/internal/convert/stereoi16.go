// Copyright 2017 The Ebiten Authors
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
	"fmt"
	"io"

	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
)

type Format int

const (
	FormatU8 Format = iota
	FormatS16
	FormatS24
)

type StereoI16ReadSeeker struct {
	source io.ReadSeeker
	mono   bool
	format Format
	eof    bool
	// buf holds the bytes read from the source but not converted yet. A short destination buffer can
	// leave a whole source frame pending.
	buf []byte
}

func NewStereoI16ReadSeeker(source io.ReadSeeker, mono bool, format Format) *StereoI16ReadSeeker {
	return &StereoI16ReadSeeker{
		source: source,
		mono:   mono,
		format: format,
	}
}

func (s *StereoI16ReadSeeker) Read(b []byte) (int, error) {
	frameSize := int(s.sourceFrameSize())
	if s.eof && len(s.buf) < frameSize {
		return 0, io.EOF
	}
	if len(b) == 0 {
		return 0, nil
	}
	// Buffer at least one sample to distinguish EOF from a short destination buffer.
	l := max(len(b)/4, 1) * frameSize

	// Read source bytes. Keep reading until one frame is available so that a source returning
	// less than one frame at a time doesn't make Read return (0, nil).
	var readErr error
	for len(s.buf) < l && !s.eof {
		origLen := len(s.buf)
		if cap(s.buf) < l {
			s.buf = append(s.buf, make([]byte, l-origLen)...)
		}

		n, err := s.source.Read(s.buf[origLen:l])
		s.buf = s.buf[:origLen+n]
		if err != nil && err != io.EOF {
			readErr = err
			break
		}
		if err == io.EOF {
			s.eof = true
		}
		if len(s.buf) >= frameSize || n == 0 {
			break
		}
	}

	if len(b) < 4 {
		if readErr != nil {
			return 0, readErr
		}
		if s.eof && len(s.buf) < frameSize {
			return 0, io.EOF
		}
		return 0, io.ErrShortBuffer
	}

	// Convert the whole frames and fill b. An incomplete frame is left for the next read.
	frames := len(s.buf) / frameSize
	if s.mono {
		switch s.format {
		case FormatU8:
			for i := range frames {
				v := int16(int(s.buf[i])*0x101 - (1 << 15))
				b[4*i] = byte(v)
				b[4*i+1] = byte(v >> 8)
				b[4*i+2] = byte(v)
				b[4*i+3] = byte(v >> 8)
			}
		case FormatS16:
			for i := range frames {
				b[4*i] = s.buf[2*i]
				b[4*i+1] = s.buf[2*i+1]
				b[4*i+2] = s.buf[2*i]
				b[4*i+3] = s.buf[2*i+1]
			}
		case FormatS24:
			for i := range frames {
				b[4*i] = s.buf[3*i+1]
				b[4*i+1] = s.buf[3*i+2]
				b[4*i+2] = s.buf[3*i+1]
				b[4*i+3] = s.buf[3*i+2]
			}
		}
	} else {
		switch s.format {
		case FormatU8:
			for i := range frames {
				v0 := int16(int(s.buf[2*i])*0x101 - (1 << 15))
				v1 := int16(int(s.buf[2*i+1])*0x101 - (1 << 15))
				b[4*i] = byte(v0)
				b[4*i+1] = byte(v0 >> 8)
				b[4*i+2] = byte(v1)
				b[4*i+3] = byte(v1 >> 8)
			}
		case FormatS16:
			copy(b[:4*frames], s.buf[:4*frames])
		case FormatS24:
			for i := range frames {
				b[4*i] = s.buf[6*i+1]
				b[4*i+1] = s.buf[6*i+2]
				b[4*i+2] = s.buf[6*i+4]
				b[4*i+3] = s.buf[6*i+5]
			}
		}
	}

	// Copy the remaining part for the next read.
	copy(s.buf, s.buf[frames*frameSize:])
	s.buf = s.buf[:len(s.buf)-frames*frameSize]

	n := frames * 4
	if readErr != nil {
		return n, readErr
	}
	if s.eof {
		return n, io.EOF
	}
	return n, nil
}

// sourceFrameSize returns the byte size of one frame of the source.
func (s *StereoI16ReadSeeker) sourceFrameSize() int64 {
	var size int64
	switch s.format {
	case FormatU8:
		size = 1
	case FormatS16:
		size = 2
	case FormatS24:
		size = 3
	}
	if !s.mono {
		size *= 2
	}
	return size
}

// presentedPosition returns the output byte position, or (0, false) if it overflows int64.
func (s *StereoI16ReadSeeker) presentedPosition(pos int64) (int64, bool) {
	return mathutil.MulDiv(pos, 4, s.sourceFrameSize())
}

func (s *StereoI16ReadSeeker) Seek(offset int64, whence int) (int64, error) {
	// Resolve the requested position before rounding the offset down to a frame boundary. An
	// unknown whence is left to the source.
	var base int64
	ok := true
	// alignedEnd is the source position just past the last whole frame. It is resolved only
	// for io.SeekEnd.
	var alignedEnd int64
	var current int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		cur, err := s.source.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		current = cur - int64(len(s.buf))
		base, ok = s.presentedPosition(current)
	case io.SeekEnd:
		// The source length is not necessarily a multiple of the frame size.
		// Resolve the offset from the last whole frame so that the source is
		// never seeked to the middle of a frame.
		cur, err := s.source.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		end, err := s.source.Seek(0, io.SeekEnd)
		if err != nil {
			return 0, err
		}
		// Undo the probe so that a rejected seek leaves the source where it was.
		if _, err := s.source.Seek(cur, io.SeekStart); err != nil {
			return 0, err
		}
		frameSize := s.sourceFrameSize()
		alignedEnd = end / frameSize * frameSize
		base, ok = s.presentedPosition(alignedEnd)
	}
	if !ok {
		return 0, fmt.Errorf("convert: position overflows int64")
	}
	// A query does not seek the source, so that the buffered bytes are kept.
	if whence == io.SeekCurrent && offset == 0 {
		return base, nil
	}
	if _, ok := mathutil.AddForSeek(base, offset); !ok {
		return 0, fmt.Errorf("convert: invalid seek position")
	}

	sourceOffset, ok := mathutil.Mul(mathutil.FloorDiv(offset, 4), s.sourceFrameSize())
	if !ok {
		return 0, fmt.Errorf("convert: source position overflows int64")
	}
	offset = sourceOffset

	if whence == io.SeekCurrent {
		if _, ok := mathutil.AddForSeek(current, offset); !ok {
			return 0, fmt.Errorf("convert: source position overflows int64")
		}
		offset -= int64(len(s.buf))
	}

	if whence == io.SeekEnd {
		offset, ok = mathutil.AddForSeek(alignedEnd, offset)
		if !ok {
			return 0, fmt.Errorf("convert: source position overflows int64")
		}
		whence = io.SeekStart
	}

	srcPos, err := s.source.Seek(offset, whence)
	if err != nil {
		return 0, err
	}

	pos, ok := s.presentedPosition(srcPos)
	if !ok {
		return 0, fmt.Errorf("convert: position overflows int64")
	}

	// Drop the buffered bytes only after the seek has succeeded, as the position this
	// wrapper presents is behind the source by their length.
	s.buf = s.buf[:0]
	s.eof = false

	return pos, nil
}
