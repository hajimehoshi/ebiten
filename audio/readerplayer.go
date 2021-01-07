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
	"runtime"
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
	src     io.Reader
	playing bool
	m       sync.Mutex
}

func (c *readerPlayerFactory) newPlayerImpl(context *Context, src io.Reader) (playerImpl, error) {
	p := &readerPlayer{
		context: context,
		player:  c.driver.NewPlayer(src),
		src:     src,
	}
	runtime.SetFinalizer(p, (*readerPlayer).Close)
	return p, nil
}

func (p *readerPlayer) Play() {
	p.m.Lock()
	defer p.m.Unlock()

	p.player.Play()
	p.playing = true
	p.context.addPlayer(p)
}

func (p *readerPlayer) Pause() {
	p.m.Lock()
	defer p.m.Unlock()

	p.player.Pause()
	p.playing = false
}

func (p *readerPlayer) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()

	return p.playing
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

	runtime.SetFinalizer(p, nil)
	p.context.removePlayer(p)
	p.playing = false
	return p.player.Close()
}

func (p *readerPlayer) Current() time.Duration {
	panic("not implemented")
}

func (p *readerPlayer) Rewind() error {
	panic("not implemented")
}

func (p *readerPlayer) Seek(offset time.Duration) error {
	panic("not implemented")
}

func (p *readerPlayer) source() io.Reader {
	return p.src
}
