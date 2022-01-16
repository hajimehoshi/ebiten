// Copyright 2020 The Ebiten Authors
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

//go:build example
// +build example

package main

import (
	"fmt"
	"syscall/js"
)

func (g *Game) loseAndRestoreContext() {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementsByTagName", "canvas").Index(0)
	context := canvas.Call("getContext", "webgl2")
	if !context.Truthy() {
		context = canvas.Call("getContext", "webgl")
		if !context.Truthy() {
			context = canvas.Call("getContext", "experimental-webgl")
		}
	}

	if g.lost {
		return
	}

	// Edge might not support the extension. See
	// https://developer.mozilla.org/en-US/docs/Web/API/WEBGL_lose_context
	ext := context.Call("getExtension", "WEBGL_lose_context")
	if !ext.Truthy() {
		fmt.Println("Fail to force context lost. Edge might not support the extension yet.")
		return
	}

	ext.Call("loseContext")
	fmt.Println("Lost the context!")
	g.lost = true
}
