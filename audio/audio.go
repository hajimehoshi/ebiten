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
)

const (
	channelNum     = 2
	bytesPerSample = 2 * channelNum
)

// A Context represents a current state of audio.
//
// At most one Context object can exist in one process.
// This means only one constant sample rate is valid in your one application.
//
// For a typical usage example, see examples/wav/main.go.
type Context struct {
	c context

	// inited represents whether the audio device is initialized and available or not.
	// On Android, audio loop cannot be started unless JVM is accessible. After updating one frame, JVM should exist.
	inited     chan struct{}
	initedOnce sync.Once

	sampleRate int
	err        error
	ready      bool

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

	c := &Context{
		sampleRate: sampleRate,
		c:          newContext(sampleRate),
		players:    map[*playerImpl]struct{}{},
		inited:     make(chan struct{}),
		semaphore:  make(chan struct{}, 1),
	}
	theContext = c

	h := getHook()
	h.OnSuspendAudio(func() {
		c.semaphore <- struct{}{}
	})
	h.OnResumeAudio(func() {
		<-c.semaphore
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
		return err
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

func (c *Context) addPlayer(p *playerImpl) {
	c.m.Lock()
	defer c.m.Unlock()
	c.players[p] = struct{}{}

	// Check the source duplication
	srcs := map[io.Reader]struct{}{}
	for p := range c.players {
		if _, ok := srcs[p.src]; ok {
			c.err = errors.New("audio: a same source is used by multiple Player")
			return
		}
		srcs[p.src] = struct{}{}
	}
}

func (c *Context) removePlayer(p *playerImpl) {
	c.m.Lock()
	delete(c.players, p)
	c.m.Unlock()
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

// Player is an audio player which has one stream.
//
// Even when all references to a Player object is gone,
// the object is not GCed until the player finishes playing.
// This means that if a Player plays an infinite stream,
// the object is never GCed unless Close is called.
type Player struct {
	p *playerImpl
}

type playerImpl struct {
	context          *Context
	src              io.Reader
	sampleRate       int
	playing          bool
	closedExplicitly bool
	isLoopActive     bool

	buf    []byte
	pos    int64
	volume float64

	m sync.Mutex
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
	p := &Player{
		&playerImpl{
			context:    context,
			src:        src,
			sampleRate: context.sampleRate,
			volume:     1,
		},
	}
	if seeker, ok := p.p.src.(io.Seeker); ok {
		// Get the current position of the source.
		pos, err := seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}
		p.p.pos = pos
	}
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
	runtime.SetFinalizer(p, nil)
	return p.p.Close()
}

func (p *playerImpl) Close() error {
	p.m.Lock()
	defer p.m.Unlock()

	p.playing = false
	if p.closedExplicitly {
		return fmt.Errorf("audio: the player is already closed")
	}
	p.closedExplicitly = true
	return nil
}

// Play plays the stream.
func (p *Player) Play() {
	p.p.Play()
}

func (p *playerImpl) Play() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closedExplicitly {
		p.context.setError(fmt.Errorf("audio: the player is already closed"))
		return
	}

	p.playing = true
	if p.isLoopActive {
		return
	}

	// Set p.isLoopActive to true here, not in the loop. This prevents duplicated active loops.
	p.isLoopActive = true
	p.context.addPlayer(p)

	go p.loop()
	return
}

func (p *playerImpl) loop() {
	<-p.context.inited

	w := p.context.c.NewPlayer()
	wclosed := make(chan struct{})
	defer func() {
		<-wclosed
		w.Close()
	}()

	defer func() {
		p.m.Lock()
		p.playing = false
		p.context.removePlayer(p)
		p.isLoopActive = false
		p.m.Unlock()
	}()

	ch := make(chan []byte)
	defer close(ch)

	go func() {
		for buf := range ch {
			if _, err := w.Write(buf); err != nil {
				p.context.setError(err)
				break
			}
			p.context.setReady()
		}
		close(wclosed)
	}()

	for {
		buf, ok := p.read()
		if !ok {
			return
		}
		ch <- buf
	}
}

func (p *playerImpl) read() ([]byte, bool) {
	if p.context.hasError() {
		return nil, false
	}

	if p.closedExplicitly {
		return nil, false
	}

	p.context.semaphore <- struct{}{}
	defer func() {
		<-p.context.semaphore
	}()

	p.m.Lock()
	defer p.m.Unlock()

	// playing can be false when pausing.
	if !p.playing {
		return nil, false
	}

	const bufSize = 2048
	newBuf := make([]byte, bufSize-len(p.buf))
	n, err := p.src.Read(newBuf)
	if err != nil {
		if err != io.EOF {
			p.context.setError(err)
			return nil, false
		}
		if n == 0 {
			return nil, false
		}
	}
	buf := append(p.buf, newBuf[:n]...)

	n2 := len(buf) - len(buf)%bytesPerSample
	buf, p.buf = buf[:n2], buf[n2:]

	for i := 0; i < len(buf)/2; i++ {
		v16 := int16(buf[2*i]) | (int16(buf[2*i+1]) << 8)
		v16 = int16(float64(v16) * p.volume)
		buf[2*i] = byte(v16)
		buf[2*i+1] = byte(v16 >> 8)
	}
	p.pos += int64(len(buf))

	return buf, true
}

// IsPlaying returns boolean indicating whether the player is playing.
func (p *Player) IsPlaying() bool {
	return p.p.IsPlaying()
}

func (p *playerImpl) IsPlaying() bool {
	p.m.Lock()
	r := p.playing
	p.m.Unlock()
	return r
}

// Rewind rewinds the current position to the start.
//
// The passed source to NewPlayer must be io.Seeker, or Rewind panics.
//
// Rewind returns error when seeking the source stream returns error.
func (p *Player) Rewind() error {
	return p.p.Rewind()
}

func (p *playerImpl) Rewind() error {
	if _, ok := p.src.(io.Seeker); !ok {
		panic("audio: player to be rewound must be io.Seeker")
	}
	return p.Seek(0)
}

// Seek seeks the position with the given offset.
//
// The passed source to NewPlayer must be io.Seeker, or Seek panics.
//
// Seek returns error when seeking the source stream returns error.
func (p *Player) Seek(offset time.Duration) error {
	return p.p.Seek(offset)
}

func (p *playerImpl) Seek(offset time.Duration) error {
	p.m.Lock()
	defer p.m.Unlock()

	o := int64(offset) * bytesPerSample * int64(p.sampleRate) / int64(time.Second)
	o = o - (o % bytesPerSample)

	seeker, ok := p.src.(io.Seeker)
	if !ok {
		panic("audio: the source must be io.Seeker when seeking")
	}
	pos, err := seeker.Seek(o, io.SeekStart)
	if err != nil {
		return err
	}

	p.buf = nil
	p.pos = pos
	return nil
}

// Pause pauses the playing.
func (p *Player) Pause() {
	p.p.Pause()
}

func (p *playerImpl) Pause() {
	p.m.Lock()
	p.playing = false
	p.m.Unlock()
}

// Current returns the current position in time.
func (p *Player) Current() time.Duration {
	return p.p.Current()
}

func (p *playerImpl) Current() time.Duration {
	p.m.Lock()
	sample := p.pos / bytesPerSample
	p.m.Unlock()
	return time.Duration(sample) * time.Second / time.Duration(p.sampleRate)
}

// Volume returns the current volume of this player [0-1].
func (p *Player) Volume() float64 {
	return p.p.Volume()
}

func (p *playerImpl) Volume() float64 {
	p.m.Lock()
	v := p.volume
	p.m.Unlock()
	return v
}

// SetVolume sets the volume of this player.
// volume must be in between 0 and 1. SetVolume panics otherwise.
func (p *Player) SetVolume(volume float64) {
	p.p.SetVolume(volume)
}

func (p *playerImpl) SetVolume(volume float64) {
	// The condition must be true when volume is NaN.
	if !(0 <= volume && volume <= 1) {
		panic("audio: volume must be in between 0 and 1")
	}

	p.m.Lock()
	p.volume = volume
	p.m.Unlock()
}
