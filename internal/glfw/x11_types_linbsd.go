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
	XID     = _Culong
	Atom    = _Culong
	Time    = _Culong
	KeySym  = _Culong
	KeyCode = uint8

	VisualID  = _Culong
	RROutput  = XID
	RRCrtc    = XID
	RRMode    = XID
	Rotation  = uint16
	XContext  = int32
	XIMStyle  = _Culong
	XrmQuark  = int32
	XcursorID = uint32

	// Region is an opaque pointer (the _XRegion struct is private to Xlib).
	Region = uintptr
)

// XEvent is the Xlib event union: 24 C longs, the first int of which is the
// event type. Access the contents through the typed view methods, which
// mirror the C union members.
type XEvent struct {
	data [24]_Clong
}

func (e *XEvent) EventType() int32               { return *(*int32)(unsafe.Pointer(e)) }
func (e *XEvent) xany() *XAnyEvent               { return (*XAnyEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xkey() *XKeyEvent               { return (*XKeyEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xbutton() *XButtonEvent         { return (*XButtonEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xmotion() *XMotionEvent         { return (*XMotionEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xcrossing() *XCrossingEvent     { return (*XCrossingEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xfocus() *XFocusChangeEvent     { return (*XFocusChangeEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xexpose() *XExposeEvent         { return (*XExposeEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xvisibility() *XVisibilityEvent { return (*XVisibilityEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xdestroywindow() *XDestroyWindowEvent {
	return (*XDestroyWindowEvent)(unsafe.Pointer(e))
}
func (e *XEvent) xunmap() *XUnmapEvent         { return (*XUnmapEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xmap() *XMapEvent             { return (*XMapEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xreparent() *XReparentEvent   { return (*XReparentEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xconfigure() *XConfigureEvent { return (*XConfigureEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xproperty() *XPropertyEvent   { return (*XPropertyEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xselectionrequest() *XSelectionRequestEvent {
	return (*XSelectionRequestEvent)(unsafe.Pointer(e))
}
func (e *XEvent) xselection() *XSelectionEvent { return (*XSelectionEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xselectionclear() *XSelectionClearEvent {
	return (*XSelectionClearEvent)(unsafe.Pointer(e))
}
func (e *XEvent) xclient() *XClientMessageEvent  { return (*XClientMessageEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xcookie() *XGenericEventCookie  { return (*XGenericEventCookie)(unsafe.Pointer(e)) }
func (e *XEvent) xkbAny() *XkbAnyEvent           { return (*XkbAnyEvent)(unsafe.Pointer(e)) }
func (e *XEvent) xkbState() *XkbStateNotifyEvent { return (*XkbStateNotifyEvent)(unsafe.Pointer(e)) }

type XAnyEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    XID
}

type XKeyEvent struct {
	Type       int32
	Serial     _Culong
	SendEvent  int32
	Display    uintptr
	Window     XID
	Root       XID
	Subwindow  XID
	Time       Time
	X          int32
	Y          int32
	XRoot      int32
	YRoot      int32
	State      uint32
	Keycode    uint32
	SameScreen int32
}

type XButtonEvent struct {
	Type       int32
	Serial     _Culong
	SendEvent  int32
	Display    uintptr
	Window     XID
	Root       XID
	Subwindow  XID
	Time       Time
	X          int32
	Y          int32
	XRoot      int32
	YRoot      int32
	State      uint32
	Button     uint32
	SameScreen int32
}

type XMotionEvent struct {
	Type       int32
	Serial     _Culong
	SendEvent  int32
	Display    uintptr
	Window     XID
	Root       XID
	Subwindow  XID
	Time       Time
	X          int32
	Y          int32
	XRoot      int32
	YRoot      int32
	State      uint32
	IsHint     uint8
	SameScreen int32
}

type XCrossingEvent struct {
	Type       int32
	Serial     _Culong
	SendEvent  int32
	Display    uintptr
	Window     XID
	Root       XID
	Subwindow  XID
	Time       Time
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

type XFocusChangeEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    XID
	Mode      int32
	Detail    int32
}

type XExposeEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    XID
	X         int32
	Y         int32
	Width     int32
	Height    int32
	Count     int32
}

type XVisibilityEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    XID
	State     int32
}

type XDestroyWindowEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Event     XID
	Window    XID
}

type XUnmapEvent struct {
	Type          int32
	Serial        _Culong
	SendEvent     int32
	Display       uintptr
	Event         XID
	Window        XID
	FromConfigure int32
}

type XMapEvent struct {
	Type             int32
	Serial           _Culong
	SendEvent        int32
	Display          uintptr
	Event            XID
	Window           XID
	OverrideRedirect int32
}

type XReparentEvent struct {
	Type             int32
	Serial           _Culong
	SendEvent        int32
	Display          uintptr
	Event            XID
	Window           XID
	Parent           XID
	X                int32
	Y                int32
	OverrideRedirect int32
}

type XConfigureEvent struct {
	Type             int32
	Serial           _Culong
	SendEvent        int32
	Display          uintptr
	Event            XID
	Window           XID
	X                int32
	Y                int32
	Width            int32
	Height           int32
	BorderWidth      int32
	Above            XID
	OverrideRedirect int32
}

type XPropertyEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    XID
	Atom      Atom
	Time      Time
	State     int32
}

type XSelectionRequestEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Owner     XID
	Requestor XID
	Selection Atom
	Target    Atom
	Property  Atom
	Time      Time
}

type XSelectionEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Requestor XID
	Selection Atom
	Target    Atom
	Property  Atom
	Time      Time
}

type XSelectionClearEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Window    XID
	Selection Atom
	Time      Time
}

type XClientMessageEvent struct {
	Type        int32
	Serial      _Culong
	SendEvent   int32
	Display     uintptr
	Window      XID
	MessageType Atom
	Format      int32
	// Data is the b/s/l union; the l member (5 C longs) covers it entirely.
	Data [5]_Clong
}

type XGenericEventCookie struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Extension int32
	Evtype    int32
	Cookie    uint32
	Data      uintptr
}

type XErrorEvent struct {
	Type        int32
	Display     uintptr
	Resourceid  XID
	Serial      _Culong
	ErrorCode   uint8
	RequestCode uint8
	MinorCode   uint8
}

type XSetWindowAttributes struct {
	BackgroundPixmap   XID
	BackgroundPixel    _Culong
	BorderPixmap       XID
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
	Colormap           XID
	Cursor             XID
}

type XWindowAttributes struct {
	X                  int32
	Y                  int32
	Width              int32
	Height             int32
	BorderWidth        int32
	Depth              int32
	Visual             uintptr
	Root               XID
	Class              int32
	BitGravity         int32
	WinGravity         int32
	BackingStore       int32
	BackingPlanes      _Culong
	BackingPixel       _Culong
	SaveUnder          int32
	Colormap           XID
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
type Visual struct {
	ExtData    uintptr
	Visualid   VisualID
	Class      int32
	RedMask    _Culong
	GreenMask  _Culong
	BlueMask   _Culong
	BitsPerRGB int32
	MapEntries int32
}

type XVisualInfo struct {
	Visual       uintptr
	Visualid     VisualID
	Screen       int32
	Depth        int32
	Class        int32
	RedMask      _Culong
	GreenMask    _Culong
	BlueMask     _Culong
	ColormapSize int32
	BitsPerRGB   int32
}

type XSizeHints struct {
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

type XWMHints struct {
	Flags        _Clong
	Input        int32
	InitialState int32
	IconPixmap   XID
	IconWindow   XID
	IconX        int32
	IconY        int32
	IconMask     XID
	WindowGroup  XID
}

type XClassHint struct {
	ResName  uintptr
	ResClass uintptr
}

type XIMStyles struct {
	CountStyles     uint16
	SupportedStyles uintptr // *XIMStyle
}

type XRectangle struct {
	X      int16
	Y      int16
	Width  uint16
	Height uint16
}

// Xkb

type XkbDescRec struct {
	Dpy        uintptr
	Flags      uint16
	DeviceSpec uint16
	MinKeyCode KeyCode
	MaxKeyCode KeyCode
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
type XkbNamesRec struct {
	_             [57]Atom
	Keys          uintptr // *XkbKeyNameRec
	KeyAliases    uintptr // *XkbKeyAliasRec
	RadioGroups   uintptr // char*
	PhysSymbols   Atom
	NumKeys       uint8
	NumKeyAliases uint8
	NumRG         uint16
}

type XkbKeyNameRec struct {
	Name [4]byte
}

type XkbKeyAliasRec struct {
	Real  [4]byte
	Alias [4]byte
}

// XkbStateRec exposes only the group field; the rest stays as padding.
type XkbStateRec struct {
	Group uint8
	_     [17]byte
}

type XkbAnyEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Time      Time
	XkbType   int32
	Device    int32
}

type XkbStateNotifyEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Time      Time
	XkbType   int32
	Device    int32
	Changed   uint32
	Group     int32
	_         [48]byte
}

// Xrender

type XRenderDirectFormat struct {
	Red       int16
	RedMask   int16
	Green     int16
	GreenMask int16
	Blue      int16
	BlueMask  int16
	Alpha     int16
	AlphaMask int16
}

type XRenderPictFormat struct {
	ID       XID
	Type     int32
	Depth    int32
	Direct   XRenderDirectFormat
	Colormap XID
}

// RandR

type XRRModeInfo struct {
	ID         RRMode
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

type XRRScreenResources struct {
	Timestamp       Time
	ConfigTimestamp Time
	Ncrtc           int32
	Crtcs           uintptr // *RRCrtc
	Noutput         int32
	Outputs         uintptr // *RROutput
	Nmode           int32
	Modes           uintptr // *XRRModeInfo
}

type XRROutputInfo struct {
	Timestamp     Time
	Crtc          RRCrtc
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

type XRRCrtcInfo struct {
	Timestamp Time
	X         int32
	Y         int32
	Width     uint32
	Height    uint32
	Mode      RRMode
	Rotation  Rotation
	Noutput   int32
	Outputs   uintptr // *RROutput
	Rotations Rotation
	Npossible int32
	Possible  uintptr // *RROutput
}

// Xinerama

type XineramaScreenInfo struct {
	ScreenNumber int32
	XOrg         int16
	YOrg         int16
	Width        int16
	Height       int16
}

// Xcursor

type XcursorImage struct {
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

type XIEventMask struct {
	Deviceid int32
	MaskLen  int32
	Mask     uintptr // *byte
}

type XIValuatorState struct {
	MaskLen int32
	Mask    uintptr // *byte
	Values  uintptr // *float64
}

type XIRawEvent struct {
	Type      int32
	Serial    _Culong
	SendEvent int32
	Display   uintptr
	Extension int32
	Evtype    int32
	Time      Time
	Deviceid  int32
	Sourceid  int32
	Detail    int32
	Flags     int32
	Valuators XIValuatorState
	RawValues uintptr // *float64
}
