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
