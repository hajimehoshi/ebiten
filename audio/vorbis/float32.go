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

	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
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
	eof      bool
}

func (r *float32BytesReadSeeker) Read(buf []byte) (int, error) {
	channels := r.r.Channels()
	if len(buf) == 0 {
		return 0, nil
	}
	if r.eof && len(r.fbuf) < channels {
		return 0, io.EOF
	}
	// Buffer at least one frame to distinguish EOF from a short destination buffer.
	l := max(len(buf)/4/channels, 1) * channels
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
	if len(buf) < 4*channels {
		if readErr != nil {
			return 0, readErr
		}
		if r.eof && len(r.fbuf) < channels {
			return 0, io.EOF
		}
		return 0, io.ErrShortBuffer
	}
	n := min(len(r.fbuf)/channels, len(buf)/4/channels) * channels
	for i := range n {
		v := math.Float32bits(r.fbuf[i])
		buf[4*i] = byte(v)
		buf[4*i+1] = byte(v >> 8)
		buf[4*i+2] = byte(v >> 16)
		buf[4*i+3] = byte(v >> 24)
	}
	copy(r.fbuf, r.fbuf[n:])
	r.fbuf = r.fbuf[:len(r.fbuf)-n]
	r.pos += int64(n * 4)
	if readErr != nil {
		return n * 4, readErr
	}
	if r.eof {
		return n * 4, io.EOF
	}
	return n * 4, nil
}

func (r *float32BytesReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if !r.seekable {
		return 0, fmt.Errorf("vorbis: the source must be io.Seeker to seek: %w", errors.ErrUnsupported)
	}

	sampleSize := int64(r.r.Channels()) * 4

	var base int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = r.pos
	case io.SeekEnd:
		if r.r.Length() == 0 {
			return 0, fmt.Errorf("vorbis: seeking from the end is not possible when the length is unknown: %w", errors.ErrUnsupported)
		}
		base = r.r.Length() * sampleSize
	default:
		return 0, fmt.Errorf("vorbis: whence must be io.SeekStart, io.SeekCurrent, or io.SeekEnd but was %d", whence)
	}
	offset, ok := mathutil.AddForSeek(base, offset)
	if !ok {
		return 0, fmt.Errorf("vorbis: invalid seek position")
	}
	pos := offset / sampleSize * sampleSize
	if err := r.r.SetPosition(pos / sampleSize); err != nil {
		return 0, err
	}
	r.pos = pos
	r.fbuf = r.fbuf[:0]
	r.eof = false
	return r.pos, nil
}
