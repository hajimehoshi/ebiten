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

package glfw

import (
	_ "embed"
	"fmt"
	"os"
	"path"
	"unsafe"

	"github.com/ebitengine/purego"
)

//go:generate go run gen.go

//go:embed libglfw.3.3.8.dylib
var library []byte

var libglfw *dylib

type dylib struct {
	lib   uintptr
	procs map[string]uintptr
}

func (d dylib) call(name string, args ...uintptr) uintptr {
	if d.procs == nil {
		d.procs = map[string]uintptr{}
	}
	var err error
	if _, ok := d.procs[name]; !ok {
		d.procs[name], err = purego.Dlsym(d.lib, name)
		if err != nil {
			panic(err)
		}
	}
	ret, _, _ := purego.SyscallN(d.procs[name], args...)
	return ret
}

func (d dylib) unload() error {
	if err := purego.Dlclose(d.lib); err != nil {
		return fmt.Errorf("glfw: %s", err)
	}
	return nil
}

func bytePtrToString(ptr *byte) string {
	var bs []byte
	for i := uintptr(0); ; i++ {
		b := *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(ptr)) + i))
		if b == 0 {
			break
		}
		bs = append(bs, b)
	}
	return string(bs)
}

type glfwError struct {
	code ErrorCode
	desc string
}

func (e *glfwError) Error() string {
	return fmt.Sprintf("glfw: %s: %s", e.code.String(), e.desc)
}

var lastErr = make(chan *glfwError, 1)

func fetchError() error {
	select {
	case err := <-lastErr:
		return err
	default:
		return nil
	}
}

func panicError() {
	if err := acceptError(); err != nil {
		panic(err)
	}
}

func flushErrors() {
	if err := fetchError(); err != nil {
		panic(fmt.Sprintf("glfw: uncaught error: %s", err.Error()))
	}
}

func acceptError(codes ...ErrorCode) error {
	err := fetchError()
	if err == nil {
		return nil
	}
	for _, c := range codes {
		if err.(*glfwError).code == c {
			return err
		}
	}
	switch err.(*glfwError).code {
	case PlatformError:
		// TODO: Should we log this?
		return nil
	case NotInitialized, NoCurrentContext, InvalidEnum, InvalidValue, OutOfMemory:
		panic(err)
	default:
		panic(fmt.Sprintf("glfw: uncaught error: %s", err.Error()))
	}
}

func goGLFWErrorCallback(code uintptr, desc *byte) uintptr {
	flushErrors()
	err := &glfwError{
		code: ErrorCode(code),
		desc: bytePtrToString(desc),
	}
	select {
	case lastErr <- err:
	default:
		panic(fmt.Sprintf("glfw: uncaught error: %s", err.Error()))
	}
	return 0
}

func init() {
	filePath, err := os.MkdirTemp("", "ebitengine-glfw-*")
	if err != nil {
		panic(fmt.Errorf("glfw: %w", err))
	}
	filePath = path.Join(filePath, "libglfw.3.3.dylib")
	file, err := os.Create(filePath)
	if err != nil {
		panic(fmt.Errorf("glfw: %w", err))
	}
	defer func(file *os.File) {
		if err := file.Close(); err != nil {
			panic(fmt.Errorf("glfw: %w", err))
		}
		if err := os.Remove(filePath); err != nil {
			panic(fmt.Errorf("glfw: %w", err))
		}
	}(file)

	if _, err := file.Write(library); err != nil {
		panic(fmt.Errorf("glfw: %w", err))
	}
	lib, err := purego.Dlopen(filePath, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("glfw: %s", err))
	}
	libglfw = &dylib{
		lib: lib,
	}
	libglfw.call("glfwSetErrorCallback", purego.NewCallback(goGLFWErrorCallback))
}
