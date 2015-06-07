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

package internal

var audioEnabled = false

const SampleRate = 44100

type channel struct {
	l                     []int16
	r                     []int16
	nextInsertionPosition int
}

var MaxChannel = 32

var channels = make([]*channel, MaxChannel)

func init() {
	for i, _ := range channels {
		channels[i] = &channel{
			l: []int16{},
			r: []int16{},
		}
	}
}

func Init() {
	initialize()
	start()
}

func isPlaying(channel int) bool {
	ch := channels[channel]
	return ch.nextInsertionPosition < len(ch.l)
}

func channelAt(i int) *channel {
	var ch *channel
	withChannels(func() {
		if i == -1 {
			for i, _ := range channels {
				if !isPlaying(i) {
					ch = channels[i]
					return
				}
			}
			return
		}
		if !isPlaying(i) {
			ch = channels[i]
			return
		}
		return
	})
	return ch
}

func Play(channel int, l []int16, r []int16) bool {
	ch := channelAt(channel)
	if ch == nil {
		return false
	}
	withChannels(func() {
		if !audioEnabled {
			return
		}
		if len(l) != len(r) {
			panic("len(l) must equal to len(r)")
		}
		d := ch.nextInsertionPosition - len(l)
		if 0 < d {
			ch.l = append(ch.l, make([]int16, d)...)
			ch.r = append(ch.r, make([]int16, d)...)
		}
		ch.l = append(ch.l, l...)
		ch.r = append(ch.r, r...)
	})
	return true
}

func Queue(channel int, l []int16, r []int16) {
	withChannels(func() {
		if !audioEnabled {
			return
		}
		if len(l) != len(r) {
			panic("len(l) must equal to len(r)")
		}
		ch := channels[channel]
		ch.l = append(ch.l, l...)
		ch.r = append(ch.r, r...)
	})
}

func Tick() {
	for _, ch := range channels {
		if 0 < len(ch.l) {
			ch.nextInsertionPosition += SampleRate / 60 // FPS
		} else {
			ch.nextInsertionPosition = 0
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isChannelsEmpty() bool {
	result := false
	withChannels(func() {
		if !audioEnabled {
			result = true
			return
		}

		for _, ch := range channels {
			if 0 < len(ch.l) {
				return
			}
		}
		result = true
		return
	})
	return result
}

func loadChannelBuffer(channel int, bufferSize int) (l, r []int16) {
	withChannels(func() {
		if !audioEnabled {
			return
		}

		ch := channels[channel]
		length := min(len(ch.l), bufferSize)
		inputL := ch.l[:length]
		inputR := ch.r[:length]
		ch.nextInsertionPosition -= length
		if ch.nextInsertionPosition < 0 {
			ch.nextInsertionPosition = 0
		}
		ch.l = ch.l[length:]
		ch.r = ch.r[length:]
		l, r = inputL, inputR
	})
	return
}

func IsPlaying(channel int) bool {
	result := false
	withChannels(func() {
		if !audioEnabled {
			return
		}
		result = isPlaying(channel)
	})
	return result
}
