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
	"io/ioutil"

	"github.com/hajimehoshi/ebiten/exp/audio"
	"github.com/hajimehoshi/go-vorbis"
)

type Stream struct {
	buf *bytes.Reader
}

func Decode(context *audio.Context, src io.Reader) (*Stream, error) {
	decoded, channels, sampleRate, err := vorbis.Decode(src)
	if err != nil {
		return nil, err
	}
	if channels != 2 {
		return nil, errors.New("audio: number of channels must be 2")
	}
	if sampleRate != context.SampleRate() {
		return nil, fmt.Errorf("audio: sample rate must be %d but %d", context.SampleRate(), sampleRate)
	}
	// TODO: Read all data once so that Seek can be implemented easily.
	// We should look for a wiser way.
	b, err := ioutil.ReadAll(decoded)
	if err != nil {
		return nil, err
	}
	s := &Stream{
		buf: bytes.NewReader(b),
	}
	return s, nil
}

func (s *Stream) Read(p []byte) (int, error) {
	return s.buf.Read(p)
}

func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.buf.Seek(offset, whence)
}
