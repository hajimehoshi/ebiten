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

const (
	bitDepthInBytesInt16   = 2
	bitDepthInBytesFloat32 = 4
)

// Stream is a decoded audio stream.
//
// The format is signed 16bit integer little endian PCM (DecodeWithoutResampling, etc.),
// or 32bit float little endian PCM (DeocdeF32).
// The channel count is 2.
type Stream struct {
	inner      io.ReadSeeker
	size       int64
	sampleRate int
}

// Read is implementation of io.Reader's Read.
func (s *Stream) Read(p []byte) (int, error) {
	return s.inner.Read(p)
}

// Seek is implementation of io.Seeker's Seek.
//
// Note that Seek can take long since decoding is a relatively heavy task.
//
// If the underlying source is not an io.Seeker, Seek panics.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.inner.Seek(offset, whence)
}

// Length returns the size of decoded stream in bytes.
func (s *Stream) Length() int64 {
	return s.size
}

// SampleRate returns the sample rate of the decoded stream.
func (s *Stream) SampleRate() int {
	return s.sampleRate
}

// DecodeF32 decodes WAV (RIFF) data to playable stream in 32bit float, little endian, 2 channels (stereo) format.
//
// The src format must be 1 or 2 channels, 8bit or 16bit little endian PCM.
// The src format is converted into 2 channels and 16bit.
//
// DecodeF32 returns error when decoding fails or IO error happens.
//
// The returned Stream's Seek is available only when src is an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
func DecodeF32(src io.Reader) (*Stream, error) {
	s, err := decode(src, bitDepthInBytesFloat32)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// DecodeWithoutResampling decodes WAV (RIFF) data to playable stream in signed 16bit integer, little endian, 2 channels (stereo) format.
//
// The src format must be 1 or 2 channels, 8bit or 16bit little endian PCM.
// The src format is converted into 2 channels and 16bit.
//
// DecodeWithoutSampleRate returns error when decoding fails or IO error happens.
//
// The returned Stream's Seek is available only when src is an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
func DecodeWithoutResampling(src io.Reader) (*Stream, error) {
	s, err := decode(src, bitDepthInBytesInt16)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// DecodeWithSampleRate decodes WAV (RIFF) data to playable stream in signed 16bit integer, little endian, 2 channels (stereo) format.
//
// The format must be 1 or 2 channels, 8bit or 16bit little endian PCM.
// The format is converted into 2 channels and 16bit.
//
// DecodeWithSampleRate returns error when decoding fails or IO error happens.
//
// DecodeWithSampleRate automatically resamples the stream to fit with sampleRate if necessary.
//
// The returned Stream's Seek is available only when src is an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
//
// Resampling can be a very heavy task. Stream has a cache for resampling, but the size is limited.
// Do not expect that Stream has a resampling cache even after whole data is played.
func DecodeWithSampleRate(sampleRate int, src io.Reader) (*Stream, error) {
	s, err := decode(src, bitDepthInBytesInt16)
	if err != nil {
		return nil, err
	}

	if sampleRate == s.sampleRate {
		return s, nil
	}

	r := convert.NewResampling(s.inner, s.size, s.sampleRate, sampleRate, bitDepthInBytesInt16)
	return &Stream{
		inner:      r,
		size:       r.Length(),
		sampleRate: sampleRate,
	}, nil
}

func decode(src io.Reader, bitDepthInBytes int) (*Stream, error) {
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
	var dataSize int64
	headerSize := int64(len(buf))
	var mono bool
	var bitsPerSample int
	var sampleRate int
chunks:
	for {
		var buf [8]byte
		n, err := io.ReadFull(src, buf[:])
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
			channelCount := int(buf[2]) | int(buf[3])<<8
			switch channelCount {
			case 1:
				mono = true
			case 2:
				mono = false
			default:
				return nil, fmt.Errorf("wav: number of channels must be 1 or 2 but was %d", channelCount)
			}
			bitsPerSample = int(buf[14]) | int(buf[15])<<8
			// TODO: Support signed 24bit integer format (#2215).
			if bitsPerSample != 8 && bitsPerSample != 16 {
				return nil, fmt.Errorf("wav: bits per sample must be 8 or 16 but was %d", bitsPerSample)
			}
			sampleRate = int(buf[4]) | int(buf[5])<<8 | int(buf[6])<<16 | int(buf[7])<<24
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

	var s io.ReadSeeker = newSectionReader(src, headerSize, dataSize)

	if mono || bitsPerSample != 16 {
		var format convert.Format
		switch bitsPerSample {
		case 8:
			format = convert.FormatU8
		case 16:
			format = convert.FormatS16
		default:
			// TODO: Support signed 24bit integer format (#2215).
			return nil, fmt.Errorf("wav: unsupported bits per sample: %d", bitsPerSample)
		}
		s = convert.NewStereoI16ReadSeeker(s, mono, format)
		if mono {
			dataSize *= 2
		}
		if bitsPerSample != 16 {
			dataSize *= 2
		}
	}

	if bitDepthInBytes == bitDepthInBytesFloat32 {
		s = convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(s)
		dataSize *= 2
	}

	return &Stream{
		inner:      s,
		size:       dataSize,
		sampleRate: sampleRate,
	}, nil
}

// Decode decodes WAV (RIFF) data to playable stream in signed 16bit integer, little endian, 2 channels (stereo) format.
//
// The format must be 1 or 2 channels, 8bit or 16bit little endian PCM.
// The format is converted into 2 channels and 16bit.
//
// Decode returns error when decoding fails or IO error happens.
//
// Decode automatically resamples the stream to fit with the audio context if necessary.
//
// The returned Stream's Seek is available only when src is an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
//
// Deprecated: as of v2.1. Use DecodeWithSampleRate instead.
func Decode(context *audio.Context, src io.Reader) (*Stream, error) {
	return DecodeWithSampleRate(context.SampleRate(), src)
}
