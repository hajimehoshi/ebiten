// Copyright 2016 Hajime Hoshi
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

package vorbis

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/hajimehoshi/ebiten/audio"
	"github.com/jfreymuth/go-vorbis/ogg/vorbis"
)

type Stream struct {
	buf *bytes.Reader
}

func newStream(v *vorbis.Vorbis) (*Stream, error) {
	data := []byte{}
	// TODO: We can delay decoding.
	for {
		out, err := v.DecodePacket()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		const channelNum = 2
		b := make([]byte, len(out[0])*2*channelNum)
		for i := 0; i < len(out[0]); i++ {
			vv := int16(out[0][i] * math.MaxInt16)
			b[4*i] = byte(vv)
			b[4*i+1] = byte(vv >> 8)
			vv = int16(out[1][i] * math.MaxInt16)
			b[4*i+2] = byte(vv)
			b[4*i+3] = byte(vv >> 8)
		}
		data = append(data, b...)
	}
	s := &Stream{
		buf: bytes.NewReader(data),
	}
	return s, nil
}

func (s *Stream) Read(p []byte) (int, error) {
	return s.buf.Read(p)
}

func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.buf.Seek(offset, whence)
}

func (s *Stream) Close() error {
	s.buf = nil
	return nil
}

// Size returns the size of decoded stream in bytes.
func (s *Stream) Size() int64 {
	return s.buf.Size()
}

// Decode decodes Ogg/Vorbis data to playable stream.
//
// The sample rate must be same as that of audio context.
func Decode(context *audio.Context, src audio.ReadSeekCloser) (*Stream, error) {
	v, err := vorbis.Open(src)
	if err != nil {
		return nil, err
	}
	// TODO: Remove this magic number
	if v.Channels() != 2 {
		return nil, errors.New("vorbis: number of channels must be 2")
	}
	if v.SampleRate() != context.SampleRate() {
		return nil, fmt.Errorf("vorbis: sample rate must be %d but %d", context.SampleRate(), v.SampleRate())
	}
	s, err := newStream(v)
	if err != nil {
		return nil, err
	}
	return s, nil
}
