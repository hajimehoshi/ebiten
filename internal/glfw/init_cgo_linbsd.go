// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

//go:build freebsd || linux || netbsd || openbsd

package glfw

// #include <stdlib.h>
import "C"

import (
	"unsafe"
)

//export _glfwParseUriList
func _glfwParseUriList(text *C.char, count *C.int) **C.char {
	paths := parseUriList(C.GoString(text))
	*count = C.int(len(paths))
	if len(paths) == 0 {
		return nil
	}

	// The caller frees each entry and the array itself with free(), so
	// both must be C heap allocations.
	array := (**C.char)(C.malloc(C.size_t(len(paths)) * C.size_t(unsafe.Sizeof((*C.char)(nil)))))
	cPaths := unsafe.Slice(array, len(paths))
	for i, path := range paths {
		cPaths[i] = C.CString(path)
	}
	return array
}
