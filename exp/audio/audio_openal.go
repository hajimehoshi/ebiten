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

// +build !js,!windows

package audio

import (
	"io"

	"golang.org/x/mobile/exp/audio"
)

type readSeekCloser struct {
	io.ReadSeeker
}

func (r *readSeekCloser) Close() error {
	return nil
}

type player struct {
	*audio.Player
}

func newPlayer(src io.ReadSeeker, sampleRate int) *Player {
	p, err := audio.NewPlayer(&readSeekCloser{src}, audio.Stereo16, int64(sampleRate))
	if err != nil {
		// TODO: Should we return errors for this method?
		panic(err)
	}
	pp := &player{p}
	return &Player{pp}
}

func (p *player) play() error {
	// TODO: audio.NewPlayer interprets WAV header, which we don't want.
	// Use OpenAL or native API instead.
	return p.Play()
}

// TODO: Implement Close method
