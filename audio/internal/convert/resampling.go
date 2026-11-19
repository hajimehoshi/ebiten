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
	"math/bits"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
)

const (
	// resamplingHalfTaps is the number of taps on each side of the source frame at or before an output frame's position
	// when the cutoff is the source Nyquist frequency.
	resamplingHalfTaps = 8

	// resamplingTaps is the number of source frames weighted for an output frame
	// when the cutoff is the source Nyquist frequency.
	resamplingTaps = 2*resamplingHalfTaps + 1

	// minResamplingCutoff is the lowest cutoff relative to the source Nyquist frequency.
	// It is below the ratio of common sample rates, such as 8000 Hz to 192000 Hz,
	// and bounds the number of taps so that the taps of an output frame lie in at most two source blocks.
	minResamplingCutoff = 1.0 / 32

	// maxResamplingPhases is the maximum number of phases in a weight table.
	maxResamplingPhases = 1024

	// resamplingBlockFrames is the number of frames in a source block.
	resamplingBlockFrames = 4096
)

// resamplingKernel is a Hann-windowed sinc function.
type resamplingKernel struct {
	// cutoff is the cutoff frequency relative to the source Nyquist frequency, in (0, 1].
	cutoff float64

	// halfTaps is the number of taps on each side of the source frame at or before an output frame's position.
	// The window spans the 2*halfTaps+1 taps.
	halfTaps int
}

// newResamplingKernel returns the kernel to resample from the sample rate from to the sample rate to.
func newResamplingKernel(from, to uint64) resamplingKernel {
	// When downsampling, the cutoff is the destination Nyquist frequency, so that the frequencies above it do not alias.
	cutoff := max(min(float64(to)/float64(from), 1), minResamplingCutoff)
	return resamplingKernel{
		cutoff: cutoff,
		// The window spans at least resamplingTaps/cutoff source frames,
		// so that it covers as many zero crossings of the sinc function as at the source Nyquist frequency.
		halfTaps: int(math.Ceil((resamplingTaps/cutoff - 1) / 2)),
	}
}

func (k resamplingKernel) taps() int {
	return 2*k.halfTaps + 1
}

// weight returns the weight of a source frame d frames before an output frame's position.
func (k resamplingKernel) weight(d float64) float64 {
	x := k.cutoff * d
	if x == 0 {
		return k.cutoff
	}
	// The sinc function is zero at the other integers.
	if x == math.Trunc(x) {
		return 0
	}
	// A Hann window that spans the taps.
	w := 0.5 + 0.5*math.Cos(2*math.Pi*d/float64(k.taps()))
	return k.cutoff * math.Sin(math.Pi*x) / (math.Pi * x) * w
}

type resamplingWeightTableKey struct {
	kernel     resamplingKernel
	phaseCount int
}

var (
	resamplingWeightTablesMu sync.Mutex

	// resamplingWeightTables maps a kernel and a phase count to their weight table.
	resamplingWeightTables = map[resamplingWeightTableKey][]float64{}
)

// resamplingWeightTable returns the tap weights of kernel for phaseCount+1 phases evenly spaced from 0 to 1 frame.
// The weight of the tap j at the phase p is at the index p*kernel.taps()+j.
// The weights of each phase sum to 1.
func resamplingWeightTable(kernel resamplingKernel, phaseCount int) []float64 {
	resamplingWeightTablesMu.Lock()
	defer resamplingWeightTablesMu.Unlock()

	key := resamplingWeightTableKey{
		kernel:     kernel,
		phaseCount: phaseCount,
	}
	if table, ok := resamplingWeightTables[key]; ok {
		return table
	}
	taps := kernel.taps()
	table := make([]float64, (phaseCount+1)*taps)
	for p := range phaseCount + 1 {
		weights := table[p*taps : (p+1)*taps]
		var sum float64
		for j := range weights {
			// The tap j is the source frame j-kernel.halfTaps frames after the one at or before the position.
			d := float64(p)/float64(phaseCount) + float64(kernel.halfTaps-j)
			weights[j] = kernel.weight(d)
			sum += weights[j]
		}
		// The sum of the truncated, windowed kernel is not exactly 1 and varies with the phase.
		// Dividing by it keeps a constant signal constant.
		for j := range weights {
			weights[j] /= sum
		}
	}
	resamplingWeightTables[key] = table
	return table
}

func gcd(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// errSourceNotReady defers output until a source block can be completed.
var errSourceNotReady = errors.New("convert: source made no progress")

type Resampling struct {
	source io.Reader

	// declaredSrcLength is the length in bytes given by the caller. -1 indicates the length is unknown.
	declaredSrcLength int64

	// derivedSrcLength is the length in bytes determined when the source reaches its end.
	// derivedSrcLength is used only when declaredSrcLength is -1. -1 indicates the length is not determined yet.
	derivedSrcLength atomic.Int64

	// srcReadEnd is the offset in bytes just past the last byte the source has delivered so far.
	srcReadEnd int64

	from            int
	to              int
	bitDepthInBytes int
	pos             int64

	// reducedFrom and reducedTo are from and to divided by their greatest common divisor.
	// The output frame t is at t*reducedFrom/reducedTo frames in the source.
	reducedFrom uint64
	reducedTo   uint64

	// srcFrameStep and srcRemStep are the whole frames and the 1/reducedTo frames
	// that the position in the source advances by per output frame.
	srcFrameStep int64
	srcRemStep   uint64

	// maxDstFrame is the last output frame whose source frames have offsets in bytes that fit in int64.
	// -1 indicates no output frame is computed, for a non-positive sample rate.
	maxDstFrame int64

	kernel resamplingKernel

	// phaseCount is the number of phases in a source frame that weights holds.
	// phaseCount equals reducedTo unless reducedTo exceeds maxResamplingPhases.
	phaseCount int

	// weights is the weight table for kernel and phaseCount, shared among Resamplings.
	weights []float64

	// interpolatedWeightsBuf holds the weights of a phase between two phases in weights.
	interpolatedWeightsBuf []float64

	// tapsBufL and tapsBufR hold the source frames of an output frame whose taps do not lie in one block.
	tapsBufL []float64
	tapsBufR []float64

	// srcBlock is the most recently used block, and srcBlockL and srcBlockR are its frames.
	// srcBlock is -1 before a block is used.
	srcBlock  int64
	srcBlockL []float64
	srcBlockR []float64

	// lastReadSrcBlock is the last block read completely or through EOF.
	// It is -1 before the first block is read. Its value is ignored when
	// lastReadSrcBlockValid is false.
	lastReadSrcBlock      int64
	lastReadSrcBlockValid bool
	srcBufL               map[int64][]float64
	srcBufR               map[int64][]float64
	lruSrcBlocks          []int64
	eof                   bool

	// partialSrcBuf is the beginning of the block partialSrcBlock read before a source error or an empty read.
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
		from:                  from,
		bitDepthInBytes:       bitDepthInBytes,
		to:                    to,
		maxDstFrame:           -1,
		srcBlock:              -1,
		lastReadSrcBlock:      -1,
		lastReadSrcBlockValid: true,
		srcBufL:               map[int64][]float64{},
		srcBufR:               map[int64][]float64{},
	}
	r.derivedSrcLength.Store(-1)

	if from > 0 && to > 0 {
		g := gcd(int64(from), int64(to))
		r.reducedFrom = uint64(int64(from) / g)
		r.reducedTo = uint64(int64(to) / g)
		r.srcFrameStep = int64(r.reducedFrom / r.reducedTo)
		r.srcRemStep = r.reducedFrom % r.reducedTo
		r.kernel = newResamplingKernel(r.reducedFrom, r.reducedTo)

		// Byte offsets through the end of the block that holds maxSrcFrame fit in int64.
		maxSrcFrame := math.MaxInt64/int64(r.bytesPerSample()) - resamplingBlockFrames
		maxDstFrame, ok := mathutil.MulDiv(maxSrcFrame-int64(r.kernel.halfTaps), int64(r.reducedTo), int64(r.reducedFrom))
		if !ok {
			maxDstFrame = math.MaxInt64
		}
		r.maxDstFrame = maxDstFrame

		r.phaseCount = int(min(r.reducedTo, maxResamplingPhases))
		r.weights = resamplingWeightTable(r.kernel, r.phaseCount)
		taps := r.kernel.taps()
		r.interpolatedWeightsBuf = make([]float64, taps)
		r.tapsBufL = make([]float64, taps)
		r.tapsBufR = make([]float64, taps)
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
	return r.derivedSrcLength.Load()
}

// resampledLength returns the length of the resampled stream in bytes for a source stream of the given length.
func (r *Resampling) resampledLength(srcLength int64) int64 {
	s, ok := mathutil.MulDiv(srcLength, int64(r.to), int64(r.from))
	if !ok {
		panic("convert: resampled length is out of range")
	}
	return s / int64(r.bytesPerSample()) * int64(r.bytesPerSample())
}

// block returns the frames of each channel in the source block at index.
// Frames past the end of the source are zero.
// It returns io.EOF and no frames when the block is beyond the bytes the source has delivered and the end is unknown.
func (r *Resampling) block(index int64) ([]float64, []float64, error) {
	if index == r.srcBlock {
		return r.srcBlockL, r.srcBlockR, nil
	}

	sizePerSample := int64(r.bytesPerSample())
	if _, ok := r.srcBufL[index]; !ok {
		blockStart := index * resamplingBlockFrames * sizePerSample
		buf := make([]byte, resamplingBlockFrames*sizePerSample)
		var c int
		if r.partialSrcBuf != nil && r.partialSrcBlock == index {
			c = copy(buf, r.partialSrcBuf)
		} else if !r.lastReadSrcBlockValid || r.lastReadSrcBlock+1 != index {
			seeker, ok := r.source.(io.Seeker)
			if !ok {
				return nil, nil, fmt.Errorf("convert: source must be io.Seeker: %w", errors.ErrUnsupported)
			}
			if _, err := seeker.Seek(blockStart, io.SeekStart); err != nil {
				return nil, nil, err
			}
		}
		r.partialSrcBuf = nil
		var eof bool
		for c < len(buf) {
			n, err := r.source.Read(buf[c:])
			c += n
			if n == 0 && err == nil {
				// Preserve the existing silence padding past a declared end.
				if length := r.srcLength(); length >= 0 && blockStart+int64(c) >= length {
					r.lastReadSrcBlockValid = false
					break
				}
				err = errSourceNotReady
			}
			if err != nil {
				if err == io.EOF {
					eof = true
					// Determine the source length the first time the source reaches its end.
					// A block beyond the bytes delivered so far is reached only by seeking, and its end
					// without any byte tells only that the source ends at or before the block.
					if r.declaredSrcLength < 0 && r.derivedSrcLength.Load() < 0 && (c > 0 || blockStart <= r.srcReadEnd) {
						r.derivedSrcLength.Store(blockStart + int64(c))
					}
					r.lastReadSrcBlock = index
					r.lastReadSrcBlockValid = true
					break
				}
				r.lastReadSrcBlockValid = false
				// Keep the bytes read so far, as a source that is not an io.Seeker cannot deliver them again.
				if c > 0 || err == errSourceNotReady {
					r.partialSrcBlock = index
					r.partialSrcBuf = buf[:c]
					if c > 0 {
						r.srcReadEnd = max(r.srcReadEnd, blockStart+int64(c))
					}
				}
				return nil, nil, err
			}
		}
		if c == len(buf) {
			r.lastReadSrcBlock = index
			r.lastReadSrcBlockValid = true
		}
		if c > 0 {
			r.srcReadEnd = max(r.srcReadEnd, blockStart+int64(c))
		}
		// A block whose end position is not determined must not be cached, so that the block is read
		// again once the preceding data has been delivered.
		if eof && c == 0 && r.srcLength() < 0 && blockStart > r.srcReadEnd {
			return nil, nil, io.EOF
		}
		buf = buf[:c]
		sl := make([]float64, resamplingBlockFrames)
		sr := make([]float64, resamplingBlockFrames)
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
		r.srcBlock = index
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
		r.srcBlock = index
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
	r.srcBlockL = r.srcBufL[r.srcBlock]
	r.srcBlockR = r.srcBufR[r.srcBlock]
	return r.srcBlockL, r.srcBlockR, nil
}

// taps returns the source frames [first, first+r.kernel.taps()) of each channel.
// Frames before the start of the source are zero. The returned slices are valid until the next call.
// It returns io.EOF when a block of the frames is beyond the bytes the source has delivered and the end is unknown.
func (r *Resampling) taps(first int64) ([]float64, []float64, error) {
	n := int64(len(r.tapsBufL))
	last := first + n - 1
	if first >= 0 && first/resamplingBlockFrames == last/resamplingBlockFrames {
		sl, sr, err := r.block(first / resamplingBlockFrames)
		if err == nil {
			i := first % resamplingBlockFrames
			return sl[i : i+n], sr[i : i+n], nil
		}
		if err != io.EOF {
			return nil, nil, err
		}
		clear(r.tapsBufL)
		clear(r.tapsBufR)
		return r.tapsBufL, r.tapsBufR, io.EOF
	}

	clear(r.tapsBufL)
	clear(r.tapsBufR)
	var eof bool
	for index := max(first, 0) / resamplingBlockFrames; index <= last/resamplingBlockFrames; index++ {
		sl, sr, err := r.block(index)
		if err == io.EOF {
			eof = true
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		blockFirst := index * resamplingBlockFrames
		start := max(first, blockFirst)
		end := min(last+1, blockFirst+resamplingBlockFrames)
		copy(r.tapsBufL[start-first:end-first], sl[start-blockFirst:end-blockFirst])
		copy(r.tapsBufR[start-first:end-first], sr[start-blockFirst:end-blockFirst])
	}
	if eof {
		return r.tapsBufL, r.tapsBufR, io.EOF
	}
	return r.tapsBufL, r.tapsBufR, nil
}

// phaseWeights returns the tap weights for an output frame srcRem/reducedTo frames after a source frame.
// The returned slice is valid until the next call.
func (r *Resampling) phaseWeights(srcRem uint64) []float64 {
	buf := r.interpolatedWeightsBuf
	n := len(buf)
	if uint64(r.phaseCount) == r.reducedTo {
		i := int(srcRem) * n
		return r.weights[i : i+n]
	}

	// Interpolate the weights linearly between the two nearest phases.
	pos := float64(srcRem) / float64(r.reducedTo) * float64(r.phaseCount)
	p := min(int(pos), r.phaseCount-1)
	a := pos - float64(p)
	w0 := r.weights[p*n:][:n]
	w1 := r.weights[(p+1)*n:][:n]
	for j := range buf {
		buf[j] = w0[j] + (w1[j]-w0[j])*a
	}
	return buf
}

// srcPosition returns the position of the output frame dstFrame in the source, srcFrame+srcRem/reducedTo frames.
// dstFrame must be in [0, maxDstFrame].
func (r *Resampling) srcPosition(dstFrame int64) (srcFrame int64, srcRem uint64) {
	hi, lo := bits.Mul64(uint64(dstFrame), r.reducedFrom)
	q, rem := bits.Div64(hi, lo, r.reducedTo)
	return int64(q), rem
}

// frame returns the output frame at srcFrame+srcRem/reducedTo frames in the source.
// It returns io.EOF when the source frames include a block beyond the bytes the source has delivered and the end is unknown.
func (r *Resampling) frame(srcFrame int64, srcRem uint64) (float64, float64, error) {
	srcL, srcR, err := r.taps(srcFrame - int64(r.kernel.halfTaps))
	if err != nil && err != io.EOF {
		return 0, 0, err
	}
	weights := r.phaseWeights(srcRem)
	srcL = srcL[:len(weights)]
	srcR = srcR[:len(weights)]
	var lv, rv float64
	for j, w := range weights {
		lv += srcL[j] * w
		rv += srcR[j] * w
	}
	return lv, rv, err
}

// frameAt returns the output frame dstFrame. See frame for the error.
func (r *Resampling) frameAt(dstFrame int64) (float64, float64, error) {
	if dstFrame > r.maxDstFrame {
		return 0, 0, io.EOF
	}
	srcFrame, srcRem := r.srcPosition(dstFrame)
	return r.frame(srcFrame, srcRem)
}

// Read reads resampled audio and returns an error wrapping [errors.ErrUnsupported] if it requires seeking a non-seekable source.
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
			_, _, err := r.frameAt(r.pos / int64(size))
			if err != nil && err != io.EOF {
				if err == errSourceNotReady {
					err = io.ErrShortBuffer
				}
				return 0, err
			}
			if (err == io.EOF && r.srcLength() < 0) || (r.srcLength() >= 0 && r.pos >= r.Length()) {
				r.eof = true
				return 0, io.EOF
			}
		}
		return 0, io.ErrShortBuffer
	}

	dstFrame := r.pos / int64(size)
	var srcFrame int64
	var srcRem uint64
	if dstFrame <= r.maxDstFrame {
		srcFrame, srcRem = r.srcPosition(dstFrame)
	}
	length := r.Length()
	n := len(b) / size * size
	for i := range n / size {
		var ldata, rdata float64
		// The source of a frame past maxDstFrame is beyond any byte the source can deliver.
		var err error = io.EOF
		if dstFrame <= r.maxDstFrame {
			ldata, rdata, err = r.frame(srcFrame, srcRem)
		}
		if err != nil && err != io.EOF {
			r.pos += int64(size * i)
			if err == errSourceNotReady {
				err = nil
			}
			return size * i, err
		}
		// EOF from the frame method indicates that the source reaches the end, and doesn't indicate the resampled data ends.
		// The length check below is the terminator of the resampled data, except when the length is still unknown:
		// then the position is beyond the source's end after seeking, and the resampled data ends here.
		if err == io.EOF && r.srcLength() < 0 {
			n = size * i
			r.eof = true
			break
		}
		// If the length is known, check whether the resampled data ends (#3352).
		if length < 0 {
			length = r.Length()
		}
		if length >= 0 && r.pos+int64(size*i) >= length {
			n = size * i
			r.eof = true
			break
		}

		switch r.bitDepthInBytes {
		case 2:
			ldata = min(max(ldata, -1), 1)
			rdata = min(max(rdata, -1), 1)
			l16 := int16(ldata * (1<<15 - 1))
			r16 := int16(rdata * (1<<15 - 1))
			b[4*i] = byte(l16)
			b[4*i+1] = byte(l16 >> 8)
			b[4*i+2] = byte(r16)
			b[4*i+3] = byte(r16 >> 8)
		case 4:
			l32b := math.Float32bits(float32(ldata))
			r32b := math.Float32bits(float32(rdata))
			b[8*i] = byte(l32b)
			b[8*i+1] = byte(l32b >> 8)
			b[8*i+2] = byte(l32b >> 16)
			b[8*i+3] = byte(l32b >> 24)
			b[8*i+4] = byte(r32b)
			b[8*i+5] = byte(r32b >> 8)
			b[8*i+6] = byte(r32b >> 16)
			b[8*i+7] = byte(r32b >> 24)
		default:
			panic("not reached")
		}

		dstFrame++
		srcFrame += r.srcFrameStep
		srcRem += r.srcRemStep
		if srcRem >= r.reducedTo {
			srcRem -= r.reducedTo
			srcFrame++
		}
	}
	r.pos += int64(n)
	if r.eof {
		return n, io.EOF
	}
	return n, nil
}

// Seek sets the stream position and returns an error wrapping [errors.ErrUnsupported] if the source is not an [io.Seeker]
// or whence is [io.SeekEnd] and the length is unknown.
func (r *Resampling) Seek(offset int64, whence int) (int64, error) {
	if _, ok := r.source.(io.Seeker); !ok {
		return 0, fmt.Errorf("convert: source must be io.Seeker: %w", errors.ErrUnsupported)
	}

	var base int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		// A query keeps the end-of-stream state.
		if offset == 0 {
			return r.pos, nil
		}
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
