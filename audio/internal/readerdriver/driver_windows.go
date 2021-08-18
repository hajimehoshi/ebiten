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
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Avoid goroutines on Windows (#1768).
// Apparently, switching contexts might take longer than other platforms.

const headerBufferSize = 4096

func IsAvailable() bool {
	return true
}

type header struct {
	waveOut uintptr
	buffer  []float32
	waveHdr *wavehdr
}

func newHeader(waveOut uintptr, bufferSizeInBytes int) (*header, error) {
	h := &header{
		waveOut: waveOut,
		buffer:  make([]float32, bufferSizeInBytes/4),
	}
	h.waveHdr = &wavehdr{
		lpData:         uintptr(unsafe.Pointer(&h.buffer[0])),
		dwBufferLength: uint32(bufferSizeInBytes),
	}
	if err := waveOutPrepareHeader(waveOut, h.waveHdr); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *header) Write(data []float32) error {
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

	waveOut uintptr
	headers []*header

	buf32 []float32

	players *players
}

var theContext *context

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

	const bitsPerSample = 32
	nBlockAlign := c.channelNum * bitsPerSample / 8
	f := &waveformatex{
		wFormatTag:      waveFormatIEEEFloat,
		nChannels:       uint16(c.channelNum),
		nSamplesPerSec:  uint32(c.sampleRate),
		nAvgBytesPerSec: uint32(c.sampleRate * nBlockAlign),
		wBitsPerSample:  bitsPerSample,
		nBlockAlign:     uint16(nBlockAlign),
	}

	// TOOD: What about using an event instead of a callback? PortAudio and other libraries do that.
	w, err := waveOutOpen(f, waveOutOpenCallback)
	const elementNotFound = 1168
	if e, ok := err.(*winmmError); ok && e.errno == elementNotFound {
		// TODO: No device was found. Return the dummy device (hajimehoshi/oto#77).
		// TODO: Retry to open the device when possible.
		return nil, nil, err
	}
	if err != nil {
		return nil, nil, err
	}

	c.waveOut = w
	c.headers = make([]*header, 0, 6)
	for len(c.headers) < cap(c.headers) {
		h, err := newHeader(c.waveOut, headerBufferSize)
		if err != nil {
			return nil, nil, err
		}
		c.headers = append(c.headers, h)
	}

	c.buf32 = make([]float32, headerBufferSize/4)
	for range c.headers {
		c.appendBuffers()
	}

	return c, ready, nil
}

func (c *context) Suspend() error {
	if err := waveOutPause(c.waveOut); err != nil {
		return err
	}
	return nil
}

func (c *context) Resume() error {
	// TODO: Ensure at least one header is queued?

	if err := waveOutRestart(c.waveOut); err != nil {
		return err
	}
	return nil
}

func (c *context) isHeaderAvailable() bool {
	for _, h := range c.headers {
		if !h.IsQueued() {
			return true
		}
	}
	return false
}

var waveOutOpenCallback = windows.NewCallbackCDecl(func(hwo, uMsg, dwInstance, dwParam1, dwParam2 uintptr) uintptr {
	const womDone = 0x3bd
	if uMsg != womDone {
		return 0
	}
	theContext.appendBuffers()
	return 0
})

func (c *context) appendBuffers() {
	for i := range c.buf32 {
		c.buf32[i] = 0
	}
	c.players.read(c.buf32)

	for _, h := range c.headers {
		if h.IsQueued() {
			continue
		}

		if err := h.Write(c.buf32); err != nil {
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
			// TODO: Treat the error corretly
			panic(fmt.Errorf("readerdriver: Queueing the header failed: %v", err))
		}
	}
}
