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
	"io"
)

type StereoF32 struct {
	source io.ReadSeeker
	mono   bool
	buf    []byte
}

func NewStereoF32(source io.ReadSeeker, mono bool) *StereoF32 {
	return &StereoF32{
		source: source,
		mono:   mono,
	}
}

func (s *StereoF32) Read(b []byte) (int, error) {
	l := len(b) / 8 * 8
	if s.mono {
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
		for i := 0; i < n/4; i++ {
			b[8*i] = s.buf[4*i]
			b[8*i+1] = s.buf[4*i+1]
			b[8*i+2] = s.buf[4*i+2]
			b[8*i+3] = s.buf[4*i+3]
			b[8*i+4] = s.buf[4*i]
			b[8*i+5] = s.buf[4*i+1]
			b[8*i+6] = s.buf[4*i+2]
			b[8*i+7] = s.buf[4*i+3]
		}
		n *= 2
	} else {
		copy(b[:n], s.buf[:n])
	}
	return n, err
}

func (s *StereoF32) Seek(offset int64, whence int) (int64, error) {
	offset = offset / 8 * 8
	if s.mono {
		offset /= 2
	}
	return s.source.Seek(offset, whence)
}
