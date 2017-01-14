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

package resampling

import (
	"io"
	"math"

	"github.com/hajimehoshi/ebiten/audio"
)

func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	return math.Sin(x) / x
}

type Stream struct {
	source    audio.ReadSeekCloser
	size      int64
	from      int
	to        int
	pos       int64
	srcPos    int64
	srcCacheL []float64
	srcCacheR []float64
}

func NewStream(source audio.ReadSeekCloser, size int64, from, to int) *Stream {
	s := &Stream{
		source: source,
		size:   size,
		from:   from,
		to:     to,
	}
	s.srcCacheL = make([]float64, s.size/4)
	s.srcCacheR = make([]float64, s.size/4)
	return s
}

func (s *Stream) Size() int64 {
	return int64(float64(s.size) * float64(s.to) / float64(s.from))
}

func (s *Stream) src(i int) (float64, float64, error) {
	// Use int here since int64 is very slow on browsers.
	// TODO: Resampling is too heavy on browsers. How about using OfflineAudioContext?
	if i < 0 {
		return 0, 0, nil
	}
	if len(s.srcCacheL) <= i {
		return 0, 0, nil
	}
	pos := int(s.srcPos) / 4
	if pos <= i {
		buf := make([]uint8, 4096)
		n, err := s.source.Read(buf)
		if err != nil && err != io.EOF {
			return 0, 0, err
		}
		n = n / 4 * 4
		buf = buf[:n]
		for i := 0; i < len(buf)/4; i++ {
			srcL := float64(int16(buf[4*i])|(int16(buf[4*i+1])<<8)) / (1<<15 - 1)
			srcR := float64(int16(buf[4*i+2])|(int16(buf[4*i+3])<<8)) / (1<<15 - 1)
			s.srcCacheL[pos+i] = srcL
			s.srcCacheR[pos+i] = srcR
		}
		s.srcPos += int64(n)
	}
	return s.srcCacheL[i], s.srcCacheR[i], nil
}

func (s *Stream) at(t int64) (float64, float64, error) {
	const windowSize = 8
	tInSrc := float64(t) * float64(s.from) / float64(s.to)
	startN := int64(tInSrc) - windowSize
	if startN < 0 {
		startN = 0
	}
	if s.size/4 < startN {
		startN = s.size / 4
	}
	endN := int64(tInSrc) + windowSize + 1
	if s.size/4 < endN {
		endN = s.size / 4
	}
	l := 0.0
	r := 0.0
	for n := startN; n < endN; n++ {
		srcL, srcR, err := s.src(int(n))
		if err != nil {
			return 0, 0, err
		}
		w := 0.5 + 0.5*math.Cos(2*math.Pi*(tInSrc-float64(n))/(windowSize*2+1))
		s := sinc(math.Pi*(tInSrc-float64(n))) * w
		l += srcL * s
		r += srcR * s
	}
	if l < -1 {
		l = -1
	}
	if l > 1 {
		l = 1
	}
	if r < -1 {
		r = -1
	}
	if r > 1 {
		r = 1
	}
	return l, r, nil
}

func (s *Stream) Read(b []uint8) (int, error) {
	if s.pos == s.Size() {
		return 0, io.EOF
	}
	n := len(b) / 4 * 4
	if s.Size()-s.pos <= int64(n) {
		n = int(s.Size() - s.pos)
	}
	for i := 0; i < n/4; i++ {
		l, r, err := s.at(s.pos/4 + int64(i))
		if err != nil {
			return 0, err
		}
		l16 := int16(l * (1<<15 - 1))
		r16 := int16(r * (1<<15 - 1))
		b[4*i] = uint8(l16)
		b[4*i+1] = uint8(l16 >> 8)
		b[4*i+2] = uint8(r16)
		b[4*i+3] = uint8(r16 >> 8)
	}
	s.pos += int64(n)
	return n, nil
}

func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		s.pos = offset
	case io.SeekCurrent:
		s.pos += offset
	case io.SeekEnd:
		s.pos += s.Size() + offset
	}
	if s.pos < 0 {
		s.pos = 0
	}
	if s.Size() <= s.pos {
		s.pos = s.Size()
	}
	return s.pos, nil
}

func (s *Stream) Close() error {
	return s.source.Close()
}
