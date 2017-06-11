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
// This package is very experimental.
package mp3

import (
	"bytes"

	"github.com/hajimehoshi/ebiten/audio"
)

type Stream struct {
	inner *bytes.Reader
}

// Read is implementation of io.Reader's Read.
func (s *Stream) Read(p []byte) (int, error) {
	return s.inner.Read(p)
}

// Seek is implementation of io.Seeker's Seek.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.inner.Seek(offset, whence)
}

// Read is implementation of io.Closer's Close.
func (s *Stream) Close() error {
	return nil
}

// Size returns the size of decoded stream in bytes.
func (s *Stream) Size() int64 {
	return int64(s.inner.Len())
}

func Decode(context *audio.Context, src audio.ReadSeekCloser) (*Stream, error) {
	var buf bytes.Buffer
	if err := decode(src, &buf); err != nil {
		return nil, err
	}
	s := &Stream{
		inner: bytes.NewReader(buf.Bytes()),
	}
	// TODO: Resampling
	return s, nil
}
