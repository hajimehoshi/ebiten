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
	"io/ioutil"
	"sync"

	"github.com/hajimehoshi/oto/v2"
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

func (c *dummyContext) NewPlayer(r io.Reader) oto.Player {
	return &dummyPlayer{
		r:      r,
		volume: 1,
	}
}

func (c *dummyContext) MaxBufferSize() int {
	return 48000 * channelNum * bitDepthInBytes / 4
}

func (c *dummyContext) Suspend() error {
	return nil
}

func (c *dummyContext) Resume() error {
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
		if _, err := ioutil.ReadAll(p.r); err != nil {
			panic(err)
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

func (p *dummyPlayer) Reset() {
	p.m.Lock()
	defer p.m.Unlock()
	p.playing = false
}

func (p *dummyPlayer) Volume() float64 {
	return p.volume
}

func (p *dummyPlayer) SetVolume(volume float64) {
	p.volume = volume
}

func (p *dummyPlayer) UnplayedBufferSize() int {
	return 0
}

func (p *dummyPlayer) Err() error {
	return nil
}

func (p *dummyPlayer) Close() error {
	p.m.Lock()
	defer p.m.Unlock()
	p.playing = false
	return nil
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
	hookForTesting = &dummyHook{}
}

func UpdateForTesting() error {
	for _, f := range hookForTesting.(*dummyHook).updates {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

func PlayersNumForTesting() int {
	c := CurrentContext()
	c.m.Lock()
	n := len(c.players)
	c.m.Unlock()
	return n
}

func ResetContextForTesting() {
	theContext = nil
}
