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
	"io"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

func IsAvailable() bool {
	return true
}

type audioQueuePoolItem struct {
	queue C.AudioQueueRef
	bufs  []C.AudioQueueBufferRef
}

const audioQueuePoolMaxItemNum = 32 // 32 is an arbitrary number.

type audioQueuePool struct {
	c      *context
	unused []audioQueuePoolItem
	used   []audioQueuePoolItem
	m      sync.Mutex
}

func (a *audioQueuePool) Prepare(context *context) error {
	a.c = context
	for i := 0; i < audioQueuePoolMaxItemNum; i++ {
		if _, _, err := a.Get(); err != nil {
			return err
		}
	}
	a.unused = a.used
	a.used = a.used[:0]
	return nil
}

func (a *audioQueuePool) Get() (C.AudioQueueRef, []C.AudioQueueBufferRef, error) {
	a.m.Lock()
	defer a.m.Unlock()

	if len(a.unused) > 0 {
		q := a.unused[0]
		a.unused = a.unused[1:]
		a.used = append(a.used, q)
		return q.queue, q.bufs, nil
	}

	flags := C.kAudioFormatFlagIsPacked
	if a.c.bitDepthInBytes != 1 {
		flags |= C.kAudioFormatFlagIsSignedInteger
	}
	desc := C.AudioStreamBasicDescription{
		mSampleRate:       C.double(a.c.sampleRate),
		mFormatID:         C.kAudioFormatLinearPCM,
		mFormatFlags:      C.UInt32(flags),
		mBytesPerPacket:   C.UInt32(a.c.channelNum * a.c.bitDepthInBytes),
		mFramesPerPacket:  1,
		mBytesPerFrame:    C.UInt32(a.c.channelNum * a.c.bitDepthInBytes),
		mChannelsPerFrame: C.UInt32(a.c.channelNum),
		mBitsPerChannel:   C.UInt32(8 * a.c.bitDepthInBytes),
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

	size := a.c.oneBufferSize()
	bufs := make([]C.AudioQueueBufferRef, 0, 2)
	for len(bufs) < cap(bufs) {
		var buf C.AudioQueueBufferRef
		if osstatus := C.AudioQueueAllocateBuffer(audioQueue, C.UInt32(size), &buf); osstatus != C.noErr {
			return nil, nil, fmt.Errorf("readerdriver: AudioQueueAllocateBuffer failed: %d", osstatus)
		}
		buf.mAudioDataByteSize = C.UInt32(size)
		bufs = append(bufs, buf)
	}

	a.used = append(a.used, audioQueuePoolItem{
		queue: audioQueue,
		bufs:  bufs,
	})

	return audioQueue, bufs, nil
}

func (a *audioQueuePool) Put(audioQueue C.AudioQueueRef) error {
	a.m.Lock()
	defer a.m.Unlock()

	for i, q := range a.used {
		if q.queue != audioQueue {
			continue
		}

		a.used = append(a.used[:i], a.used[i+1:]...)
		if len(a.unused)+len(a.used) < audioQueuePoolMaxItemNum {
			a.unused = append(a.unused, q)
			break
		}

		// As the pool is too big, remove the AudioQueue.
		for _, b := range q.bufs {
			if osstatus := C.AudioQueueFreeBuffer(q.queue, b); osstatus != C.noErr {
				return fmt.Errorf("readerdriver: AudioQueueFreeBuffer failed: %d", osstatus)
			}
		}
		if osstatus := C.AudioQueueDispose(q.queue, C.true); osstatus != C.noErr {
			return fmt.Errorf("readerdriver: AudioQueueDispose failed: %d", osstatus)
		}
		break
	}
	return nil
}

type context struct {
	sampleRate      int
	channelNum      int
	bitDepthInBytes int

	audioQueuePool audioQueuePool
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
	if err := c.audioQueuePool.Prepare(c); err != nil {
		return nil, nil, err
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
	players  map[C.AudioQueueRef]*playerImpl
	toResume map[*playerImpl]struct{}
	m        sync.Mutex
}

func (p *players) add(player *playerImpl, audioQueue C.AudioQueueRef) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.players == nil {
		p.players = map[C.AudioQueueRef]*playerImpl{}
	}
	p.players[audioQueue] = player
}

func (p *players) get(audioQueue C.AudioQueueRef) *playerImpl {
	p.m.Lock()
	defer p.m.Unlock()
	return p.players[audioQueue]
}

func (p *players) remove(audioQueue C.AudioQueueRef) {
	p.m.Lock()
	defer p.m.Unlock()

	pl, ok := p.players[audioQueue]
	if !ok {
		return
	}
	delete(p.players, audioQueue)
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
	// playerImpl's Play can touch p. Avoid the deadlock.
	p.m.Lock()
	players := map[*playerImpl]struct{}{}
	for pl := range p.toResume {
		players[pl] = struct{}{}
		delete(p.toResume, pl)
	}
	p.m.Unlock()

	for pl := range players {
		pl.Play()
		if err := pl.Err(); err != nil {
			return err
		}
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
	// Call Play asynchronously since AudioQueuePrime and AudioQueuePlay might take long.
	ch := make(chan struct{})
	go func() {
		p.cond.L.Lock()
		defer p.cond.L.Unlock()
		close(ch)
		p.playImpl()
	}()

	// Wait until the mutex is locked in the above goroutine.
	<-ch
}

func (p *playerImpl) playImpl() {
	if p.err != nil {
		return
	}
	if p.state != playerPaused {
		return
	}

	var runLoop bool
	if p.audioQueue == nil {
		audioQueue, audioQueueBuffers, err := p.context.audioQueuePool.Get()
		if err != nil {
			p.setErrorImpl(err)
			return
		}
		p.audioQueue = audioQueue
		p.unqueuedBufs = audioQueueBuffers
		C.AudioQueueSetParameter(p.audioQueue, C.kAudioQueueParam_Volume, C.AudioQueueParameterValue(p.volume))
		thePlayers.add(p, p.audioQueue)

		runLoop = true
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

	for {
		if osstatus := C.AudioQueuePrime(p.audioQueue, 0, nil); osstatus != C.noErr {
			// AudioQueuePrime might fail just after recovering from Siri.
			if osstatus == C.AVAudioSessionErrorCodeSiriIsRecording {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			p.setErrorImpl(fmt.Errorf("readerdriver: AudioQueuePrime failed: %d", osstatus))
			return
		}
		break
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
	q := p.audioQueue
	p.cond.L.Unlock()
	osstatus := C.AudioQueueReset(q)
	p.cond.L.Lock()
	if osstatus != C.noErr && p.err == nil {
		p.setErrorImpl(fmt.Errorf("readerdriver: AudioQueueReset failed: %d", osstatus))
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
	return p.closeImpl(false)
}

func (p *playerImpl) closeForReuse() error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.closeImpl(true)
}

func (p *playerImpl) closeImpl(reuseLater bool) error {
	if p.audioQueue != nil {
		// Even if reuseLater is true, AudioQueuePause is not efficient for reusing.
		// AudioQueueStart takes long if the AudioQueueStop is not called.

		// AudioQueueStop might invoke AudioQueueReset. Unlock the mutex here to avoid a deadlock.
		q := p.audioQueue
		p.cond.L.Unlock()
		osstatus := C.AudioQueueStop(q, C.true)
		p.cond.L.Lock()

		if osstatus != C.noErr && p.err != nil {
			// setErrorImpl calls closeImpl. Do not call this.
			p.err = fmt.Errorf("readerdriver: AudioQueueStop failed: %d", osstatus)
		}

		// All the AudioQueueBuffers are already dequeued. It is safe to dispose the AudioQueue and its buffers.
		if err := p.context.audioQueuePool.Put(p.audioQueue); err != nil && p.err != nil {
			p.err = err
		}
		thePlayers.remove(p.audioQueue)
		p.audioQueue = nil
	}

	// When reuseLater is true, this playerImpl can be reused later even though the AudioQueue is removed.
	if reuseLater {
		p.state = playerPaused
		p.buf = p.buf[:0]
		p.eof = false
		p.unqueuedBufs = nil
	} else {
		p.state = playerClosed
	}
	p.cond.Signal()
	return p.err
}

//export ebiten_readerdriver_render
func ebiten_readerdriver_render(inUserData unsafe.Pointer, inAQ C.AudioQueueRef, inBuffer C.AudioQueueBufferRef) {
	p := thePlayers.get(inAQ)
	queued, err := p.appendBuffer(inBuffer)
	if err != nil {
		p.setError(err)
		return
	}
	if !queued {
		p.unqueuedBufs = append(p.unqueuedBufs, inBuffer)
		if len(p.unqueuedBufs) == 2 && p.eof {
			p.closeForReuse()
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
	p.closeImpl(false)
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
