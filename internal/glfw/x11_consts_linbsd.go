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
	_None    = 0
	_Success = 0

	_InputOnly = 2

	_CWEventMask        = 1 << 11
	_PropertyChangeMask = 1 << 22

	_AnyPropertyType = _Atom(0)
)

// Xatom.h
const (
	_XA_ATOM     = _Atom(4)
	_XA_CARDINAL = _Atom(6)
	_XA_STRING   = _Atom(31)
	_XA_WINDOW   = _Atom(33)
)

// Xlib.h
const (
	_XIMPreeditNothing = 0x0008
	_XIMStatusNothing  = 0x0400

	_QueuedAfterReading = 1

	_XBufferOverflow = -1
	_XLookupChars    = 2
	_XLookupBoth     = 4
)

// XKB.h
const (
	_XkbUseCoreKbd     = 0x0100
	_XkbKeyNamesMask   = 1 << 9
	_XkbKeyAliasesMask = 1 << 10
	_XkbKeyNameLength  = 4
	_XkbEventCode      = 0
	_XkbStateNotify    = 2
	_XkbGroupStateMask = 1 << 4
)

// X.h (continued)
const (
	_CurrentTime = _Time(0)
	_NoSymbol    = _KeySym(0)

	_KeyPress         = 2
	_KeyRelease       = 3
	_ButtonPress      = 4
	_ButtonRelease    = 5
	_MotionNotify     = 6
	_EnterNotify      = 7
	_LeaveNotify      = 8
	_FocusIn          = 9
	_FocusOut         = 10
	_KeymapNotify     = 11
	_Expose           = 12
	_VisibilityNotify = 15
	_DestroyNotify    = 17
	_UnmapNotify      = 18
	_MapNotify        = 19
	_ReparentNotify   = 21
	_ConfigureNotify  = 22
	_PropertyNotify   = 28
	_SelectionClear   = 29
	_SelectionRequest = 30
	_SelectionNotify  = 31
	_ClientMessage    = 33
	_MappingNotify    = 34
	_GenericEvent     = 35

	_NoEventMask              = 0
	_KeyPressMask             = 1 << 0
	_KeyReleaseMask           = 1 << 1
	_ButtonPressMask          = 1 << 2
	_ButtonReleaseMask        = 1 << 3
	_EnterWindowMask          = 1 << 4
	_LeaveWindowMask          = 1 << 5
	_PointerMotionMask        = 1 << 6
	_ExposureMask             = 1 << 15
	_VisibilityChangeMask     = 1 << 16
	_StructureNotifyMask      = 1 << 17
	_SubstructureNotifyMask   = 1 << 19
	_SubstructureRedirectMask = 1 << 20
	_FocusChangeMask          = 1 << 21

	_ShiftMask   = 1 << 0
	_LockMask    = 1 << 1
	_ControlMask = 1 << 2
	_Mod1Mask    = 1 << 3
	_Mod2Mask    = 1 << 4
	_Mod4Mask    = 1 << 6

	_Button1 = 1
	_Button2 = 2
	_Button3 = 3
	_Button4 = 4
	_Button5 = 5
	_Button6 = 6
	_Button7 = 7

	_GrabModeAsync = 1
	_GrabSuccess   = 0

	_AllocNone   = 0
	_InputOutput = 1

	_CWBorderPixel      = 1 << 3
	_CWOverrideRedirect = 1 << 9
	_CWColormap         = 1 << 13
	_CWCursor           = 1 << 14

	_StaticGravity = 10

	_WithdrawnState = 0
	_NormalState    = 1
	_IconicState    = 3

	_IsViewable = 2

	_PropModeReplace = 0
	_PropModeAppend  = 2

	_PropertyNewValue = 0
	_PropertyDelete   = 1

	_RevertToParent = 2

	_BadWindow = 3

	// Notify modes
	_NotifyNormal = 0
	_NotifyGrab   = 1
	_NotifyUngrab = 2

	// Notify detail
	_NotifyInferior = 2

	_DontPreferBlanking = 0
	_DefaultExposures   = 2

	_AnyModifier = 1 << 15
)

// Xutil.h (continued)
const (
	_PPosition   = 1 << 2
	_PMinSize    = 1 << 4
	_PMaxSize    = 1 << 5
	_PAspect     = 1 << 7
	_PWinGravity = 1 << 9

	_StateHint = 1 << 1
)

// shapeconst.h
const (
	_ShapeSet      = 0
	_ShapeBounding = 0
	_ShapeInput    = 2

	_ShapeNotifyMask = 1 << 0
	_ShapeNotify     = 0
)

// XI2.h
const (
	_XIAllMasterDevices = 1
	_XI_RawMotion       = 17
)

// cursorfont.h
const (
	_XC_crosshair         = 34
	_XC_fleur             = 52
	_XC_hand2             = 60
	_XC_left_ptr          = 68
	_XC_sb_h_double_arrow = 108
	_XC_sb_v_double_arrow = 116
	_XC_xterm             = 152
)

// randr.h
const (
	_RROutputChangeNotifyMask = 1 << 2

	_RRNotify = 1

	_RR_Connected = 0

	_RR_Rotate_90  = 2
	_RR_Rotate_270 = 8

	_RR_Interlace = 0x00000010
)

// Xutil.h
const (
	_VisualIDMask     = 0x1
	_VisualScreenMask = 0x2
)

// egl.h and eglext.h
const (
	_EGL_SUCCESS             = 0x3000
	_EGL_NOT_INITIALIZED     = 0x3001
	_EGL_BAD_ACCESS          = 0x3002
	_EGL_BAD_ALLOC           = 0x3003
	_EGL_BAD_ATTRIBUTE       = 0x3004
	_EGL_BAD_CONFIG          = 0x3005
	_EGL_BAD_CONTEXT         = 0x3006
	_EGL_BAD_CURRENT_SURFACE = 0x3007
	_EGL_BAD_DISPLAY         = 0x3008
	_EGL_BAD_MATCH           = 0x3009
	_EGL_BAD_NATIVE_PIXMAP   = 0x300a
	_EGL_BAD_NATIVE_WINDOW   = 0x300b
	_EGL_BAD_PARAMETER       = 0x300c
	_EGL_BAD_SURFACE         = 0x300d
	_EGL_CONTEXT_LOST        = 0x300e

	_EGL_COLOR_BUFFER_TYPE      = 0x303f
	_EGL_RGB_BUFFER             = 0x308e
	_EGL_SURFACE_TYPE           = 0x3033
	_EGL_WINDOW_BIT             = 0x0004
	_EGL_RENDERABLE_TYPE        = 0x3040
	_EGL_OPENGL_ES_BIT          = 0x0001
	_EGL_OPENGL_ES2_BIT         = 0x0004
	_EGL_OPENGL_BIT             = 0x0008
	_EGL_ALPHA_SIZE             = 0x3021
	_EGL_BLUE_SIZE              = 0x3022
	_EGL_GREEN_SIZE             = 0x3023
	_EGL_RED_SIZE               = 0x3024
	_EGL_DEPTH_SIZE             = 0x3025
	_EGL_STENCIL_SIZE           = 0x3026
	_EGL_SAMPLES                = 0x3031
	_EGL_OPENGL_ES_API          = 0x30a0
	_EGL_OPENGL_API             = 0x30a2
	_EGL_NONE                   = 0x3038
	_EGL_RENDER_BUFFER          = 0x3086
	_EGL_SINGLE_BUFFER          = 0x3085
	_EGL_EXTENSIONS             = 0x3055
	_EGL_CONTEXT_CLIENT_VERSION = 0x3098
	_EGL_NATIVE_VISUAL_ID       = 0x302e

	_EGL_NO_SURFACE = 0
	_EGL_NO_DISPLAY = 0
	_EGL_NO_CONTEXT = 0

	_EGL_CONTEXT_OPENGL_FORWARD_COMPATIBLE_BIT_KHR      = 0x00000002
	_EGL_CONTEXT_OPENGL_CORE_PROFILE_BIT_KHR            = 0x00000001
	_EGL_CONTEXT_OPENGL_COMPATIBILITY_PROFILE_BIT_KHR   = 0x00000002
	_EGL_CONTEXT_OPENGL_DEBUG_BIT_KHR                   = 0x00000001
	_EGL_CONTEXT_OPENGL_RESET_NOTIFICATION_STRATEGY_KHR = 0x31bd
	_EGL_NO_RESET_NOTIFICATION_KHR                      = 0x31be
	_EGL_LOSE_CONTEXT_ON_RESET_KHR                      = 0x31bf
	_EGL_CONTEXT_OPENGL_ROBUST_ACCESS_BIT_KHR           = 0x00000004
	_EGL_CONTEXT_MAJOR_VERSION_KHR                      = 0x3098
	_EGL_CONTEXT_MINOR_VERSION_KHR                      = 0x30fb
	_EGL_CONTEXT_OPENGL_PROFILE_MASK_KHR                = 0x30fd
	_EGL_CONTEXT_FLAGS_KHR                              = 0x30fc
	_EGL_CONTEXT_OPENGL_NO_ERROR_KHR                    = 0x31b3
	_EGL_GL_COLORSPACE_KHR                              = 0x309d
	_EGL_GL_COLORSPACE_SRGB_KHR                         = 0x3089
	_EGL_CONTEXT_RELEASE_BEHAVIOR_KHR                   = 0x2097
	_EGL_CONTEXT_RELEASE_BEHAVIOR_NONE_KHR              = 0
	_EGL_CONTEXT_RELEASE_BEHAVIOR_FLUSH_KHR             = 0x2098
)

// glx.h and glxext.h
const (
	_GLX_VENDOR                                  = 1
	_GLX_RGBA_BIT                                = 0x00000001
	_GLX_WINDOW_BIT                              = 0x00000001
	_GLX_DRAWABLE_TYPE                           = 0x8010
	_GLX_RENDER_TYPE                             = 0x8011
	_GLX_RGBA_TYPE                               = 0x8014
	_GLX_DOUBLEBUFFER                            = 5
	_GLX_STEREO                                  = 6
	_GLX_AUX_BUFFERS                             = 7
	_GLX_RED_SIZE                                = 8
	_GLX_GREEN_SIZE                              = 9
	_GLX_BLUE_SIZE                               = 10
	_GLX_ALPHA_SIZE                              = 11
	_GLX_DEPTH_SIZE                              = 12
	_GLX_STENCIL_SIZE                            = 13
	_GLX_ACCUM_RED_SIZE                          = 14
	_GLX_ACCUM_GREEN_SIZE                        = 15
	_GLX_ACCUM_BLUE_SIZE                         = 16
	_GLX_ACCUM_ALPHA_SIZE                        = 17
	_GLX_SAMPLES                                 = 0x186a1
	_GLX_VISUAL_ID                               = 0x800b
	_GLX_FRAMEBUFFER_SRGB_CAPABLE_ARB            = 0x20b2
	_GLX_CONTEXT_DEBUG_BIT_ARB                   = 0x00000001
	_GLX_CONTEXT_COMPATIBILITY_PROFILE_BIT_ARB   = 0x00000002
	_GLX_CONTEXT_CORE_PROFILE_BIT_ARB            = 0x00000001
	_GLX_CONTEXT_PROFILE_MASK_ARB                = 0x9126
	_GLX_CONTEXT_FORWARD_COMPATIBLE_BIT_ARB      = 0x00000002
	_GLX_CONTEXT_MAJOR_VERSION_ARB               = 0x2091
	_GLX_CONTEXT_MINOR_VERSION_ARB               = 0x2092
	_GLX_CONTEXT_FLAGS_ARB                       = 0x2094
	_GLX_CONTEXT_ES2_PROFILE_BIT_EXT             = 0x00000004
	_GLX_CONTEXT_ROBUST_ACCESS_BIT_ARB           = 0x00000004
	_GLX_LOSE_CONTEXT_ON_RESET_ARB               = 0x8252
	_GLX_CONTEXT_RESET_NOTIFICATION_STRATEGY_ARB = 0x8256
	_GLX_NO_RESET_NOTIFICATION_ARB               = 0x8261
	_GLX_CONTEXT_RELEASE_BEHAVIOR_ARB            = 0x2097
	_GLX_CONTEXT_RELEASE_BEHAVIOR_NONE_ARB       = 0
	_GLX_CONTEXT_RELEASE_BEHAVIOR_FLUSH_ARB      = 0x2098
	_GLX_CONTEXT_OPENGL_NO_ERROR_ARB             = 0x31b3

	_GLXBadProfileARB = 13
)

// keysymdef.h
const (
	_XK_0                = 0x0030
	_XK_1                = 0x0031
	_XK_2                = 0x0032
	_XK_3                = 0x0033
	_XK_4                = 0x0034
	_XK_5                = 0x0035
	_XK_6                = 0x0036
	_XK_7                = 0x0037
	_XK_8                = 0x0038
	_XK_9                = 0x0039
	_XK_Alt_L            = 0xffe9
	_XK_Alt_R            = 0xffea
	_XK_BackSpace        = 0xff08
	_XK_Caps_Lock        = 0xffe5
	_XK_Control_L        = 0xffe3
	_XK_Control_R        = 0xffe4
	_XK_Delete           = 0xffff
	_XK_Down             = 0xff54
	_XK_End              = 0xff57
	_XK_Escape           = 0xff1b
	_XK_F1               = 0xffbe
	_XK_F10              = 0xffc7
	_XK_F11              = 0xffc8
	_XK_F12              = 0xffc9
	_XK_F13              = 0xffca
	_XK_F14              = 0xffcb
	_XK_F15              = 0xffcc
	_XK_F16              = 0xffcd
	_XK_F17              = 0xffce
	_XK_F18              = 0xffcf
	_XK_F19              = 0xffd0
	_XK_F2               = 0xffbf
	_XK_F20              = 0xffd1
	_XK_F21              = 0xffd2
	_XK_F22              = 0xffd3
	_XK_F23              = 0xffd4
	_XK_F24              = 0xffd5
	_XK_F25              = 0xffd6
	_XK_F3               = 0xffc0
	_XK_F4               = 0xffc1
	_XK_F5               = 0xffc2
	_XK_F6               = 0xffc3
	_XK_F7               = 0xffc4
	_XK_F8               = 0xffc5
	_XK_F9               = 0xffc6
	_XK_Home             = 0xff50
	_XK_ISO_Level3_Shift = 0xfe03
	_XK_Insert           = 0xff63
	_XK_KP_0             = 0xffb0
	_XK_KP_1             = 0xffb1
	_XK_KP_2             = 0xffb2
	_XK_KP_3             = 0xffb3
	_XK_KP_4             = 0xffb4
	_XK_KP_5             = 0xffb5
	_XK_KP_6             = 0xffb6
	_XK_KP_7             = 0xffb7
	_XK_KP_8             = 0xffb8
	_XK_KP_9             = 0xffb9
	_XK_KP_Add           = 0xffab
	_XK_KP_Decimal       = 0xffae
	_XK_KP_Delete        = 0xff9f
	_XK_KP_Divide        = 0xffaf
	_XK_KP_Down          = 0xff99
	_XK_KP_End           = 0xff9c
	_XK_KP_Enter         = 0xff8d
	_XK_KP_Equal         = 0xffbd
	_XK_KP_Home          = 0xff95
	_XK_KP_Insert        = 0xff9e
	_XK_KP_Left          = 0xff96
	_XK_KP_Multiply      = 0xffaa
	_XK_KP_Page_Down     = 0xff9b
	_XK_KP_Page_Up       = 0xff9a
	_XK_KP_Right         = 0xff98
	_XK_KP_Separator     = 0xffac
	_XK_KP_Subtract      = 0xffad
	_XK_KP_Up            = 0xff97
	_XK_Left             = 0xff51
	_XK_Menu             = 0xff67
	_XK_Meta_L           = 0xffe7
	_XK_Meta_R           = 0xffe8
	_XK_Mode_switch      = 0xff7e
	_XK_Num_Lock         = 0xff7f
	_XK_Page_Down        = 0xff56
	_XK_Page_Up          = 0xff55
	_XK_Pause            = 0xff13
	_XK_Print            = 0xff61
	_XK_Return           = 0xff0d
	_XK_Right            = 0xff53
	_XK_Scroll_Lock      = 0xff14
	_XK_Shift_L          = 0xffe1
	_XK_Shift_R          = 0xffe2
	_XK_Super_L          = 0xffeb
	_XK_Super_R          = 0xffec
	_XK_Tab              = 0xff09
	_XK_Up               = 0xff52
	_XK_a                = 0x0061
	_XK_apostrophe       = 0x0027
	_XK_b                = 0x0062
	_XK_backslash        = 0x005c
	_XK_bracketleft      = 0x005b
	_XK_bracketright     = 0x005d
	_XK_c                = 0x0063
	_XK_comma            = 0x002c
	_XK_d                = 0x0064
	_XK_e                = 0x0065
	_XK_equal            = 0x003d
	_XK_f                = 0x0066
	_XK_g                = 0x0067
	_XK_grave            = 0x0060
	_XK_h                = 0x0068
	_XK_i                = 0x0069
	_XK_j                = 0x006a
	_XK_k                = 0x006b
	_XK_l                = 0x006c
	_XK_less             = 0x003c
	_XK_m                = 0x006d
	_XK_minus            = 0x002d
	_XK_n                = 0x006e
	_XK_o                = 0x006f
	_XK_p                = 0x0070
	_XK_period           = 0x002e
	_XK_q                = 0x0071
	_XK_r                = 0x0072
	_XK_s                = 0x0073
	_XK_semicolon        = 0x003b
	_XK_slash            = 0x002f
	_XK_space            = 0x0020
	_XK_t                = 0x0074
	_XK_u                = 0x0075
	_XK_v                = 0x0076
	_XK_w                = 0x0077
	_XK_x                = 0x0078
	_XK_y                = 0x0079
	_XK_z                = 0x007a
)
