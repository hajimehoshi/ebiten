// Copyright 2015 Hajime Hoshi
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

package ebitenutil

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/gopherjs/gopherwasm/js"
)

type file struct {
	*bytes.Reader
}

func (f *file) Close() error {
	return nil
}

func OpenFile(path string) (ReadSeekCloser, error) {
	var err error
	var content js.Value
	ch := make(chan struct{})
	req := js.Global().Get("XMLHttpRequest").New()
	req.Call("open", "GET", path, true)
	req.Set("responseType", "arraybuffer")
	loadCallback := js.NewCallback(func([]js.Value) {
		defer close(ch)
		status := req.Get("status").Int()
		if 200 <= status && status < 400 {
			content = req.Get("response")
			return
		}
		err = errors.New(fmt.Sprintf("http error: %d", status))
	})
	defer loadCallback.Release()
	req.Call("addEventListener", "load", loadCallback)
	errorCallback := js.NewCallback(func([]js.Value) {
		defer close(ch)
		err = errors.New(fmt.Sprintf("XMLHttpRequest error: %s", req.Get("statusText").String()))
	})
	req.Call("addEventListener", "error", errorCallback)
	defer errorCallback.Release()
	req.Call("send")
	<-ch
	if err != nil {
		return nil, err
	}

	uint8contentWrapper := js.Global().Get("Uint8Array").New(content)
	data := make([]byte, uint8contentWrapper.Get("byteLength").Int())
	arr := js.TypedArrayOf(data)
	arr.Call("set", uint8contentWrapper)
	arr.Release()
	f := &file{bytes.NewReader(data)}
	return f, nil
}
