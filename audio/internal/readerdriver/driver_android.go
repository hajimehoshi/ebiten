// Copyright 2021 The Ebiten Authors
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

package readerdriver

import (
	"github.com/hajimehoshi/ebiten/v2/audio/internal/oboe"
)

type context struct {
	sampleRate      int
	channelNum      int
	bitDepthInBytes int

	players *players
}

func newContext(sampleRate int, channelNum int, bitDepthInBytes int) (*context, chan struct{}, error) {
	ready := make(chan struct{})
	close(ready)

	c := &context{
		sampleRate:      sampleRate,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
		players:         newPlayers(),
	}
	if err := oboe.Play(sampleRate, channelNum, bitDepthInBytes, c.players.read); err != nil {
		return nil, nil, err
	}
	return c, ready, nil
}

func (c *context) Suspend() error {
	return oboe.Suspend()
}

func (c *context) Resume() error {
	return oboe.Resume()
}
