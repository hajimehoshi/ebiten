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

func initialize() {
	context = js.Global.Get("AudioContext").New()
	// TODO: ScriptProcessorNode will be replaced with Audio WebWorker.
	// https://developer.mozilla.org/ja/docs/Web/API/ScriptProcessorNode
	node = context.Call("createScriptProcessor", bufferSize, 0, 2)
	node.Call("addEventListener", "audioprocess", func(e js.Object) {
		defer func() {
			currentPosition += bufferSize
		}()

		l := e.Get("outputBuffer").Call("getChannelData", 0)
		r := e.Get("outputBuffer").Call("getChannelData", 1)
		inputL, inputR := loadChannelBuffers()
		nextInsertionPosition -= min(bufferSize, nextInsertionPosition)
		const max = 1 << 15
		for i := 0; i < bufferSize; i++ {
			// TODO: Use copyFromChannel?
			if len(inputL) <= i {
				l.SetIndex(i, 0)
				r.SetIndex(i, 0)
				continue
			}
			l.SetIndex(i, float64(inputL[i])/max)
			r.SetIndex(i, float64(inputR[i])/max)
		}
	})
	audioEnabled = true
}

func start() {
	// TODO: For iOS, node should be connected with a buffer node.
	node.Call("connect", context.Get("destination"))
}
