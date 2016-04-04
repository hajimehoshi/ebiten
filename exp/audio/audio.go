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

package audio

import (
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten"
)

type mixingStream struct {
	sampleRate   int
	writtenBytes int
	frames       int
	players      map[*Player]struct{}
	sync.Mutex
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

const (
	channelNum     = 2
	bytesPerSample = 2
	bitsPerSample  = bytesPerSample * 8

	// TODO: This assumes that channelNum is a power of 2.
	mask = ^(channelNum*bytesPerSample - 1)
)

func (s *mixingStream) Read(b []byte) (int, error) {
	s.Lock()
	defer s.Unlock()

	bytesPerFrame := s.sampleRate * bytesPerSample * channelNum / ebiten.FPS
	x := s.frames*bytesPerFrame + len(b)
	if x <= s.writtenBytes {
		return 0, nil
	}

	if len(s.players) == 0 {
		l := min(len(b), x-s.writtenBytes)
		l &= mask
		copy(b, make([]byte, l))
		s.writtenBytes += l
		return l, nil
	}
	closed := []*Player{}
	l := len(b)
	for p := range s.players {
		_, err := p.readToBuffer(l)
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
	s.writtenBytes += l
	return l, nil
}

func (s *mixingStream) update() {
	s.Lock()
	defer s.Unlock()
	s.frames++
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
	s.Lock()
	defer s.Unlock()
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
	s.Lock()
	defer s.Unlock()
	sample := player.pos / bytesPerSample / channelNum
	return time.Duration(sample) * time.Second / time.Duration(s.sampleRate)
}

// TODO: Enable to specify the format like Mono8?

type Context struct {
	sampleRate int
	stream     *mixingStream
	errorCh    chan error
}

func NewContext(sampleRate int) (*Context, error) {
	// TODO: Panic if one context exists.
	c := &Context{
		sampleRate: sampleRate,
		errorCh:    make(chan error),
	}
	c.stream = &mixingStream{
		sampleRate: sampleRate,
		players:    map[*Player]struct{}{},
	}
	p, err := newPlayer(c.stream, c.sampleRate)
	if err != nil {
		return nil, err
	}
	go func() {
		// TODO: Is it OK to close asap?
		defer p.close()
		for {
			err := p.proceed()
			if err == io.EOF {
				break
			}
			if err != nil {
				c.errorCh <- err
				return
			}
			time.Sleep(1 * time.Millisecond)
		}
	}()
	return c, nil
}

// Update proceeds the inner (logical) time of the context by 1/60 second.
// This is expected to be called in the game's updating function (sync mode)
// or an independent goroutine with timers (unsync mode).
// In sync mode, the game logical time syncs the audio logical time and
// you will find audio stops when the game stops e.g. when the window is deactivated.
// In unsync mode, the audio never stops even when the game stops.
func (c *Context) Update() error {
	select {
	case err := <-c.errorCh:
		return err
	default:
	}
	c.stream.update()
	return nil
}

// SampleRate returns the sample rate.
// All audio source must have the same sample rate.
func (c *Context) SampleRate() int {
	return c.sampleRate
}

type ReadSeekCloser interface {
	io.ReadSeeker
	io.Closer
}

type Player struct {
	stream *mixingStream
	src    ReadSeekCloser
	buf    []byte
	pos    int64
	volume float64
}

// NewPlayer creates a new player with the given data to the given channel.
// The given data is queued to the end of the buffer.
// This may not be played immediately when data already exists in the buffer.
//
// src's format must be linear PCM (16bits, 2 channel stereo, little endian)
// without a header (e.g. RIFF header).
func (c *Context) NewPlayer(src ReadSeekCloser) (*Player, error) {
	return c.stream.newPlayer(src)
}

func (p *Player) Close() error {
	return p.stream.closePlayer(p)
}

func (p *Player) readToBuffer(length int) (int, error) {
	bb := make([]byte, length)
	n, err := p.src.Read(bb)
	if 0 < n {
		p.buf = append(p.buf, bb[:n]...)
	}
	return n, err
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

func (p *Player) Play() error {
	p.stream.addPlayer(p)
	return nil
}

func (p *Player) IsPlaying() bool {
	return p.stream.hasPlayer(p)
}

func (p *Player) Rewind() error {
	return p.Seek(0)
}

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

func (p *Player) Pause() error {
	p.stream.removePlayer(p)
	return nil
}

func (p *Player) Current() time.Duration {
	return p.stream.playerCurrent(p)
}

func (p *Player) Volume() float64 {
	return p.volume
}

// SetVolume sets the volume.
// volume must be in between 0 and 1. This function panics otherwise.
func (p *Player) SetVolume(volume float64) {
	// The condition must be true when volume is NaN.
	if !(0 <= volume && volume <= 1) {
		panic("audio: volume must be in between 0 and 1")
	}
	p.volume = volume
}

// TODO: Panning
