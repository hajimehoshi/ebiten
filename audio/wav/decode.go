// Copyright 2016 Hajime Hoshi
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

// Package wav provides WAV (RIFF) decoder.
package wav

import (
	"bytes"
	"fmt"
	"io"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
)

// Stream is a decoded audio stream.
type Stream struct {
	inner io.ReadSeeker
	size  int64
}

// Read is implementation of io.Reader's Read.
func (s *Stream) Read(p []byte) (int, error) {
	return s.inner.Read(p)
}

// Seek is implementation of io.Seeker's Seek.
//
// Note that Seek can take long since decoding is a relatively heavy task.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.inner.Seek(offset, whence)
}

// Length returns the size of decoded stream in bytes.
func (s *Stream) Length() int64 {
	return s.size
}

type stream struct {
	src        io.ReadSeeker
	headerSize int64
	dataSize   int64
	remaining  int64
}

// Read is implementation of io.Reader's Read.
func (s *stream) Read(p []byte) (int, error) {
	if s.remaining <= 0 {
		return 0, io.EOF
	}
	if s.remaining < int64(len(p)) {
		p = p[0:s.remaining]
	}
	n, err := s.src.Read(p)
	s.remaining -= int64(n)
	return n, err
}

// Seek is implementation of io.Seeker's Seek.
func (s *stream) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		offset = offset + s.headerSize
	case io.SeekCurrent:
	case io.SeekEnd:
		offset = s.headerSize + s.dataSize + offset
		whence = io.SeekStart
	}
	n, err := s.src.Seek(offset, whence)
	if err != nil {
		return 0, err
	}
	if n-s.headerSize < 0 {
		return 0, fmt.Errorf("wav: invalid offset")
	}
	s.remaining = s.dataSize - (n - s.headerSize)
	// There could be a tail in wav file.
	if s.remaining < 0 {
		s.remaining = 0
		return s.dataSize, nil
	}
	return n - s.headerSize, nil
}

// DecodeWithSampleRate decodes WAV (RIFF) data to playable stream.
//
// The format must be 1 or 2 channels, 8bit or 16bit little endian PCM.
// The format is converted into 2 channels and 16bit.
//
// DecodeWithSampleRate returns error when decoding fails or IO error happens.
//
// DecodeWithSampleRate automatically resamples the stream to fit with sampleRate if necessary.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
func DecodeWithSampleRate(sampleRate int, src io.ReadSeeker) (*Stream, error) {
	buf := make([]byte, 12)
	n, err := io.ReadFull(src, buf)
	if n != len(buf) {
		return nil, fmt.Errorf("wav: invalid header")
	}
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(buf[0:4], []byte("RIFF")) {
		return nil, fmt.Errorf("wav: invalid header: 'RIFF' not found")
	}
	if !bytes.Equal(buf[8:12], []byte("WAVE")) {
		return nil, fmt.Errorf("wav: invalid header: 'WAVE' not found")
	}

	// Read chunks
	dataSize := int64(0)
	headerSize := int64(len(buf))
	sampleRateFrom := 0
	sampleRateTo := 0
	mono := false
	bitsPerSample := 0
chunks:
	for {
		buf := make([]byte, 8)
		n, err := io.ReadFull(src, buf)
		if n != len(buf) {
			return nil, fmt.Errorf("wav: invalid header")
		}
		if err != nil {
			return nil, err
		}
		headerSize += 8
		size := int64(buf[4]) | int64(buf[5])<<8 | int64(buf[6])<<16 | int64(buf[7])<<24
		switch {
		case bytes.Equal(buf[0:4], []byte("fmt ")):
			// Size of 'fmt' header is usually 16, but can be more than 16.
			if size < 16 {
				return nil, fmt.Errorf("wav: invalid header: maybe non-PCM file?")
			}
			buf := make([]byte, size)
			n, err := io.ReadFull(src, buf)
			if n != len(buf) {
				return nil, fmt.Errorf("wav: invalid header")
			}
			if err != nil {
				return nil, err
			}
			format := int(buf[0]) | int(buf[1])<<8
			if format != 1 {
				return nil, fmt.Errorf("wav: format must be linear PCM")
			}
			channelNum := int(buf[2]) | int(buf[3])<<8
			switch channelNum {
			case 1:
				mono = true
			case 2:
				mono = false
			default:
				return nil, fmt.Errorf("wav: channel num must be 1 or 2 but was %d", channelNum)
			}
			bitsPerSample = int(buf[14]) | int(buf[15])<<8
			if bitsPerSample != 8 && bitsPerSample != 16 {
				return nil, fmt.Errorf("wav: bits per sample must be 8 or 16 but was %d", bitsPerSample)
			}
			origSampleRate := int64(buf[4]) | int64(buf[5])<<8 | int64(buf[6])<<16 | int64(buf[7])<<24
			if int64(sampleRate) != origSampleRate {
				sampleRateFrom = int(origSampleRate)
				sampleRateTo = sampleRate
			}
			headerSize += size
		case bytes.Equal(buf[0:4], []byte("data")):
			dataSize = size
			break chunks
		default:
			buf := make([]byte, size)
			n, err := io.ReadFull(src, buf)
			if n != len(buf) {
				return nil, fmt.Errorf("wav: invalid header")
			}
			if err != nil {
				return nil, err
			}
			headerSize += size
		}
	}
	var s io.ReadSeeker = &stream{
		src:        src,
		headerSize: headerSize,
		dataSize:   dataSize,
		remaining:  dataSize,
	}

	if mono || bitsPerSample != 16 {
		s = convert.NewStereo16(s, mono, bitsPerSample != 16)
		if mono {
			dataSize *= 2
		}
		if bitsPerSample != 16 {
			dataSize *= 2
		}
	}
	if sampleRateFrom != sampleRateTo {
		r := convert.NewResampling(s, dataSize, sampleRateFrom, sampleRateTo)
		s = r
		dataSize = r.Length()
	}
	ss := &Stream{inner: s, size: dataSize}
	return ss, nil
}

// Decode decodes WAV (RIFF) data to playable stream.
//
// The format must be 1 or 2 channels, 8bit or 16bit little endian PCM.
// The format is converted into 2 channels and 16bit.
//
// Decode returns error when decoding fails or IO error happens.
//
// Decode automatically resamples the stream to fit with the audio context if necessary.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
//
// Deprecated: as of v2.1. Use DecodeWithSampleRate instead.
func Decode(context *audio.Context, src io.ReadSeeker) (*Stream, error) {
	return DecodeWithSampleRate(context.SampleRate(), src)
}
