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
	"syscall/js"

	"github.com/hajimehoshi/ebiten/internal/jsutil"
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
	loadf := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer close(ch)
		status := req.Get("status").Int()
		if 200 <= status && status < 400 {
			content = req.Get("response")
			return nil
		}
		err = errors.New(fmt.Sprintf("http error: %d", status))
		return nil
	})
	defer loadf.Release()
	req.Call("addEventListener", "load", loadf)
	errorf := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer close(ch)
		err = errors.New(fmt.Sprintf("XMLHttpRequest error: %s", req.Get("statusText").String()))
		return nil
	})
	req.Call("addEventListener", "error", errorf)
	defer errorf.Release()
	req.Call("send")
	<-ch
	if err != nil {
		return nil, err
	}

	f := &file{bytes.NewReader(jsutil.ArrayBufferToSlice(content))}
	return f, nil
}
