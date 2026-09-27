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

// Package mp3 provides an MP3 decoder.
//
// On desktops and mobiles, a pure Go decoder is used.
// On browsers, a native decoder on the browser is used.
package mp3

import (
	"errors"
	"fmt"
	"io"

	"github.com/hajimehoshi/go-mp3"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
)

const (
	channelCount           = 2
	bitDepthInBytesInt16   = 2
	bitDepthInBytesFloat32 = 4
)

// Stream is a decoded stream.
type Stream struct {
	readSeeker io.ReadSeeker
	length     int64
	sampleRate int
	seekable   bool

	// atEnd reports whether the stream has been sought to or past its end.
	atEnd bool

	// endPos is the position of the stream while atEnd is true.
	endPos int64

	// bytesPerSample is the size in bytes of one sample across all the channels.
	bytesPerSample int64
}

// Read is an implementation of io.Reader's Read.
//
// Read returns only whole samples (4 bytes each in the 16-bit integer format, 8 bytes each in the 32-bit float format).
// For a non-empty buffer shorter than one sample, Read returns [io.ErrShortBuffer], or [io.EOF] once the stream has ended.
func (s *Stream) Read(buf []byte) (int, error) {
	if s.atEnd {
		return 0, io.EOF
	}
	return s.readSeeker.Read(buf)
}

// Seek is an implementation of io.Seeker's Seek.
//
// Seek returns an error wrapping [errors.ErrUnsupported] when the source is not an io.Seeker.
//
// The returned position can differ from the requested one with a nil error: a position in the middle of a sample is
// rounded down to a sample boundary.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	if !s.seekable {
		return 0, fmt.Errorf("mp3: the source must be io.Seeker to seek: %w", errors.ErrUnsupported)
	}

	// A query for the current position does not seek the decoder. A seek restarts decoding from the
	// previous frame, which can change the data that follows.
	if offset == 0 && whence == io.SeekCurrent {
		if s.atEnd {
			return s.endPos, nil
		}
		return s.readSeeker.Seek(0, io.SeekCurrent)
	}

	// Resolve the position here: the underlying decoder panics for a negative position.
	var base int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = s.endPos
		if !s.atEnd {
			cur, err := s.readSeeker.Seek(0, io.SeekCurrent)
			if err != nil {
				return 0, err
			}
			base = cur
		}
	case io.SeekEnd:
		base = s.length
	default:
		return 0, fmt.Errorf("mp3: whence must be io.SeekStart, io.SeekCurrent, or io.SeekEnd but was %d", whence)
	}
	pos, ok := mathutil.AddForSeek(base, offset)
	if !ok {
		return 0, fmt.Errorf("mp3: invalid seek offset %d for position %d", offset, base)
	}
	// A position in the middle of a sample is rounded down to a sample boundary, as reading from there
	// would return bytes straddling two samples.
	pos = pos / s.bytesPerSample * s.bytesPerSample
	// The underlying decoder fails to seek to its end and panics past it, so a position there is not
	// delegated (#3619).
	if s.length >= 0 && pos >= s.length {
		s.atEnd = true
		s.endPos = pos
		return pos, nil
	}
	n, err := s.readSeeker.Seek(pos, io.SeekStart)
	if err != nil {
		return 0, err
	}
	s.atEnd = false
	return n, nil
}

// Length returns the size of decoded stream in bytes.
//
// Length returns -1 when the source is not an io.Seeker.
func (s *Stream) Length() int64 {
	if !s.seekable {
		return -1
	}
	return s.length
}

// SampleRate returns the sample rate of the decoded stream.
func (s *Stream) SampleRate() int {
	return s.sampleRate
}

// DecodeF32 decodes an MP3 source and returns a decoded stream in 32bit float, little endian, 2 channels (stereo) format.
//
// DecodeF32 returns error when decoding fails or IO error happens.
//
// The returned Stream's Seek is available only when src is an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
func DecodeF32(src io.Reader) (*Stream, error) {
	d, err := mp3.NewDecoder(src)
	if err != nil {
		return nil, err
	}
	_, seekable := src.(io.Seeker)
	r := convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(d)
	s := &Stream{
		readSeeker:     r,
		length:         d.Length() / bitDepthInBytesInt16 * bitDepthInBytesFloat32,
		sampleRate:     d.SampleRate(),
		seekable:       seekable,
		bytesPerSample: channelCount * bitDepthInBytesFloat32,
	}
	return s, nil
}

// DecodeWithoutResampling decodes an MP3 source and returns a decoded stream in signed 16bit integer, little endian, 2 channels (stereo) format.
//
// DecodeWithoutResampling returns error when decoding fails or IO error happens.
//
// The returned Stream's Seek is available only when src is an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
func DecodeWithoutResampling(src io.Reader) (*Stream, error) {
	d, err := mp3.NewDecoder(src)
	if err != nil {
		return nil, err
	}
	_, seekable := src.(io.Seeker)
	s := &Stream{
		readSeeker:     newInt16ReadSeeker(d),
		length:         d.Length(),
		sampleRate:     d.SampleRate(),
		seekable:       seekable,
		bytesPerSample: channelCount * bitDepthInBytesInt16,
	}
	return s, nil
}

// newInt16ReadSeeker returns a stream of d whose reads return whole samples.
func newInt16ReadSeeker(d *mp3.Decoder) io.ReadSeeker {
	return convert.NewStereoI16ReadSeeker(d, false, convert.FormatS16)
}

// DecodeWithSampleRate decodes an MP3 source and returns a decoded stream in signed 16bit integer, little endian, 2 channels (stereo) format.
//
// DecodeWithSampleRate returns error when decoding fails or IO error happens, or when sampleRate is not positive.
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
	if sampleRate <= 0 {
		return nil, fmt.Errorf("mp3: sample rate must be positive but was %d", sampleRate)
	}
	d, err := mp3.NewDecoder(src)
	if err != nil {
		return nil, err
	}
	_, seekable := src.(io.Seeker)

	r := newInt16ReadSeeker(d)
	length := d.Length()
	if d.SampleRate() != sampleRate {
		r2 := convert.NewResampling(d, d.Length(), d.SampleRate(), sampleRate, bitDepthInBytesInt16)
		r = r2
		length = r2.Length()
	}
	s := &Stream{
		readSeeker:     r,
		length:         length,
		sampleRate:     sampleRate,
		seekable:       seekable,
		bytesPerSample: channelCount * bitDepthInBytesInt16,
	}
	return s, nil
}

// Decode decodes MP3 source and returns a decoded stream in signed 16bit integer, little endian, 2 channels (stereo) format.
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
