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

var (
	bufferL = make([]float32, 0)
	bufferR = make([]float32, 0)
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Init() {
	context = js.Global.Get("AudioContext").New()
	// TODO: ScriptProcessorNode will be replaced Audio WebWorker.
	// https://developer.mozilla.org/ja/docs/Web/API/ScriptProcessorNode
	const bufLen = 1024
	node = context.Call("createScriptProcessor", bufLen, 0, 2)
	node.Call("addEventListener", "audioprocess", func(e js.Object) {
		l := e.Get("outputBuffer").Call("getChannelData", 0)
		r := e.Get("outputBuffer").Call("getChannelData", 1)
		for i := 0; i < bufLen; i++ {
			// TODO: Use copyFromChannel?
			if len(bufferL) <= i {
				l.SetIndex(i, 0)
				r.SetIndex(i, 0)
				continue
			}
			l.SetIndex(i, bufferL[i])
			r.SetIndex(i, bufferR[i])
		}
		// TODO: Will the array heads be released properly on GopherJS?
		usedLen := min(bufLen, len(bufferL))
		bufferL = bufferL[usedLen:]
		bufferR = bufferR[usedLen:]
	})
}

func Start() {
	// TODO: For iOS, node should be connected with a buffer node.
	node.Call("connect", context.Get("destination"))
}

func Append(l []float32, r []float32) {
	if len(l) != len(r) {
		panic("len(l) must equal to len(r)")
	}
	bufferL = append(bufferL, l...)
	bufferR = append(bufferR, r...)
}

func Add(l []float32, r []float32) {
	// TODO: Adjust timing for frame?
	if len(l) != len(r) {
		panic("len(l) must equal to len(r)")
	}
	m := min(len(l), len(bufferL))
	for i := 0; i < m; i++ {
		bufferL[i] += l[i]
		bufferR[i] += r[i]
	}
	if m < len(l) {
		bufferL = append(bufferL, l[m:]...)
		bufferR = append(bufferR, r[m:]...)
	}
}
