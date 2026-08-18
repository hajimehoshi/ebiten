// Copyright 2018 The Ebiten Authors
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

type (
	dummyContext struct{}
	dummyPlayer  struct {
		r       io.Reader
		playing bool
		volume  float64

		// readGen is incremented by Reset to stop the goroutine reading r.
		readGen int

		mu sync.Mutex
	}
)

func (c *dummyContext) NewPlayer(r io.Reader) player {
	return &dummyPlayer{
		r:      r,
		volume: 1,
	}
}

func (c *dummyContext) MaxBufferSize() int {
	return 48000 * channelCount * bitDepthInBytesInt16 / 4
}

func (c *dummyContext) Suspend() error {
	return nil
}

func (c *dummyContext) Resume() error {
	return nil
}

func (c *dummyContext) Err() error {
	return nil
}

func (p *dummyPlayer) Pause() {
	p.mu.Lock()
	p.playing = false
	p.mu.Unlock()
}

func (p *dummyPlayer) Play() {
	p.mu.Lock()
	p.playing = true
	gen := p.readGen
	p.mu.Unlock()
	go func() {
		var buf [4096]byte
		for {
			stopped, err := p.readOnce(gen, buf[:])
			if stopped {
				return
			}
			if err != nil {
				if err != io.EOF {
					panic(err)
				}
				break
			}
			time.Sleep(time.Millisecond)
		}
		p.mu.Lock()
		p.playing = false
		p.mu.Unlock()
	}()
}

// readOnce performs one read from the source with the mutex held, so that Reset waits for an
// in-flight read. stopped reports that Reset was called and reading must not continue.
func (p *dummyPlayer) readOnce(gen int, buf []byte) (stopped bool, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.readGen != gen {
		return true, nil
	}
	_, err = p.r.Read(buf)
	return false, err
}

func (p *dummyPlayer) Reset() {
	p.mu.Lock()
	p.playing = false
	p.readGen++
	p.mu.Unlock()
}

func (p *dummyPlayer) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.playing
}

func (p *dummyPlayer) Volume() float64 {
	return p.volume
}

func (p *dummyPlayer) SetVolume(volume float64) {
	p.volume = volume
}

func (p *dummyPlayer) BufferedSize() int {
	return 0
}

func (p *dummyPlayer) Err() error {
	return nil
}

func (p *dummyPlayer) SetBufferSize(bufferSize int) {
}

func (p *dummyPlayer) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func init() {
	driverForTesting = &dummyContext{}
}

type dummyHook struct {
	updates []func(vmGuest bool) error
}

func (h *dummyHook) OnSuspendAudio(f func() error) {
}

func (h *dummyHook) OnResumeAudio(f func() error) {
}

func (h *dummyHook) AppendHookOnBeforeUpdateWithVMGuestInfo(f func(vmGuest bool) error) {
	h.updates = append(h.updates, f)
}

func init() {
	hookerForTesting = &dummyHook{}
}

func UpdateForTesting() error {
	for _, f := range hookerForTesting.(*dummyHook).updates {
		if err := f(false); err != nil {
			return err
		}
	}
	return nil
}

func PlayersCountForTesting() int {
	c := CurrentContext()
	c.m.Lock()
	n := len(c.playingPlayers)
	c.m.Unlock()
	return n
}

// ContextCreatedForTesting reports whether the underlying audio device has been created.
func ContextCreatedForTesting() bool {
	c := CurrentContext()
	if c == nil {
		return false
	}
	return c.playerFactory.currentContext() != nil
}

func ResetContextForTesting() {
	theContext = nil
}

func (i *InfiniteLoop) SetNoBlendForTesting(value bool) {
	i.noBlendForTesting = value
}
