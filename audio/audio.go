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
// The stream format must be 16-bit little endian or 32-bit float little endian, and 2 channels. The format is as follows:
//
//	[data]      = [sample 1] [sample 2] [sample 3] ...
//	[sample *]  = [channel 1] ...
//	[channel *] = [byte 1] [byte 2] ...
//
// An audio context (audio.Context object) has a sample rate you can specify
// and all streams you want to play must have the same sample rate.
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
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/convert"
	"github.com/hajimehoshi/ebiten/v2/internal/hook"
)

const (
	channelCount           = 2
	bitDepthInBytesInt16   = 2
	bitDepthInBytesFloat32 = 4
)

// A Context represents a current state of audio.
//
// At most one Context object can exist in one process.
// This means only one constant sample rate is valid in your one application.
//
// For a typical usage example, see examples/wav/main.go.
type Context struct {
	playerFactory *playerFactory

	sampleRate int
	err        error
	ready      bool

	playingPlayers map[*playerImpl]struct{}

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
		sampleRate:     sampleRate,
		playerFactory:  newPlayerFactory(sampleRate),
		playingPlayers: map[*playerImpl]struct{}{},
		semaphore:      make(chan struct{}, 1),
	}
	theContext = c

	h := getHook()
	h.OnSuspendAudio(func() error {
		c.semaphore <- struct{}{}
		if err := c.playerFactory.suspend(); err != nil {
			return err
		}
		if err := c.onSuspend(); err != nil {
			return err
		}
		return nil
	})
	h.OnResumeAudio(func() error {
		<-c.semaphore
		if err := c.playerFactory.resume(); err != nil {
			return err
		}
		if err := c.onResume(); err != nil {
			return err
		}
		return nil
	})

	h.AppendHookOnBeforeUpdate(func() error {
		var err error
		theContextLock.Lock()
		if theContext != nil {
			err = theContext.error()
		}
		theContextLock.Unlock()
		if err != nil {
			return err
		}

		// Initialize the context here in the case when there is no player and
		// the program waits for IsReady() to be true (#969, #970, #2715).
		ready, err := c.playerFactory.initContextIfNeeded()
		if err != nil {
			return err
		}
		if ready != nil {
			go func() {
				<-ready
				c.setReady()
			}()
		}
		return nil
	})

	// In the current Ebitengine implementation, update might not be called when the window is in background (#3154).
	// In this case, an audio player position is not updated correctly with AppendHookOnBeforeUpdate.
	// Use a distinct goroutine to update the player states.
	go func() {
		for {
			if err := c.updatePlayers(); err != nil {
				c.setError(err)
				return
			}
			time.Sleep(time.Second / 100)
		}
	}()

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
		return fmt.Errorf("audio: audio error: %w", c.err)
	}
	if err := c.playerFactory.error(); err != nil {
		return fmt.Errorf("audio: audio error: %w", err)
	}
	return nil
}

func (c *Context) setReady() {
	c.m.Lock()
	c.ready = true
	c.m.Unlock()
}

func (c *Context) addPlayingPlayer(p *playerImpl) {
	c.m.Lock()
	defer c.m.Unlock()
	c.playingPlayers[p] = struct{}{}

	if !reflect.ValueOf(p.sourceIdent()).Comparable() {
		return
	}

	// Check the source duplication
	srcs := map[any]struct{}{}
	for p := range c.playingPlayers {
		if _, ok := srcs[p.sourceIdent()]; ok {
			c.err = errors.New("audio: the same source must not be used by multiple Player objects")
			return
		}
		srcs[p.sourceIdent()] = struct{}{}
	}
}

func (c *Context) removePlayingPlayer(p *playerImpl) {
	c.m.Lock()
	delete(c.playingPlayers, p)
	c.m.Unlock()
}

func (c *Context) onSuspend() error {
	// A Context must not call playerImpl's functions with a lock, or this causes a deadlock (#2737).
	// Copy the playerImpls and iterate them without a lock.
	var players []*playerImpl
	c.m.Lock()
	players = make([]*playerImpl, 0, len(c.playingPlayers))
	for p := range c.playingPlayers {
		players = append(players, p)
	}
	c.m.Unlock()

	for _, p := range players {
		if err := p.Err(); err != nil {
			return err
		}
		p.onContextSuspended()
	}

	return nil
}

func (c *Context) onResume() error {
	// A Context must not call playerImpl's functions with a lock, or this causes a deadlock (#2737).
	// Copy the playerImpls and iterate them without a lock.
	var players []*playerImpl
	c.m.Lock()
	players = make([]*playerImpl, 0, len(c.playingPlayers))
	for p := range c.playingPlayers {
		players = append(players, p)
	}
	c.m.Unlock()

	for _, p := range players {
		if err := p.Err(); err != nil {
			return err
		}
		p.onContextResumed()
	}

	return nil
}

func (c *Context) updatePlayers() error {
	// A Context must not call playerImpl's functions with a lock, or this causes a deadlock (#2737).
	// Copy the playerImpls and iterate them without a lock.
	var players []*playerImpl
	c.m.Lock()
	players = make([]*playerImpl, 0, len(c.playingPlayers))
	for p := range c.playingPlayers {
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
		p.updatePosition()
		if !p.IsPlaying() {
			playersToRemove = append(playersToRemove, p)
		}
	}

	c.m.Lock()
	for _, p := range playersToRemove {
		delete(c.playingPlayers, p)
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
	return c.ready
}

// SampleRate returns the sample rate.
func (c *Context) SampleRate() int {
	return c.sampleRate
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
//
// For new code, NewPlayerF32 is preferrable to NewPlayer, since Ebitengine will treat only 32bit float audio internally in the future.
//
// A Player for 16bit integer must be used with 16bit integer version of audio APIs, like vorbis.DecodeWithoutResampling or audio.NewInfiniteLoop, not or vorbis.DecodeF32 or audio.NewInfiniteLoopF32.
func (c *Context) NewPlayer(src io.Reader) (*Player, error) {
	_, seekable := src.(io.Seeker)
	f32Src := convert.NewFloat32BytesReaderFromInt16BytesReader(src)
	pi, err := c.playerFactory.newPlayer(c, f32Src, seekable, src, bitDepthInBytesFloat32)
	if err != nil {
		return nil, err
	}

	p := &Player{pi}

	runtime.SetFinalizer(p, (*Player).finalize)

	return p, nil
}

// NewPlayerF32 creates a new player with the given stream.
//
// src's format must be linear PCM (32bit float, little endian, 2 channel stereo)
// without a header (e.g. RIFF header).
// The sample rate must be same as that of the audio context.
//
// The player is seekable when src is io.Seeker.
// Attempt to seek the player that is not io.Seeker causes panic.
//
// Note that the given src can't be shared with other Player objects.
//
// NewPlayerF32 tries to call Seek of src to get the current position.
// NewPlayerF32 returns error when the Seek returns error.
//
// A Player doesn't close src even if src implements io.Closer.
// Closing the source is src owner's responsibility.
//
// For new code, NewPlayerF32 is preferrable to NewPlayer, since Ebitengine will treat only 32bit float audio internally in the future.
//
// A Player for 32bit float must be used with 32bit float version of audio APIs, like vorbis.DecodeF32 or audio.NewInfiniteLoopF32, not vorbis.DecodeWithoutResampling or audio.NewInfiniteLoop.
func (c *Context) NewPlayerF32(src io.Reader) (*Player, error) {
	_, seekable := src.(io.Seeker)
	pi, err := c.playerFactory.newPlayer(c, src, seekable, src, bitDepthInBytesFloat32)
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

// NewPlayerF32FromBytes creates a new player with the given bytes.
//
// As opposed to NewPlayerF32, you don't have to care if src is already used by another player or not.
// src can be shared by multiple players.
//
// The format of src should be same as noted at NewPlayerF32.
func (c *Context) NewPlayerF32FromBytes(src []byte) *Player {
	p, err := c.NewPlayerF32(bytes.NewReader(src))
	if err != nil {
		// Errors should never happen.
		panic(fmt.Sprintf("audio: %v at NewPlayerFromBytesF32", err))
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

type hooker interface {
	OnSuspendAudio(f func() error)
	OnResumeAudio(f func() error)
	AppendHookOnBeforeUpdate(f func() error)
}

var hookerForTesting hooker

func getHook() hooker {
	if hookerForTesting != nil {
		return hookerForTesting
	}
	return &hookerImpl{}
}

type hookerImpl struct{}

func (h *hookerImpl) OnSuspendAudio(f func() error) {
	hook.OnSuspendAudio(f)
}

func (h *hookerImpl) OnResumeAudio(f func() error) {
	hook.OnResumeAudio(f)
}

func (h *hookerImpl) AppendHookOnBeforeUpdate(f func() error) {
	hook.AppendHookOnBeforeUpdate(f)
}

// ResampleReader converts the sample rate of the given singed 16bit integer, little-endian, 2 channels (stereo) stream.
// size is the length of the source stream in bytes.
// from is the original sample rate.
// to is the target sample rate.
//
// If the original sample rate equals to the new one, ResampleReader returns source as it is.
//
// The returned value implements io.Seeker when the source implements io.Seeker.
// The returned value might implement io.Seeker even when the source doesn't implement io.Seeker, but
// there is no guarantee that the Seek function works correctly.
func ResampleReader(source io.Reader, size int64, from, to int) io.Reader {
	if from == to {
		return source
	}
	return convert.NewResampling(source, size, from, to, bitDepthInBytesInt16)
}

// ResampleReaderF32 converts the sample rate of the given 32bit float, little-endian, 2 channels (stereo) stream.
// size is the length of the source stream in bytes.
// from is the original sample rate.
// to is the target sample rate.
//
// If the original sample rate equals to the new one, ResampleReaderF32 returns source as it is.
//
// The returned value implements io.Seeker when the source implements io.Seeker.
// The returned value might implement io.Seeker even when the source doesn't implement io.Seeker, but
// there is no guarantee that the Seek function works correctly.
func ResampleReaderF32(source io.Reader, size int64, from, to int) io.Reader {
	if from == to {
		return source
	}
	return convert.NewResampling(source, size, from, to, bitDepthInBytesFloat32)
}

// Resample converts the sample rate of the given singed 16bit integer, little-endian, 2 channels (stereo) stream.
// size is the length of the source stream in bytes.
// from is the original sample rate.
// to is the target sample rate.
//
// If the original sample rate equals to the new one, Resample returns source as it is.
//
// Deprecated: as of v2.9. Use ResampleReader instead.
func Resample(source io.ReadSeeker, size int64, from, to int) io.ReadSeeker {
	if from == to {
		return source
	}
	return convert.NewResampling(source, size, from, to, bitDepthInBytesInt16)
}

// ResampleF32 converts the sample rate of the given 32bit float, little-endian, 2 channels (stereo) stream.
// size is the length of the source stream in bytes.
// from is the original sample rate.
// to is the target sample rate.
//
// If the original sample rate equals to the new one, ResampleF32 returns source as it is.
//
// Deprecated: as of v2.9. Use ResampleReaderF32 instead.
func ResampleF32(source io.ReadSeeker, size int64, from, to int) io.ReadSeeker {
	if from == to {
		return source
	}
	return convert.NewResampling(source, size, from, to, bitDepthInBytesFloat32)
}
