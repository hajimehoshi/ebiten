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

package stb

import (
	"fmt"
	"io"

	"github.com/gopherjs/gopherwasm/js"
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
	samples [][]float32
	length  int64
}

func (s *Samples) Read(buf []float32) (int, error) {
	if len(s.samples) == 0 {
		return 0, io.EOF
	}
	n := copy(buf, s.samples[0])
	s.samples[0] = s.samples[0][n:]
	if len(s.samples[0]) == 0 {
		s.samples = s.samples[1:]
	}
	return n, nil
}

func (s *Samples) Length() int64 {
	return s.length
}

func DecodeVorbis(buf []byte) (*Samples, int, int, error) {
	ch := make(chan error)
	samples := &Samples{}
	channels := 0
	sampleRate := 0

	var f js.Callback
	f = js.NewCallback(func(args []js.Value) {
		r := args[0]

		if e := r.Get("error"); e != js.Null() {
			ch <- fmt.Errorf("audio/vorbis/internal/stb: decode error: %s", e.String())
			close(ch)
			f.Release()
			return
		}

		if r.Get("eof").Bool() {
			close(ch)
			f.Release()
			return
		}

		if channels == 0 {
			channels = r.Get("data").Length()
		}
		if sampleRate == 0 {
			sampleRate = r.Get("sampleRate").Int()
		}

		flattened := flatten.Invoke(r.Get("data"))
		s := make([]float32, flattened.Length())
		arr := js.TypedArrayOf(s)
		arr.Call("set", flattened)
		arr.Release()

		samples.samples = append(samples.samples, s)
		samples.length += int64(len(s)) / int64(channels)
	})

	arr := js.TypedArrayOf(buf)
	js.Global().Get("stbvorbis").Call("decode", arr, f)
	arr.Release()

	if err := <-ch; err != nil {
		return nil, 0, 0, err
	}

	return samples, channels, sampleRate, nil
}
