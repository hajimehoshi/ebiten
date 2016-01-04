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

// +build !js,!windows

package internal

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/timshannon/go-openal/openal"
)

var channelsMutex = sync.Mutex{}

func withChannels(f func()) {
	channelsMutex.Lock()
	defer channelsMutex.Unlock()
	f()
}

func initialize() {
	// Creating OpenAL device must be done after initializing UI. I'm not sure the reason.
	ch := make(chan struct{})
	go func() {
		runtime.LockOSThread()

		device := openal.OpenDevice("")
		context := device.CreateContext()
		context.Activate()

		if err := openal.Err(); err != nil {
			log.Printf("OpenAL initialize error: %v", err)
			close(ch)
			// Graceful ending: Audio is not available on Travis CI.
			return
		}

		audioEnabled = true
		sources := openal.NewSources(MaxChannel)
		close(ch)

		const bufferSize = 2048
		emptyBytes := make([]byte, bufferSize)

		for _, source := range sources {
			// 3 is the least number?
			// http://stackoverflow.com/questions/14932004/play-sound-with-openalstream
			const bufferNum = 4
			buffers := openal.NewBuffers(bufferNum)
			for _, buffer := range buffers {
				buffer.SetData(openal.FormatStereo16, emptyBytes, SampleRate)
				source.QueueBuffer(buffer)
			}
			source.Play()
			if err := openal.Err(); err != nil {
				panic(fmt.Sprintf("OpenAL error: %v", err))
			}
		}

		for {
			oneProcessed := false
			for ch, source := range sources {
				processed := source.BuffersProcessed()
				if processed == 0 {
					continue
				}

				oneProcessed = true
				buffers := make([]openal.Buffer, processed)
				source.UnqueueBuffers(buffers)
				for _, buffer := range buffers {
					b := make([]byte, bufferSize)
					copy(b, loadChannelBuffer(ch, bufferSize))
					buffer.SetData(openal.FormatStereo16, b, SampleRate)
					source.QueueBuffer(buffer)
				}
				if source.State() == openal.Stopped {
					source.Rewind()
					source.Play()
				}
			}
			if !oneProcessed {
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()
	<-ch
}

func start() {
	// Do nothing
}
