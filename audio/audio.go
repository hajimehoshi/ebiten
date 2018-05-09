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
// Ebiten's game progress tries to synchronizes with audio progress, but
// delay can happen if, e.g., decoding an audio source takes long.
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

	"github.com/hajimehoshi/oto"

	"github.com/hajimehoshi/ebiten/internal/clock"
	"github.com/hajimehoshi/ebiten/internal/hooks"
	"github.com/hajimehoshi/ebiten/internal/web"
)

type players struct {
	players map[*Player]struct{}
	sync.RWMutex
}

const (
	channelNum     = 2
	bytesPerSample = 2

	// TODO: This assumes that channelNum is a power of 2.
	mask = ^(channelNum*bytesPerSample - 1)
)

func (p *players) Read(b []byte) (int, error) {
	p.Lock()
	defer p.Unlock()

	if len(p.players) == 0 {
		l := len(b)
		l &= mask
		copy(b, make([]byte, l))
		return l, nil
	}

	l := len(b)
	l &= mask

	b16s := [][]int16{}
	for player := range p.players {
		buf, err := player.bufferToInt16(l)
		if err != nil {
			return 0, err
		}
		b16s = append(b16s, buf)
	}
	for i := 0; i < l/2; i++ {
		x := 0
		for _, b16 := range b16s {
			x += int(b16[i])
		}
		if x > (1<<15)-1 {
			x = (1 << 15) - 1
		}
		if x < -(1 << 15) {
			x = -(1 << 15)
		}
		b[2*i] = byte(x)
		b[2*i+1] = byte(x >> 8)
	}

	closed := []*Player{}
	for player := range p.players {
		if player.eof() {
			closed = append(closed, player)
		}
	}
	for _, player := range closed {
		delete(p.players, player)
	}

	return l, nil
}

func (p *players) addPlayer(player *Player) {
	p.Lock()
	p.players[player] = struct{}{}
	p.Unlock()
}

func (p *players) removePlayer(player *Player) {
	p.Lock()
	delete(p.players, player)
	p.Unlock()
}

func (p *players) hasPlayer(player *Player) bool {
	p.RLock()
	_, ok := p.players[player]
	p.RUnlock()
	return ok
}

func (p *players) hasSource(src io.ReadCloser) bool {
	p.RLock()
	defer p.RUnlock()
	for player := range p.players {
		if player.src == src {
			return true
		}
	}
	return false
}

// A Context represents a current state of audio.
//
// At most one Context object can exist in one process.
// This means only one constant sample rate is valid in your one application.
//
// For a typical usage example, see examples/wav/main.go.
type Context struct {
	players    *players
	initCh     chan struct{}
	initedCh   chan struct{}
	pingCount  int
	sampleRate int
	err        error

	m sync.Mutex
}

var (
	theContext     *Context
	theContextLock sync.Mutex
)

func init() {
	hooks.AppendHookOnBeforeUpdate(func() error {
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
}

// NewContext creates a new audio context with the given sample rate.
//
// The sample rate is also used for decoding MP3 with audio/mp3 package
// or other formats as the target sample rate.
//
// sampleRate should be 44100 or 48000.
// Other values might not work.
// For example, 22050 causes error on Safari when decoding MP3.
//
// Error returned by NewContext is always nil as of 1.5.0-alpha.
//
// NewContext panics when an audio context is already created.
func NewContext(sampleRate int) (*Context, error) {
	theContextLock.Lock()
	defer theContextLock.Unlock()
	if theContext != nil {
		panic("audio: context is already created")
	}
	c := &Context{
		sampleRate: sampleRate,
	}
	theContext = c
	c.players = &players{
		players: map[*Player]struct{}{},
	}

	go c.loop()

	return c, nil
}

// CurrentContext returns the current context or nil if there is no context.
func CurrentContext() *Context {
	theContextLock.Lock()
	c := theContext
	theContextLock.Unlock()
	return c
}

func (c *Context) ping() {
	if c.initCh != nil {
		close(c.initCh)
		c.initCh = nil
	}
	<-c.initedCh

	c.m.Lock()
	c.pingCount = 5
	c.m.Unlock()
}

func (c *Context) loop() {
	c.initCh = make(chan struct{})
	c.initedCh = make(chan struct{})

	// Copy the channel since c.initCh can be set as nil after clock.RegisterPing.
	initCh := c.initCh

	clock.RegisterPing(c.ping)

	// Initialize oto.Player lazily to enable calling NewContext in an 'init' function.
	// Accessing oto.Player functions requires the environment to be already initialized,
	// but if Ebiten is used for a shared library, the timing when init functions are called
	// is unexpectable.
	// e.g. a variable for JVM on Android might not be set.
	<-initCh

	// On most desktop environments, 4096 [bytes] is enough
	// but there are some known environment that is too short (e.g. Windows on Parallels, iOS).
	p, err := oto.NewPlayer(c.sampleRate, channelNum, bytesPerSample, 8192)
	if err != nil {
		c.err = err
		return
	}
	defer p.Close()

	close(c.initedCh)

	bytesPerFrame := c.sampleRate * bytesPerSample * channelNum / clock.FPS
	written := int64(0)
	prevWritten := int64(0)
	for {
		c.m.Lock()
		if c.pingCount == 0 {
			c.m.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}
		c.pingCount--
		c.m.Unlock()

		const n = 2048
		if _, err := io.CopyN(p, c.players, n); err != nil {
			c.err = err
			return
		}

		written += int64(n)
		fs := written/int64(bytesPerFrame) - prevWritten/int64(bytesPerFrame)
		clock.ProceedAudioTimer(fs)
		prevWritten = written
	}
}

// Update is deprecated as of 1.6.0-alpha.
//
// As of 1.6.0-alpha, Update always returns nil and does nothing related to updating the state.
// You don't have to call this function any longer.
// The internal audio error is returned at ebiten.Run instead.
func (c *Context) Update() error {
	return nil
}

// SampleRate returns the sample rate.
func (c *Context) SampleRate() int {
	return c.sampleRate
}

// ReadSeekCloser is an io.ReadSeeker and io.Closer.
type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

type bytesReadSeekCloser struct {
	reader *bytes.Reader
}

func (b *bytesReadSeekCloser) Read(buf []byte) (int, error) {
	return b.reader.Read(buf)
}

func (b *bytesReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	return b.reader.Seek(offset, whence)
}

func (b *bytesReadSeekCloser) Close() error {
	b.reader = nil
	return nil
}

// BytesReadSeekCloser creates ReadSeekCloser from bytes.
func BytesReadSeekCloser(b []byte) ReadSeekCloser {
	return &bytesReadSeekCloser{reader: bytes.NewReader(b)}
}

// Player is an audio player which has one stream.
type Player struct {
	players    *players
	src        io.ReadCloser
	srcEOF     bool
	sampleRate int

	buf    []byte
	pos    int64
	volume float64

	closeCh         chan struct{}
	closedCh        chan struct{}
	readLoopEndedCh chan struct{}
	seekCh          chan seekArgs
	seekedCh        chan error
	proceedCh       chan []int16
	proceededCh     chan proceededValues
	syncCh          chan func()
}

type seekArgs struct {
	offset int64
	whence int
}

type proceededValues struct {
	buf []int16
	err error
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
func NewPlayer(context *Context, src io.ReadCloser) (*Player, error) {
	if context.players.hasSource(src) {
		return nil, errors.New("audio: src cannot be shared with another Player")
	}
	p := &Player{
		players:         context.players,
		src:             src,
		sampleRate:      context.sampleRate,
		buf:             []byte{},
		volume:          1,
		closeCh:         make(chan struct{}),
		closedCh:        make(chan struct{}),
		readLoopEndedCh: make(chan struct{}),
		seekCh:          make(chan seekArgs),
		seekedCh:        make(chan error),
		proceedCh:       make(chan []int16),
		proceededCh:     make(chan proceededValues),
		syncCh:          make(chan func()),
	}
	if seeker, ok := p.src.(io.Seeker); ok {
		// Get the current position of the source.
		pos, err := seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}
		p.pos = pos
	}
	runtime.SetFinalizer(p, (*Player).Close)

	go func() {
		p.readLoop()
	}()
	return p, nil
}

// NewPlayerFromBytes creates a new player with the given bytes.
//
// As opposed to NewPlayer, you don't have to care if src is already used by another player or not.
// src can be shared by multiple players.
//
// The format of src should be same as noted at NewPlayer.
//
// NewPlayerFromBytes's error is always nil as of 1.5.0-alpha.
func NewPlayerFromBytes(context *Context, src []byte) (*Player, error) {
	b := BytesReadSeekCloser(src)
	p, err := NewPlayer(context, b)
	if err != nil {
		// Errors should never happen.
		panic(err)
	}
	return p, nil
}

// Close closes the stream.
//
// When closing, the stream owned by the player will also be closed by calling its Close.
// This means that the source stream passed via NewPlayer will also be closed.
//
// Close returns error when closing the source returns error.
func (p *Player) Close() error {
	runtime.SetFinalizer(p, nil)
	p.players.removePlayer(p)

	select {
	case p.closeCh <- struct{}{}:
		<-p.closedCh
		return nil
	case <-p.readLoopEndedCh:
		return fmt.Errorf("audio: the player is already closed")
	}
}

func (p *Player) bufferToInt16(lengthInBytes int) ([]int16, error) {
	select {
	case p.proceedCh <- make([]int16, lengthInBytes/2):
		r := <-p.proceededCh
		return r.buf, r.err
	case <-p.readLoopEndedCh:
		return nil, fmt.Errorf("audio: the player is already closed")
	}
}

// Play plays the stream.
//
// Play always returns nil.
func (p *Player) Play() error {
	p.players.addPlayer(p)
	return nil
}

func (p *Player) readLoop() {
	defer func() {
		// Note: the error is ignored
		p.src.Close()
		// Receiving from a closed channel returns quickly
		// i.e. `case <-p.readLoopEndedCh:` can check if this loops is ended.
		close(p.readLoopEndedCh)
	}()

	timer := time.NewTimer(0)
	timerCh := timer.C
	var readErr error
	for {
		select {
		case <-p.closeCh:
			p.closedCh <- struct{}{}
			return

		case s := <-p.seekCh:
			seeker, ok := p.src.(io.Seeker)
			if !ok {
				panic("not reached")
			}
			pos, err := seeker.Seek(s.offset, s.whence)
			p.buf = nil
			p.pos = pos
			p.srcEOF = false
			p.seekedCh <- err
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(time.Millisecond)
			timerCh = timer.C
			break

		case <-timerCh:
			// If the buffer has 1 second, that's enough.
			if len(p.buf) >= p.sampleRate*bytesPerSample*channelNum {
				if timer != nil {
					timer.Stop()
				}
				timer = time.NewTimer(100 * time.Millisecond)
				timerCh = timer.C
				break
			}

			// Try to read the buffer for 1/60[s].
			s := 60
			if web.IsAndroidChrome() {
				s = 10
			} else if web.IsBrowser() {
				s = 20
			}
			l := p.sampleRate * bytesPerSample * channelNum / s
			l &= mask
			buf := make([]byte, l)
			n, err := p.src.Read(buf)

			p.buf = append(p.buf, buf[:n]...)
			if err == io.EOF {
				p.srcEOF = true
			}
			if p.srcEOF && len(p.buf) == 0 {
				if timer != nil {
					timer.Stop()
				}
				timer = nil
				timerCh = nil
				break
			}
			if err != nil && err != io.EOF {
				readErr = err
				if timer != nil {
					timer.Stop()
				}
				timer = nil
				timerCh = nil
				break
			}
			if timer != nil {
				timer.Stop()
			}
			if web.IsBrowser() {
				timer = time.NewTimer(10 * time.Millisecond)
			} else {
				timer = time.NewTimer(time.Millisecond)
			}
			timerCh = timer.C

		case buf := <-p.proceedCh:
			if readErr != nil {
				p.proceededCh <- proceededValues{buf, readErr}
				return
			}

			lengthInBytes := len(buf) * 2
			l := lengthInBytes

			if len(p.buf) < lengthInBytes && !p.srcEOF {
				p.proceededCh <- proceededValues{buf, nil}
				break
			}
			if l > len(p.buf) {
				l = len(p.buf)
			}
			for i := 0; i < l/2; i++ {
				buf[i] = int16(p.buf[2*i]) | (int16(p.buf[2*i+1]) << 8)
				buf[i] = int16(float64(buf[i]) * p.volume)
			}
			p.pos += int64(l)
			p.buf = p.buf[l:]

			p.proceededCh <- proceededValues{buf, nil}

		case f := <-p.syncCh:
			f()
		}
	}
}

func (p *Player) sync(f func()) bool {
	ch := make(chan struct{})
	ff := func() {
		f()
		close(ch)
	}
	select {
	case p.syncCh <- ff:
		<-ch
		return true
	case <-p.readLoopEndedCh:
		return false
	}
}

func (p *Player) eof() bool {
	r := false
	p.sync(func() {
		r = p.srcEOF && len(p.buf) == 0
	})
	return r
}

// IsPlaying returns boolean indicating whether the player is playing.
func (p *Player) IsPlaying() bool {
	return p.players.hasPlayer(p)
}

// Rewind rewinds the current position to the start.
//
// The passed source to NewPlayer must be io.Seeker, or Rewind panics.
//
// Rewind returns error when seeking the source stream returns error.
func (p *Player) Rewind() error {
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
	if _, ok := p.src.(io.Seeker); !ok {
		panic("audio: player to be sought must be io.Seeker")
	}
	o := int64(offset) * bytesPerSample * channelNum * int64(p.sampleRate) / int64(time.Second)
	o &= mask
	select {
	case p.seekCh <- seekArgs{o, io.SeekStart}:
		return <-p.seekedCh
	case <-p.readLoopEndedCh:
		return fmt.Errorf("audio: the player is already closed")
	}
}

// Pause pauses the playing.
//
// Pause always returns nil.
func (p *Player) Pause() error {
	p.players.removePlayer(p)
	return nil
}

// Current returns the current position.
func (p *Player) Current() time.Duration {
	sample := int64(0)
	p.sync(func() {
		sample = p.pos / bytesPerSample / channelNum
	})
	return time.Duration(sample) * time.Second / time.Duration(p.sampleRate)
}

// Volume returns the current volume of this player [0-1].
func (p *Player) Volume() float64 {
	v := 0.0
	p.sync(func() {
		v = p.volume
	})
	return v
}

// SetVolume sets the volume of this player.
// volume must be in between 0 and 1. This function panics otherwise.
func (p *Player) SetVolume(volume float64) {
	// The condition must be true when volume is NaN.
	if !(0 <= volume && volume <= 1) {
		panic("audio: volume must be in between 0 and 1")
	}

	p.sync(func() {
		p.volume = volume
	})
}
