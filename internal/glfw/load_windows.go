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
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

type dll struct {
	d     *windows.LazyDLL
	procs map[string]*windows.LazyProc
}

func (d *dll) call(name string, args ...uintptr) uintptr {
	if d.procs == nil {
		d.procs = map[string]*windows.LazyProc{}
	}
	if _, ok := d.procs[name]; !ok {
		d.procs[name] = d.d.NewProc(name)
	}
	// It looks like there is no way to handle Windows errors correctly.
	r, _, _ := d.procs[name].Call(args...)
	return r
}

func (d *dll) unload() error {
	if err := windows.FreeLibrary(windows.Handle(d.d.Handle())); err != nil {
		return err
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

func fetchError() *glfwError {
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
		if err.code == c {
			return err
		}
	}
	switch err.code {
	case PlatformError:
		// TODO: Should we log this?
		return nil
	case NotInitialized, NoCurrentContext, InvalidEnum, InvalidValue, OutOfMemory:
		panic(err)
	default:
		panic(fmt.Sprintf("glfw: uncaught error: %s", err.Error()))
	}
	return err
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

var glfwDLL *dll

func init() {
	dll, err := loadDLL()
	if err != nil {
		panic(err)
	}
	glfwDLL = dll

	glfwDLL.call("glfwSetErrorCallback", windows.NewCallbackCDecl(goGLFWErrorCallback))
}
