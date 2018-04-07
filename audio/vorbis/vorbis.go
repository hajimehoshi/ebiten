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

// Package vorbis provides Ogg/Vorbis decoder.
package vorbis

import (
	"fmt"
	"io"
	"runtime"

	"github.com/jfreymuth/oggvorbis"

	"github.com/hajimehoshi/ebiten/audio"
	"github.com/hajimehoshi/ebiten/audio/internal/convert"
)

// Stream is a decoded audio stream.
type Stream struct {
	decoded audio.ReadSeekCloser
	size    int64
}

// Read is implementation of io.Reader's Read.
func (s *Stream) Read(p []byte) (int, error) {
	return s.decoded.Read(p)
}

// Seek is implementation of io.Seeker's Seek.
//
// Note that Seek can take long since decoding is a relatively heavy task.
func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.decoded.Seek(offset, whence)
}

// Close is implementation of io.Closer's Close.
func (s *Stream) Close() error {
	runtime.SetFinalizer(s, nil)
	if err := s.decoded.Close(); err != nil {
		return err
	}
	return nil
}

// Length returns the size of decoded stream in bytes.
func (s *Stream) Length() int64 {
	return s.size
}

// Size is deprecated as of version 1.6.0-alpha. Use Length instead.
func (s *Stream) Size() int64 {
	return s.Length()
}

type decoded struct {
	data       []float32
	totalBytes int
	readBytes  int
	posInBytes int
	source     io.Closer
	decoder    *oggvorbis.Reader
}

func (d *decoded) readUntil(posInBytes int) error {
	buffer := make([]float32, 8192)
	for d.readBytes < posInBytes {
		n, err := d.decoder.Read(buffer)
		if n > 0 {
			// Actual read bytes might exceed the total bytes.
			if d.readBytes+n*2 > d.totalBytes {
				n = (d.totalBytes - d.readBytes) / 2
			}
			p := d.readBytes / 2
			for i := 0; i < n; i++ {
				d.data[p+i] = buffer[i]
			}
			d.readBytes += n * 2
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		runtime.Gosched()
	}
	return nil
}

func (d *decoded) Read(b []uint8) (int, error) {
	l := d.totalBytes - d.posInBytes
	if l > len(b) {
		l = len(b)
	}
	if l < 0 {
		return 0, io.EOF
	}
	// l must be even so that d.posInBytes is always even.
	l = l / 2 * 2
	if err := d.readUntil(d.posInBytes + l); err != nil {
		return 0, err
	}
	for i := 0; i < l/2; i++ {
		f := d.data[d.posInBytes/2+i]
		s := int16(f * (1<<15 - 1))
		b[2*i] = uint8(s)
		b[2*i+1] = uint8(s >> 8)
	}
	d.posInBytes += l
	if d.posInBytes == d.totalBytes {
		return l, io.EOF
	}
	return l, nil
}

func (d *decoded) Seek(offset int64, whence int) (int64, error) {
	next := int64(0)
	switch whence {
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = int64(d.posInBytes) + offset
	case io.SeekEnd:
		next = int64(d.totalBytes) + offset
	}
	// pos should be always even
	next = next / 2 * 2
	d.posInBytes = int(next)
	if err := d.readUntil(d.posInBytes); err != nil {
		return 0, err
	}
	return next, nil
}

func (d *decoded) Close() error {
	runtime.SetFinalizer(d, nil)
	if err := d.source.Close(); err != nil {
		return err
	}
	return nil
}

func (d *decoded) Length() int64 {
	return int64(d.totalBytes)
}

// decode accepts an ogg stream and returns a decorded stream.
func decode(in audio.ReadSeekCloser) (*decoded, int, int, error) {
	r, err := oggvorbis.NewReader(in)
	if err != nil {
		return nil, 0, 0, err
	}
	d := &decoded{
		data: make([]float32, int(r.Length())*r.Channels()),

		// TODO: r.Length() returns 0 when the format is unknown.
		// Should we check that?
		totalBytes: int(r.Length()) * r.Channels() * 2, // 2 means 16bit per sample.
		posInBytes: 0,
		source:     in,
		decoder:    r,
	}
	runtime.SetFinalizer(d, (*decoded).Close)
	if _, err := d.Read(make([]uint8, 65536)); err != nil && err != io.EOF {
		return nil, 0, 0, err
	}
	if _, err := d.Seek(0, io.SeekStart); err != nil {
		return nil, 0, 0, err
	}
	return d, r.Channels(), r.SampleRate(), nil
}

// Decode decodes Ogg/Vorbis data to playable stream.
//
// Decode returns error when decoding fails or IO error happens.
//
// Decode automatically resamples the stream to fit with the audio context if necessary.
func Decode(context *audio.Context, src audio.ReadSeekCloser) (*Stream, error) {
	decoded, channelNum, sampleRate, err := decode(src)
	if err != nil {
		return nil, err
	}
	if channelNum != 1 && channelNum != 2 {
		return nil, fmt.Errorf("vorbis: number of channels must be 1 or 2 but was %d", channelNum)
	}
	var s audio.ReadSeekCloser = decoded
	size := decoded.Length()
	if channelNum == 1 {
		s = convert.NewStereo16(s, true, false)
		size *= 2
	}
	if sampleRate != context.SampleRate() {
		r := convert.NewResampling(s, size, sampleRate, context.SampleRate())
		s = r
		size = r.Length()
	}
	stream := &Stream{decoded: s, size: size}
	runtime.SetFinalizer(stream, (*Stream).Close)
	return stream, nil
}
