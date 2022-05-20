// SPDX-License-Identifier: MIT

package gl

import (
	"github.com/ebiten/purego"
)

// this comment tells the linker to link to the OpenGL framework at runtime
//go:cgo_import_dynamic _ _ "/System/Library/Frameworks/OpenGL.framework/Versions/Current/OpenGL"

func getProcAddress(name string) uintptr {
	return purego.Dlsym(purego.RTLD_DEFAULT, name)
}
