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

// fakeWindowDocument is a stand-in for a window and a document. A test controls readyState and
// body, and when the load event is fired.
type fakeWindowDocument struct {
	window   js.Value
	document js.Value

	// loadBody is the body the document has when the load event is fired.
	loadBody js.Value

	// loadDelay is how long after a load listener was registered the load event is fired.
	// A negative value means that the load event is not fired anymore, as it was already fired
	// before the listener was registered.
	loadDelay time.Duration

	// numLoadListeners is the number of the load listeners registered on the window.
	numLoadListeners int
}

func newFakeWindowDocument(readyState string, body js.Value) *fakeWindowDocument {
	f := &fakeWindowDocument{
		loadDelay: -1,
	}
	f.document = js.Global().Get("Object").New()
	f.document.Set("readyState", readyState)
	f.document.Set("body", body)
	f.window = js.Global().Get("Object").New()
	f.window.Set("addEventListener", js.FuncOf(func(this js.Value, args []js.Value) any {
		if args[0].String() != "load" {
			return nil
		}
		f.numLoadListeners++
		if f.loadDelay < 0 {
			return nil
		}
		listener := args[1]
		body := f.loadBody
		// The load event is fired in a task, as a load event that is queued but not fired yet is.
		js.Global().Get("setTimeout").Invoke(js.FuncOf(func(this js.Value, args []js.Value) any {
			f.document.Set("body", body)
			listener.Invoke()
			return nil
		}), float64(f.loadDelay/time.Millisecond))
		return nil
	}))
	return f
}

// loadAt makes the window fire the load event, which gives the document the given body, after the
// given duration once a load listener was registered.
func (f *fakeWindowDocument) loadAt(body js.Value, d time.Duration) {
	f.loadBody = body
	f.loadDelay = d
}

// TestWaitForBodyWaitsForAPendingLoadEvent tests that a document without its body is waited for
// until the load event is fired while the load event has not been fired yet, which is the case
// while readyState is "loading" or "interactive". A page can give its body in a load handler, and
// waiting only for a moment would lose that body.
func TestWaitForBodyWaitsForAPendingLoadEvent(t *testing.T) {
	const delay = 100 * time.Millisecond

	for _, readyState := range []string{"loading", "interactive"} {
		t.Run(readyState, func(t *testing.T) {
			f := newFakeWindowDocument(readyState, js.Null())
			f.loadAt(js.Global().Get("Object").New(), delay)

			if err := ui.WaitForBodyForTest(f.window, f.document); err != nil {
				t.Errorf("waitForBody with readyState %q and with a load handler giving the body failed: %v", readyState, err)
			}
			if !f.document.Get("body").Truthy() {
				t.Errorf("waitForBody with readyState %q: document.body is not the body the load event gave", readyState)
			}
			if got := f.numLoadListeners; got != 1 {
				t.Errorf("waitForBody with readyState %q registered %d load listeners; want 1", readyState, got)
			}
		})
	}
}

// TestWaitForBodyWithAQueuedLoadEvent tests that the load event is waited for even after readyState
// became "complete", for readyState becomes "complete" before the load event is fired, and the load
// event can still be queued when waitForBody is called, as when it is called from a
// readystatechange handler.
func TestWaitForBodyWithAQueuedLoadEvent(t *testing.T) {
	f := newFakeWindowDocument("complete", js.Null())
	// The load event is fired as a task queued by the steps making readyState "complete", so it
	// is fired before any task added afterwards.
	f.loadAt(js.Global().Get("Object").New(), 0)

	if err := ui.WaitForBodyForTest(f.window, f.document); err != nil {
		t.Errorf("waitForBody with the load event queued failed: %v", err)
	}
	if !f.document.Get("body").Truthy() {
		t.Error("waitForBody with the load event queued: document.body is not the body the load event gave")
	}
}

// TestWaitForBodyWithoutAPendingLoadEvent tests that the load event is not waited for once it was
// fired, for a load listener added after the event was fired is never invoked and waiting for it
// would wait forever. The nil body must not be used, either, so an error is reported.
func TestWaitForBodyWithoutAPendingLoadEvent(t *testing.T) {
	t.Run("no load event", func(t *testing.T) {
		f := newFakeWindowDocument("complete", js.Null())

		if err := ui.WaitForBodyForTest(f.window, f.document); err == nil {
			t.Error("waitForBody with a document that has no body succeeded; want an error")
		}
	})

	t.Run("load event fired later", func(t *testing.T) {
		f := newFakeWindowDocument("complete", js.Null())
		// The load event is fired long after the task that tells that the load event was
		// already fired, so it is a load event that waitForBody must not wait for.
		f.loadAt(js.Global().Get("Object").New(), 200*time.Millisecond)

		if err := ui.WaitForBodyForTest(f.window, f.document); err == nil {
			t.Error("waitForBody with a document that has no body succeeded; want an error")
		}
	})
}

// TestWaitForBodyWithBody tests that a document that has its body is not waited for whatever its
// loading state is.
func TestWaitForBodyWithBody(t *testing.T) {
	for _, readyState := range []string{"loading", "interactive", "complete"} {
		t.Run(readyState, func(t *testing.T) {
			f := newFakeWindowDocument(readyState, js.Global().Get("Object").New())

			if err := ui.WaitForBodyForTest(f.window, f.document); err != nil {
				t.Errorf("waitForBody with readyState %q and with a body failed: %v", readyState, err)
			}
			if got := f.numLoadListeners; got != 0 {
				t.Errorf("waitForBody with readyState %q and with a body registered %d load listeners; want 0", readyState, got)
			}
		})
	}
}

// TestInitWithNilBody tests that the initialization of a document that has no body reports an
// error after the load event was fired. The initialization panics if it uses the nil body, and
// waits forever if it waits for a load event that was already fired. The reported error makes the
// program stop at the package initialization, for no game is running yet to receive an error.
func TestInitWithNilBody(t *testing.T) {
	// While the load event has not been fired yet, a document without its body is just a document
	// that is still being loaded, and nothing can be told about it.
	document := js.Global().Get("document")
	if document.Get("readyState").String() != "complete" {
		t.Skip("the document has not been loaded yet")
	}
	if !document.Get("body").Truthy() {
		t.Fatal("document.body is nil")
	}

	// Shadow document.body with null instead of removing the body element from the document, so
	// that the document is not resized while the body is gone.
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
