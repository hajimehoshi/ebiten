// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2009-2016 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2024 The Ebitengine Authors

package glfw

// #include "internal_unix.h"
import "C"

//export _glfwInitTimerNS
func _glfwInitTimerNS() {
	var info mach_timebase_info_data_t
	mach_timebase_info(&info)
	C._glfw.timer.ns.frequency = C.ulonglong(info.denom*1e9) / C.ulonglong(info.numer)
}

//export _glfwPlatformGetTimerValue
func _glfwPlatformGetTimerValue() uint64 {
	return mach_absolute_time()
}

//export _glfwPlatformGetTimerFrequency
func _glfwPlatformGetTimerFrequency() uint64 {
	return uint64(C._glfw.timer.ns.frequency)
}
