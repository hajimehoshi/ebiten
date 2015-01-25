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

// +build !js

package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/timshannon/go-openal/openal"
	"log"
	"runtime"
	"time"
)

func toBytes(l, r []int16) []byte {
	if len(l) != len(r) {
		panic("len(l) must equal to len(r)")
	}
	b := &bytes.Buffer{}
	for i, _ := range l {
		if err := binary.Write(b, binary.LittleEndian, []int16{l[i], r[i]}); err != nil {
			panic(err)
		}
	}
	return b.Bytes()
}

type sample struct {
	l []float32
	r []float32
}

func initialize() {
	ch := make(chan struct{})
	go func() {
		runtime.LockOSThread()

		device := openal.OpenDevice("")
		context := device.CreateContext()
		context.Activate()

		if alErr := openal.GetError(); alErr != 0 {
			log.Printf("OpenAL initialize error: %d", alErr)
			close(ch)
			return
		}

		audioEnabled = true
		source := openal.NewSource()
		close(ch)

		emptyBytes := make([]byte, 4*bufferSize)

		const bufferNum = 16
		buffers := openal.NewBuffers(bufferNum)
		for _, buffer := range buffers {
			buffer.SetData(openal.FormatStereo16, emptyBytes, SampleRate)
			source.QueueBuffer(buffer)
		}
		source.Play()
		if alErr := openal.GetError(); alErr != 0 {
			panic(fmt.Sprintf("OpenAL error: %d", alErr))
		}

		for {
			if source.State() != openal.Playing {
				panic(fmt.Sprintf("invalid source state: %d (0x%[1]x)", source.State()))
			}
			processed := source.BuffersProcessed()
			if processed == 0 {
				time.Sleep(1)
				continue
			}
			buffers := make([]openal.Buffer, processed)
			source.UnqueueBuffers(buffers)
			for _, buffer := range buffers {
				l, r := loadChannelBuffers()
				b := toBytes(l, r)
				buffer.SetData(openal.FormatStereo16, b, SampleRate)
				source.QueueBuffer(buffer)
				nextInsertionPosition -= min(bufferSize, nextInsertionPosition)
			}
		}
	}()
	<-ch
}

func start() {
	// Do nothing
}
