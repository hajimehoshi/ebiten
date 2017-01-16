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
	"strings"

	"github.com/gopherjs/gopherjs/js"
)

type Player struct {
	sampleRate        int
	channelNum        int
	bytesPerSample    int
	positionInSamples int64
	bufferedData      []byte
	context           *js.Object
}

func isIOS() bool {
	ua := js.Global.Get("navigator").Get("userAgent").String()
	if !strings.Contains(ua, "iPhone") {
		return false
	}
	return true
}

func isAndroidChrome() bool {
	ua := js.Global.Get("navigator").Get("userAgent").String()
	if !strings.Contains(ua, "Android") {
		return false
	}
	if !strings.Contains(ua, "Chrome") {
		return false
	}
	return true
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
		bufferedData:   []byte{},
		context:        class.New(),
	}
	// iOS and Android Chrome requires touch event to use AudioContext.
	if isIOS() || isAndroidChrome() {
		js.Global.Get("document").Call("addEventListener", "touchend", func() {
			// Resuming is necessary as of Chrome 55+ in some cases like different
			// domain page in an iframe.
			p.context.Call("resume")
			p.context.Call("createBufferSource").Call("start", 0)
			p.positionInSamples = int64(p.context.Get("currentTime").Float() * float64(p.sampleRate))
		})
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
	p.bufferedData = append(p.bufferedData, data...)
	c := int64(p.context.Get("currentTime").Float() * float64(p.sampleRate))
	if p.positionInSamples < c {
		p.positionInSamples = c
	}
	// Heuristic data size which doesn't cause too much noise and too much delay (#299)
	dataSize := int(float64(p.sampleRate)/13.5) / 4 * 4
	for dataSize <= len(p.bufferedData) {
		data := p.bufferedData[:dataSize]
		size := len(data) / p.bytesPerSample / p.channelNum
		// TODO: size must be const or you'll get noise (e.g. sample rate is 22050)
		buf := p.context.Call("createBuffer", p.channelNum, size, p.sampleRate)
		l := buf.Call("getChannelData", 0)
		r := buf.Call("getChannelData", 1)
		il, ir := toLR(data)
		const max = 1 << 15
		for i := 0; i < len(il); i++ {
			l.SetIndex(i, float64(il[i])/max)
			r.SetIndex(i, float64(ir[i])/max)
		}
		s := p.context.Call("createBufferSource")
		s.Set("buffer", buf)
		s.Call("connect", p.context.Get("destination"))
		s.Call("start", float64(p.positionInSamples)/float64(p.sampleRate))
		p.positionInSamples += int64(len(il))
		p.bufferedData = p.bufferedData[dataSize:]
	}
	return nil
}

func (p *Player) Close() error {
	return nil
}
