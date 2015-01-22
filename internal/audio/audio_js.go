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

// Keep this so as not to be destroyed by GC.
var node js.Object
var context js.Object

const bufferSize = 1024
const SampleRate = 44100

type channel struct {
	l []float32
	r []float32
}

var channels = make([]*channel, 16)

func init() {
	for i, _ := range channels {
		channels[i] = &channel{
			l: []float32{},
			r: []float32{},
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var currentBytes = 0

func CurrentBytes() int {
	return currentBytes
}

func Init() {
	context = js.Global.Get("AudioContext").New()
	// TODO: ScriptProcessorNode will be replaced with Audio WebWorker.
	// https://developer.mozilla.org/ja/docs/Web/API/ScriptProcessorNode
	node = context.Call("createScriptProcessor", bufferSize, 0, 2)
	node.Call("addEventListener", "audioprocess", func(e js.Object) {
		defer func() {
			currentBytes += bufferSize
		}()

		l := e.Get("outputBuffer").Call("getChannelData", 0)
		r := e.Get("outputBuffer").Call("getChannelData", 1)
		inputL := make([]float32, bufferSize)
		inputR := make([]float32, bufferSize)
		for _, ch := range channels {
			if len(ch.l) == 0 {
				continue
			}
			l := min(len(ch.l), bufferSize)
			for i := 0; i < l; i++ {
				inputL[i] += ch.l[i]
				inputR[i] += ch.r[i]
			}
			// TODO: Use copyFromChannel?
			usedLen := min(bufferSize, len(ch.l))
			ch.l = ch.l[usedLen:]
			ch.r = ch.r[usedLen:]
		}
		for i := 0; i < bufferSize; i++ {
			// TODO: Use copyFromChannel?
			if len(inputL) <= i {
				l.SetIndex(i, 0)
				r.SetIndex(i, 0)
				continue
			}
			l.SetIndex(i, inputL[i])
			r.SetIndex(i, inputR[i])
		}
	})
}

func Start() {
	// TODO: For iOS, node should be connected with a buffer node.
	node.Call("connect", context.Get("destination"))
}

func channelAt(i int) *channel {
	if i == -1 {
		for _, ch := range channels {
			if 0 < len(ch.l) {
				continue
			}
			return ch
		}
		return nil
	}
	ch := channels[i]
	// TODO: Can we append even though all data is not consumed? Need game timer?
	if 0 < len(ch.l) {
		return nil
	}
	return ch
}

func Append(i int, l []float32, r []float32) bool {
	// TODO: Mutex (especially for OpenAL)
	if len(l) != len(r) {
		panic("len(l) must equal to len(r)")
	}
	ch := channelAt(i)
	if ch == nil {
		return false
	}
	ch.l = append(ch.l, l...)
	ch.r = append(ch.r, r...)
	return true
}
