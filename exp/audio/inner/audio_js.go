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

package internal

import (
	"github.com/gopherjs/gopherjs/js"
)

func withChannels(f func()) {
	f()
}

// Keep this so as not to be destroyed by GC.
var (
	nodes   = []*js.Object{}
	dummies = []*js.Object{} // Dummy source nodes for iOS.
	context *js.Object
)

const bufferSize = 1024

type audioProcessor struct {
	channel int
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

func (a *audioProcessor) Process(e *js.Object) {
	// Can't use 'go' here. Probably it may cause race conditions.
	b := e.Get("outputBuffer")
	l := b.Call("getChannelData", 0)
	r := b.Call("getChannelData", 1)
	inputL, inputR := toLR(loadChannelBuffer(a.channel, bufferSize*4))
	const max = 1 << 15
	for i := 0; i < len(inputL); i++ {
		// TODO: Use copyToChannel?
		l.SetIndex(i, float64(inputL[i])/max)
		r.SetIndex(i, float64(inputR[i])/max)
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
	// TODO: ScriptProcessorNode will be replaced with Audio WebWorker.
	// https://developer.mozilla.org/ja/docs/Web/API/ScriptProcessorNode
	for i := 0; i < MaxChannel; i++ {
		node := context.Call("createScriptProcessor", bufferSize, 0, 2)
		processor := &audioProcessor{i}
		node.Call("addEventListener", "audioprocess", processor.Process)
		nodes = append(nodes, node)

		dummy := context.Call("createBufferSource")
		dummies = append(dummies, dummy)
	}
	audioEnabled = true
}

func start() {
	// Do nothing in node.js.
	if js.Global.Get("require") != js.Undefined {
		return
	}

	destination := context.Get("destination")
	for i, node := range nodes {
		dummy := dummies[i]
		dummy.Call("connect", node)
		node.Call("connect", destination)
	}
}
