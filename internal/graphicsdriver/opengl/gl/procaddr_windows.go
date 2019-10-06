// SPDX-License-Identifier: MIT

package gl

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	opengl32          = windows.NewLazySystemDLL("opengl32")
	wglGetProcAddress = opengl32.NewProc("wglGetProcAddress")
)

func getProcAddress(namea string) uintptr {
	cname, err := windows.BytePtrFromString(namea)
	if err != nil {
		panic(err)
	}
	if r, _, _ := wglGetProcAddress.Call(uintptr(unsafe.Pointer(cname))); r != 0 {
		return r
	}
	p := opengl32.NewProc(namea)
	if err := p.Find(); err != nil {
		// The proc is not found.
		return 0
	}
	return p.Addr()
}
