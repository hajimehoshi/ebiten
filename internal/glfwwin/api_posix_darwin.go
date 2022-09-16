package glfwwin

import (
	"github.com/ebitengine/purego"
	"unsafe"
)

var (
	libSystem = purego.Dlopen("libSystem.dylib", purego.RTLD_GLOBAL)

	proc_pthread_key_create = purego.Dlsym(libSystem, "pthread_key_create")
)

type pthread_key uint32

func pthread_key_create(key *pthread_key, destructor uintptr) int32 {
	r, _, _ := purego.SyscallN(proc_pthread_key_create, uintptr(unsafe.Pointer(key)), destructor)
	return int32(r)
}
