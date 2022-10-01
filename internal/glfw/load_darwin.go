package glfw

import (
	_ "embed"
	"fmt"
	"os"
	"path"
	"unsafe"

	"github.com/ebitengine/purego"
)

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
	if _, ok := d.procs[name]; !ok {
		d.procs[name] = purego.Dlsym(d.lib, name)
	}
	ret, _, _ := purego.SyscallN(d.procs[name], args...)
	return ret
}

func (d dylib) unload() error {
	if purego.Dlclose(d.lib) {
		return fmt.Errorf("glfw: %s", purego.Dlerror())
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
	dir, err := os.UserCacheDir()
	if err != nil {
		panic(fmt.Errorf("glfw: %w", err))
	}
	filePath := path.Join(dir, "ebitengine")
	err = os.MkdirAll(filePath, 0750)
	if err != nil {
		panic(fmt.Errorf("glfw: %w", err))
	}
	filePath = path.Join(filePath, "libglfw.3.3.8.dylib")
	file, err := os.Create(filePath)
	if err != nil {
		panic(fmt.Errorf("glfw: %w", err))
	}
	defer file.Close()
	_, err = file.Write(library)
	if err != nil {
		panic(fmt.Errorf("glfw: %w", err))
	}

	lib := purego.Dlopen(filePath, purego.RTLD_GLOBAL)
	if lib == 0 {
		panic(fmt.Errorf("glfw: %s", purego.Dlerror()))
	}
	libglfw = &dylib{
		lib: lib,
	}
	libglfw.call("glfwSetErrorCallback", purego.NewCallback(goGLFWErrorCallback))
}
