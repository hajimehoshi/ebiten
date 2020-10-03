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

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type Stereo16 struct {
	source audio.ReadSeekCloser
	mono   bool
	eight  bool
}

func NewStereo16(source audio.ReadSeekCloser, mono, eight bool) *Stereo16 {
	return &Stereo16{
		source: source,
		mono:   mono,
		eight:  eight,
	}
}

func (s *Stereo16) Read(b []uint8) (int, error) {
	l := len(b)
	if s.mono {
		l /= 2
	}
	if s.eight {
		l /= 2
	}
	buf := make([]uint8, l)
	n, err := s.source.Read(buf)
	if err != nil && err != io.EOF {
		return 0, err
	}
	switch {
	case s.mono && s.eight:
		for i := 0; i < n; i++ {
			v := int16(int(buf[i])*0x101 - (1 << 15))
			b[4*i] = uint8(v)
			b[4*i+1] = uint8(v >> 8)
			b[4*i+2] = uint8(v)
			b[4*i+3] = uint8(v >> 8)
		}
	case s.mono && !s.eight:
		for i := 0; i < n/2; i++ {
			b[4*i] = buf[2*i]
			b[4*i+1] = buf[2*i+1]
			b[4*i+2] = buf[2*i]
			b[4*i+3] = buf[2*i+1]
		}
	case !s.mono && s.eight:
		for i := 0; i < n/2; i++ {
			v0 := int16(int(buf[2*i])*0x101 - (1 << 15))
			v1 := int16(int(buf[2*i+1])*0x101 - (1 << 15))
			b[4*i] = uint8(v0)
			b[4*i+1] = uint8(v0 >> 8)
			b[4*i+2] = uint8(v1)
			b[4*i+3] = uint8(v1 >> 8)
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

func (s *Stereo16) Close() error {
	return s.source.Close()
}
