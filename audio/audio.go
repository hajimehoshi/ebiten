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

// Package audio provides audio players.
//
// The stream format must be 16-bit little endian and 2 channels. The format is as follows:
//
//	[data]      = [sample 1] [sample 2] [sample 3] ...
//	[sample *]  = [channel 1] ...
//	[channel *] = [byte 1] [byte 2] ...
//
// An audio context (audio.Context object) has a sample rate you can specify and all streams you want to play must have the same
// sample rate. However, decoders in e.g. audio/mp3 package adjust sample rate automatically,
// and you don't have to care about it as long as you use those decoders.
//
// An audio context can generate 'players' (audio.Player objects),
// and you can play sound by calling Play function of players.
// When multiple players play, mixing is automatically done.
// Note that too many players may cause distortion.
//
// For the simplest example to play sound, see wav package in the examples.
package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

const (
	channelCount         = 2
	bitDepthInBytesInt16 = 2
	bytesPerSampleInt16  = bitDepthInBytesInt16 * channelCount
)

// A Context represents a current state of audio.
//
// At most one Context object can exist in one process.
// This means only one constant sample rate is valid in your one application.
//
// For a typical usage example, see examples/wav/main.go.
type Context struct {
	playerFactory *playerFactory

	// inited represents whether the audio device is initialized and available or not.
	// On Android, audio loop cannot be started unless JVM is accessible. After updating one frame, JVM should exist.
	inited     chan struct{}
	initedOnce sync.Once

	sampleRate int
	err        error
	ready      bool
	readyOnce  sync.Once

	players map[*playerImpl]struct{}

	m         sync.Mutex
	semaphore chan struct{}
}

var (
	theContext     *Context
	theContextLock sync.Mutex
)

// NewContext creates a new audio context with the given sample rate.
//
// sampleRate specifies the number of samples that should be played during one second.
// Usual numbers are 44100 or 48000. One context has only one sample rate. You cannot play multiple audio
// sources with different sample rates at the same time.
//
// NewContext panics when an audio context is already created.
func NewContext(sampleRate int) *Context {
	theContextLock.Lock()
	defer theContextLock.Unlock()

	if theContext != nil {
		panic("audio: context is already created")
	}

	c := &Context{
		sampleRate:    sampleRate,
		playerFactory: newPlayerFactory(sampleRate),
		players:       map[*playerImpl]struct{}{},
		inited:        make(chan struct{}),
		semaphore:     make(chan struct{}, 1),
	}
	theContext = c

	h := getHook()
	h.OnSuspendAudio(func() error {
		c.semaphore <- struct{}{}
		if err := c.playerFactory.suspend(); err != nil {
			return err
		}
		return nil
	})
	h.OnResumeAudio(func() error {
		<-c.semaphore
		if err := c.playerFactory.resume(); err != nil {
			return err
		}
		return nil
	})

	h.AppendHookOnBeforeUpdate(func() error {
		c.initedOnce.Do(func() {
			close(c.inited)
		})

		var err error
		theContextLock.Lock()
		if theContext != nil {
			err = theContext.error()
		}
		theContextLock.Unlock()
		if err != nil {
			return err
		}

		if err := c.gcPlayers(); err != nil {
			return err
		}
		return nil
	})

	return c
}

// CurrentContext returns the current context or nil if there is no context.
func CurrentContext() *Context {
	theContextLock.Lock()
	c := theContext
	theContextLock.Unlock()
	return c
}

func (c *Context) setError(err error) {
	// TODO: What if c.err already exists?
	c.m.Lock()
	c.err = err
	c.m.Unlock()
}

func (c *Context) error() error {
	c.m.Lock()
	defer c.m.Unlock()
	if c.err != nil {
		return c.err
	}
	return c.playerFactory.error()
}

func (c *Context) setReady() {
	c.m.Lock()
	c.ready = true
	c.m.Unlock()
}

func (c *Context) addPlayer(p *playerImpl) {
	c.m.Lock()
	defer c.m.Unlock()
	c.players[p] = struct{}{}

	// Check the source duplication
	srcs := map[io.Reader]struct{}{}
	for p := range c.players {
		if _, ok := srcs[p.source()]; ok {
			c.err = errors.New("audio: a same source is used by multiple Player")
			return
		}
		srcs[p.source()] = struct{}{}
	}
}

func (c *Context) removePlayer(p *playerImpl) {
	c.m.Lock()
	delete(c.players, p)
	c.m.Unlock()
}

func (c *Context) gcPlayers() error {
	// A Context must not call playerImpl's functions with a lock, or this causes a deadlock (#2737).
	// Copy the playerImpls and iterate them without a lock.
	var players []*playerImpl
	c.m.Lock()
	players = make([]*playerImpl, 0, len(c.players))
	for p := range c.players {
		players = append(players, p)
	}
	c.m.Unlock()

	var playersToRemove []*playerImpl

	// Now reader players cannot call removePlayers from themselves in the current implementation.
	// Underlying playering can be the pause state after fishing its playing,
	// but there is no way to notify this to players so far.
	// Instead, let's check the states proactively every frame.
	for _, p := range players {
		if err := p.Err(); err != nil {
			return err
		}
		if !p.IsPlaying() {
			playersToRemove = append(playersToRemove, p)
		}
	}

	c.m.Lock()
	for _, p := range playersToRemove {
		delete(c.players, p)
	}
	c.m.Unlock()

	return nil
}

// IsReady returns a boolean value indicating whether the audio is ready or not.
//
// On some browsers, user interaction like click or pressing keys is required to start audio.
func (c *Context) IsReady() bool {
	c.m.Lock()
	defer c.m.Unlock()

	if c.ready {
		return true
	}
	if len(c.players) != 0 {
		return false
	}

	c.readyOnce.Do(func() {
		// Create another goroutine since (*Player).Play can lock the context's mutex.
		// TODO: Is this needed for reader players?
		go func() {
			// The audio context is never ready unless there is a player. This is
			// problematic when a user tries to play audio after the context is ready.
			// Play a dummy player to avoid the blocking (#969).
			// Use a long enough buffer so that writing doesn't finish immediately (#970).
			p := NewPlayerFromBytes(c, make([]byte, 16384))
			p.Play()
		}()
	})

	return false
}

// SampleRate returns the sample rate.
func (c *Context) SampleRate() int {
	return c.sampleRate
}

func (c *Context) acquireSemaphore() {
	c.semaphore <- struct{}{}
}

func (c *Context) releaseSemaphore() {
	<-c.semaphore
}

func (c *Context) waitUntilInited() {
	<-c.inited
}

// Player is an audio player which has one stream.
//
// Even when all references to a Player object is gone,
// the object is not GCed until the player finishes playing.
// This means that if a Player plays an infinite stream,
// the object is never GCed unless Close is called.
type Player struct {
	p *playerImpl
}

// NewPlayer creates a new player with the given stream.
//
// src's format must be linear PCM (signed 16bits little endian, 2 channel stereo)
// without a header (e.g. RIFF header).
// The sample rate must be same as that of the audio context.
//
// The player is seekable when src is io.Seeker.
// Attempt to seek the player that is not io.Seeker causes panic.
//
// Note that the given src can't be shared with other Player objects.
//
// NewPlayer tries to call Seek of src to get the current position.
// NewPlayer returns error when the Seek returns error.
//
// A Player doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
func (c *Context) NewPlayer(src io.Reader) (*Player, error) {
	pi, err := c.playerFactory.newPlayer(c, src)
	if err != nil {
		return nil, err
	}

	p := &Player{pi}

	runtime.SetFinalizer(p, (*Player).finalize)

	return p, nil
}

// NewPlayer creates a new player with the given stream.
//
// Deprecated: as of v2.2. Use (*Context).NewPlayer instead.
func NewPlayer(context *Context, src io.Reader) (*Player, error) {
	return context.NewPlayer(src)
}

// NewPlayerFromBytes creates a new player with the given bytes.
//
// As opposed to NewPlayer, you don't have to care if src is already used by another player or not.
// src can be shared by multiple players.
//
// The format of src should be same as noted at NewPlayer.
func (c *Context) NewPlayerFromBytes(src []byte) *Player {
	p, err := c.NewPlayer(bytes.NewReader(src))
	if err != nil {
		// Errors should never happen.
		panic(fmt.Sprintf("audio: %v at NewPlayerFromBytes", err))
	}
	return p
}

// NewPlayerFromBytes creates a new player with the given bytes.
//
// Deprecated: as of v2.2. Use (*Context).NewPlayerFromBytes instead.
func NewPlayerFromBytes(context *Context, src []byte) *Player {
	return context.NewPlayerFromBytes(src)
}

func (p *Player) finalize() {
	runtime.SetFinalizer(p, nil)
	if !p.IsPlaying() {
		_ = p.Close()
	}
}

// Close closes the stream.
//
// When Close is called, the stream owned by the player is NOT closed,
// even if the stream implements io.Closer.
//
// Close returns error when the player is already closed.
func (p *Player) Close() error {
	return p.p.Close()
}

// Play plays the stream.
func (p *Player) Play() {
	p.p.Play()
}

// IsPlaying returns boolean indicating whether the player is playing.
func (p *Player) IsPlaying() bool {
	return p.p.IsPlaying()
}

// Rewind rewinds the current position to the start.
//
// The passed source to NewPlayer must be io.Seeker, or Rewind panics.
//
// Rewind returns error when seeking the source stream returns error.
func (p *Player) Rewind() error {
	return p.p.Rewind()
}

// SetPosition sets the position with the given offset.
//
// The passed source to NewPlayer must be io.Seeker, or SetPosition panics.
//
// SetPosition returns error when seeking the source stream returns an error.
func (p *Player) SetPosition(offset time.Duration) error {
	return p.p.SetPosition(offset)
}

// Seek seeks the position with the given offset.
//
// Deprecated: as of v2.6. Use SetPosition instead.
func (p *Player) Seek(offset time.Duration) error {
	return p.SetPosition(offset)
}

// Pause pauses the playing.
func (p *Player) Pause() {
	p.p.Pause()
}

// Position returns the current position in time.
//
// As long as the player continues to play, Position's returning value is increased monotonically,
// even though the source stream loops and its position goes back.
func (p *Player) Position() time.Duration {
	return p.p.Position()
}

// Current returns the current position in time.
//
// Deprecated: as of v2.6. Use Position instead.
func (p *Player) Current() time.Duration {
	return p.Position()
}

// Volume returns the current volume of this player [0-1].
func (p *Player) Volume() float64 {
	return p.p.Volume()
}

// SetVolume sets the volume of this player.
// volume must be in between 0 and 1. SetVolume panics otherwise.
func (p *Player) SetVolume(volume float64) {
	p.p.SetVolume(volume)
}

// SetBufferSize adjusts the buffer size of the player.
// If 0 is specified, the default buffer size is used.
// A small buffer size is useful if you want to play a real-time PCM for example.
// Note that the audio quality might be affected if you modify the buffer size.
func (p *Player) SetBufferSize(bufferSize time.Duration) {
	p.p.SetBufferSize(bufferSize)
}

type hook interface {
	OnSuspendAudio(f func() error)
	OnResumeAudio(f func() error)
	AppendHookOnBeforeUpdate(f func() error)
}

var hookForTesting hook

func getHook() hook {
	if hookForTesting != nil {
		return hookForTesting
	}
	return &hookImpl{}
}

type hookImpl struct{}

func (h *hookImpl) OnSuspendAudio(f func() error) {
	hooks.OnSuspendAudio(f)
}

func (h *hookImpl) OnResumeAudio(f func() error) {
	hooks.OnResumeAudio(f)
}

func (h *hookImpl) AppendHookOnBeforeUpdate(f func() error) {
	hooks.AppendHookOnBeforeUpdate(f)
}

// Resample converts the sample rate of the given stream.
// size is the length of the source stream in bytes.
// from is the original sample rate.
// to is the target sample rate.
//
// If the original sample rate equals to the new one, Resample returns source as it is.
func Resample(source io.ReadSeeker, size int64, from, to int) io.ReadSeeker {
	if from == to {
		return source
	}
	return convert.NewResampling(source, size, from, to)
}
