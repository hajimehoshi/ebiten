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

// The common players in players_unix.go are not used on Windows.
// Mixing on Go side can cause bigger delays (#1710).

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
	if err := waveOutWrite(h.waveOut, h.waveHdr); err != nil {
		return err
	}
	return nil
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
	return c, ready, nil
}

func (c *context) Suspend() error {
	return thePlayers.suspend()
}

func (c *context) Resume() error {
	return thePlayers.resume()
}

type players struct {
	players  map[uintptr]*playerImpl
	toResume map[*playerImpl]struct{}
	cond     *sync.Cond
}

func (p *players) add(player *playerImpl, waveOut uintptr) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	if p.players == nil {
		p.players = map[uintptr]*playerImpl{}
	}
	runLoop := len(p.players) == 0
	p.players[waveOut] = player
	if runLoop {
		// Use the only one loop. Windows' context switching is not efficent and
		// using too many goroutines might be problematic.
		go p.loop()
	}
}

func (p *players) remove(waveOut uintptr) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	pl, ok := p.players[waveOut]
	if !ok {
		return
	}
	delete(p.players, waveOut)
	delete(p.toResume, pl)

	p.cond.Signal()
}

func (p *players) suspend() error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	for _, pl := range p.players {
		if !pl.IsPlaying() {
			continue
		}
		pl.Pause()
		if p.toResume == nil {
			p.toResume = map[*playerImpl]struct{}{}
		}
		p.toResume[pl] = struct{}{}
	}
	return nil
}

func (p *players) resume() error {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	for pl := range p.toResume {
		pl.Play()
		delete(p.toResume, pl)
	}
	return nil
}

func (p *players) shouldWait() bool {
	if len(p.players) == 0 {
		return false
	}

	for _, pl := range p.players {
		if pl.canProceed() {
			return false
		}
	}
	return true
}

func (p *players) wait() bool {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	for p.shouldWait() {
		p.cond.Wait()
	}
	return len(p.players) > 0
}

func (p *players) loop() {
	for {
		if !p.wait() {
			return
		}
		p.cond.L.Lock()
		for _, pl := range p.players {
			pl.readAndWriteBuffer()
		}
		p.cond.L.Unlock()
	}
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
	waveOut uintptr
	state   playerState
	headers []*header
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

	if p.waveOut == 0 {
		numBlockAlign := p.context.channelNum * p.context.bitDepthInBytes
		f := &waveformatex{
			wFormatTag:      waveFormatPCM,
			nChannels:       uint16(p.context.channelNum),
			nSamplesPerSec:  uint32(p.context.sampleRate),
			nAvgBytesPerSec: uint32(p.context.sampleRate * numBlockAlign),
			wBitsPerSample:  uint16(p.context.bitDepthInBytes * 8),
			nBlockAlign:     uint16(numBlockAlign),
		}

		// TOOD: What about using an event instead of a callback? PortAudio and other libraries do that.
		w, err := waveOutOpen(f, waveOutOpenCallback)
		const elementNotFound = 1168
		if e, ok := err.(*winmmError); ok && e.errno == elementNotFound {
			// TODO: No device was found. Return the dummy device (hajimehoshi/oto#77).
			// TODO: Retry to open the device when possible.
			p.setErrorImpl(err)
			return
		}
		if err != nil {
			p.setErrorImpl(err)
			return
		}

		p.waveOut = w
		p.headers = make([]*header, 0, 6)
		for len(p.headers) < cap(p.headers) {
			h, err := newHeader(p.waveOut, headerBufferSize)
			if err != nil {
				p.setErrorImpl(err)
				return
			}
			p.headers = append(p.headers, h)
		}

		thePlayers.add(p, p.waveOut)
	}

	if p.eof && len(p.buf) == 0 {
		return
	}

	// Set the state first as readAndWriteBufferImpl checks the current player state.
	p.state = playerPlay

	// Call readAndWriteBufferImpl to ensure at least one header is queued.
	p.readAndWriteBufferImpl()

	if err := waveOutRestart(p.waveOut); err != nil {
		p.setErrorImpl(err)
		return
	}

	// Switching goroutines is very inefficient on Windows. Avoid a dedicated goroutine for a player.
}

func (p *playerImpl) queuedHeadersNum() int {
	var c int
	for _, h := range p.headers {
		if h.IsQueued() {
			c++
		}
	}
	return c
}

func (p *playerImpl) canProceed() bool {
	p.m.Lock()
	defer p.m.Unlock()
	return p.queuedHeadersNum() < len(p.headers) && p.state == playerPlay
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
	if p.waveOut == 0 {
		return
	}

	// waveOutPause never return when there is no queued header.
	if p.queuedHeadersNum() > 0 {
		if err := waveOutPause(p.waveOut); err != nil {
			p.setErrorImpl(err)
			return
		}
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
	if p.waveOut == 0 {
		return
	}

	// waveOutReset and waveOutPause never return when there is no queued header.
	if p.queuedHeadersNum() > 0 {
		err := waveOutReset(p.waveOut)
		if err != nil {
			p.setErrorImpl(err)
			return
		}

		err = waveOutPause(p.waveOut)
		if err != nil {
			p.setErrorImpl(err)
			return
		}
	}

	// Now all the headers are WHDR_DONE. Recreate the headers.
	for i, h := range p.headers {
		if err := h.Close(); err != nil {
			p.setErrorImpl(err)
			return
		}
		h, err := newHeader(p.waveOut, headerBufferSize)
		if err != nil {
			p.setErrorImpl(err)
			return
		}
		p.headers[i] = h
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
	if p.waveOut != 0 {
		for _, h := range p.headers {
			if err := h.Close(); err != nil && p.err == nil {
				p.err = err
			}
		}
		p.headers = p.headers[:0]
		if err := waveOutClose(p.waveOut); err != nil && p.err == nil {
			p.err = err
		}

		// This player's lock might block thePlayer's lock. Unlock this first.
		p.m.Unlock()
		thePlayers.remove(p.waveOut)
		p.m.Lock()

		p.waveOut = 0
	}
	return p.err
}

var waveOutOpenCallback = windows.NewCallbackCDecl(func(hwo, uMsg, dwInstance, dwParam1, dwParam2 uintptr) uintptr {
	const womDone = 0x3bd
	if uMsg != womDone {
		return 0
	}
	thePlayers.cond.Signal()
	return 0
})

func (p *playerImpl) readAndWriteBuffer() {
	p.m.Lock()
	defer p.m.Unlock()
	p.readAndWriteBufferImpl()
}

func (p *playerImpl) readAndWriteBufferImpl() {
	if p.state != playerPlay {
		return
	}

	for len(p.buf) < p.context.maxBufferSize() && !p.eof {
		buf := make([]byte, p.context.maxBufferSize())
		n, err := p.src.Read(buf)
		if err != nil && err != io.EOF {
			p.setErrorImpl(err)
			return
		}
		p.buf = append(p.buf, buf[:n]...)
		if err == io.EOF {
			if len(p.buf) == 0 {
				p.eof = true
			}
			break
		}
	}

	for _, h := range p.headers {
		if len(p.buf) == 0 {
			break
		}
		if h.IsQueued() {
			continue
		}

		n := headerBufferSize
		if n > len(p.buf) {
			n = len(p.buf)
		}
		buf := p.buf[:n]

		// Adjust the volume
		if p.volume < 1 {
			switch p.context.bitDepthInBytes {
			case 1:
				const (
					max    = 127
					min    = -128
					offset = 128
				)
				for i, b := range buf {
					x := int16(b) - offset
					x = int16(float64(x) * p.volume)
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
					x := int32(int16(buf[2*i]) | (int16(buf[2*i+1]) << 8))
					x = int32(float64(x) * p.volume)
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
		}

		if err := h.Write(buf); err != nil {
			// This error can happen when e.g. a new HDMI connection is detected (hajimehoshi/oto#51).
			const errorNotFound = 1168
			if werr := err.(*winmmError); werr.fname == "waveOutWrite" {
				switch {
				case werr.mmresult == mmsyserrNomem:
					continue
				case werr.errno == errorNotFound:
					// TODO: Retry later.
				}
			}
			p.setErrorImpl(err)
			return
		}

		p.buf = p.buf[n:]

		// 4 is an arbitrary number that doesn't cause a problem at examples/piano (#1653).
		if p.queuedHeadersNum() >= 4 {
			break
		}
	}

	if p.queuedHeadersNum() == 0 && p.eof {
		p.pauseImpl()
	}
}

func (p *playerImpl) setErrorImpl(err error) {
	p.err = err
	p.closeImpl()
}
