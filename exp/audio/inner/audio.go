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
	buffer                []byte
	nextInsertionPosition int
}

var MaxChannel = 32

var channels = make([]*channel, MaxChannel)

func init() {
	for i, _ := range channels {
		channels[i] = &channel{
			buffer: []byte{},
		}
	}
}

func Init() {
	initialize()
	start()
}

func isPlaying(channel int) bool {
	ch := channels[channel]
	return ch.nextInsertionPosition < len(ch.buffer)
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

func Play(channel int, data []byte) bool {
	ch := channelAt(channel)
	if ch == nil {
		return false
	}
	withChannels(func() {
		if !audioEnabled {
			return
		}
		d := ch.nextInsertionPosition - len(data)
		if 0 < d {
			ch.buffer = append(ch.buffer, make([]byte, d)...)
		}
		ch.buffer = append(ch.buffer, data...)
	})
	return true
}

func Queue(channel int, data []byte) {
	withChannels(func() {
		if !audioEnabled {
			return
		}
		ch := channels[channel]
		ch.buffer = append(ch.buffer, data...)
	})
}

func Tick() {
	for _, ch := range channels {
		ch.nextInsertionPosition += SampleRate * 4 / 60
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
			if 0 < len(ch.buffer) {
				return
			}
		}
		result = true
		return
	})
	return result
}

func loadChannelBuffer(channel int, bufferSize int) []byte {
	var r []byte
	withChannels(func() {
		if !audioEnabled {
			return
		}

		ch := channels[channel]
		length := min(len(ch.buffer), bufferSize)
		input := ch.buffer[:length]
		ch.buffer = ch.buffer[length:]

		ch.nextInsertionPosition -= bufferSize
		if ch.nextInsertionPosition < 0 {
			ch.nextInsertionPosition = 0
		}

		r = input
	})
	return r
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
