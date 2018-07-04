// Copyright 2018 The Ebiten Authors
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

// +build js

package vorbis

import (
	"io"
	"io/ioutil"

	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/vorbis/internal/stb"
)

type decoderImpl struct {
	data       []float32
	channels   int
	sampleRate int
}

func (d *decoderImpl) Read(buf []float32) (int, error) {
	if len(d.data) == 0 {
		return 0, io.EOF
	}

	n := copy(buf, d.data)
	d.data = d.data[n:]
	return n, nil
}

func (d *decoderImpl) Length() int64 {
	return int64(len(d.data) / d.channels)
}

func (d *decoderImpl) Channels() int {
	return d.channels
}

func (d *decoderImpl) SampleRate() int {
	return d.sampleRate
}

func newDecoder(in audio.ReadSeekCloser) (decoder, error) {
	buf, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	data, channels, sampleRate, err := stb.DecodeVorbis(buf)
	if err != nil {
		return nil, err
	}
	return &decoderImpl{data, channels, sampleRate}, nil
}
