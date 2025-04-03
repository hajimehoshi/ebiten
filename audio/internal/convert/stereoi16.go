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
}

func NewStereoI16ReadSeeker(source io.ReadSeeker, mono bool, format Format) *StereoI16ReadSeeker {
	return &StereoI16ReadSeeker{
		source: source,
		mono:   mono,
		format: format,
	}
}

func (s *StereoI16ReadSeeker) Read(b []byte) (int, error) {
	l := len(b) / 4 * 4
	if s.mono {
		l /= 2
	}
	switch s.format {
	case FormatU8:
		l /= 2
	case FormatS16:
	case FormatS24:
		l *= 3
		l /= 2
	}

	if cap(s.buf) < l {
		s.buf = make([]byte, l)
	}

	n, err := s.source.Read(s.buf[:l])
	if err != nil && err != io.EOF {
		return 0, err
	}
	if s.mono {
		switch s.format {
		case FormatU8:
			for i := 0; i < n; i++ {
				v := int16(int(s.buf[i])*0x101 - (1 << 15))
				b[4*i] = byte(v)
				b[4*i+1] = byte(v >> 8)
				b[4*i+2] = byte(v)
				b[4*i+3] = byte(v >> 8)
			}
		case FormatS16:
			for i := 0; i < n/2; i++ {
				b[4*i] = s.buf[2*i]
				b[4*i+1] = s.buf[2*i+1]
				b[4*i+2] = s.buf[2*i]
				b[4*i+3] = s.buf[2*i+1]
			}
		case FormatS24:
			for i := 0; i < n/3; i++ {
				b[4*i] = s.buf[3*i+1]
				b[4*i+1] = s.buf[3*i+2]
				b[4*i+2] = s.buf[3*i+1]
				b[4*i+3] = s.buf[3*i+2]
			}
		}
	} else {
		switch s.format {
		case FormatU8:
			for i := 0; i < n/2; i++ {
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
			for i := 0; i < n/6; i++ {
				b[4*i] = s.buf[6*i+1]
				b[4*i+1] = s.buf[6*i+2]
				b[4*i+2] = s.buf[6*i+4]
				b[4*i+3] = s.buf[6*i+5]
			}
		}
	}
	if s.mono {
		n *= 2
	}
	switch s.format {
	case FormatU8:
		n *= 2
	case FormatS16:
	case FormatS24:
		n *= 2
		n /= 3
	}
	return n, err
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
	return s.source.Seek(offset, whence)
}
