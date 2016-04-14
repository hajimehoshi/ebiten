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
// sample rate.
//
// An audio context can generate 'players' (instances of audio.Player),
// and you can play sound by calling Play function of players.
// When multiple players play, mixing is automatically done.
// Note that too many players may cause distortion.
package audio

import (
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/exp/audio/internal/driver"
)

type mixingStream struct {
	sampleRate int
	players    map[*Player]struct{}

	// Note that Read (and other methods) need to be concurrent safe
	// because Read is called from another groutine (see NewContext).
	sync.RWMutex
}

const (
	channelNum     = 2
	bytesPerSample = 2

	// TODO: This assumes that channelNum is a power of 2.
	mask = ^(channelNum*bytesPerSample - 1)
)

func (s *mixingStream) SampleRate() int {
	return s.sampleRate
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *mixingStream) Read(b []byte) (int, error) {
	s.Lock()
	defer s.Unlock()

	if len(s.players) == 0 {
		l := len(b)
		l &= mask
		copy(b, make([]byte, l))
		return l, nil
	}
	closed := []*Player{}
	l := len(b)
	for p := range s.players {
		err := p.readToBuffer(l)
		if err == io.EOF {
			closed = append(closed, p)
		} else if err != nil {
			return 0, err
		}
		l = min(p.bufferLength(), l)
	}
	l &= mask
	b16s := [][]int16{}
	for p := range s.players {
		b16s = append(b16s, p.bufferToInt16(l))
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
	for p := range s.players {
		p.proceed(l)
	}
	for _, p := range closed {
		delete(s.players, p)
	}
	return l, nil
}

func (s *mixingStream) newPlayer(src ReadSeekCloser) (*Player, error) {
	s.Lock()
	defer s.Unlock()
	p := &Player{
		stream: s,
		src:    src,
		buf:    []byte{},
		volume: 1,
	}
	// Get the current position of the source.
	pos, err := p.src.Seek(0, 1)
	if err != nil {
		return nil, err
	}
	p.pos = pos
	runtime.SetFinalizer(p, (*Player).Close)
	return p, nil
}

func (s *mixingStream) closePlayer(player *Player) error {
	s.Lock()
	defer s.Unlock()
	runtime.SetFinalizer(player, nil)
	return player.src.Close()
}

func (s *mixingStream) addPlayer(player *Player) {
	s.Lock()
	defer s.Unlock()
	s.players[player] = struct{}{}
}

func (s *mixingStream) removePlayer(player *Player) {
	s.Lock()
	defer s.Unlock()
	delete(s.players, player)
}

func (s *mixingStream) hasPlayer(player *Player) bool {
	s.RLock()
	defer s.RUnlock()
	_, ok := s.players[player]
	return ok
}

func (s *mixingStream) seekPlayer(player *Player, offset time.Duration) error {
	s.Lock()
	defer s.Unlock()
	o := int64(offset) * bytesPerSample * channelNum * int64(s.sampleRate) / int64(time.Second)
	o &= mask
	return player.seek(o)
}

func (s *mixingStream) playerCurrent(player *Player) time.Duration {
	s.RLock()
	defer s.RUnlock()
	sample := player.pos / bytesPerSample / channelNum
	return time.Duration(sample) * time.Second / time.Duration(s.sampleRate)
}

// TODO: Enable to specify the format like Mono8?

// A Context is a current state of audio.
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
	stream       *mixingStream
	driver       *driver.Player
	frames       int
	writtenBytes int
}

// NewContext creates a new audio context with the given sample rate (e.g. 44100).
func NewContext(sampleRate int) (*Context, error) {
	// TODO: Panic if one context exists.
	c := &Context{}
	c.stream = &mixingStream{
		sampleRate: sampleRate,
		players:    map[*Player]struct{}{},
	}
	// TODO: Rename this other than player
	p, err := driver.NewPlayer(sampleRate, channelNum, bytesPerSample)
	c.driver = p
	if err != nil {
		return nil, err
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
func (c *Context) Update() error {
	c.frames++
	bytesPerFrame := c.stream.sampleRate * bytesPerSample * channelNum / ebiten.FPS
	l := (c.frames * bytesPerFrame) - c.writtenBytes
	l &= mask
	c.writtenBytes += l
	buf := make([]byte, l)
	n, err := io.ReadFull(c.stream, buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return c.driver.Close()
	}
	err = c.driver.Proceed(buf)
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
	return c.stream.SampleRate()
}

// ReadSeekCloser is an io.ReadSeeker and io.Closer.
type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

// Player is an audio player which has one stream.
type Player struct {
	stream *mixingStream
	src    ReadSeekCloser
	buf    []byte
	pos    int64
	volume float64
}

// NewPlayer creates a new player with the given stream.
//
// src's format must be linear PCM (16bits little endian, 2 channel stereo)
// without a header (e.g. RIFF header).
// The sample rate must be same as that of the audio context.
func (c *Context) NewPlayer(src ReadSeekCloser) (*Player, error) {
	return c.stream.newPlayer(src)
}

// Close closes the stream. Ths source stream passed by NewPlayer will also be closed.
func (p *Player) Close() error {
	return p.stream.closePlayer(p)
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
func (p *Player) Play() error {
	p.stream.addPlayer(p)
	return nil
}

// IsPlaying returns boolean indicating whether the player is playing.
func (p *Player) IsPlaying() bool {
	return p.stream.hasPlayer(p)
}

// Rewind rewinds the current position to the start.
func (p *Player) Rewind() error {
	return p.Seek(0)
}

// Seek seeks the position with the given offset.
func (p *Player) Seek(offset time.Duration) error {
	return p.stream.seekPlayer(p, offset)
}

func (p *Player) seek(offset int64) error {
	p.buf = []byte{}
	pos, err := p.src.Seek(offset, 0)
	if err != nil {
		return err
	}
	p.pos = pos
	return nil
}

// Pause pauses the playing.
func (p *Player) Pause() error {
	p.stream.removePlayer(p)
	return nil
}

// Current returns the current position.
func (p *Player) Current() time.Duration {
	return p.stream.playerCurrent(p)
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

// TODO: Panning
