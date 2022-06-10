// SPDX-License-Identifier: MIT

package gl

import (
	"github.com/ebitengine/purego"
)

var opengl = purego.Dlopen("/System/Library/Frameworks/OpenGL.framework/Versions/Current/OpenGL", purego.RTLD_GLOBAL)

func getProcAddress(name string) uintptr {
	return purego.Dlsym(opengl, name)
}
