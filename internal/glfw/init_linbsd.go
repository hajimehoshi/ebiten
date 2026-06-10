// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2018 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

//go:build freebsd || linux || netbsd || openbsd

package glfw

// #include <stdlib.h>
import "C"

import (
	"strconv"
	"strings"
	"unsafe"
)

// parseUriList splits and translates a text/uri-list into separate file
// paths.
func parseUriList(text string) []string {
	const prefix = "file://"

	var paths []string

	for _, line := range strings.FieldsFunc(text, func(r rune) bool {
		return r == '\r' || r == '\n'
	}) {
		if strings.HasPrefix(line, "#") {
			continue
		}

		if strings.HasPrefix(line, prefix) {
			line = line[len(prefix):]
			// TODO: Validate hostname
			slash := strings.IndexByte(line, '/')
			if slash < 0 {
				// A file URI without a path is malformed.
				continue
			}
			line = line[slash:]
		}

		var path strings.Builder
		for i := 0; i < len(line); i++ {
			if line[i] == '%' && i+2 < len(line) {
				if b, err := strconv.ParseUint(line[i+1:i+3], 16, 8); err == nil {
					path.WriteByte(byte(b))
					i += 2
					continue
				}
				// An invalid escape sequence is kept as-is.
			}
			path.WriteByte(line[i])
		}
		paths = append(paths, path.String())
	}

	return paths
}

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
