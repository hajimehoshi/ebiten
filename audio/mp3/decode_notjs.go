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

// +build !js

// Package mp3 provides MP3 decoder.
package mp3

import (
	"github.com/hajimehoshi/go-mp3"

	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/internal/convert"
)

type Stream struct {
	inner audio.ReadSeekCloser
	size  int64
}

// Read is implementation of io.Reader's Read.
func (s *Stream) Read(buf []byte) (int, error) {
	return s.inner.Read(buf)
}

// Seek is implementation of io.Seeker's Seek.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.inner.Seek(offset, whence)
}

// Read is implementation of io.Closer's Close.
func (s *Stream) Close() error {
	return s.inner.Close()
}

// Size returns the size of decoded stream in bytes.
func (s *Stream) Size() int64 {
	return s.size
}

func Decode(context *audio.Context, src audio.ReadSeekCloser) (*Stream, error) {
	d, err := mp3.Decode(src)
	if err != nil {
		return nil, err
	}
	// TODO: Resampling
	var s audio.ReadSeekCloser = d
	size := d.Length()
	if d.SampleRate() != context.SampleRate() {
		r := convert.NewResampling(s, d.Length(), d.SampleRate(), context.SampleRate())
		s = r
		size = r.Size()
	}
	return &Stream{
		inner: s,
		size:  size,
	}, nil
}
