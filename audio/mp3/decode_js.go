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
	"math"
	"time"

	"github.com/gopherjs/gopherwasm/js"
	"github.com/hajimehoshi/ebiten/audio"
)

// TODO: This just uses decodeAudioData, that can treat audio files other than MP3.

type Stream struct {
	leftData   []float32
	rightData  []float32
	posInBytes int
}

var (
	errTryAgain = errors.New("audio/mp3: try again")
	errTimeout  = errors.New("audio/mp3: timeout")
)

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
		b[4*i] = byte(il)
		b[4*i+1] = byte(il >> 8)
		b[4*i+2] = byte(ir)
		b[4*i+3] = byte(ir >> 8)
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
	case io.SeekStart:
		next = offset
	case io.SeekCurrent:
		next = int64(s.posInBytes) + offset
	case io.SeekEnd:
		next = s.Length() + offset
	}
	s.posInBytes = int(next)
	return next, nil
}

func (s *Stream) Close() error {
	return nil
}

func (s *Stream) Length() int64 {
	return int64(len(s.leftData) * 4)
}

// Size is deprecated as of 1.6.0-alpha. Use Length instead.
func (s *Stream) Size() int64 {
	return s.Length()
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
	try := 0
	for {
		s, err = decode(context, b, try)
		switch err {
		case errTryAgain:
			buf, ok := seekNextFrame(b)
			if !ok {
				return nil, fmt.Errorf("audio/mp3: Decode failed: invalid format?")
			}
			b = buf
			continue
		case errTimeout:
			try++
			continue
		default:
			if err != nil {
				return nil, err
			}
		}
		break
	}
	return s, nil
}

var offlineAudioContextClass = js.Null()

func init() {
	if klass := js.Global().Get("OfflineAudioContext"); klass != js.Undefined() {
		offlineAudioContextClass = klass
		return
	}
	if klass := js.Global().Get("webkitOfflineAudioContext"); klass != js.Undefined() {
		offlineAudioContextClass = klass
		return
	}
}

func float32ArrayToSlice(arr js.Value) []float32 {
	f := make([]float32, arr.Length())
	a := js.TypedArrayOf(f)
	a.Call("set", arr)
	a.Release()
	return f
}

func decode(context *audio.Context, buf []byte, try int) (*Stream, error) {
	if offlineAudioContextClass == js.Null() {
		return nil, errors.New("audio/mp3: OfflineAudioContext is not available")
	}

	s := &Stream{}
	ch := make(chan error)

	// TODO: webkitOfflineAudioContext's constructor might causes 'Syntax' error
	// when the sample rate is not so usual like 22050 on Safari.
	// Should this be handled before calling this?

	// TODO: 1 is a correct second argument?
	oc := offlineAudioContextClass.New(2, 1, context.SampleRate())

	u8 := js.TypedArrayOf(buf)
	a := u8.Get("buffer").Call("slice", u8.Get("byteOffset"), u8.Get("byteOffset").Int()+u8.Get("byteLength").Int())

	oc.Call("decodeAudioData", a, js.NewCallback(func(args []js.Value) {
		buf := args[0]
		s.leftData = float32ArrayToSlice(buf.Call("getChannelData", 0))
		switch n := buf.Get("numberOfChannels").Int(); n {
		case 1:
			s.rightData = s.leftData
			close(ch)
		case 2:
			s.rightData = float32ArrayToSlice(buf.Call("getChannelData", 1))
			close(ch)
		default:
			ch <- fmt.Errorf("audio/mp3: number of channels must be 1 or 2 but %d", n)
		}
	}), js.NewCallback(func(args []js.Value) {
		err := args[0]
		if err != js.Null() || err != js.Undefined() {
			ch <- fmt.Errorf("audio/mp3: decodeAudioData failed: %v", err)
		} else {
			// On Safari, error value might be null and it is needed to retry decoding
			// from the next frame (#438).
			ch <- errTryAgain
		}
	}))
	u8.Release()

	timeout := time.Duration(math.Pow(2, float64(try))) * time.Second

	select {
	case err := <-ch:
		if err != nil {
			return nil, err
		}
	case <-time.After(timeout):
		// Sometimes decode fails without calling the callbacks (#464).
		// Let's just try again in this case.
		return nil, errTimeout
	}
	return s, nil
}
