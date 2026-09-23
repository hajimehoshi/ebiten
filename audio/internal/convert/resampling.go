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

	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
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
	source io.Reader

	// declaredSrcLength is the length in bytes given by the caller. -1 indicates the length is unknown.
	declaredSrcLength int64

	// derivedSrcLength is the length in bytes determined when the source reaches its end.
	// derivedSrcLength is used only when declaredSrcLength is -1. -1 indicates the length is not determined yet.
	derivedSrcLength int64

	// srcReadEnd is the offset in bytes just past the last byte the source has delivered so far.
	srcReadEnd int64

	from            int
	to              int
	bitDepthInBytes int
	pos             int64
	srcBlock        int64
	// lastReadSrcBlock is the last block read completely or through EOF.
	// It is -1 before the first block is read. Its value is ignored when
	// lastReadSrcBlockValid is false.
	lastReadSrcBlock      int64
	lastReadSrcBlockValid bool
	srcBufL               map[int64][]float64
	srcBufR               map[int64][]float64
	lruSrcBlocks          []int64
	eof                   bool

	// partialSrcBuf is the beginning of the block partialSrcBlock read before a source error.
	// The source position is right after it. partialSrcBuf is nil when there is no such block.
	partialSrcBlock int64
	partialSrcBuf   []byte
}

// NewResampling returns a stream that converts the sample rate of source.
// length is the length of source in bytes. A negative length indicates the length is unknown.
func NewResampling(source io.Reader, length int64, from, to int, bitDepthInBytes int) *Resampling {
	if length < 0 {
		length = -1
	}
	r := &Resampling{
		source:                source,
		declaredSrcLength:     length,
		derivedSrcLength:      -1,
		from:                  from,
		bitDepthInBytes:       bitDepthInBytes,
		to:                    to,
		srcBlock:              -1,
		lastReadSrcBlock:      -1,
		lastReadSrcBlockValid: true,
		srcBufL:               map[int64][]float64{},
		srcBufR:               map[int64][]float64{},
	}
	return r
}

func (r *Resampling) bytesPerSample() int {
	const channelNum = 2
	return r.bitDepthInBytes * channelNum
}

// Length returns the length of the resampled stream in bytes, or -1 when the length is unknown.
func (r *Resampling) Length() int64 {
	srcLength := r.srcLength()
	if srcLength < 0 {
		return -1
	}
	return r.resampledLength(srcLength)
}

// srcLength returns the length of the source stream in bytes, or -1 when the length is unknown.
func (r *Resampling) srcLength() int64 {
	if r.declaredSrcLength >= 0 {
		return r.declaredSrcLength
	}
	return r.derivedSrcLength
}

// resampledLength returns the length of the resampled stream in bytes for a source stream of the given length.
func (r *Resampling) resampledLength(srcLength int64) int64 {
	s, ok := mathutil.MulDiv(srcLength, int64(r.to), int64(r.from))
	if !ok {
		panic("convert: resampled length is out of range")
	}
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
		blockStart := nextPos * resamplingBufferSize * sizePerSample
		buf := make([]byte, resamplingBufferSize*sizePerSample)
		var c int
		if r.partialSrcBuf != nil && r.partialSrcBlock == nextPos {
			c = copy(buf, r.partialSrcBuf)
		} else if !r.lastReadSrcBlockValid || r.lastReadSrcBlock+1 != nextPos {
			seeker, ok := r.source.(io.Seeker)
			if !ok {
				return 0, 0, fmt.Errorf("convert: source must be io.Seeker")
			}
			if _, err := seeker.Seek(blockStart, io.SeekStart); err != nil {
				return 0, 0, err
			}
		}
		r.partialSrcBuf = nil
		var eof bool
		for c < len(buf) {
			n, err := r.source.Read(buf[c:])
			c += n
			if err != nil {
				if err == io.EOF {
					eof = true
					// Determine the source length the first time the source reaches its end.
					// A block beyond the bytes delivered so far is reached only by seeking, and its end
					// without any byte tells only that the source ends at or before the block.
					if r.declaredSrcLength < 0 && r.derivedSrcLength < 0 && (c > 0 || blockStart <= r.srcReadEnd) {
						r.derivedSrcLength = blockStart + int64(c)
					}
					r.lastReadSrcBlock = nextPos
					r.lastReadSrcBlockValid = true
					break
				}
				r.lastReadSrcBlockValid = false
				// Keep the bytes read so far, as a source that is not an io.Seeker cannot deliver them again.
				if c > 0 {
					r.partialSrcBlock = nextPos
					r.partialSrcBuf = buf[:c]
					r.srcReadEnd = max(r.srcReadEnd, blockStart+int64(c))
				}
				return 0, 0, err
			}
			// A source making no progress must not spin here.
			if n == 0 {
				r.lastReadSrcBlockValid = false
				break
			}
		}
		if c == len(buf) {
			r.lastReadSrcBlock = nextPos
			r.lastReadSrcBlockValid = true
		}
		if c > 0 {
			r.srcReadEnd = max(r.srcReadEnd, blockStart+int64(c))
		}
		// A block whose end position is not determined must not be cached, so that the block is read
		// again once the preceding data has been delivered.
		if eof && c == 0 && r.srcLength() < 0 && blockStart > r.srcReadEnd {
			return 0, 0, io.EOF
		}
		buf = buf[:c]
		sl := make([]float64, resamplingBufferSize)
		sr := make([]float64, resamplingBufferSize)
		switch r.bitDepthInBytes {
		case 2:
			for i := range len(buf) / int(sizePerSample) {
				sl[i] = float64(int16(buf[4*i])|(int16(buf[4*i+1])<<8)) / (1<<15 - 1)
				sr[i] = float64(int16(buf[4*i+2])|(int16(buf[4*i+3])<<8)) / (1<<15 - 1)
			}
		case 4:
			for i := range len(buf) / int(sizePerSample) {
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
			panic("not reached")
		}
		r.lruSrcBlocks = append(r.lruSrcBlocks[:idx], r.lruSrcBlocks[idx+1:]...)
		r.lruSrcBlocks = append(r.lruSrcBlocks, r.srcBlock)
	}
	ii := i % resamplingBufferSize
	var err error
	// An unknown length is determined once the source reaches its end.
	if srcLength := r.srcLength(); srcLength >= 0 && srcLength/sizePerSample <= i {
		err = io.EOF
	}
	return r.srcBufL[r.srcBlock][ii], r.srcBufR[r.srcBlock][ii], err
}

func (r *Resampling) at(t int64) (float64, float64, error) {
	windowSize := 8.0
	tInSrc := float64(t) * float64(r.from) / float64(r.to)
	startN := max(int64(tInSrc-windowSize), 0)
	endN := int64(tInSrc + windowSize)
	var lv, rv float64
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
	if r.eof || (r.srcLength() >= 0 && r.pos >= r.Length()) {
		return 0, io.EOF
	}
	if len(b) == 0 {
		return 0, nil
	}

	size := r.bytesPerSample()
	if len(b) < size {
		if r.srcLength() < 0 {
			// Resolve EOF through the source cache without advancing the output position.
			_, _, err := r.at(r.pos / int64(size))
			if err != nil && err != io.EOF {
				return 0, err
			}
			if (err == io.EOF && r.srcLength() < 0) || (r.srcLength() >= 0 && r.pos >= r.Length()) {
				r.eof = true
				return 0, io.EOF
			}
		}
		return 0, io.ErrShortBuffer
	}

	n := len(b) / size * size
	switch r.bitDepthInBytes {
	case 2:
		for i := range n / size {
			ldata, rdata, err := r.at(r.pos/int64(size) + int64(i))
			if err != nil && err != io.EOF {
				r.pos += int64(size * i)
				return size * i, err
			}
			// EOF from the at method indicates that the source reaches the end, and doesn't indicate the resampled data ends.
			// The length check below is the terminator of the resampled data, except when the length is still unknown:
			// then the position is beyond the source's end after seeking, and the resampled data ends here.
			if err == io.EOF && r.srcLength() < 0 {
				n = size * i
				r.eof = true
				break
			}
			l16 := int16(ldata * (1<<15 - 1))
			r16 := int16(rdata * (1<<15 - 1))
			b[4*i] = byte(l16)
			b[4*i+1] = byte(l16 >> 8)
			b[4*i+2] = byte(r16)
			b[4*i+3] = byte(r16 >> 8)
			// If the length is known, check whether the resampled data ends (#3352).
			if srcLength := r.srcLength(); srcLength >= 0 && r.pos+int64(size*i) >= r.resampledLength(srcLength) {
				n = size * i
				r.eof = true
				break
			}
		}
	case 4:
		for i := range n / size {
			ldata, rdata, err := r.at(r.pos/int64(size) + int64(i))
			if err != nil && err != io.EOF {
				r.pos += int64(size * i)
				return size * i, err
			}
			if err == io.EOF && r.srcLength() < 0 {
				n = size * i
				r.eof = true
				break
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
			if srcLength := r.srcLength(); srcLength >= 0 && r.pos+int64(size*i) >= r.resampledLength(srcLength) {
				n = size * i
				r.eof = true
				break
			}
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

	var base int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = r.pos
	case io.SeekEnd:
		if r.srcLength() < 0 {
			return 0, fmt.Errorf("convert: seeking from the end is not possible when the length is unknown: %w", errors.ErrUnsupported)
		}
		base = r.Length()
	default:
		return 0, fmt.Errorf("convert: whence must be io.SeekStart, io.SeekCurrent, or io.SeekEnd but was %d", whence)
	}
	pos, ok := mathutil.AddForSeek(base, offset)
	if !ok {
		return 0, fmt.Errorf("convert: invalid seek position")
	}
	r.eof = false
	r.pos = pos
	size := r.bytesPerSample()
	r.pos = r.pos / int64(size) * int64(size)
	return r.pos, nil
}
