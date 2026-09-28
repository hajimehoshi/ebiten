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

//go:build js

package ui

import (
	"syscall/js"
	"testing"
	"time"
)

// fakeWindowDocument is a stand-in for a window and a document. A test controls readyState and
// body, and fires the load event when it wants to.
type fakeWindowDocument struct {
	window   js.Value
	document js.Value

	// listeners are the load listeners registered on the window.
	listeners []js.Value

	// loadListenerRegistered receives a signal when a load listener is registered.
	loadListenerRegistered chan struct{}
}

func newFakeWindowDocument(readyState string, body js.Value) *fakeWindowDocument {
	f := &fakeWindowDocument{
		loadListenerRegistered: make(chan struct{}, 1),
	}
	f.document = js.Global().Get("Object").New()
	f.document.Set("readyState", readyState)
	f.document.Set("body", body)
	f.window = js.Global().Get("Object").New()
	f.window.Set("addEventListener", js.FuncOf(func(this js.Value, args []js.Value) any {
		if args[0].String() != "load" {
			return nil
		}
		f.listeners = append(f.listeners, args[1])
		select {
		case f.loadListenerRegistered <- struct{}{}:
		default:
		}
		return nil
	}))
	return f
}

// numLoadListeners returns the number of the load listeners registered on the window.
func (f *fakeWindowDocument) numLoadListeners() int {
	return len(f.listeners)
}

// fireLoad gives the document the given body and fires the load event, as a page whose load handler
// restores the body does.
func (f *fakeWindowDocument) fireLoad(body js.Value) {
	f.document.Set("body", body)
	// Fire a copy of the listeners: firing one unblocks the goroutine that registered it, and the
	// test might be over before this loop continues.
	listeners := append([]js.Value(nil), f.listeners...)
	for _, l := range listeners {
		l.Invoke()
	}
}

// TestWaitForBodyWaitsForAPendingLoadEvent tests that a document without its body is waited for
// while the load event can still be fired, which is the case until the load event is fired. A page
// can restore its body in a load handler, so the initialization must not go on without the body
// before the load event is fired.
func TestWaitForBodyWaitsForAPendingLoadEvent(t *testing.T) {
	for _, readyState := range []string{"loading", "interactive"} {
		t.Run(readyState, func(t *testing.T) {
			f := newFakeWindowDocument(readyState, js.Null())
			body := js.Global().Get("Object").New()

			errs := make(chan error, 1)
			go func() {
				errs <- waitForBody(f.window, f.document)
			}()

			select {
			case <-f.loadListenerRegistered:
			case <-time.After(10 * time.Second):
				t.Fatalf("waitForBody with readyState %q did not register a load listener", readyState)
			}
			f.fireLoad(body)

			select {
			case err := <-errs:
				if err != nil {
					t.Errorf("waitForBody with readyState %q and with the body restored by a load handler failed: %v", readyState, err)
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("waitForBody with readyState %q did not return even after the load event was fired", readyState)
			}

			if got := f.numLoadListeners(); got != 1 {
				t.Errorf("waitForBody with readyState %q registered %d load listeners; want 1", readyState, got)
			}
			if !f.document.Get("body").Truthy() {
				t.Errorf("waitForBody with readyState %q: document.body is not restored", readyState)
			}
		})
	}
}

// TestWaitForBodyWithoutAPendingLoadEvent tests that a document without its body is not waited for
// after the load event was fired, for a load listener added after the event was fired is never
// fired and waiting for it would wait forever. The nil body must not be used either, so an error
// is reported instead.
func TestWaitForBodyWithoutAPendingLoadEvent(t *testing.T) {
	f := newFakeWindowDocument("complete", js.Null())

	if err := waitForBody(f.window, f.document); err == nil {
		t.Error("waitForBody with a document that has no body succeeded; want an error")
	}
	if got := f.numLoadListeners(); got != 0 {
		t.Errorf("waitForBody with a document that has no body registered %d load listeners; want 0", got)
	}
}

// TestWaitForBodyWithBody tests that a document that has its body is not waited for whatever its
// loading state is.
func TestWaitForBodyWithBody(t *testing.T) {
	for _, readyState := range []string{"loading", "interactive", "complete"} {
		t.Run(readyState, func(t *testing.T) {
			f := newFakeWindowDocument(readyState, js.Global().Get("Object").New())

			if err := waitForBody(f.window, f.document); err != nil {
				t.Errorf("waitForBody with readyState %q and with a body failed: %v", readyState, err)
			}
			if got := f.numLoadListeners(); got != 0 {
				t.Errorf("waitForBody with readyState %q and with a body registered %d load listeners; want 0", readyState, got)
			}
		})
	}
}

// TestInitWithNilBody tests that the initialization with a nil body reports an error after the
// document was loaded. The initialization panics if the nil body is used, and waits forever if a
// load event that was already fired is waited for.
func TestInitWithNilBody(t *testing.T) {
	// While the load event can still be fired, a document without its body is just a document
	// that is still being loaded, and nothing can be told about it.
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

	u := &UserInterface{}
	if err := u.init(); err == nil {
		t.Error("init with a nil body succeeded; want an error")
	}
}
