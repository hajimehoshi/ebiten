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

import (
	"sync"
)

var audioEnabled = false

const SampleRate = 44100

var currentPosition = 0

type channel struct {
	l                     []int16
	r                     []int16
	nextInsertionPosition int
}

var MaxChannel = 32

var channels = make([]*channel, MaxChannel)

// NOTE: In GopherJS, sync.Mutex blocks a function and requires gopherjs:blocking comments.
var channelsLock sync.Mutex

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

func Play(channel int, l []int16, r []int16) bool {
	channelsLock.Lock()
	defer channelsLock.Unlock()

	if !audioEnabled {
		return false
	}

	if len(l) != len(r) {
		panic("len(l) must equal to len(r)")
	}
	ch := channelAt(channel)
	if ch == nil {
		return false
	}
	ch.l = append(ch.l, make([]int16, ch.nextInsertionPosition-len(ch.l))...)
	ch.r = append(ch.r, make([]int16, ch.nextInsertionPosition-len(ch.r))...)
	ch.l = append(ch.l, l...)
	ch.r = append(ch.r, r...)
	return true
}

func Queue(channel int, l []int16, r []int16) {
	channelsLock.Lock()
	defer channelsLock.Unlock()

	if !audioEnabled {
		return
	}

	if len(l) != len(r) {
		panic("len(l) must equal to len(r)")
	}
	ch := channels[channel]
	ch.l = append(ch.l, l...)
	ch.r = append(ch.r, r...)
}

func Tick() {
	for _, ch := range channels {
		ch.nextInsertionPosition += SampleRate / 60
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isChannelsEmpty() bool {
	channelsLock.Lock()
	defer channelsLock.Unlock()

	if !audioEnabled {
		return true
	}

	for _, ch := range channels {
		if 0 < len(ch.l) {
			return false
		}
	}
	return true
}

func loadChannelBuffer(channel int, bufferSize int) (l, r []int16) {
	channelsLock.Lock()
	defer channelsLock.Unlock()

	if !audioEnabled {
		return nil, nil
	}

	ch := channels[channel]
	length := min(len(ch.l), bufferSize)
	inputL := make([]int16, length)
	inputR := make([]int16, length)
	copy(inputL, ch.l[:length])
	copy(inputR, ch.r[:length])
	ch.l = ch.l[length:]
	ch.r = ch.r[length:]
	ch.nextInsertionPosition -= min(bufferSize, ch.nextInsertionPosition)
	return inputL, inputR
}

func IsPlaying(channel int) bool {
	channelsLock.Lock()
	defer channelsLock.Unlock()

	if !audioEnabled {
		return false
	}

	return isPlaying(channel)
}
