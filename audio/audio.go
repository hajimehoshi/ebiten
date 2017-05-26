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

// Package audio provides audio players. This can be used with or without ebiten package.
//
// The stream format must be 16-bit little endian and 2 channels.
//
// An audio context has a sample rate you can set and all streams you want to play must have the same
// sample rate. However, decoders like audio/vorbis and audio/wav adjust sample rate,
// and you don't have to care about it as long as you use those decoders.
//
// An audio context can generate 'players' (instances of audio.Player),
// and you can play sound by calling Play function of players.
// When multiple players play, mixing is automatically done.
// Note that too many players may cause distortion.
package audio

import (
	"bytes"
	"errors"
	"io"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/internal/sync"
	"github.com/hajimehoshi/oto"
)

type players struct {
	players  map[*Player]struct{}
	seekings map[*Player]struct{}
	sync.RWMutex
}

const (
	channelNum     = 2
	bytesPerSample = 2

	// TODO: This assumes that channelNum is a power of 2.
	mask = ^(channelNum*bytesPerSample - 1)
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (p *players) Read(b []byte) (int, error) {
	p.Lock()
	defer p.Unlock()

	if len(p.players) == 0 {
		l := len(b)
		l &= mask
		copy(b, make([]byte, l))
		return l, nil
	}
	closed := []*Player{}
	l := len(b)
	for player := range p.players {
		if _, ok := p.seekings[player]; ok {
			continue
		}
		if err := player.readToBuffer(l); err == io.EOF {
			closed = append(closed, player)
		} else if err != nil {
			return 0, err
		}
		l = min(player.bufferLength(), l)
	}
	l &= mask
	b16s := [][]int16{}
	for player := range p.players {
		if _, ok := p.seekings[player]; ok {
			continue
		}
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
	for player := range p.players {
		if _, ok := p.seekings[player]; ok {
			continue
		}
		player.proceed(l)
	}
	for _, pl := range closed {
		delete(p.players, pl)
	}
	return l, nil
}

func (p *players) addPlayer(player *Player) {
	p.Lock()
	defer p.Unlock()
	p.players[player] = struct{}{}
}

func (p *players) removePlayer(player *Player) {
	p.Lock()
	defer p.Unlock()
	delete(p.players, player)
}

func (p *players) addSeeking(player *Player) {
	p.Lock()
	defer p.Unlock()
	p.seekings[player] = struct{}{}
}

func (p *players) removeSeeking(player *Player) {
	p.Lock()
	defer p.Unlock()
	delete(p.seekings, player)
}

func (p *players) hasPlayer(player *Player) bool {
	p.RLock()
	defer p.RUnlock()
	_, ok := p.players[player]
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

// A Context is a current state of audio.
//
// There should be at most one Context object.
// This means only one constant sample rate is valid in your one application.
//
// The typical usage with ebiten package is:
//
//    var audioContext *audio.Context
//
//    func update(screen *ebiten.Image) error {
//        // Update updates the audio stream by 1/60 [sec].
//        if err := audioContext.Update(); err != nil {
//            return err
//        }
//        // ...
//    }
//
//    func main() {
//        audioContext, err = audio.NewContext(sampleRate)
//        if err != nil {
//            panic(err)
//        }
//        ebiten.Run(run, update, 320, 240, 2, "Audio test")
//    }
//
// This is 'sync mode' in that game's (logical) time and audio time are synchronized.
// You can also call Update independently from the game loop as 'async mode'.
// In this case, audio goes on even when the game stops e.g. by diactivating the screen.
type Context struct {
	players      *players
	driver       *oto.Player
	sampleRate   int
	frames       int64
	writtenBytes int64
}

var (
	theContext     *Context
	theContextLock sync.Mutex
)

// NewContext creates a new audio context with the given sample rate (e.g. 44100).
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
		players:  map[*Player]struct{}{},
		seekings: map[*Player]struct{}{},
	}
	return c, nil

}

// Update proceeds the inner (logical) time of the context by 1/60 second.
//
// This is expected to be called in the game's updating function (sync mode)
// or an independent goroutine with timers (async mode).
// In sync mode, the game logical time syncs the audio logical time and
// you will find audio stops when the game stops e.g. when the window is deactivated.
// In async mode, the audio never stops even when the game stops.
//
// Update returns error when IO error occurs in the underlying IO object.
func (c *Context) Update() error {
	// Initialize c.driver lazily to enable calling NewContext in an 'init' function.
	// Accessing driver functions requires the environment to be already initialized,
	// but if Ebiten is used for a shared library, the timing when init functions are called
	// is unexpectable.
	// e.g. a variable for JVM on Android might not be set.
	if c.driver == nil {
		// TODO: Rename this other than player
		p, err := oto.NewPlayer(c.sampleRate, channelNum, bytesPerSample)
		c.driver = p
		if err != nil {
			return err
		}
	}
	c.frames++
	bytesPerFrame := c.sampleRate * bytesPerSample * channelNum / ebiten.FPS
	l := (c.frames * int64(bytesPerFrame)) - c.writtenBytes
	l &= mask
	c.writtenBytes += l
	buf := make([]byte, l)
	n, err := io.ReadFull(c.players, buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return c.driver.Close()
	}
	_, err = c.driver.Write(buf)
	if err == io.EOF {
		return c.driver.Close()
	}
	if err != nil {
		return err
	}
	return nil
}

// SampleRate returns the sample rate.
// All audio source must have the same sample rate.
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
	return &bytesReadSeekCloser{bytes.NewReader(b)}
}

// Player is an audio player which has one stream.
type Player struct {
	players    *players
	src        ReadSeekCloser
	buf        []byte
	sampleRate int
	pos        int64
	volume     float64
}

// NewPlayer creates a new player with the given stream.
//
// src's format must be linear PCM (16bits little endian, 2 channel stereo)
// without a header (e.g. RIFF header).
// The sample rate must be same as that of the audio context.
//
// Note that the given src can't be shared with other Players.
//
// NewPlayer tries to rewind src by calling Seek to get the current position.
// NewPlayer returns error when the Seek returns error.
func NewPlayer(context *Context, src ReadSeekCloser) (*Player, error) {
	if context.players.hasSource(src) {
		return nil, errors.New("audio: src cannot be shared with another Player")
	}
	p := &Player{
		players:    context.players,
		src:        src,
		sampleRate: context.sampleRate,
		buf:        []byte{},
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
// NewPlayerFromBytes returns error in the same situation of NewPlayer.
func NewPlayerFromBytes(context *Context, src []byte) (*Player, error) {
	b := BytesReadSeekCloser(src)
	return NewPlayer(context, b)
}

// Close closes the stream. Ths source stream passed by NewPlayer will also be closed.
//
// When closing, the stream owned by the player will also be closed by calling its Close.
//
// Close returns error when closing the source returns error.
func (p *Player) Close() error {
	p.players.removePlayer(p)
	runtime.SetFinalizer(p, nil)
	return p.src.Close()
}

func (p *Player) readToBuffer(length int) error {
	bb := make([]byte, length)
	n, err := p.src.Read(bb)
	if 0 < n {
		p.buf = append(p.buf, bb[:n]...)
	}
	return err
}

func (p *Player) bufferToInt16(lengthInBytes int) []int16 {
	r := make([]int16, lengthInBytes/2)
	for i := 0; i < lengthInBytes/2; i++ {
		r[i] = int16(p.buf[2*i]) | (int16(p.buf[2*i+1]) << 8)
		r[i] = int16(float64(r[i]) * p.volume)
	}
	return r
}

func (p *Player) proceed(length int) {
	p.buf = p.buf[length:]
	p.pos += int64(length)
}

func (p *Player) bufferLength() int {
	return len(p.buf)
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
// Rewind returns error when seeking the source returns error.
func (p *Player) Rewind() error {
	return p.Seek(0)
}

// Seek seeks the position with the given offset.
//
// Seek returns error when seeking the source returns error.
func (p *Player) Seek(offset time.Duration) error {
	p.players.addSeeking(p)
	defer p.players.removeSeeking(p)
	o := int64(offset) * bytesPerSample * channelNum * int64(p.sampleRate) / int64(time.Second)
	o &= mask
	p.buf = []byte{}
	pos, err := p.src.Seek(o, io.SeekStart)
	if err != nil {
		return err
	}
	p.pos = pos
	return nil
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
	sample := p.pos / bytesPerSample / channelNum
	return time.Duration(sample) * time.Second / time.Duration(p.sampleRate)
}

// Volume returns the current volume of this player [0-1].
func (p *Player) Volume() float64 {
	return p.volume
}

// SetVolume sets the volume of this player.
// volume must be in between 0 and 1. This function panics otherwise.
func (p *Player) SetVolume(volume float64) {
	// The condition must be true when volume is NaN.
	if !(0 <= volume && volume <= 1) {
		panic("audio: volume must be in between 0 and 1")
	}
	p.volume = volume
}
