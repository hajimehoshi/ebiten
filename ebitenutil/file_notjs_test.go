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

//go:build !android && !ios && !js

package ebitenutil_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func TestOpenFileMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing")
	f, err := ebitenutil.OpenFile(filepath.ToSlash(path))
	if f != nil {
		t.Errorf("OpenFile returned a non-nil stream on error: %T", f)
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("OpenFile error = %v, want os.ErrNotExist", err)
	}
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || pathErr.Path != path {
		t.Errorf("OpenFile error = %v, want original path error for %q", err, path)
	}
}

func TestOpenFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("abc"), 0600); err != nil {
		t.Fatal(err)
	}
	f, err := ebitenutil.OpenFile(filepath.ToSlash(path))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()
	if _, err := f.Seek(1, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "bc" {
		t.Errorf("read = %q, want bc", b)
	}
}
