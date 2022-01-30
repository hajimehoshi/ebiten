// SPDX-License-Identifier: MIT

package gl

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	opengl32              = windows.NewLazySystemDLL("opengl32")
	procWglGetProcAddress = opengl32.NewProc("wglGetProcAddress")
)

func getProcAddress(namea string) uintptr {
	cname, err := windows.BytePtrFromString(namea)
	if err != nil {
		panic(err)
	}

	r, _, err := procWglGetProcAddress.Call(uintptr(unsafe.Pointer(cname)))
	if r != 0 {
		return r
	}
	if err != nil && err != windows.ERROR_SUCCESS && err != windows.ERROR_PROC_NOT_FOUND {
		panic(fmt.Sprintf("gl: wglGetProcAddress failed: %s", err.Error()))
	}

	p := opengl32.NewProc(namea)
	if err := p.Find(); err != nil {
		// The proc is not found.
		return 0
	}
	return p.Addr()
}
