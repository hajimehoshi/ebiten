// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2014 Eric Woroshow

//go:build !android && !darwin && !js && !windows && !opengles

package gl

/*
#cgo linux freebsd openbsd CFLAGS: -DTAG_POSIX
#cgo linux,!nintendosdk freebsd,!nintendosdk openbsd,!nintendosdk pkg-config: gl
#cgo egl nintendosdk CFLAGS: -DTAG_EGL
#cgo egl,!nintendosdk pkg-config: egl
#cgo nintendosdk LDFLAGS: -Wl,-unresolved-symbols=ignore-all

#if defined(TAG_EGL)
	#include <stdlib.h>
	#include <EGL/egl.h>
	static void* getProcAddress(const char* name) {
		return eglGetProcAddress(name);
	}
#elif defined(TAG_POSIX)
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
