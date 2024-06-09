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

package glfw

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type handleError windows.Handle

func (h handleError) Error() string {
	return fmt.Sprintf("HANDLE(%d)", h)
}

// math.MaxUint was added at Go 1.17. See https://github.com/golang/go/issues/28538
const (
	intSize = 32 << (^uint(0) >> 63)
)

// For the definitions, see https://github.com/wine-mirror/wine
const (
	_BI_BITFIELDS                                              = 3
	_CCHDEVICENAME                                             = 32
	_CCHFORMNAME                                               = 32
	_CDS_TEST                                                  = 0x00000002
	_CDS_FULLSCREEN                                            = 0x00000004
	_CS_HREDRAW                                                = 0x00000002
	_CS_OWNDC                                                  = 0x00000020
	_CS_VREDRAW                                                = 0x00000001
	_CW_USEDEFAULT                                             = int32(^0x7fffffff)
	_DBT_DEVTYP_DEVICEINTERFACE                                = 0x00000005
	_DEVICE_NOTIFY_WINDOW_HANDLE                               = 0x00000000
	_DIB_RGB_COLORS                                            = 0
	_DISP_CHANGE_SUCCESSFUL                                    = 0
	_DISP_CHANGE_RESTART                                       = 1
	_DISP_CHANGE_FAILED                                        = -1
	_DISP_CHANGE_BADMODE                                       = -2
	_DISP_CHANGE_NOTUPDATED                                    = -3
	_DISP_CHANGE_BADFLAGS                                      = -4
	_DISP_CHANGE_BADPARAM                                      = -5
	_DISP_CHANGE_BADDUALVIEW                                   = -6
	_DISPLAY_DEVICE_ACTIVE                                     = 0x00000001
	_DISPLAY_DEVICE_MODESPRUNED                                = 0x08000000
	_DISPLAY_DEVICE_PRIMARY_DEVICE                             = 0x00000004
	_DM_BITSPERPEL                                             = 0x00040000
	_DM_PELSWIDTH                                              = 0x00080000
	_DM_PELSHEIGHT                                             = 0x00100000
	_DM_DISPLAYFREQUENCY                                       = 0x00400000
	_DWM_BB_BLURREGION                                         = 0x00000002
	_DWM_BB_ENABLE                                             = 0x00000001
	_EDS_ROTATEDMODE                                           = 0x00000004
	_ENUM_CURRENT_SETTINGS                        uint32       = 0xffffffff
	_GCLP_HICON                                                = -14
	_GCLP_HICONSM                                              = -34
	_GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS                    = 0x00000004
	_GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT              = 0x00000002
	_GWL_EXSTYLE                                               = -20
	_GWL_STYLE                                                 = -16
	_HTCLIENT                                                  = 1
	_HORZSIZE                                                  = 4
	_HWND_NOTOPMOST                               windows.HWND = (1 << intSize) - 2
	_HWND_TOP                                     windows.HWND = 0
	_HWND_TOPMOST                                 windows.HWND = (1 << intSize) - 1
	_ICON_BIG                                                  = 1
	_ICON_SMALL                                                = 0
	_IDC_ARROW                                                 = 32512
	_IDI_APPLICATION                                           = 32512
	_IMAGE_CURSOR                                              = 2
	_IMAGE_ICON                                                = 1
	_KF_ALTDOWN                                                = 0x2000
	_KF_DLGMODE                                                = 0x0800
	_KF_EXTENDED                                               = 0x0100
	_KF_MENUMODE                                               = 0x1000
	_KF_REPEAT                                                 = 0x4000
	_KF_UP                                                     = 0x8000
	_LOGPIXELSX                                                = 88
	_LOGPIXELSY                                                = 90
	_LR_DEFAULTSIZE                                            = 0x0040
	_LR_SHARED                                                 = 0x8000
	_LWA_ALPHA                                                 = 0x00000002
	_MAPVK_VK_TO_VSC                                           = 0
	_MAPVK_VSC_TO_VK                                           = 1
	_MONITOR_DEFAULTTONEAREST                                  = 0x00000002
	_MOUSE_MOVE_ABSOLUTE                                       = 0x01
	_MOUSE_VIRTUAL_DESKTOP                                     = 0x02
	_MSGFLT_ALLOW                                              = 1
	_OCR_CROSS                                                 = 32515
	_OCR_HAND                                                  = 32649
	_OCR_IBEAM                                                 = 32513
	_OCR_NO                                                    = 32648
	_OCR_NORMAL                                                = 32512
	_OCR_SIZEALL                                               = 32646
	_OCR_SIZENESW                                              = 32643
	_OCR_SIZENS                                                = 32645
	_OCR_SIZENWSE                                              = 32642
	_OCR_SIZEWE                                                = 32644
	_PM_NOREMOVE                                               = 0x0000
	_PM_REMOVE                                                 = 0x0001
	_PFD_DRAW_TO_WINDOW                                        = 0x00000004
	_PFD_DOUBLEBUFFER                                          = 0x00000001
	_PFD_GENERIC_ACCELERATED                                   = 0x00001000
	_PFD_GENERIC_FORMAT                                        = 0x00000040
	_PFD_STEREO                                                = 0x00000002
	_PFD_SUPPORT_OPENGL                                        = 0x00000020
	_PFD_TYPE_RGBA                                             = 0
	_QS_ALLEVENTS                                              = _QS_INPUT | _QS_POSTMESSAGE | _QS_TIMER | _QS_PAINT | _QS_HOTKEY
	_QS_ALLINPUT                                               = _QS_INPUT | _QS_POSTMESSAGE | _QS_TIMER | _QS_PAINT | _QS_HOTKEY | _QS_SENDMESSAGE
	_QS_HOTKEY                                                 = 0x0080
	_QS_INPUT                                                  = _QS_MOUSE | _QS_KEY | _QS_RAWINPUT
	_QS_KEY                                                    = 0x0001
	_QS_MOUSE                                                  = _QS_MOUSEMOVE | _QS_MOUSEBUTTON
	_QS_MOUSEBUTTON                                            = 0x0004
	_QS_MOUSEMOVE                                              = 0x0002
	_QS_PAINT                                                  = 0x0020
	_QS_POSTMESSAGE                                            = 0x0008
	_QS_RAWINPUT                                               = 0x0400
	_QS_SENDMESSAGE                                            = 0x0040
	_QS_TIMER                                                  = 0x0010
	_RID_INPUT                                                 = 0x10000003
	_RIDEV_REMOVE                                              = 0x00000001
	_SC_KEYMENU                                                = 0xf100
	_SC_MONITORPOWER                                           = 0xf170
	_SC_SCREENSAVE                                             = 0xf140
	_SIZE_MAXIMIZED                                            = 2
	_SIZE_MINIMIZED                                            = 1
	_SIZE_RESTORED                                             = 0
	_SM_CXCURSOR                                               = 13
	_SM_CXICON                                                 = 11
	_SM_CXSCREEN                                               = 0
	_SM_CXSMICON                                               = 49
	_SM_CYCAPTION                                              = 4
	_SM_CYCURSOR                                               = 14
	_SM_CYICON                                                 = 12
	_SM_CYSCREEN                                               = 1
	_SM_CYSMICON                                               = 50
	_SM_CXVIRTUALSCREEN                                        = 78
	_SM_CYVIRTUALSCREEN                                        = 79
	_SM_REMOTESESSION                                          = 0x1000
	_SPI_GETFOREGROUNDLOCKTIMEOUT                              = 0x2000
	_SPI_GETMOUSETRAILS                                        = 94
	_SPI_SETFOREGROUNDLOCKTIMEOUT                              = 0x2001
	_SPI_SETMOUSETRAILS                                        = 93
	_SPIF_SENDCHANGE                                           = _SPIF_SENDWININICHANGE
	_SPIF_SENDWININICHANGE                                     = 2
	_SW_HIDE                                                   = 0
	_SW_MAXIMIZE                                               = _SW_SHOWMAXIMIZED
	_SW_MINIMIZE                                               = 6
	_SW_RESTORE                                                = 9
	_SW_SHOWNA                                                 = 8
	_SW_SHOWMAXIMIZED                                          = 3
	_SWP_FRAMECHANGED                                          = 0x0020
	_SWP_NOACTIVATE                                            = 0x0010
	_SWP_NOCOPYBITS                                            = 0x0100
	_SWP_NOMOVE                                                = 0x0002
	_SWP_NOOWNERZORDER                                         = 0x0200
	_SWP_NOSIZE                                                = 0x0001
	_SWP_NOZORDER                                              = 0x0004
	_SWP_SHOWWINDOW                                            = 0x0040
	_TLS_OUT_OF_INDEXES                           uint32       = 0xffffffff
	_TME_LEAVE                                                 = 0x00000002
	_UNICODE_NOCHAR                                            = 0xffff
	_USER_DEFAULT_SCREEN_DPI                                   = 96
	_VERTSIZE                                                  = 6
	_VK_ADD                                                    = 0x6B
	_VK_CAPITAL                                                = 0x14
	_VK_CONTROL                                                = 0x11
	_VK_DECIMAL                                                = 0x6E
	_VK_DIVIDE                                                 = 0x6F
	_VK_LSHIFT                                                 = 0xA0
	_VK_LWIN                                                   = 0x5B
	_VK_MENU                                                   = 0x12
	_VK_MULTIPLY                                               = 0x6A
	_VK_NUMLOCK                                                = 0x90
	_VK_NUMPAD0                                                = 0x60
	_VK_NUMPAD1                                                = 0x61
	_VK_NUMPAD2                                                = 0x62
	_VK_NUMPAD3                                                = 0x63
	_VK_NUMPAD4                                                = 0x64
	_VK_NUMPAD5                                                = 0x65
	_VK_NUMPAD6                                                = 0x66
	_VK_NUMPAD7                                                = 0x67
	_VK_NUMPAD8                                                = 0x68
	_VK_NUMPAD9                                                = 0x69
	_VK_PROCESSKEY                                             = 0xE5
	_VK_RSHIFT                                                 = 0xA1
	_VK_RWIN                                                   = 0x5C
	_VK_SHIFT                                                  = 0x10
	_VK_SNAPSHOT                                               = 0x2C
	_VK_SUBTRACT                                               = 0x6D
	_WAIT_FAILED                                               = 0xffffffff
	_WHEEL_DELTA                                               = 120
	_WGL_ACCUM_BITS_ARB                                        = 0x201D
	_WGL_ACCELERATION_ARB                                      = 0x2003
	_WGL_ACCUM_ALPHA_BITS_ARB                                  = 0x2021
	_WGL_ACCUM_BLUE_BITS_ARB                                   = 0x2020
	_WGL_ACCUM_GREEN_BITS_ARB                                  = 0x201F
	_WGL_ACCUM_RED_BITS_ARB                                    = 0x201E
	_WGL_AUX_BUFFERS_ARB                                       = 0x2024
	_WGL_ALPHA_BITS_ARB                                        = 0x201B
	_WGL_ALPHA_SHIFT_ARB                                       = 0x201C
	_WGL_BLUE_BITS_ARB                                         = 0x2019
	_WGL_BLUE_SHIFT_ARB                                        = 0x201A
	_WGL_COLOR_BITS_ARB                                        = 0x2014
	_WGL_COLORSPACE_EXT                                        = 0x309D
	_WGL_COLORSPACE_SRGB_EXT                                   = 0x3089
	_WGL_CONTEXT_COMPATIBILITY_PROFILE_BIT_ARB                 = 0x00000002
	_WGL_CONTEXT_CORE_PROFILE_BIT_ARB                          = 0x00000001
	_WGL_CONTEXT_DEBUG_BIT_ARB                                 = 0x0001
	_WGL_CONTEXT_ES2_PROFILE_BIT_EXT                           = 0x00000004
	_WGL_CONTEXT_FLAGS_ARB                                     = 0x2094
	_WGL_CONTEXT_FORWARD_COMPATIBLE_BIT_ARB                    = 0x0002
	_WGL_CONTEXT_MAJOR_VERSION_ARB                             = 0x2091
	_WGL_CONTEXT_MINOR_VERSION_ARB                             = 0x2092
	_WGL_CONTEXT_OPENGL_NO_ERROR_ARB                           = 0x31B3
	_WGL_CONTEXT_PROFILE_MASK_ARB                              = 0x9126
	_WGL_CONTEXT_RELEASE_BEHAVIOR_ARB                          = 0x2097
	_WGL_CONTEXT_RELEASE_BEHAVIOR_NONE_ARB                     = 0x0000
	_WGL_CONTEXT_RELEASE_BEHAVIOR_FLUSH_ARB                    = 0x2098
	_WGL_CONTEXT_RESET_NOTIFICATION_STRATEGY_ARB               = 0x8256
	_WGL_CONTEXT_ROBUST_ACCESS_BIT_ARB                         = 0x00000004
	_WGL_DEPTH_BITS_ARB                                        = 0x2022
	_WGL_DRAW_TO_BITMAP_ARB                                    = 0x2002
	_WGL_DRAW_TO_WINDOW_ARB                                    = 0x2001
	_WGL_DOUBLE_BUFFER_ARB                                     = 0x2011
	_WGL_FRAMEBUFFER_SRGB_CAPABLE_ARB                          = 0x20A9
	_WGL_GREEN_BITS_ARB                                        = 0x2017
	_WGL_GREEN_SHIFT_ARB                                       = 0x2018
	_WGL_LOSE_CONTEXT_ON_RESET_ARB                             = 0x8252
	_WGL_NEED_PALETTE_ARB                                      = 0x2004
	_WGL_NEED_SYSTEM_PALETTE_ARB                               = 0x2005
	_WGL_NO_ACCELERATION_ARB                                   = 0x2025
	_WGL_NO_RESET_NOTIFICATION_ARB                             = 0x8261
	_WGL_NUMBER_OVERLAYS_ARB                                   = 0x2008
	_WGL_NUMBER_PIXEL_FORMATS_ARB                              = 0x2000
	_WGL_NUMBER_UNDERLAYS_ARB                                  = 0x2009
	_WGL_PIXEL_TYPE_ARB                                        = 0x2013
	_WGL_RED_BITS_ARB                                          = 0x2015
	_WGL_RED_SHIFT_ARB                                         = 0x2016
	_WGL_SAMPLES_ARB                                           = 0x2042
	_WGL_SHARE_ACCUM_ARB                                       = 0x200E
	_WGL_SHARE_DEPTH_ARB                                       = 0x200C
	_WGL_SHARE_STENCIL_ARB                                     = 0x200D
	_WGL_STENCIL_BITS_ARB                                      = 0x2023
	_WGL_STEREO_ARB                                            = 0x2012
	_WGL_SUPPORT_GDI_ARB                                       = 0x200F
	_WGL_SUPPORT_OPENGL_ARB                                    = 0x2010
	_WGL_SWAP_LAYER_BUFFERS_ARB                                = 0x2006
	_WGL_SWAP_METHOD_ARB                                       = 0x2007
	_WGL_TRANSPARENT_ARB                                       = 0x200A
	_WGL_TRANSPARENT_ALPHA_VALUE_ARB                           = 0x203A
	_WGL_TRANSPARENT_BLUE_VALUE_ARB                            = 0x2039
	_WGL_TRANSPARENT_GREEN_VALUE_ARB                           = 0x2038
	_WGL_TRANSPARENT_INDEX_VALUE_ARB                           = 0x203B
	_WGL_TRANSPARENT_RED_VALUE_ARB                             = 0x2037
	_WGL_TYPE_RGBA_ARB                                         = 0x202B
	_WM_CAPTURECHANGED                                         = 0x0215
	_WM_CHAR                                                   = 0x0102
	_WM_CLOSE                                                  = 0x0010
	_WM_COPYDATA                                               = 0x004a
	_WM_COPYGLOBALDATA                                         = 0x0049
	_WM_DISPLAYCHANGE                                          = 0x007e
	_WM_DPICHANGED                                             = 0x02e0
	_WM_DROPFILES                                              = 0x0233
	_WM_DWMCOMPOSITIONCHANGED                                  = 0x031E
	_WM_DWMCOLORIZATIONCOLORCHANGED                            = 0x0320
	_WM_ENTERMENULOOP                                          = 0x0211
	_WM_ENTERSIZEMOVE                                          = 0x0231
	_WM_ERASEBKGND                                             = 0x0014
	_WM_EXITMENULOOP                                           = 0x0212
	_WM_EXITSIZEMOVE                                           = 0x0232
	_WM_GETDPISCALEDSIZE                                       = 0x02e4
	_WM_GETMINMAXINFO                                          = 0x0024
	_WM_INPUT                                                  = 0x00ff
	_WM_INPUTLANGCHANGE                                        = 0x0051
	_WM_KEYDOWN                                                = _WM_KEYFIRST
	_WM_KEYFIRST                                               = 0x0100
	_WM_KEYUP                                                  = 0x0101
	_WM_KILLFOCUS                                              = 0x0008
	_WM_LBUTTONDOWN                                            = 0x0201
	_WM_LBUTTONUP                                              = 0x0202
	_WM_MBUTTONDOWN                                            = 0x0207
	_WM_MBUTTONUP                                              = 0x0208
	_WM_NCACTIVATE                                             = 0x0086
	_WM_NCPAINT                                                = 0x0085
	_WM_NULL                                                   = 0x0000
	_WM_MOUSEACTIVATE                                          = 0x0021
	_WM_MOUSEFIRST                                             = 0x0200
	_WM_MOUSEHWHEEL                                            = 0x020E
	_WM_MOUSELEAVE                                             = 0x02A3
	_WM_MOUSEMOVE                                              = _WM_MOUSEFIRST
	_WM_MOUSEWHEEL                                             = 0x020A
	_WM_MOVE                                                   = 0x0003
	_WM_NCCREATE                                               = 0x0081
	_WM_PAINT                                                  = 0x000f
	_WM_QUIT                                                   = 0x0012
	_WM_RBUTTONDOWN                                            = 0x0204
	_WM_RBUTTONUP                                              = 0x0205
	_WM_SETCURSOR                                              = 0x0020
	_WM_SETFOCUS                                               = 0x0007
	_WM_SETICON                                                = 0x0080
	_WM_SIZE                                                   = 0x0005
	_WM_SIZING                                                 = 0x0214
	_WM_SYSCHAR                                                = 0x0106
	_WM_SYSCOMMAND                                             = 0x0112
	_WM_SYSKEYDOWN                                             = 0x0104
	_WM_SYSKEYUP                                               = 0x0105
	_WM_UNICHAR                                                = 0x0109
	_WM_XBUTTONDOWN                                            = 0x020B
	_WM_XBUTTONUP                                              = 0x020C
	_WMSZ_BOTTOM                                               = 6
	_WMSZ_BOTTOMLEFT                                           = 7
	_WMSZ_BOTTOMRIGHT                                          = 8
	_WMSZ_LEFT                                                 = 1
	_WMSZ_RIGHT                                                = 2
	_WMSZ_TOP                                                  = 3
	_WMSZ_TOPLEFT                                              = 4
	_WMSZ_TOPRIGHT                                             = 5
	_WS_BORDER                                                 = 0x00800000
	_WS_CAPTION                                                = _WS_BORDER | _WS_DLGFRAME
	_WS_CLIPSIBLINGS                                           = 0x04000000
	_WS_CLIPCHILDREN                                           = 0x02000000
	_WS_DLGFRAME                                               = 0x00400000
	_WS_EX_APPWINDOW                                           = 0x00040000
	_WS_EX_CLIENTEDGE                                          = 0x00000200
	_WS_EX_LAYERED                                             = 0x00080000
	_WS_EX_OVERLAPPEDWINDOW                                    = _WS_EX_WINDOWEDGE | _WS_EX_CLIENTEDGE
	_WS_EX_TOPMOST                                             = 0x00000008
	_WS_EX_TRANSPARENT                                         = 0x00000020
	_WS_EX_WINDOWEDGE                                          = 0x00000100
	_WS_MAXIMIZE                                               = 0x01000000
	_WS_MAXIMIZEBOX                                            = 0x00010000
	_WS_MINIMIZEBOX                                            = 0x00020000
	_WS_OVERLAPPED                                             = 0x00000000
	_WS_OVERLAPPEDWINDOW                                       = _WS_OVERLAPPED | _WS_CAPTION | _WS_SYSMENU | _WS_THICKFRAME | _WS_MINIMIZEBOX | _WS_MAXIMIZEBOX
	_WS_POPUP                                                  = 0x80000000
	_WS_SYSMENU                                                = 0x00080000
	_WS_THICKFRAME                                             = 0x00040000
	_XBUTTON1                                                  = 0x0001
)

type (
	_ATOM       uint16
	_BOOL       int32
	_COLORREF   uint32
	_HBITMAP    windows.Handle
	_HBRUSH     windows.Handle
	_HCURSOR    windows.Handle
	_HDC        windows.Handle
	_HDEVNOTIFY windows.Handle
	_HDROP      windows.Handle
	_HGDIOBJ    windows.Handle
	_HGLRC      windows.Handle
	_HICON      windows.Handle
	_HINSTANCE  windows.Handle
	_HMENU      windows.Handle
	_HMODULE    windows.Handle
	_HMONITOR   windows.Handle
	_HRAWINPUT  windows.Handle
	_HRGN       windows.Handle
	_LPARAM     uintptr
	_LRESULT    uintptr
	_WNDPROC    uintptr
	_WPARAM     uintptr
)

func _GET_X_LPARAM(lp _LPARAM) int {
	return int(int16(_LOWORD(uint32(lp))))
}

func _GET_XBUTTON_WPARAM(wParam _WPARAM) uint16 {
	return _HIWORD(uint32(wParam))
}

func _GET_Y_LPARAM(lp _LPARAM) int {
	return int(int16(_HIWORD(uint32(lp))))
}

func _HIWORD(dwValue uint32) uint16 {
	return uint16(dwValue >> 16)
}

func _LOWORD(dwValue uint32) uint16 {
	return uint16(dwValue)
}

type _DPI_AWARENESS_CONTEXT windows.Handle

const (
	_DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 _DPI_AWARENESS_CONTEXT = (1 << intSize) - 4
)

type _EXECUTION_STATE uint32

const (
	_ES_CONTINUOUS       _EXECUTION_STATE = 0x80000000
	_ES_DISPLAY_REQUIRED _EXECUTION_STATE = 0x00000002
)

type _MONITOR_DPI_TYPE int32

const (
	_MDT_EFFECTIVE_DPI _MONITOR_DPI_TYPE = 0
	_MDT_ANGULAR_DPI   _MONITOR_DPI_TYPE = 1
	_MDT_RAW_DPI       _MONITOR_DPI_TYPE = 2
	_MDT_DEFAULT       _MONITOR_DPI_TYPE = _MDT_EFFECTIVE_DPI
)

type _PROCESS_DPI_AWARENESS int32

const (
	_PROCESS_DPI_UNAWARE           _PROCESS_DPI_AWARENESS = 0
	_PROCESS_SYSTEM_DPI_AWARE      _PROCESS_DPI_AWARENESS = 1
	_PROCESS_PER_MONITOR_DPI_AWARE _PROCESS_DPI_AWARENESS = 2
)

type _BITMAPV5HEADER struct {
	bV5Size          uint32
	bV5Width         int32
	bV5Height        int32
	bV5Planes        uint16
	bV5BitCount      uint16
	bV5Compression   uint32
	bV5SizeImage     uint32
	bV5XPelsPerMeter int32
	bV5YPelsPerMeter int32
	bV5ClrUsed       uint32
	bV5ClrImportant  uint32
	bV5RedMask       uint32
	bV5GreenMask     uint32
	bV5BlueMask      uint32
	bV5AlphaMask     uint32
	bV5CSType        uint32
	bV5Endpoints     _CIEXYZTRIPLE
	bV5GammaRed      uint32
	bV5GammaGreen    uint32
	bV5GammaBlue     uint32
	bV5Intent        uint32
	bV5ProfileData   uint32
	bV5ProfileSize   uint32
	bV5Reserved      uint32
}

type _CHANGEFILTERSTRUCT struct {
	cbSize    uint32
	ExtStatus uint32
}

type _CIEXYZ struct {
	ciexyzX _FXPT2DOT30
	ciexyzY _FXPT2DOT30
	ciexyzZ _FXPT2DOT30
}

type _CIEXYZTRIPLE struct {
	ciexyzRed   _CIEXYZ
	ciexyzGreen _CIEXYZ
	ciexyzBlue  _CIEXYZ
}

type _CREATESTRUCTW struct {
	lpCreateParams unsafe.Pointer
	hInstance      _HINSTANCE
	hMenu          _HMENU
	hwndParent     windows.HWND
	cy             int32
	cx             int32
	y              int32
	x              int32
	style          int32
	lpszName       *uint16
	lpszClass      *uint16
	dwExStyle      uint32
}

type _DEV_BROADCAST_DEVICEINTERFACE_W struct {
	dbcc_size       uint32
	dbcc_devicetype uint32
	dbcc_reserved   uint32
	dbcc_classguid  windows.GUID
	dbcc_name       [1]uint16
}

type _DEVMODEW struct {
	dmDeviceName       [_CCHDEVICENAME]uint16
	dmSpecVersion      uint16
	dmDriverVersion    uint16
	dmSize             uint16
	dmDriverExtra      uint16
	dmFields           uint32
	dmPosition         _POINTL
	_                  [8]byte // the rest of union
	dmColor            int16
	dmDuplex           int16
	dmYResolution      int16
	dmTTOption         int16
	dmCollate          int16
	dmFormName         [_CCHFORMNAME]uint16
	dmLogPixels        uint16
	dmBitsPerPel       uint32
	dmPelsWidth        uint32
	dmPelsHeight       uint32
	dmDisplayFlags     uint32 // union with DWORD dmNup
	dmDisplayFrequency uint32
	dmICMMethod        uint32
	dmICMIntent        uint32
	dmMediaType        uint32
	dmDitherType       uint32
	dmReserved1        uint32
	dmReserved2        uint32
	dmPanningWidth     uint32
	dmPanningHeight    uint32
}

type _DISPLAY_DEVICEW struct {
	cb           uint32
	DeviceName   [32]uint16
	DeviceString [128]uint16
	StateFlags   uint32
	DeviceID     [128]uint16
	DeviceKey    [128]uint16
}

type _DWM_BLURBEHIND struct {
	dwFlags                uint32
	fEnable                int32
	hRgnBlur               _HRGN
	fTransitionOnMaximized int32
}

type _FXPT2DOT30 int32

type _ICONINFO struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  _HBITMAP
	hbmColor _HBITMAP
}

type _MINMAXINFO struct {
	ptReserved     _POINT
	ptMaxSize      _POINT
	ptMaxPosition  _POINT
	ptMinTrackSize _POINT
	ptMaxTrackSize _POINT
}

type _MONITORINFO struct {
	cbSize    uint32
	rcMonitor _RECT
	rcWork    _RECT
	dwFlags   uint32
}

type _MONITORINFOEXW struct {
	cbSize    uint32
	rcMonitor _RECT
	rcWork    _RECT
	dwFlags   uint32
	szDevice  [_CCHDEVICENAME]uint16
}

type _MSG struct {
	hwnd     windows.HWND
	message  uint32
	wParam   _WPARAM
	lParam   _LPARAM
	time     uint32
	pt       _POINT
	lPrivate uint32
}

type _PIXELFORMATDESCRIPTOR struct {
	nSize           uint16
	nVersion        uint16
	dwFlags         uint32
	iPixelType      byte
	cColorBits      byte
	cRedBits        byte
	cRedShift       byte
	cGreenBits      byte
	cGreenShift     byte
	cBlueBits       byte
	cBlueShift      byte
	cAlphaBits      byte
	cAlphaShift     byte
	cAccumBits      byte
	cAccumRedBits   byte
	cAccumGreenBits byte
	cAccumBlueBits  byte
	cAccumAlphaBits byte
	cDepthBits      byte
	cStencilBits    byte
	cAuxBuffers     byte
	iLayerType      byte
	bReserved       byte
	dwLayerMask     uint32
	dwVisibleMask   uint32
	dwDamageMask    uint32
}

type _POINT struct {
	x int32
	y int32
}

type _POINTL struct {
	x int32
	y int32
}

type _RAWINPUT struct {
	header _RAWINPUTHEADER
	mouse  _RAWMOUSE

	// RAWMOUSE is the biggest among RAWHID, RAWKEYBOARD, and RAWMOUSE.
	// Then, padding is not needed here.
}

type _RAWINPUTDEVICE struct {
	usUsagePage uint16
	usUsage     uint16
	dwFlags     uint32
	hwndTarget  windows.HWND
}

type _RAWINPUTHEADER struct {
	dwType  uint32
	dwSize  uint32
	hDevice windows.Handle
	wParam  uintptr
}

type _RAWMOUSE struct {
	usFlags            uint16
	ulButtons          uint32 // TODO: Check alignments
	ulRawButtons       uint32
	lLastX             int32
	lLastY             int32
	ulExtraInformation uint32
}

type _RECT struct {
	left   int32
	top    int32
	right  int32
	bottom int32
}

type _SIZE struct {
	cx int32
	cy int32
}

type _TRACKMOUSEEVENT struct {
	cbSize      uint32
	dwFlags     uint32
	hwndTrack   windows.HWND
	dwHoverTime uint32
}

type _WINDOWPLACEMENT struct {
	length           uint32
	flags            uint32
	showCmd          uint32
	ptMinPosition    _POINT
	ptMaxPosition    _POINT
	rcNormalPosition _RECT
	rcDevice         _RECT
}

type _WNDCLASSEXW struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   _WNDPROC
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     _HINSTANCE
	hIcon         _HICON
	hCursor       _HCURSOR
	hbrBackground _HBRUSH
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       _HICON
}

var (
	dwmapi   = windows.NewLazySystemDLL("dwmapi.dll")
	gdi32    = windows.NewLazySystemDLL("gdi32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	opengl32 = windows.NewLazySystemDLL("opengl32.dll")
	shcore   = windows.NewLazySystemDLL("shcore.dll")
	shell32  = windows.NewLazySystemDLL("shell32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")

	procDwmEnableBlurBehindWindow = dwmapi.NewProc("DwmEnableBlurBehindWindow")
	procDwmGetColorizationColor   = dwmapi.NewProc("DwmGetColorizationColor")
	procDwmFlush                  = dwmapi.NewProc("DwmFlush")
	procDwmIsCompositionEnabled   = dwmapi.NewProc("DwmIsCompositionEnabled")

	procChoosePixelFormat   = gdi32.NewProc("ChoosePixelFormat")
	procCreateBitmap        = gdi32.NewProc("CreateBitmap")
	procCreateDIBSection    = gdi32.NewProc("CreateDIBSection")
	procCreateRectRgn       = gdi32.NewProc("CreateRectRgn")
	procDeleteObject        = gdi32.NewProc("DeleteObject")
	procDescribePixelFormat = gdi32.NewProc("DescribePixelFormat")
	procGetDeviceCaps       = gdi32.NewProc("GetDeviceCaps")
	procSetPixelFormat      = gdi32.NewProc("SetPixelFormat")
	procSwapBuffers         = gdi32.NewProc("SwapBuffers")

	procGetModuleHandleExW      = kernel32.NewProc("GetModuleHandleExW")
	procSetThreadExecutionState = kernel32.NewProc("SetThreadExecutionState")
	procTlsAlloc                = kernel32.NewProc("TlsAlloc")
	procTlsFree                 = kernel32.NewProc("TlsFree")
	procTlsGetValue             = kernel32.NewProc("TlsGetValue")
	procTlsSetValue             = kernel32.NewProc("TlsSetValue")

	procWGLCreateContext     = opengl32.NewProc("wglCreateContext")
	procWGLDeleteContext     = opengl32.NewProc("wglDeleteContext")
	procWGLGetCurrentContext = opengl32.NewProc("wglGetCurrentContext")
	procWGLGetCurrentDC      = opengl32.NewProc("wglGetCurrentDC")
	procWGLGetProcAddress    = opengl32.NewProc("wglGetProcAddress")
	procWGLMakeCurrent       = opengl32.NewProc("wglMakeCurrent")
	procWGLShareLists        = opengl32.NewProc("wglShareLists")

	// Extension functions should be obtained from wglGetProcAddress instead of opengl32.dll (#2101).
	procWGLCreateContextAttribsARB   uintptr
	procWGLGetExtensionsStringARB    uintptr
	procWGLGetExtensionsStringEXT    uintptr
	procWGLGetPixelFormatAttribivARB uintptr
	procWGLSwapIntervalEXT           uintptr

	procGetDpiForMonitor       = shcore.NewProc("GetDpiForMonitor")
	procSetProcessDpiAwareness = shcore.NewProc("SetProcessDpiAwareness")

	procDragAcceptFiles = shell32.NewProc("DragAcceptFiles")
	procDragFinish      = shell32.NewProc("DragFinish")
	procDragQueryFileW  = shell32.NewProc("DragQueryFileW")
	procDragQueryPoint  = shell32.NewProc("DragQueryPoint")

	procAdjustWindowRectEx            = user32.NewProc("AdjustWindowRectEx")
	procAdjustWindowRectExForDpi      = user32.NewProc("AdjustWindowRectExForDpi")
	procBringWindowToTop              = user32.NewProc("BringWindowToTop")
	procChangeDisplaySettingsExW      = user32.NewProc("ChangeDisplaySettingsExW")
	procChangeWindowMessageFilterEx   = user32.NewProc("ChangeWindowMessageFilterEx")
	procClientToScreen                = user32.NewProc("ClientToScreen")
	procClipCursor                    = user32.NewProc("ClipCursor")
	procCreateCursor                  = user32.NewProc("CreateCursor")
	procCreateIconIndirect            = user32.NewProc("CreateIconIndirect")
	procCreateWindowExW               = user32.NewProc("CreateWindowExW")
	procDefWindowProcW                = user32.NewProc("DefWindowProcW")
	procDestroyCursor                 = user32.NewProc("DestroyCursor")
	procDestroyIcon                   = user32.NewProc("DestroyIcon")
	procDestroyWindow                 = user32.NewProc("DestroyWindow")
	procDispatchMessageW              = user32.NewProc("DispatchMessageW")
	procEnableNonClientDpiScaling     = user32.NewProc("EnableNonClientDpiScaling")
	procEnumDisplayDevicesW           = user32.NewProc("EnumDisplayDevicesW")
	procEnumDisplayMonitors           = user32.NewProc("EnumDisplayMonitors")
	procEnumDisplaySettingsW          = user32.NewProc("EnumDisplaySettingsW")
	procEnumDisplaySettingsExW        = user32.NewProc("EnumDisplaySettingsExW")
	procFlashWindow                   = user32.NewProc("FlashWindow")
	procGetActiveWindow               = user32.NewProc("GetActiveWindow")
	procGetClassLongPtrW              = user32.NewProc("GetClassLongPtrW")
	procGetClientRect                 = user32.NewProc("GetClientRect")
	procGetCursorPos                  = user32.NewProc("GetCursorPos")
	procGetDC                         = user32.NewProc("GetDC")
	procGetDpiForWindow               = user32.NewProc("GetDpiForWindow")
	procGetKeyState                   = user32.NewProc("GetKeyState")
	procGetLayeredWindowAttributes    = user32.NewProc("GetLayeredWindowAttributes")
	procGetMessageTime                = user32.NewProc("GetMessageTime")
	procGetMonitorInfoW               = user32.NewProc("GetMonitorInfoW")
	procGetRawInputData               = user32.NewProc("GetRawInputData")
	procGetSystemMetrics              = user32.NewProc("GetSystemMetrics")
	procGetSystemMetricsForDpi        = user32.NewProc("GetSystemMetricsForDpi")
	procGetWindowLongW                = user32.NewProc("GetWindowLongW")
	procGetWindowPlacement            = user32.NewProc("GetWindowPlacement")
	procGetWindowRect                 = user32.NewProc("GetWindowRect")
	procIsIconic                      = user32.NewProc("IsIconic")
	procIsWindowVisible               = user32.NewProc("IsWindowVisible")
	procIsZoomed                      = user32.NewProc("IsZoomed")
	procLoadCursorW                   = user32.NewProc("LoadCursorW")
	procLoadImageW                    = user32.NewProc("LoadImageW")
	procMapVirtualKeyW                = user32.NewProc("MapVirtualKeyW")
	procMonitorFromWindow             = user32.NewProc("MonitorFromWindow")
	procMoveWindow                    = user32.NewProc("MoveWindow")
	procMsgWaitForMultipleObjects     = user32.NewProc("MsgWaitForMultipleObjects")
	procOffsetRect                    = user32.NewProc("OffsetRect")
	procPeekMessageW                  = user32.NewProc("PeekMessageW")
	procPostMessageW                  = user32.NewProc("PostMessageW")
	procPtInRect                      = user32.NewProc("PtInRect")
	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procRegisterDeviceNotificationW   = user32.NewProc("RegisterDeviceNotificationW")
	procRegisterRawInputDevices       = user32.NewProc("RegisterRawInputDevices")
	procReleaseCapture                = user32.NewProc("ReleaseCapture")
	procReleaseDC                     = user32.NewProc("ReleaseDC")
	procScreenToClient                = user32.NewProc("ScreenToClient")
	procSendMessageW                  = user32.NewProc("SendMessageW")
	procSetCapture                    = user32.NewProc("SetCapture")
	procSetCursor                     = user32.NewProc("SetCursor")
	procSetCursorPos                  = user32.NewProc("SetCursorPos")
	procSetFocus                      = user32.NewProc("SetFocus")
	procSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	procSetLayeredWindowAttributes    = user32.NewProc("SetLayeredWindowAttributes")
	procSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetWindowLongW                = user32.NewProc("SetWindowLongW")
	procSetWindowPlacement            = user32.NewProc("SetWindowPlacement")
	procSetWindowPos                  = user32.NewProc("SetWindowPos")
	procSetWindowTextW                = user32.NewProc("SetWindowTextW")
	procShowWindow                    = user32.NewProc("ShowWindow")
	procSystemParametersInfoW         = user32.NewProc("SystemParametersInfoW")
	procToUnicode                     = user32.NewProc("ToUnicode")
	procTranslateMessage              = user32.NewProc("TranslateMessage")
	procTrackMouseEvent               = user32.NewProc("TrackMouseEvent")
	procUnregisterClassW              = user32.NewProc("UnregisterClassW")
	procUnregisterDeviceNotification  = user32.NewProc("UnregisterDeviceNotification")
	procWaitMessage                   = user32.NewProc("WaitMessage")
	procWindowFromPoint               = user32.NewProc("WindowFromPoint")
)

func initWGLExtensionFunctions() {
	procWGLCreateContextAttribsARB = wglGetProcAddress("wglCreateContextAttribsARB")
	procWGLGetExtensionsStringARB = wglGetProcAddress("wglGetExtensionsStringARB")
	procWGLGetExtensionsStringEXT = wglGetProcAddress("wglGetExtensionsStringEXT")
	procWGLGetPixelFormatAttribivARB = wglGetProcAddress("wglGetPixelFormatAttribivARB")
	procWGLSwapIntervalEXT = wglGetProcAddress("wglSwapIntervalEXT")
}

func _AdjustWindowRectEx(lpRect *_RECT, dwStyle uint32, menu bool, dwExStyle uint32) error {
	var bMenu uintptr
	if menu {
		bMenu = 1
	}
	r, _, e := procAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(lpRect)), uintptr(dwStyle), bMenu, uintptr(dwExStyle))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: AdjustWindowRectEx failed: %w", e)
	}
	return nil
}

func _AdjustWindowRectExForDpi(lpRect *_RECT, dwStyle uint32, menu bool, dwExStyle uint32, dpi uint32) error {
	var bMenu uintptr
	if menu {
		bMenu = 1
	}
	r, _, e := procAdjustWindowRectExForDpi.Call(uintptr(unsafe.Pointer(lpRect)), uintptr(dwStyle), bMenu, uintptr(dwExStyle), uintptr(dpi))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: AdjustWindowRectExForDpi failed: %w", e)
	}
	return nil
}

func _BringWindowToTop(hWnd windows.HWND) error {
	r, _, e := procBringWindowToTop.Call(uintptr(hWnd))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: BringWindowToTop failed: %w", e)
	}
	return nil
}

func _ChangeDisplaySettingsExW(deviceName string, lpDevMode *_DEVMODEW, hwnd windows.HWND, dwflags uint32, lParam unsafe.Pointer) int32 {
	var lpszDeviceName *uint16
	if deviceName != "" {
		var err error
		lpszDeviceName, err = windows.UTF16PtrFromString(deviceName)
		if err != nil {
			panic("glfw: device name must not include a NUL character")
		}
	}

	r, _, _ := procChangeDisplaySettingsExW.Call(uintptr(unsafe.Pointer(lpszDeviceName)), uintptr(unsafe.Pointer(lpDevMode)), uintptr(hwnd), uintptr(dwflags), uintptr(lParam))
	runtime.KeepAlive(lpszDeviceName)
	runtime.KeepAlive(lpDevMode)

	return int32(r)
}

func _ChangeWindowMessageFilterEx(hwnd windows.HWND, message uint32, action uint32, pChangeFilterStruct *_CHANGEFILTERSTRUCT) error {
	r, _, e := procChangeWindowMessageFilterEx.Call(uintptr(hwnd), uintptr(message), uintptr(action), uintptr(unsafe.Pointer(pChangeFilterStruct)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: ChangeWindowMessageFilterEx failed: %w", e)
	}
	return nil
}

func _ChoosePixelFormat(hdc _HDC, ppfd *_PIXELFORMATDESCRIPTOR) (int32, error) {
	r, _, e := procChoosePixelFormat.Call(uintptr(hdc), uintptr(unsafe.Pointer(ppfd)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("glfw: ChoosePixelFormat failed: %w", e)
	}
	return int32(r), nil
}

func _ClientToScreen(hWnd windows.HWND, lpPoint *_POINT) error {
	r, _, e := procClientToScreen.Call(uintptr(hWnd), uintptr(unsafe.Pointer(lpPoint)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: ClientToScreen failed: %w", e)
	}
	return nil
}

func _ClipCursor(lpRect *_RECT) error {
	r, _, e := procClipCursor.Call(uintptr(unsafe.Pointer(lpRect)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: ClipCursor failed: %w", e)
	}
	return nil
}

func _CreateCursor(hInst _HINSTANCE, xHotSpot int32, yHotSpot int32, nWidth int32, nHeight int32, pvANDPlane, pvXORPlane []byte) (_HCURSOR, error) {
	var andPlane *byte
	if len(pvANDPlane) > 0 {
		andPlane = &pvANDPlane[0]
	}
	var xorPlane *byte
	if len(pvXORPlane) > 0 {
		xorPlane = &pvXORPlane[0]
	}

	r, _, e := procCreateCursor.Call(uintptr(hInst), uintptr(xHotSpot), uintptr(yHotSpot), uintptr(nWidth), uintptr(nHeight), uintptr(unsafe.Pointer(andPlane)), uintptr(unsafe.Pointer(xorPlane)))
	runtime.KeepAlive(pvANDPlane)
	runtime.KeepAlive(pvXORPlane)

	if _HCURSOR(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("glfw: CreateCursor failed: %w", e)
	}
	return _HCURSOR(r), nil
}

func _CreateBitmap(nWidth int32, nHeight int32, nPlanes uint32, nBitCount uint32, lpBits unsafe.Pointer) (_HBITMAP, error) {
	r, _, e := procCreateBitmap.Call(uintptr(nWidth), uintptr(nHeight), uintptr(nPlanes), uintptr(nBitCount), uintptr(lpBits))
	if _HBITMAP(r) == 0 {
		return 0, fmt.Errorf("glfw: CreateBitmap failed: %w", e)
	}
	return _HBITMAP(r), nil
}

func _CreateDIBSection(hdc _HDC, pbmi *_BITMAPV5HEADER, usage uint32, hSection windows.Handle, offset uint32) (_HBITMAP, *byte, error) {
	// pbmi is originally *BITMAPINFO.
	var bits *byte
	r, _, e := procCreateDIBSection.Call(uintptr(hdc), uintptr(unsafe.Pointer(pbmi)), uintptr(usage), uintptr(unsafe.Pointer(&bits)), uintptr(hSection), uintptr(offset))
	if _HBITMAP(r) == 0 {
		return 0, nil, fmt.Errorf("glfw: CreateDIBSection failed: %w", e)
	}
	return _HBITMAP(r), bits, nil
}

func _CreateRectRgn(x1, y1, x2, y2 int32) (_HRGN, error) {
	r, _, e := procCreateRectRgn.Call(uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2))
	if _HRGN(r) == 0 {
		return 0, fmt.Errorf("glfw: CreateRectRgn failed: %w", e)
	}
	return _HRGN(r), nil
}

func _CreateIconIndirect(piconinfo *_ICONINFO) (_HICON, error) {
	r, _, e := procCreateIconIndirect.Call(uintptr(unsafe.Pointer(piconinfo)))
	if _HICON(r) == 0 {
		return 0, fmt.Errorf("glfw: CreateIconIndirect failed: %w", e)
	}
	return _HICON(r), nil
}

func _CreateWindowExW(dwExStyle uint32, className string, windowName string, dwStyle uint32, x, y, nWidth, nHeight int32, hWndParent windows.HWND, hMenu _HMENU, hInstance _HINSTANCE, lpParam unsafe.Pointer) (windows.HWND, error) {
	var lpClassName *uint16
	if className != "" {
		var err error
		lpClassName, err = windows.UTF16PtrFromString(className)
		if err != nil {
			panic("glfw: class name msut not include a NUL character")
		}
	}

	var lpWindowName *uint16
	if windowName != "" {
		var err error
		lpWindowName, err = windows.UTF16PtrFromString(windowName)
		if err != nil {
			panic("glfw: window name msut not include a NUL character")
		}
	}

	r, _, e := procCreateWindowExW.Call(
		uintptr(dwExStyle), uintptr(unsafe.Pointer(lpClassName)), uintptr(unsafe.Pointer(lpWindowName)), uintptr(dwStyle),
		uintptr(x), uintptr(y), uintptr(nWidth), uintptr(nHeight),
		uintptr(hWndParent), uintptr(hMenu), uintptr(hInstance), uintptr(lpParam))
	runtime.KeepAlive(lpClassName)
	runtime.KeepAlive(lpWindowName)

	if windows.HWND(r) == 0 {
		return 0, fmt.Errorf("glfw: CreateWindowExW failed: %w", e)
	}
	return windows.HWND(r), nil
}

func _DefWindowProcW(hWnd windows.HWND, uMsg uint32, wParam _WPARAM, lParam _LPARAM) _LRESULT {
	r, _, _ := procDefWindowProcW.Call(uintptr(hWnd), uintptr(uMsg), uintptr(wParam), uintptr(lParam))
	return _LRESULT(r)
}

func _DestroyCursor(hCursor _HCURSOR) error {
	r, _, e := procDestroyCursor.Call(uintptr(hCursor))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: DestroyCursor failed: %w", e)
	}
	return nil
}

func _DestroyIcon(hIcon _HICON) error {
	r, _, e := procDestroyIcon.Call(uintptr(hIcon))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: DestroyIcon failed: %w", e)
	}
	return nil
}

func _DestroyWindow(hWnd windows.HWND) error {
	r, _, e := procDestroyWindow.Call(uintptr(hWnd))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: DestroyWindow failed: %w", e)
	}
	return nil
}

func _DeleteObject(ho _HGDIOBJ) error {
	r, _, e := procDeleteObject.Call(uintptr(ho))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: DeleteObject failed: %w", e)
	}
	return nil
}

func _DescribePixelFormat(hdc _HDC, iPixelFormat int32, nBytes uint32, ppfd *_PIXELFORMATDESCRIPTOR) (int32, error) {
	r, _, e := procDescribePixelFormat.Call(uintptr(hdc), uintptr(iPixelFormat), uintptr(nBytes), uintptr(unsafe.Pointer(ppfd)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("glfw: DescribePixelFormat failed: %w", e)
	}
	return int32(r), nil
}

func _DispatchMessageW(lpMsg *_MSG) _LRESULT {
	r, _, _ := procDispatchMessageW.Call(uintptr(unsafe.Pointer(lpMsg)))
	return _LRESULT(r)
}

func _DragAcceptFiles(hWnd windows.HWND, accept bool) {
	var fAccept uintptr
	if accept {
		fAccept = 1
	}
	_, _, _ = procDragAcceptFiles.Call(uintptr(hWnd), fAccept)
}

func _DragFinish(hDrop _HDROP) {
	_, _, _ = procDragFinish.Call(uintptr(hDrop))
}

func _DragQueryFileW(hDrop _HDROP, iFile uint32, file []uint16) uint32 {
	var filePtr unsafe.Pointer
	if len(file) > 0 {
		filePtr = unsafe.Pointer(&file[0])
	}
	r, _, _ := procDragQueryFileW.Call(uintptr(hDrop), uintptr(iFile), uintptr(filePtr), uintptr(len(file)))
	return uint32(r)
}

func _DragQueryPoint(hDrop _HDROP) (_POINT, bool) {
	var pt _POINT
	r, _, _ := procDragQueryPoint.Call(uintptr(hDrop), uintptr(unsafe.Pointer(&pt)))
	if int32(r) == 0 {
		return _POINT{}, false
	}
	return pt, true
}

func _DwmEnableBlurBehindWindow(hWnd windows.HWND, pBlurBehind *_DWM_BLURBEHIND) error {
	r, _, _ := procDwmEnableBlurBehindWindow.Call(uintptr(hWnd), uintptr(unsafe.Pointer(pBlurBehind)))
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("glfw: DwmEnableBlurBehindWindow failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func _DwmGetColorizationColor() (uint32, bool, error) {
	var colorization uint32
	var opaqueBlend int32
	r, _, _ := procDwmGetColorizationColor.Call(uintptr(unsafe.Pointer(&colorization)), uintptr(unsafe.Pointer(&opaqueBlend)))
	if uint32(r) != uint32(windows.S_OK) {
		return 0, false, fmt.Errorf("glfw: DwmGetColorizationColor failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return colorization, opaqueBlend != 0, nil
}

func _DwmFlush() error {
	r, _, _ := procDwmFlush.Call()
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("glfw: DwmFlush failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func _DwmIsCompositionEnabled() (bool, error) {
	var enabled int32
	r, _, _ := procDwmIsCompositionEnabled.Call(uintptr(unsafe.Pointer(&enabled)))
	if uint32(r) != uint32(windows.S_OK) {
		return false, fmt.Errorf("glfw: DwmIsCompositionEnabled failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return enabled != 0, nil
}

func _EnableNonClientDpiScaling(hwnd windows.HWND) error {
	r, _, e := procEnableNonClientDpiScaling.Call(uintptr(hwnd))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: EnableNonClientDpiScaling failed: %w", e)
	}
	return nil
}

func _EnumDisplayDevicesW(device string, iDevNum uint32, dwFlags uint32) (_DISPLAY_DEVICEW, bool) {
	var lpDevice *uint16
	if device != "" {
		var err error
		lpDevice, err = windows.UTF16PtrFromString(device)
		if err != nil {
			panic("glfw: device name must not include a NUL character")
		}
	}

	var displayDevice _DISPLAY_DEVICEW
	displayDevice.cb = uint32(unsafe.Sizeof(displayDevice))
	r, _, _ := procEnumDisplayDevicesW.Call(uintptr(unsafe.Pointer(lpDevice)), uintptr(iDevNum), uintptr(unsafe.Pointer(&displayDevice)), uintptr(dwFlags))
	runtime.KeepAlive(lpDevice)

	if int32(r) == 0 {
		return _DISPLAY_DEVICEW{}, false
	}
	return displayDevice, true
}

func _EnumDisplayMonitors(hdc _HDC, lprcClip *_RECT, lpfnEnum uintptr, dwData _LPARAM) error {
	r, _, e := procEnumDisplayMonitors.Call(uintptr(hdc), uintptr(unsafe.Pointer(lprcClip)), uintptr(lpfnEnum), uintptr(dwData))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: EnumDisplayMonitors failed: %w", e)
	}
	return nil
}

func _EnumDisplaySettingsExW(deviceName string, iModeNum uint32, dwFlags uint32) (_DEVMODEW, bool) {
	var lpszDeviceName *uint16
	if deviceName != "" {
		var err error
		lpszDeviceName, err = windows.UTF16PtrFromString(deviceName)
		if err != nil {
			panic("glfw: device name must not include a NUL character")
		}
	}

	var dm _DEVMODEW
	dm.dmSize = uint16(unsafe.Sizeof(dm))

	r, _, _ := procEnumDisplaySettingsExW.Call(uintptr(unsafe.Pointer(lpszDeviceName)), uintptr(iModeNum), uintptr(unsafe.Pointer(&dm)), uintptr(dwFlags))
	runtime.KeepAlive(lpszDeviceName)

	if int32(r) == 0 {
		return _DEVMODEW{}, false
	}
	return dm, true
}

func _EnumDisplaySettingsW(deviceName string, iModeNum uint32) (_DEVMODEW, bool) {
	var lpszDeviceName *uint16
	if deviceName != "" {
		var err error
		lpszDeviceName, err = windows.UTF16PtrFromString(deviceName)
		if err != nil {
			panic("glfw: device name must not include a NUL character")
		}
	}

	var dm _DEVMODEW
	dm.dmSize = uint16(unsafe.Sizeof(dm))

	r, _, _ := procEnumDisplaySettingsW.Call(uintptr(unsafe.Pointer(lpszDeviceName)), uintptr(iModeNum), uintptr(unsafe.Pointer(&dm)))
	runtime.KeepAlive(lpszDeviceName)

	if int32(r) == 0 {
		return _DEVMODEW{}, false
	}
	return dm, true
}

func _FlashWindow(hWnd windows.HWND, invert bool) bool {
	var bInvert uintptr
	if invert {
		bInvert = 1
	}
	r, _, _ := procFlashWindow.Call(uintptr(hWnd), bInvert)
	return int32(r) != 0
}

func _GetActiveWindow() windows.HWND {
	r, _, _ := procGetActiveWindow.Call()
	return windows.HWND(r)
}

func _GetClassLongPtrW(hWnd windows.HWND, nIndex int32) (uintptr, error) {
	r, _, e := procGetClassLongPtrW.Call(uintptr(hWnd), uintptr(nIndex))
	if r == 0 {
		return 0, fmt.Errorf("glfw: GetClassLongPtrW failed: %w", e)
	}
	return r, nil
}

func _GetClientRect(hWnd windows.HWND) (_RECT, error) {
	var rect _RECT
	r, _, e := procGetClientRect.Call(uintptr(hWnd), uintptr(unsafe.Pointer(&rect)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return _RECT{}, fmt.Errorf("glfw: GetClientRect failed: %w", e)
	}
	return rect, nil
}

func _GetCursorPos() (_POINT, error) {
	var point _POINT
	r, _, e := procGetCursorPos.Call(uintptr(unsafe.Pointer(&point)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return _POINT{}, fmt.Errorf("glfw: GetCursorPos failed: %w", e)
	}
	return point, nil
}

func _GetDC(hWnd windows.HWND) (_HDC, error) {
	r, _, e := procGetDC.Call(uintptr(hWnd))
	if _HDC(r) == 0 {
		return 0, fmt.Errorf("glfw: GetDC failed: %w", e)
	}
	return _HDC(r), nil
}

func _GetDeviceCaps(hdc _HDC, index int32) int32 {
	r, _, _ := procGetDeviceCaps.Call(uintptr(hdc), uintptr(index))
	return int32(r)
}

func _GetDpiForWindow(hwnd windows.HWND) uint32 {
	r, _, _ := procGetDpiForWindow.Call(uintptr(hwnd))
	return uint32(r)
}

func _GetKeyState(nVirtKey int32) int16 {
	r, _, _ := procGetKeyState.Call(uintptr(nVirtKey))
	return int16(r)
}

func _GetLayeredWindowAttributes(hWnd windows.HWND) (key _COLORREF, alpha byte, flags uint32, err error) {
	r, _, e := procGetLayeredWindowAttributes.Call(uintptr(hWnd), uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&alpha)), uintptr(unsafe.Pointer(&flags)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, 0, 0, fmt.Errorf("glfw: GetLayeredWindowAttributes failed: %w", e)
	}
	return
}

func _GetMessageTime() int32 {
	r, _, _ := procGetMessageTime.Call()
	return int32(r)
}

func _GetModuleHandleExW(dwFlags uint32, lpModuleName any) (_HMODULE, error) {
	var ptr unsafe.Pointer
	switch moduleName := lpModuleName.(type) {
	case string:
		if moduleName != "" {
			p, err := windows.UTF16PtrFromString(moduleName)
			if err != nil {
				panic("glfw: module name must not include a NUL character")
			}
			ptr = unsafe.Pointer(p)
		}
	case unsafe.Pointer:
		ptr = moduleName
	default:
		return 0, fmt.Errorf("glfw: GetModuleHandleExW: lpModuleName must be a string or an unsafe.Pointer but %T", moduleName)
	}

	var module _HMODULE
	r, _, e := procGetModuleHandleExW.Call(uintptr(dwFlags), uintptr(ptr), uintptr(unsafe.Pointer(&module)))
	runtime.KeepAlive(ptr)

	if int32(r) != 1 {
		return 0, fmt.Errorf("glfw: GetModuleHandleExW failed: %w", e)
	}
	return module, nil
}

func _GetMonitorInfoW(hMonitor _HMONITOR) (_MONITORINFO, bool) {
	var mi _MONITORINFO
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	r, _, _ := procGetMonitorInfoW.Call(uintptr(hMonitor), uintptr(unsafe.Pointer(&mi)))
	if int32(r) == 0 {
		return _MONITORINFO{}, false
	}
	return mi, true
}

func _GetMonitorInfoW_Ex(hMonitor _HMONITOR) (_MONITORINFOEXW, bool) {
	var mi _MONITORINFOEXW
	mi.cbSize = uint32(unsafe.Sizeof(mi))
	r, _, _ := procGetMonitorInfoW.Call(uintptr(hMonitor), uintptr(unsafe.Pointer(&mi)))
	if int32(r) == 0 {
		return _MONITORINFOEXW{}, false
	}
	return mi, true
}

func _GetDpiForMonitor(hmonitor _HMONITOR, dpiType _MONITOR_DPI_TYPE) (dpiX, dpiY uint32, err error) {
	r, _, _ := procGetDpiForMonitor.Call(uintptr(hmonitor), uintptr(dpiType), uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))
	if uint32(r) != uint32(windows.S_OK) {
		return 0, 0, fmt.Errorf("glfw: GetDpiForMonitor failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return dpiX, dpiY, nil
}

func _GetRawInputData(hRawInput _HRAWINPUT, uiCommand uint32, pData unsafe.Pointer, pcbSize *uint32) (uint32, error) {
	r, _, e := procGetRawInputData.Call(uintptr(hRawInput), uintptr(uiCommand), uintptr(pData), uintptr(unsafe.Pointer(pcbSize)), unsafe.Sizeof(_RAWINPUTHEADER{}))
	if uint32(r) == (1<<32)-1 {
		return 0, fmt.Errorf("glfw: GetRawInputData failed: %w", e)
	}
	return uint32(r), nil
}

func _GetSystemMetrics(nIndex int32) (int32, error) {
	r, _, e := procGetSystemMetrics.Call(uintptr(nIndex))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("glfw: GetSystemMetrics failed: %w", e)
	}
	return int32(r), nil
}

func _GetSystemMetricsForDpi(nIndex int32, dpi uint32) (int32, error) {
	r, _, e := procGetSystemMetricsForDpi.Call(uintptr(nIndex), uintptr(dpi))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("glfw: GetSystemMetrics failed: %w", e)
	}
	return int32(r), nil
}

func _GetWindowLongW(hWnd windows.HWND, nIndex int32) (int32, error) {
	r, _, e := procGetWindowLongW.Call(uintptr(hWnd), uintptr(nIndex))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("glfw: GetWindowLongW failed: %w", e)
	}
	return int32(r), nil
}

func _GetWindowPlacement(hWnd windows.HWND) (_WINDOWPLACEMENT, error) {
	var wp _WINDOWPLACEMENT
	wp.length = uint32(unsafe.Sizeof(wp))

	r, _, e := procGetWindowPlacement.Call(uintptr(hWnd), uintptr(unsafe.Pointer(&wp)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return _WINDOWPLACEMENT{}, fmt.Errorf("glfw: GetWindowPlacement failed: %w", e)
	}
	return wp, nil
}

func _GetWindowRect(hWnd windows.HWND) (_RECT, error) {
	var rect _RECT
	r, _, e := procGetWindowRect.Call(uintptr(hWnd), uintptr(unsafe.Pointer(&rect)))
	if int(r) == 0 {
		return _RECT{}, fmt.Errorf("glfw: GetWindowRect failed: %w", e)
	}
	return rect, nil
}

func _IsIconic(hWnd windows.HWND) bool {
	r, _, _ := procIsIconic.Call(uintptr(hWnd))
	return int32(r) != 0
}

func _IsWindowVisible(hWnd windows.HWND) bool {
	r, _, _ := procIsWindowVisible.Call(uintptr(hWnd))
	return int32(r) != 0
}

func _IsZoomed(hWnd windows.HWND) bool {
	r, _, _ := procIsZoomed.Call(uintptr(hWnd))
	return int32(r) != 0
}

func _LoadCursorW(hInstance _HINSTANCE, lpCursorName uintptr) (_HCURSOR, error) {
	r, _, e := procLoadCursorW.Call(uintptr(hInstance), lpCursorName)
	if _HCURSOR(r) == 0 {
		return 0, fmt.Errorf("glfw: LoadCursorW: %w", e)
	}
	return _HCURSOR(r), nil
}

func _LoadImageW(hInst _HINSTANCE, name uintptr, typ uint32, cx int32, cy int32, fuLoad uint32) (windows.Handle, error) {
	r, _, e := procLoadImageW.Call(uintptr(hInst), name, uintptr(typ), uintptr(cx), uintptr(cy), uintptr(fuLoad))
	if windows.Handle(r) == 0 {
		return 0, fmt.Errorf("glfw: LoadImageW: %w", e)
	}
	return windows.Handle(r), nil
}

func _MapVirtualKeyW(uCode uint32, uMapType uint32) uint32 {
	r, _, _ := procMapVirtualKeyW.Call(uintptr(uCode), uintptr(uMapType))
	return uint32(r)
}

func _MonitorFromWindow(hwnd windows.HWND, dwFlags uint32) _HMONITOR {
	r, _, _ := procMonitorFromWindow.Call(uintptr(hwnd), uintptr(dwFlags))
	return _HMONITOR(r)
}

func _MoveWindow(hWnd windows.HWND, x, y, nWidth, nHeight int32, repaint bool) error {
	var bRepaint uintptr
	if repaint {
		bRepaint = 1
	}
	r, _, e := procMoveWindow.Call(uintptr(hWnd), uintptr(x), uintptr(y), uintptr(nWidth), uintptr(nHeight), bRepaint)
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: MoveWindow: %w", e)
	}
	return nil
}

func _MsgWaitForMultipleObjects(nCount uint32, pHandles *windows.Handle, waitAll bool, dwMilliseconds uint32, dwWakeMask uint32) (uint32, error) {
	var fWaitAll uintptr
	if waitAll {
		fWaitAll = 1
	}
	r, _, e := procMsgWaitForMultipleObjects.Call(uintptr(nCount), uintptr(unsafe.Pointer(pHandles)), fWaitAll, uintptr(dwMilliseconds), uintptr(dwWakeMask))
	if uint32(r) == _WAIT_FAILED {
		return 0, fmt.Errorf("glfw: MsgWaitForMultipleObjects failed: %w", e)
	}
	return uint32(r), nil
}

func _OffsetRect(lprect *_RECT, dx int32, dy int32) bool {
	r, _, _ := procOffsetRect.Call(uintptr(unsafe.Pointer(lprect)), uintptr(dx), uintptr(dy))
	return int32(r) != 0
}

func _PeekMessageW(lpMsg *_MSG, hWnd windows.HWND, wMsgFilterMin uint32, wMsgFilterMax uint32, wRemoveMsg uint32) bool {
	r, _, _ := procPeekMessageW.Call(uintptr(unsafe.Pointer(lpMsg)), uintptr(hWnd), uintptr(wMsgFilterMin), uintptr(wMsgFilterMax), uintptr(wRemoveMsg))
	return int32(r) != 0
}

func _PostMessageW(hWnd windows.HWND, msg uint32, wParam _WPARAM, lParam _LPARAM) error {
	r, _, e := procPostMessageW.Call(uintptr(hWnd), uintptr(msg), uintptr(wParam), uintptr(lParam))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: PostMessageW failed: %w", e)
	}
	return nil
}

func _PtInRect(lprc *_RECT, pt _POINT) bool {
	var r uintptr
	if unsafe.Sizeof(uintptr(0)) == unsafe.Sizeof(uint64(0)) {
		r, _, _ = procPtInRect.Call(uintptr(unsafe.Pointer(lprc)), uintptr(pt.x)|uintptr(pt.y)<<32)
	} else {
		switch runtime.GOARCH {
		case "386":
			r, _, _ = procPtInRect.Call(uintptr(unsafe.Pointer(lprc)), uintptr(pt.x), uintptr(pt.y))
		case "arm":
			// Adjust the alignment for ARM.
			r, _, _ = procPtInRect.Call(uintptr(unsafe.Pointer(lprc)), 0, uintptr(pt.x), uintptr(pt.y))
		default:
			panic(fmt.Sprintf("glfw: GOARCH=%s is not supported", runtime.GOARCH))
		}
	}
	return int32(r) != 0
}

func _RegisterClassExW(unnamedParam1 *_WNDCLASSEXW) (_ATOM, error) {
	r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(unnamedParam1)))
	if _ATOM(r) == 0 {
		return 0, fmt.Errorf("glfw: RegisterClassExW failed: %w", e)
	}
	return _ATOM(r), nil
}

func _RegisterDeviceNotificationW(hRecipient windows.Handle, notificationFilter unsafe.Pointer, flags uint32) (_HDEVNOTIFY, error) {
	r, _, e := procRegisterDeviceNotificationW.Call(uintptr(hRecipient), uintptr(notificationFilter), uintptr(flags))
	if _HDEVNOTIFY(r) == 0 {
		return 0, fmt.Errorf("glfw: RegisterDeviceNotificationW failed: %w", e)
	}
	return _HDEVNOTIFY(r), nil
}

func _RegisterRawInputDevices(pRawInputDevices []_RAWINPUTDEVICE) error {
	var rawInputDevices unsafe.Pointer
	if len(pRawInputDevices) > 0 {
		rawInputDevices = unsafe.Pointer(&pRawInputDevices[0])
	}
	r, _, e := procRegisterRawInputDevices.Call(uintptr(rawInputDevices), uintptr(len(pRawInputDevices)), unsafe.Sizeof(_RAWINPUTDEVICE{}))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: RegisterRawInputDevices failed: %w", e)
	}
	return nil
}

func _ReleaseCapture() error {
	r, _, e := procReleaseCapture.Call()
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: ReleaseCapture failed: %w", e)
	}
	return nil
}

func _ReleaseDC(hWnd windows.HWND, hDC _HDC) int32 {
	r, _, _ := procReleaseDC.Call(uintptr(hWnd), uintptr(hDC))
	return int32(r)
}

func _ScreenToClient(hWnd windows.HWND, lpPoint *_POINT) error {
	r, _, e := procScreenToClient.Call(uintptr(hWnd), uintptr(unsafe.Pointer(lpPoint)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: ScreenToClient failed: %w", e)
	}
	return nil
}

func _SendMessageW(hWnd windows.HWND, msg uint32, wParam _WPARAM, lParam _LPARAM) _LRESULT {
	r, _, _ := procSendMessageW.Call(uintptr(hWnd), uintptr(msg), uintptr(wParam), uintptr(lParam))
	return _LRESULT(r)
}

func _SetCapture(hWnd windows.HWND) windows.HWND {
	r, _, _ := procSetCapture.Call(uintptr(hWnd))
	return windows.HWND(r)
}

func _SetCursor(hCursor _HCURSOR) _HCURSOR {
	r, _, _ := procSetCursor.Call(uintptr(hCursor))
	return _HCURSOR(r)
}

func _SetCursorPos(x, y int32) error {
	r, _, e := procSetCursorPos.Call(uintptr(x), uintptr(y))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: SetCursorPos failed: %w", e)
	}
	return nil
}

func _SetFocus(hWnd windows.HWND) (windows.HWND, error) {
	r, _, e := procSetFocus.Call(uintptr(hWnd))
	if windows.HWND(r) == 0 {
		return 0, fmt.Errorf("glfw: SetFocus failed: %w", e)
	}
	return windows.HWND(r), nil
}

func _SetForegroundWindow(hWnd windows.HWND) bool {
	r, _, _ := procSetForegroundWindow.Call(uintptr(hWnd))
	return int32(r) != 0
}

func _SetLayeredWindowAttributes(hwnd windows.HWND, crKey _COLORREF, bAlpha byte, dwFlags uint32) error {
	r, _, e := procSetLayeredWindowAttributes.Call(uintptr(hwnd), uintptr(crKey), uintptr(bAlpha), uintptr(dwFlags))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: SetLayeredWindowAttributes failed: %w", e)
	}
	return nil
}

func _SetPixelFormat(hdc _HDC, format int32, ppfd *_PIXELFORMATDESCRIPTOR) error {
	r, _, e := procSetPixelFormat.Call(uintptr(hdc), uintptr(format), uintptr(unsafe.Pointer(ppfd)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: SetPixelFormat failed: %w", e)
	}
	return nil
}

func _SetProcessDPIAware() bool {
	r, _, _ := procSetProcessDPIAware.Call()
	return int32(r) != 0
}

func _SetProcessDpiAwareness(value _PROCESS_DPI_AWARENESS) error {
	r, _, _ := procSetProcessDpiAwareness.Call(uintptr(value))
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("glfw: SetProcessDpiAwareness failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func _SetProcessDpiAwarenessContext(value _DPI_AWARENESS_CONTEXT) error {
	r, _, e := procSetProcessDpiAwarenessContext.Call(uintptr(value))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: SetProcessDpiAwarenessContext failed: %w", e)
	}
	return nil
}

func _SetThreadExecutionState(esFlags _EXECUTION_STATE) _EXECUTION_STATE {
	r, _, _ := procSetThreadExecutionState.Call(uintptr(esFlags))
	return _EXECUTION_STATE(r)
}

func _SetWindowLongW(hWnd windows.HWND, nIndex int32, dwNewLong int32) (int32, error) {
	r, _, e := procSetWindowLongW.Call(uintptr(hWnd), uintptr(nIndex), uintptr(dwNewLong))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("glfw: SetWindowLongW failed: %w", e)
	}
	return int32(r), nil
}

func _SetWindowPlacement(hWnd windows.HWND, lpwndpl *_WINDOWPLACEMENT) error {
	r, _, e := procSetWindowPlacement.Call(uintptr(hWnd), uintptr(unsafe.Pointer(lpwndpl)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: SetWindowPlacement failed: %w", e)
	}
	return nil
}

func _SetWindowPos(hWnd windows.HWND, hWndInsertAfter windows.HWND, x, y, cx, cy int32, uFlags uint32) error {
	r, _, e := procSetWindowPos.Call(uintptr(hWnd), uintptr(hWndInsertAfter), uintptr(x), uintptr(y), uintptr(cx), uintptr(cy), uintptr(uFlags))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: SetWindowPos failed: %w", e)
	}
	return nil
}

func _SetWindowTextW(hWnd windows.HWND, str string) error {
	// An empty string is also a valid value. Always create an uint16 pointer.
	lpString, err := windows.UTF16PtrFromString(str)
	if err != nil {
		panic("glfw: str must not include a NUL character")
	}

	r, _, e := procSetWindowTextW.Call(uintptr(hWnd), uintptr(unsafe.Pointer(lpString)))
	runtime.KeepAlive(lpString)

	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: SetWindowTextW failed: %w", e)
	}
	return nil
}

func _ShowWindow(hWnd windows.HWND, nCmdShow int32) bool {
	r, _, _ := procShowWindow.Call(uintptr(hWnd), uintptr(nCmdShow))
	return int32(r) != 0
}

func _SwapBuffers(unnamedParam1 _HDC) error {
	r, _, e := procSwapBuffers.Call(uintptr(unnamedParam1))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: SwapBuffers failed: %w", e)
	}
	return nil
}

func _SystemParametersInfoW(uiAction uint32, uiParam uint32, pvParam uintptr, fWinIni uint32) error {
	r, _, e := procSystemParametersInfoW.Call(uintptr(uiAction), uintptr(uiParam), pvParam, uintptr(fWinIni))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: SystemParametersInfoW failed: %w", e)
	}
	return nil
}

func _TlsAlloc() (uint32, error) {
	r, _, e := procTlsAlloc.Call()
	if uint32(r) == _TLS_OUT_OF_INDEXES {
		return 0, fmt.Errorf("glfw: TlsAlloc failed: %w", e)
	}
	return uint32(r), nil
}

func _TlsFree(dwTlsIndex uint32) error {
	r, _, e := procTlsFree.Call(uintptr(dwTlsIndex))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: TlsFree failed: %w", e)
	}
	return nil
}

func _TlsGetValue(dwTlsIndex uint32) (uintptr, error) {
	r, _, e := procTlsGetValue.Call(uintptr(dwTlsIndex))
	if r == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return 0, fmt.Errorf("glfw: TlsGetValue failed: %w", e)
	}
	return r, nil
}

func _TlsSetValue(dwTlsIndex uint32, lpTlsValue uintptr) error {
	r, _, e := procTlsSetValue.Call(uintptr(dwTlsIndex), lpTlsValue)
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: TlsSetValue failed: %w", e)
	}
	return nil
}

func _ToUnicode(wVirtualKey uint32, wScanCode uint32, keyState []byte, buff []uint16, cchBuff int32, wFlags uint32) int32 {
	var lpKeyState *byte
	if len(keyState) > 0 {
		lpKeyState = &keyState[0]
	}
	var pwszBuff *uint16
	if len(buff) > 0 {
		pwszBuff = &buff[0]
	}

	r, _, _ := procToUnicode.Call(uintptr(wVirtualKey), uintptr(wScanCode), uintptr(unsafe.Pointer(lpKeyState)),
		uintptr(unsafe.Pointer(pwszBuff)), uintptr(cchBuff), uintptr(wFlags))
	runtime.KeepAlive(lpKeyState)
	runtime.KeepAlive(pwszBuff)

	return int32(r)
}

func _TranslateMessage(lpMsg *_MSG) bool {
	r, _, _ := procTranslateMessage.Call(uintptr(unsafe.Pointer(lpMsg)))
	return int32(r) != 0
}

func _TrackMouseEvent(lpEventTrack *_TRACKMOUSEEVENT) error {
	r, _, e := procTrackMouseEvent.Call(uintptr(unsafe.Pointer(lpEventTrack)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: TrackMouseEvent failed: %w", e)
	}
	return nil
}

func _UnregisterClassW(className string, hInstance _HINSTANCE) error {
	var lpClassName *uint16
	if className != "" {
		var err error
		lpClassName, err = windows.UTF16PtrFromString(className)
		if err != nil {
			panic("glfw: class name must not include a NUL character")
		}
	}

	r, _, e := procUnregisterClassW.Call(uintptr(unsafe.Pointer(lpClassName)), uintptr(hInstance))
	runtime.KeepAlive(lpClassName)

	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: UnregisterClassW failed: %w", e)
	}
	return nil
}

func _UnregisterDeviceNotification(handle _HDEVNOTIFY) error {
	r, _, e := procUnregisterDeviceNotification.Call(uintptr(handle))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: UnregisterDeviceNotification failed: %w", e)
	}
	return nil
}

func _WaitMessage() error {
	r, _, e := procWaitMessage.Call()
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: WaitMessage failed: %w", e)
	}
	return nil
}

func wglCreateContext(unnamedParam1 _HDC) (_HGLRC, error) {
	r, _, e := procWGLCreateContext.Call(uintptr(unnamedParam1))
	if _HGLRC(r) == 0 {
		return 0, fmt.Errorf("glfw: wglCreateContext failed: %w", e)
	}
	return _HGLRC(r), nil
}

func wglCreateContextAttribsARB(hDC _HDC, hshareContext _HGLRC, attribList *int32) (_HGLRC, error) {
	r, _, e := syscall.Syscall(procWGLCreateContextAttribsARB, 3, uintptr(hDC), uintptr(hshareContext), uintptr(unsafe.Pointer(attribList)))
	if _HGLRC(r) == 0 {
		// TODO: Show more detailed error? See the original implementation.
		return 0, fmt.Errorf("glfw: wglCreateContextAttribsARB failed: %w", e)
	}
	return _HGLRC(r), nil
}

func wglDeleteContext(unnamedParam1 _HGLRC) error {
	r, _, e := procWGLDeleteContext.Call(uintptr(unnamedParam1))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: wglDeleteContext failed: %w", e)
	}
	return nil
}

func wglGetCurrentContext() _HGLRC {
	r, _, _ := procWGLGetCurrentContext.Call()
	return _HGLRC(r)
}

func wglGetCurrentDC() _HDC {
	r, _, _ := procWGLGetCurrentDC.Call()
	return _HDC(r)
}

func wglGetExtensionsStringARB(hdc _HDC) string {
	r, _, _ := syscall.Syscall(procWGLGetExtensionsStringARB, 1, uintptr(hdc), 0, 0)
	return windows.BytePtrToString((*byte)(unsafe.Pointer(r)))
}

func wglGetExtensionsStringARB_Available() bool {
	return procWGLGetExtensionsStringARB != 0
}

func wglGetExtensionsStringEXT() string {
	r, _, _ := syscall.Syscall(procWGLGetExtensionsStringEXT, 0, 0, 0, 0)
	return windows.BytePtrToString((*byte)(unsafe.Pointer(r)))
}

func wglGetExtensionsStringEXT_Available() bool {
	return procWGLGetExtensionsStringEXT != 0
}

func wglGetPixelFormatAttribivARB(hdc _HDC, iPixelFormat int32, iLayerPlane int32, nAttributes uint32, piAttributes *int32, piValues *int32) error {
	r, _, e := syscall.Syscall6(procWGLGetPixelFormatAttribivARB, 6, uintptr(hdc), uintptr(iPixelFormat), uintptr(iLayerPlane), uintptr(nAttributes), uintptr(unsafe.Pointer(piAttributes)), uintptr(unsafe.Pointer(piValues)))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: wglGetPixelFormatAttribivARB failed: %w", e)
	}
	return nil
}

func wglGetProcAddress(unnamedParam1 string) uintptr {
	ptr, err := windows.BytePtrFromString(unnamedParam1)
	if err != nil {
		panic("glfw: unnamedParam1 must not include a NUL character")
	}
	r, _, _ := procWGLGetProcAddress.Call(uintptr(unsafe.Pointer(ptr)))
	return r
}

func wglMakeCurrent(unnamedParam1 _HDC, unnamedParam2 _HGLRC) error {
	r, _, e := procWGLMakeCurrent.Call(uintptr(unnamedParam1), uintptr(unnamedParam2))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: wglMakeCurrent failed: %w", e)
	}
	return nil
}

func wglShareLists(unnamedParam1 _HGLRC, unnamedParam2 _HGLRC) error {
	r, _, e := procWGLShareLists.Call(uintptr(unnamedParam1), uintptr(unnamedParam2))
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: wglShareLists failed: %w", e)
	}
	return nil
}

func wglSwapIntervalEXT(interval int32) error {
	r, _, e := syscall.Syscall(procWGLSwapIntervalEXT, 1, uintptr(interval), 0, 0)
	if int32(r) == 0 && !errors.Is(e, windows.ERROR_SUCCESS) {
		return fmt.Errorf("glfw: wglSwapIntervalEXT failed: %w", e)
	}
	return nil
}

func _WindowFromPoint(point _POINT) windows.HWND {
	var r uintptr
	if unsafe.Sizeof(uintptr(0)) == unsafe.Sizeof(uint64(0)) {
		r, _, _ = procWindowFromPoint.Call(uintptr(point.x) | uintptr(point.y)<<32)
	} else {
		r, _, _ = procWindowFromPoint.Call(uintptr(point.x), uintptr(point.y))
	}
	return windows.HWND(r)
}
