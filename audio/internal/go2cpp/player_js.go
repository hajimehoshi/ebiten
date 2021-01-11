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

package go2cpp

import (
	"io"
	"runtime"
	"sync"
	"syscall/js"
)

type Context struct {
	v js.Value
}

func NewContext(sampleRate int) *Context {
	v := js.Global().Get("go2cpp").Call("createAudio", sampleRate, 2, 2, 8192)
	return &Context{
		v: v,
	}
}

func (c *Context) NewPlayer(r io.Reader) *Player {
	cond := sync.NewCond(&sync.Mutex{})
	onwritten := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		cond.Signal()
		return nil
	})
	v := c.v.Call("createPlayer", onwritten)
	p := &Player{
		src:       r,
		v:         v,
		cond:      cond,
		onWritten: onwritten,
	}
	runtime.SetFinalizer(p, (*Player).Close)

	p.v.Set("onWritten", p.onWritten)

	go p.loop()
	return p
}

func (c *Context) Close() error {
	return nil
}

type playerState int

const (
	playerStatePaused playerState = iota
	playerStatePlaying
	playerStateClosed
	playerStateError
)

type Player struct {
	src   io.Reader
	v     js.Value
	state playerState
	cond  *sync.Cond
	err   error

	onWritten js.Func
}

func (p *Player) Pause() {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.state == playerStateClosed {
		return
	}
	p.v.Call("pause")
	p.state = playerStatePaused
}

func (p *Player) Play() {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.state == playerStateClosed {
		return
	}
	p.v.Call("play")
	p.state = playerStatePlaying
	p.cond.Signal()
}

func (p *Player) Reset() {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.state == playerStateClosed {
		return
	}

	p.v.Call("reset")
	p.cond.Signal()
}

func (p *Player) Volume() float64 {
	return p.v.Get("volume").Float()
}

func (p *Player) SetVolume(volume float64) {
	p.v.Set("volume", volume)
}

func (p *Player) Close() error {
	runtime.SetFinalizer(p, nil)

	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.state == playerStateError {
		return p.err
	}

	p.v.Call("close")
	p.state = playerStateClosed
	p.cond.Signal()
	p.onWritten.Release()
	return nil
}

func (p *Player) setError(err error) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.state == playerStateError {
		return
	}

	p.v.Call("close")
	p.err = err
	p.state = playerStateClosed
	p.cond.Signal()
}

func (p *Player) waitUntilUnpaused() bool {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	for p.state == playerStatePaused || (p.state == playerStatePlaying && !p.v.Call("isWritable").Bool()) {
		p.cond.Wait()
	}
	return p.state == playerStatePlaying
}

func (p *Player) loop() {
	const size = 4096

	buf := make([]byte, size)
	dst := js.Global().Get("Uint8Array").New(size)

	for {
		if !p.waitUntilUnpaused() {
			return
		}

		n, err := p.src.Read(buf)
		if err != nil && err != io.EOF {
			p.setError(err)
			return
		}
		if n > 0 {
			js.CopyBytesToJS(dst, buf[:n])
			p.v.Call("write", dst, n)
		}

		if err == io.EOF {
			p.Close()
			return
		}
	}
}
