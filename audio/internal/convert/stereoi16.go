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
	"io"
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
	// buf holds the bytes read from the source but not converted yet. After Read, buf is
	// shorter than one source frame.
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
	// A buffer shorter than one destination frame cannot receive any converted data.
	if len(b) < 4 {
		return 0, io.ErrShortBuffer
	}

	l := len(b) / 4 * frameSize

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

func (s *StereoI16ReadSeeker) Seek(offset int64, whence int) (int64, error) {
	offset = offset / 4 * 4
	if s.mono {
		offset /= 2
	}
	switch s.format {
	case FormatU8:
		offset /= 2
	case FormatS16:
	case FormatS24:
		offset *= 3
		offset /= 2
	}

	if whence == io.SeekCurrent {
		// The buffered bytes were read from the source but are not converted yet, so the
		// source is ahead of the position this wrapper presents.
		offset -= int64(len(s.buf))
	}

	if whence == io.SeekEnd {
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
		offset += end / frameSize * frameSize
		whence = io.SeekStart
	}

	pos, err := s.source.Seek(offset, whence)
	if err != nil {
		return 0, err
	}

	// Drop the buffered bytes only after the seek has succeeded, as the position this
	// wrapper presents is behind the source by their length.
	s.buf = s.buf[:0]
	s.eof = false

	// Convert the returned position from the source format's byte space to the
	// stereo-i16 byte space this wrapper presents.
	if s.mono {
		pos *= 2
	}
	switch s.format {
	case FormatU8:
		pos *= 2
	case FormatS16:
	case FormatS24:
		pos *= 2
		pos /= 3
	}
	return pos, nil
}
