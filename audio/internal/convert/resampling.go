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
	"math"

	"github.com/hajimehoshi/ebiten/audio"
)

func sinc(x float64) float64 {
	if x == 0 {
		return 1
	}
	return math.Sin(x) / x
}

type Resampling struct {
	source    audio.ReadSeekCloser
	size      int64
	from      int
	to        int
	pos       int64
	srcPos    int64
	srcCacheL []float64
	srcCacheR []float64
}

func NewResampling(source audio.ReadSeekCloser, size int64, from, to int) *Resampling {
	r := &Resampling{
		source: source,
		size:   size,
		from:   from,
		to:     to,
	}
	r.srcCacheL = make([]float64, r.size/4)
	r.srcCacheR = make([]float64, r.size/4)
	return r
}

func (r *Resampling) Size() int64 {
	s := int64(float64(r.size) * float64(r.to) / float64(r.from))
	return s / 4 * 4
}

func (r *Resampling) src(i int) (float64, float64, error) {
	// Use int here since int64 is very slow on browsers.
	// TODO: Resampling is too heavy on browsers. How about using OfflineAudioContext?
	if i < 0 {
		return 0, 0, nil
	}
	if len(r.srcCacheL) <= i {
		return 0, 0, nil
	}
	pos := int(r.srcPos) / 4
	if pos <= i {
		buf := make([]uint8, 4096)
		n, err := r.source.Read(buf)
		if err != nil && err != io.EOF {
			return 0, 0, err
		}
		n = n / 4 * 4
		buf = buf[:n]
		for i := 0; i < len(buf)/4; i++ {
			srcL := float64(int16(buf[4*i])|(int16(buf[4*i+1])<<8)) / (1<<15 - 1)
			srcR := float64(int16(buf[4*i+2])|(int16(buf[4*i+3])<<8)) / (1<<15 - 1)
			r.srcCacheL[pos+i] = srcL
			r.srcCacheR[pos+i] = srcR
		}
		r.srcPos += int64(n)
	}
	return r.srcCacheL[i], r.srcCacheR[i], nil
}

func (r *Resampling) at(t int64) (float64, float64, error) {
	const windowSize = 8
	tInSrc := float64(t) * float64(r.from) / float64(r.to)
	startN := int64(tInSrc) - windowSize
	if startN < 0 {
		startN = 0
	}
	if r.size/4 < startN {
		startN = r.size / 4
	}
	endN := int64(tInSrc) + windowSize + 1
	if r.size/4 < endN {
		endN = r.size / 4
	}
	lv := 0.0
	rv := 0.0
	for n := startN; n < endN; n++ {
		srcL, srcR, err := r.src(int(n))
		if err != nil {
			return 0, 0, err
		}
		w := 0.5 + 0.5*math.Cos(2*math.Pi*(tInSrc-float64(n))/(windowSize*2+1))
		s := sinc(math.Pi*(tInSrc-float64(n))) * w
		lv += srcL * s
		rv += srcR * s
	}
	if lv < -1 {
		lv = -1
	}
	if lv > 1 {
		lv = 1
	}
	if rv < -1 {
		rv = -1
	}
	if rv > 1 {
		rv = 1
	}
	return lv, rv, nil
}

func (r *Resampling) Read(b []uint8) (int, error) {
	if r.pos == r.Size() {
		return 0, io.EOF
	}
	n := len(b) / 4 * 4
	if r.Size()-r.pos <= int64(n) {
		n = int(r.Size() - r.pos)
	}
	for i := 0; i < n/4; i++ {
		l, r, err := r.at(r.pos/4 + int64(i))
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
	r.pos += int64(n)
	return n, nil
}

func (r *Resampling) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		r.pos = offset
	case io.SeekCurrent:
		r.pos += offset
	case io.SeekEnd:
		r.pos += r.Size() + offset
	}
	if r.pos < 0 {
		r.pos = 0
	}
	if r.Size() <= r.pos {
		r.pos = r.Size()
	}
	return r.pos, nil
}

func (r *Resampling) Close() error {
	return r.source.Close()
}
