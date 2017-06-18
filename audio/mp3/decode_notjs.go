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

func (f *frame) decodeL3() []uint8 {
	out := make([]uint8, 576*4*2)
	nch := f.header.numberOfChannels()
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
			f.l3SubbandSynthesis(gr, ch, out[576*4*gr:])
		}
	}
	return out
}

type source struct {
	reader      io.Reader
	readerCache []uint8
	readerPos   int
	readerEOF   bool
}

func (s *source) getByte() (uint8, error) {
	for len(s.readerCache) == 0 && !s.readerEOF {
		buf := make([]uint8, 4096)
		n, err := s.reader.Read(buf)
		s.readerCache = append(s.readerCache, buf[:n]...)
		if err != nil {
			if err == io.EOF {
				s.readerEOF = true
			} else {
				return 0, err
			}
		}
	}
	if len(s.readerCache) == 0 {
		return 0, io.EOF
	}
	b := s.readerCache[0]
	s.readerCache = s.readerCache[1:]
	s.readerPos++
	return b, nil
}

func (s *source) getBytes(buf []int) (int, error) {
	for i := range buf {
		v, err := s.getByte()
		buf[i] = int(v)
		if err == io.EOF {
			return i, io.EOF
		}
	}
	return len(buf), nil
}

func (s *source) getFilepos() int {
	return s.readerPos
}

var eof = errors.New("mp3: expected EOF")

func decode(r io.Reader, w io.Writer) error {
	s := &source{
		reader: r,
	}
	var f *frame
	for {
		var err error
		f, err = s.readNextFrame(f)
		if err == nil {
			out := f.decodeL3()
			if _, err := w.Write(out); err != nil {
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
