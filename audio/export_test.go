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
		m       sync.Mutex
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
	p.m.Lock()
	p.playing = false
	p.m.Unlock()
}

func (p *dummyPlayer) Play() {
	p.m.Lock()
	p.playing = true
	p.m.Unlock()
	go func() {
		var buf [4096]byte
		for {
			_, err := p.r.Read(buf[:])
			if err != nil {
				if err != io.EOF {
					panic(err)
				}
				break
			}
			time.Sleep(time.Millisecond)
		}
		p.m.Lock()
		p.playing = false
		p.m.Unlock()
	}()
}

func (p *dummyPlayer) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()
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
	updates []func() error
}

func (h *dummyHook) OnSuspendAudio(f func() error) {
}

func (h *dummyHook) OnResumeAudio(f func() error) {
}

func (h *dummyHook) AppendHookOnBeforeUpdate(f func() error) {
	h.updates = append(h.updates, f)
}

func init() {
	hookerForTesting = &dummyHook{}
}

func UpdateForTesting() error {
	for _, f := range hookerForTesting.(*dummyHook).updates {
		if err := f(); err != nil {
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

func ResetContextForTesting() {
	theContext = nil
}

func (i *InfiniteLoop) SetNoBlendForTesting(value bool) {
	i.noBlendForTesting = value
}
