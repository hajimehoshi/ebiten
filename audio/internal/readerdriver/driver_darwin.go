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

// +build !ios

package readerdriver

// #cgo LDFLAGS: -framework AudioToolbox
//
// #import <AudioToolbox/AudioToolbox.h>
//
// void ebiten_readerdriver_render(void* inUserData, AudioQueueRef inAQ, AudioQueueBufferRef inBuffer);
//
// void ebiten_readerdriver_setNotificationHandler();
//
// // ebiten_readerdriver_AudioQueueNewOutput is a wrapper for AudioQueueNewOutput.
// // This is to avoid go-vet warnings of an unsafe.Pointer usage.
// // TODO: Use cgo.Handle (https://tip.golang.org/pkg/runtime/cgo/#Handle) when Go 1.17 becomes the minimum supported version.
// static OSStatus ebiten_readerdriver_AudioQueueNewOutput(
//     const AudioStreamBasicDescription *inFormat,
//     AudioQueueOutputCallback inCallbackProc,
//     uintptr_t inUserData,
//     CFRunLoopRef inCallbackRunLoop,
//     CFStringRef inCallbackRunLoopMode,
//     UInt32 inFlags,
//     AudioQueueRef *outAQ) {
//   return AudioQueueNewOutput(inFormat, inCallbackProc, (void*)(inUserData), inCallbackRunLoop, inCallbackRunLoopMode, inFlags, outAQ);
// }
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
}

// TOOD: Convert the error code correctly.
// See https://stackoverflow.com/questions/2196869/how-do-you-convert-an-iphone-osstatus-code-to-something-useful

func NewContext(sampleRate, channelNum, bitDepthInBytes int) (Context, chan struct{}, error) {
	ready := make(chan struct{})
	close(ready)

	c := &context{
		sampleRate:      sampleRate,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
	}
	C.ebiten_readerdriver_setNotificationHandler()
	return c, ready, nil
}

func (c *context) Suspend() error {
	return thePlayers.suspend()
}

func (c *context) Resume() error {
	return thePlayers.resume()
}

type player struct {
	p *playerImpl
}

type playerImpl struct {
	context      *context
	src          io.Reader
	id           int
	audioQueue   C.AudioQueueRef
	buf          []byte
	unqueuedBufs []C.AudioQueueBufferRef
	state        playerState
	err          error
	eof          bool
	cond         *sync.Cond
	volume       float64
}

type players struct {
	players  map[int]*playerImpl
	toResume map[*playerImpl]struct{}
	m        sync.Mutex
}

func (p *players) add(player *playerImpl) int {
	p.m.Lock()
	defer p.m.Unlock()

	id := 1
	for {
		if _, ok := p.players[id]; ok {
			id++
			continue
		}
		break
	}
	if p.players == nil {
		p.players = map[int]*playerImpl{}
	}
	p.players[id] = player
	return id
}

func (p *players) get(id int) *playerImpl {
	p.m.Lock()
	defer p.m.Unlock()
	return p.players[id]
}

func (p *players) remove(id int) {
	p.m.Lock()
	defer p.m.Unlock()

	pl, ok := p.players[id]
	if !ok {
		return
	}
	delete(p.players, id)
	delete(p.toResume, pl)
}

func (p *players) suspend() error {
	p.m.Lock()
	defer p.m.Unlock()

	for _, pl := range p.players {
		if !pl.IsPlaying() {
			continue
		}
		// TODO: Is this OK to Pause instead of Close?
		// Oboe (Android) closes players when suspending to avoid hogging audio resources which other apps could use.
		pl.Pause()
		if err := pl.Err(); err != nil {
			return err
		}
		if p.toResume == nil {
			p.toResume = map[*playerImpl]struct{}{}
		}
		p.toResume[pl] = struct{}{}
	}
	return nil
}

func (p *players) resume() error {
	p.m.Lock()
	defer p.m.Unlock()

	for pl := range p.toResume {
		pl.Play()
		if err := pl.Err(); err != nil {
			return err
		}
		delete(p.toResume, pl)
	}
	return nil
}

var thePlayers players

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
	p.p.id = thePlayers.add(p.p)
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
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.err != nil {
		return
	}
	if p.state != playerPaused {
		return
	}

	var runLoop bool
	if p.audioQueue == nil {
		c := p.context
		flags := C.kAudioFormatFlagIsPacked
		if c.bitDepthInBytes != 1 {
			flags |= C.kAudioFormatFlagIsSignedInteger
		}
		desc := C.AudioStreamBasicDescription{
			mSampleRate:       C.double(c.sampleRate),
			mFormatID:         C.kAudioFormatLinearPCM,
			mFormatFlags:      C.UInt32(flags),
			mBytesPerPacket:   C.UInt32(c.channelNum * c.bitDepthInBytes),
			mFramesPerPacket:  1,
			mBytesPerFrame:    C.UInt32(c.channelNum * c.bitDepthInBytes),
			mChannelsPerFrame: C.UInt32(c.channelNum),
			mBitsPerChannel:   C.UInt32(8 * c.bitDepthInBytes),
		}

		var audioQueue C.AudioQueueRef
		if osstatus := C.ebiten_readerdriver_AudioQueueNewOutput(
			&desc,
			(C.AudioQueueOutputCallback)(C.ebiten_readerdriver_render),
			C.uintptr_t(p.id),
			(C.CFRunLoopRef)(0),
			(C.CFStringRef)(0),
			0,
			&audioQueue); osstatus != C.noErr {
			p.setErrorImpl(fmt.Errorf("readerdriver: AudioQueueNewFormat with StreamFormat failed: %d", osstatus))
			return
		}
		p.audioQueue = audioQueue

		size := c.oneBufferSize()
		for len(p.unqueuedBufs) < 2 {
			var buf C.AudioQueueBufferRef
			if osstatus := C.AudioQueueAllocateBuffer(audioQueue, C.UInt32(size), &buf); osstatus != C.noErr {
				p.setErrorImpl(fmt.Errorf("readerdriver: AudioQueueAllocateBuffer failed: %d", osstatus))
				return
			}
			buf.mAudioDataByteSize = C.UInt32(size)
			p.unqueuedBufs = append(p.unqueuedBufs, buf)
		}

		C.AudioQueueSetParameter(p.audioQueue, C.kAudioQueueParam_Volume, C.AudioQueueParameterValue(p.volume))

		runLoop = true
	}

	for len(p.buf) < p.context.maxBufferSize() {
		buf := make([]byte, p.context.maxBufferSize())
		n, err := p.src.Read(buf)
		if err != nil && err != io.EOF {
			p.setErrorImpl(err)
			return
		}
		p.buf = append(p.buf, buf[:n]...)
		if err == io.EOF {
			p.eof = true
			break
		}
	}

	bufs := make([]C.AudioQueueBufferRef, len(p.unqueuedBufs))
	copy(bufs, p.unqueuedBufs)
	var unenqueued []C.AudioQueueBufferRef
	for _, buf := range bufs {
		queued, err := p.appendBufferImpl(buf)
		if err != nil {
			p.setErrorImpl(err)
			return
		}
		if !queued {
			unenqueued = append(unenqueued, buf)
		}
	}
	p.unqueuedBufs = unenqueued
	if len(p.unqueuedBufs) == 2 && p.eof {
		p.state = playerPaused
		return
	}

	if osstatus := C.AudioQueueStart(p.audioQueue, nil); osstatus != C.noErr {
		p.setErrorImpl(fmt.Errorf("readerdriver: AudioQueueStart failed: %d", osstatus))
		return
	}
	p.state = playerPlay
	p.cond.Signal()

	if runLoop {
		go p.loop()
	}
}

func (p *player) Pause() {
	p.p.Pause()
}

func (p *playerImpl) Pause() {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.err != nil {
		return
	}
	if p.state != playerPlay {
		return
	}
	if p.audioQueue == nil {
		return
	}

	if osstatus := C.AudioQueuePause(p.audioQueue); osstatus != C.noErr && p.err == nil {
		p.setErrorImpl(fmt.Errorf("readerdriver: AudioQueuePause failed: %d", osstatus))
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

	if p.err != nil {
		return
	}
	if p.state == playerClosed {
		return
	}
	if p.audioQueue == nil {
		return
	}

	if osstatus := C.AudioQueuePause(p.audioQueue); osstatus != C.noErr && p.err == nil {
		p.setErrorImpl(fmt.Errorf("readerdriver: AudioQueuePause failed: %d", osstatus))
		return
	}
	// AudioQueueReset invokes the callback directry.
	p.cond.L.Unlock()
	if osstatus := C.AudioQueueReset(p.audioQueue); osstatus != C.noErr && p.err == nil {
		p.setErrorImpl(fmt.Errorf("readerdriver: AudioQueueReset failed: %d", osstatus))
		p.cond.L.Lock()
		return
	}
	p.cond.L.Lock()
	if osstatus := C.AudioQueueFlush(p.audioQueue); osstatus != C.noErr && p.err == nil {
		p.setErrorImpl(fmt.Errorf("readerdriver: AudioQueueFlush failed: %d", osstatus))
		return
	}

	p.state = playerPaused
	p.buf = p.buf[:0]
	p.eof = false
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
	if p.audioQueue == nil {
		return
	}
	C.AudioQueueSetParameter(p.audioQueue, C.kAudioQueueParam_Volume, C.AudioQueueParameterValue(volume))
}

func (p *player) UnplayedBufferSize() int64 {
	return p.p.UnplayedBufferSize()
}

func (p *playerImpl) UnplayedBufferSize() int64 {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return int64(len(p.buf))
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
	if p.audioQueue != nil {
		if osstatus := C.AudioQueueStop(p.audioQueue, C.true); osstatus != C.noErr && p.err != nil {
			// setErrorImpl calls closeImpl. Do not call this.
			p.err = fmt.Errorf("readerdriver: AudioQueueStop failed: %d", osstatus)
		}
		for _, b := range p.unqueuedBufs {
			if osstatus := C.AudioQueueFreeBuffer(p.audioQueue, b); osstatus != C.noErr && p.err != nil {
				p.err = fmt.Errorf("readerdriver: AudioQueueFreeBuffer failed: %d", osstatus)
			}
		}
		p.unqueuedBufs = nil
		if osstatus := C.AudioQueueDispose(p.audioQueue, C.true); osstatus != C.noErr && p.err != nil {
			p.err = fmt.Errorf("readerdriver: AudioQueueDispose failed: %d", osstatus)
		}
		p.audioQueue = nil
	}
	p.state = playerClosed
	p.cond.Signal()
	thePlayers.remove(p.id)
	return p.err
}

//export ebiten_readerdriver_render
func ebiten_readerdriver_render(inUserData unsafe.Pointer, inAQ C.AudioQueueRef, inBuffer C.AudioQueueBufferRef) {
	id := int(uintptr(inUserData))
	p := thePlayers.get(id)
	queued, err := p.appendBuffer(inBuffer)
	if err != nil {
		p.setError(err)
		return
	}
	if !queued {
		p.unqueuedBufs = append(p.unqueuedBufs, inBuffer)
		if len(p.unqueuedBufs) == 2 && p.eof {
			p.Pause()
		}
	}
}

func (p *playerImpl) appendBuffer(inBuffer C.AudioQueueBufferRef) (bool, error) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.appendBufferImpl(inBuffer)
}

func (p *playerImpl) appendBufferImpl(inBuffer C.AudioQueueBufferRef) (bool, error) {
	if p.eof && len(p.buf) == 0 {
		return false, nil
	}

	bs := make([]byte, p.context.oneBufferSize())
	n := copy(bs, p.buf)
	var signal bool
	if len(p.buf[n:]) < p.context.maxBufferSize() {
		signal = true
	}

	for i, b := range bs {
		*(*byte)(unsafe.Pointer(uintptr(inBuffer.mAudioData) + uintptr(i))) = b
	}

	if osstatus := C.AudioQueueEnqueueBuffer(p.audioQueue, inBuffer, 0, nil); osstatus != C.noErr {
		// This can happen just after resetting.
		if osstatus == C.kAudioQueueErr_EnqueueDuringReset {
			return false, nil
		}
		return false, fmt.Errorf("readerdriver: AudioQueueEnqueueBuffer failed: %d", osstatus)
	}

	p.buf = p.buf[n:]
	if signal {
		p.cond.Signal()
	}
	return true, nil
}

func (p *playerImpl) shouldWait() bool {
	switch p.state {
	case playerPaused:
		return true
	case playerPlay:
		return len(p.buf) >= p.context.maxBufferSize() || p.eof
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
		l := len(p.buf)
		p.cond.L.Unlock()

		if err == io.EOF && l == 0 {
			p.cond.L.Lock()
			p.eof = true
			p.cond.L.Unlock()
		}
	}
}

//export ebiten_readerdriver_setGlobalPause
func ebiten_readerdriver_setGlobalPause() {
	thePlayers.suspend()
}

//export ebiten_readerdriver_setGlobalResume
func ebiten_readerdriver_setGlobalResume() {
	thePlayers.resume()
}
