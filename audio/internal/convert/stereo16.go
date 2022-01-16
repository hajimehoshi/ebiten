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

type Stereo16 struct {
	source io.ReadSeeker
	mono   bool
	eight  bool
	buf    []byte
}

func NewStereo16(source io.ReadSeeker, mono, eight bool) *Stereo16 {
	return &Stereo16{
		source: source,
		mono:   mono,
		eight:  eight,
	}
}

func (s *Stereo16) Read(b []byte) (int, error) {
	l := len(b)
	if s.mono {
		l /= 2
	}
	if s.eight {
		l /= 2
	}

	if cap(s.buf) < l {
		s.buf = make([]byte, l)
	}

	n, err := s.source.Read(s.buf[:l])
	if err != nil && err != io.EOF {
		return 0, err
	}
	switch {
	case s.mono && s.eight:
		for i := 0; i < n; i++ {
			v := int16(int(s.buf[i])*0x101 - (1 << 15))
			b[4*i] = byte(v)
			b[4*i+1] = byte(v >> 8)
			b[4*i+2] = byte(v)
			b[4*i+3] = byte(v >> 8)
		}
	case s.mono && !s.eight:
		for i := 0; i < n/2; i++ {
			b[4*i] = s.buf[2*i]
			b[4*i+1] = s.buf[2*i+1]
			b[4*i+2] = s.buf[2*i]
			b[4*i+3] = s.buf[2*i+1]
		}
	case !s.mono && s.eight:
		for i := 0; i < n/2; i++ {
			v0 := int16(int(s.buf[2*i])*0x101 - (1 << 15))
			v1 := int16(int(s.buf[2*i+1])*0x101 - (1 << 15))
			b[4*i] = byte(v0)
			b[4*i+1] = byte(v0 >> 8)
			b[4*i+2] = byte(v1)
			b[4*i+3] = byte(v1 >> 8)
		}
	}
	if s.mono {
		n *= 2
	}
	if s.eight {
		n *= 2
	}
	return n, err
}

func (s *Stereo16) Seek(offset int64, whence int) (int64, error) {
	if s.mono {
		offset /= 2
	}
	if s.eight {
		offset /= 2
	}
	return s.source.Seek(offset, whence)
}
