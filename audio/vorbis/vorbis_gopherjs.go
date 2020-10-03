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
// +build !wasm

package vorbis

import (
	"io/ioutil"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/vorbis/internal/stb"
)

type decoderImpl struct {
	samples    *stb.Samples
	channels   int
	sampleRate int
}

func (d *decoderImpl) Read(buf []float32) (int, error) {
	return d.samples.Read(buf)
}

func (d *decoderImpl) Length() int64 {
	return d.samples.Length()
}

func (d *decoderImpl) Channels() int {
	return d.channels
}

func (d *decoderImpl) SampleRate() int {
	return d.sampleRate
}

func (d *decoderImpl) SetPosition(pos int64) error {
	return d.samples.SetPosition(pos)
}

func newDecoder(in audio.ReadSeekCloser) (decoder, error) {
	buf, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}

	samples, channels, sampleRate, err := stb.DecodeVorbis(buf)
	if err != nil {
		return nil, err
	}
	return &decoderImpl{samples, channels, sampleRate}, nil
}
