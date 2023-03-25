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

package glfwwin

import (
	"reflect"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

type NSOpenGLPixelFormatAttribute uint32
type NSOpenGLContextParameter cocoa.NSInteger

const (
	NSOpenGLPFADoubleBuffer       NSOpenGLPixelFormatAttribute = 5
	NSOpenGLPFAAuxBuffers         NSOpenGLPixelFormatAttribute = 7
	NSOpenGLPFAColorSize          NSOpenGLPixelFormatAttribute = 8
	NSOpenGLPFAAlphaSize          NSOpenGLPixelFormatAttribute = 11
	NSOpenGLPFADepthSize          NSOpenGLPixelFormatAttribute = 12
	NSOpenGLPFAStencilSize        NSOpenGLPixelFormatAttribute = 13
	NSOpenGLPFAAccumSize          NSOpenGLPixelFormatAttribute = 14
	NSOpenGLPFASampleBuffers      NSOpenGLPixelFormatAttribute = 55
	NSOpenGLPFAAccelerated        NSOpenGLPixelFormatAttribute = 73
	NSOpenGLPFAClosestPolicy      NSOpenGLPixelFormatAttribute = 74
	NSOpenGLPFAOpenGLProfile      NSOpenGLPixelFormatAttribute = 99
	NSOpenGLProfileVersion4_1Core NSOpenGLPixelFormatAttribute = 0x4100
	NSOpenGLProfileVersion3_2Core NSOpenGLPixelFormatAttribute = 0x3200
)

const (
	NSOpenGLContextParameterSwapInterval NSOpenGLContextParameter = 222
)

var (
	class_NSOpenGLPixelFormat = objc.GetClass("NSOpenGLPixelFormat")
	class_NSOpenGLContext     = objc.GetClass("NSOpenGLContext")
)

type NSOpenGLPixelFormat struct {
	objc.ID
}

func NSOpenGLPixelFormat_alloc() NSOpenGLPixelFormat {
	return NSOpenGLPixelFormat{objc.ID(class_NSOpenGLPixelFormat).Send(objc.RegisterName("alloc"))}
}

func (f NSOpenGLPixelFormat) initWithAttributes(attribs []NSOpenGLPixelFormatAttribute) NSOpenGLPixelFormat {
	if attribs[len(attribs)-1] != 0 {
		panic("glfwwin: attribs must end with null terminator")
	}
	return NSOpenGLPixelFormat{f.Send(objc.RegisterName("initWithAttributes:"), &attribs[0])}
}

type NSOpenGLContext struct {
	objc.ID
}

func NSOpenGLContext_alloc() NSOpenGLContext {
	return NSOpenGLContext{objc.ID(class_NSOpenGLContext).Send(objc.RegisterName("alloc"))}
}

func NSOpenGLContext_clearCurrentContext() {
	objc.ID(class_NSOpenGLContext).Send(objc.RegisterName("clearCurrentContext"))
}

func (c NSOpenGLContext) initWithFormat_shareContext(format NSOpenGLPixelFormat, share NSOpenGLContext) NSOpenGLContext {
	return NSOpenGLContext{c.Send(objc.RegisterName("initWithFormat:shareContext:"), format.ID, share.ID)}
}
func (c NSOpenGLContext) makeCurrentContext() {
	c.Send(objc.RegisterName("makeCurrentContext"))
}

func (c NSOpenGLContext) setValues_forParameter(vals *int, param NSOpenGLContextParameter) {
	c.Send(objc.RegisterName("setValues:forParameter:"), vals, param)
}

func (c NSOpenGLContext) flushBuffer() {
	c.Send(objc.RegisterName("flushBuffer"))
}

var sel_init = objc.RegisterName("init")

// CString converts a go string to *byte that can be passed to C code.
func CString(name string) []byte {
	if strings.HasSuffix(name, "\x00") {
		return *(*[]byte)(unsafe.Pointer(&name))
	}
	var b = make([]byte, len(name)+1)
	copy(b, name)
	return b
}

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
	_CoreGraphics, _             = purego.Dlopen("CoreGraphics.framework/CoreGraphics", purego.RTLD_GLOBAL)
	procCGGetOnlineDisplayList   uintptr
	procCGDisplayIsAsleep        uintptr
	procCGDisplayUnitNumber      uintptr
	procCGDisplayCopyDisplayMode uintptr
	procCGDisplayModeRelease     uintptr
	procCGDisplayModeGetWidth    uintptr
	procCGDisplayModeGetHeight   uintptr
)

func init() {
	procCGGetOnlineDisplayList, _ = purego.Dlsym(_CoreGraphics, "CGGetOnlineDisplayList")
	procCGDisplayIsAsleep, _ = purego.Dlsym(_CoreGraphics, "CGDisplayIsAsleep")
	procCGDisplayUnitNumber, _ = purego.Dlsym(_CoreGraphics, "CGDisplayUnitNumber")
	procCGDisplayCopyDisplayMode, _ = purego.Dlsym(_CoreGraphics, "CGDisplayCopyDisplayMode")
	procCGDisplayModeRelease, _ = purego.Dlsym(_CoreGraphics, "CGDisplayModeRelease")
	procCGDisplayModeGetWidth, _ = purego.Dlsym(_CoreGraphics, "CGDisplayModeGetWidth")
	procCGDisplayModeGetHeight, _ = purego.Dlsym(_CoreGraphics, "CGDisplayModeGetHeight")
}

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
