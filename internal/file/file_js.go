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

package file

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"syscall/js"
	"time"
)

type FileEntryFS struct {
	rootEntries []js.Value
}

func NewFileEntryFS(rootEntries []js.Value) (*FileEntryFS, error) {
	// Check all the full paths are the same.
	var fullpath string
	for _, ent := range rootEntries {
		if fullpath == "" {
			fullpath = ent.Get("fullPath").String()
			continue
		}
		if fullpath != ent.Get("fullPath").String() {
			return nil, errors.New("file: all the full paths must be the same")
		}
	}
	return &FileEntryFS{
		rootEntries: rootEntries,
	}, nil
}

func (f *FileEntryFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrNotExist,
		}
	}

	if name == "." {
		var dirName string
		for _, ent := range f.rootEntries {
			if dirName == "" {
				dirName = ent.Get("name").String()
				continue
			}
			if dirName != ent.Get("name").String() {
				return nil, &fs.PathError{
					Op:   "open",
					Path: name,
					Err:  errors.New("invalid directory"),
				}
			}
		}
		return &dir{
			name:       dirName,
			dirEntries: f.rootEntries,
		}, nil
	}

	for _, ent := range f.rootEntries {
		var chEntry chan js.Value
		cbSuccess := js.FuncOf(func(this js.Value, args []js.Value) any {
			chEntry <- args[0]
			close(chEntry)
			return nil
		})
		defer cbSuccess.Release()

		cbFailure := js.FuncOf(func(this js.Value, args []js.Value) any {
			close(chEntry)
			return nil
		})
		defer cbFailure.Release()

		chEntry = make(chan js.Value)
		ent.Call("getFile", name, nil, cbSuccess, cbFailure)
		if entry := <-chEntry; entry.Truthy() {
			return &file{entry: entry}, nil
		}

		chEntry = make(chan js.Value)
		ent.Call("getDirectory", name, nil, cbSuccess, cbFailure)
		if entry := <-chEntry; entry.Truthy() {
			return &dir{
				name:       entry.Get("name").String(),
				dirEntries: []js.Value{entry},
			}, nil
		}
	}

	return nil, &fs.PathError{
		Op:   "open",
		Path: name,
		Err:  fs.ErrNotExist,
	}
}

type file struct {
	entry      js.Value
	file       js.Value
	offset     int64
	uint8Array js.Value
}

func getFile(entry js.Value) js.Value {
	ch := make(chan js.Value, 1)
	cb := js.FuncOf(func(this js.Value, args []js.Value) any {
		ch <- args[0]
		return nil
	})
	defer cb.Release()

	entry.Call("file", cb)
	return <-ch
}

func (f *file) ensureFile() js.Value {
	if f.file.Truthy() {
		return f.file
	}

	f.file = getFile(f.entry)
	return f.file
}

func (f *file) Stat() (fs.FileInfo, error) {
	return &fileInfo{
		name: f.entry.Get("name").String(),
		file: f.ensureFile(),
	}, nil
}

func (f *file) Read(buf []byte) (int, error) {
	if !f.uint8Array.Truthy() {
		chArrayBuffer := make(chan js.Value, 1)
		cbThen := js.FuncOf(func(this js.Value, args []js.Value) any {
			chArrayBuffer <- args[0]
			return nil
		})
		defer cbThen.Release()

		chError := make(chan js.Value, 1)
		cbCatch := js.FuncOf(func(this js.Value, args []js.Value) any {
			chError <- args[0]
			return nil
		})
		defer cbCatch.Release()

		f.ensureFile().Call("arrayBuffer").Call("then", cbThen).Call("catch", cbCatch)
		select {
		case ab := <-chArrayBuffer:
			f.uint8Array = js.Global().Get("Uint8Array").New(ab)
		case err := <-chError:
			return 0, fmt.Errorf("%s", err.Call("toString").String())
		}
	}

	size := int64(f.uint8Array.Get("byteLength").Float())
	if f.offset >= size {
		return 0, io.EOF
	}

	if len(buf) == 0 {
		return 0, nil
	}

	slice := f.uint8Array.Call("subarray", f.offset, f.offset+int64(len(buf)))
	n := slice.Get("byteLength").Int()
	js.CopyBytesToGo(buf[:n], slice)
	f.offset += int64(n)
	if f.offset >= size {
		return n, io.EOF
	}
	return n, nil
}

func (f *file) Close() error {
	return nil
}

type dir struct {
	name        string
	dirEntries  []js.Value
	fileEntries []js.Value
	offset      int
}

func (d *dir) Stat() (fs.FileInfo, error) {
	return &fileInfo{
		name: d.name,
	}, nil
}

func (d *dir) Read(buf []byte) (int, error) {
	return 0, &fs.PathError{
		Op:   "read",
		Path: d.name,
		Err:  errors.New("is a directory"),
	}
}

func (d *dir) Close() error {
	return nil
}

func (d *dir) ReadDir(count int) ([]fs.DirEntry, error) {
	if d.fileEntries == nil {
		names := map[string]struct{}{}
		for _, dirEntry := range d.dirEntries {
			ch := make(chan struct{})
			var rec js.Func
			cb := js.FuncOf(func(this js.Value, args []js.Value) any {
				entries := args[0]
				if entries.Length() == 0 {
					close(ch)
					return nil
				}
				for i := 0; i < entries.Length(); i++ {
					ent := entries.Index(i)
					name := ent.Get("name").String()
					// A name can be empty when this directory is a root directory.
					if name == "" {
						continue
					}
					// Avoid entry duplications. Entry duplications happen when multiple files are dropped on Chrome.
					if _, ok := names[name]; ok {
						continue
					}
					if !ent.Get("isFile").Bool() && !ent.Get("isDirectory").Bool() {
						continue
					}
					d.fileEntries = append(d.fileEntries, ent)
					names[name] = struct{}{}
				}
				rec.Value.Call("call")
				return nil
			})
			defer cb.Release()

			reader := dirEntry.Call("createReader")
			rec = js.FuncOf(func(this js.Value, args []js.Value) any {
				reader.Call("readEntries", cb)
				return nil
			})
			defer rec.Release()

			rec.Value.Call("call")
			<-ch
		}
	}

	n := len(d.fileEntries) - d.offset

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
		entry := d.fileEntries[d.offset+i]
		fi := &fileInfo{
			name: entry.Get("name").String(),
		}
		if entry.Get("isFile").Bool() {
			fi.file = getFile(entry)
		}
		ents[i] = fs.FileInfoToDirEntry(fi)
	}
	d.offset += n

	return ents, nil
}

type fileInfo struct {
	name string
	file js.Value
}

func (f *fileInfo) Name() string {
	return f.name
}

func (f *fileInfo) Size() int64 {
	if !f.file.Truthy() {
		return 0
	}
	return int64(f.file.Get("size").Float())
}

func (f *fileInfo) Mode() fs.FileMode {
	if !f.file.Truthy() {
		return 0555 | fs.ModeDir
	}
	return 0444
}

func (f *fileInfo) ModTime() time.Time {
	if !f.file.Truthy() {
		return time.Time{}
	}
	return time.UnixMilli(int64(f.file.Get("lastModified").Float()))
}

func (f *fileInfo) IsDir() bool {
	if !f.file.Truthy() {
		return true
	}
	return false
}

func (f *fileInfo) Sys() any {
	return nil
}
