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

	"github.com/hajimehoshi/ebiten/v2/audio/internal/readerdriver"
)

type (
	dummyWriterContext struct{}
	dummyWriterPlayer  struct{}
)

func (c *dummyWriterContext) NewPlayer() io.WriteCloser {
	return &dummyWriterPlayer{}
}

func (c *dummyWriterContext) Close() error {
	return nil
}

func (p *dummyWriterPlayer) Write(b []byte) (int, error) {
	return len(b), nil
}

func (p *dummyWriterPlayer) Close() error {
	return nil
}

func init() {
	writerDriverForTesting = &dummyWriterContext{}
}

type (
	dummyReaderContext struct{}
	dummyReaderPlayer  struct {
		r       io.Reader
		playing bool
		volume  float64
		m       sync.Mutex
	}
)

func (c *dummyReaderContext) NewPlayer(r io.Reader) readerdriver.Player {
	return &dummyReaderPlayer{
		r:      r,
		volume: 1,
	}
}

func (c *dummyReaderContext) MaxBufferSize() int {
	return 48000 * channelNum * bitDepthInBytes / 4
}

func (c *dummyReaderContext) Suspend() {
}

func (c *dummyReaderContext) Resume() {
}

func (c *dummyReaderContext) Close() error {
	return nil
}

func (p *dummyReaderPlayer) Pause() {
	p.m.Lock()
	p.playing = false
	p.m.Unlock()
}

func (p *dummyReaderPlayer) Play() {
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

func (p *dummyReaderPlayer) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()
	return p.playing
}

func (p *dummyReaderPlayer) Reset() {
	p.m.Lock()
	defer p.m.Unlock()
	p.playing = false
}

func (p *dummyReaderPlayer) Volume() float64 {
	return p.volume
}

func (p *dummyReaderPlayer) SetVolume(volume float64) {
	p.volume = volume
}

func (p *dummyReaderPlayer) UnplayedBufferSize() int64 {
	return 0
}

func (p *dummyReaderPlayer) Err() error {
	return nil
}

func (p *dummyReaderPlayer) Close() error {
	p.m.Lock()
	defer p.m.Unlock()
	p.playing = false
	return nil
}

func init() {
	readerDriverForTesting = &dummyReaderContext{}
}

type dummyHook struct {
	updates []func() error
}

func (h *dummyHook) OnSuspendAudio(f func()) {
}

func (h *dummyHook) OnResumeAudio(f func()) {
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
