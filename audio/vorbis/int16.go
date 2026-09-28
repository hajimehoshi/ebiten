// Copyright 2019 The Ebiten Authors
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
	"io"
)

type float32Reader interface {
	Read([]float32) (int, error)
}

func newInt16BytesReaderFromFloat32Reader(r float32Reader, channels int) io.Reader {
	return &int16BytesReader{r: r, channels: channels}
}

type int16BytesReader struct {
	r        float32Reader
	channels int
	eof      bool
	fbuf     []float32
}

func (r *int16BytesReader) Read(buf []byte) (int, error) {
	channels := r.channels
	if r.eof && len(r.fbuf) < channels {
		return 0, io.EOF
	}
	if len(buf) == 0 {
		return 0, nil
	}
	// Buffer at least one frame to distinguish EOF from a short destination buffer.
	l := max(len(buf)/2/channels, 1) * channels
	var readErr error
	for len(r.fbuf) < l && !r.eof {
		origLen := len(r.fbuf)
		if cap(r.fbuf) < l {
			r.fbuf = append(r.fbuf, make([]float32, l-origLen)...)
		}
		n, err := r.r.Read(r.fbuf[origLen:l])
		r.fbuf = r.fbuf[:origLen+n]
		if err != nil && err != io.EOF {
			readErr = err
			break
		}
		if err == io.EOF {
			r.eof = true
		}
		if len(r.fbuf) >= channels || n == 0 {
			break
		}
	}
	if len(buf) < 2*channels {
		if readErr != nil {
			return 0, readErr
		}
		if r.eof && len(r.fbuf) < channels {
			return 0, io.EOF
		}
		return 0, io.ErrShortBuffer
	}
	n := min(len(r.fbuf)/channels, len(buf)/2/channels) * channels
	for i := range n {
		f := min(max(r.fbuf[i], -1), 1)
		s := int16(f * (1<<15 - 1))
		buf[2*i] = byte(s)
		buf[2*i+1] = byte(s >> 8)
	}
	copy(r.fbuf, r.fbuf[n:])
	r.fbuf = r.fbuf[:len(r.fbuf)-n]

	if readErr != nil {
		return n * 2, readErr
	}
	if r.eof {
		return n * 2, io.EOF
	}
	return n * 2, nil
}
