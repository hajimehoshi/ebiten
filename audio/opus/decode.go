// Copyright 2026 The Ebitengine Authors
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

// Package opus provides Opus decoder.
package opus

import (
	"errors"
	"fmt"
	"io"

	"github.com/kazzmir/opus-go"
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
	// Resolve the position here: the underlying decoder clamps a negative position to 0
	// instead of reporting an error.
	var pos int64
	switch whence {
	case io.SeekStart:
		pos = offset
	case io.SeekCurrent:
		cur, err := s.readSeeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		pos = cur + offset
	case io.SeekEnd:
		if s.length < 0 {
			return 0, fmt.Errorf("opus: seeking from the end is not possible when the length is unknown: %w", errors.ErrUnsupported)
		}
		pos = s.length + offset
	default:
		return 0, fmt.Errorf("opus: whence must be io.SeekStart, io.SeekCurrent, or io.SeekEnd but was %d", whence)
	}
	if pos < 0 {
		return 0, fmt.Errorf("opus: position must be >= 0 but was %d", pos)
	}
	return s.readSeeker.Seek(pos, io.SeekStart)
}

// Length returns the size of decoded stream in bytes.
//
// Length returns -1 if the source did not implement io.Seeker.
func (s *Stream) Length() int64 {
	return s.length
}

// SampleRate returns the sample rate of the decoded stream.
func (s *Stream) SampleRate() int {
	return s.sampleRate
}

// DecodeF32 decodes an Opus source and returns a decoded stream in 32bit float, little endian, 2 channels (stereo) format.
//
// DecodeF32 returns error when decoding fails or IO error happens.
//
// The returned Stream's Seek is available only when src is an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
func DecodeF32(src io.Reader) (*Stream, error) {
	d, err := opusgo.NewPlayerF32FromReader(src)
	if err != nil {
		return nil, err
	}

	var length int64 = -1
	if _, ok := src.(io.Seeker); ok {
		length = d.Length()
		if _, err := d.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}

	s := &Stream{
		readSeeker: d,
		length:     length,
		sampleRate: d.SampleRate(),
	}
	return s, nil
}

// Decode decodes an Opus source and returns a decoded stream in signed 16bit integer, little endian, 2 channels (stereo) format.
//
// Decode returns error when decoding fails or IO error happens.
//
// The returned Stream's Seek is available only when src is an io.Seeker.
//
// A Stream doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
func Decode(src io.Reader) (*Stream, error) {
	d, err := opusgo.NewPlayerFromReader(src)
	if err != nil {
		return nil, err
	}

	var length int64 = -1
	if _, ok := src.(io.Seeker); ok {
		length = d.Length()
		if _, err := d.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}

	s := &Stream{
		readSeeker: d,
		length:     length,
		sampleRate: d.SampleRate(),
	}
	return s, nil
}
