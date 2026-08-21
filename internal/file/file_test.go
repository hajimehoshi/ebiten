// Copyright 2025 The Ebitengine Authors
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

//go:build !js

package file_test

import (
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/file"
)

func TestFSReadDir(t *testing.T) {
	vfs := file.NewVirtualFS([]string{"testdata/foo.txt", "testdata/dir"})

	rootEnts, err := fs.ReadDir(vfs, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(rootEnts) != 2 {
		t.Errorf("len(ents): got: %d, want: %d", len(rootEnts), 2)
	}
	slices.SortFunc(rootEnts, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})
	if got, want := rootEnts[0].Name(), "dir"; got != want {
		t.Errorf("ents[0].Name(): got: %s, want: %s", got, want)
	}
	if got, want := rootEnts[0].IsDir(), true; got != want {
		t.Errorf("ents[0].IsDir(): got: false, want: true")
	}
	if got, want := rootEnts[1].Name(), "foo.txt"; got != want {
		t.Errorf("ents[1].Name(): got: %s, want: %s", got, want)
	}
	if got, want := rootEnts[1].IsDir(), false; got != want {
		t.Errorf("ents[1].IsDir(): got: true, want: false")
	}

	subEnts, err := fs.ReadDir(vfs, "dir")
	if err != nil {
		t.Fatal(err)
	}
	if len(subEnts) != 2 {
		t.Errorf("len(ents): got: %d, want: %d", len(subEnts), 1)
	}
	slices.SortFunc(subEnts, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})
	if got, want := subEnts[0].Name(), "foo.txt"; got != want {
		t.Errorf("ents[0].Name(): got: %s, want: %s", got, want)
	}
	if got, want := subEnts[0].IsDir(), false; got != want {
		t.Errorf("ents[0].IsDir(): got: false, want: true")
	}
	if got, want := subEnts[1].Name(), "qux.txt"; got != want {
		t.Errorf("ents[1].Name(): got: %s, want: %s", got, want)
	}
	if got, want := subEnts[1].IsDir(), false; got != want {
		t.Errorf("ents[1].IsDir(): got: true, want: false")
	}

	if _, err := fs.ReadDir(vfs, "baz.txt"); err == nil {
		t.Errorf("fs.ReadDir on a file must return an error")
	}
}

func TestFSReadFile(t *testing.T) {
	vfs := file.NewVirtualFS([]string{"testdata/foo.txt", "testdata/dir"})

	content, err := fs.ReadFile(vfs, "foo.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "foo\n"; got != want {
		t.Errorf("content: got: %q, want: %q", got, want)
	}

	subContent, err := fs.ReadFile(vfs, "dir/foo.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(subContent), "dir/foo\n"; got != want {
		t.Errorf("content: got: %q, want: %q", got, want)
	}

	if _, err := fs.ReadFile(vfs, "."); err == nil {
		t.Errorf("fs.ReadFile on the root directory must return an error")
	}

	if _, err := fs.ReadFile(vfs, "dir"); err == nil {
		t.Errorf("fs.ReadFile on a directory must return an error")
	}

	if _, err := fs.ReadFile(vfs, "bar.txt"); err == nil {
		t.Errorf("fs.ReadFile on a file not in the file system must return an error")
	}
}
