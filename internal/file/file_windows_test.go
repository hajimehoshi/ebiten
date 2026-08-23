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

	if got, want := dirEntryNames(t, vfs, "."), []string{"C:", `\\host\share`}; !slices.Equal(got, want) {
		t.Errorf("names: got: %v, want: %v", got, want)
	}
}

func TestFSUNCShareRootPath(t *testing.T) {
	vfs := newVirtualFS(t, []string{`\\host\share`, `C:\a\x.txt`, `C:\b\y.txt`})

	// A share root is a root directory path and is skipped, so only the volume C: is left.
	if got, want := dirEntryNames(t, vfs, "."), []string{"a", "b"}; !slices.Equal(got, want) {
		t.Errorf("names: got: %v, want: %v", got, want)
	}
}
