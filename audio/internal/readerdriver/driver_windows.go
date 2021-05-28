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

import (
	"io"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const headerBufferSize = 2048

func IsAvailable() bool {
	return true
}

type header struct {
	waveOut uintptr
	buffer  []byte
	waveHdr *wavehdr
}

func newHeader(waveOut uintptr, bufferSize int) (*header, error) {
	h := &header{
		waveOut: waveOut,
		buffer:  make([]byte, bufferSize),
	}
	h.waveHdr = &wavehdr{
		lpData:         uintptr(unsafe.Pointer(&h.buffer[0])),
		dwBufferLength: uint32(bufferSize),
	}
	if err := waveOutPrepareHeader(waveOut, h.waveHdr); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *header) Write(data []byte) error {
	if n := len(h.buffer) - len(data); n > 0 {
		data = append(data, make([]byte, n)...)
	}
	copy(h.buffer, data)
	return waveOutWrite(h.waveOut, h.waveHdr)
}

func (h *header) IsQueued() bool {
	return h.waveHdr.dwFlags&whdrInqueue != 0
}

func (h *header) Close() error {
	return waveOutUnprepareHeader(h.waveOut, h.waveHdr)
}

type context struct {
	sampleRate      int
	channelNum      int
	bitDepthInBytes int
}

func NewContext(sampleRate, channelNum, bitDepthInBytes int) (Context, chan struct{}, error) {
	ready := make(chan struct{})
	close(ready)

	c := &context{
		sampleRate:      sampleRate,
		channelNum:      channelNum,
		bitDepthInBytes: bitDepthInBytes,
	}
	thePlayers.setContext(c)
	return c, ready, nil
}

func (c *context) Suspend() error {
	return thePlayers.suspend()
}

func (c *context) Resume() error {
	return thePlayers.resume()
}

type players struct {
	context *context
	players map[*playerImpl]struct{}
	buf     []byte
	err     error

	waveOut uintptr
	headers []*header

	cond *sync.Cond
}

func (p *players) setContext(context *context) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	p.context = context
}

func (p *players) add(player *playerImpl) error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.err != nil {
		return p.err
	}

	if p.players == nil {
		p.players = map[*playerImpl]struct{}{}
	}
	p.players[player] = struct{}{}
	p.cond.Signal()

	if p.waveOut != 0 {
		return nil
	}

	numBlockAlign := p.context.channelNum * p.context.bitDepthInBytes
	f := &waveformatex{
		wFormatTag:      waveFormatPCM,
		nChannels:       uint16(p.context.channelNum),
		nSamplesPerSec:  uint32(p.context.sampleRate),
		nAvgBytesPerSec: uint32(p.context.sampleRate * numBlockAlign),
		wBitsPerSample:  uint16(p.context.bitDepthInBytes * 8),
		nBlockAlign:     uint16(numBlockAlign),
	}

	w, err := waveOutOpen(f, waveOutOpenCallback)
	const elementNotFound = 1168
	if e, ok := err.(*winmmError); ok && e.errno == elementNotFound {
		// TODO: No device was found. Return the dummy device (hajimehoshi/oto#77).
		// TODO: Retry to open the device when possible.
		return err
	}
	if err != nil {
		return err
	}

	p.waveOut = w
	p.headers = make([]*header, 0, 4)
	for len(p.headers) < cap(p.headers) {
		h, err := newHeader(p.waveOut, headerBufferSize)
		if err != nil {
			return err
		}
		p.headers = append(p.headers, h)
	}

	if err := p.readAndWriteBuffers(); err != nil {
		return err
	}

	go p.loop()

	return nil
}

func (p *players) remove(player *playerImpl) error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()
	return p.removeImpl(player)
}

func (p *players) removeImpl(player *playerImpl) error {
	if p.err != nil {
		return p.err
	}
	delete(p.players, player)
	return nil
}

func (p *players) shouldWait() bool {
	if p.waveOut == 0 {
		return false
	}

	if len(p.players) == 0 {
		return true
	}

	if len(p.buf) < headerBufferSize*len(p.headers) {
		return false
	}

	for _, h := range p.headers {
		if !h.IsQueued() {
			return false
		}
	}

	return true
}

func (p *players) loop() {
	for {
		p.cond.L.Lock()
		for p.shouldWait() {
			p.cond.Wait()
		}
		if p.waveOut == 0 {
			p.cond.L.Unlock()
			return
		}
		if err := p.readAndWriteBuffers(); err != nil {
			p.err = err
			p.cond.L.Unlock()
			break
		}
		p.cond.L.Unlock()
	}
}

func (p *players) suspend() error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.waveOut == 0 {
		return nil
	}
	if err := waveOutPause(p.waveOut); err != nil {
		return err
	}
	return nil
}

func (p *players) resume() error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.waveOut == 0 {
		return nil
	}
	if err := waveOutRestart(p.waveOut); err != nil {
		return err
	}
	p.cond.Signal()
	return nil
}

var waveOutOpenCallback = windows.NewCallbackCDecl(func(hwo, uMsg, dwInstance, dwParam1, dwParam2 uintptr) uintptr {
	const womDone = 0x3bd
	if uMsg != womDone {
		return 0
	}
	thePlayers.cond.Signal()
	return 0
})

func (p *players) readAndWriteBuffers() error {
	if len(p.players) == 0 {
		return nil
	}

	headerNum := 0
	for _, h := range p.headers {
		if h.IsQueued() {
			continue
		}
		headerNum++
	}
	if headerNum == 0 {
		return nil
	}

	if n := headerBufferSize*headerNum - len(p.buf); n > 0 {
		// Do mixing of the current players instead of mixing on the OS side.
		// Apparently, mixing on the Go side is more effient and requires less buffers.
		//
		// waveOutSetVolume is not used since it doesn't work correctly in some environments.
		var volumes []float64
		var bufs [][]byte
		for pl := range p.players {
			buf := make([]byte, n)
			n := pl.read(buf)
			bufs = append(bufs, buf[:n])
			volumes = append(volumes, pl.Volume())
		}

		buf := make([]byte, n)
		switch p.context.bitDepthInBytes {
		case 1:
			const (
				max    = 127
				min    = -128
				offset = 128
			)
			for i := 0; i < n; i++ {
				var x int16
				for j, b := range bufs {
					if len(b) <= i {
						continue
					}
					xx := int16(b[i]) - offset
					x += int16(float64(xx) * volumes[j])
				}
				if x > max {
					x = max
				}
				if x < min {
					x = min
				}
				buf[i] = byte(x + offset)
			}
		case 2:
			const (
				max = (1 << 15) - 1
				min = -(1 << 15)
			)
			for i := 0; i < n/2; i++ {
				var x int32
				for j, b := range bufs {
					if len(b) <= 2*i {
						continue
					}
					xx := int32(int16(b[2*i]) | (int16(b[2*i+1]) << 8))
					x += int32(float64(xx) * volumes[j])
				}
				if x > max {
					x = max
				}
				if x < min {
					x = min
				}
				buf[2*i] = byte(x)
				buf[2*i+1] = byte(x >> 8)
			}
		}
		p.buf = append(p.buf, buf...)
	}

	for _, h := range p.headers {
		if len(p.buf) < headerBufferSize {
			break
		}

		if h.IsQueued() {
			continue
		}

		if err := h.Write(p.buf[:headerBufferSize]); err != nil {
			// This error can happen when e.g. a new HDMI connection is detected (hajimehoshi/oto#51).
			const errorNotFound = 1168
			if werr := err.(*winmmError); werr.fname == "waveOutWrite" && werr.errno == errorNotFound {
				// TODO: Retry later.
			}
			return err
		}
		p.buf = p.buf[headerBufferSize:]
	}

	return nil
}

var thePlayers = players{
	cond: sync.NewCond(&sync.Mutex{}),
}

type player struct {
	p *playerImpl
}

type playerImpl struct {
	context *context
	src     io.Reader
	err     error
	state   playerState
	buf     []byte
	eof     bool
	volume  float64

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
	// Call Play asynchronously since playImpl might take long.
	ch := make(chan struct{})
	go func() {
		p.m.Lock()
		defer p.m.Unlock()
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

	if p.eof && len(p.buf) == 0 {
		return
	}

	// Set the state before adding the player so that the audio loop can start to play it immediately.
	// This is a little tricky since this depends on the timing of Signal().
	p.state = playerPlay

	// thePlayers can has another mutex, and double mutex might introduce a deadlock.
	p.m.Unlock()
	err := thePlayers.add(p)
	p.m.Lock()

	if err != nil {
		p.setErrorImpl(err)
		return
	}

	// Do not create the player's own loop. Scheduling on Winodws is inefficient compared to the other OSes.
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
	if p.err != nil {
		return
	}
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
	if p.err != nil {
		return
	}
	if p.state == playerClosed {
		return
	}

	p.state = playerPaused
	p.buf = p.buf[:0]
	p.eof = false
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
	p.state = playerClosed

	p.m.Unlock()
	err := thePlayers.remove(p)
	p.m.Lock()

	if err != nil && p.err == nil {
		p.err = err
	}
	return p.err
}

func (p *playerImpl) setError(err error) {
	p.m.Lock()
	defer p.m.Unlock()
	p.setErrorImpl(err)
}

func (p *playerImpl) setErrorImpl(err error) {
	p.err = err
	p.closeImpl()
}

func (p *playerImpl) read(buf []byte) int {
	p.m.Lock()
	defer p.m.Unlock()

	if p.state != playerPlay {
		return 0
	}

	if len(p.buf) == 0 && p.eof {
		p.pauseImpl()
		return 0
	}

	if len(p.buf) < p.context.maxBufferSize() {
		buf := make([]byte, p.context.maxBufferSize())
		n, err := p.src.Read(buf)
		if err != nil && err != io.EOF {
			p.setErrorImpl(err)
			return len(buf)
		}

		p.buf = append(p.buf, buf[:n]...)
		if err == io.EOF {
			p.eof = true
		}
	}

	bytesPerSample := p.context.channelNum * p.context.bitDepthInBytes
	n := len(p.buf) / bytesPerSample * bytesPerSample
	n = copy(buf, p.buf[:n])
	p.buf = p.buf[n:]

	return n
}
