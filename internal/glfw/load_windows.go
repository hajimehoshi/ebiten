// Copyright 2018 The Ebiten Authors
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

package glfw

import (
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"

	"golang.org/x/sys/windows"
)

type dll struct {
	d    *windows.LazyDLL
	path string
}

func createTempDLL(content io.Reader) (string, error) {
	f, err := ioutil.TempFile("", "glfw.*.dll")
	if err != nil {
		return "", err
	}
	defer f.Close()

	fn := f.Name()

	if _, err := io.Copy(f, content); err != nil {
		return "", err
	}

	return fn, nil
}

func loadDLL() (*dll, error) {
	f, err := gzip.NewReader(bytes.NewReader(glfwDLLCompressed))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fn, err := createTempDLL(f)
	if err != nil {
		return nil, err
	}

	return &dll{
		d:    windows.NewLazyDLL(fn),
		path: fn,
	}, nil
}

func (d *dll) unload() error {
	if err := windows.FreeLibrary(windows.Handle(d.d.Handle())); err != nil {
		return err
	}
	if err := os.Remove(d.path); err != nil {
		return err
	}
	return nil
}

func init() {
	if _, err := loadDLL(); err != nil {
		panic(err)
	}
}
