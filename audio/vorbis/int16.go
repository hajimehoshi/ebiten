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
	if r.eof {
		return 0, io.EOF
	}
	if len(buf) == 0 {
		return 0, nil
	}
	// A buffer shorter than one frame cannot receive any converted data.
	if len(buf) < 2*r.channels {
		return 0, io.ErrShortBuffer
	}

	l := len(buf) / 2 / r.channels * r.channels
	if cap(r.fbuf) < l {
		r.fbuf = make([]float32, l)
	}

	n, err := r.r.Read(r.fbuf[:l])
	if err != nil && err != io.EOF {
		return 0, err
	}
	if err == io.EOF {
		r.eof = true
	}

	for i := range n {
		f := r.fbuf[i]
		s := int16(f * (1<<15 - 1))
		buf[2*i] = byte(s)
		buf[2*i+1] = byte(s >> 8)
	}

	return n * 2, err
}
