// Copyright 2021 The Ebiten Authors
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

package readerdriver

// #cgo LDFLAGS: -framework AudioToolbox
//
// #import <AudioToolbox/AudioToolbox.h>
//
// void ebiten_readerdriver_render(void* inUserData, AudioQueueRef inAQ, AudioQueueBufferRef inBuffer);
//
// void ebiten_readerdriver_setNotificationHandler();
import "C"

import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

const (
	float32SizeInBytes = 4
)

func newAudioQueue(sampleRate, channelNum, bitDepthInBytes int) (C.AudioQueueRef, []C.AudioQueueBufferRef, error) {
	desc := C.AudioStreamBasicDescription{
		mSampleRate:       C.double(sampleRate),
		mFormatID:         C.kAudioFormatLinearPCM,
		mFormatFlags:      C.kAudioFormatFlagIsFloat,
		mBytesPerPacket:   C.UInt32(channelNum * float32SizeInBytes),
		mFramesPerPacket:  1,
		mBytesPerFrame:    C.UInt32(channelNum * float32SizeInBytes),
		mChannelsPerFrame: C.UInt32(channelNum),
		mBitsPerChannel:   C.UInt32(8 * float32SizeInBytes),
	}

	var audioQueue C.AudioQueueRef
	if osstatus := C.AudioQueueNewOutput(
		&desc,
		(C.AudioQueueOutputCallback)(C.ebiten_readerdriver_render),
		nil,
		(C.CFRunLoopRef)(0),
		(C.CFStringRef)(0),
		0,
		&audioQueue); osstatus != C.noErr {
		return nil, nil, fmt.Errorf("readerdriver: AudioQueueNewFormat with StreamFormat failed: %d", osstatus)
	}

	bufs := make([]C.AudioQueueBufferRef, 0, 4)
	for len(bufs) < cap(bufs) {
		var buf C.AudioQueueBufferRef
		if osstatus := C.AudioQueueAllocateBuffer(audioQueue, bufferSizeInBytes, &buf); osstatus != C.noErr {
			return nil, nil, fmt.Errorf("readerdriver: AudioQueueAllocateBuffer failed: %d", osstatus)
		}
		buf.mAudioDataByteSize = bufferSizeInBytes
		bufs = append(bufs, buf)
	}

	return audioQueue, bufs, nil
}

type context struct {
	sampleRate      int
	channelNum      int
	bitDepthInBytes int

	audioQueue      C.AudioQueueRef
	unqueuedBuffers []C.AudioQueueBufferRef

	cond *sync.Cond

	players *players
}

// TOOD: Convert the error code correctly.
// See https://stackoverflow.com/questions/2196869/how-do-you-convert-an-iphone-osstatus-code-to-something-useful

var theContext *context

func newContext(sampleRate, channelNum, bitDepthInBytes int) (*context, chan struct{}, error) {
	ready := make(chan struct{})
	close(ready)

	c := &context{
		sampleRate:      sampleRate,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
		cond:            sync.NewCond(&sync.Mutex{}),
		players:         newPlayers(),
	}
	theContext = c

	q, bs, err := newAudioQueue(sampleRate, channelNum, bitDepthInBytes)
	if err != nil {
		return nil, nil, err
	}
	c.audioQueue = q
	c.unqueuedBuffers = bs

	C.ebiten_readerdriver_setNotificationHandler()

	if osstatus := C.AudioQueueStart(c.audioQueue, nil); osstatus != C.noErr {
		return nil, nil, fmt.Errorf("readerdriver: AudioQueueStart failed: %d", osstatus)
	}

	go c.loop()

	return c, ready, nil
}

func (c *context) wait() {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	for len(c.unqueuedBuffers) == 0 {
		c.cond.Wait()
	}
}

func (c *context) loop() {
	buf32 := make([]float32, bufferSizeInBytes/4)
	for {
		c.wait()
		c.appendBuffer(buf32)
	}
}

func (c *context) appendBuffer(buf32 []float32) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	buf := c.unqueuedBuffers[0]
	c.unqueuedBuffers = c.unqueuedBuffers[1:]

	for i := range buf32 {
		buf32[i] = 0
	}
	c.players.read(buf32)
	for i, f := range buf32 {
		*(*float32)(unsafe.Pointer(uintptr(buf.mAudioData) + uintptr(i)*float32SizeInBytes)) = f
	}

	if osstatus := C.AudioQueueEnqueueBuffer(c.audioQueue, buf, 0, nil); osstatus != C.noErr {
		// TODO: Treat the error correctly
		panic(fmt.Errorf("readerdriver: AudioQueueEnqueueBuffer failed: %d", osstatus))
	}
}

func (c *context) Suspend() error {
	if osstatus := C.AudioQueuePause(c.audioQueue); osstatus != C.noErr {
		return fmt.Errorf("readerdriver: AudioQueuePause failed: %d", osstatus)
	}
	return nil
}

func (c *context) Resume() error {
try:
	if osstatus := C.AudioQueueStart(c.audioQueue, nil); osstatus != C.noErr {
		const AVAudioSessionErrorCodeSiriIsRecording = 0x73697269 // 'siri'
		if osstatus == AVAudioSessionErrorCodeSiriIsRecording {
			time.Sleep(10 * time.Millisecond)
			goto try
		}
		return fmt.Errorf("readerdriver: AudioQueueStart failed: %d", osstatus)
	}
	return nil
}

//export ebiten_readerdriver_render
func ebiten_readerdriver_render(inUserData unsafe.Pointer, inAQ C.AudioQueueRef, inBuffer C.AudioQueueBufferRef) {
	theContext.cond.L.Lock()
	defer theContext.cond.L.Unlock()
	theContext.unqueuedBuffers = append(theContext.unqueuedBuffers, inBuffer)
	theContext.cond.Signal()
}

//export ebiten_readerdriver_setGlobalPause
func ebiten_readerdriver_setGlobalPause() {
	theContext.Suspend()
}

//export ebiten_readerdriver_setGlobalResume
func ebiten_readerdriver_setGlobalResume() {
	theContext.Resume()
}
