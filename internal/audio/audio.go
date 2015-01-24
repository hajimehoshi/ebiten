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

package audio

const bufferSize = 1024
const SampleRate = 44100

var nextInsertion = 0
var currentBytes = 0

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

func Init() {
	initialize()
}

func Start() {
	start()
}

func Append(channel int, l []float32, r []float32) bool {
	// TODO: Mutex (especially for OpenAL)
	if len(l) != len(r) {
		panic("len(l) must equal to len(r)")
	}
	ch := channelAt(channel)
	if ch == nil {
		return false
	}
	ch.l = append(ch.l, make([]float32, nextInsertion-len(ch.l))...)
	ch.r = append(ch.r, make([]float32, nextInsertion-len(ch.r))...)
	ch.l = append(ch.l, l...)
	ch.r = append(ch.r, r...)
	return true
}

func CurrentBytes() int {
	return currentBytes + nextInsertion
}

func Update() {
	nextInsertion += SampleRate / 60
}

func channelAt(i int) *channel {
	if i == -1 {
		for _, ch := range channels {
			if len(ch.l) <= nextInsertion {
				return ch
			}
		}
		return nil
	}
	ch := channels[i]
	if len(ch.l) <= nextInsertion {
		return ch
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func loadChannelBuffers() (l, r []float32) {
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
	return inputL, inputR
}
