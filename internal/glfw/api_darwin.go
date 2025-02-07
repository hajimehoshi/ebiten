// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2009-2016 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2024 The Ebitengine Authors

package glfw

// #include <pthread.h>
import "C"

import (
	"fmt"

	"github.com/ebitengine/purego"
)

type mach_timebase_info_data_t struct {
	numer uint32
	denom uint32
}

var (
	mach_absolute_time    func() uint64
	mach_timebase_info    func(*mach_timebase_info_data_t)
	pthread_key_create    func(key *C.pthread_key_t, destructor uintptr) int32
	pthread_key_delete    func(key C.pthread_key_t) int32
	pthread_getspecific   func(key C.pthread_key_t) uintptr
	pthread_setspecific   func(key C.pthread_key_t, value uintptr) int32
	pthread_mutex_init    func(mutex *C.pthread_mutex_t, attr *C.pthread_mutexattr_t) int32
	pthread_mutex_destroy func(mutex *C.pthread_mutex_t) int32
	pthread_mutex_lock    func(mutex *C.pthread_mutex_t) int32
	pthread_mutex_unlock  func(mutex *C.pthread_mutex_t) int32
)

// TODO: replace with Go error handling
var _glfwInputError func(code int32, format *C.char)

func init() {
	purego.RegisterLibFunc(&_glfwInputError, purego.RTLD_DEFAULT, "_glfwInputError")

	libSystem, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(fmt.Errorf("glfw: failed to dlopen: %w", err))
	}
	purego.RegisterLibFunc(&mach_absolute_time, libSystem, "mach_absolute_time")
	purego.RegisterLibFunc(&mach_timebase_info, libSystem, "mach_timebase_info")
	purego.RegisterLibFunc(&pthread_key_create, libSystem, "pthread_key_create")
	purego.RegisterLibFunc(&pthread_key_delete, libSystem, "pthread_key_delete")
	purego.RegisterLibFunc(&pthread_getspecific, libSystem, "pthread_getspecific")
	purego.RegisterLibFunc(&pthread_setspecific, libSystem, "pthread_setspecific")
	purego.RegisterLibFunc(&pthread_mutex_init, libSystem, "pthread_mutex_init")
	purego.RegisterLibFunc(&pthread_mutex_destroy, libSystem, "pthread_mutex_destroy")
	purego.RegisterLibFunc(&pthread_mutex_lock, libSystem, "pthread_mutex_lock")
	purego.RegisterLibFunc(&pthread_mutex_unlock, libSystem, "pthread_mutex_unlock")
}
