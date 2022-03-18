// Copyright 2021 The Ebiten Authors
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

//go:build !ebitenexternaldll
// +build !ebitenexternaldll

package glfw

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
	"golang.org/x/sys/windows"
)

func writeDLLFile(name string) error {
	f, err := gzip.NewReader(bytes.NewReader(glfwDLLCompressed))
	if err != nil {
		return err
	}
	defer f.Close()

	out, err := os.Create(name)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, f); err != nil {
		return err
	}
	return nil
}

func loadDLL() (*dll, error) {
	cachedir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	dir := filepath.Join(cachedir, "ebiten")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// Make the file operation atomic among multiple processes.
	fl := flock.New(filepath.Join(dir, glfwDLLHash+".lock"))
	fl.Lock()
	defer fl.Unlock()

	fn := filepath.Join(dir, glfwDLLHash+".dll")
	if _, err := os.Stat(fn); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}

		// Create a DLL as a temporary file and then rename it later.
		// Without the temporary file, writing a DLL might fail in the process of writing and Ebiten cannot
		// notice that the DLL file is incomplete.
		if err := writeDLLFile(fn + ".tmp"); err != nil {
			return nil, err
		}

		if err := os.Rename(fn+".tmp", fn); err != nil {
			return nil, err
		}
	}

	return &dll{
		d: windows.NewLazyDLL(fn),
	}, nil
}
