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

//go:build !android && !darwin && !js && !nintendosdk && !windows && !opengles

package gl

// #cgo LDFLAGS: -ldl
//
// #include <dlfcn.h>
// #include <stdlib.h>
//
// static void* getProcAddress(const char* name) {
//   static void* libGL;
//   if (!libGL) {
//     libGL = dlopen("libGL.so", RTLD_NOW | RTLD_GLOBAL);
//   }
//   static void*(*glXGetProcAddress)(const char*);
//   if (!glXGetProcAddress) {
//     glXGetProcAddress = dlsym(libGL, "glXGetProcAddress");
//     if (!glXGetProcAddress) {
//       glXGetProcAddress = dlsym(libGL, "glXGetProcAddressARB");
//     }
//   }
//   return glXGetProcAddress(name);
// }
import "C"

import "unsafe"

var isES = false

func getProcAddress(namea string) unsafe.Pointer {
	cname := C.CString(namea)
	defer C.free(unsafe.Pointer(cname))
	return C.getProcAddress(cname)
}
