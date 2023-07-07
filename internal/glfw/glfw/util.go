// SPDX-License-Identifier: BSD-3-Clause
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build !windows

package glfw

// #include <stdlib.h>
import "C"

func glfwbool(b C.int) bool {
	return b == C.int(True)
}

func bytes(origin []byte) (pointer *uint8, free func()) {
	n := len(origin)
	if n == 0 {
		return nil, func() {}
	}

	ptr := C.CBytes(origin)
	return (*uint8)(ptr), func() { C.free(ptr) }
}
