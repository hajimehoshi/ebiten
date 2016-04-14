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

package driver

// #cgo LDFLAGS: -lwinmm
//
// #include <windows.h>
// #include <mmsystem.h>
//
// #define sizeOfWavehdr (sizeof(WAVEHDR))
//
// MMRESULT waveOutOpen2(HWAVEOUT* waveOut, WAVEFORMATEX* format);
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

type header struct {
	buffer     unsafe.Pointer
	bufferSize int
	waveHdr    C.WAVEHDR
}

func newHeader(waveOut C.HWAVEOUT, bufferSize int) (*header, error) {
	// NOTE: This is never freed so far.
	buf := C.malloc(C.size_t(bufferSize))
	h := &header{
		buffer:     buf,
		bufferSize: bufferSize,
		waveHdr: C.WAVEHDR{
			lpData:         C.LPSTR(buf),
			dwBufferLength: C.DWORD(bufferSize),
		},
	}
	// TODO: Need to unprepare to avoid memory leak?
	if err := C.waveOutPrepareHeader(waveOut, &h.waveHdr, C.sizeOfWavehdr); err != C.MMSYSERR_NOERROR {
		return nil, fmt.Errorf("driver: waveOutPrepareHeader error: %d", err)
	}
	return h, nil
}

func (h *header) Write(waveOut C.HWAVEOUT, data []byte) error {
	if len(data) != h.bufferSize {
		return errors.New("driver: len(data) must equal to h.bufferSize")
	}
	C.memcpy(h.buffer, unsafe.Pointer(&data[0]), C.size_t(h.bufferSize))
	if err := C.waveOutWrite(waveOut, &h.waveHdr, C.sizeOfWavehdr); err != C.MMSYSERR_NOERROR {
		return fmt.Errorf("driver: waveOutWriter error: %d", err)
	}
	return nil
}

const numHeader = 8

var sem = make(chan struct{}, numHeader)

//export releaseSemaphore
func releaseSemaphore() {
	<-sem
}

type Player struct {
	out     C.HWAVEOUT
	buffer  []byte
	headers []*header
}

const bufferSize = 1024

func NewPlayer(sampleRate, channelNum, bytesPerSample int) (*Player, error) {
	numBlockAlign := channelNum * bytesPerSample
	f := C.WAVEFORMATEX{
		wFormatTag:      C.WAVE_FORMAT_PCM,
		nChannels:       C.WORD(channelNum),
		nSamplesPerSec:  C.DWORD(sampleRate),
		nAvgBytesPerSec: C.DWORD(sampleRate * numBlockAlign),
		wBitsPerSample:  C.WORD(bytesPerSample * 8),
		nBlockAlign:     C.WORD(numBlockAlign),
	}
	var w C.HWAVEOUT
	if err := C.waveOutOpen2(&w, &f); err != C.MMSYSERR_NOERROR {
		return nil, fmt.Errorf("driver: waveOutOpen error: %d", err)
	}
	p := &Player{
		out:     w,
		buffer:  []byte{},
		headers: make([]*header, numHeader),
	}
	for i := 0; i < numHeader; i++ {
		var err error
		p.headers[i], err = newHeader(w, bufferSize)
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *Player) Proceed(data []byte) error {
	p.buffer = append(p.buffer, data...)
	if bufferSize > len(p.buffer) {
		return nil
	}
	sem <- struct{}{}
	headerToWrite := (*header)(nil)
	for _, h := range p.headers {
		// TODO: Need to check WHDR_DONE?
		if h.waveHdr.dwFlags&C.WHDR_INQUEUE == 0 {
			headerToWrite = h
			break
		}
	}
	if headerToWrite == nil {
		return errors.New("driver: no available buffers")
	}
	if err := headerToWrite.Write(p.out, p.buffer[:bufferSize]); err != nil {
		return err
	}
	p.buffer = p.buffer[bufferSize:]
	return nil
}

func (p *Player) Close() {
	// TODO: Implement this
}
