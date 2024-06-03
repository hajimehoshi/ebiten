// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2017 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2024 The Ebitengine Authors

package glfw

// #include "internal_unix.h"
import "C"

import "unsafe"

// TODO: make these methods on GLFWtls like on windows
// TODO: use uintptr instead of unsafe.Pointer once matching C API is no longer needed

//export _glfwPlatformCreateTls
func _glfwPlatformCreateTls(tls *C._GLFWtls) C.GLFWbool {
	if tls.posix.allocated != False {
		panic("glfw: TLS must not be allocated")
	}
	if pthread_key_create(&tls.posix.key, 0) != 0 {
		errstr := C.CString("POSIX: Failed to create context TLS")
		defer C.free(unsafe.Pointer(errstr))
		_glfwInputError(int32(PlatformError), errstr)
		return False
	}
	tls.posix.allocated = True
	return True
}

//export _glfwPlatformDestroyTls
func _glfwPlatformDestroyTls(tls *C._GLFWtls) {
	if tls.posix.allocated != 0 {
		pthread_key_delete(tls.posix.key)
	}
	*tls = C._GLFWtls{}
}

//export _glfwPlatformGetTls
func _glfwPlatformGetTls(tls *C._GLFWtls) unsafe.Pointer {
	if tls.posix.allocated != True {
		panic("glfw: TLS must be allocated")
	}
	var p = pthread_getspecific(tls.posix.key)
	return *(*unsafe.Pointer)(unsafe.Pointer(&p)) // TODO: replace with uintptr
}

//export _glfwPlatformSetTls
func _glfwPlatformSetTls(tls *C._GLFWtls, value unsafe.Pointer) {
	if tls.posix.allocated != True {
		panic("glfw: TLS must be allocated")
	}
	pthread_setspecific(tls.posix.key, uintptr(value))
}

//export _glfwPlatformCreateMutex
func _glfwPlatformCreateMutex(mutex *C._GLFWmutex) C.GLFWbool {
	if mutex.posix.allocated != False {
		panic("glfw: mutex must not be allocated")
	}
	if pthread_mutex_init(&mutex.posix.handle, nil) != 0 {
		errstr := C.CString("POSIX: Failed to create mutex")
		defer C.free(unsafe.Pointer(errstr))
		_glfwInputError(int32(PlatformError), errstr)
		return False
	}
	mutex.posix.allocated = True
	return True
}

//export _glfwPlatformDestroyMutex
func _glfwPlatformDestroyMutex(mutex *C._GLFWmutex) {
	if mutex.posix.allocated != 0 {
		pthread_mutex_destroy(&mutex.posix.handle)
	}
	*mutex = C._GLFWmutex{}
}

//export _glfwPlatformLockMutex
func _glfwPlatformLockMutex(mutex *C._GLFWmutex) {
	if mutex.posix.allocated != True {
		panic("glfw: mutex must be allocated")
	}
	pthread_mutex_lock(&mutex.posix.handle)
}

//export _glfwPlatformUnlockMutex
func _glfwPlatformUnlockMutex(mutex *C._GLFWmutex) {
	if mutex.posix.allocated != True {
		panic("glfw: mutex must be allocated")
	}
	pthread_mutex_unlock(&mutex.posix.handle)
}
