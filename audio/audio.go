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
//   [data]      = [sample 1] [sample 2] [sample 3] ...
//   [sample *]  = [channel 1] ...
//   [channel *] = [byte 1] [byte 2] ...
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

	"github.com/hajimehoshi/ebiten/v2/audio/internal/readerdriver"
	"github.com/hajimehoshi/ebiten/v2/internal/hooks"
)

const (
	channelNum      = 2
	bitDepthInBytes = 2
	bytesPerSample  = bitDepthInBytes * channelNum
)

type newPlayerImpler interface {
	newPlayerImpl(context *Context, src io.Reader) (playerImpl, error)
}

// A Context represents a current state of audio.
//
// At most one Context object can exist in one process.
// This means only one constant sample rate is valid in your one application.
//
// For a typical usage example, see examples/wav/main.go.
type Context struct {
	np newPlayerImpler

	// inited represents whether the audio device is initialized and available or not.
	// On Android, audio loop cannot be started unless JVM is accessible. After updating one frame, JVM should exist.
	inited     chan struct{}
	initedOnce sync.Once

	sampleRate int
	err        error
	ready      bool

	players map[playerImpl]struct{}

	m         sync.Mutex
	semaphore chan struct{}
}

var (
	theContext     *Context
	theContextLock sync.Mutex
)

// NewContext creates a new audio context with the given sample rate.
//
// The sample rate is also used for decoding MP3 with audio/mp3 package
// or other formats as the target sample rate.
//
// sampleRate should be 44100 or 48000.
// Other values might not work.
// For example, 22050 causes error on Safari when decoding MP3.
//
// NewContext panics when an audio context is already created.
func NewContext(sampleRate int) *Context {
	theContextLock.Lock()
	defer theContextLock.Unlock()

	if theContext != nil {
		panic("audio: context is already created")
	}

	var np newPlayerImpler
	if readerdriver.IsAvailable() {
		// 'Reader players' are players that implement io.Reader. This is the new way and
		// not all the environments support reader players. Reader players can have enough
		// buffers so that clicking noises can be avoided compared to writer players.
		// Reder players will replace writer players in any platforms in the future.
		np = newReaderPlayerFactory(sampleRate)
	} else {
		// 'Writer players' are players that implement io.Writer. This is the old way but
		// all the environments support writer players. Writer players cannot have enough
		// buffers and clicking noises are sometimes problematic (#1356, #1458).
		np = newWriterPlayerFactory(sampleRate)
	}

	c := &Context{
		sampleRate: sampleRate,
		np:         np,
		players:    map[playerImpl]struct{}{},
		inited:     make(chan struct{}),
		semaphore:  make(chan struct{}, 1),
	}
	theContext = c

	h := getHook()
	h.OnSuspendAudio(func() {
		c.semaphore <- struct{}{}
		if s, ok := np.(interface{ suspend() }); ok {
			s.suspend()
		}
	})
	h.OnResumeAudio(func() {
		<-c.semaphore
		if s, ok := np.(interface{ resume() }); ok {
			s.resume()
		}
	})

	h.AppendHookOnBeforeUpdate(func() error {
		c.initedOnce.Do(func() {
			close(c.inited)
		})

		var err error
		theContextLock.Lock()
		if theContext != nil {
			theContext.m.Lock()
			err = theContext.err
			theContext.m.Unlock()
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

func (c *Context) hasError() bool {
	c.m.Lock()
	r := c.err != nil
	c.m.Unlock()
	return r
}

func (c *Context) setError(err error) {
	// TODO: What if c.err already exists?
	c.m.Lock()
	c.err = err
	c.m.Unlock()
}

func (c *Context) setReady() {
	c.m.Lock()
	c.ready = true
	c.m.Unlock()
}

func (c *Context) addPlayer(p playerImpl) {
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

func (c *Context) removePlayer(p playerImpl) {
	c.m.Lock()
	delete(c.players, p)
	c.m.Unlock()
}

func (c *Context) gcPlayers() error {
	c.m.Lock()
	defer c.m.Unlock()

	// Now reader players cannot call removePlayers from themselves in the current implementation.
	// Underlying playering can be the pause state after fishing its playing,
	// but there is no way to notify this to readerPlayers so far.
	// Instead, let's check the states proactively every frame.
	for p := range c.players {
		rp, ok := p.(*readerPlayer)
		if !ok {
			return nil
		}
		if err := rp.Err(); err != nil {
			return err
		}
		if !rp.IsPlaying() {
			delete(c.players, p)
		}
	}

	return nil
}

// IsReady returns a boolean value indicating whether the audio is ready or not.
//
// On some browsers, user interaction like click or pressing keys is required to start audio.
func (c *Context) IsReady() bool {
	c.m.Lock()
	defer c.m.Unlock()

	r := c.ready
	if r {
		return r
	}
	if len(c.players) != 0 {
		return r
	}

	// Create another goroutine since (*Player).Play can lock the context's mutex.
	go func() {
		// The audio context is never ready unless there is a player. This is
		// problematic when a user tries to play audio after the context is ready.
		// Play a dummy player to avoid the blocking (#969).
		// Use a long enough buffer so that writing doesn't finish immediately (#970).
		p := NewPlayerFromBytes(c, make([]byte, bufferSize()*2))
		p.Play()
	}()

	return r
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
	p playerImpl
}

type playerImpl interface {
	io.Closer

	Play()
	IsPlaying() bool
	Pause()
	Volume() float64
	SetVolume(volume float64)
	Current() time.Duration
	Rewind() error
	Seek(offset time.Duration) error

	source() io.Reader
}

// NewPlayer creates a new player with the given stream.
//
// src's format must be linear PCM (16bits little endian, 2 channel stereo)
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
func NewPlayer(context *Context, src io.Reader) (*Player, error) {
	pi, err := context.np.newPlayerImpl(context, src)
	if err != nil {
		return nil, err
	}

	p := &Player{pi}

	runtime.SetFinalizer(p, (*Player).finalize)

	return p, nil
}

// NewPlayerFromBytes creates a new player with the given bytes.
//
// As opposed to NewPlayer, you don't have to care if src is already used by another player or not.
// src can be shared by multiple players.
//
// The format of src should be same as noted at NewPlayer.
func NewPlayerFromBytes(context *Context, src []byte) *Player {
	b := bytes.NewReader(src)
	p, err := NewPlayer(context, b)
	if err != nil {
		// Errors should never happen.
		panic(fmt.Sprintf("audio: %v at NewPlayerFromBytes", err))
	}
	return p
}

func (p *Player) finalize() {
	runtime.SetFinalizer(p, nil)
	if !p.IsPlaying() {
		p.Close()
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

// Seek seeks the position with the given offset.
//
// The passed source to NewPlayer must be io.Seeker, or Seek panics.
//
// Seek returns error when seeking the source stream returns error.
func (p *Player) Seek(offset time.Duration) error {
	return p.p.Seek(offset)
}

// Pause pauses the playing.
func (p *Player) Pause() {
	p.p.Pause()
}

// Current returns the current position in time.
//
// As long as the player continues to play, Current's returning value is increased monotonically,
// even though the source stream loops and its position goes back.
func (p *Player) Current() time.Duration {
	return p.p.Current()
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

type hook interface {
	OnSuspendAudio(f func())
	OnResumeAudio(f func())
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

func (h *hookImpl) OnSuspendAudio(f func()) {
	hooks.OnSuspendAudio(f)
}

func (h *hookImpl) OnResumeAudio(f func()) {
	hooks.OnResumeAudio(f)
}

func (h *hookImpl) AppendHookOnBeforeUpdate(f func() error) {
	hooks.AppendHookOnBeforeUpdate(f)
}
