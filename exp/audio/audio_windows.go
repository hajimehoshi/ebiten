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

package audio

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
	"fmt"
	"io"
	"runtime"
	"unsafe"
)

type header struct {
	buffer     unsafe.Pointer
	bufferSize int
	waveHdr    C.WAVEHDR
}

func newHeader(waveOut C.HWAVEOUT, bufferSize int) header {
	// NOTE: This is never freed so far.
	buf := C.malloc(C.size_t(bufferSize))
	h := header{
		buffer:     buf,
		bufferSize: bufferSize,
		waveHdr: C.WAVEHDR{
			lpData:         C.LPSTR(buf),
			dwBufferLength: C.DWORD(bufferSize),
		},
	}
	// TODO: Need to unprepare to avoid memory leak?
	if err := C.waveOutPrepareHeader(waveOut, &h.waveHdr, C.sizeOfWavehdr); err != C.MMSYSERR_NOERROR {
		panic(fmt.Sprintf("audio: waveOutPrepareHeader error %d", err))
	}
	return h
}

func (h *header) Write(waveOut C.HWAVEOUT, data []byte) {
	if len(data) != h.bufferSize {
		panic("audio: len(data) must equal to h.bufferSize")
	}
	C.memcpy(h.buffer, unsafe.Pointer(&data[0]), C.size_t(h.bufferSize))
	if err := C.waveOutWrite(waveOut, &h.waveHdr, C.sizeOfWavehdr); err != C.MMSYSERR_NOERROR {
		panic(fmt.Sprintf("audio: waveOutWriter error %d", err))
	}
}

const numHeader = 8

var sem = make(chan struct{}, numHeader)

//export releaseSemaphore
func releaseSemaphore() {
	<-sem
}

type player struct {
}

func startPlaying(src io.Reader, sampleRate int) (*player, error) {
	const numChannels = 2
	const bitsPerSample = 16
	const numBlockAlign = numChannels * bitsPerSample / 8
	f := C.WAVEFORMATEX{
		wFormatTag:      C.WAVE_FORMAT_PCM,
		nChannels:       numChannels,
		nSamplesPerSec:  C.DWORD(sampleRate),
		nAvgBytesPerSec: C.DWORD(sampleRate) * numBlockAlign,
		wBitsPerSample:  bitsPerSample,
		nBlockAlign:     numBlockAlign,
	}
	var w C.HWAVEOUT
	if err := C.waveOutOpen2(&w, &f); err != C.MMSYSERR_NOERROR {
		panic(fmt.Sprintf("audio: waveOutOpen error: %d", err))
	}
	go func() {
		const bufferSize = 1024
		b := []byte{}
		bb := make([]byte, bufferSize)
		headers := make([]header, numHeader)
		for i := 0; i < numHeader; i++ {
			headers[i] = newHeader(w, bufferSize)
		}
		i := 0
		for {
			n, err := src.Read(bb)
			if 0 < n {
				b = append(b, bb[:n]...)
				for bufferSize <= len(b) {
					sem <- struct{}{}
					headers[i].Write(w, b[:bufferSize])
					b = b[bufferSize:]
					i++
					i %= len(headers)
				}
			}
			if err == io.EOF {
				break
			}
			if err != nil {
				// TODO: Propagate this error?
				panic(err)
			}
			runtime.Gosched()
		}
		// TODO: Finalize the wave handler
	}()
	return &player{}, nil
}
