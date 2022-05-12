// SPDX-License-Identifier: MIT

// This file implements GlowGetProcAddress for every supported platform. The
// correct version is chosen automatically based on build tags:
//
// darwin: CGL
// linux freebsd openbsd: GLX
//
// Use of EGL instead of the platform's default (listed above) is made possible
// via the "egl" build tag.
//
// It is also possible to install your own function outside this package for
// retrieving OpenGL function pointers, to do this see InitWithProcAddrFunc.

package gl

import (
	"github.com/hajimehoshi/ebiten/v2/internal/os/dl"
	"runtime"
	"strings"
	"unsafe"
)

func _CString(name string) *byte {
	if strings.HasSuffix(name, "\x00") {
		panic(name + "has null suffix")
	}
	var b = make([]byte, len(name)+1)
	copy(b, name)
	return &b[0]
}

func getProcAddress(namea string) unsafe.Pointer {
	cname := _CString(namea)
	defer func() {
		runtime.KeepAlive(cname)
		cname = nil
	}()
	return unsafe.Pointer(dl.Sym(dl.RTLD_DEFAULT, cname))
}
