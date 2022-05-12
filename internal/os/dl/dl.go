package dl

import (
	"github.com/hajimehoshi/ebiten/v2/internal/os/syscall"
	"unsafe"
)

const RTLD_DEFAULT = ^uintptr(1)

func Sym(handle uintptr, name *byte) uintptr {
	ret, _, _ := syscall.SyscallX(dlsymABI0, handle, uintptr(unsafe.Pointer(name)), 0)
	return ret
}
