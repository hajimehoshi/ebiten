// Copyright 2015 Hajime Hoshi
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

// +build js

package audio

import (
	"io"
	"time"

	"github.com/gopherjs/gopherjs/js"
)

var context *js.Object

type player struct {
	src          io.Reader
	sampleRate   int
	position     float64
	bufferSource *js.Object
}

var currentPlayer *player

func startPlaying(src io.Reader, sampleRate int) error {
	// Do nothing in node.js.
	if js.Global.Get("require") != js.Undefined {
		return nil
	}

	if currentPlayer != nil || context != nil {
		panic("audio: currentPlayer already exists")
	}
	class := js.Global.Get("AudioContext")
	if class == js.Undefined {
		class = js.Global.Get("webkitAudioContext")
	}
	if class == js.Undefined {
		panic("audio: audio couldn't be initialized")
	}
	context = class.New()

	currentPlayer = &player{
		src:          src,
		sampleRate:   sampleRate,
		position:     context.Get("currentTime").Float(),
		bufferSource: nil,
	}
	if err := currentPlayer.start(); err != nil {
		return err
	}
	return nil
}

func toLR(data []byte) ([]int16, []int16) {
	l := make([]int16, len(data)/4)
	r := make([]int16, len(data)/4)
	for i := 0; i < len(data)/4; i++ {
		l[i] = int16(data[4*i]) | int16(data[4*i+1])<<8
		r[i] = int16(data[4*i+2]) | int16(data[4*i+3])<<8
	}
	return l, r
}

func (p *player) proceed() error {
	buf := make([]byte, 4096)
	n, err := p.src.Read(buf)
	if 0 < n {
		const channelNum = 2
		const bytesPerSample = channelNum * 16 / 8
		b := context.Call("createBuffer", channelNum, n/bytesPerSample, p.sampleRate)
		l := b.Call("getChannelData", 0)
		r := b.Call("getChannelData", 1)
		il, ir := toLR(buf[:n])
		const max = 1 << 15
		for i := 0; i < len(il); i++ {
			l.SetIndex(i, float64(il[i])/max)
			r.SetIndex(i, float64(ir[i])/max)
		}
		p.bufferSource = context.Call("createBufferSource")
		p.bufferSource.Set("buffer", b)
		p.bufferSource.Call("connect", context.Get("destination"))
		p.bufferSource.Call("start", p.position)
		p.position += b.Get("duration").Float()
	}
	return err
}

func (p *player) start() error {
	// TODO: What if play is already called?
	go func() {
		defer p.close()
		for {
			err := p.proceed()
			if err == io.EOF {
				break
			}
			if err != nil {
				// TODO: Record the last error
				panic(err)
			}
			time.Sleep(1)
		}
	}()
	return nil
}

func (p *player) close() error {
	if p.bufferSource == nil {
		return nil
	}
	p.bufferSource.Call("stop")
	p.bufferSource.Call("disconnect")
	p.bufferSource = nil
	return nil
}
