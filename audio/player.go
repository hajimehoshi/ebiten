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
	return &playerFactory{
		sampleRate: sampleRate,
	}
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

	// volume is the player's volume in the range [0, 1].
	// The value is kept here so that a volume set before the underlying player is created is preserved.
	volume float64

	// pendingPlay is whether Play was requested before the underlying player was created.
	// Such a player starts playing once the audio device is created.
	pendingPlay bool

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
		volume:         1,
	}
	// AddCleanup is not set for p. The caller of newPlayer should set a finalizer for p's wrapper.
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

func (f *playerFactory) initContextIfNeeded(vmGuest bool) (<-chan struct{}, error) {
	f.m.Lock()
	defer f.m.Unlock()

	if f.context != nil {
		return nil, nil
	}

	if driverForTesting != nil {
		f.context = driverForTesting
		ready := make(chan struct{})
		close(ready)
		return ready, nil
	}

	c, ready, err := newContext(f.sampleRate, vmGuest)
	if err != nil {
		return nil, err
	}
	f.context = c
	return ready, nil
}

func (f *playerFactory) currentContext() context {
	f.m.Lock()
	defer f.m.Unlock()
	return f.context
}

func (p *playerImpl) ensureStream() error {
	if p.stream != nil {
		return nil
	}
	s, err := newTimeStream(p.src, p.seekable, p.factory.sampleRate, p.bytesPerSample/channelCount)
	if err != nil {
		return err
	}
	p.stream = s
	return nil
}

// ensurePlayer creates the underlying player if the audio device exists.
//
// The audio device is created lazily from the before-update hook, which runs after the UI
// backend is initialized. Until then ensurePlayer leaves the underlying player nil, and the
// caller must record the requested operation (play, volume, position) to apply once the device
// is created.
func (p *playerImpl) ensurePlayer() error {
	if err := p.ensureStream(); err != nil {
		return err
	}
	if p.player != nil {
		return nil
	}

	c := p.factory.currentContext()
	if c == nil {
		return nil
	}

	pl := c.NewPlayer(p.stream)
	if p.initBufferSize != 0 {
		pl.SetBufferSize(p.initBufferSize)
		p.initBufferSize = 0
	}
	pl.SetVolume(p.volume)
	p.player = pl
	return nil
}

// startIfPending starts playing if Play was requested before the audio device was created.
// startIfPending must not be called with p.m locked.
func (p *playerImpl) startIfPending() error {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed || !p.pendingPlay {
		return nil
	}
	if err := p.ensurePlayer(); err != nil {
		return err
	}
	if p.player == nil {
		// The audio device is not created yet. This is the normal state on every tick between
		// Play and the first before-update hook (which creates the device); keep the play pending.
		return nil
	}
	if !p.player.IsPlaying() {
		p.player.Play()
		p.stopwatch.start()
	}
	p.pendingPlay = false
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

	if p.player != nil {
		p.pendingPlay = false
		if p.player.IsPlaying() {
			return
		}
		p.player.Play()
		p.stopwatch.start()
	} else {
		// The audio device is not created yet. Remember the request and start playing
		// once the device is created (from the before-update hook).
		if p.pendingPlay {
			return
		}
		p.pendingPlay = true
	}
	// Add the player to the context. The context's reference guards the player from being
	// garbage-collected while it is playing, even after all the references to its Player wrapper
	// are gone (see the Player documentation), and lets the context track and start the player.
	// The player only needs to be added once; the early returns above avoid redundant adds.
	p.context.addPlayingPlayer(p)
}

func (p *playerImpl) Pause() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		p.context.setError(fmt.Errorf("audio: Pause for a closed player"))
		return
	}

	if p.player == nil {
		// The audio device is not created yet. Cancel a pending play request if any.
		if p.pendingPlay {
			p.pendingPlay = false
			p.context.removePlayingPlayer(p)
		}
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
		return p.pendingPlay
	}
	return p.player.IsPlaying()
}

func (p *playerImpl) Volume() float64 {
	p.m.Lock()
	defer p.m.Unlock()

	if p.player == nil {
		return p.volume
	}
	return p.player.Volume()
}

func (p *playerImpl) SetVolume(volume float64) {
	p.m.Lock()
	defer p.m.Unlock()

	p.volume = volume
	if p.player != nil {
		p.player.SetVolume(volume)
	}
}

func (p *playerImpl) Close() error {
	p.m.Lock()
	defer p.m.Unlock()

	if p.closed {
		return fmt.Errorf("audio: player is already closed")
	}
	p.closed = true
	p.pendingPlay = false

	if p.player != nil {
		defer func() {
			p.player = nil
		}()
		p.player.Pause()
		// Release the device player if it holds resources beyond this process (the
		// virtualization guest's forwarded player).
		if closer, ok := p.player.(io.Closer); ok {
			_ = closer.Close()
		}
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

	if p.player != nil {
		// The device is available. Seek via the underlying player so that its buffer is reset.
		pos := p.stream.timeDurationToPos(offset)
		if _, err := p.player.Seek(pos, io.SeekStart); err != nil {
			return addErrorInfo(err)
		}
		p.afterSeek()
		return nil
	}

	// The audio device is not created yet. Record the position by seeking the stream so that
	// the underlying player starts from this position once it is created.

	// Seeking to the start before the stream is created needs no source access.
	if offset == 0 && p.stream == nil {
		p.adjustedPosition = 0
		return nil
	}

	if err := p.ensureStream(); err != nil {
		return err
	}

	// A non-seekable stream is always at the beginning since the player has not read it yet,
	// so seeking to 0 is unnecessary. A non-zero offset requires a seek, which panics for a
	// non-seekable source as documented.
	if offset != 0 || p.seekable {
		pos := p.stream.timeDurationToPos(offset)
		if _, err := p.stream.Seek(pos, io.SeekStart); err != nil {
			return addErrorInfo(err)
		}
	}
	p.afterSeek()
	return nil
}

func (p *playerImpl) afterSeek() {
	p.lastSamples = -1
	// Just after setting a position, the buffer size should be 0 as no data is sent.
	p.adjustedPosition = int64(p.stream.positionInTimeDuration())
	p.stopwatch.reset()
	if p.isPlaying() {
		p.stopwatch.start()
	}
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
		// The underlying player is not created yet. Keep the position recorded by
		// SetPosition (0 by default) until the player starts.
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
