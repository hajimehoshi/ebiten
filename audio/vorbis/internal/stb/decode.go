// Copyright 2018 The Ebiten Authors
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

// +build js,!wasm

package stb

import (
	"fmt"
	"io"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/internal/jsutil"
)

var flatten = js.Global().Get("window").Call("eval", `(function(arr) {
  var ch = arr.length;
  var len = arr[0].length;
  var result = new Float32Array(ch * len);
  for (var j = 0; j < len; j++) {
    for (var i = 0; i < ch; i++) {
      result[j*ch+i] = arr[i][j];
    }
  }
  return result;
})`)

func init() {
	// Eval wasm.js first to set the Wasm binary to Module.
	js.Global().Get("window").Call("eval", string(stbvorbis_js))
}

type Samples struct {
	samples         [][]float32
	channels        int
	lengthInSamples int64
	posInSamples    int64
}

func (s *Samples) Read(buf []float32) (int, error) {
	if s.posInSamples == s.lengthInSamples {
		return 0, io.EOF
	}
	if len(buf) == 0 {
		return 0, nil
	}

	var p int64
	idx := 0
	for idx < len(s.samples) {
		l := int64(len(s.samples[idx])) / int64(s.channels)
		if p+l > s.posInSamples {
			break
		}
		p += l
		idx++
	}
	start := (s.posInSamples - p) * int64(s.channels)
	if start == int64(len(s.samples[idx])) {
		idx++
		start = 0
	}
	if len(s.samples[idx]) == 0 {
		panic(fmt.Sprintf("stb: len(samples[%d]) must be > 0", idx))
	}
	n := copy(buf, s.samples[idx][start:])
	s.posInSamples += int64(n) / int64(s.channels)
	return n, nil
}

func (s *Samples) Length() int64 {
	return s.lengthInSamples
}

func (s *Samples) SetPosition(pos int64) error {
	s.posInSamples = pos
	return nil
}

func DecodeVorbis(buf []byte) (*Samples, int, int, error) {
	ch := make(chan error)
	samples := &Samples{}
	sampleRate := 0

	f := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		r := args[0]

		if e := r.Get("error"); e != js.Null() {
			ch <- fmt.Errorf("audio/vorbis/internal/stb: decode error: %s", e.String())
			close(ch)
			return nil
		}

		if r.Get("eof").Bool() {
			close(ch)
			return nil
		}

		if samples.channels == 0 {
			samples.channels = r.Get("data").Length()

		}
		if sampleRate == 0 {
			sampleRate = r.Get("sampleRate").Int()
		}

		flattened := flatten.Invoke(r.Get("data"))
		if flattened.Length() == 0 {
			return nil
		}

		s := make([]float32, flattened.Length())
		arr, free := jsutil.SliceToTypedArray(s)
		arr.Call("set", flattened)
		free()

		samples.samples = append(samples.samples, s)
		samples.lengthInSamples += int64(len(s)) / int64(samples.channels)
		return nil
	})
	defer f.Release()

	arr, free := jsutil.SliceToTypedArray(buf)
	js.Global().Get("stbvorbis").Call("decode", arr, f)
	free()

	if err := <-ch; err != nil {
		return nil, 0, 0, err
	}

	return samples, samples.channels, sampleRate, nil
}
