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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *mixedPlayersStream) Read(b []byte) (int, error) {
	s.context.Lock()
	defer s.context.Unlock()

	l := len(b) / 4 * 4
	if len(s.context.players) == 0 {
		copy(b, make([]byte, l))
		return l, nil
	}
	closed := []*Player{}
	bb := make([]byte, max(1024, l))
	ll := l
	for p := range s.context.players {
		n, err := p.src.Read(bb)
		if 0 < n {
			p.buf = append(p.buf, bb[:n]...)
		}
		if err == io.EOF {
			closed = append(closed, p)
			continue
		}
		if err != nil {
			return 0, err
		}
		ll = min(len(p.buf)/4*4, ll)
	}
	for _, p := range closed {
		if len(p.buf) < ll {
			p.buf = append(p.buf, make([]byte, ll-len(p.buf))...)
		}
	}
	for i := 0; i < ll/2; i++ {
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
		p.buf = p.buf[ll:]
	}
	for _, p := range closed {
		delete(s.context.players, p)
	}
	return ll, nil
}

// TODO: Enable to specify the format like Mono8?

type Context struct {
	sampleRate int
	stream     *mixedPlayersStream
	players    map[*Player]struct{}
	sync.Mutex
}

func NewContext(sampleRate int) *Context {
	// TODO: Panic if one context exists.
	c := &Context{
		sampleRate: sampleRate,
		players:    map[*Player]struct{}{},
	}
	c.stream = &mixedPlayersStream{c}
	if err := startPlaying(c.stream, c.sampleRate); err != nil {
		panic(fmt.Sprintf("audio: NewContext error: %v", err))
	}
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
