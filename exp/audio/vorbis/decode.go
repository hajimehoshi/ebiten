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

	"github.com/hajimehoshi/ebiten/exp/audio"
)

// TODO: src should be ReadCloser?

func Decode(context *audio.Context, src audio.ReadSeekCloser) (*Stream, error) {
	decoded, channelNum, sampleRate, err := decode(src)
	if err != nil {
		return nil, err
	}
	// TODO: Remove this magic number
	if channelNum != 2 {
		return nil, errors.New("vorbis: number of channels must be 2")
	}
	if sampleRate != context.SampleRate() {
		return nil, fmt.Errorf("vorbis: sample rate must be %d but %d", context.SampleRate(), sampleRate)
	}
	s := &Stream{
		buf: bytes.NewReader(decoded),
	}
	return s, nil
}
