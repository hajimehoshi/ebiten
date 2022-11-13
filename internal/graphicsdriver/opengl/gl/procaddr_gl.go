// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2014 Eric Woroshow

//go:build !android && !darwin && !js && !nintendosdk && !windows && !opengles

package gl

// #cgo pkg-config: gl
//
// #include <stdlib.h>
// #include <GL/glx.h>
//
// static void* getProcAddress(const char* name) {
//   return glXGetProcAddress((const GLubyte *) name);
// }
import "C"

import "unsafe"

func getProcAddress(namea string) unsafe.Pointer {
	cname := C.CString(namea)
	defer C.free(unsafe.Pointer(cname))
	return C.getProcAddress(cname)
}
