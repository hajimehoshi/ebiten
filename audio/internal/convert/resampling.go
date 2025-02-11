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
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
)

var (
	// cosTable contains values of cosine applied to the range [0, π/2).
	// It must be initialised the first time it is referenced
	// in a function via its lazy load wrapper ensureCosTable().
	cosTable     []float64
	cosTableOnce sync.Once
)

func ensureCosTable() []float64 {
	cosTableOnce.Do(func() {
		cosTable = make([]float64, 65536)
		for i := range cosTable {
			cosTable[i] = math.Cos(float64(i) * math.Pi / 2 / float64(len(cosTable)))
		}
	})
	return cosTable
}

func fastCos01(x float64) float64 {
	if x < 0 {
		x = -x
	}

	cosTable := ensureCosTable()
	i := int(4 * float64(len(cosTable)) * x)
	if 4*len(cosTable) < i {
		i %= 4 * len(cosTable)
	}
	sign := 1
	switch {
	case i < len(cosTable):
	case i < len(cosTable)*2:
		i = len(cosTable)*2 - i
		sign = -1
	case i < len(cosTable)*3:
		i -= len(cosTable) * 2
		sign = -1
	default:
		i = len(cosTable)*4 - i
	}
	if i == len(cosTable) {
		return 0
	}
	return float64(sign) * cosTable[i]
}

func fastSin01(x float64) float64 {
	return fastCos01(x - 0.25)
}

func sinc01(x float64) float64 {
	if math.Abs(x) < 1e-8 {
		return 1
	}
	return fastSin01(x) / (x * 2 * math.Pi)
}

type Resampling struct {
	source          io.Reader
	size            int64
	from            int
	to              int
	bitDepthInBytes int
	pos             int64
	srcBlock        int64
	srcBufL         map[int64][]float64
	srcBufR         map[int64][]float64
	lruSrcBlocks    []int64
	eof             bool
	eofBufIndex     int64
}

func NewResampling(source io.Reader, size int64, from, to int, bitDepthInBytes int) *Resampling {
	r := &Resampling{
		source:          source,
		size:            size,
		from:            from,
		bitDepthInBytes: bitDepthInBytes,
		to:              to,
		srcBlock:        -1,
		srcBufL:         map[int64][]float64{},
		srcBufR:         map[int64][]float64{},
		eofBufIndex:     -1,
	}
	return r
}

func (r *Resampling) bytesPerSample() int {
	const channelNum = 2
	return r.bitDepthInBytes * channelNum
}

func (r *Resampling) Length() int64 {
	s := int64(float64(r.size) * float64(r.to) / float64(r.from))
	return s / int64(r.bytesPerSample()) * int64(r.bytesPerSample())
}

func (r *Resampling) src(i int64) (float64, float64, error) {
	const resamplingBufferSize = 4096

	if i < 0 {
		return 0, 0, nil
	}
	sizePerSample := int64(r.bytesPerSample())
	nextPos := int64(i) / resamplingBufferSize
	if _, ok := r.srcBufL[nextPos]; !ok {
		if r.srcBlock+1 != nextPos {
			seeker, ok := r.source.(io.Seeker)
			if !ok {
				return 0, 0, fmt.Errorf("convert: source must be io.Seeker")
			}
			if _, err := seeker.Seek(nextPos*resamplingBufferSize*sizePerSample, io.SeekStart); err != nil {
				return 0, 0, err
			}
		}
		buf := make([]byte, resamplingBufferSize*sizePerSample)
		var c int
		for c < len(buf) {
			n, err := r.source.Read(buf[c:])
			c += n
			if err != nil {
				if err == io.EOF {
					r.eofBufIndex = nextPos
					break
				}
				return 0, 0, err
			}
		}
		buf = buf[:c]
		sl := make([]float64, resamplingBufferSize)
		sr := make([]float64, resamplingBufferSize)
		switch r.bitDepthInBytes {
		case 2:
			for i := 0; i < len(buf)/int(sizePerSample); i++ {
				sl[i] = float64(int16(buf[4*i])|(int16(buf[4*i+1])<<8)) / (1<<15 - 1)
				sr[i] = float64(int16(buf[4*i+2])|(int16(buf[4*i+3])<<8)) / (1<<15 - 1)
			}
		case 4:
			for i := 0; i < len(buf)/int(sizePerSample); i++ {
				sl[i] = float64(math.Float32frombits(uint32(buf[8*i]) | uint32(buf[8*i+1])<<8 | uint32(buf[8*i+2])<<16 | uint32(buf[8*i+3])<<24))
				sr[i] = float64(math.Float32frombits(uint32(buf[8*i+4]) | uint32(buf[8*i+5])<<8 | uint32(buf[8*i+6])<<16 | uint32(buf[8*i+7])<<24))
			}
		default:
			panic("not reached")
		}
		r.srcBlock = nextPos
		r.srcBufL[r.srcBlock] = sl
		r.srcBufR[r.srcBlock] = sr
		// To keep srcBufL/R not too big, let's remove the least used buffers.
		if len(r.lruSrcBlocks) >= 4 {
			p := r.lruSrcBlocks[0]
			delete(r.srcBufL, p)
			delete(r.srcBufR, p)
			copy(r.lruSrcBlocks, r.lruSrcBlocks[1:])
			r.lruSrcBlocks = r.lruSrcBlocks[:len(r.lruSrcBlocks)-1]
		}
		r.lruSrcBlocks = append(r.lruSrcBlocks, r.srcBlock)
	} else {
		r.srcBlock = nextPos
		idx := -1
		for i, p := range r.lruSrcBlocks {
			if p == r.srcBlock {
				idx = i
				break
			}
		}
		if idx == -1 {
			panic("not reach")
		}
		r.lruSrcBlocks = append(r.lruSrcBlocks[:idx], r.lruSrcBlocks[idx+1:]...)
		r.lruSrcBlocks = append(r.lruSrcBlocks, r.srcBlock)
	}
	ii := i % resamplingBufferSize
	var err error
	if r.eofBufIndex == r.srcBlock && ii >= int64(len(r.srcBufL[r.srcBlock])-1) {
		err = io.EOF
	}
	if _, ok := r.source.(io.Seeker); ok {
		if r.size/sizePerSample <= i {
			err = io.EOF
		}
	}
	return r.srcBufL[r.srcBlock][ii], r.srcBufR[r.srcBlock][ii], err
}

func (r *Resampling) at(t int64) (float64, float64, error) {
	windowSize := 8.0
	tInSrc := float64(t) * float64(r.from) / float64(r.to)
	startN := int64(tInSrc - windowSize)
	if startN < 0 {
		startN = 0
	}
	endN := int64(tInSrc + windowSize)
	lv := 0.0
	rv := 0.0
	var eof bool
	for n := startN; n <= endN; n++ {
		srcL, srcR, err := r.src(n)
		if err != nil && err != io.EOF {
			return 0, 0, err
		}
		if err == io.EOF {
			eof = true
		}
		d := tInSrc - float64(n)
		w := 0.5 + 0.5*fastCos01(d/(windowSize*2+1))
		s := sinc01(d/2) * w
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
	if eof {
		return lv, rv, io.EOF
	}
	return lv, rv, nil
}

func (r *Resampling) Read(b []byte) (int, error) {
	if r.eof {
		return 0, io.EOF
	}

	size := r.bytesPerSample()
	n := len(b) / size * size
	switch r.bitDepthInBytes {
	case 2:
		for i := 0; i < n/size; i++ {
			ldata, rdata, err := r.at(r.pos/int64(size) + int64(i))
			if err != nil && err != io.EOF {
				return 0, err
			}
			if err == io.EOF {
				r.eof = true
			}
			l16 := int16(ldata * (1<<15 - 1))
			r16 := int16(rdata * (1<<15 - 1))
			b[4*i] = byte(l16)
			b[4*i+1] = byte(l16 >> 8)
			b[4*i+2] = byte(r16)
			b[4*i+3] = byte(r16 >> 8)
		}
	case 4:
		for i := 0; i < n/size; i++ {
			ldata, rdata, err := r.at(r.pos/int64(size) + int64(i))
			if err != nil && err != io.EOF {
				return 0, err
			}
			if err == io.EOF {
				r.eof = true
			}
			l32 := float32(ldata)
			r32 := float32(rdata)
			l32b := math.Float32bits(l32)
			r32b := math.Float32bits(r32)
			b[8*i] = byte(l32b)
			b[8*i+1] = byte(l32b >> 8)
			b[8*i+2] = byte(l32b >> 16)
			b[8*i+3] = byte(l32b >> 24)
			b[8*i+4] = byte(r32b)
			b[8*i+5] = byte(r32b >> 8)
			b[8*i+6] = byte(r32b >> 16)
			b[8*i+7] = byte(r32b >> 24)
		}
	default:
		panic("not reached")
	}
	r.pos += int64(n)
	if r.eof {
		return n, io.EOF
	}
	return n, nil
}

func (r *Resampling) Seek(offset int64, whence int) (int64, error) {
	if _, ok := r.source.(io.Seeker); !ok {
		return 0, fmt.Errorf("convert: source must be io.Seeker: %w", errors.ErrUnsupported)
	}

	r.eof = false
	switch whence {
	case io.SeekStart:
		r.pos = offset
	case io.SeekCurrent:
		r.pos += offset
	case io.SeekEnd:
		r.pos += r.Length() + offset
	}
	if r.pos < 0 {
		r.pos = 0
	}
	if r.Length() <= r.pos {
		r.pos = r.Length()
	}
	size := r.bytesPerSample()
	r.pos = r.pos / int64(size) * int64(size)
	return r.pos, nil
}
