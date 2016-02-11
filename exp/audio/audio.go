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

package audio

import (
	"io"
)

type Player struct {
	player *player
}

// NewPlayer creates a new player with the given data to the given channel.
// The given data is queued to the end of the buffer.
// This may not be played immediately when data already exists in the buffer.
//
// src's format must be linear PCM (16bits, 2 channel stereo, little endian)
// without a header (e.g. RIFF header).
//
// TODO: Pass sample rate and num of channels.
func NewPlayer(src io.ReadSeeker, sampleRate int) *Player {
	return newPlayer(src, sampleRate)
}

func (p *Player) Play() error {
	return p.player.play()
}

func (p *Player) Close() error {
	return p.player.close()
}
