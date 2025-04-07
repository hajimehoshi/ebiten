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
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// player is almost the same as the interface oto.Player.
// This is defined in order to remove the dependency on Oto from this file.
type player interface {
	Pause()
	Play()
	IsPlaying() bool
	Volume() float64
	SetVolume(volume float64)
	BufferedSize() int
	Err() error
	SetBufferSize(bufferSize int)
	io.Seeker

	// As of Oto v3.4.0-alpha.7, Close does nothing.
}

type context interface {
	NewPlayer(io.Reader) player
	Suspend() error
	Resume() error
	Err() error
}

type playerFactory struct {
	context    context
	sampleRate int

	m sync.Mutex
}

var driverForTesting context

func newPlayerFactory(sampleRate int) *playerFactory {
	f := &playerFactory{
		sampleRate: sampleRate,
	}
	if driverForTesting != nil {
		f.context = driverForTesting
	}
	return f
}

type playerImpl struct {
	context        *Context
	player         player
	src            io.Reader
	seekable       bool
	srcIdent       any
	stream         *timeStream
	factory        *playerFactory
	initBufferSize int
	bytesPerSample int

	// adjustedPosition is the player's more accurate position as time.Duration.
	// The underlying buffer might not be changed even if the player is playing.
	// adjustedPosition is adjusted by the time duration during the player position doesn't change while its playing.
	adjustedPosition int64

	// lastSamples is the last value of the number of samples.
	// When lastSamples is a negative number, this value is not initialized yet.
	lastSamples int64

	// stopwatch is a stopwatch to measure the time duration during the player position doesn't change while its playing.
	stopwatch stopwatch

	closed bool

	m sync.Mutex
}

func (f *playerFactory) newPlayer(context *Context, src io.Reader, seekable bool, srcIdent any, bitDepthInBytes int) (*playerImpl, error) {
	f.m.Lock()
	defer f.m.Unlock()

	p := &playerImpl{
		src:            src,
		seekable:       seekable,
		srcIdent:       srcIdent,
		context:        context,
		factory:        f,
		lastSamples:    -1,
		bytesPerSample: bitDepthInBytes * channelCount,
	}
	runtime.SetFinalizer(p, (*playerImpl).Close)
	return p, nil
}

func (f *playerFactory) suspend() error {
	f.m.Lock()
	defer f.m.Unlock()

	if f.context == nil {
		return nil
	}
	return addErrorInfo(f.context.Suspend())
}

func (f *playerFactory) resume() error {
	f.m.Lock()
	defer f.m.Unlock()

	if f.context == nil {
		return nil
	}
	return addErrorInfo(f.context.Resume())
}

func (f *playerFactory) error() error {
	f.m.Lock()
	defer f.m.Unlock()

	if f.context == nil {
		return nil
	}
	return addErrorInfo(f.context.Err())
}

func (f *playerFactory) initContextIfNeeded() (<-chan struct{}, error) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.context != nil {
		return nil, nil
	}

	c, ready, err := newContext(f.sampleRate)
	if err != nil {
		return nil, err
	}
	f.context = c
	return ready, nil
}

func (p *playerImpl) ensurePlayer() error {
	// Initialize the underlying player lazily to enable calling NewContext in an 'init' function.
	// Accessing the underlying player functions requires the environment to be already initialized,
	// but if Ebitengine is used for a shared library, the timing when init functions are called
	// is unexpectable.
	// e.g. a variable for JVM on Android might not be set.
	ready, err := p.factory.initContextIfNeeded()
	if err != nil {
		return err
	}
	if ready != nil {
		go func() {
			<-ready
			p.context.setReady()
		}()
	}

	if p.stream == nil {
		s, err := newTimeStream(p.src, p.seekable, p.factory.sampleRate, p.bytesPerSample/channelCount)
		if err != nil {
			return err
		}
		p.stream = s
	}
	if p.player == nil {
		p.player = p.factory.context.NewPlayer(p.stream)
		if p.initBufferSize != 0 {
			p.player.SetBufferSize(p.initBufferSize)
			p.initBufferSize = 0
		}
	}
	return nil
}

func (p *playerImpl) Play() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		p.context.setError(fmt.Errorf("audio: Play for a closed player"))
		return
	}

	if err := p.ensurePlayer(); err != nil {
		p.context.setError(err)
		return
	}
	if p.player.IsPlaying() {
		return
	}
	p.player.Play()
	p.context.addPlayingPlayer(p)
	p.stopwatch.start()
}

func (p *playerImpl) Pause() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		p.context.setError(fmt.Errorf("audio: Pause for a closed player"))
		return
	}

	if p.player == nil {
		return
	}
	if !p.player.IsPlaying() {
		return
	}

	p.player.Pause()
	p.context.removePlayingPlayer(p)
	p.stopwatch.stop()
}

func (p *playerImpl) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		return false
	}
	return p.isPlaying()
}

func (p *playerImpl) isPlaying() bool {
	if p.player == nil {
		return false
	}
	return p.player.IsPlaying()
}

func (p *playerImpl) Volume() float64 {
	p.m.Lock()
	defer p.m.Unlock()

	if err := p.ensurePlayer(); err != nil {
		p.context.setError(err)
		return 0
	}
	return p.player.Volume()
}

func (p *playerImpl) SetVolume(volume float64) {
	p.m.Lock()
	defer p.m.Unlock()

	if err := p.ensurePlayer(); err != nil {
		p.context.setError(err)
		return
	}
	p.player.SetVolume(volume)
}

func (p *playerImpl) Close() error {
	p.m.Lock()
	defer p.m.Unlock()
	runtime.SetFinalizer(p, nil)

	if p.closed {
		return fmt.Errorf("audio: player is already closed")
	}
	p.closed = true

	if p.player != nil {
		defer func() {
			p.player = nil
		}()
		p.player.Pause()
		p.stopwatch.stop()
	}
	return nil
}

func (p *playerImpl) Position() time.Duration {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		return 0
	}
	return time.Duration(p.adjustedPosition)
}

func (p *playerImpl) Rewind() error {
	return p.SetPosition(0)
}

func (p *playerImpl) SetPosition(offset time.Duration) error {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		return fmt.Errorf("audio: player is already closed")
	}

	if offset == 0 && p.player == nil {
		p.adjustedPosition = 0
		return nil
	}

	if err := p.ensurePlayer(); err != nil {
		return err
	}

	pos := p.stream.timeDurationToPos(offset)
	if _, err := p.player.Seek(pos, io.SeekStart); err != nil {
		return addErrorInfo(err)
	}
	p.lastSamples = -1
	// Just after setting a position, the buffer size should be 0 as no data is sent.
	p.adjustedPosition = int64(p.stream.positionInTimeDuration())
	p.stopwatch.reset()
	if p.isPlaying() {
		p.stopwatch.start()
	}
	return nil
}

func (p *playerImpl) Err() error {
	p.m.Lock()
	defer p.m.Unlock()

	if p.player == nil {
		return nil
	}
	return addErrorInfo(p.player.Err())
}

func (p *playerImpl) SetBufferSize(bufferSize time.Duration) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		p.context.setError(fmt.Errorf("audio: SetBufferSize for a closed player"))
		return
	}

	bufferSizeInBytes := int(bufferSize * time.Duration(p.bytesPerSample) * time.Duration(p.factory.sampleRate) / time.Second)
	bufferSizeInBytes = bufferSizeInBytes / p.bytesPerSample * p.bytesPerSample
	if p.player == nil {
		p.initBufferSize = bufferSizeInBytes
		return
	}
	p.player.SetBufferSize(bufferSizeInBytes)
}

func (p *playerImpl) sourceIdent() any {
	return p.srcIdent
}

func (p *playerImpl) onContextSuspended() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.player == nil {
		return
	}
	p.stopwatch.stop()
}

func (p *playerImpl) onContextResumed() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.player == nil {
		return
	}
	if p.isPlaying() {
		p.stopwatch.start()
	}
}

func (p *playerImpl) updatePosition() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.player == nil {
		p.adjustedPosition = 0
		return
	}
	if !p.context.IsReady() {
		p.adjustedPosition = 0
		return
	}

	samples := (p.stream.position() - int64(p.player.BufferedSize())) / int64(p.bytesPerSample)

	var adjustingTime time.Duration
	if p.lastSamples >= 0 && p.lastSamples == samples {
		// If the number of samples is not changed from the last tick,
		// the underlying buffer is not updated yet. Adjust the position by the time (#2901).
		adjustingTime = p.stopwatch.current()
	} else {
		p.lastSamples = samples
		p.stopwatch.reset()
		if p.isPlaying() {
			p.stopwatch.start()
		}
	}

	// Update the adjusted position every tick. This is necessary to keep the position accurate.
	p.adjustedPosition = int64(time.Duration(samples)*time.Second/time.Duration(p.factory.sampleRate) + adjustingTime)
}

type timeStream struct {
	r              io.Reader
	seekable       bool
	sampleRate     int
	pos            atomic.Int64
	bytesPerSample int

	// m is a mutex for this stream.
	// All the exported functions are protected by this mutex as Read can be read from a different goroutine than Seek.
	m sync.Mutex
}

func newTimeStream(r io.Reader, seekable bool, sampleRate int, bitDepthInBytes int) (*timeStream, error) {
	s := &timeStream{
		r:              r,
		seekable:       seekable,
		sampleRate:     sampleRate,
		bytesPerSample: bitDepthInBytes * channelCount,
	}
	if seekable {
		// Get the current position of the source.
		pos, err := s.r.(io.Seeker).Seek(0, io.SeekCurrent)
		if err != nil {
			if !errors.Is(err, errors.ErrUnsupported) {
				return nil, err
			}
			// Ignore the error, as the undelrying source might not support Seek (#3192).
			// This happens when vorbis.Decode* is used, as vorbis.Stream is io.Seeker whichever the underlying source is.
			pos = 0
		}
		s.pos.Store(pos)
	}
	return s, nil
}

func (s *timeStream) Read(buf []byte) (int, error) {
	s.m.Lock()
	defer s.m.Unlock()

	n, err := s.r.Read(buf)
	s.pos.Add(int64(n))
	return n, err
}

func (s *timeStream) Seek(offset int64, whence int) (int64, error) {
	s.m.Lock()
	defer s.m.Unlock()

	if !s.seekable {
		// TODO: Should this return an error?
		panic("audio: the source must be io.Seeker when seeking but not")
	}
	pos, err := s.r.(io.Seeker).Seek(offset, whence)
	if err != nil {
		return pos, err
	}

	s.pos.Store(pos)
	return pos, nil
}

func (s *timeStream) timeDurationToPos(offset time.Duration) int64 {
	o := int64(offset) * int64(s.bytesPerSample) * int64(s.sampleRate) / int64(time.Second)

	// Align the byte position with the samples.
	o -= o % int64(s.bytesPerSample)
	o += s.pos.Load() % int64(s.bytesPerSample)

	return o
}

func (s *timeStream) position() int64 {
	return s.pos.Load()
}

func (s *timeStream) positionInTimeDuration() time.Duration {
	return time.Duration(s.pos.Load()) * time.Second / (time.Duration(s.sampleRate * s.bytesPerSample))
}
