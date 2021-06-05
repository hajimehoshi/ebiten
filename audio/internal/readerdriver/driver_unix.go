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
	"io"
	"runtime"
	"sync"
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

	players map[*playerImpl]struct{}
	buf     []float32
	cond    *sync.Cond
}

var theContext *context

func NewContext(sampleRate, channelNum, bitDepthInBytes int) (Context, chan struct{}, error) {
	ready := make(chan struct{})
	close(ready)

	c := &context{
		sampleRate:      sampleRate,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
		cond:            sync.NewCond(&sync.Mutex{}),
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

	if C.pa_threaded_mainloop_start(c.mainloop) != 0 {
		return nil, nil, fmt.Errorf("readerdriver: pa_threaded_mainloop_start failed")
	}
	if C.pa_context_connect(c.context, nil, C.PA_CONTEXT_NOAUTOSPAWN, nil) != 0 {
		return nil, nil, fmt.Errorf("readerdriver: pa_context_connect failed")
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
		tlength:   2048,
		prebuf:    defaultValue,
		minreq:    defaultValue,
	}
	var streamFlags C.pa_stream_flags_t = C.PA_STREAM_START_CORKED | C.PA_STREAM_INTERPOLATE_TIMING |
		C.PA_STREAM_NOT_MONOTONIC | C.PA_STREAM_AUTO_TIMING_UPDATE |
		C.PA_STREAM_ADJUST_LATENCY

	if C.pa_stream_connect_playback(c.stream, nil, &bufferAttr, streamFlags, nil, nil) != 0 {
		return nil, nil, fmt.Errorf("readerdriver: pa_stream_connect_playback failed")
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

	go c.loop()

	return c, ready, nil
}

func (c *context) shouldWait() bool {
	for p := range c.players {
		if !p.isBufferFull() {
			return false
		}
	}
	return true
}

func (c *context) wait() {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	for c.shouldWait() {
		c.cond.Wait()
	}
}

func (c *context) loop() {
	var players []*playerImpl
	for {
		c.wait()

		c.cond.L.Lock()
		players = players[:0]
		for p := range c.players {
			players = append(players, p)
		}
		c.cond.L.Unlock()

		for _, p := range players {
			p.load()
		}
	}
}

func (c *context) Suspend() error {
	C.pa_stream_cork(c.stream, 1, C.pa_stream_success_cb_t(C.ebiten_readerdriver_streamSuccessCallback), unsafe.Pointer(c.mainloop))
	return nil
}

func (c *context) Resume() error {
	C.pa_stream_cork(c.stream, 0, C.pa_stream_success_cb_t(C.ebiten_readerdriver_streamSuccessCallback), unsafe.Pointer(c.mainloop))
	return nil
}

func (c *context) addPlayer(player *playerImpl) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	if c.players == nil {
		c.players = map[*playerImpl]struct{}{}
	}
	c.players[player] = struct{}{}
	c.cond.Signal()
}

func (c *context) removePlayer(player *playerImpl) {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	delete(c.players, player)
	c.cond.Signal()
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
	var bytesToFill C.size_t = 256
	var players []*playerImpl
	for n := int(requestedBytes); n > 0; n -= int(bytesToFill) {
		c.cond.L.Lock()
		players = players[:0]
		for p := range c.players {
			players = append(players, p)
		}
		c.cond.L.Unlock()

		C.pa_stream_begin_write(stream, &buf, &bytesToFill)
		if len(buf32) < int(bytesToFill)/4 {
			buf32 = make([]float32, bytesToFill/4)
		} else {
			for i := 0; i < int(bytesToFill)/4; i++ {
				buf32[i] = 0
			}
		}
		for _, p := range players {
			p.addBuffer(buf32[:bytesToFill/4])
		}
		c.cond.Signal()
		for i := uintptr(0); i < uintptr(bytesToFill/4); i++ {
			*(*float32)(unsafe.Pointer(uintptr(buf) + 4*i)) = buf32[i]
		}

		C.pa_stream_write(stream, buf, bytesToFill, nil, 0, C.PA_SEEK_RELATIVE)
	}
}

type player struct {
	p *playerImpl
}

type playerImpl struct {
	context *context
	src     io.Reader
	volume  float64
	err     error
	state   playerState
	buf     []byte

	m sync.Mutex
}

func (c *context) NewPlayer(src io.Reader) Player {
	p := &player{
		p: &playerImpl{
			context: c,
			src:     src,
			volume:  1,
		},
	}
	runtime.SetFinalizer(p, (*player).Close)
	return p
}

func (p *player) Err() error {
	return p.p.Err()
}

func (p *playerImpl) Err() error {
	p.m.Lock()
	defer p.m.Unlock()

	return p.err
}

func (p *player) Play() {
	p.p.Play()
}

func (p *playerImpl) Play() {
	p.m.Lock()
	defer p.m.Unlock()
	p.playImpl()
}

func (p *playerImpl) playImpl() {
	if p.err != nil {
		return
	}
	if p.state != playerPaused {
		return
	}

	buf := make([]byte, p.context.maxBufferSize())
	for len(p.buf) < p.context.maxBufferSize() {
		n, err := p.src.Read(buf)
		if err != nil && err != io.EOF {
			p.setErrorImpl(err)
			return
		}
		p.buf = append(p.buf, buf[:n]...)
		if err == io.EOF {
			break
		}
	}

	p.state = playerPlay

	p.m.Unlock()
	p.context.addPlayer(p)
	p.m.Lock()
}

func (p *player) Pause() {
	p.p.Pause()
}

func (p *playerImpl) Pause() {
	p.m.Lock()
	defer p.m.Unlock()
	p.pauseImpl()
}

func (p *playerImpl) pauseImpl() {
	if p.state != playerPlay {
		return
	}
	p.state = playerPaused
}

func (p *player) Reset() {
	p.p.Reset()
}

func (p *playerImpl) Reset() {
	p.m.Lock()
	defer p.m.Unlock()
	p.resetImpl()
}

func (p *playerImpl) resetImpl() {
	if p.state == playerClosed {
		return
	}
	p.state = playerPaused
	p.buf = p.buf[:0]
}

func (p *player) IsPlaying() bool {
	return p.p.IsPlaying()
}

func (p *playerImpl) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()
	return p.state == playerPlay
}

func (p *player) Volume() float64 {
	return p.p.Volume()
}

func (p *playerImpl) Volume() float64 {
	p.m.Lock()
	defer p.m.Unlock()
	return p.volume
}

func (p *player) SetVolume(volume float64) {
	p.p.SetVolume(volume)
}

func (p *playerImpl) SetVolume(volume float64) {
	p.m.Lock()
	defer p.m.Unlock()
	p.volume = volume
}

func (p *player) UnplayedBufferSize() int {
	return p.p.UnplayedBufferSize()
}

func (p *playerImpl) UnplayedBufferSize() int {
	p.m.Lock()
	defer p.m.Unlock()
	return len(p.buf)
}

func (p *player) Close() error {
	runtime.SetFinalizer(p, nil)
	return p.p.Close()
}

func (p *playerImpl) Close() error {
	p.m.Lock()
	defer p.m.Unlock()
	return p.closeImpl()
}

func (p *playerImpl) closeImpl() error {
	p.m.Unlock()
	p.context.removePlayer(p)
	p.m.Lock()

	if p.state == playerClosed {
		return nil
	}
	p.state = playerClosed
	p.buf = nil
	return p.err
}

func (p *playerImpl) addBuffer(buf []float32) int {
	p.m.Lock()

	if p.state != playerPlay {
		p.m.Unlock()
		return 0
	}

	bitDepthInBytes := p.context.bitDepthInBytes
	n := len(p.buf) / bitDepthInBytes
	if n > len(buf) {
		n = len(buf)
	}
	volume := float32(p.volume)
	src := p.buf[:n*bitDepthInBytes]
	p.buf = p.buf[n*bitDepthInBytes:]
	p.m.Unlock()

	for i := 0; i < n; i++ {
		var v float32
		switch bitDepthInBytes {
		case 1:
			v8 := src[i]
			v = float32(v8-(1<<7)) / (1 << 7)
		case 2:
			v16 := int16(src[2*i]) | (int16(src[2*i+1]) << 8)
			v = float32(v16) / (1 << 15)
		}
		buf[i] += v * volume
	}
	return n
}

func (p *playerImpl) isBufferFull() bool {
	p.m.Lock()
	defer p.m.Unlock()
	return len(p.buf) >= p.context.maxBufferSize()
}

func (p *playerImpl) load() {
	p.m.Lock()
	defer p.m.Unlock()

	if p.err != nil {
		return
	}
	if p.state == playerClosed {
		return
	}

	if len(p.buf) >= p.context.maxBufferSize() {
		return
	}
	buf := make([]byte, p.context.maxBufferSize())
	n, err := p.src.Read(buf)
	if err != nil && err != io.EOF {
		p.setErrorImpl(err)
		return
	}
	p.buf = append(p.buf, buf[:n]...)
	if err == io.EOF && len(p.buf) == 0 {
		p.resetImpl()
	}
}

func (p *playerImpl) setErrorImpl(err error) {
	p.err = err
	p.closeImpl()
}
