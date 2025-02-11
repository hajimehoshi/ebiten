// Copyright 2024 The Ebitengine Authors
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
	"io"
	"math"

	"github.com/jfreymuth/oggvorbis"
)

var _ io.ReadSeeker = (*float32BytesReadSeeker)(nil)

func newFloat32BytesReadSeeker(r *oggvorbis.Reader, seekable bool) *float32BytesReadSeeker {
	return &float32BytesReadSeeker{
		r:        r,
		seekable: seekable,
	}
}

type float32BytesReadSeeker struct {
	r        *oggvorbis.Reader
	seekable bool
	fbuf     []float32
	pos      int64
}

func (r *float32BytesReadSeeker) Read(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}

	l := max(len(buf)/4/r.r.Channels()*r.r.Channels(), 1)
	if cap(r.fbuf) < l {
		r.fbuf = make([]float32, l)
	}

	n, err := r.r.Read(r.fbuf[:l])
	if err != nil && err != io.EOF {
		return 0, err
	}

	for i := 0; i < n; i++ {
		v := math.Float32bits(r.fbuf[i])
		buf[4*i] = byte(v)
		buf[4*i+1] = byte(v >> 8)
		buf[4*i+2] = byte(v >> 16)
		buf[4*i+3] = byte(v >> 24)
	}

	r.pos += int64(n * 4)

	return n * 4, err
}

func (r *float32BytesReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if !r.seekable {
		return 0, fmt.Errorf("vorbis: the source must be io.Seeker but not: %w", errors.ErrUnsupported)
	}

	sampleSize := int64(r.r.Channels()) * 4
	offset = offset / sampleSize * sampleSize

	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += r.pos
	case io.SeekEnd:
		offset += r.r.Length()
	}
	r.pos = offset
	if err := r.r.SetPosition(r.pos / sampleSize); err != nil {
		return 0, err
	}
	return r.pos, nil
}
