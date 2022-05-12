// SPDX-License-Identifier: MIT

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
