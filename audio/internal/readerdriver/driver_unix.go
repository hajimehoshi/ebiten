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

// +build aix dragonfly freebsd hurd illumos linux netbsd openbsd solaris
// +build !android

package readerdriver

// #cgo pkg-config: libpulse
// #cgo LDFLAGS: -lpulse
//
// #include <pulse/pulseaudio.h>
//
// void ebiten_readerdriver_contextStateCallback(pa_context *context, void *userdata);
// void ebiten_readerdriver_streamWriteCallback(pa_stream *stream, size_t requested_bytes, void *userdata);
// void ebiten_readerdriver_streamStateCallback(pa_stream *stream, void *userdata);
// void ebiten_readerdriver_streamSuccessCallback(pa_stream *stream, void *userdata);
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

func IsAvailable() bool {
	return true
}

type context struct {
	sampleRate      int
	channelNum      int
	bitDepthInBytes int

	mainloop *C.pa_threaded_mainloop
	context  *C.pa_context
	stream   *C.pa_stream

	players *players
}

var theContext *context

const bufferSize = 4096

func NewContext(sampleRate, channelNum, bitDepthInBytes int) (Context, chan struct{}, error) {
	ready := make(chan struct{})
	close(ready)

	c := &context{
		sampleRate:      sampleRate,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
		players:         newPlayers(),
	}
	theContext = c

	c.mainloop = C.pa_threaded_mainloop_new()
	if c.mainloop == nil {
		return nil, nil, fmt.Errorf("readerdriver: pa_threaded_mainloop_new failed")
	}
	mainloopAPI := C.pa_threaded_mainloop_get_api(c.mainloop)
	if mainloopAPI == nil {
		return nil, nil, fmt.Errorf("readerdriver: pa_threaded_mainloop_get_api failed")
	}

	contextName := C.CString("pcm-playback")
	defer C.free(unsafe.Pointer(contextName))
	c.context = C.pa_context_new(mainloopAPI, contextName)
	if c.context == nil {
		return nil, nil, fmt.Errorf("readerdriver: pa_context_new failed")
	}

	C.pa_context_set_state_callback(c.context, C.pa_context_notify_cb_t(C.ebiten_readerdriver_contextStateCallback), unsafe.Pointer(c.mainloop))

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	C.pa_threaded_mainloop_lock(c.mainloop)
	defer C.pa_threaded_mainloop_unlock(c.mainloop)

	if code := C.pa_threaded_mainloop_start(c.mainloop); code != 0 {
		return nil, nil, fmt.Errorf("readerdriver: pa_threaded_mainloop_start failed: %s", C.GoString(C.pa_strerror(code)))
	}
	if code := C.pa_context_connect(c.context, nil, C.PA_CONTEXT_NOAUTOSPAWN, nil); code != 0 {
		return nil, nil, fmt.Errorf("readerdriver: pa_context_connect failed: %s", C.GoString(C.pa_strerror(code)))
	}

	// Wait until the context is ready.
	for {
		contextState := C.pa_context_get_state(c.context)
		if C.PA_CONTEXT_IS_GOOD(contextState) == 0 {
			return nil, nil, fmt.Errorf("readerdriver: context state is bad")
		}
		if contextState == C.PA_CONTEXT_READY {
			break
		}
		C.pa_threaded_mainloop_wait(c.mainloop)
	}

	sampleSpecificatiom := C.pa_sample_spec{
		format:   C.PA_SAMPLE_FLOAT32LE,
		rate:     C.uint(sampleRate),
		channels: C.uchar(channelNum),
	}
	var m C.pa_channel_map
	switch channelNum {
	case 1:
		C.pa_channel_map_init_mono(&m)
	case 2:
		C.pa_channel_map_init_stereo(&m)
	}

	streamName := C.CString("Playback")
	defer C.free(unsafe.Pointer(streamName))
	c.stream = C.pa_stream_new(c.context, streamName, &sampleSpecificatiom, &m)
	C.pa_stream_set_state_callback(c.stream, C.pa_stream_notify_cb_t(C.ebiten_readerdriver_streamStateCallback), unsafe.Pointer(c.mainloop))
	C.pa_stream_set_write_callback(c.stream, C.pa_stream_request_cb_t(C.ebiten_readerdriver_streamWriteCallback), nil)

	const defaultValue = 0xffffffff
	bufferAttr := C.pa_buffer_attr{
		maxlength: defaultValue,
		tlength:   bufferSize,
		prebuf:    defaultValue,
		minreq:    defaultValue,
	}
	var streamFlags C.pa_stream_flags_t = C.PA_STREAM_START_CORKED | C.PA_STREAM_INTERPOLATE_TIMING |
		C.PA_STREAM_NOT_MONOTONIC | C.PA_STREAM_AUTO_TIMING_UPDATE |
		C.PA_STREAM_ADJUST_LATENCY

	if code := C.pa_stream_connect_playback(c.stream, nil, &bufferAttr, streamFlags, nil, nil); code != 0 {
		return nil, nil, fmt.Errorf("readerdriver: pa_stream_connect_playback failed: %s", C.GoString(C.pa_strerror(code)))
	}

	// Wait until the stream is ready.
	for {
		streamState := C.pa_stream_get_state(c.stream)
		if C.PA_STREAM_IS_GOOD(streamState) == 0 {
			return nil, nil, fmt.Errorf("readerdriver: stream state is bad")
		}
		if streamState == C.PA_STREAM_READY {
			break
		}
		C.pa_threaded_mainloop_wait(c.mainloop)
	}

	C.pa_stream_cork(c.stream, 0, C.pa_stream_success_cb_t(C.ebiten_readerdriver_streamSuccessCallback), unsafe.Pointer(c.mainloop))

	go c.players.loop()

	return c, ready, nil
}

func (c *context) Suspend() error {
	C.pa_stream_cork(c.stream, 1, C.pa_stream_success_cb_t(C.ebiten_readerdriver_streamSuccessCallback), unsafe.Pointer(c.mainloop))
	return nil
}

func (c *context) Resume() error {
	C.pa_stream_cork(c.stream, 0, C.pa_stream_success_cb_t(C.ebiten_readerdriver_streamSuccessCallback), unsafe.Pointer(c.mainloop))
	return nil
}

//export ebiten_readerdriver_contextStateCallback
func ebiten_readerdriver_contextStateCallback(context *C.pa_context, mainloop unsafe.Pointer) {
	C.pa_threaded_mainloop_signal((*C.pa_threaded_mainloop)(mainloop), 0)
}

//export ebiten_readerdriver_streamStateCallback
func ebiten_readerdriver_streamStateCallback(stream *C.pa_stream, mainloop unsafe.Pointer) {
	C.pa_threaded_mainloop_signal((*C.pa_threaded_mainloop)(mainloop), 0)
}

//export ebiten_readerdriver_streamSuccessCallback
func ebiten_readerdriver_streamSuccessCallback(stream *C.pa_stream, userdata unsafe.Pointer) {
}

//export ebiten_readerdriver_streamWriteCallback
func ebiten_readerdriver_streamWriteCallback(stream *C.pa_stream, requestedBytes C.size_t, userdata unsafe.Pointer) {
	c := theContext

	var buf unsafe.Pointer
	var buf32 []float32
	var bytesToFill C.size_t = bufferSize
	for n := int(requestedBytes); n > 0; n -= int(bytesToFill) {
		C.pa_stream_begin_write(stream, &buf, &bytesToFill)
		if len(buf32) < int(bytesToFill)/4 {
			buf32 = make([]float32, bytesToFill/4)
		} else {
			for i := 0; i < int(bytesToFill)/4; i++ {
				buf32[i] = 0
			}
		}

		c.players.read(buf32[:bytesToFill/4])

		for i := uintptr(0); i < uintptr(bytesToFill/4); i++ {
			*(*float32)(unsafe.Pointer(uintptr(buf) + 4*i)) = buf32[i]
		}

		C.pa_stream_write(stream, buf, bytesToFill, nil, 0, C.PA_SEEK_RELATIVE)
	}
}
