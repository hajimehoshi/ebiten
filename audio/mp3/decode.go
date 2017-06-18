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
	"io"

	"github.com/hajimehoshi/ebiten/audio"
)

type Stream struct {
	inner *Decoder
	data  []uint8
	pos   int64
	eof   bool
}

// Read is implementation of io.Reader's Read.
func (s *Stream) Read(buf []byte) (int, error) {
	for int64(len(s.data)) <= s.pos && !s.eof {
		buf := make([]uint8, 4096)
		n, err := s.inner.Read(buf)
		s.data = append(s.data, buf[:n]...)
		if err != nil {
			if err == io.EOF {
				s.eof = true
				break
			}
			return 0, err
		}
	}
	if int64(len(s.data)) <= s.pos && s.eof {
		return 0, io.EOF
	}
	n := copy(buf, s.data[s.pos:])
	s.pos += int64(n)
	return n, nil
}

// Seek is implementation of io.Seeker's Seek.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		s.pos = offset
	case io.SeekCurrent:
		s.pos += offset
	case io.SeekEnd:
		panic("not implemented")
	}
	return s.pos, nil
}

// Read is implementation of io.Closer's Close.
func (s *Stream) Close() error {
	return nil
}

// Size returns the size of decoded stream in bytes.
func (s *Stream) Size() int64 {
	return int64(s.inner.Length())
}

func Decode(context *audio.Context, src audio.ReadSeekCloser) (*Stream, error) {
	d, err := decode(src)
	if err != nil {
		return nil, err
	}
	s := &Stream{
		inner: d,
	}
	// TODO: Resampling
	return s, nil
}
