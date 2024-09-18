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

//go:build (freebsd || linux || netbsd || openbsd) && !nintendosdk && !playstation5

package gl

import (
	"errors"
	"fmt"

	"github.com/ebitengine/purego"
)

var (
	libGL   uintptr
	libGLES uintptr
)

func (c *defaultContext) init() error {
	var errs []error

	// Try OpenGL ES first. Some machines like Android and Raspberry Pi might work only with OpenGL ES.
	for _, name := range []string{"libGLESv2.so", "libGLESv2.so.2", "libGLESv2.so.1", "libGLESv2.so.0"} {
		lib, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			libGLES = lib
			c.isES = true
			return nil
		}
		errs = append(errs, fmt.Errorf("gl: Dlopen failed: name: %s: %w", name, err))
	}

	// Try OpenGL next.
	// Usually libGL.so or libGL.so.1 is used. libGL.so.2 might exist only on NetBSD.
	// TODO: Should "libOpenGL.so.0" [1] and "libGLX.so.0" [2] be added? These were added as of GLFW 3.3.9.
	// [1] https://github.com/glfw/glfw/commit/55aad3c37b67f17279378db52da0a3ab81bbf26d
	// [2] https://github.com/glfw/glfw/commit/c18851f52ec9704eb06464058a600845ec1eada1
	for _, name := range []string{"libGL.so", "libGL.so.2", "libGL.so.1", "libGL.so.0"} {
		lib, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			libGL = lib
			return nil
		}
		errs = append(errs, fmt.Errorf("gl: Dlopen failed: name: %s: %w", name, err))
	}

	errs = append([]error{fmt.Errorf("gl: failed to load libGL.so and libGLESv2.so: ")}, errs...)
	return errors.Join(errs...)
}

func (c *defaultContext) getProcAddress(name string) (uintptr, error) {
	if c.isES {
		return getProcAddressGLES(name)
	}
	return getProcAddressGL(name)
}

var glXGetProcAddress func(name string) uintptr

func getProcAddressGL(name string) (uintptr, error) {
	if glXGetProcAddress == nil {
		if _, err := purego.Dlsym(libGL, "glXGetProcAddress"); err == nil {
			purego.RegisterLibFunc(&glXGetProcAddress, libGL, "glXGetProcAddress")
		} else if _, err := purego.Dlsym(libGL, "glXGetProcAddressARB"); err == nil {
			purego.RegisterLibFunc(&glXGetProcAddress, libGL, "glXGetProcAddressARB")
		}
	}
	if glXGetProcAddress == nil {
		return 0, fmt.Errorf("gl: failed to find glXGetProcAddress or glXGetProcAddressARB in libGL.so")
	}

	return glXGetProcAddress(name), nil
}

func getProcAddressGLES(name string) (uintptr, error) {
	proc, err := purego.Dlsym(libGLES, name)
	if err != nil {
		return 0, err
	}
	return proc, nil
}
