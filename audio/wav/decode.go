// Copyright 2016 Hajime Hoshi
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

package wav

import (
	"bytes"
	"fmt"
	"io"

	"github.com/hajimehoshi/ebiten/audio"
)

type Stream struct {
	src        audio.ReadSeekCloser
	headerSize int64
	dataSize   int64
}

func (s *Stream) Read(p []byte) (int, error) {
	return s.src.Read(p)
}

func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	if whence == 0 {
		offset += s.headerSize
	}
	return s.src.Seek(offset, whence)
}

func (s *Stream) Close() error {
	return s.src.Close()
}

func (s *Stream) Size() int64 {
	return s.dataSize
}

func Decode(context *audio.Context, src audio.ReadSeekCloser) (*Stream, error) {
	buf := make([]byte, 12)
	n, err := io.ReadFull(src, buf)
	if n != len(buf) {
		return nil, fmt.Errorf("wav: invalid header")
	}
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(buf[0:4], []byte("RIFF")) {
		return nil, fmt.Errorf("wav: invalid header: 'RIFF' not found")
	}
	if !bytes.Equal(buf[8:12], []byte("WAVE")) {
		return nil, fmt.Errorf("wav: invalid header: 'WAVE' not found")
	}

	// Read chunks
	dataSize := int64(0)
	headerSize := int64(0)
chunks:
	for {
		buf := make([]byte, 8)
		n, err := io.ReadFull(src, buf)
		if n != len(buf) {
			return nil, fmt.Errorf("wav: invalid header")
		}
		if err != nil {
			return nil, err
		}
		headerSize += 8
		size := int64(buf[4]) | int64(buf[5])<<8 | int64(buf[6])<<16 | int64(buf[7]<<24)
		switch {
		case bytes.Equal(buf[0:4], []byte("fmt ")):
			if size != 16 {
				return nil, fmt.Errorf("wav: invalid header: maybe non-PCM file?")
			}
			buf := make([]byte, size)
			n, err := io.ReadFull(src, buf)
			if n != len(buf) {
				return nil, fmt.Errorf("wav: invalid header")
			}
			if err != nil {
				return nil, err
			}
			format := int(buf[0]) | int(buf[1])<<8
			if format != 1 {
				return nil, fmt.Errorf("wav: format must be linear PCM")
			}
			channelNum := int(buf[2]) | int(buf[3])<<8
			// TODO: Remove this magic number
			if channelNum != 2 {
				return nil, fmt.Errorf("wav: channel num must be 2")
			}
			bitsPerSample := int(buf[14]) | int(buf[15])<<8
			// TODO: Remove this magic number
			if bitsPerSample != 16 {
				return nil, fmt.Errorf("wav: bits per sample must be 16")
			}
			sampleRate := int64(buf[4]) | int64(buf[5])<<8 | int64(buf[6])<<16 | int64(buf[7]<<24)
			if int64(context.SampleRate()) != sampleRate {
				return nil, fmt.Errorf("wav: sample rate must be %d but %d", context.SampleRate(), sampleRate)
			}
			headerSize += size
		case bytes.Equal(buf[0:4], []byte("data")):
			dataSize = size
			break chunks
		default:
			buf := make([]byte, size)
			n, err := io.ReadFull(src, buf)
			if n != len(buf) {
				return nil, fmt.Errorf("wav: invalid header")
			}
			if err != nil {
				return nil, err
			}
			headerSize += size
		}
	}
	s := &Stream{
		src:        src,
		headerSize: headerSize,
		dataSize:   dataSize,
	}
	return s, nil
}
