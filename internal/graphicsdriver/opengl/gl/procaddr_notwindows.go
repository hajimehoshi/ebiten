// SPDX-License-Identifier: MIT

// +build !js
// +build !windows

// This file implements GlowGetProcAddress for every supported platform. The
// correct version is chosen automatically based on build tags:
//
// darwin: CGL
// linux freebsd: GLX
//
// Use of EGL instead of the platform's default (listed above) is made possible
// via the "egl" build tag.
//
// It is also possible to install your own function outside this package for
// retrieving OpenGL function pointers, to do this see InitWithProcAddrFunc.

package gl

/*
#cgo darwin CFLAGS: -DTAG_DARWIN
#cgo darwin LDFLAGS: -framework OpenGL
#cgo linux freebsd CFLAGS: -DTAG_POSIX
#cgo linux freebsd pkg-config: gl
#cgo egl CFLAGS: -DTAG_EGL
#cgo egl pkg-config: egl
// Check the EGL tag first as it takes priority over the platform's default
// configuration of WGL/GLX/CGL.
#if defined(TAG_EGL)
	#include <stdlib.h>
	#include <EGL/egl.h>
	static void* GlowGetProcAddress_gl21(const char* name) {
		return eglGetProcAddress(name);
	}
#elif defined(TAG_DARWIN)
	#include <stdlib.h>
	#include <dlfcn.h>
	static void* GlowGetProcAddress_gl21(const char* name) {
		return dlsym(RTLD_DEFAULT, name);
	}
#elif defined(TAG_POSIX)
	#include <stdlib.h>
	#include <GL/glx.h>
	static void* GlowGetProcAddress_gl21(const char* name) {
		return glXGetProcAddress((const GLubyte *) name);
	}
#endif
*/
import "C"
import "unsafe"

func getProcAddress(namea string) unsafe.Pointer {
	cname := C.CString(namea)
	defer C.free(unsafe.Pointer(cname))
	return C.GlowGetProcAddress_gl21(cname)
}
