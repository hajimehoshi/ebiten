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

	"github.com/gopherjs/gopherwasm/js"
)

func init() {
	// Eval wasm.js first to set the Wasm binary to Module.
	js.Global().Get("window").Call("eval", string(wasm_js))

	js.Global().Get("window").Call("eval", string(stbvorbis_js))
	js.Global().Get("window").Call("eval", string(decode_js))

	ch := make(chan struct{})
	js.Global().Get("_ebiten").Call("initializeVorbisDecoder", js.NewCallback(func([]js.Value) {
		close(ch)
	}))
	<-ch
}

func DecodeVorbis(buf []byte) ([]int16, int, int, error) {
	r := js.Global().Get("_ebiten").Call("decodeVorbis", buf)
	if r == js.Null() {
		return nil, 0, 0, fmt.Errorf("audio/vorbis/internal/stb: decode failed")
	}
	data := make([]int16, r.Get("data").Get("length").Int())
	// TODO: Use js.TypeArrayOf
	arr := js.ValueOf(data)
	arr.Call("set", r.Get("data"))
	return data, r.Get("channels").Int(), r.Get("sampleRate").Int(), nil
}
