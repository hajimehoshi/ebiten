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
	"errors"
	"io"

	"github.com/gopherjs/gopherjs/js"
	"github.com/hajimehoshi/ebiten"
)

type player struct {
	src               io.Reader
	sampleRate        int
	positionInSamples int64
	context           *js.Object
	bufferSource      *js.Object
}

func newPlayer(src io.Reader, sampleRate int) (*player, error) {
	class := js.Global.Get("AudioContext")
	if class == js.Undefined {
		class = js.Global.Get("webkitAudioContext")
	}
	if class == js.Undefined {
		return nil, errors.New("audio: audio couldn't be initialized")
	}
	p := &player{
		src:          src,
		sampleRate:   sampleRate,
		bufferSource: nil,
		context:      class.New(),
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

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (p *player) proceed() error {
	bufferSize := p.sampleRate * bytesPerSample * channelNum / ebiten.FPS
	c := int64(p.context.Get("currentTime").Float() * float64(p.sampleRate))
	if p.positionInSamples < c {
		p.positionInSamples = c
	}
	b := make([]byte, bufferSize)
	n, err := p.src.Read(b)
	if 0 < n {
		buf := p.context.Call("createBuffer", channelNum, n/bytesPerSample/channelNum, p.sampleRate)
		l := buf.Call("getChannelData", 0)
		r := buf.Call("getChannelData", 1)
		il, ir := toLR(b[:n])
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
	}
	return err
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
