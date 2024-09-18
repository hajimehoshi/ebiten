// Copyright 2023 The Ebitengine Authors
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

package file

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type VirtualFS struct {
	paths []string
}

func NewVirtualFS(paths []string) *VirtualFS {
	fs := &VirtualFS{}
	fs.paths = make([]string, len(paths))
	copy(fs.paths, paths)
	return fs
}

func (v *VirtualFS) newRootFS() *virtualFSRoot {
	var root virtualFSRoot
	root.addRealPaths(v.paths)
	return &root
}

func (v *VirtualFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	if name == "." {
		return v.newRootFS(), nil
	}

	// A valid path must not include a token "." or "..", except for "." itself.
	es := strings.Split(name, "/")
	for _, realPath := range v.paths {
		if filepath.Base(realPath) != es[0] {
			continue
		}
		// os.File should implement fs.File interface, so this should be fine even on Windows.
		// See https://cs.opensource.google/go/go/+/refs/tags/go1.23.0:src/os/file.go;l=695-710
		return os.Open(filepath.Join(append([]string{realPath}, es[1:]...)...))
	}

	return nil, &fs.PathError{
		Op:   "open",
		Path: name,
		Err:  fs.ErrNotExist,
	}
}

type virtualFSRoot struct {
	realPaths []string
	offset    int
	m         sync.Mutex
}

func (v *virtualFSRoot) addRealPaths(paths []string) {
	v.m.Lock()
	defer v.m.Unlock()

	for _, path := range paths {
		// If the path consists entirely of separators, filepath.Base returns a single separator.
		// On Windows, filepath.Base(`C:\`) == `\`.
		// Skip root directory paths on purpose. This is almost the same behavior as the Chrome browser.
		if filepath.Base(path) == string(filepath.Separator) {
			continue
		}
		v.realPaths = append(v.realPaths, path)
	}
	sort.Strings(v.realPaths)
}

func (v *virtualFSRoot) Stat() (fs.FileInfo, error) {
	return &virtualFSRootFileInfo{}, nil
}

func (v *virtualFSRoot) Read([]byte) (int, error) {
	return 0, &fs.PathError{
		Op:   "read",
		Path: "/",
		Err:  errors.New("is a directory"),
	}
}

func (v *virtualFSRoot) Close() error {
	return nil
}

func (v *virtualFSRoot) ReadDir(count int) ([]fs.DirEntry, error) {
	v.m.Lock()
	defer v.m.Unlock()

	n := len(v.realPaths) - v.offset

	if n == 0 {
		if count <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}

	if count > 0 && n > count {
		n = count
	}

	ents := make([]fs.DirEntry, n)
	for i := range ents {
		fi, err := os.Stat(v.realPaths[v.offset+i])
		if err != nil {
			if count <= 0 {
				return ents, err
			}
			return nil, err
		}
		ents[i] = fs.FileInfoToDirEntry(fi)
	}
	v.offset += n

	return ents, nil
}

type virtualFSRootFileInfo struct {
}

func (v *virtualFSRootFileInfo) Name() string {
	return "."
}

func (v *virtualFSRootFileInfo) Size() int64 {
	return 0
}

func (v *virtualFSRootFileInfo) Mode() fs.FileMode {
	return 0555 | fs.ModeDir
}

func (v *virtualFSRootFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (v *virtualFSRootFileInfo) IsDir() bool {
	return true
}

func (v *virtualFSRootFileInfo) Sys() any {
	return nil
}
