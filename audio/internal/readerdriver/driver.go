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
	"io"
)

type Context interface {
	NewPlayer(io.Reader) Player
	Suspend() error
	Resume() error
	io.Closer
}

type Player interface {
	Pause()
	Play()
	IsPlaying() bool
	Reset()
	Volume() float64
	SetVolume(volume float64)
	UnplayedBufferSize() int64
	Err() error
	io.Closer
}

type playerState int

const (
	playerPaused playerState = iota
	playerPlay
	playerClosed
)

// TODO: The term 'buffer' is confusing. Name each buffer with good terms.

// oneBufferSize returns the size of one buffer in the player implementation.
func (c *context) oneBufferSize() int {
	bytesPerSample := c.channelNum * c.bitDepthInBytes
	s := c.sampleRate * bytesPerSample / 4

	// Align s in multiples of bytes per sample, or a buffer could have extra bytes.
	return s / bytesPerSample * bytesPerSample
}

// maxBufferSize returns the maximum size of the buffer for the audio source.
// This buffer is used when unreading on pausing the player.
func (c *context) maxBufferSize() int {
	// The number of underlying buffers should be 2.
	return c.oneBufferSize() * 2
}
