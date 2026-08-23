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
	"testing"
)

func TestFSDifferentVolumes(t *testing.T) {
	vfs := newVirtualFS(t, []string{`C:\a\x.txt`, `D:\b\y.txt`})

	// The paths have no common ancestor directory. Volume names are listed instead.
	if got, want := dirEntryNames(t, vfs, "."), []string{"C:", "D:"}; !slices.Equal(got, want) {
		t.Errorf("names: got: %v, want: %v", got, want)
	}

	if got, want := dirEntryNames(t, vfs, "C:"), []string{"a"}; !slices.Equal(got, want) {
		t.Errorf("names: got: %v, want: %v", got, want)
	}
}

func TestFSSameUNCShare(t *testing.T) {
	vfs := newVirtualFS(t, []string{`\\host\share\a\x.txt`, `\\host\share\b\y.txt`})

	// The paths are on the same share, whose root is their common ancestor directory.
	if got, want := dirEntryNames(t, vfs, "."), []string{"a", "b"}; !slices.Equal(got, want) {
		t.Errorf("names: got: %v, want: %v", got, want)
	}
}

func TestFSDifferentUNCShares(t *testing.T) {
	vfs := newVirtualFS(t, []string{`\\host\share1\a\x.txt`, `\\host\share2\b\y.txt`})

	// A volume name is one path element even though it contains separators.
	if got, want := dirEntryNames(t, vfs, "."), []string{`\\host\share1`, `\\host\share2`}; !slices.Equal(got, want) {
		t.Errorf("names: got: %v, want: %v", got, want)
	}

	if got, want := dirEntryNames(t, vfs, `\\host\share1`), []string{"a"}; !slices.Equal(got, want) {
		t.Errorf("names: got: %v, want: %v", got, want)
	}
}

func TestFSUNCAndDriveLetter(t *testing.T) {
	vfs := newVirtualFS(t, []string{`C:\a\x.txt`, `\\host\share\b\y.txt`})

	// A drive letter and a share are different volumes, so both are listed at the root.
	if got, want := dirEntryNames(t, vfs, "."), []string{"C:", `\\host\share`}; !slices.Equal(got, want) {
		t.Errorf("names: got: %v, want: %v", got, want)
	}

	wantAbsPaths := map[string]string{
		"C:":           `C:\`,
		`\\host\share`: `\\host\share\`,
	}
	ents, err := fs.ReadDir(vfs, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range ents {
		if got, want := absPath(t, ent), wantAbsPaths[ent.Name()]; got != want {
			t.Errorf("AbsPath() for %s: got: %s, want: %s", ent.Name(), got, want)
		}
	}

	wantSubAbsPaths := map[string]string{
		"C:":           `C:\a`,
		`\\host\share`: `\\host\share\b`,
	}
	for _, ent := range ents {
		subEnts, err := fs.ReadDir(vfs, ent.Name())
		if err != nil {
			t.Fatal(err)
		}
		if len(subEnts) != 1 {
			t.Fatalf("len(subEnts) for %s: got: %d, want: %d", ent.Name(), len(subEnts), 1)
		}
		if got, want := absPath(t, subEnts[0]), wantSubAbsPaths[ent.Name()]; got != want {
			t.Errorf("AbsPath() under %s: got: %s, want: %s", ent.Name(), got, want)
		}
	}
}

func TestFSUNCShareRootPath(t *testing.T) {
	vfs := newVirtualFS(t, []string{`\\host\share`, `C:\a\x.txt`, `C:\b\y.txt`})

	// A share root is a root directory path and is skipped, so only the volume C: is left.
	if got, want := dirEntryNames(t, vfs, "."), []string{"a", "b"}; !slices.Equal(got, want) {
		t.Errorf("names: got: %v, want: %v", got, want)
	}
}

func TestFSAbsPathDifferentVolumes(t *testing.T) {
	vfs := newVirtualFS(t, []string{`C:\a\x.txt`, `D:\b\y.txt`})

	// The file system has no root directory, so a child of the root is a volume root.
	want := map[string]string{
		"C:": `C:\`,
		"D:": `D:\`,
	}
	ents, err := fs.ReadDir(vfs, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range ents {
		if got, want := absPath(t, ent), want[ent.Name()]; got != want {
			t.Errorf("AbsPath() for %s: got: %s, want: %s", ent.Name(), got, want)
		}
	}

	subEnts, err := fs.ReadDir(vfs, "C:")
	if err != nil {
		t.Fatal(err)
	}
	if len(subEnts) != 1 {
		t.Fatalf("len(subEnts): got: %d, want: %d", len(subEnts), 1)
	}
	if got, want := absPath(t, subEnts[0]), `C:\a`; got != want {
		t.Errorf("AbsPath(): got: %s, want: %s", got, want)
	}
}

func TestFSAbsPathUNC(t *testing.T) {
	vfs := newVirtualFS(t, []string{`\\host\share1\a\x.txt`, `\\host\share2\b\y.txt`})

	// The shares are different volumes, so a child of the root is a share root.
	want := map[string]string{
		`\\host\share1`: `\\host\share1\`,
		`\\host\share2`: `\\host\share2\`,
	}
	ents, err := fs.ReadDir(vfs, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range ents {
		if got, want := absPath(t, ent), want[ent.Name()]; got != want {
			t.Errorf("AbsPath() for %s: got: %s, want: %s", ent.Name(), got, want)
		}
	}

	subEnts, err := fs.ReadDir(vfs, `\\host\share1`)
	if err != nil {
		t.Fatal(err)
	}
	if len(subEnts) != 1 {
		t.Fatalf("len(subEnts): got: %d, want: %d", len(subEnts), 1)
	}
	if got, want := absPath(t, subEnts[0]), `\\host\share1\a`; got != want {
		t.Errorf("AbsPath(): got: %s, want: %s", got, want)
	}
}

func TestFSAbsPathSameUNCShare(t *testing.T) {
	vfs := newVirtualFS(t, []string{`\\host\share\a\x.txt`, `\\host\share\b\y.txt`})

	// The share root is the common ancestor directory, so it is the root of the file system.
	want := map[string]string{
		"a": `\\host\share\a`,
		"b": `\\host\share\b`,
	}
	ents, err := fs.ReadDir(vfs, ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, ent := range ents {
		if got, want := absPath(t, ent), want[ent.Name()]; got != want {
			t.Errorf("AbsPath() for %s: got: %s, want: %s", ent.Name(), got, want)
		}
	}
}
