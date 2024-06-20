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
	"fmt"
	"io"
)

// Setero objects convert little-endian audio mono or stereo audio streams into
// 16-bit stereo stereo streams.
type Stereo struct {
	source io.ReadSeeker
	mono   bool
	bytes  int
	buf    []byte
}

// IsValidResolution returns true if the given bit resolution is supported.
func IsValidResolution(r int) bool {
	switch r {
	case 8, 16, 32, 24:
		return true
	}
	return false
}

// NewStereo accepts an io.ReadSeeker into an audio stream. Regardless of the input
// stream, subsequent calls to Stereo.Read will return 16-bit little-endian stereo
// audio samples.
//
// Valid values for resolution are: [8,16,24,32]. Any invalid input will panic.
func NewStereo(source io.ReadSeeker, mono bool, resolution int) *Stereo {
	if !IsValidResolution(resolution) {
		panic(fmt.Errorf("unsupported resolution: %d", resolution))
	}
	return &Stereo{source, mono, resolution / 8, nil}
}

// Read returns audio data from the input stream as 16-bit stereo (little-endian).
func (s *Stereo) Read(b []byte) (int, error) {
	// Calculate how large we need our buffer to be, and allocate it.
	l := (len(b) * s.bytes) / 2
	if s.mono {
		l /= 2
	}
	if cap(s.buf) < l {
		s.buf = make([]byte, l)
	}

	// Copy over the bits.
	n, err := s.source.Read(s.buf[:l])
	if err != nil && err != io.EOF {
		return 0, err
	}

	// We now need to tweak the data into the required format.
	switch {
	case s.mono && s.bytes == 1:
		for i := 0; i < n; i++ {
			v := int16(int(s.buf[i]) * 0x100)
			b[4*i+0] = byte(v)
			b[4*i+1] = byte(v >> 8)
			b[4*i+2] = byte(v)
			b[4*i+3] = byte(v >> 8)
		}
	case !s.mono && s.bytes == 1:
		for i := 0; i < n; i++ {
			v := int16(int(s.buf[i]) * 0x100)
			b[2*i+0] = byte(v)
			b[2*i+1] = byte(v >> 8)
		}

	case s.mono:
		m := s.bytes
		for i := 0; i < n/m; i++ {
			b[4*i+0] = s.buf[m*(i+1)-2]
			b[4*i+1] = s.buf[m*(i+1)-1]
			b[4*i+2] = s.buf[m*(i+1)-2]
			b[4*i+3] = s.buf[m*(i+1)-1]
		}
	case !s.mono:
		m := s.bytes
		for i := 0; i < n/m; i++ {
			b[2*i+0] = s.buf[m*(i+1)-2]
			b[2*i+1] = s.buf[m*(i+1)-1]
		}
	}

	// Return the number of bytes read.
	if s.mono {
		return (n * 4) / s.bytes, err
	}
	return (n * 2) / s.bytes, err
}

// Seek moves the location for next Read call in the audio stream.
func (s *Stereo) Seek(offset int64, whence int) (int64, error) {
	offset = (offset * int64(s.bytes))
	if s.mono {
		offset /= 2
	}
	return s.source.Seek(offset, whence)
}
