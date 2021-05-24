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

// #cgo pkg-config: portaudio-2.0
// #cgo LDFLAGS: -lportaudio
//
// #include "portaudio.h"
//
// int ebiten_readerdriver_streamCallback(void *input, void *output, unsigned long frameCount, PaStreamCallbackTimeInfo *timeInfo, PaStreamCallbackFlags statusFlags, void *userData);
import "C"

import (
	"fmt"
	"io"
	"runtime"
	"sync"
	"unsafe"
)

type paError struct {
	value C.PaError
	fname string
}

func (e *paError) Error() string {
	return fmt.Sprintf("PortAudio %s failed: %s", e.fname, C.GoString(C.Pa_GetErrorText(e.value)))
}

func paErrorToError(err C.PaError, fname string) error {
	if err == C.paNoError {
		return nil
	}
	return &paError{
		value: err,
		fname: fname,
	}
}

func IsAvailable() bool {
	return true
}

type context struct {
	sampleRate      int
	channelNum      int
	bitDepthInBytes int

	stream  unsafe.Pointer // *C.PaStream
	players map[*playerImpl]struct{}
	buf     []float32
	m       sync.Mutex
}

func NewContext(sampleRate, channelNum, bitDepthInBytes int) (Context, chan struct{}, error) {
	ready := make(chan struct{})
	close(ready)

	c := &context{
		sampleRate:      sampleRate,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
	}

	if err := paErrorToError(C.Pa_Initialize(), "Pa_Initialize"); err != nil {
		return nil, nil, err
	}

	if err := paErrorToError(C.Pa_OpenDefaultStream(
		&c.stream,
		0,
		C.int(c.channelNum),
		C.paFloat32,
		C.double(c.sampleRate),
		C.paFramesPerBufferUnspecified,
		(*C.PaStreamCallback)(C.ebiten_readerdriver_streamCallback),
		unsafe.Pointer(c),
	), "Pa_OpenDefaultStream"); err != nil {
		return nil, nil, err
	}

	if err := paErrorToError(C.Pa_StartStream(c.stream), "Pa_StartStream"); err != nil {
		return nil, nil, err
	}

	return c, ready, nil
}

func (c *context) Suspend() error {
	if err := paErrorToError(C.Pa_StopStream(c.stream), "Pa_StopStream"); err != nil {
		return err
	}
	return nil
}

func (c *context) Resume() error {
	if err := paErrorToError(C.Pa_StartStream(c.stream), "Pa_StartStream"); err != nil {
		return err
	}
	return nil
}

func (c *context) addPlayer(player *playerImpl) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.players == nil {
		c.players = map[*playerImpl]struct{}{}
	}
	c.players[player] = struct{}{}
}

func (c *context) removePlayer(player *playerImpl) {
	c.m.Lock()
	defer c.m.Unlock()
	delete(c.players, player)
}

//export ebiten_readerdriver_streamCallback
func ebiten_readerdriver_streamCallback(input, output unsafe.Pointer, frameCount C.ulong, timeInfo *C.PaStreamCallbackTimeInfo, statusFlags C.PaStreamCallbackFlags, userData unsafe.Pointer) C.int {
	c := (*context)(userData)

	var n int
	if statusFlags&(C.paOutputUnderflow|C.paOutputOverflow) == 0 {
		n = int(frameCount) * c.channelNum
	}

	if len(c.buf) < n {
		c.buf = make([]float32, n)
	} else {
		for i := 0; i < n; i++ {
			c.buf[i] = 0
		}
	}

	if n > 0 {
		c.m.Lock()
		players := make([]*playerImpl, 0, len(c.players))
		for p := range c.players {
			players = append(players, p)
		}
		c.m.Unlock()

		for _, p := range players {
			p.addBuffer(c.buf[:n])
		}
		for i := uintptr(0); i < uintptr(n); i++ {
			*(*float32)(unsafe.Pointer(uintptr(output) + 4*i)) = c.buf[i]
		}
	}

	for i := uintptr(n); i < uintptr(frameCount); i++ {
		*(*float32)(unsafe.Pointer(uintptr(output) + 4*i)) = 0
	}
	return C.paContinue
}

type player struct {
	p *playerImpl
}

type playerImpl struct {
	context *context
	src     io.Reader
	cond    *sync.Cond
	volume  float64
	err     error
	state   playerState
	buf     []byte
	hasLoop bool
}

func (c *context) NewPlayer(src io.Reader) Player {
	p := &player{
		p: &playerImpl{
			context: c,
			src:     src,
			cond:    sync.NewCond(&sync.Mutex{}),
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
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	return p.err
}

func (p *player) Play() {
	p.p.Play()
}

func (p *playerImpl) Play() {
	ch := make(chan struct{})
	go func() {
		p.cond.L.Lock()
		defer p.cond.L.Unlock()
		close(ch)
		p.playImpl()
	}()
	<-ch
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

	p.cond.L.Unlock()
	p.context.addPlayer(p)
	p.cond.L.Lock()

	p.cond.Signal()

	if !p.hasLoop {
		go p.loop()
		p.hasLoop = true
	}
}

func (p *player) Pause() {
	p.p.Pause()
}

func (p *playerImpl) Pause() {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	p.pauseImpl()
}

func (p *playerImpl) pauseImpl() {
	if p.state != playerPlay {
		return
	}
	p.state = playerPaused
	p.cond.Signal()
}

func (p *player) Reset() {
	p.p.Reset()
}

func (p *playerImpl) Reset() {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	p.resetImpl()
}

func (p *playerImpl) resetImpl() {
	if p.state == playerClosed {
		return
	}
	p.state = playerPaused
	p.buf = p.buf[:0]
	p.cond.Signal()
}

func (p *player) IsPlaying() bool {
	return p.p.IsPlaying()
}

func (p *playerImpl) IsPlaying() bool {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.state == playerPlay
}

func (p *player) Volume() float64 {
	return p.p.Volume()
}

func (p *playerImpl) Volume() float64 {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.volume
}

func (p *player) SetVolume(volume float64) {
	p.p.SetVolume(volume)
}

func (p *playerImpl) SetVolume(volume float64) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	p.volume = volume
}

func (p *player) UnplayedBufferSize() int {
	return p.p.UnplayedBufferSize()
}

func (p *playerImpl) UnplayedBufferSize() int {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return len(p.buf)
}

func (p *player) Close() error {
	runtime.SetFinalizer(p, nil)
	return p.p.Close()
}

func (p *playerImpl) Close() error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.closeImpl()
}

func (p *playerImpl) closeImpl() error {
	p.cond.L.Unlock()
	p.context.removePlayer(p)
	p.cond.L.Lock()

	if p.state == playerClosed {
		return nil
	}
	p.state = playerClosed
	p.buf = nil
	p.cond.Signal()
	return p.err
}

func (p *playerImpl) addBuffer(buf []float32) int {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.state != playerPlay {
		return 0
	}

	bitDepthInBytes := p.context.bitDepthInBytes
	n := len(p.buf) / bitDepthInBytes
	if n > len(buf) {
		n = len(buf)
	}
	volume := float32(p.volume)
	for i := 0; i < n; i++ {
		var v float32
		switch bitDepthInBytes {
		case 1:
			v8 := p.buf[i]
			v = float32(v8-(1<<7)) / (1 << 7)
		case 2:
			v16 := int16(p.buf[2*i]) | (int16(p.buf[2*i+1]) << 8)
			v = float32(v16) / (1 << 15)
		}
		buf[i] += v * volume
	}
	p.buf = p.buf[n*bitDepthInBytes:]
	if n > 0 {
		p.cond.Signal()
	}
	return n
}

func (p *playerImpl) shouldWait() bool {
	switch p.state {
	case playerPaused:
		return true
	case playerPlay:
		// If the buffer has too much data, wait until the buffer data is consumed.
		// If the source reaches EOF, wait until the state is reset.
		return len(p.buf) >= p.context.maxBufferSize()
	case playerClosed:
		return false
	default:
		panic("not reached")
	}
}

func (p *playerImpl) wait() bool {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	for p.shouldWait() {
		p.cond.Wait()
	}
	return p.state == playerPlay
}

func (p *playerImpl) setError(err error) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	p.setErrorImpl(err)
}

func (p *playerImpl) setErrorImpl(err error) {
	p.err = err
	p.closeImpl()
}

func (p *playerImpl) loop() {
	buf := make([]byte, 4096)
	for {
		if !p.wait() {
			return
		}

		n, err := p.src.Read(buf)
		if err != nil && err != io.EOF {
			p.setError(err)
			return
		}

		p.cond.L.Lock()
		p.buf = append(p.buf, buf[:n]...)
		if err == io.EOF && len(p.buf) == 0 {
			p.resetImpl()
		}
		p.cond.L.Unlock()
	}
}
