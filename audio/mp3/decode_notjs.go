// Copyright 2017 The Ebiten Authors
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

// +build !js

package mp3

import (
	"errors"
	"io"
)

var (
	reader      io.Reader
	readerCache []uint8
	readerPos   int
	readerEOF   bool
	writer      io.Writer
)

func (f *frame) decodeL3() error {
	out := make([]uint32, 576)
	// Number of channels(1 for mono and 2 for stereo)
	nch := 2
	if f.header.mode == mpeg1ModeSingleChannel {
		nch = 1
	}
	for gr := 0; gr < 2; gr++ {
		for ch := 0; ch < nch; ch++ {
			f.l3Requantize(gr, ch)
			// Reorder short blocks
			f.l3Reorder(gr, ch)
		}
		f.l3Stereo(gr)
		for ch := 0; ch < nch; ch++ {
			f.l3Antialias(gr, ch)
			// (IMDCT,windowing,overlapp add)
			f.l3HybridSynthesis(gr, ch)
			f.l3FrequencyInversion(gr, ch)
			// Polyphase subband synthesis
			f.l3SubbandSynthesis(gr, ch, out)
		}
		if err := f.audioWriteRaw(out); err != nil {
			return err
		}
	}
	return nil
}

func (f *frame) audioWriteRaw(samples []uint32) error {
	nch := 2
	if f.header.mode == mpeg1ModeSingleChannel {
		nch = 1
	}
	s := make([]uint8, len(samples)*2*nch)
	for i, v := range samples {
		if nch == 1 {
			s[2*i] = uint8(v)
			s[2*i+1] = uint8(v >> 8)
		} else {
			s[4*i] = uint8(v)
			s[4*i+1] = uint8(v >> 8)
			s[4*i+2] = uint8(v >> 16)
			s[4*i+3] = uint8(v >> 24)
		}
	}
	if _, err := writer.Write(s); err != nil {
		return err
	}
	return nil
}

func getByte() (uint8, error) {
	for len(readerCache) == 0 && !readerEOF {
		buf := make([]uint8, 4096)
		n, err := reader.Read(buf)
		readerCache = append(readerCache, buf[:n]...)
		if err != nil {
			if err == io.EOF {
				readerEOF = true
			} else {
				return 0, err
			}
		}
	}
	if len(readerCache) == 0 {
		return 0, io.EOF
	}
	b := readerCache[0]
	readerCache = readerCache[1:]
	readerPos++
	return b, nil
}

func getBytes(buf []int) (int, error) {
	for i := range buf {
		v, err := getByte()
		buf[i] = int(v)
		if err == io.EOF {
			return i, io.EOF
		}
	}
	return len(buf), nil
}

func getFilepos() int {
	return readerPos
}

var eof = errors.New("mp3: expected EOF")

func decode(r io.Reader, w io.Writer) error {
	// TODO: Decoder should know number of channels
	reader = r
	writer = w
	for {
		f, err := readFrame()
		if err == nil {
			if err := f.decodeL3(); err != nil {
				return err
			}
			continue
		}
		if err == eof {
			break
		}
		return err
	}
	return nil
}
