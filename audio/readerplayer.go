// Copyright 2021 The Ebiten Authors
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
	"sync"
	"time"
)

// readerDriver represents a driver using io.ReadClosers.
type readerDriver interface {
	NewPlayer(io.Reader) readerDriverPlayer
	io.Closer
}

type readerDriverPlayer interface {
	Pause()
	Play()
	IsPlaying() bool
	Reset()
	Volume() float64
	SetVolume(volume float64)
	io.Closer
}

type readerPlayerFactory struct {
	driver readerDriver
}

func newReaderPlayerFactory(sampleRate int) *readerPlayerFactory {
	return &readerPlayerFactory{
		driver: newReaderDriverImpl(sampleRate),
	}
	// TODO: Consider the hooks.
}

type readerPlayer struct {
	context *Context
	player  readerDriverPlayer
	src     *timeStream
	m       sync.Mutex
}

func (c *readerPlayerFactory) newPlayerImpl(context *Context, src io.Reader) (playerImpl, error) {
	s, err := newTimeStream(src, context.SampleRate())
	if err != nil {
		return nil, err
	}

	p := &readerPlayer{
		context: context,
		player:  c.driver.NewPlayer(s),
		src:     s,
	}
	return p, nil
}

func (p *readerPlayer) Play() {
	p.m.Lock()
	defer p.m.Unlock()

	p.player.Play()
	p.context.addPlayer(p)
}

func (p *readerPlayer) Pause() {
	p.m.Lock()
	defer p.m.Unlock()

	p.player.Pause()
}

func (p *readerPlayer) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()

	return p.player.IsPlaying()
}

func (p *readerPlayer) Volume() float64 {
	p.m.Lock()
	defer p.m.Unlock()

	return p.player.Volume()
}

func (p *readerPlayer) SetVolume(volume float64) {
	p.m.Lock()
	defer p.m.Unlock()

	p.player.SetVolume(volume)
}

func (p *readerPlayer) Close() error {
	p.m.Lock()
	defer p.m.Unlock()

	p.context.removePlayer(p)
	return p.player.Close()
}

func (p *readerPlayer) Current() time.Duration {
	p.m.Lock()
	defer p.m.Unlock()

	// TODO: Add a new function to readerDriverPlayer and use it.
	return p.src.Current()
}

func (p *readerPlayer) Rewind() error {
	return p.Seek(0)
}

func (p *readerPlayer) Seek(offset time.Duration) error {
	p.m.Lock()
	defer p.m.Unlock()

	p.player.Reset()
	return p.src.Seek(offset)
}

func (p *readerPlayer) source() io.Reader {
	return p.src
}

type timeStream struct {
	r          io.Reader
	sampleRate int
	pos        int64
}

func newTimeStream(r io.Reader, sampleRate int) (*timeStream, error) {
	s := &timeStream{
		r:          r,
		sampleRate: sampleRate,
	}
	if seeker, ok := s.r.(io.Seeker); ok {
		// Get the current position of the source.
		pos, err := seeker.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}
		s.pos = pos
	}
	return s, nil
}

func (s *timeStream) Read(buf []byte) (int, error) {
	n, err := s.r.Read(buf)
	s.pos += int64(n)
	return n, err
}

func (s *timeStream) Seek(offset time.Duration) error {
	o := int64(offset) * bytesPerSample * int64(s.sampleRate) / int64(time.Second)

	// Align the byte position with the samples.
	o -= o % bytesPerSample
	o += s.pos % bytesPerSample

	seeker, ok := s.r.(io.Seeker)
	if !ok {
		panic("audio: the source must be io.Seeker when seeking but not")
	}
	pos, err := seeker.Seek(o, io.SeekStart)
	if err != nil {
		return err
	}

	s.pos = pos
	return nil
}

func (s *timeStream) Current() time.Duration {
	sample := s.pos / bytesPerSample
	return time.Duration(sample) * time.Second / time.Duration(s.sampleRate)
}
