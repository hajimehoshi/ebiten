// Copyright 2015 Hajime Hoshi
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

// Package vorbis provides Ogg/Vorbis decoder.
package vorbis

import (
	"io"
	"runtime"

	"github.com/jfreymuth/oggvorbis"
)

type decoded struct {
	data       []float32
	posInBytes int
}

func (d *decoded) Read(b []byte) (int, error) {
	total := len(d.data) * 2
	l := total - d.posInBytes
	if l > len(b) {
		l = len(b)
	}
	// l must be even so that d.posInBytes is always even.
	l = l / 2 * 2
	for i := 0; i < l/2; i++ {
		f := d.data[d.posInBytes/2+i]
		s := int16(f * (1<<15 - 1))
		b[2*i] = uint8(s)
		b[2*i+1] = uint8(s >> 8)
	}
	d.posInBytes += l
	if d.posInBytes == total {
		return l, io.EOF
	}
	return l, nil
}

func (d *decoded) Seek(offset int64, whence int) (int64, error) {
	next := int64(0)
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = int64(d.posInBytes) + offset
	case io.SeekEnd:
		next = int64(len(d.data)*2) + offset
	}
	d.posInBytes = int(next)
	return next, nil
}

func (d *decoded) Close() error {
	runtime.SetFinalizer(d, nil)
	return nil
}

func (d *decoded) Size() int64 {
	return int64(len(d.data) * 2)
}

// decode accepts an ogg stream and returns a decorded stream.
func decode(in io.ReadCloser) (*decoded, int, int, error) {
	data, format, err := oggvorbis.ReadAll(in)
	if err != nil {
		return nil, 0, 0, err
	}
	if err := in.Close(); err != nil {
		return nil, 0, 0, err
	}
	d := &decoded{
		data:       data,
		posInBytes: 0,
	}
	runtime.SetFinalizer(d, (*decoded).Close)
	return d, format.Channels, format.SampleRate, nil
}
