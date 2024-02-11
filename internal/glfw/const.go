// Copyright 2018 The Ebiten Authors
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

//go:build darwin || freebsd || linux || netbsd || openbsd || windows

package glfw

import (
	"fmt"
)

type (
	Action          int
	ErrorCode       int
	Hint            int
	InputMode       int
	Key             int
	ModifierKey     int
	MouseButton     int
	PeripheralEvent int
	StandardCursor  int
)

const (
	DontCare = -1
	False    = 0
	True     = 1
)

const (
	Release = Action(0)
	Press   = Action(1)
	Repeat  = Action(2)
)

const (
	ModAlt      = ModifierKey(0x0004)
	ModCapsLock = ModifierKey(0x0010)
	ModControl  = ModifierKey(0x0002)
	ModNumLock  = ModifierKey(0x0020)
	ModShift    = ModifierKey(0x0001)
	ModSuper    = ModifierKey(0x0008)
)

const (
	MouseButton1      = MouseButton(0)
	MouseButton2      = MouseButton(1)
	MouseButton3      = MouseButton(2)
	MouseButton4      = MouseButton(3)
	MouseButton5      = MouseButton(4)
	MouseButton6      = MouseButton(5)
	MouseButton7      = MouseButton(6)
	MouseButton8      = MouseButton(7)
	MouseButtonLast   = MouseButton8
	MouseButtonLeft   = MouseButton1
	MouseButtonRight  = MouseButton2
	MouseButtonMiddle = MouseButton3
)

const (
	AccumAlphaBits         = Hint(0x0002100A)
	AccumBlueBits          = Hint(0x00021009)
	AccumGreenBits         = Hint(0x00021008)
	AccumRedBits           = Hint(0x00021007)
	AlphaBits              = Hint(0x00021004)
	AutoIconify            = Hint(0x00020006)
	AuxBuffers             = Hint(0x0002100B)
	BlueBits               = Hint(0x00021003)
	CenterCursor           = Hint(0x00020009)
	ClientAPI              = Hint(0x00022001)
	ContextCreationAPI     = Hint(0x0002200B)
	ContextNoError         = Hint(0x0002200A)
	ContextReleaseBehavior = Hint(0x00022009)
	ContextRevision        = Hint(0x00022004)
	ContextRobustness      = Hint(0x00022005)
	ContextVersionMajor    = Hint(0x00022002)
	ContextVersionMinor    = Hint(0x00022003)
	Decorated              = Hint(0x00020005)
	DepthBits              = Hint(0x00021005)
	DoubleBuffer           = Hint(0x00021010)
	Floating               = Hint(0x00020007)
	Focused                = Hint(0x00020001)
	FocusOnShow            = Hint(0x0002000C)
	GreenBits              = Hint(0x00021002)
	Hovered                = Hint(0x0002000B)
	Iconified              = Hint(0x00020002)
	Maximized              = Hint(0x00020008)
	MousePassthrough       = Hint(0x0002000D)
	OpenGLDebugContext     = Hint(0x00022007)
	OpenGLForwardCompat    = Hint(0x00022006)
	OpenGLProfile          = Hint(0x00022008)
	RedBits                = Hint(0x00021001)
	RefreshRate            = Hint(0x0002100F)
	Resizable              = Hint(0x00020003)
	Samples                = Hint(0x0002100D)
	ScaleToMonitor         = Hint(0x0002200C)
	SRGBCapable            = Hint(0x0002100E)
	StencilBits            = Hint(0x00021006)
	Stereo                 = Hint(0x0002100C)
	TransparentFramebuffer = Hint(0x0002000A)
	Visible                = Hint(0x00020004)
	X11ClassName           = Hint(0x00024001)
	X11InstanceName        = Hint(0x00024002)
)

const (
	CursorMode             = InputMode(0x00033001)
	LockKeyMods            = InputMode(0x00033004)
	RawMouseMotion         = InputMode(0x00033005)
	StickyKeysMode         = InputMode(0x00033002)
	StickyMouseButtonsMode = InputMode(0x00033003)
)

const (
	AnyReleaseBehavior   = 0
	CursorDisabled       = 0x00034003
	CursorHidden         = 0x00034002
	CursorNormal         = 0x00034001
	EGLContextAPI        = 0x00036002
	LoseContextOnReset   = 0x00031002
	NativeContextAPI     = 0x00036001
	NoAPI                = 0
	NoResetNotification  = 0x00031001
	NoRobustness         = 0
	OpenGLAPI            = 0x00030001
	OpenGLAnyProfile     = 0
	OpenGLCompatProfile  = 0x00032002
	OpenGLCoreProfile    = 0x00032001
	OpenGLESAPI          = 0x00030002
	OSMesaContextAPI     = 0x00036003
	ReleaseBehaviorFlush = 0x00035001
	ReleaseBehaviorNone  = 0x00035002
)

const (
	NotInitialized     = ErrorCode(0x00010001)
	NoCurrentContext   = ErrorCode(0x00010002)
	InvalidEnum        = ErrorCode(0x00010003)
	InvalidValue       = ErrorCode(0x00010004)
	OutOfMemory        = ErrorCode(0x00010005)
	APIUnavailable     = ErrorCode(0x00010006)
	VersionUnavailable = ErrorCode(0x00010007)
	PlatformError      = ErrorCode(0x00010008)
	FormatUnavailable  = ErrorCode(0x00010009)
	NoWindowContext    = ErrorCode(0x0001000A)
)

func (e ErrorCode) Error() string {
	switch e {
	case NotInitialized:
		return "the GLFW library is not initialized"
	case NoCurrentContext:
		return "there is no current context"
	case InvalidEnum:
		return "invalid argument for enum parameter"
	case InvalidValue:
		return "invalid value for parameter"
	case OutOfMemory:
		return "out of memory"
	case APIUnavailable:
		return "the requested API is unavailable"
	case VersionUnavailable:
		return "the requested API version is unavailable"
	case PlatformError:
		return "a platform-specific error occurred"
	case FormatUnavailable:
		return "the requested format is unavailable"
	case NoWindowContext:
		return "the specified window has no context"
	default:
		return fmt.Sprintf("GLFW error (%d)", e)
	}
}

const (
	Connected    = PeripheralEvent(0x00040001)
	Disconnected = PeripheralEvent(0x00040002)
)

const (
	ArrowCursor     = StandardCursor(0x00036001)
	IBeamCursor     = StandardCursor(0x00036002)
	CrosshairCursor = StandardCursor(0x00036003)
	HandCursor      = StandardCursor(0x00036004)
	HResizeCursor   = StandardCursor(0x00036005)
	VResizeCursor   = StandardCursor(0x00036006)

	// v3.4
	ResizeNWSECursor = StandardCursor(0x00036007)
	ResizeNESWCursor = StandardCursor(0x00036008)
	ResizeAllCursor  = StandardCursor(0x00036009)
	NotAllowedCursor = StandardCursor(0x0003600A)
)
