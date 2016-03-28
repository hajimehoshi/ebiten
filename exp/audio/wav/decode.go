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
	"io/ioutil"
	"time"

	"github.com/hajimehoshi/ebiten/exp/audio"
)

const (
	headerSize = 44
	riffHeader = "RIFF"
	waveHeader = "WAVE"
)

type Stream struct {
	buf        *bytes.Reader
	sampleRate int
}

func (s *Stream) Read(p []byte) (int, error) {
	return s.buf.Read(p)
}

func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	return s.buf.Seek(offset, whence)
}

func (s *Stream) Close() error {
	s.buf = nil
	return nil
}

func (s *Stream) Len() time.Duration {
	const bytesPerSample = 4
	return time.Duration(s.buf.Len()/bytesPerSample) * time.Second / time.Duration(s.sampleRate)
}

func Decode(context *audio.Context, src io.Reader) (*Stream, error) {
	buf := make([]byte, headerSize)
	n, err := io.ReadFull(src, buf)
	if n != headerSize {
		return nil, fmt.Errorf("wav: invalid header")
	}
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(buf[0:4], []byte(riffHeader)) {
		return nil, fmt.Errorf("wav: invalid header: RIFF not found")
	}
	if !bytes.Equal(buf[8:12], []byte(waveHeader)) {
		return nil, fmt.Errorf("wav: invalid header: WAVE not found")
	}
	channels, depth := buf[22], buf[34]
	if channels != 2 {
		return nil, fmt.Errorf("wav: invalid header: channel num must be 2")
	}
	if depth != 16 {
		return nil, fmt.Errorf("wav: invalid header: depth must be 16")
	}
	sampleRate := int(buf[24]) | int(buf[25])<<8 | int(buf[26])<<16 | int(buf[27]<<24)
	if context.SampleRate() != sampleRate {
		return nil, fmt.Errorf("wav: sample rate must be %d but %d", context.SampleRate(), sampleRate)
	}
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	s := &Stream{
		buf:        bytes.NewReader(b),
		sampleRate: sampleRate,
	}
	return s, nil
}
