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

// Package mp3 provides MP3 decoder.
//
// On desktops and mobiles, a pure Go decoder is used.
// On browsers, a native decoder on the browser is used.
package mp3

import (
	"io"

	"github.com/hajimehoshi/go-mp3"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
)

const (
	bitDepthInBytesInt16   = 2
	bitDepthInBytesFloat32 = 4
)

// Stream is a decoded stream.
type Stream struct {
	readSeeker io.ReadSeeker
	length     int64
	sampleRate int
}

// Read is implementation of io.Reader's Read.
func (s *Stream) Read(buf []byte) (int, error) {
	return s.readSeeker.Read(buf)
}

// Seek is implementation of io.Seeker's Seek.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.readSeeker.Seek(offset, whence)
}

// Length returns the size of decoded stream in bytes.
func (s *Stream) Length() int64 {
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
	r := convert.NewFloat32BytesReadSeekerFromInt16BytesReadSeeker(d)
	s := &Stream{
		readSeeker: r,
		length:     d.Length() / bitDepthInBytesInt16 * bitDepthInBytesFloat32,
		sampleRate: d.SampleRate(),
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
	s := &Stream{
		readSeeker: d,
		length:     d.Length(),
		sampleRate: d.SampleRate(),
	}
	return s, nil
}

// DecodeWithSampleRate decodes an MP3 source and returns a decoded stream in signed 16bit integer, little endian, 2 channels (stereo) format.
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
	d, err := mp3.NewDecoder(src)
	if err != nil {
		return nil, err
	}

	var r io.ReadSeeker = d
	length := d.Length()
	if d.SampleRate() != sampleRate {
		r2 := convert.NewResampling(d, d.Length(), d.SampleRate(), sampleRate, bitDepthInBytesInt16)
		r = r2
		length = r2.Length()
	}
	s := &Stream{
		readSeeker: r,
		length:     length,
		sampleRate: sampleRate,
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
