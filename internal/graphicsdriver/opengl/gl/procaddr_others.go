// Copyright 2022 The Ebitengine Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !darwin && !js && !nintendosdk && !windows

package gl

// #cgo LDFLAGS: -ldl
//
// #include <dlfcn.h>
// #include <stdlib.h>
//
// static void* getProcAddressGL(void* libGL, const char* name) {
//   static void*(*glXGetProcAddress)(const char*);
//   if (!glXGetProcAddress) {
//     glXGetProcAddress = dlsym(libGL, "glXGetProcAddress");
//     if (!glXGetProcAddress) {
//       glXGetProcAddress = dlsym(libGL, "glXGetProcAddressARB");
//     }
//   }
//   return glXGetProcAddress(name);
// }
//
// static void* getProcAddressGLES(void* libGLES, const char* name) {
//   return dlsym(libGLES, name);
// }
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"unsafe"
)

var (
	libGL   unsafe.Pointer
	libGLES unsafe.Pointer
)

func (c *defaultContext) init() error {
	var preferES bool
	if runtime.GOOS == "android" {
		preferES = true
	}
	if !preferES {
		for _, t := range strings.Split(os.Getenv("EBITENGINE_OPENGL"), ",") {
			switch strings.TrimSpace(t) {
			case "es":
				preferES = true
				break
			}
		}
	}

	// Try OpenGL first. OpenGL is preferable as this doesn't cause context losses.
	if !preferES {
		// Usually libGL.so or libGL.so.1 is used. libGL.so.2 might exist only on NetBSD.
		for _, name := range []string{"libGL.so", "libGL.so.2", "libGL.so.1", "libGL.so.0"} {
			cname := C.CString(name)
			lib := C.dlopen(cname, C.RTLD_LAZY|C.RTLD_GLOBAL)
			C.free(unsafe.Pointer(cname))
			if lib != nil {
				libGL = lib
				return nil
			}
		}
	}

	// Try OpenGL ES.
	for _, name := range []string{"libGLESv2.so", "libGLESv2.so.2", "libGLESv2.so.1", "libGLESv2.so.0"} {
		cname := C.CString(name)
		lib := C.dlopen(cname, C.RTLD_LAZY|C.RTLD_GLOBAL)
		C.free(unsafe.Pointer(cname))
		if lib != nil {
			libGLES = lib
			c.isES = true
			return nil
		}
	}

	return fmt.Errorf("gl: failed to load libGL.so and libGLESv2.so")
}

func (c *defaultContext) getProcAddress(name string) (uintptr, error) {
	if c.isES {
		return getProcAddressGLES(name), nil
	}
	return getProcAddressGL(name), nil
}

func getProcAddressGL(name string) uintptr {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return uintptr(C.getProcAddressGL(libGL, cname))
}

func getProcAddressGLES(name string) uintptr {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return uintptr(C.getProcAddressGLES(libGLES, cname))
}
