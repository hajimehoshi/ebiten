// Copyright 2024 The Ebitengine Authors
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

package gl

// Unfortunately, PureGo's dlopen didn't work well on some Android devices (#3052).
// Use Cgo instead until PureGo is fixed.

// #include <dlfcn.h>
// #include <stdlib.h>
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

var (
	libGLES unsafe.Pointer
)

func (c *defaultContext) init() error {
	// TODO: Use multiple %w-s as of Go 1.20.
	var errors []string

	// Try OpenGL ES.
	for _, name := range []string{"libGLESv2.so", "libGLESv2.so.2", "libGLESv2.so.1", "libGLESv2.so.0"} {
		cname := C.CString(name)
		defer C.free(unsafe.Pointer(cname))
		lib := C.dlopen(cname, C.RTLD_LAZY|C.RTLD_GLOBAL)
		if lib != nil {
			libGLES = lib
			c.isES = true
			return nil
		}
		if cerr := C.dlerror(); cerr != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, C.GoString(cerr)))
		}
	}

	return fmt.Errorf("gl: failed to load libGLESv2.so: %s", strings.Join(errors, ", "))
}

func (c *defaultContext) getProcAddress(name string) (uintptr, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	addr := C.dlsym(libGLES, cname)
	if addr == nil {
		if cerr := C.dlerror(); cerr != nil {
			return 0, fmt.Errorf("gl: failed to load %s: %v", name, C.GoString(cerr))
		}
		return 0, fmt.Errorf("gl: failed to load %s", name)
	}
	return uintptr(addr), nil
}
