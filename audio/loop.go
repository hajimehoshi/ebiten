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
)

// InfiniteLoop represents a looped stream which never ends.
type InfiniteLoop struct {
	src     io.ReadSeeker
	lstart  int64
	llength int64
	pos     int64

	// extra is the remainder in the case when the read byte sizes are not multiple of bitDepthInBytesInt16.
	extra []byte

	// afterLoop is data after the loop.
	afterLoop []byte

	// blending represents whether the loop start and afterLoop are blended or not.
	blending bool

	noBlendForTesting bool
}

// NewInfiniteLoop creates a new infinite loop stream with a source stream and length in bytes.
//
// If the loop's total length is exactly the same as src's length, you might hear noises around the loop joint.
// This noise can be heard especially when src is decoded from a lossy compression format like Ogg/Vorbis and MP3.
// In this case, try to add more (about 0.1[s]) data to src after the loop end.
// If src has data after the loop end, an InfiniteLoop uses part of the data to blend with the loop start
// to make the loop joint smooth.
func NewInfiniteLoop(src io.ReadSeeker, length int64) *InfiniteLoop {
	return NewInfiniteLoopWithIntro(src, 0, length)
}

// NewInfiniteLoopWithIntro creates a new infinite loop stream with an intro part.
// NewInfiniteLoopWithIntro accepts a source stream src, introLength in bytes and loopLength in bytes.
//
// If the loop's total length is exactly the same as src's length, you might hear noises around the loop joint.
// This noise can be heard especially when src is decoded from a lossy compression format like Ogg/Vorbis and MP3.
// In this case, try to add more (about 0.1[s]) data to src after the loop end.
// If src has data after the loop end, an InfiniteLoop uses part of the data to blend with the loop start
// to make the loop joint smooth.
func NewInfiniteLoopWithIntro(src io.ReadSeeker, introLength int64, loopLength int64) *InfiniteLoop {
	return &InfiniteLoop{
		src:     src,
		lstart:  introLength / bytesPerSampleInt16 * bytesPerSampleInt16,
		llength: loopLength / bytesPerSampleInt16 * bytesPerSampleInt16,
		pos:     -1,
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

func (i *InfiniteLoop) blendRate(pos int64) float64 {
	if pos < i.lstart {
		return 0
	}
	if pos >= i.lstart+int64(len(i.afterLoop)) {
		return 0
	}
	p := (pos - i.lstart) / bytesPerSampleInt16
	l := len(i.afterLoop) / bytesPerSampleInt16
	return 1 - float64(p)/float64(l)
}

// Read is implementation of ReadSeeker's Read.
func (i *InfiniteLoop) Read(b []byte) (int, error) {
	if err := i.ensurePos(); err != nil {
		return 0, err
	}

	if i.pos+int64(len(b)) > i.length() {
		b = b[:i.length()-i.pos]
	}

	extralen := len(i.extra)
	copy(b, i.extra)
	i.extra = i.extra[:0]

	n, err := i.src.Read(b[extralen:])
	n += extralen
	i.pos += int64(n)
	if i.pos > i.length() {
		panic(fmt.Sprintf("audio: position must be <= length but not at (*InfiniteLoop).Read: pos: %d, length: %d", i.pos, i.length()))
	}

	// Save the remainder part to extra. This will be used at the next Read.
	if rem := n % bitDepthInBytesInt16; rem != 0 {
		i.extra = append(i.extra, b[n-rem:n]...)
		b = b[:n-rem]
		n = n - rem
	}

	// Blend afterLoop and the loop start to reduce noises (#1888).
	// Ideally, afterLoop and the loop start should be identical, but they can have very slight differences.
	if !i.noBlendForTesting && i.blending && i.pos >= i.lstart && i.pos-int64(n) < i.lstart+int64(len(i.afterLoop)) {
		if n%bitDepthInBytesInt16 != 0 {
			panic(fmt.Sprintf("audio: n must be a multiple of bitDepthInBytesInt16 but not: %d", n))
		}
		for idx := 0; idx < n/bitDepthInBytesInt16; idx++ {
			abspos := i.pos - int64(n) + int64(idx)*bitDepthInBytesInt16
			rate := i.blendRate(abspos)
			if rate == 0 {
				continue
			}

			// This assumes that bitDepthInBytesInt16 is 2.
			relpos := abspos - i.lstart
			afterLoop := int16(i.afterLoop[relpos]) | (int16(i.afterLoop[relpos+1]) << 8)
			orig := int16(b[2*idx]) | (int16(b[2*idx+1]) << 8)

			newval := int16(float64(afterLoop)*rate + float64(orig)*(1-rate))
			b[2*idx] = byte(newval)
			b[2*idx+1] = byte(newval >> 8)
		}
	}

	if err != nil && err != io.EOF {
		return 0, err
	}

	// Read the afterLoop part if necessary.
	if i.pos == i.length() && err == nil {
		if i.afterLoop == nil {
			buflen := int64(256 * bytesPerSampleInt16)
			if buflen > i.length() {
				buflen = i.length()
			}

			buf := make([]byte, buflen)
			pos := 0
			for pos < len(buf) {
				n, err := i.src.Read(buf[pos:])
				if err != nil && err != io.EOF {
					return 0, err
				}
				pos += n
				if err == io.EOF {
					break
				}
			}
			i.afterLoop = buf[:pos]
		}
		if len(i.afterLoop) > 0 {
			i.blending = true
		}
	}

	if i.pos == i.length() || err == io.EOF {
		// Ignore the new position returned by Seek since the source position might not be match with the position
		// managed by this.
		if _, err := i.src.Seek(i.lstart, io.SeekStart); err != nil {
			return 0, err
		}
		i.pos = i.lstart
	}
	return n, nil
}

// Seek is implementation of ReadSeeker's Seek.
func (i *InfiniteLoop) Seek(offset int64, whence int) (int64, error) {
	i.blending = false
	if err := i.ensurePos(); err != nil {
		return 0, err
	}

	next := int64(0)
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = i.pos + offset
	case io.SeekEnd:
		return 0, fmt.Errorf("audio: whence must be io.SeekStart or io.SeekCurrent for InfiniteLoop")
	}
	if next < 0 {
		return 0, fmt.Errorf("audio: position must >= 0")
	}
	if next > i.lstart {
		next = ((next - i.lstart) % i.llength) + i.lstart
	}
	// Ignore the new position returned by Seek since the source position might not be match with the position
	// managed by this.
	if _, err := i.src.Seek(next, io.SeekStart); err != nil {
		return 0, err
	}
	i.pos = next
	return i.pos, nil
}
