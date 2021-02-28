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
	UnwrittenBufferSize() int64
	io.Closer
}

type readerPlayerFactory struct {
	driver     readerDriver
	sampleRate int
}

func newReaderPlayerFactory(sampleRate int) *readerPlayerFactory {
	return &readerPlayerFactory{
		sampleRate: sampleRate,
	}
	// TODO: Consider the hooks.
}

type readerPlayer struct {
	context *Context
	player  readerDriverPlayer
	src     *timeStream
	factory *readerPlayerFactory
	m       sync.Mutex
}

func (f *readerPlayerFactory) newPlayerImpl(context *Context, src io.Reader) (playerImpl, error) {
	sampleRate := context.SampleRate()
	s, err := newTimeStream(src, sampleRate)
	if err != nil {
		return nil, err
	}

	p := &readerPlayer{
		context: context,
		src:     s,
		factory: f,
	}
	return p, nil
}

func (p *readerPlayer) ensurePlayer() {
	// Initialize the underlying player lazily to enable calling NewContext in an 'init' function.
	// Accessing the underlying player functions requires the environment to be already initialized,
	// but if Ebiten is used for a shared library, the timing when init functions are called
	// is unexpectable.
	// e.g. a variable for JVM on Android might not be set.
	if p.factory.driver == nil {
		p.factory.driver = newReaderDriverImpl(p.factory.sampleRate)
	}
	if p.player == nil {
		p.player = p.factory.driver.NewPlayer(p.src)
	}
	// TODO: If some error happens, call p.context.setError().
}

func (p *readerPlayer) Play() {
	p.m.Lock()
	defer p.m.Unlock()
	p.ensurePlayer()

	p.player.Play()
	p.context.addPlayer(p)
}

func (p *readerPlayer) Pause() {
	p.m.Lock()
	defer p.m.Unlock()
	p.ensurePlayer()

	p.player.Pause()
}

func (p *readerPlayer) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()
	p.ensurePlayer()

	return p.player.IsPlaying()
}

func (p *readerPlayer) Volume() float64 {
	p.m.Lock()
	defer p.m.Unlock()
	p.ensurePlayer()

	return p.player.Volume()
}

func (p *readerPlayer) SetVolume(volume float64) {
	p.m.Lock()
	defer p.m.Unlock()
	p.ensurePlayer()

	p.player.SetVolume(volume)
}

func (p *readerPlayer) Close() error {
	p.m.Lock()
	defer p.m.Unlock()
	p.ensurePlayer()

	p.context.removePlayer(p)
	return p.player.Close()
}

func (p *readerPlayer) Current() time.Duration {
	p.m.Lock()
	defer p.m.Unlock()
	p.ensurePlayer()

	sample := (p.src.Current() - p.player.UnwrittenBufferSize()) / bytesPerSample
	return time.Duration(sample) * time.Second / time.Duration(p.factory.sampleRate)
}

func (p *readerPlayer) Rewind() error {
	return p.Seek(0)
}

func (p *readerPlayer) Seek(offset time.Duration) error {
	p.m.Lock()
	defer p.m.Unlock()
	p.ensurePlayer()

	if p.player.IsPlaying() {
		defer func() {
			p.player.Play()
		}()
	}
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

func (s *timeStream) Current() int64 {
	return s.pos
}
