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

package file_test

import (
	"errors"
	"io"
	"io/fs"
	"slices"
	"sync"
	"syscall/js"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/file"
)

// newFakeDirEntry returns a fake FileSystemDirectoryEntry with the given child directories.
// Its readers invoke their callbacks asynchronously, like a browser does.
func newFakeDirEntry(t *testing.T, name string, childNames []string) js.Value {
	var funcs []js.Func
	t.Cleanup(func() {
		for _, f := range funcs {
			f.Release()
		}
	})
	register := func(f js.Func) js.Func {
		funcs = append(funcs, f)
		return f
	}

	children := make([]any, 0, len(childNames))
	for _, n := range childNames {
		children = append(children, map[string]any{
			"name":        n,
			"fullPath":    "/" + name + "/" + n,
			"isFile":      false,
			"isDirectory": true,
		})
	}

	createReader := register(js.FuncOf(func(this js.Value, args []js.Value) any {
		var read bool
		readEntries := register(js.FuncOf(func(this js.Value, args []js.Value) any {
			ents := []any{}
			if !read {
				ents = children
				read = true
			}
			js.Global().Call("setTimeout", args[0], 0, js.ValueOf(ents))
			return nil
		}))
		return map[string]any{
			"readEntries": readEntries,
		}
	}))

	return js.ValueOf(map[string]any{
		"name":         name,
		"fullPath":     "/" + name,
		"isFile":       false,
		"isDirectory":  true,
		"createReader": createReader,
	})
}

func TestDirReadDirConcurrently(t *testing.T) {
	childNames := []string{"a", "b", "c"}
	root := newFakeDirEntry(t, "root", childNames)

	fsys, err := file.NewFileEntryFS([]js.Value{root})
	if err != nil {
		t.Fatal(err)
	}
	f, err := fsys.Open(".")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = f.Close()
	}()

	d, ok := f.(fs.ReadDirFile)
	if !ok {
		t.Fatalf("a directory must implement fs.ReadDirFile but not: %T", f)
	}

	const goroutines = 2
	entss := make([][]fs.DirEntry, goroutines)
	errs := make([]error, goroutines)
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			entss[i], errs[i] = d.ReadDir(-1)
		}()
	}
	wg.Wait()

	var names []string
	for i := range goroutines {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		for _, ent := range entss[i] {
			names = append(names, ent.Name())
		}
	}
	slices.Sort(names)

	if got, want := names, childNames; !slices.Equal(got, want) {
		t.Errorf("names: got: %v, want: %v", got, want)
	}
}

// fakeFileCalls counts the asynchronous calls a fake file entry receives.
type fakeFileCalls struct {
	file        int
	arrayBuffer int
}

// newFakeFileEntry returns a fake FileSystemDirectoryEntry with one child file of the given
// content. Its getFile, file and arrayBuffer settle asynchronously, like a browser does.
func newFakeFileEntry(t *testing.T, name string, content []byte, calls *fakeFileCalls) js.Value {
	var funcs []js.Func
	t.Cleanup(func() {
		for _, f := range funcs {
			f.Release()
		}
	})
	register := func(f js.Func) js.Func {
		funcs = append(funcs, f)
		return f
	}

	u8 := js.Global().Get("Uint8Array").New(len(content))
	js.CopyBytesToJS(u8, content)
	buf := u8.Get("buffer")

	arrayBuffer := register(js.FuncOf(func(this js.Value, args []js.Value) any {
		calls.arrayBuffer++
		executor := register(js.FuncOf(func(this js.Value, args []js.Value) any {
			js.Global().Call("setTimeout", args[0], 0, buf)
			return nil
		}))
		return js.Global().Get("Promise").New(executor)
	}))

	fileObj := js.ValueOf(map[string]any{
		"name":         name,
		"size":         len(content),
		"lastModified": 0,
		"arrayBuffer":  arrayBuffer,
	})

	fileFn := register(js.FuncOf(func(this js.Value, args []js.Value) any {
		calls.file++
		js.Global().Call("setTimeout", args[0], 0, fileObj)
		return nil
	}))

	getFile := register(js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Call("setTimeout", args[2], 0, js.ValueOf(map[string]any{
			"name":        name,
			"fullPath":    "/root/" + name,
			"isFile":      true,
			"isDirectory": false,
			"file":        fileFn,
		}))
		return nil
	}))

	return js.ValueOf(map[string]any{
		"name":        "root",
		"fullPath":    "/root",
		"isFile":      false,
		"isDirectory": true,
		"getFile":     getFile,
	})
}

func openFakeFile(t *testing.T, content []byte, calls *fakeFileCalls) fs.File {
	t.Helper()

	root := newFakeFileEntry(t, "a.txt", content, calls)
	fsys, err := file.NewFileEntryFS([]js.Value{root})
	if err != nil {
		t.Fatal(err)
	}
	f, err := fsys.Open("a.txt")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = f.Close()
	})
	return f
}

func TestFileReadConcurrently(t *testing.T) {
	content := []byte("0123456789abcdefghij")
	var calls fakeFileCalls
	f := openFakeFile(t, content, &calls)

	const goroutines = 2
	bufs := make([][]byte, goroutines)
	ns := make([]int, goroutines)
	errs := make([]error, goroutines)
	var wg sync.WaitGroup
	for i := range goroutines {
		wg.Go(func() {
			bufs[i] = make([]byte, len(content)/goroutines)
			ns[i], errs[i] = f.Read(bufs[i])
		})
	}
	wg.Wait()

	if got, want := calls.arrayBuffer, 1; got != want {
		t.Errorf("arrayBuffer calls: got: %d, want: %d", got, want)
	}

	var total int
	for i := range goroutines {
		if err := errs[i]; err != nil && !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}
		total += ns[i]
	}
	if got, want := total, len(content); got != want {
		t.Errorf("read bytes in total: got: %d, want: %d", got, want)
	}

	// The reads must cover the content exactly once, in either order.
	got := string(bufs[0][:ns[0]]) + string(bufs[1][:ns[1]])
	half := len(content) / goroutines
	if want, wantSwapped := string(content), string(content[half:])+string(content[:half]); got != want && got != wantSwapped {
		t.Errorf("concatenated reads: got: %q, want: %q or %q", got, want, wantSwapped)
	}
}

func TestFileStatAndReadConcurrently(t *testing.T) {
	content := []byte("0123456789")
	var calls fakeFileCalls
	f := openFakeFile(t, content, &calls)

	var size int64
	var statErr, readErr error
	var wg sync.WaitGroup
	wg.Go(func() {
		fi, err := f.Stat()
		if err != nil {
			statErr = err
			return
		}
		size = fi.Size()
	})
	wg.Go(func() {
		buf := make([]byte, len(content))
		_, readErr = f.Read(buf)
	})
	wg.Wait()

	if statErr != nil {
		t.Fatal(statErr)
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		t.Fatal(readErr)
	}
	if got, want := calls.file, 1; got != want {
		t.Errorf("file calls: got: %d, want: %d", got, want)
	}
	if got, want := size, int64(len(content)); got != want {
		t.Errorf("size: got: %d, want: %d", got, want)
	}
}
