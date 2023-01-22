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
	rootEntry js.Value
}

func NewFileEntryFS(root js.Value) *FileEntryFS {
	return &FileEntryFS{
		rootEntry: root,
	}
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
		return &dir{entry: f.rootEntry}, nil
	}

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
	f.rootEntry.Call("getFile", name, nil, cbSuccess, cbFailure)
	if entry := <-chEntry; entry.Truthy() {
		return &file{entry: entry}, nil
	}

	chEntry = make(chan js.Value)
	f.rootEntry.Call("getDirectory", name, nil, cbSuccess, cbFailure)
	if entry := <-chEntry; entry.Truthy() {
		return &dir{entry: entry}, nil
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
		entry: f.entry,
		file:  f.ensureFile(),
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
	entry   js.Value
	entries []js.Value
	offset  int
}

func (d *dir) Stat() (fs.FileInfo, error) {
	return &fileInfo{
		entry: d.entry,
	}, nil
}

func (d *dir) Read(buf []byte) (int, error) {
	return 0, &fs.PathError{
		Op:   "read",
		Path: d.entry.Get("name").String(),
		Err:  errors.New("is a directory"),
	}
}

func (d *dir) Close() error {
	return nil
}

func (d *dir) ReadDir(count int) ([]fs.DirEntry, error) {
	if d.entries == nil {
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
				// A name can be empty when this directory is a root directory.
				if ent.Get("name").String() == "" {
					continue
				}
				if !ent.Get("isFile").Bool() && !ent.Get("isDirectory").Bool() {
					continue
				}
				d.entries = append(d.entries, ent)
			}
			rec.Value.Call("call")
			return nil
		})
		defer cb.Release()

		reader := d.entry.Call("createReader")
		rec = js.FuncOf(func(this js.Value, args []js.Value) any {
			reader.Call("readEntries", cb)
			return nil
		})
		defer rec.Release()

		rec.Value.Call("call")
		<-ch
	}

	n := len(d.entries) - d.offset

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
		fi := &fileInfo{
			entry: d.entries[d.offset+i],
		}
		if fi.entry.Get("isFile").Bool() {
			fi.file = getFile(fi.entry)
		}
		ents[i] = fs.FileInfoToDirEntry(fi)
	}
	d.offset += n

	return ents, nil
}

type fileInfo struct {
	entry js.Value
	file  js.Value
}

func (f *fileInfo) Name() string {
	return f.entry.Get("name").String()
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
