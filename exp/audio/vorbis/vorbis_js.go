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

// +build js

package vorbis

import (
	"bytes"
	"io"
	"io/ioutil"
	"runtime"

	"github.com/gopherjs/gopherjs/js"
	"github.com/hajimehoshi/ebiten/exp/audio"
)

// TODO: This just uses decodeAudioData can treat audio files other than Ogg/Vorbis.
// TODO: This doesn't work on iOS which doesn't have Ogg/Vorbis decoder.

func Decode(context *audio.Context, src io.Reader) (*Stream, error) {
	b, err := ioutil.ReadAll(src)
	if err != nil {
		return nil, err
	}
	s := &Stream{
		sampleRate: context.SampleRate(),
	}
	ch := make(chan struct{})

	// TODO: 1 is a correct second argument?
	oc := js.Global.Get("OfflineAudioContext").New(2, 1, context.SampleRate())
	oc.Call("decodeAudioData", js.NewArrayBuffer(b), func(buf *js.Object) {
		go func() {
			defer close(ch)
			il := buf.Call("getChannelData", 0).Interface().([]float32)
			ir := buf.Call("getChannelData", 1).Interface().([]float32)
			b := make([]byte, len(il)*4)
			for i := 0; i < len(il); i++ {
				l := int16(il[i] * (1 << 15))
				r := int16(ir[i] * (1 << 15))
				b[4*i] = uint8(l)
				b[4*i+1] = uint8(l >> 8)
				b[4*i+2] = uint8(r)
				b[4*i+3] = uint8(r >> 8)
				if i%16384 == 0 {
					runtime.Gosched()
				}
			}
			s.buf = bytes.NewReader(b)
		}()
	})
	<-ch
	return s, nil
}
