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

// Context is the main object in Oto. It interacts with the audio drivers.
//
// To play sound with Oto, first create a context. Then use the context to create
// an arbitrary number of players. Then use the players to play sound.
//
// There can only be one context at any time. Closing a context and opening a new one is allowed.
type Context struct {
	context *context
}

// NewPlayer creates a new, ready-to-use Player belonging to the Context.
//
// The r's format is as follows:
//   [data]      = [sample 1] [sample 2] [sample 3] ...
//   [sample *]  = [channel 1] ...
//   [channel *] = [byte 1] [byte 2] ...
// Byte ordering is little endian.
//
// A player has some amount of an underlying buffer.
// Read data from r is queued to the player's underlying buffer.
// The underlying buffer is consumed by its playing.
// Then, r's position and the current playing position don't necessarily match.
// If you want to clear the underlying buffer for some reasons e.g., you want to seek the position of r,
// call the player's Reset function.
//
// You cannot share r by multiple players.
func (c *Context) NewPlayer(r io.Reader) Player {
	return c.context.NewPlayer(r)
}

// Suspend suspends the entire audio play.
func (c *Context) Suspend() error {
	return c.context.Suspend()
}

// Resume resumes the entire audio play, which was suspended by Suspend.
func (c *Context) Resume() error {
	return c.context.Resume()
}

// NewContext creates a new context, that creates and holds ready-to-use Player objects,
// and returns a context, a channel that is closed when the context is ready, and an error if it exists.
//
// The sampleRate argument specifies the number of samples that should be played during one second.
// Usual numbers are 44100 or 48000.
//
// The channelNum argument specifies the number of channels. One channel is mono playback. Two
// channels are stereo playback. No other values are supported.
//
// The bitDepthInBytes argument specifies the number of bytes per sample per channel. The usual value
// is 2. Only values 1 and 2 are supported.
func NewContext(sampleRate int, channelNum int, bitDepthInBytes int) (*Context, chan struct{}, error) {
	ctx, ready, err := newContext(sampleRate, channelNum, bitDepthInBytes)
	if err != nil {
		return nil, nil, err
	}
	return &Context{context: ctx}, ready, nil
}

// Player is a PCM (pulse-code modulation) audio player.
type Player interface {
	// Pause pauses its playing.
	Pause()

	// Play starts its playing if it doesn't play.
	Play()

	// IsPlaying reports whether this player is playing.
	IsPlaying() bool

	// Reset clears the underyling buffer and pauses its playing.
	Reset()

	// Volume returns the current volume in between [0, 1].
	// The default volume is 1.
	Volume() float64

	// SetVolume sets the current volume in between [0, 1].
	SetVolume(volume float64)

	// UnplayedBufferSize returns the byte size in the underlying buffer that is not played yet.
	UnplayedBufferSize() int

	// Err returns an error if this player has an error.
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
