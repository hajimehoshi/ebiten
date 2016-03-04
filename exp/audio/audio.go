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
	"fmt"
	"io"
	"sync"
)

// TODO: In JavaScript, mixing should be done by WebAudio for performance.
type mixedPlayersStream struct {
	context *Context
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *mixedPlayersStream) Read(b []byte) (int, error) {
	s.context.Lock()
	defer s.context.Unlock()

	l := len(b)
	if len(s.context.players) == 0 {
		copy(b, make([]byte, l))
		return l, nil
	}
	closed := []*Player{}
	for p := range s.context.players {
		ll := l - len(p.buf)
		if ll <= 0 {
			continue
		}
		b := make([]byte, ll)
		n, err := p.src.Read(b)
		if 0 < n {
			p.buf = append(p.buf, b[:n]...)
		}
		if err == io.EOF {
			if len(p.buf) < l {
				p.buf = append(p.buf, make([]byte, l-len(p.buf))...)
			}
			closed = append(closed, p)
		}
	}
	resultLen := l
	for p := range s.context.players {
		resultLen = min(len(p.buf)/2*2, resultLen)
	}
	for i := 0; i < resultLen/2; i++ {
		x := 0
		for p := range s.context.players {
			x += int(int16(p.buf[2*i]) | (int16(p.buf[2*i+1]) << 8))
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
	for p := range s.context.players {
		p.buf = p.buf[resultLen:]
	}
	for _, p := range closed {
		delete(s.context.players, p)
	}
	return resultLen, nil
}

// TODO: Enable to specify the format like Mono8?

type Context struct {
	sampleRate  int
	stream      *mixedPlayersStream
	innerPlayer *player
	players     map[*Player]struct{}
	sync.Mutex
}

func NewContext(sampleRate int) *Context {
	// TODO: Panic if one context exists.
	c := &Context{
		sampleRate: sampleRate,
		players:    map[*Player]struct{}{},
	}
	c.stream = &mixedPlayersStream{c}
	var err error
	c.innerPlayer, err = newPlayer(c.stream, c.sampleRate)
	if err != nil {
		panic(fmt.Sprintf("audio: NewContext error: %v", err))
	}
	c.innerPlayer.play()
	return c
}

type Player struct {
	context *Context
	src     io.ReadSeeker
	buf     []byte
}

// NewPlayer creates a new player with the given data to the given channel.
// The given data is queued to the end of the buffer.
// This may not be played immediately when data already exists in the buffer.
//
// src's format must be linear PCM (16bits, 2 channel stereo, little endian)
// without a header (e.g. RIFF header).
//
// TODO: Pass sample rate and num of channels.
func (c *Context) NewPlayer(src io.ReadSeeker) (*Player, error) {
	c.Lock()
	defer c.Unlock()

	p := &Player{
		context: c,
		src:     src,
		buf:     []byte{},
	}
	return p, nil
}

func (p *Player) Play() error {
	p.context.Lock()
	defer p.context.Unlock()

	p.context.players[p] = struct{}{}
	return nil
}

// TODO: IsPlaying
// TODO: Stop
// TODO: Seek

func (p *Player) Pause() error {
	p.context.Lock()
	defer p.context.Unlock()

	delete(p.context.players, p)
	return nil
}
