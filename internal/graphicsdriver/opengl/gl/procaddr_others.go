// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2014 Eric Woroshow

//go:build !android && !darwin && !js && !windows && !opengles

package gl

/*
#cgo !nintendosdk pkg-config: gl
#cgo nintendosdk  CFLAGS: -DTAG_NINTENDOSDK
#cgo nintendosdk  LDFLAGS: -Wl,-unresolved-symbols=ignore-all

#if defined(TAG_NINTENDOSDK)
	#include <stdlib.h>
	#include <EGL/egl.h>
	static void* getProcAddress(const char* name) {
		return eglGetProcAddress(name);
	}
#else
	#include <stdlib.h>
	#include <GL/glx.h>
	static void* getProcAddress(const char* name) {
		return glXGetProcAddress((const GLubyte *) name);
	}
#endif
*/
import "C"

import "unsafe"

func getProcAddress(namea string) unsafe.Pointer {
	cname := C.CString(namea)
	defer C.free(unsafe.Pointer(cname))
	return C.getProcAddress(cname)
}
