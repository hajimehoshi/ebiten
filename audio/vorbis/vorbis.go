// Copyright 2015 Hajime Hoshi
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

// Package vorbis provides Ogg/Vorbis decoder.
package vorbis

import (
	"errors"
	"fmt"
	"io"

	"github.com/jfreymuth/oggvorbis"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
)

const (
	bitDepthInBytesInt16   = 2
	bitDepthInBytesFloat32 = 4
)

// Stream is a decoded audio stream.
type Stream struct {
	readSeeker io.ReadSeeker
	length     int64
	sampleRate int
}

// Read is implementation of io.Reader's Read.
func (s *Stream) Read(p []byte) (int, error) {
	return s.readSeeker.Read(p)
}

// Seek is implementation of io.Seeker's Seek.
//
// Note that Seek can take long since decoding is a relatively heavy task.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.readSeeker.Seek(offset, whence)
}

// Length returns the size of decoded stream in bytes.
//
// If the source is not io.Seeker, Length returns 0.
func (s *Stream) Length() int64 {
	return s.length
}

// SampleRate returns the sample rate of the decoded stream.
func (s *Stream) SampleRate() int {
	return s.sampleRate
}

// DecodeF32 decodes Ogg/Vorbis data to playable stream in 32bit float, little endian, 2 channels (stereo) format.
//
// DecodeF32 returns error when decoding fails or IO error happens.
//
// The returned Stream's Seek is available only when src is an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
func DecodeF32(src io.Reader) (*Stream, error) {
	r, err := oggvorbis.NewReader(src)
	if err != nil {
		return nil, err
	}
	if r.Channels() != 1 && r.Channels() != 2 {
		return nil, fmt.Errorf("vorbis: number of channels must be 1 or 2 but was %d", r.Channels())
	}

	_, seekable := src.(io.Seeker)
	var s io.ReadSeeker = newFloat32BytesReadSeeker(r, seekable)
	length := r.Length() * int64(r.Channels()) * bitDepthInBytesFloat32
	if r.Channels() == 1 {
		s = convert.NewStereoF32(s, true)
		length *= 2
	}

	stream := &Stream{
		readSeeker: s,
		length:     length,
		sampleRate: r.SampleRate(),
	}
	// Read some data for performance (#297).
	if seekable {
		if _, err := stream.Read(make([]byte, 65536)); err != nil && err != io.EOF {
			return nil, err
		}
		if _, err := stream.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}
	return stream, nil
}

type i16Stream struct {
	posInBytes   int64
	seekable     bool
	vorbisReader *oggvorbis.Reader
	i16Reader    io.Reader
}

func (s *i16Stream) Read(b []byte) (int, error) {
	if s.i16Reader == nil {
		s.i16Reader = newInt16BytesReaderFromFloat32Reader(s.vorbisReader)
	}

	l := int64(len(b))
retry:
	n, err := s.i16Reader.Read(b[:l])
	if err != nil && err != io.EOF {
		return 0, err
	}
	if n == 0 && l > 0 && err != io.EOF {
		// When l is too small, decoder's Read might return 0 for a while. Let's retry.
		goto retry
	}
	s.posInBytes += int64(n)
	return n, err
}

func (s *i16Stream) Seek(offset int64, whence int) (int64, error) {
	if !s.seekable {
		return 0, fmt.Errorf("vorbis: the source must be io.Seeker but not: %w", errors.ErrUnsupported)
	}

	next := int64(0)
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = int64(s.posInBytes) + offset
	case io.SeekEnd:
		next = int64(s.totalBytes()) + offset
	}
	sampleSize := int64(s.vorbisReader.Channels()) * bitDepthInBytesInt16
	s.posInBytes = next / sampleSize * sampleSize
	if err := s.vorbisReader.SetPosition(next / sampleSize); err != nil {
		return 0, err
	}
	s.i16Reader = nil
	return next, nil
}

func (s *i16Stream) totalBytes() int64 {
	// TODO: r.Length() returns 0 when the format is unknown.
	// Should we check that?
	return s.vorbisReader.Length() * int64(s.vorbisReader.Channels()) * bitDepthInBytesInt16
}

// decodeI16 accepts an ogg stream and returns a decorded stream.
func decodeI16(in io.Reader) (*i16Stream, error) {
	r, err := oggvorbis.NewReader(in)
	if err != nil {
		return nil, err
	}
	if r.Channels() != 1 && r.Channels() != 2 {
		return nil, fmt.Errorf("vorbis: number of channels must be 1 or 2 but was %d", r.Channels())
	}

	_, seekable := in.(io.Seeker)
	s := &i16Stream{
		seekable:     seekable,
		posInBytes:   0,
		vorbisReader: r,
	}
	// Read some data for performance (#297).
	if seekable {
		if _, err := s.Read(make([]byte, 65536)); err != nil && err != io.EOF {
			return nil, err
		}
		if _, err := s.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// DecodeWithoutResampling decodes Ogg/Vorbis data to playable stream in signed 16bit integer, little endian, 2 channels (stereo) format.
//
// DecodeWithoutResampling returns error when decoding fails or IO error happens.
//
// The returned Stream's Seek is available only when src is an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
func DecodeWithoutResampling(src io.Reader) (*Stream, error) {
	i16Stream, err := decodeI16(src)
	if err != nil {
		return nil, err
	}

	var s io.ReadSeeker = i16Stream
	length := i16Stream.totalBytes()
	if i16Stream.vorbisReader.Channels() == 1 {
		s = convert.NewStereoI16ReadSeeker(s, true, convert.FormatS16)
		length *= 2
	}

	stream := &Stream{
		readSeeker: s,
		length:     length,
		sampleRate: i16Stream.vorbisReader.SampleRate(),
	}
	return stream, nil
}

// DecodeWithSampleRate decodes Ogg/Vorbis data to playable stream in signed 16bit integer, little endian, 2 channels (stereo) format.
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
	i16Stream, err := decodeI16(src)
	if err != nil {
		return nil, err
	}

	var s io.ReadSeeker = i16Stream
	length := i16Stream.totalBytes()
	if i16Stream.vorbisReader.Channels() == 1 {
		s = convert.NewStereoI16ReadSeeker(s, true, convert.FormatS16)
		length *= 2
	}
	if i16Stream.vorbisReader.SampleRate() != sampleRate {
		r := convert.NewResampling(s, length, i16Stream.vorbisReader.SampleRate(), sampleRate, bitDepthInBytesInt16)
		s = r
		length = r.Length()
	}
	stream := &Stream{
		readSeeker: s,
		length:     length,
		sampleRate: sampleRate,
	}
	return stream, nil
}

// Decode decodes Ogg/Vorbis data to playable stream in signed 16bit integer, little endian, 2 channels (stereo) format.
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
