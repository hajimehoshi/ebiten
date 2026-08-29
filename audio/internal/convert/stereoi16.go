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
	buf    []byte
	// rest is a partial source frame that was read from the source but is not
	// converted yet, so the source is ahead of the consumer by len(rest).
	rest []byte
}

func NewStereoI16ReadSeeker(source io.ReadSeeker, mono bool, format Format) *StereoI16ReadSeeker {
	return &StereoI16ReadSeeker{
		source: source,
		mono:   mono,
		format: format,
	}
}

const dstFrameSize = 4

func (s *StereoI16ReadSeeker) srcFrameSize() int {
	switch s.format {
	case FormatU8:
		if s.mono {
			return 1
		}
		return 2
	case FormatS16:
		if s.mono {
			return 2
		}
		return 4
	case FormatS24:
		if s.mono {
			return 3
		}
		return 6
	}
	panic("not reached")
}

func (s *StereoI16ReadSeeker) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	if len(b) < dstFrameSize {
		return 0, io.ErrShortBuffer
	}
	srcFrameSize := s.srcFrameSize()
	l := len(b) / dstFrameSize * srcFrameSize

	if cap(s.buf) < l {
		s.buf = make([]byte, l)
	}
	s.buf = s.buf[:l]

	n := copy(s.buf, s.rest)
	var err error
	for n < srcFrameSize {
		var dn int
		dn, err = s.source.Read(s.buf[n:])
		n += dn
		if err != nil || dn == 0 {
			break
		}
	}

	s.rest = append(s.rest[:0], s.buf[n/srcFrameSize*srcFrameSize:n]...)
	n = n / srcFrameSize * srcFrameSize

	if s.mono {
		switch s.format {
		case FormatU8:
			for i := range n {
				v := int16(int(s.buf[i])*0x101 - (1 << 15))
				b[4*i] = byte(v)
				b[4*i+1] = byte(v >> 8)
				b[4*i+2] = byte(v)
				b[4*i+3] = byte(v >> 8)
			}
		case FormatS16:
			for i := range n / 2 {
				b[4*i] = s.buf[2*i]
				b[4*i+1] = s.buf[2*i+1]
				b[4*i+2] = s.buf[2*i]
				b[4*i+3] = s.buf[2*i+1]
			}
		case FormatS24:
			for i := range n / 3 {
				b[4*i] = s.buf[3*i+1]
				b[4*i+1] = s.buf[3*i+2]
				b[4*i+2] = s.buf[3*i+1]
				b[4*i+3] = s.buf[3*i+2]
			}
		}
	} else {
		switch s.format {
		case FormatU8:
			for i := range n / 2 {
				v0 := int16(int(s.buf[2*i])*0x101 - (1 << 15))
				v1 := int16(int(s.buf[2*i+1])*0x101 - (1 << 15))
				b[4*i] = byte(v0)
				b[4*i+1] = byte(v0 >> 8)
				b[4*i+2] = byte(v1)
				b[4*i+3] = byte(v1 >> 8)
			}
		case FormatS16:
			copy(b[:n], s.buf[:n])
		case FormatS24:
			for i := range n / 6 {
				b[4*i] = s.buf[6*i+1]
				b[4*i+1] = s.buf[6*i+2]
				b[4*i+2] = s.buf[6*i+4]
				b[4*i+3] = s.buf[6*i+5]
			}
		}
	}

	return n / srcFrameSize * dstFrameSize, err
}

func (s *StereoI16ReadSeeker) Seek(offset int64, whence int) (int64, error) {
	srcFrameSize := int64(s.srcFrameSize())
	o := offset / dstFrameSize * srcFrameSize
	if whence == io.SeekCurrent {
		o -= int64(len(s.rest))
	}
	pos, err := s.source.Seek(o, whence)
	if err != nil {
		return 0, err
	}
	s.rest = s.rest[:0]

	// Convert the returned position from the source format's byte space to the
	// stereo-i16 byte space this wrapper presents.
	return pos / srcFrameSize * dstFrameSize, nil
}
