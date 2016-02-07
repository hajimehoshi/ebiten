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

var audioEnabled = false

const SampleRate = 44100

type channel struct {
	buffer                []byte
	nextInsertionPosition int
}

const MaxChannel = 32

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
	if i == -1 {
		for i, _ := range channels {
			if !isPlaying(i) {
				return channels[i]
			}
		}
		return nil
	}
	if !isPlaying(i) {
		return channels[i]
	}
	return nil
}

func Queue(channel int, data []byte) bool {
	result := true
	withChannels(func() {
		if !audioEnabled {
			result = false
			return
		}
		ch := channelAt(channel)
		if ch == nil {
			result = false
			return
		}
		ch.buffer = append(ch.buffer, data...)
	})
	return result
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
