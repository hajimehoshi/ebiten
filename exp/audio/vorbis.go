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

// +build !js

package audio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/hajimehoshi/go-vorbis"
)

type VorbisStream struct {
	buf *bytes.Reader
}

// TODO: Rename to DecodeVorbis or Decode?

func (c *Context) NewVorbisStream(src io.Reader) (*VorbisStream, error) {
	decoded, channels, sampleRate, err := vorbis.Decode(src)
	if err != nil {
		return nil, err
	}
	if channels != 2 {
		return nil, errors.New("audio: number of channels must be 2")
	}
	if sampleRate != c.sampleRate {
		return nil, fmt.Errorf("audio: sample rate must be %d but %d", c.sampleRate, sampleRate)
	}
	// TODO: Read all data once so that Seek can be implemented easily.
	// We should look for a wiser way.
	b, err := ioutil.ReadAll(decoded)
	if err != nil {
		return nil, err
	}
	s := &VorbisStream{
		buf: bytes.NewReader(b),
	}
	return s, nil
}

func (s *VorbisStream) Read(p []byte) (int, error) {
	return s.buf.Read(p)
}

func (s *VorbisStream) Seek(offset int64, whence int) (int64, error) {
	return s.buf.Seek(offset, whence)
}
