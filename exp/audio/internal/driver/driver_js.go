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

package driver

import (
	"errors"

	"github.com/gopherjs/gopherjs/js"
)

type Player struct {
	sampleRate        int
	channelNum        int
	bytesPerSample    int
	positionInSamples int64
	context           *js.Object
	bufferSource      *js.Object
}

func NewPlayer(sampleRate, channelNum, bytesPerSample int) (*Player, error) {
	class := js.Global.Get("AudioContext")
	if class == js.Undefined {
		class = js.Global.Get("webkitAudioContext")
	}
	if class == js.Undefined {
		return nil, errors.New("driver: audio couldn't be initialized")
	}
	p := &Player{
		sampleRate:     sampleRate,
		channelNum:     channelNum,
		bytesPerSample: bytesPerSample,
		bufferSource:   nil,
		context:        class.New(),
	}
	p.positionInSamples = int64(p.context.Get("currentTime").Float() * float64(p.sampleRate))
	return p, nil
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

func (p *Player) Proceed(data []byte) error {
	c := int64(p.context.Get("currentTime").Float() * float64(p.sampleRate))
	if p.positionInSamples < c {
		p.positionInSamples = c
	}
	n := len(data)
	buf := p.context.Call("createBuffer", p.channelNum, n/p.bytesPerSample/p.channelNum, p.sampleRate)
	l := buf.Call("getChannelData", 0)
	r := buf.Call("getChannelData", 1)
	il, ir := toLR(data)
	const max = 1 << 15
	for i := 0; i < len(il); i++ {
		l.SetIndex(i, float64(il[i])/max)
		r.SetIndex(i, float64(ir[i])/max)
	}
	p.bufferSource = p.context.Call("createBufferSource")
	p.bufferSource.Set("buffer", buf)
	p.bufferSource.Call("connect", p.context.Get("destination"))
	p.bufferSource.Call("start", float64(p.positionInSamples)/float64(p.sampleRate))
	// Call 'stop' or we'll get noisy sound especially on Chrome.
	p.bufferSource.Call("stop", float64(p.positionInSamples+int64(len(il)))/float64(p.sampleRate))
	p.positionInSamples += int64(len(il))
	return nil
}

func (p *Player) Close() error {
	if p.bufferSource == nil {
		return nil
	}
	p.bufferSource.Call("stop")
	p.bufferSource.Call("disconnect")
	p.bufferSource = nil
	return nil
}
