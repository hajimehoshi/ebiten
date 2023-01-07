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
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unsafe"
)

var (
	libGL   unsafe.Pointer
	libGLES unsafe.Pointer
)

// listLibs returns an appropriate library file paths based on the given library paths and the library name as a prefix.
// Note that the found libraries might not be available e.g. due to architecture mismatches.
func listLibs(libraryPaths []string, libName string) ([]string, error) {
	// LD_LIBRARY_PATH might be empty. Use the original name as a candidate.
	libNames := []string{libName}

	// Look for a library file. In some environments like Steam, a library with the exactly same name might not exist (#2523).
	// For example, libGL.so.1 might exist instead of libGL.so.
	for _, dir := range libraryPaths {
		libs, err := listLibsInDirectory(dir, libName)
		if err != nil {
			return nil, err
		}
		if len(libs) == 0 {
			continue
		}

		// The file names are sorted in the alphabetical order.
		// TODO: What is the best version to use?
		sort.Strings(libs)

		libNames = append(libNames, libs...)
	}

	return libNames, nil
}

// listLibsInDirectory returns library file paths with the given prefix in the directory.
// Note that the found libraries might not be available e.g. due to architecture mismatches.
func listLibsInDirectory(dir string, prefix string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, ent := range ents {
		if ent.IsDir() {
			continue
		}
		if ent.Name() == prefix {
			files = append(files, filepath.Join(dir, ent.Name()))
			continue
		}
		if strings.HasPrefix(ent.Name(), prefix+".") {
			files = append(files, filepath.Join(dir, ent.Name()))
			continue
		}
	}

	return files, nil
}

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

	libraryPaths := filepath.SplitList(os.Getenv("LD_LIBRARY_PATH"))

	// Try OpenGL first. OpenGL is preferrable as this doesn't cause context losts.
	if !preferES {
		names, err := listLibs(libraryPaths, "libGL.so")
		if err != nil {
			return err
		}
		for _, name := range names {
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
	names, err := listLibs(libraryPaths, "libGLESv2.so")
	if err != nil {
		return err
	}
	for _, name := range names {
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

func (c *defaultContext) getProcAddress(name string) unsafe.Pointer {
	if c.isES {
		return getProcAddressGLES(name)
	}
	return getProcAddressGL(name)
}

func getProcAddressGL(name string) unsafe.Pointer {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return C.getProcAddressGL(libGL, cname)
}

func getProcAddressGLES(name string) unsafe.Pointer {
	name = strings.TrimSuffix(name, "EXT")
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	return C.getProcAddressGLES(libGLES, cname)
}
