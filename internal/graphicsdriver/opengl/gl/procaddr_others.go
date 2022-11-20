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
// static void* libGL() {
//   static void* so;
//   if (!so) {
//     so = dlopen("libGL.so", RTLD_LAZY | RTLD_GLOBAL);
//   }
//   return so;
// }
//
// static void* libGLES() {
//   static void* so;
//   if (!so) {
//     so = dlopen("libGLESv2.so", RTLD_LAZY | RTLD_GLOBAL);
//   }
//   return so;
// }
//
// static void* getProcAddressGL(const char* name) {
//   static void*(*glXGetProcAddress)(const char*);
//   if (!glXGetProcAddress) {
//     glXGetProcAddress = dlsym(libGL(), "glXGetProcAddress");
//     if (!glXGetProcAddress) {
//       glXGetProcAddress = dlsym(libGL(), "glXGetProcAddressARB");
//     }
//   }
//   return glXGetProcAddress(name);
// }
//
// static void* getProcAddressGLES(const char* name) {
//   return dlsym(libGLES(), name);
// }
import "C"

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"unsafe"
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

	// Try OpenGL first. OpenGL is preferrable as this doesn't cause context losts.
	if !preferES {
		if C.libGL() != nil {
			return nil
		}
	}

	// Try OpenGL ES.
	if C.libGLES() != nil {
		c.isES = true
		return nil
	}

	return fmt.Errorf("gl: failed to load libGL.so and libGLESv2.so")
}

func (c *defaultContext) getProcAddress(name string) unsafe.Pointer {
	if c.isES {
		return getProcAddressGLES(name)
	}
	return getProcAddressGL(name)
}

func getProcAddressGL(name string) unsafe.Pointer {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return C.getProcAddressGL(cname)
}

func getProcAddressGLES(name string) unsafe.Pointer {
	name = strings.TrimSuffix(name, "EXT")
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return C.getProcAddressGLES(cname)
}
