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
	"fmt"
	"io"
	"io/ioutil"

	"github.com/gopherjs/gopherjs/js"
	"github.com/hajimehoshi/ebiten/audio"
)

// TODO: This just uses decodeAudioData, that can treat audio files other than MP3.

type Stream struct {
	leftData   []float32
	rightData  []float32
	posInBytes int
}

var errTryAgain = errors.New("audio/mp3: try again")

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

// seekNextFrame seeks the next frame and returns the new buffer with the new position.
// seekNextFrame also returns true when seeking is successful, or false otherwise.
//
// Seeking is necessary when decoding fails. Safari's MP3 decoder can't treat IDs well (#438).
func seekNextFrame(buf []byte) ([]byte, bool) {
	// TODO: Need to skip tags explicitly? (hajimehoshi/go-mp3#9)

	if len(buf) < 1 {
		return nil, false
	}
	buf = buf[1:]

	for {
		if buf[0] == 0xff && buf[1]&0xfe == 0xfe {
			break
		}
		buf = buf[1:]
		if len(buf) < 2 {
			return nil, false
		}
	}
	return buf, true
}

func Decode(context *audio.Context, src audio.ReadSeekCloser) (*Stream, error) {
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	if err := src.Close(); err != nil {
		return nil, err
	}
	var s *Stream
	for {
		s, err = decode(context, b)
		if err == errTryAgain {
			buf, ok := seekNextFrame(b)
			if !ok {
				return nil, fmt.Errorf("audio/mp3: Decode failed: invalid format?")
			}
			b = buf
			continue
		}
		if err != nil {
			return nil, err
		}
		break
	}
	return s, nil
}

var offlineAudioContextClass *js.Object

func init() {
	if klass := js.Global.Get("OfflineAudioContext"); klass != js.Undefined {
		offlineAudioContextClass = klass
		return
	}
	if klass := js.Global.Get("webkitOfflineAudioContext"); klass != js.Undefined {
		offlineAudioContextClass = klass
		return
	}
}

func decode(context *audio.Context, buf []byte) (*Stream, error) {
	if offlineAudioContextClass == nil {
		return nil, errors.New("audio/mp3: OfflineAudioContext is not available")
	}

	s := &Stream{}
	ch := make(chan error)

	// TODO: webkitOfflineAudioContext's constructor might causes 'Syntax' error
	// when the sample rate is not so usual like 22050 on Safari.
	// Should this be handled before calling this?

	// TODO: 1 is a correct second argument?
	oc := offlineAudioContextClass.New(2, 1, context.SampleRate())
	oc.Call("decodeAudioData", js.NewArrayBuffer(buf), func(buf *js.Object) {
		s.leftData = buf.Call("getChannelData", 0).Interface().([]float32)
		switch n := buf.Get("numberOfChannels").Int(); n {
		case 1:
			s.rightData = s.leftData
			close(ch)
		case 2:
			s.rightData = buf.Call("getChannelData", 1).Interface().([]float32)
			close(ch)
		default:
			ch <- fmt.Errorf("audio/mp3: number of channels must be 1 or 2 but %d", n)
		}
	}, func(err *js.Object) {
		if err != nil {
			ch <- fmt.Errorf("audio/mp3: decodeAudioData failed: %v", err)
		} else {
			// On Safari, error value might be null and it is needed to retry decoding
			// from the next frame (#438).
			ch <- errTryAgain
		}
	})

	if err := <-ch; err != nil {
		return nil, err
	}
	return s, nil
}
