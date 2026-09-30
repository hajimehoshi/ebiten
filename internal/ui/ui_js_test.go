// Copyright 2026 The Ebitengine Authors
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

package ui_test

import (
	"syscall/js"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

// waitFrames waits for the given number of rendering updates, and reports whether they happened.
func waitFrames(n int) bool {
	const timeout = 10 * time.Second
	for i := 0; i < n; i++ {
		ch := make(chan struct{})
		onFrame := js.FuncOf(func(this js.Value, args []js.Value) any {
			close(ch)
			return nil
		})
		id := js.Global().Call("requestAnimationFrame", onFrame)
		select {
		case <-ch:
			onFrame.Release()
		case <-time.After(timeout):
			js.Global().Call("cancelAnimationFrame", id)
			onFrame.Release()
			return false
		}
	}
	return true
}

func TestInitWithNilBody(t *testing.T) {
	// document is undefined on node.js.
	document := js.Global().Get("document")
	if !document.Truthy() {
		t.Skip("document is not defined")
	}

	// A document that is still being loaded may still get its body, so the load event must have
	// been fired. readyState "complete" does not tell that, but loadEventEnd does.
	// https://w3c.github.io/navigation-timing/#dom-performancenavigationtiming-loadeventend
	navigation := js.Global().Get("performance").Call("getEntriesByType", "navigation")
	if navigation.Length() == 0 || navigation.Index(0).Get("loadEventEnd").Float() == 0 {
		t.Skip("the load event has not been fired yet")
	}

	if !document.Get("body").Truthy() {
		t.Fatal("document.body is nil")
	}

	// The pending first delivery of the ResizeObserver on the canvas of the user interface created
	// by the package initialization uses document.body, which panics while it is shadowed, and
	// observations are delivered in rendering updates, so let a couple of frames pass.
	if !waitFrames(2) {
		t.Skip("no frame is rendered")
	}

	// Shadowing document.body keeps the document size, unlike removing the body element.
	shadow := js.Global().Get("Object").New()
	shadow.Set("configurable", true)
	shadow.Set("value", js.Null())
	js.Global().Get("Object").Call("defineProperty", document, "body", shadow)
	t.Cleanup(func() {
		js.Global().Get("Reflect").Call("deleteProperty", document, "body")
	})

	u := &ui.UserInterface{}
	if err := u.InitForTest(); err == nil {
		t.Error("init with a nil body succeeded; want an error")
	}
}
