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
	"fmt"
	"io"
)

type StereoF32 struct {
	source io.ReadSeeker
	mono   bool
	eof    bool
	// buf holds the bytes read from the source but not converted yet. After Read, buf is
	// shorter than one source frame.
	buf []byte
}

func NewStereoF32(source io.ReadSeeker, mono bool) *StereoF32 {
	return &StereoF32{
		source: source,
		mono:   mono,
	}
}

func (s *StereoF32) Read(b []byte) (int, error) {
	frameSize := int(s.sourceFrameSize())
	if s.eof && len(s.buf) < frameSize {
		return 0, io.EOF
	}
	if len(b) == 0 {
		return 0, nil
	}
	// A buffer shorter than one destination frame cannot receive any converted data.
	if len(b) < 8 {
		return 0, io.ErrShortBuffer
	}

	l := len(b) / 8 * frameSize

	// Read source bytes. Keep reading until one frame is available so that a source returning
	// less than one frame at a time doesn't make Read return (0, nil).
	for len(s.buf) < l && !s.eof {
		origLen := len(s.buf)
		if cap(s.buf) < l {
			s.buf = append(s.buf, make([]byte, l-origLen)...)
		}

		n, err := s.source.Read(s.buf[origLen:l])
		if err != nil && err != io.EOF {
			return 0, err
		}
		if err == io.EOF {
			s.eof = true
		}
		s.buf = s.buf[:origLen+n]
		if len(s.buf) >= frameSize || n == 0 {
			break
		}
	}

	// Convert the whole frames and fill b. An incomplete frame is left for the next read.
	frames := len(s.buf) / frameSize
	if s.mono {
		for i := range frames {
			b[8*i] = s.buf[4*i]
			b[8*i+1] = s.buf[4*i+1]
			b[8*i+2] = s.buf[4*i+2]
			b[8*i+3] = s.buf[4*i+3]
			b[8*i+4] = s.buf[4*i]
			b[8*i+5] = s.buf[4*i+1]
			b[8*i+6] = s.buf[4*i+2]
			b[8*i+7] = s.buf[4*i+3]
		}
	} else {
		copy(b[:8*frames], s.buf[:8*frames])
	}

	// Copy the remaining part for the next read.
	copy(s.buf, s.buf[frames*frameSize:])
	s.buf = s.buf[:len(s.buf)-frames*frameSize]

	n := frames * 8
	if s.eof {
		return n, io.EOF
	}
	return n, nil
}

// sourceFrameSize returns the byte size of one frame of the source.
func (s *StereoF32) sourceFrameSize() int64 {
	if s.mono {
		return 4
	}
	return 8
}

// presentedPosition converts a position in the source's byte space to the stereo-f32 byte space
// this wrapper presents.
func (s *StereoF32) presentedPosition(pos int64) int64 {
	if s.mono {
		pos *= 2
	}
	return pos
}

func (s *StereoF32) Seek(offset int64, whence int) (int64, error) {
	// Resolve the requested position before rounding the offset toward the frame boundary
	// below, as the rounding truncates toward zero and would turn a small negative position
	// into 0. An unknown whence is left to the source.
	var pos int64
	// alignedEnd is the source position just past the last whole frame. It is resolved only
	// for io.SeekEnd.
	var alignedEnd int64
	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		// The position this wrapper presents is never negative, so only a negative offset can
		// resolve before the start.
		if offset < 0 {
			cur, err := s.source.Seek(0, io.SeekCurrent)
			if err != nil {
				return 0, err
			}
			// The buffered bytes were read from the source but are not converted yet, so the
			// source is ahead of the position this wrapper presents.
			pos = s.presentedPosition(cur-int64(len(s.buf))) + offset
		}
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
		pos = s.presentedPosition(alignedEnd) + offset
	}
	if pos < 0 {
		return 0, fmt.Errorf("convert: position must be >= 0 but was %d", pos)
	}

	offset = offset / 8 * 8
	if s.mono {
		offset /= 2
	}

	if whence == io.SeekCurrent {
		offset -= int64(len(s.buf))
	}

	if whence == io.SeekEnd {
		offset += alignedEnd
		whence = io.SeekStart
	}

	srcPos, err := s.source.Seek(offset, whence)
	if err != nil {
		return 0, err
	}

	// Drop the buffered bytes only after the seek has succeeded, as the position this
	// wrapper presents is behind the source by their length.
	s.buf = s.buf[:0]
	s.eof = false

	return s.presentedPosition(srcPos), nil
}
