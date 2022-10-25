package glfwwin

import (
	"github.com/ebitengine/purego"
)

var (
	libSystem = purego.Dlopen("libSystem.dylib", purego.RTLD_GLOBAL)
)

type pthread_key uint32

var pthread_key_create func(key *pthread_key, destructor uintptr) int32

var pthread_getspecific func(key pthread_key) uintptr

var pthread_key_delete func(key pthread_key) int32

func init() {
	purego.RegisterLibFunc(&pthread_getspecific, libSystem, "pthread_getspecific")
	purego.RegisterLibFunc(&pthread_key_create, libSystem, "pthread_key_create")
	purego.RegisterLibFunc(&pthread_key_delete, libSystem, "pthread_key_delete")
}
