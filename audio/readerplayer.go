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
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"
)

// TODO: The term 'buffer' is confusing. Name each buffer with good terms.

// oneBufferSize returns the size of one buffer in the player implementation.
func oneBufferSize(sampleRate int) int {
	return sampleRate * channelNum * bitDepthInBytes / 4
}

// maxBufferSize returns the maximum size of the buffer for the audio source.
// This buffer is used when unreading on pausing the player.
func maxBufferSize(sampleRate int) int {
	// The number of underlying buffers should be 2.
	return oneBufferSize(sampleRate) * 2
}

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
	UnplayedBufferSize() int64
	io.Closer
}

type readerPlayerFactory struct {
	driver     readerDriver
	sampleRate int
}

var readerDriverForTesting readerDriver

func newReaderPlayerFactory(sampleRate int) *readerPlayerFactory {
	f := &readerPlayerFactory{
		sampleRate: sampleRate,
	}
	if readerDriverForTesting != nil {
		f.driver = readerDriverForTesting
	}
	// TODO: Consider the hooks.
	return f
}

type readerPlayer struct {
	context *Context
	player  readerDriverPlayer
	stream  *timeStream
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
		stream:  s,
		factory: f,
	}
	runtime.SetFinalizer(p, (*readerPlayer).Close)
	return p, nil
}

func (p *readerPlayer) ensurePlayer() error {
	// Initialize the underlying player lazily to enable calling NewContext in an 'init' function.
	// Accessing the underlying player functions requires the environment to be already initialized,
	// but if Ebiten is used for a shared library, the timing when init functions are called
	// is unexpectable.
	// e.g. a variable for JVM on Android might not be set.
	if p.factory.driver == nil {
		d, err := newReaderDriverImpl(p.context)
		if err != nil {
			return err
		}
		p.factory.driver = d
	}
	if p.player == nil {
		p.player = p.factory.driver.NewPlayer(p.stream)
	}
	return nil
}

func (p *readerPlayer) Play() {
	p.m.Lock()
	defer p.m.Unlock()
	if err := p.ensurePlayer(); err != nil {
		p.context.setError(err)
		return
	}

	p.player.Play()
	p.context.addPlayer(p)
}

func (p *readerPlayer) Pause() {
	p.m.Lock()
	defer p.m.Unlock()
	if err := p.ensurePlayer(); err != nil {
		p.context.setError(err)
		return
	}

	n := p.player.UnplayedBufferSize()
	p.player.Pause()
	p.stream.Unread(int(n))
	p.context.removePlayer(p)
}

func (p *readerPlayer) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()
	if err := p.ensurePlayer(); err != nil {
		p.context.setError(err)
		return false
	}

	return p.player.IsPlaying()
}

func (p *readerPlayer) Volume() float64 {
	p.m.Lock()
	defer p.m.Unlock()
	if err := p.ensurePlayer(); err != nil {
		p.context.setError(err)
		return 0
	}

	return p.player.Volume()
}

func (p *readerPlayer) SetVolume(volume float64) {
	p.m.Lock()
	defer p.m.Unlock()
	if err := p.ensurePlayer(); err != nil {
		p.context.setError(err)
		return
	}

	p.player.SetVolume(volume)
}

func (p *readerPlayer) Close() error {
	p.m.Lock()
	defer p.m.Unlock()
	runtime.SetFinalizer(p, nil)

	if p.player != nil {
		defer func() {
			p.player = nil
		}()
		p.player.Pause()
		return p.player.Close()
	}
	return nil
}

func (p *readerPlayer) Current() time.Duration {
	p.m.Lock()
	defer p.m.Unlock()
	if err := p.ensurePlayer(); err != nil {
		p.context.setError(err)
		return 0
	}

	sample := (p.stream.Current() - p.player.UnplayedBufferSize()) / bytesPerSample
	return time.Duration(sample) * time.Second / time.Duration(p.factory.sampleRate)
}

func (p *readerPlayer) Rewind() error {
	return p.Seek(0)
}

func (p *readerPlayer) Seek(offset time.Duration) error {
	p.m.Lock()
	defer p.m.Unlock()

	if p.player != nil {
		if p.player.IsPlaying() {
			defer func() {
				p.player.Play()
			}()
		}
		p.player.Reset()
	}
	return p.stream.Seek(offset)
}

func (p *readerPlayer) source() io.Reader {
	return p.stream.r
}

type timeStream struct {
	r          io.Reader
	sampleRate int
	pos        int64
	buf        []byte
	unread     int
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

func (s *timeStream) Unread(n int) {
	if s.unread+n > len(s.buf) {
		panic(fmt.Sprintf("audio: too much unreading: %d, the buffer size: %d, unreading position: %d", n, len(s.buf), s.unread))
	}
	s.unread += n
	s.pos -= int64(n)
}

func (s *timeStream) Read(buf []byte) (int, error) {
	if s.unread > 0 {
		n := copy(buf, s.buf[len(s.buf)-s.unread:])
		s.unread -= n
		s.pos += int64(n)
		return n, nil
	}

	n, err := s.r.Read(buf)
	s.pos += int64(n)
	s.buf = append(s.buf, buf[:n]...)
	if m := maxBufferSize(s.sampleRate); len(s.buf) > m {
		s.buf = s.buf[len(s.buf)-m:]
	}
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
	s.buf = s.buf[:0]
	s.unread = 0
	return nil
}

func (s *timeStream) Current() int64 {
	return s.pos
}
