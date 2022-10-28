package glfwwin

import (
	"reflect"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

var sel_init = objc.RegisterName("init")

func GoString(p uintptr) string {
	if p == 0 {
		return ""
	}
	var length int
	for {
		// use unsafe.Add once we reach 1.17
		if *(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(length))) == '\x00' {
			break
		}
		length++
	}
	// use unsafe.Slice once we reach 1.17
	s := make([]byte, length)
	var src []byte
	h := (*reflect.SliceHeader)(unsafe.Pointer(&src))
	h.Data = uintptr(unsafe.Pointer(p))
	h.Len = length
	h.Cap = length
	copy(s, src)
	return string(s)
}

var (
	_CoreGraphics                = purego.Dlopen("CoreGraphics.framework/CoreGraphics", purego.RTLD_GLOBAL)
	procCGGetOnlineDisplayList   = purego.Dlsym(_CoreGraphics, "CGGetOnlineDisplayList")
	procCGDisplayIsAsleep        = purego.Dlsym(_CoreGraphics, "CGDisplayIsAsleep")
	procCGDisplayUnitNumber      = purego.Dlsym(_CoreGraphics, "CGDisplayUnitNumber")
	procCGDisplayCopyDisplayMode = purego.Dlsym(_CoreGraphics, "CGDisplayCopyDisplayMode")
	procCGDisplayModeRelease     = purego.Dlsym(_CoreGraphics, "CGDisplayModeRelease")
	procCGDisplayModeGetWidth    = purego.Dlsym(_CoreGraphics, "CGDisplayModeGetWidth")
	procCGDisplayModeGetHeight   = purego.Dlsym(_CoreGraphics, "CGDisplayModeGetHeight")
)

type _CGDirectDisplayID uint32
type _CGDisplayModeRef uintptr
type _CGError uint32

func _CGGetOnlineDisplayList(maxDisplays uint32, onlineDisplays *_CGDirectDisplayID, displayCount *uint32) _CGError {
	ret, _, _ := purego.SyscallN(procCGGetOnlineDisplayList, uintptr(maxDisplays), uintptr(unsafe.Pointer(onlineDisplays)), uintptr(unsafe.Pointer(displayCount)))
	return _CGError(ret)
}

func _CGDisplayIsAsleep(display _CGDirectDisplayID) bool {
	ret, _, _ := purego.SyscallN(procCGDisplayIsAsleep, uintptr(display))
	return ret != 0
}

func _CGDisplayUnitNumber(display _CGDirectDisplayID) uint32 {
	ret, _, _ := purego.SyscallN(procCGDisplayUnitNumber, uintptr(display))
	return uint32(ret)
}

var _CGDisplayBounds func(display _CGDirectDisplayID) cocoa.CGRect

func _CGDisplayCopyDisplayMode(display _CGDirectDisplayID) _CGDisplayModeRef {
	ret, _, _ := purego.SyscallN(procCGDisplayCopyDisplayMode, uintptr(display))
	return _CGDisplayModeRef(ret)
}

func _CGDisplayModeRelease(mode _CGDisplayModeRef) {
	purego.SyscallN(procCGDisplayModeRelease, uintptr(mode))
}

func _CGDisplayModeGetWidth(mode _CGDisplayModeRef) uintptr {
	ret, _, _ := purego.SyscallN(procCGDisplayModeGetWidth, uintptr(mode))
	return ret
}

func _CGDisplayModeGetHeight(mode _CGDisplayModeRef) uintptr {
	ret, _, _ := purego.SyscallN(procCGDisplayModeGetHeight, uintptr(mode))
	return ret
}

var _CGDisplayModeGetRefreshRate func(mode _CGDisplayModeRef) float64

func init() {
	purego.RegisterLibFunc(&_CGDisplayModeGetRefreshRate, _CoreGraphics, "CGDisplayModeGetRefreshRate")
	purego.RegisterLibFunc(&_CGDisplayBounds, _CoreGraphics, "CGDisplayBounds")
}
