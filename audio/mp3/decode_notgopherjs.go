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

// +build !js wasm

// Package mp3 provides MP3 decoder.
//
// On desktops and mobiles, a pure Go decoder is used.
// On browsers, a native decoder on the browser is used.
package mp3

import (
	"io"
	"runtime"

	"github.com/hajimehoshi/go-mp3"

	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/internal/convert"
)

// Stream is a decoded stream.
type Stream struct {
	orig       *mp3.Decoder
	resampling *convert.Resampling
	toClose    io.Closer
}

// Read is implementation of io.Reader's Read.
func (s *Stream) Read(buf []byte) (int, error) {
	if s.resampling != nil {
		return s.resampling.Read(buf)
	}
	return s.orig.Read(buf)
}

// Seek is implementation of io.Seeker's Seek.
//
// If the underlying source is not io.Seeker, Seek panics.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	if s.resampling != nil {
		return s.resampling.Seek(offset, whence)
	}
	return s.orig.Seek(offset, whence)
}

// Close is implementation of io.Closer's Close.
func (s *Stream) Close() error {
	runtime.SetFinalizer(s, nil)
	return s.toClose.Close()
}

// Length returns the size of decoded stream in bytes.
func (s *Stream) Length() int64 {
	if s.resampling != nil {
		return s.resampling.Length()
	}
	return s.orig.Length()
}

// Size returns the size of decoded stream in bytes.
//
// Deprecated: (as of 1.6.0) Use Length instead.
func (s *Stream) Size() int64 {
	return s.Length()
}

// Decode decodes MP3 source and returns a decoded stream.
//
// Decode returns error when decoding fails or IO error happens.
//
// Decode automatically resamples the stream to fit with the audio context if necessary.
//
// Decode takes the ownership of src, and Stream's Close function closes src.
func Decode(context *audio.Context, src io.ReadCloser) (*Stream, error) {
	d, err := mp3.NewDecoder(src)
	if err != nil {
		return nil, err
	}

	var r *convert.Resampling
	if d.SampleRate() != context.SampleRate() {
		r = convert.NewResampling(d, d.Length(), d.SampleRate(), context.SampleRate())
	}
	s := &Stream{
		orig:       d,
		resampling: r,
		toClose:    src,
	}
	runtime.SetFinalizer(s, (*Stream).Close)
	return s, nil
}
