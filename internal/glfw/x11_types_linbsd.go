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
	"unsafe"
)

// This file mirrors the C struct layouts of Xlib and the X extension
// libraries. Field order and types must reproduce the C layouts exactly;
// x11_types_linbsd_test.go verifies sizes and offsets against golden values
// extracted from the C headers with offsetof/sizeof.
//
// Type mapping: C int/Bool -> int32, C long/unsigned long (and the XID
// family) -> _Clong/_Culong, C pointers -> uintptr. Pointers received from
// Xlib are C-owned memory; they are stored as uintptr and must be released
// with XFree (or the extension's free function) where the X11 protocol
// requires it.
//
// The X11 resource ID types Window, Cursor, Pixmap, and Colormap are all
// represented as XID; the names Window and Cursor are already taken by GLFW
// types in this package.

// C long and unsigned long are pointer-sized on every Unix ABI (ILP32 or
// LP64), as are Go's int and uint.
type (
	_Clong  = int
	_Culong = uint
)

type (
	_XID     = _Culong
	_Atom    = _Culong
	_Time    = _Culong
	_KeySym  = _Culong
	_KeyCode = uint8

	_VisualID  = _Culong
	_RROutput  = _XID
	_RRCrtc    = _XID
	_RRMode    = _XID
	_Rotation  = uint16
	_XContext  = int32
	_XIMStyle  = _Culong
	_XrmQuark  = int32
	_XcursorID = uint32

	// Region is an opaque pointer (the _XRegion struct is private to Xlib).
	_Region = uintptr
)

// XEvent is the Xlib event union: 24 C longs, the first int of which is the
// event type. Access the contents through the typed view methods, which
// mirror the C union members.
type _XEvent struct {
	data [24]_Clong
}

func (e *_XEvent) EventType() int32                { return *(*int32)(unsafe.Pointer(e)) }
func (e *_XEvent) xany() *_XAnyEvent               { return (*_XAnyEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xkey() *_XKeyEvent               { return (*_XKeyEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xbutton() *_XButtonEvent         { return (*_XButtonEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xmotion() *_XMotionEvent         { return (*_XMotionEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xcrossing() *_XCrossingEvent     { return (*_XCrossingEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xfocus() *_XFocusChangeEvent     { return (*_XFocusChangeEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xexpose() *_XExposeEvent         { return (*_XExposeEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xvisibility() *_XVisibilityEvent { return (*_XVisibilityEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xdestroywindow() *_XDestroyWindowEvent {
	return (*_XDestroyWindowEvent)(unsafe.Pointer(e))
}
func (e *_XEvent) xunmap() *_XUnmapEvent         { return (*_XUnmapEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xmap() *_XMapEvent             { return (*_XMapEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xreparent() *_XReparentEvent   { return (*_XReparentEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xconfigure() *_XConfigureEvent { return (*_XConfigureEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xproperty() *_XPropertyEvent   { return (*_XPropertyEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xselectionrequest() *_XSelectionRequestEvent {
	return (*_XSelectionRequestEvent)(unsafe.Pointer(e))
}
func (e *_XEvent) xselection() *_XSelectionEvent { return (*_XSelectionEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xselectionclear() *_XSelectionClearEvent {
	return (*_XSelectionClearEvent)(unsafe.Pointer(e))
}
func (e *_XEvent) xclient() *_XClientMessageEvent  { return (*_XClientMessageEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xcookie() *_XGenericEventCookie  { return (*_XGenericEventCookie)(unsafe.Pointer(e)) }
func (e *_XEvent) xkbAny() *_XkbAnyEvent           { return (*_XkbAnyEvent)(unsafe.Pointer(e)) }
func (e *_XEvent) xkbState() *_XkbStateNotifyEvent { return (*_XkbStateNotifyEvent)(unsafe.Pointer(e)) }

type _XAnyEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    _XID
}

type _XKeyEvent struct {
	Type       int32
	Serial     _Culong
	SendEvent  int32
	Display    uintptr
	Window     _XID
	Root       _XID
	Subwindow  _XID
	Time       _Time
	X          int32
	Y          int32
	XRoot      int32
	YRoot      int32
	State      uint32
	Keycode    uint32
	SameScreen int32
}

type _XButtonEvent struct {
	Type       int32
	Serial     _Culong
	SendEvent  int32
	Display    uintptr
	Window     _XID
	Root       _XID
	Subwindow  _XID
	Time       _Time
	X          int32
	Y          int32
	XRoot      int32
	YRoot      int32
	State      uint32
	Button     uint32
	SameScreen int32
}

type _XMotionEvent struct {
	Type       int32
	Serial     _Culong
	SendEvent  int32
	Display    uintptr
	Window     _XID
	Root       _XID
	Subwindow  _XID
	Time       _Time
	X          int32
	Y          int32
	XRoot      int32
	YRoot      int32
	State      uint32
	IsHint     uint8
	SameScreen int32
}

type _XCrossingEvent struct {
	Type       int32
	Serial     _Culong
	SendEvent  int32
	Display    uintptr
	Window     _XID
	Root       _XID
	Subwindow  _XID
	Time       _Time
	X          int32
	Y          int32
	XRoot      int32
	YRoot      int32
	Mode       int32
	Detail     int32
	SameScreen int32
	Focus      int32
	State      uint32
}

type _XFocusChangeEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    _XID
	Mode      int32
	Detail    int32
}

type _XExposeEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    _XID
	X         int32
	Y         int32
	Width     int32
	Height    int32
	Count     int32
}

type _XVisibilityEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    _XID
	State     int32
}

type _XDestroyWindowEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Event     _XID
	Window    _XID
}

type _XUnmapEvent struct {
	Type          int32
	Serial        _Culong
	SendEvent     int32
	Display       uintptr
	Event         _XID
	Window        _XID
	FromConfigure int32
}

type _XMapEvent struct {
	Type             int32
	Serial           _Culong
	SendEvent        int32
	Display          uintptr
	Event            _XID
	Window           _XID
	OverrideRedirect int32
}

type _XReparentEvent struct {
	Type             int32
	Serial           _Culong
	SendEvent        int32
	Display          uintptr
	Event            _XID
	Window           _XID
	Parent           _XID
	X                int32
	Y                int32
	OverrideRedirect int32
}

type _XConfigureEvent struct {
	Type             int32
	Serial           _Culong
	SendEvent        int32
	Display          uintptr
	Event            _XID
	Window           _XID
	X                int32
	Y                int32
	Width            int32
	Height           int32
	BorderWidth      int32
	Above            _XID
	OverrideRedirect int32
}

type _XPropertyEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    _XID
	Atom      _Atom
	Time      _Time
	State     int32
}

type _XSelectionRequestEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Owner     _XID
	Requestor _XID
	Selection _Atom
	Target    _Atom
	Property  _Atom
	Time      _Time
}

type _XSelectionEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Requestor _XID
	Selection _Atom
	Target    _Atom
	Property  _Atom
	Time      _Time
}

type _XSelectionClearEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    _XID
	Selection _Atom
	Time      _Time
}

type _XClientMessageEvent struct {
	Type        int32
	Serial      _Culong
	SendEvent   int32
	Display     uintptr
	Window      _XID
	MessageType _Atom
	Format      int32
	// Data is the b/s/l union; the l member (5 C longs) covers it entirely.
	Data [5]_Clong
}

type _XGenericEventCookie struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Extension int32
	Evtype    int32
	Cookie    uint32
	Data      uintptr
}

type _XErrorEvent struct {
	Type        int32
	Display     uintptr
	Resourceid  _XID
	Serial      _Culong
	ErrorCode   uint8
	RequestCode uint8
	MinorCode   uint8
}

type _XSetWindowAttributes struct {
	BackgroundPixmap   _XID
	BackgroundPixel    _Culong
	BorderPixmap       _XID
	BorderPixel        _Culong
	BitGravity         int32
	WinGravity         int32
	BackingStore       int32
	BackingPlanes      _Culong
	BackingPixel       _Culong
	SaveUnder          int32
	EventMask          _Clong
	DoNotPropagateMask _Clong
	OverrideRedirect   int32
	Colormap           _XID
	Cursor             _XID
}

type _XWindowAttributes struct {
	X                  int32
	Y                  int32
	Width              int32
	Height             int32
	BorderWidth        int32
	Depth              int32
	Visual             uintptr
	Root               _XID
	Class              int32
	BitGravity         int32
	WinGravity         int32
	BackingStore       int32
	BackingPlanes      _Culong
	BackingPixel       _Culong
	SaveUnder          int32
	Colormap           _XID
	MapInstalled       int32
	MapState           int32
	AllEventMasks      _Clong
	YourEventMask      _Clong
	DoNotPropagateMask _Clong
	OverrideRedirect   int32
	Screen             uintptr
}

// Visual is accessed only through its visualid (the XVisualIDFromVisual
// macro). The remaining fields are spelled out so the layout matches Xlib on
// both LP64 and ILP32 (a fixed byte pad would be wrong on 32-bit).
type _Visual struct {
	ExtData    uintptr
	Visualid   _VisualID
	Class      int32
	RedMask    _Culong
	GreenMask  _Culong
	BlueMask   _Culong
	BitsPerRGB int32
	MapEntries int32
}

type _XVisualInfo struct {
	Visual       uintptr
	Visualid     _VisualID
	Screen       int32
	Depth        int32
	Class        int32
	RedMask      _Culong
	GreenMask    _Culong
	BlueMask     _Culong
	ColormapSize int32
	BitsPerRGB   int32
}

type _XSizeHints struct {
	Flags      _Clong
	X          int32
	Y          int32
	Width      int32
	Height     int32
	MinWidth   int32
	MinHeight  int32
	MaxWidth   int32
	MaxHeight  int32
	WidthInc   int32
	HeightInc  int32
	MinAspect  struct{ X, Y int32 }
	MaxAspect  struct{ X, Y int32 }
	BaseWidth  int32
	BaseHeight int32
	WinGravity int32
}

type _XWMHints struct {
	Flags        _Clong
	Input        int32
	InitialState int32
	IconPixmap   _XID
	IconWindow   _XID
	IconX        int32
	IconY        int32
	IconMask     _XID
	WindowGroup  _XID
}

type _XClassHint struct {
	ResName  uintptr
	ResClass uintptr
}

type _XIMStyles struct {
	CountStyles     uint16
	SupportedStyles uintptr // *XIMStyle
}

type _XRectangle struct {
	X      int16
	Y      int16
	Width  uint16
	Height uint16
}

// Xkb

type _XkbDescRec struct {
	Dpy        uintptr
	Flags      uint16
	DeviceSpec uint16
	MinKeyCode _KeyCode
	MaxKeyCode _KeyCode
	Ctrls      uintptr
	Server     uintptr
	Map        uintptr
	Indicators uintptr
	Names      uintptr // *XkbNamesRec
	Compat     uintptr
	Geom       uintptr
}

// XkbNamesRec mirrors the XkbNamesRec field types (rather than fixed byte
// pads) so the offsets the key-table setup reads (Keys, KeyAliases, NumKeys,
// NumKeyAliases) are correct on both LP64 and ILP32. The leading run is 57
// word-sized Atom fields: keycodes, geometry, symbols, types, compat,
// vmods[16], indicators[32], and groups[4].
type _XkbNamesRec struct {
	_             [57]_Atom
	Keys          uintptr // *XkbKeyNameRec
	KeyAliases    uintptr // *XkbKeyAliasRec
	RadioGroups   uintptr // char*
	PhysSymbols   _Atom
	NumKeys       uint8
	NumKeyAliases uint8
	NumRG         uint16
}

type _XkbKeyNameRec struct {
	Name [4]byte
}

type _XkbKeyAliasRec struct {
	Real  [4]byte
	Alias [4]byte
}

// XkbStateRec exposes only the group field; the rest stays as padding.
type _XkbStateRec struct {
	Group uint8
	_     [17]byte
}

type _XkbAnyEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Time      _Time
	XkbType   int32
	Device    int32
}

type _XkbStateNotifyEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Time      _Time
	XkbType   int32
	Device    int32
	Changed   uint32
	Group     int32
	_         [48]byte
}

// Xrender

type _XRenderDirectFormat struct {
	Red       int16
	RedMask   int16
	Green     int16
	GreenMask int16
	Blue      int16
	BlueMask  int16
	Alpha     int16
	AlphaMask int16
}

type _XRenderPictFormat struct {
	ID       _XID
	Type     int32
	Depth    int32
	Direct   _XRenderDirectFormat
	Colormap _XID
}

// RandR

type _XRRModeInfo struct {
	ID         _RRMode
	Width      uint32
	Height     uint32
	DotClock   _Culong
	HSyncStart uint32
	HSyncEnd   uint32
	HTotal     uint32
	HSkew      uint32
	VSyncStart uint32
	VSyncEnd   uint32
	VTotal     uint32
	Name       uintptr
	NameLength uint32
	ModeFlags  _Culong
}

type _XRRScreenResources struct {
	Timestamp       _Time
	ConfigTimestamp _Time
	Ncrtc           int32
	Crtcs           uintptr // *RRCrtc
	Noutput         int32
	Outputs         uintptr // *RROutput
	Nmode           int32
	Modes           uintptr // *XRRModeInfo
}

type _XRROutputInfo struct {
	Timestamp     _Time
	Crtc          _RRCrtc
	Name          uintptr // *byte
	NameLen       int32
	MmWidth       _Culong
	MmHeight      _Culong
	Connection    uint16
	SubpixelOrder uint16
	Ncrtc         int32
	Crtcs         uintptr // *RRCrtc
	Nclone        int32
	Clones        uintptr // *RROutput
	Nmode         int32
	Npreferred    int32
	Modes         uintptr // *RRMode
}

type _XRRCrtcInfo struct {
	Timestamp _Time
	X         int32
	Y         int32
	Width     uint32
	Height    uint32
	Mode      _RRMode
	Rotation  _Rotation
	Noutput   int32
	Outputs   uintptr // *RROutput
	Rotations _Rotation
	Npossible int32
	Possible  uintptr // *RROutput
}

// Xinerama

type _XineramaScreenInfo struct {
	ScreenNumber int32
	XOrg         int16
	YOrg         int16
	Width        int16
	Height       int16
}

// Xcursor

type _XcursorImage struct {
	Version uint32
	Size    uint32
	Width   uint32
	Height  uint32
	Xhot    uint32
	Yhot    uint32
	Delay   uint32
	Pixels  uintptr // *uint32 (XcursorPixel)
}

// XInput2

type _XIEventMask struct {
	Deviceid int32
	MaskLen  int32
	Mask     uintptr // *byte
}

type _XIValuatorState struct {
	MaskLen int32
	Mask    uintptr // *byte
	Values  uintptr // *float64
}

type _XIRawEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Extension int32
	Evtype    int32
	Time      _Time
	Deviceid  int32
	Sourceid  int32
	Detail    int32
	Flags     int32
	Valuators _XIValuatorState
	RawValues uintptr // *float64
}
