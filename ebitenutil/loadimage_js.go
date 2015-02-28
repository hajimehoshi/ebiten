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
	"github.com/gopherjs/gopherjs/js"
	"github.com/hajimehoshi/ebiten"
	"image"
)

func NewImageFromFile(path string, filter ebiten.Filter) (*ebiten.Image, image.Image, error) {
	var err error
	var content *js.Object
	ch := make(chan struct{})
	req := js.Global.Get("XMLHttpRequest").New()
	req.Call("open", "GET", path, true)
	req.Set("responseType", "arraybuffer")
	req.Call("addEventListener", "load", func() {
		defer close(ch)
		status := req.Get("status").Int()
		if 200 <= status && status < 400 {
			content = req.Get("response")
			return
		}
		err = errors.New(fmt.Sprintf("http error: %d", status))
	})
	req.Call("addEventListener", "error", func() {
		defer close(ch)
		err = errors.New(fmt.Sprintf("XMLHttpRequest error: %s", req.Get("statusText").String()))
	})
	req.Call("send")
	<-ch
	if err != nil {
		return nil, nil, err
	}

	data := js.Global.Get("Uint8Array").New(content).Interface().([]uint8)
	b := bytes.NewBuffer(data)
	img, _, err := image.Decode(b)
	if err != nil {
		return nil, nil, err
	}
	img2, err := ebiten.NewImageFromImage(img, filter)
	if err != nil {
		return nil, nil, err
	}
	return img2, img, nil
}
