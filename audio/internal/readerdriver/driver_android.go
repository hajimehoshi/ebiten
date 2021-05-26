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

package readerdriver

import (
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio/internal/oboe"
)

func IsAvailable() bool {
	return true
}

type context struct {
	sampleRate      int
	channelNum      int
	bitDepthInBytes int
}

func NewContext(sampleRate int, channelNum int, bitDepthInBytes int) (Context, chan struct{}, error) {
	ready := make(chan struct{})
	close(ready)

	c := &context{
		sampleRate:      sampleRate,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
	}
	return c, ready, nil
}

func (c *context) NewPlayer(src io.Reader) Player {
	p := &player{
		context: c,
		src:     src,
		cond:    sync.NewCond(&sync.Mutex{}),
		volume:  1,
	}
	runtime.SetFinalizer(p, (*player).Close)
	return p
}

func (c *context) Suspend() error {
	return oboe.Suspend()
}

func (c *context) Resume() error {
	return oboe.Resume()
}

type player struct {
	context *context
	p       *oboe.Player
	src     io.Reader
	err     error
	cond    *sync.Cond
	closed  bool
	volume  float64
	eof     bool
}

func (p *player) Pause() {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.err != nil {
		return
	}
	if p.closed {
		return
	}
	if p.p == nil {
		return
	}
	if err := p.p.Pause(); err != nil {
		p.setErrorImpl(err)
		return
	}
	p.cond.Signal()
}

func (p *player) Play() {
	// Call Play asynchronously since Oboe's Play might take long.
	ch := make(chan struct{})
	go func() {
		p.cond.L.Lock()
		defer p.cond.L.Unlock()
		close(ch)
		p.playImpl()
	}()

	// Wait until the mutex is locked in the above goroutine.
	<-ch
}

func (p *player) playImpl() {
	if p.err != nil {
		return
	}
	if p.p != nil && p.p.IsPlaying() {
		return
	}
	defer p.cond.Signal()
	var runLoop bool
	if p.p == nil {
		p.p = oboe.NewPlayer(p.context.sampleRate, p.context.channelNum, p.context.bitDepthInBytes, p.volume, func() {
			p.cond.Signal()
		})
		runLoop = true
	}

	buf := make([]byte, p.context.maxBufferSize())
	for p.p.UnplayedBufferSize() < p.context.maxBufferSize() {
		n, err := p.src.Read(buf)
		if err != nil && err != io.EOF {
			p.setErrorImpl(err)
			return
		}
		p.p.AppendBuffer(buf[:n])
		if err == io.EOF {
			p.eof = true
			break
		}
	}

	if err := p.p.Play(); err != nil {
		p.setErrorImpl(err)
		return
	}
	if runLoop {
		go p.loop()
	}
}

func (p *player) IsPlaying() bool {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	if p.p == nil {
		return false
	}
	return p.p.IsPlaying()
}

func (p *player) Reset() {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.err != nil {
		return
	}
	if p.closed {
		return
	}
	if p.p == nil {
		return
	}
	if err := p.p.Close(); err != nil {
		p.setErrorImpl(err)
		return
	}
	p.p = nil
	p.eof = false
	p.cond.Signal()
}

func (p *player) Volume() float64 {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.volume
}

func (p *player) SetVolume(volume float64) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	p.volume = volume
	if p.p == nil {
		return
	}
	p.p.SetVolume(volume)
}

func (p *player) UnplayedBufferSize() int {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.p == nil {
		return 0
	}
	return p.p.UnplayedBufferSize()
}

func (p *player) Err() error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.err
}

func (p *player) Close() error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.closeImpl()
}

func (p *player) closeImpl() error {
	defer p.cond.Signal()

	runtime.SetFinalizer(p, nil)
	p.closed = true
	if p.p == nil {
		return p.err
	}
	if err := p.p.Close(); err != nil && p.err == nil {
		// Do not call setErrorImpl, or this can cause infinite recursive.
		p.err = err
		return p.err
	}
	p.p = nil
	return p.err
}

func (p *player) setError(err error) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	p.setErrorImpl(err)
}

func (p *player) setErrorImpl(err error) {
	p.err = err
	p.closeImpl()
}

func (p *player) shouldWait() bool {
	if p.closed {
		return false
	}
	if p.p == nil {
		return false
	}

	// Wait when the player is paused.
	if !p.p.IsPlaying() {
		return true
	}

	// When the source reaches EOF, wait until all the data is consumed.
	if p.eof {
		return p.p.UnplayedBufferSize() > 0
	}

	return p.p.UnplayedBufferSize() >= p.context.maxBufferSize()
}

func (p *player) wait() bool {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	for p.shouldWait() {
		p.cond.Wait()
	}
	return p.p != nil && p.p.IsPlaying()
}

func (p *player) loop() {
	buf := make([]byte, 4096)
	for {
		if !p.wait() {
			return
		}

		n, err := p.src.Read(buf)
		if err != nil && err != io.EOF {
			p.setError(err)
			return
		}

		p.cond.L.Lock()
		p.p.AppendBuffer(buf[:n])
		if err == io.EOF {
			p.eof = true
		}

		// Now p.resetImpl() doesn't close the stream gracefully. Then buffer size check is necessary here.
		if p.eof && p.p.UnplayedBufferSize() == 0 {
			// Even when the unplayed buffer size is 0,
			// the audio data in the hardware might not be played yet (#1632).
			// Just wait for a while.
			p.cond.L.Unlock()
			time.Sleep(100 * time.Millisecond)
			p.Reset()
			return
		}
		p.cond.L.Unlock()
	}
}
