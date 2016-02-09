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
	"github.com/gopherjs/gopherjs/js"
)

var context *js.Object

type audioProcessor struct {
	channel    int
	sampleRate int
	position   float64
}

var audioProcessors [MaxChannel]*audioProcessor

func toLR(data []byte) ([]int16, []int16) {
	l := make([]int16, len(data)/4)
	r := make([]int16, len(data)/4)
	for i := 0; i < len(data)/4; i++ {
		l[i] = int16(data[4*i]) | int16(data[4*i+1])<<8
		r[i] = int16(data[4*i+2]) | int16(data[4*i+3])<<8
	}
	return l, r
}

func (a *audioProcessor) playChunk(buf []byte) {
	if len(buf) == 0 {
		return
	}
	const channelNum = 2
	const bytesPerSample = channelNum * 16 / 8
	b := context.Call("createBuffer", channelNum, len(buf)/bytesPerSample, a.sampleRate)
	l := b.Call("getChannelData", 0)
	r := b.Call("getChannelData", 1)
	il, ir := toLR(buf)
	const max = 1 << 15
	for i := 0; i < len(il); i++ {
		l.SetIndex(i, float64(il[i])/max)
		r.SetIndex(i, float64(ir[i])/max)
	}
	s := context.Call("createBufferSource")
	s.Set("buffer", b)
	s.Call("connect", context.Get("destination"))

	s.Call("start", a.position)
	a.position += float64(len(il)) / float64(a.sampleRate)
}

func isPlaying(channel int) bool {
	ch := channels[channel]
	if 0 < len(ch.buffer) {
		return true
	}
	a := audioProcessors[channel]
	c := context.Get("currentTime").Float()
	return c < a.position
}

func tick() {
	const bufferSize = 1024
	c := context.Get("currentTime").Float()
	for _, a := range audioProcessors {
		if a.position < c {
			a.position = c
		}
		// TODO: 4 is a magic number
		a.playChunk(loadChannelBuffer(a.channel, bufferSize*4))
	}
}

func initialize() {
	// Do nothing in node.js.
	if js.Global.Get("require") != js.Undefined {
		return
	}

	class := js.Global.Get("AudioContext")
	if class == js.Undefined {
		class = js.Global.Get("webkitAudioContext")
	}
	if class == js.Undefined {
		return
	}
	context = class.New()
	audioEnabled = true
	for i := 0; i < len(audioProcessors); i++ {
		audioProcessors[i] = &audioProcessor{
			channel:    i,
			sampleRate: 44100, // TODO: Change this for each chunks
			position:   0,
		}
	}
}
