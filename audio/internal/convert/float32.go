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

package convert

import (
	"io"
)

type Float32Reader interface {
	Read([]float32) (int, error)
}

func NewReaderFromFloat32Reader(r Float32Reader) io.Reader {
	return &f32Reader{r: r}
}

type f32Reader struct {
	r    Float32Reader
	eof  bool
	buf  *byte
	fbuf []float32
}

func max(a, b int) int {
	if a < b {
		return b
	}
	return a
}

func (f *f32Reader) Read(buf []byte) (int, error) {
	if f.eof {
		return 0, io.EOF
	}
	if len(buf) == 0 {
		return 0, nil
	}
	if f.buf != nil {
		buf[0] = *f.buf
		f.buf = nil
		return 1, nil
	}

	l := max(len(buf)/2, 1)
	ll := len(f.fbuf)
	if ll < 16 {
		ll = 16
	}
	for ll < l {
		ll *= 2
	}
	if len(f.fbuf) < ll {
		f.fbuf = make([]float32, ll)
	}

	n, err := f.r.Read(f.fbuf[:l])
	if err != nil && err != io.EOF {
		return 0, err
	}
	if err == io.EOF {
		f.eof = true
	}

	b := buf
	if len(buf) == 1 && n > 0 {
		b = make([]byte, 2)
	}
	for i := 0; i < n; i++ {
		f := f.fbuf[i]
		s := int16(f * (1<<15 - 1))
		b[2*i] = uint8(s)
		b[2*i+1] = uint8(s >> 8)
	}

	if len(buf) == 1 && len(b) == 2 {
		buf[0] = b[0]
		f.buf = &b[1]
		return 1, err
	}
	return n * 2, err
}
