// Copyright 2026 The Ebitengine Authors
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

//go:build freebsd || linux || netbsd

package glfw

import (
	"runtime"
)

// X protocol and extension constants, with the values from the C headers.
// The names match the C constants for ease of comparison with the C sources.

// lcCType is the LC_CTYPE locale category from locale.h: 0 on Linux (both
// glibc and musl), 2 on the BSDs.
var lcCType = func() int32 {
	if runtime.GOOS == "linux" {
		return 0
	}
	return 2
}()

// X.h
const (
	None    = 0
	Success = 0

	InputOnly = 2

	CWEventMask        = 1 << 11
	PropertyChangeMask = 1 << 22

	AnyPropertyType = Atom(0)
)

// Xatom.h
const (
	XA_ATOM     = Atom(4)
	XA_CARDINAL = Atom(6)
	XA_STRING   = Atom(31)
	XA_WINDOW   = Atom(33)
)

// Xlib.h
const (
	XIMPreeditNothing = 0x0008
	XIMStatusNothing  = 0x0400

	QueuedAfterReading = 1

	XBufferOverflow = -1
	XLookupChars    = 2
	XLookupBoth     = 4
)

// XKB.h
const (
	XkbUseCoreKbd     = 0x0100
	XkbKeyNamesMask   = 1 << 9
	XkbKeyAliasesMask = 1 << 10
	XkbKeyNameLength  = 4
	XkbEventCode      = 0
	XkbStateNotify    = 2
	XkbGroupStateMask = 1 << 4
)

// X.h (continued)
const (
	CurrentTime = Time(0)
	NoSymbol    = KeySym(0)

	KeyPress         = 2
	KeyRelease       = 3
	ButtonPress      = 4
	ButtonRelease    = 5
	MotionNotify     = 6
	EnterNotify      = 7
	LeaveNotify      = 8
	FocusIn          = 9
	FocusOut         = 10
	KeymapNotify     = 11
	Expose           = 12
	VisibilityNotify = 15
	DestroyNotify    = 17
	UnmapNotify      = 18
	MapNotify        = 19
	ReparentNotify   = 21
	ConfigureNotify  = 22
	PropertyNotify   = 28
	SelectionClear   = 29
	SelectionRequest = 30
	SelectionNotify  = 31
	ClientMessage    = 33
	MappingNotify    = 34
	GenericEvent     = 35

	NoEventMask              = 0
	KeyPressMask             = 1 << 0
	KeyReleaseMask           = 1 << 1
	ButtonPressMask          = 1 << 2
	ButtonReleaseMask        = 1 << 3
	EnterWindowMask          = 1 << 4
	LeaveWindowMask          = 1 << 5
	PointerMotionMask        = 1 << 6
	ExposureMask             = 1 << 15
	VisibilityChangeMask     = 1 << 16
	StructureNotifyMask      = 1 << 17
	SubstructureNotifyMask   = 1 << 19
	SubstructureRedirectMask = 1 << 20
	FocusChangeMask          = 1 << 21

	ShiftMask   = 1 << 0
	LockMask    = 1 << 1
	ControlMask = 1 << 2
	Mod1Mask    = 1 << 3
	Mod2Mask    = 1 << 4
	Mod4Mask    = 1 << 6

	Button1 = 1
	Button2 = 2
	Button3 = 3
	Button4 = 4
	Button5 = 5
	Button6 = 6
	Button7 = 7

	GrabModeAsync = 1
	GrabSuccess   = 0

	AllocNone   = 0
	InputOutput = 1

	CWBorderPixel      = 1 << 3
	CWOverrideRedirect = 1 << 9
	CWColormap         = 1 << 13
	CWCursor           = 1 << 14

	StaticGravity = 10

	WithdrawnState = 0
	NormalState    = 1
	IconicState    = 3

	IsViewable = 2

	PropModeReplace = 0
	PropModeAppend  = 2

	PropertyNewValue = 0
	PropertyDelete   = 1

	RevertToParent = 2

	BadWindow = 3

	// Notify modes
	NotifyNormal = 0
	NotifyGrab   = 1
	NotifyUngrab = 2

	// Notify detail
	NotifyInferior = 2

	DontPreferBlanking = 0
	DefaultExposures   = 2

	AnyModifier = 1 << 15
)

// Xutil.h (continued)
const (
	PPosition   = 1 << 2
	PMinSize    = 1 << 4
	PMaxSize    = 1 << 5
	PAspect     = 1 << 7
	PWinGravity = 1 << 9

	StateHint = 1 << 1
)

// shapeconst.h
const (
	ShapeSet      = 0
	ShapeBounding = 0
	ShapeInput    = 2

	ShapeNotifyMask = 1 << 0
	ShapeNotify     = 0
)

// XI2.h
const (
	XIAllMasterDevices = 1
	XI_RawMotion       = 17
)

// cursorfont.h
const (
	XC_crosshair         = 34
	XC_fleur             = 52
	XC_hand2             = 60
	XC_left_ptr          = 68
	XC_sb_h_double_arrow = 108
	XC_sb_v_double_arrow = 116
	XC_xterm             = 152
)

// randr.h
const (
	RROutputChangeNotifyMask = 1 << 2

	RRNotify = 1

	RR_Connected = 0

	RR_Rotate_90  = 2
	RR_Rotate_270 = 8

	RR_Interlace = 0x00000010
)

// Xutil.h
const (
	VisualIDMask     = 0x1
	VisualScreenMask = 0x2
)

// egl.h and eglext.h
const (
	EGL_SUCCESS             = 0x3000
	EGL_NOT_INITIALIZED     = 0x3001
	EGL_BAD_ACCESS          = 0x3002
	EGL_BAD_ALLOC           = 0x3003
	EGL_BAD_ATTRIBUTE       = 0x3004
	EGL_BAD_CONFIG          = 0x3005
	EGL_BAD_CONTEXT         = 0x3006
	EGL_BAD_CURRENT_SURFACE = 0x3007
	EGL_BAD_DISPLAY         = 0x3008
	EGL_BAD_MATCH           = 0x3009
	EGL_BAD_NATIVE_PIXMAP   = 0x300a
	EGL_BAD_NATIVE_WINDOW   = 0x300b
	EGL_BAD_PARAMETER       = 0x300c
	EGL_BAD_SURFACE         = 0x300d
	EGL_CONTEXT_LOST        = 0x300e

	EGL_COLOR_BUFFER_TYPE      = 0x303f
	EGL_RGB_BUFFER             = 0x308e
	EGL_SURFACE_TYPE           = 0x3033
	EGL_WINDOW_BIT             = 0x0004
	EGL_RENDERABLE_TYPE        = 0x3040
	EGL_OPENGL_ES_BIT          = 0x0001
	EGL_OPENGL_ES2_BIT         = 0x0004
	EGL_OPENGL_BIT             = 0x0008
	EGL_ALPHA_SIZE             = 0x3021
	EGL_BLUE_SIZE              = 0x3022
	EGL_GREEN_SIZE             = 0x3023
	EGL_RED_SIZE               = 0x3024
	EGL_DEPTH_SIZE             = 0x3025
	EGL_STENCIL_SIZE           = 0x3026
	EGL_SAMPLES                = 0x3031
	EGL_OPENGL_ES_API          = 0x30a0
	EGL_OPENGL_API             = 0x30a2
	EGL_NONE                   = 0x3038
	EGL_RENDER_BUFFER          = 0x3086
	EGL_SINGLE_BUFFER          = 0x3085
	EGL_EXTENSIONS             = 0x3055
	EGL_CONTEXT_CLIENT_VERSION = 0x3098
	EGL_NATIVE_VISUAL_ID       = 0x302e

	EGL_NO_SURFACE = 0
	EGL_NO_DISPLAY = 0
	EGL_NO_CONTEXT = 0

	EGL_CONTEXT_OPENGL_FORWARD_COMPATIBLE_BIT_KHR      = 0x00000002
	EGL_CONTEXT_OPENGL_CORE_PROFILE_BIT_KHR            = 0x00000001
	EGL_CONTEXT_OPENGL_COMPATIBILITY_PROFILE_BIT_KHR   = 0x00000002
	EGL_CONTEXT_OPENGL_DEBUG_BIT_KHR                   = 0x00000001
	EGL_CONTEXT_OPENGL_RESET_NOTIFICATION_STRATEGY_KHR = 0x31bd
	EGL_NO_RESET_NOTIFICATION_KHR                      = 0x31be
	EGL_LOSE_CONTEXT_ON_RESET_KHR                      = 0x31bf
	EGL_CONTEXT_OPENGL_ROBUST_ACCESS_BIT_KHR           = 0x00000004
	EGL_CONTEXT_MAJOR_VERSION_KHR                      = 0x3098
	EGL_CONTEXT_MINOR_VERSION_KHR                      = 0x30fb
	EGL_CONTEXT_OPENGL_PROFILE_MASK_KHR                = 0x30fd
	EGL_CONTEXT_FLAGS_KHR                              = 0x30fc
	EGL_CONTEXT_OPENGL_NO_ERROR_KHR                    = 0x31b3
	EGL_GL_COLORSPACE_KHR                              = 0x309d
	EGL_GL_COLORSPACE_SRGB_KHR                         = 0x3089
	EGL_CONTEXT_RELEASE_BEHAVIOR_KHR                   = 0x2097
	EGL_CONTEXT_RELEASE_BEHAVIOR_NONE_KHR              = 0
	EGL_CONTEXT_RELEASE_BEHAVIOR_FLUSH_KHR             = 0x2098
)

// glx.h and glxext.h
const (
	GLX_VENDOR                                  = 1
	GLX_RGBA_BIT                                = 0x00000001
	GLX_WINDOW_BIT                              = 0x00000001
	GLX_DRAWABLE_TYPE                           = 0x8010
	GLX_RENDER_TYPE                             = 0x8011
	GLX_RGBA_TYPE                               = 0x8014
	GLX_DOUBLEBUFFER                            = 5
	GLX_STEREO                                  = 6
	GLX_AUX_BUFFERS                             = 7
	GLX_RED_SIZE                                = 8
	GLX_GREEN_SIZE                              = 9
	GLX_BLUE_SIZE                               = 10
	GLX_ALPHA_SIZE                              = 11
	GLX_DEPTH_SIZE                              = 12
	GLX_STENCIL_SIZE                            = 13
	GLX_ACCUM_RED_SIZE                          = 14
	GLX_ACCUM_GREEN_SIZE                        = 15
	GLX_ACCUM_BLUE_SIZE                         = 16
	GLX_ACCUM_ALPHA_SIZE                        = 17
	GLX_SAMPLES                                 = 0x186a1
	GLX_VISUAL_ID                               = 0x800b
	GLX_FRAMEBUFFER_SRGB_CAPABLE_ARB            = 0x20b2
	GLX_CONTEXT_DEBUG_BIT_ARB                   = 0x00000001
	GLX_CONTEXT_COMPATIBILITY_PROFILE_BIT_ARB   = 0x00000002
	GLX_CONTEXT_CORE_PROFILE_BIT_ARB            = 0x00000001
	GLX_CONTEXT_PROFILE_MASK_ARB                = 0x9126
	GLX_CONTEXT_FORWARD_COMPATIBLE_BIT_ARB      = 0x00000002
	GLX_CONTEXT_MAJOR_VERSION_ARB               = 0x2091
	GLX_CONTEXT_MINOR_VERSION_ARB               = 0x2092
	GLX_CONTEXT_FLAGS_ARB                       = 0x2094
	GLX_CONTEXT_ES2_PROFILE_BIT_EXT             = 0x00000004
	GLX_CONTEXT_ROBUST_ACCESS_BIT_ARB           = 0x00000004
	GLX_LOSE_CONTEXT_ON_RESET_ARB               = 0x8252
	GLX_CONTEXT_RESET_NOTIFICATION_STRATEGY_ARB = 0x8256
	GLX_NO_RESET_NOTIFICATION_ARB               = 0x8261
	GLX_CONTEXT_RELEASE_BEHAVIOR_ARB            = 0x2097
	GLX_CONTEXT_RELEASE_BEHAVIOR_NONE_ARB       = 0
	GLX_CONTEXT_RELEASE_BEHAVIOR_FLUSH_ARB      = 0x2098
	GLX_CONTEXT_OPENGL_NO_ERROR_ARB             = 0x31b3

	GLXBadProfileARB = 13
)

// keysymdef.h
const (
	XK_0                = 0x0030
	XK_1                = 0x0031
	XK_2                = 0x0032
	XK_3                = 0x0033
	XK_4                = 0x0034
	XK_5                = 0x0035
	XK_6                = 0x0036
	XK_7                = 0x0037
	XK_8                = 0x0038
	XK_9                = 0x0039
	XK_Alt_L            = 0xffe9
	XK_Alt_R            = 0xffea
	XK_BackSpace        = 0xff08
	XK_Caps_Lock        = 0xffe5
	XK_Control_L        = 0xffe3
	XK_Control_R        = 0xffe4
	XK_Delete           = 0xffff
	XK_Down             = 0xff54
	XK_End              = 0xff57
	XK_Escape           = 0xff1b
	XK_F1               = 0xffbe
	XK_F10              = 0xffc7
	XK_F11              = 0xffc8
	XK_F12              = 0xffc9
	XK_F13              = 0xffca
	XK_F14              = 0xffcb
	XK_F15              = 0xffcc
	XK_F16              = 0xffcd
	XK_F17              = 0xffce
	XK_F18              = 0xffcf
	XK_F19              = 0xffd0
	XK_F2               = 0xffbf
	XK_F20              = 0xffd1
	XK_F21              = 0xffd2
	XK_F22              = 0xffd3
	XK_F23              = 0xffd4
	XK_F24              = 0xffd5
	XK_F25              = 0xffd6
	XK_F3               = 0xffc0
	XK_F4               = 0xffc1
	XK_F5               = 0xffc2
	XK_F6               = 0xffc3
	XK_F7               = 0xffc4
	XK_F8               = 0xffc5
	XK_F9               = 0xffc6
	XK_Home             = 0xff50
	XK_ISO_Level3_Shift = 0xfe03
	XK_Insert           = 0xff63
	XK_KP_0             = 0xffb0
	XK_KP_1             = 0xffb1
	XK_KP_2             = 0xffb2
	XK_KP_3             = 0xffb3
	XK_KP_4             = 0xffb4
	XK_KP_5             = 0xffb5
	XK_KP_6             = 0xffb6
	XK_KP_7             = 0xffb7
	XK_KP_8             = 0xffb8
	XK_KP_9             = 0xffb9
	XK_KP_Add           = 0xffab
	XK_KP_Decimal       = 0xffae
	XK_KP_Delete        = 0xff9f
	XK_KP_Divide        = 0xffaf
	XK_KP_Down          = 0xff99
	XK_KP_End           = 0xff9c
	XK_KP_Enter         = 0xff8d
	XK_KP_Equal         = 0xffbd
	XK_KP_Home          = 0xff95
	XK_KP_Insert        = 0xff9e
	XK_KP_Left          = 0xff96
	XK_KP_Multiply      = 0xffaa
	XK_KP_Page_Down     = 0xff9b
	XK_KP_Page_Up       = 0xff9a
	XK_KP_Right         = 0xff98
	XK_KP_Separator     = 0xffac
	XK_KP_Subtract      = 0xffad
	XK_KP_Up            = 0xff97
	XK_Left             = 0xff51
	XK_Menu             = 0xff67
	XK_Meta_L           = 0xffe7
	XK_Meta_R           = 0xffe8
	XK_Mode_switch      = 0xff7e
	XK_Num_Lock         = 0xff7f
	XK_Page_Down        = 0xff56
	XK_Page_Up          = 0xff55
	XK_Pause            = 0xff13
	XK_Print            = 0xff61
	XK_Return           = 0xff0d
	XK_Right            = 0xff53
	XK_Scroll_Lock      = 0xff14
	XK_Shift_L          = 0xffe1
	XK_Shift_R          = 0xffe2
	XK_Super_L          = 0xffeb
	XK_Super_R          = 0xffec
	XK_Tab              = 0xff09
	XK_Up               = 0xff52
	XK_a                = 0x0061
	XK_apostrophe       = 0x0027
	XK_b                = 0x0062
	XK_backslash        = 0x005c
	XK_bracketleft      = 0x005b
	XK_bracketright     = 0x005d
	XK_c                = 0x0063
	XK_comma            = 0x002c
	XK_d                = 0x0064
	XK_e                = 0x0065
	XK_equal            = 0x003d
	XK_f                = 0x0066
	XK_g                = 0x0067
	XK_grave            = 0x0060
	XK_h                = 0x0068
	XK_i                = 0x0069
	XK_j                = 0x006a
	XK_k                = 0x006b
	XK_l                = 0x006c
	XK_less             = 0x003c
	XK_m                = 0x006d
	XK_minus            = 0x002d
	XK_n                = 0x006e
	XK_o                = 0x006f
	XK_p                = 0x0070
	XK_period           = 0x002e
	XK_q                = 0x0071
	XK_r                = 0x0072
	XK_s                = 0x0073
	XK_semicolon        = 0x003b
	XK_slash            = 0x002f
	XK_space            = 0x0020
	XK_t                = 0x0074
	XK_u                = 0x0075
	XK_v                = 0x0076
	XK_w                = 0x0077
	XK_x                = 0x0078
	XK_y                = 0x0079
	XK_z                = 0x007a
)
