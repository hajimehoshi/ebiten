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
// Ebiten's game progress always synchronizes with audio progress.
//
// For the simplest example to play sound, see wav package in the examples.
package audio

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/oto"

	"github.com/hajimehoshi/ebiten/internal/audiobinding"
	"github.com/hajimehoshi/ebiten/internal/clock"
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

func (p *players) Read(b []uint8) (int, error) {
	p.Lock()
	defer p.Unlock()

	players := []*Player{}
	for player := range p.players {
		players = append(players, player)
	}
	if len(players) == 0 {
		l := len(b)
		l &= mask
		copy(b, make([]uint8, l))
		return l, nil
	}
	closed := []*Player{}

	for _, player := range players {
		player.resetBufferIfSeeking()
	}

	l := len(b)
	for _, player := range players {
		n, err := player.readToBuffer(l)
		if err == io.EOF {
			closed = append(closed, player)
		} else if err != nil {
			return 0, err
		}
		if n < l {
			l = n
		}
	}
	l &= mask

	b16s := [][]int16{}
	for _, player := range players {
		b16s = append(b16s, player.bufferToInt16(l))
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

	for _, player := range players {
		player.proceed(l)
	}

	for _, pl := range closed {
		delete(p.players, pl)
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

func (p *players) hasSource(src ReadSeekCloser) bool {
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
	players        *players
	initCh         chan struct{}
	initedCh       chan struct{}
	pingCount      int
	sampleRate     int
	frames         int64
	framesReadOnly int64
	writtenBytes   int64
	m              sync.Mutex
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

	// This is a heuristic decision of audio buffer size.
	// On most desktops, 1/30[s] is enough but there are some known environment that is too short (e.g. Windows on Parallels).
	// On browsers, 1/15[s] should work with any sample rate except for Android Chrome.
	// On mobiles, we don't have enough data. For iOS, 1/30[s] is too short and 1/20[s] seems fine. 1/15[s] is safer.
	bufferSize := c.sampleRate * channelNum * bytesPerSample / 15
	if web.IsAndroidChrome() {
		// On Android Chrome, it looks like 9600 * 4 is a sweet spot of the buffer size
		// regardless of the sample rate. This is about 1/5[s] for 48000[Hz].
		bufferSize = 9600 * channelNum * bytesPerSample
	}
	p, err := oto.NewPlayer(c.sampleRate, channelNum, bytesPerSample, bufferSize)
	if err != nil {
		audiobinding.SetError(err)
		return
	}
	defer p.Close()

	close(c.initedCh)

	for {
		c.m.Lock()
		if c.pingCount == 0 {
			c.m.Unlock()
			time.Sleep(10 * time.Millisecond)
			continue
		}
		c.pingCount--
		c.m.Unlock()
		c.frames++
		clock.ProceedPrimaryTimer()
		bytesPerFrame := c.sampleRate * bytesPerSample * channelNum / clock.FPS
		l := (c.frames * int64(bytesPerFrame)) - c.writtenBytes
		l &= mask
		c.writtenBytes += l
		buf := make([]uint8, l)
		if _, err := io.ReadFull(c.players, buf); err != nil {
			audiobinding.SetError(err)
			return
		}
		if _, err = p.Write(buf); err != nil {
			audiobinding.SetError(err)
			return
		}
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

func (b *bytesReadSeekCloser) Read(buf []uint8) (int, error) {
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
func BytesReadSeekCloser(b []uint8) ReadSeekCloser {
	return &bytesReadSeekCloser{reader: bytes.NewReader(b)}
}

type readingResult struct {
	data []uint8
	err  error
}

// Player is an audio player which has one stream.
type Player struct {
	players    *players
	src        ReadSeekCloser
	sampleRate int

	buf    []uint8
	pos    int64
	volume float64

	seeking bool
	nextPos int64

	srcM sync.Mutex
	m    sync.RWMutex
}

// NewPlayer creates a new player with the given stream.
//
// src's format must be linear PCM (16bits little endian, 2 channel stereo)
// without a header (e.g. RIFF header).
// The sample rate must be same as that of the audio context.
//
// Note that the given src can't be shared with other Player objects.
//
// NewPlayer tries to call Seek of src to get the current position.
// NewPlayer returns error when the Seek returns error.
func NewPlayer(context *Context, src ReadSeekCloser) (*Player, error) {
	if context.players.hasSource(src) {
		return nil, errors.New("audio: src cannot be shared with another Player")
	}
	p := &Player{
		players:    context.players,
		src:        src,
		sampleRate: context.sampleRate,
		buf:        []uint8{},
		volume:     1,
	}
	// Get the current position of the source.
	pos, err := p.src.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	p.pos = pos
	runtime.SetFinalizer(p, (*Player).Close)
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
func NewPlayerFromBytes(context *Context, src []uint8) (*Player, error) {
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
	p.players.removePlayer(p)
	runtime.SetFinalizer(p, nil)
	p.srcM.Lock()
	err := p.src.Close()
	p.srcM.Unlock()
	return err
}

func (p *Player) readToBuffer(length int) (int, error) {
	b := make([]uint8, length)
	p.srcM.Lock()
	n, err := p.src.Read(b)
	p.srcM.Unlock()
	if err != nil {
		return 0, err
	}
	p.buf = append(p.buf, b[:n]...)
	return len(p.buf), nil
}

func (p *Player) bufferToInt16(lengthInBytes int) []int16 {
	r := make([]int16, lengthInBytes/2)
	// This function must be called on the same goruotine of readToBuffer.
	p.m.RLock()
	for i := 0; i < lengthInBytes/2; i++ {
		r[i] = int16(p.buf[2*i]) | (int16(p.buf[2*i+1]) << 8)
		r[i] = int16(float64(r[i]) * p.volume)
	}
	p.m.RUnlock()
	return r
}

func (p *Player) proceed(length int) {
	// This function must be called on the same goruotine of readToBuffer.
	p.m.Lock()
	p.buf = p.buf[length:]
	p.pos += int64(length)
	p.m.Unlock()
}

// Play plays the stream.
//
// Play always returns nil.
func (p *Player) Play() error {
	p.players.addPlayer(p)
	return nil
}

// IsPlaying returns boolean indicating whether the player is playing.
func (p *Player) IsPlaying() bool {
	return p.players.hasPlayer(p)
}

// Rewind rewinds the current position to the start.
//
// Rewind returns error when seeking the source stream returns error.
func (p *Player) Rewind() error {
	return p.Seek(0)
}

// Seek seeks the position with the given offset.
//
// Seek returns error when seeking the source stream returns error.
func (p *Player) Seek(offset time.Duration) error {
	o := int64(offset) * bytesPerSample * channelNum * int64(p.sampleRate) / int64(time.Second)
	o &= mask
	p.srcM.Lock()
	pos, err := p.src.Seek(o, io.SeekStart)
	p.srcM.Unlock()
	if err != nil {
		return err
	}
	p.m.Lock()
	p.seeking = true
	p.nextPos = pos
	p.m.Unlock()
	return nil
}

func (p *Player) resetBufferIfSeeking() {
	// This function must be called on the same goruotine of readToBuffer.
	p.m.Lock()
	if p.seeking {
		p.buf = []uint8{}
		p.pos = p.nextPos
		p.seeking = false
	}
	p.m.Unlock()
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
	p.m.RLock()
	sample := p.pos / bytesPerSample / channelNum
	t := time.Duration(sample) * time.Second / time.Duration(p.sampleRate)
	p.m.RUnlock()
	return t
}

// Volume returns the current volume of this player [0-1].
func (p *Player) Volume() float64 {
	p.m.RLock()
	v := p.volume
	p.m.RUnlock()
	return v
}

// SetVolume sets the volume of this player.
// volume must be in between 0 and 1. This function panics otherwise.
func (p *Player) SetVolume(volume float64) {
	p.m.Lock()
	// The condition must be true when volume is NaN.
	if !(0 <= volume && volume <= 1) {
		panic("audio: volume must be in between 0 and 1")
	}
	p.volume = volume
	p.m.Unlock()
}
