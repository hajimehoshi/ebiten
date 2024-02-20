// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2009-2016 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2024 The Ebitengine Authors

package glfw

import "C"
import "github.com/ebitengine/purego"

type mach_timebase_info_data_t struct {
	numer uint32
	denom uint32
}

var mach_absolute_time func() uint64
var mach_timebase_info func(*mach_timebase_info_data_t)

func init() {
	libSystem, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		panic(err)
	}
	purego.RegisterLibFunc(&mach_absolute_time, libSystem, "mach_absolute_time")
	purego.RegisterLibFunc(&mach_timebase_info, libSystem, "mach_timebase_info")
}
