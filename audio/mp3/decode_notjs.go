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
	reader      io.ReadCloser
	readerCache []uint8
	readerPos   int
	readerEOF   bool
}

func (s *source) Close() error {
	return s.reader.Close()
}

func (s *source) rewind() error {
	seeker := s.reader.(io.Seeker)
	if _, err := seeker.Seek(0, io.SeekStart); err != nil {
		return err
	}
	s.readerCache = nil
	s.readerPos = 0
	s.readerEOF = false
	return nil
}

func (s *source) getByte() (uint8, error) {
	for len(s.readerCache) == 0 && !s.readerEOF {
		buf := make([]uint8, 4096)
		n, err := s.reader.Read(buf)
		s.readerCache = append(s.readerCache, buf[:n]...)
		if err != nil {
			if err == io.EOF {
				s.readerEOF = true
				break
			}
			return 0, err
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

type Decoder struct {
	source     *source
	sampleRate int
	length     int64
	buf        []uint8
	frame      *frame
	eof        bool
}

func (d *Decoder) read() error {
	var err error
	d.frame, err = d.source.readNextFrame(d.frame)
	if err != nil {
		if err == io.EOF {
			d.eof = true
		}
		return err
	}
	d.buf = append(d.buf, d.frame.decodeL3()...)
	return nil
}

// Read is io.Reader's Read.
func (d *Decoder) Read(buf []uint8) (int, error) {
	for len(d.buf) == 0 && !d.eof {
		if err := d.read(); err != nil {
			return 0, err
		}
	}
	if d.eof {
		return 0, io.EOF
	}
	n := copy(buf, d.buf)
	d.buf = d.buf[n:]
	return n, nil
}

// Close is io.Closer's Close.
func (d *Decoder) Close() error {
	return d.source.Close()
}

// SampleRate returns the sample rate like 44100.
//
// Note that the sample rate is retrieved from the first frame.
func (d *Decoder) SampleRate() int {
	return d.sampleRate
}

// Length returns the total size in bytes.
//
// Length returns -1 when the total size is not available
// e.g. when the given source is not io.Seeker.
func (d *Decoder) Length() int64 {
	return d.length
}

func decode(r io.ReadCloser) (*Decoder, error) {
	s := &source{
		reader: r,
	}
	d := &Decoder{
		source: s,
		length: -1,
	}
	if _, ok := r.(io.Seeker); ok {
		l := int64(0)
		var f *frame
		for {
			var err error
			f, err = s.readNextFrame(f)
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			l += 576 * 4 * 2
		}
		if err := s.rewind(); err != nil {
			return nil, err
		}
		d.length = l
	}
	if err := d.read(); err != nil {
		return nil, err
	}
	d.sampleRate = samplingFrequency[d.frame.header.sampling_frequency]
	return d, nil
}
