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

func newInt16BytesReaderFromFloat32Reader(r float32Reader) io.Reader {
	return &int16BytesReader{r: r}
}

type int16BytesReader struct {
	r         float32Reader
	eof       bool
	hasRemain bool
	remain    byte
	fbuf      []float32
}

func (r *int16BytesReader) Read(buf []byte) (int, error) {
	if r.eof {
		return 0, io.EOF
	}
	if len(buf) == 0 {
		return 0, nil
	}
	if r.hasRemain {
		buf[0] = r.remain
		r.hasRemain = false
		return 1, nil
	}

	l := max(len(buf)/2, 1)
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

	b := buf
	if len(buf) == 1 && n > 0 {
		b = make([]byte, 2)
	}
	for i := 0; i < n; i++ {
		f := r.fbuf[i]
		s := int16(f * (1<<15 - 1))
		b[2*i] = byte(s)
		b[2*i+1] = byte(s >> 8)
	}

	if len(buf) == 1 && len(b) == 2 {
		buf[0] = b[0]
		r.remain = b[1]
		r.hasRemain = true
		return 1, err
	}
	return n * 2, err
}
