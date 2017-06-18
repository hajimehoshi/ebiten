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

// +build js

package mp3

import (
	"errors"
	"io"
	"io/ioutil"

	"github.com/gopherjs/gopherjs/js"
	"github.com/hajimehoshi/ebiten/audio"
)

// TODO: This just uses decodeAudioData can treat audio files other than MP3.

type Stream struct {
	leftData   []float32
	rightData  []float32
	posInBytes int
}

func (s *Stream) Read(b []byte) (int, error) {
	l := len(s.leftData)*4 - s.posInBytes
	if l > len(b) {
		l = len(b)
	}
	l = l / 4 * 4
	const max = 1<<15 - 1
	for i := 0; i < l/4; i++ {
		il := int32(s.leftData[s.posInBytes/4+i] * max)
		if il > max {
			il = max
		}
		if il < -max {
			il = -max
		}
		ir := int32(s.rightData[s.posInBytes/4+i] * max)
		if ir > max {
			ir = max
		}
		if ir < -max {
			ir = -max
		}
		b[4*i] = uint8(il)
		b[4*i+1] = uint8(il >> 8)
		b[4*i+2] = uint8(ir)
		b[4*i+3] = uint8(ir >> 8)
	}
	s.posInBytes += l
	if s.posInBytes == len(s.leftData)*4 {
		return l, io.EOF
	}
	return l, nil
}

func (s *Stream) Seek(offset int64, whence int) (int64, error) {
	next := int64(0)
	switch whence {
	case 0:
		next = offset
	case 1:
		next = int64(s.posInBytes) + offset
	case 2:
		next = s.Size() + offset
	}
	s.posInBytes = int(next)
	return next, nil
}

func (s *Stream) Close() error {
	return nil
}

func (s *Stream) Size() int64 {
	return int64(len(s.leftData) * 4)
}

func Decode(context *audio.Context, src audio.ReadSeekCloser) (*Stream, error) {
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	if err := src.Close(); err != nil {
		return nil, err
	}
	s := &Stream{}
	ch := make(chan struct{})

	klass := js.Global.Get("OfflineAudioContext")
	if klass == js.Undefined {
		klass = js.Global.Get("webkitOfflineAudioContext")
	}
	if klass == js.Undefined {
		return nil, errors.New("vorbis: OfflineAudioContext is not available")
	}
	// TODO: 1 is a correct second argument?
	oc := klass.New(2, 1, context.SampleRate())
	oc.Call("decodeAudioData", js.NewArrayBuffer(b), func(buf *js.Object) {
		s.leftData = buf.Call("getChannelData", 0).Interface().([]float32)
		s.rightData = buf.Call("getChannelData", 1).Interface().([]float32)
		close(ch)
	})
	<-ch
	return s, nil
}
