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

package vorbis

import (
	"errors"
	"fmt"

	"github.com/hajimehoshi/ebiten/audio"
)

// Stream is a decoded audio stream.
type Stream struct {
	decoded *decoded
}

// Read is implementation of io.Reader's Read.
func (s *Stream) Read(p []byte) (int, error) {
	return s.decoded.Read(p)
}

// Seek is implementation of io.Seeker's Seek.
//
// Note that Seek can take long since decoding is a relatively heavy task.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.decoded.Seek(offset, whence)
}

// Read is implementation of io.Closer's Close.
func (s *Stream) Close() error {
	return s.decoded.Close()
}

// Size returns the size of decoded stream in bytes.
func (s *Stream) Size() int64 {
	return s.decoded.Size()
}

// Decode decodes Ogg/Vorbis data to playable stream.
//
// The sample rate must be same as that of audio context.
//
// This function returns error on Safari.
func Decode(context *audio.Context, src audio.ReadSeekCloser) (*Stream, error) {
	decoded, channelNum, sampleRate, err := decode(src)
	if err != nil {
		return nil, err
	}
	// TODO: Remove this magic number
	if channelNum != 2 {
		return nil, errors.New("vorbis: number of channels must be 2")
	}
	if sampleRate != context.SampleRate() {
		return nil, fmt.Errorf("vorbis: sample rate must be %d but %d", context.SampleRate(), sampleRate)
	}
	s := &Stream{
		decoded: decoded,
	}
	return s, nil
}
