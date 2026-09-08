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

package audio

import (
	"fmt"
	"io"
	"math"
)

// InfiniteLoop represents a looped stream which never ends as long as its source has data to loop.
type InfiniteLoop struct {
	src     io.ReadSeeker
	lstart  int64
	llength int64

	// pos is the position of src. This is ahead of the position of the data returned so far by len(extra).
	pos int64

	bitDepthInBytes int
	bytesPerSample  int

	// extra is the remainder in the case when the read byte sizes are not multiple of the bit depth.
	extra []byte

	// afterLoop is data after the loop.
	afterLoop []byte

	// blending represents whether the loop start and afterLoop are blended or not.
	blending bool

	noBlendForTesting bool
}

// NewInfiniteLoop creates a new infinite loop stream with a source stream and length in bytes.
//
// src is a signed 16bit integer little endian stream, 2 channels (stereo).
//
// If the loop's total length is exactly the same as src's length, you might hear noises around the loop joint.
// This noise can be heard especially when src is decoded from a lossy compression format like Ogg/Vorbis and MP3.
// In this case, try to add more (about 0.1[s]) data to src after the loop end.
// If src has data after the loop end, an InfiniteLoop uses part of the data to blend with the loop start
// to make the loop joint smooth.
func NewInfiniteLoop(src io.ReadSeeker, length int64) *InfiniteLoop {
	return newInfiniteLoopWithIntro(src, 0, length, bitDepthInBytesInt16)
}

// NewInfiniteLoopF32 creates a new infinite loop stream with a source stream and length in bytes.
//
// src is a 32bit float little endian stream, 2 channels (stereo).
//
// If the loop's total length is exactly the same as src's length, you might hear noises around the loop joint.
// This noise can be heard especially when src is decoded from a lossy compression format like Ogg/Vorbis and MP3.
// In this case, try to add more (about 0.1[s]) data to src after the loop end.
// If src has data after the loop end, an InfiniteLoop uses part of the data to blend with the loop start
// to make the loop joint smooth.
func NewInfiniteLoopF32(src io.ReadSeeker, length int64) *InfiniteLoop {
	return newInfiniteLoopWithIntro(src, 0, length, bitDepthInBytesFloat32)
}

// NewInfiniteLoopWithIntro creates a new infinite loop stream with an intro part.
// NewInfiniteLoopWithIntro accepts a source stream src, introLength in bytes and loopLength in bytes.
//
// src is a signed 16bit integer little endian stream, 2 channels (stereo).
//
// If the loop's total length is exactly the same as src's length, you might hear noises around the loop joint.
// This noise can be heard especially when src is decoded from a lossy compression format like Ogg/Vorbis and MP3.
// In this case, try to add more (about 0.1[s]) data to src after the loop end.
// If src has data after the loop end, an InfiniteLoop uses part of the data to blend with the loop start
// to make the loop joint smooth.
func NewInfiniteLoopWithIntro(src io.ReadSeeker, introLength int64, loopLength int64) *InfiniteLoop {
	return newInfiniteLoopWithIntro(src, introLength, loopLength, bitDepthInBytesInt16)
}

// NewInfiniteLoopWithIntroF32 creates a new infinite loop stream with an intro part.
// NewInfiniteLoopWithIntroF32 accepts a source stream src, introLength in bytes and loopLength in bytes.
//
// src is a 32bit float little endian stream, 2 channels (stereo).
//
// If the loop's total length is exactly the same as src's length, you might hear noises around the loop joint.
// This noise can be heard especially when src is decoded from a lossy compression format like Ogg/Vorbis and MP3.
// In this case, try to add more (about 0.1[s]) data to src after the loop end.
// If src has data after the loop end, an InfiniteLoop uses part of the data to blend with the loop start
// to make the loop joint smooth.
func NewInfiniteLoopWithIntroF32(src io.ReadSeeker, introLength int64, loopLength int64) *InfiniteLoop {
	return newInfiniteLoopWithIntro(src, introLength, loopLength, bitDepthInBytesFloat32)
}

func newInfiniteLoopWithIntro(src io.ReadSeeker, introLength int64, loopLength int64, bitDepthInBytes int) *InfiniteLoop {
	bytesPerSample := bitDepthInBytes * channelCount
	lstart := introLength / int64(bytesPerSample) * int64(bytesPerSample)
	llength := loopLength / int64(bytesPerSample) * int64(bytesPerSample)
	if llength <= 0 {
		panic(fmt.Sprintf("audio: loop length must be a positive multiple of %d bytes but was %d", bytesPerSample, loopLength))
	}
	return &InfiniteLoop{
		src:             src,
		lstart:          lstart,
		llength:         llength,
		pos:             -1,
		bitDepthInBytes: bitDepthInBytes,
		bytesPerSample:  bytesPerSample,
	}
}

func (i *InfiniteLoop) length() int64 {
	return i.lstart + i.llength
}

func (i *InfiniteLoop) ensurePos() error {
	if i.pos >= 0 {
		return nil
	}
	pos, err := i.src.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if pos >= i.length() {
		return fmt.Errorf("audio: stream position must be less than the specified length")
	}
	i.pos = pos
	return nil
}

func (i *InfiniteLoop) blendRate(pos int64) float32 {
	if pos < i.lstart {
		return 0
	}
	if pos >= i.lstart+int64(len(i.afterLoop)) {
		return 0
	}
	p := (pos - i.lstart) / int64(i.bytesPerSample)
	l := len(i.afterLoop) / i.bytesPerSample
	if l == 0 {
		return 0
	}
	return 1 - float32(p)/float32(l)
}

// Read is implementation of ReadSeeker's Read.
//
// If the source ends before the loop, the loop has nothing to repeat and Read returns [io.EOF].
// If the source ends inside the loop, the loop is shortened to end there.
func (i *InfiniteLoop) Read(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	// A buffer shorter than one sample cannot receive any data, and cannot hold the remainder
	// carried over from the previous Read.
	if len(b) < i.bitDepthInBytes {
		return 0, io.ErrShortBuffer
	}

	if err := i.ensurePos(); err != nil {
		return 0, err
	}

	// When the source or the loop reaches its end, go back to the loop start and read again so that
	// this doesn't return (0, nil). A retry either returns data, stops at the loop start, or grows
	// the remainder, which is shorter than one value, so the retries end.
	for rewinds := 0; ; rewinds++ {
		n, err := i.read(b)
		if err != nil && err != io.EOF {
			return 0, err
		}
		atEnd := i.pos == i.length() || err == io.EOF
		if !atEnd {
			return n, nil
		}
		if n == 0 && (i.pos == i.lstart || rewinds > 0) {
			// The loop has no data to repeat.
			return 0, io.EOF
		}
		if err := i.rewind(); err != nil {
			return 0, err
		}
		if n > 0 {
			return n, nil
		}
	}
}

// read reads data at the current position, and returns [io.EOF] when the source reaches its end.
func (i *InfiniteLoop) read(b []byte) (int, error) {
	extralen := len(i.extra)
	if i.pos+int64(len(b))-int64(extralen) > i.length() {
		b = b[:i.length()-i.pos+int64(extralen)]
	}

	copy(b, i.extra)
	i.extra = i.extra[:0]

	// Keep reading until one sample is available so that a source returning less than one sample
	// at a time doesn't make Read return (0, nil).
	var n int
	var err error
	for {
		var m int
		m, err = i.src.Read(b[extralen+n:])
		n += m
		if err != nil || m == 0 || extralen+n >= i.bitDepthInBytes {
			break
		}
	}
	i.pos += int64(n)
	n += extralen
	if i.pos > i.length() {
		panic(fmt.Sprintf("audio: position %d exceeds length %d at (*InfiniteLoop).Read", i.pos, i.length()))
	}

	// bpos is the stream position of b[0], which must be calculated before the remainder is removed from b.
	bpos := i.pos - int64(n)

	// Save the remainder part to extra. This will be used at the next Read.
	if rem := n % i.bitDepthInBytes; rem != 0 {
		i.extra = append(i.extra, b[n-rem:n]...)
		b = b[:n-rem]
		n = n - rem
	}

	// Blend afterLoop and the loop start to reduce noises (#1888).
	// Ideally, afterLoop and the loop start should be identical, but they can have very slight differences.
	if !i.noBlendForTesting && i.blending && i.pos >= i.lstart && bpos < i.lstart+int64(len(i.afterLoop)) {
		if n%i.bitDepthInBytes != 0 {
			panic(fmt.Sprintf("audio: n (%d) must be a multiple of bit depth %d [bytes]", n, i.bitDepthInBytes))
		}
		for idx := 0; idx < n/i.bitDepthInBytes; idx++ {
			abspos := bpos + int64(idx)*int64(i.bitDepthInBytes)
			rate := i.blendRate(abspos)
			if rate == 0 {
				continue
			}

			relpos := abspos - i.lstart
			switch i.bitDepthInBytes {
			case 2:
				afterLoop := int16(i.afterLoop[relpos]) | (int16(i.afterLoop[relpos+1]) << 8)
				orig := int16(b[2*idx]) | (int16(b[2*idx+1]) << 8)
				newVal := int16(float32(afterLoop)*rate + float32(orig)*(1-rate))
				b[2*idx] = byte(newVal)
				b[2*idx+1] = byte(newVal >> 8)
			case 4:
				afterLoop := math.Float32frombits(uint32(i.afterLoop[relpos]) | (uint32(i.afterLoop[relpos+1]) << 8) | (uint32(i.afterLoop[relpos+2]) << 16) | (uint32(i.afterLoop[relpos+3]) << 24))
				orig := math.Float32frombits(uint32(b[4*idx]) | (uint32(b[4*idx+1]) << 8) | (uint32(b[4*idx+2]) << 16) | (uint32(b[4*idx+3]) << 24))
				newVal := float32(afterLoop*rate + orig*(1-rate))
				newValBits := math.Float32bits(newVal)
				b[4*idx] = byte(newValBits)
				b[4*idx+1] = byte(newValBits >> 8)
				b[4*idx+2] = byte(newValBits >> 16)
				b[4*idx+3] = byte(newValBits >> 24)
			default:
				panic("not reached")
			}
		}
	}

	if err != nil && err != io.EOF {
		return 0, err
	}

	// Read the afterLoop part if necessary.
	if i.pos == i.length() && err == nil {
		if i.afterLoop == nil {
			buflen := min(int64(256*i.bytesPerSample), i.length())

			buf := make([]byte, buflen)
			var pos int
			for pos < len(buf) {
				n, err := i.src.Read(buf[pos:])
				if err != nil && err != io.EOF {
					return 0, err
				}
				pos += n
				// Break on EOF, and also when no progress is made so that a
				// source returning (0, nil) does not spin here forever. The
				// main read loop above has the same guard.
				if err != nil || n == 0 {
					break
				}
			}
			i.afterLoop = buf[:pos]
		}
		if len(i.afterLoop) > 0 {
			i.blending = true
		}
	}

	return n, err
}

// rewind moves the position back to the loop start.
func (i *InfiniteLoop) rewind() error {
	// Ignore the new position returned by Seek since the source position might not be match with the position
	// managed by this.
	if _, err := i.src.Seek(i.lstart, io.SeekStart); err != nil {
		return err
	}
	i.pos = i.lstart
	i.extra = i.extra[:0]
	return nil
}

// Seek is implementation of ReadSeeker's Seek.
//
// whence must be [io.SeekStart] or [io.SeekCurrent] since an [InfiniteLoop] has no end.
//
// The returned position can differ from the requested one with a nil error: a position beyond the loop end is folded
// into the loop, and a position in the middle of a value is rounded down to a value boundary.
func (i *InfiniteLoop) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart, io.SeekCurrent:
	default:
		return 0, fmt.Errorf("audio: whence must be io.SeekStart or io.SeekCurrent for InfiniteLoop but was %d", whence)
	}

	i.blending = false
	if err := i.ensurePos(); err != nil {
		return 0, err
	}

	var next int64
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = i.pos - int64(len(i.extra)) + offset
	}
	if next < 0 {
		return 0, fmt.Errorf("audio: position must be >= 0 but was %d", next)
	}
	// A position in the middle of a value is not a position this stream can be at: reading from
	// there would return values straddling two of the source's.
	next = next / int64(i.bitDepthInBytes) * int64(i.bitDepthInBytes)
	if next > i.lstart {
		next = ((next - i.lstart) % i.llength) + i.lstart
	}
	// Ignore the new position returned by Seek since the source position might not be match with the position
	// managed by this.
	if _, err := i.src.Seek(next, io.SeekStart); err != nil {
		return 0, err
	}
	i.pos = next
	i.extra = i.extra[:0]
	return i.pos, nil
}
