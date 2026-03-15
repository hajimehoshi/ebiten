// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2017 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2024 The Ebitengine Authors

//go:build freebsd || linux || netbsd || openbsd

package glfw

// #include "internal_unix.h"
import "C"

import "time"

//export _glfwInitTimerPOSIX
func _glfwInitTimerPOSIX() {
	// Go's time.Now() always uses a monotonic clock, so
	// the C struct fields are no longer needed.
	// Set them for consistency with any C code that might read them.
	C._glfw.timer.posix.monotonic = True
	C._glfw.timer.posix.frequency = 1000000000
}

//export _glfwPlatformGetTimerValue
func _glfwPlatformGetTimerValue() C.uint64_t {
	return C.uint64_t(time.Now().UnixNano())
}

//export _glfwPlatformGetTimerFrequency
func _glfwPlatformGetTimerFrequency() C.uint64_t {
	return 1000000000
}
