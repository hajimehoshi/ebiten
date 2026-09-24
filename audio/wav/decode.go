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

// Package wav provides a WAV (RIFF) decoder.
package wav

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
)

const (
	channelCount           = 2
	bitDepthInBytesInt16   = 2
	bitDepthInBytesFloat32 = 4
)

// Stream is a decoded audio stream.
//
// The format is signed 16bit integer little endian PCM (DecodeWithoutResampling, etc.),
// or 32bit float little endian PCM (DecodeF32).
// The channel count is 2.
type Stream struct {
	inner      io.ReadSeeker
	size       int64
	sampleRate int

	// bytesPerSample is the size in bytes of one sample across all the channels.
	bytesPerSample int64
}

// Read is an implementation of io.Reader's Read.
func (s *Stream) Read(p []byte) (int, error) {
	return s.inner.Read(p)
}

// Seek is an implementation of io.Seeker's Seek.
//
// If the underlying source is not an io.Seeker, Seek returns an error.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	// A query for the current position must not move the stream, even when a read has left it in the
	// middle of a sample.
	if offset == 0 && whence == io.SeekCurrent {
		return s.inner.Seek(offset, whence)
	}

	var base int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		cur, err := s.inner.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		base = cur
	case io.SeekEnd:
		// The end is unknown, so the underlying reader reports the error.
		if s.size < 0 {
			return s.inner.Seek(offset, whence)
		}
		base = s.size
	default:
		return 0, fmt.Errorf("wav: whence must be io.SeekStart, io.SeekCurrent, or io.SeekEnd but was %d", whence)
	}
	pos, ok := mathutil.AddForSeek(base, offset)
	if !ok {
		return 0, fmt.Errorf("wav: invalid seek offset %d for position %d", offset, base)
	}
	// A position in the middle of a sample is rounded down to a sample boundary, as reading from there
	// would return bytes straddling two samples.
	pos = pos / s.bytesPerSample * s.bytesPerSample
	return s.inner.Seek(pos, io.SeekStart)
}

// Length returns the size of decoded stream in bytes.
//
// Length returns -1 when the size is unknown. The size is unknown when the 'data' chunk does not
// declare its size and the source cannot seek to its end.
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
// The src format is converted into 2 channels and 32bit float.
//
// DecodeF32 returns error when decoding fails or IO error happens.
//
// The returned Stream's Seek returns an error when src is not an io.Seeker.
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
// DecodeWithoutResampling returns error when decoding fails or IO error happens.
//
// The returned Stream's Seek returns an error when src is not an io.Seeker.
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
// DecodeWithSampleRate returns error when decoding fails or IO error happens, or when sampleRate is not positive.
//
// DecodeWithSampleRate automatically resamples the stream to fit with sampleRate if necessary.
//
// The returned Stream's Seek returns an error when src is not an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
//
// Resampling can be a very heavy task. Stream has a cache for resampling, but the size is limited.
// Do not expect that Stream has a resampling cache even after whole data is played.
func DecodeWithSampleRate(sampleRate int, src io.Reader) (*Stream, error) {
	if sampleRate <= 0 {
		return nil, fmt.Errorf("wav: sample rate must be positive but was %d", sampleRate)
	}
	s, err := decode(src, bitDepthInBytesInt16)
	if err != nil {
		return nil, err
	}

	if sampleRate == s.sampleRate {
		return s, nil
	}

	r := convert.NewResampling(s.inner, s.size, s.sampleRate, sampleRate, bitDepthInBytesInt16)
	return &Stream{
		inner:          r,
		size:           r.Length(),
		sampleRate:     sampleRate,
		bytesPerSample: channelCount * bitDepthInBytesInt16,
	}, nil
}

// readHeaderPart fills buf from src. A source that ends before buf is filled is reported as an invalid
// header, and any other failure of the source is returned wrapped. what names the part for the error.
func readHeaderPart(src io.Reader, buf []byte, what string) error {
	if _, err := io.ReadFull(src, buf); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return fmt.Errorf("wav: invalid header: %s too short", what)
		}
		return fmt.Errorf("wav: failed to read %s: %w", what, err)
	}
	return nil
}

func decode(src io.Reader, bitDepthInBytes int) (*Stream, error) {
	buf := make([]byte, 12)
	if err := readHeaderPart(src, buf, "header"); err != nil {
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
		if err := readHeaderPart(src, buf[:], "chunk header"); err != nil {
			return nil, err
		}
		headerSize += 8
		size := int64(buf[4]) | int64(buf[5])<<8 | int64(buf[6])<<16 | int64(buf[7])<<24
		// A chunk is word-aligned: an odd-sized chunk is followed by a pad byte, which is not counted in the size.
		paddedSize := size + size%2
		switch {
		case bytes.Equal(buf[0:4], []byte("fmt ")):
			// Size of 'fmt' header is usually 16, but can be more than 16.
			if size < 16 {
				return nil, fmt.Errorf("wav: invalid header: maybe non-PCM file?")
			}
			var fmtBuf [16]byte
			if err := readHeaderPart(src, fmtBuf[:], "'fmt ' chunk"); err != nil {
				return nil, err
			}
			format := int(fmtBuf[0]) | int(fmtBuf[1])<<8
			if format != 1 {
				return nil, fmt.Errorf("wav: format must be linear PCM")
			}
			channelCount := int(fmtBuf[2]) | int(fmtBuf[3])<<8
			switch channelCount {
			case 1:
				mono = true
			case 2:
				mono = false
			default:
				return nil, fmt.Errorf("wav: number of channels must be 1 or 2 but was %d", channelCount)
			}
			bitsPerSample = int(fmtBuf[14]) | int(fmtBuf[15])<<8
			// TODO: Support signed 24bit integer format (#2215).
			if bitsPerSample != 8 && bitsPerSample != 16 {
				return nil, fmt.Errorf("wav: bits per sample must be 8 or 16 but was %d", bitsPerSample)
			}
			sampleRate = int(fmtBuf[4]) | int(fmtBuf[5])<<8 | int(fmtBuf[6])<<16 | int(fmtBuf[7])<<24
			if sampleRate <= 0 {
				return nil, fmt.Errorf("wav: sample rate must be positive but was %d", sampleRate)
			}
			if _, err := io.CopyN(io.Discard, src, paddedSize-16); err != nil {
				return nil, fmt.Errorf("wav: invalid header: failed to skip 'fmt ' chunk: %w", err)
			}
			headerSize += paddedSize
		case bytes.Equal(buf[0:4], []byte("data")):
			dataSize = size
			break chunks
		default:
			if _, err := io.CopyN(io.Discard, src, paddedSize); err != nil {
				return nil, fmt.Errorf("wav: invalid header: failed to skip chunk: %w", err)
			}
			headerSize += paddedSize
		}
	}

	if bitsPerSample == 0 {
		return nil, fmt.Errorf("wav: invalid header: 'fmt ' not found before 'data'")
	}

	// A 'data' chunk size of 0 or 0xffffffff is a placeholder, and the data then extends to the end
	// of src. A writer that streams the data or is terminated abnormally never patches the size and
	// leaves the initial 0 (e.g. https://sourceforge.net/p/flac/bugs/190/) or a -1. RF64 also sets the
	// 32-bit size fields to -1 (0xffffffff) to indicate that the actual sizes are in the 'ds64'
	// chunk (https://en.wikipedia.org/wiki/RF64).
	if dataSize == 0 || dataSize == 0xffffffff {
		size, err := sizeToEnd(src)
		if err != nil {
			return nil, err
		}
		dataSize = size
	}

	bytesPerFrame := int64(bitsPerSample / 8)
	if !mono {
		bytesPerFrame *= 2
	}

	var s io.ReadSeeker
	if dataSize < 0 {
		s = newFrameAlignedReader(src, int(bytesPerFrame))
	} else {
		// A partial frame at the tail of the data chunk cannot be decoded. Discard it.
		dataSize = dataSize / bytesPerFrame * bytesPerFrame
		s = newSectionReader(src, headerSize, dataSize)
	}

	// sizeScale is the ratio of the decoded size to the 'data' chunk size.
	sizeScale := int64(1)
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
			sizeScale *= 2
		}
		if bitsPerSample != 16 {
			sizeScale *= 2
		}
	}

	if bitDepthInBytes == bitDepthInBytesFloat32 {
		s = convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(s)
		sizeScale *= 2
	}

	// An unknown size stays -1 (see Stream.Length).
	size := int64(-1)
	if dataSize >= 0 {
		size = dataSize * sizeScale
	}
	return &Stream{
		inner:          s,
		size:           size,
		sampleRate:     sampleRate,
		bytesPerSample: int64(channelCount * bitDepthInBytes),
	}, nil
}

// sizeToEnd returns the number of bytes from the current position of src to its end, or -1 when
// src cannot tell, e.g. src is not an io.Seeker or is a pipe. On success, src stays at its position.
func sizeToEnd(src io.Reader) (int64, error) {
	seeker, ok := src.(io.Seeker)
	if !ok {
		return -1, nil
	}
	cur, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil {
		return -1, nil
	}
	end, err := seeker.Seek(0, io.SeekEnd)
	if err != nil {
		return -1, nil
	}
	if _, err := seeker.Seek(cur, io.SeekStart); err != nil {
		return 0, err
	}
	return end - cur, nil
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
// The returned Stream's Seek returns an error when src is not an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
//
// Deprecated: as of v2.1. Use DecodeWithSampleRate instead.
func Decode(context *audio.Context, src io.Reader) (*Stream, error) {
	return DecodeWithSampleRate(context.SampleRate(), src)
}
