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

package glfw_test

import (
	"math/bits"
	"testing"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

// The golden values are LP64 sizeof/offsetof results extracted from the C
// headers (Xlib.h, Xutil.h, XKBlib.h, Xrender.h, Xrandr.h, Xinerama.h,
// Xcursor.h, XInput2.h) with a C program. ILP32 golden values are not
// covered yet.

func TestXlibStructLayouts(t *testing.T) {
	if bits.UintSize != 64 {
		t.Skip("the golden values are for LP64 platforms")
	}
	for _, tt := range []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"sizeof XEvent", unsafe.Sizeof(glfw.XEvent{}), 192},

		{"XAnyEvent.Type", unsafe.Offsetof(glfw.XAnyEvent{}.Type), 0},
		{"XAnyEvent.Serial", unsafe.Offsetof(glfw.XAnyEvent{}.Serial), 8},
		{"XAnyEvent.SendEvent", unsafe.Offsetof(glfw.XAnyEvent{}.SendEvent), 16},
		{"XAnyEvent.Display", unsafe.Offsetof(glfw.XAnyEvent{}.Display), 24},
		{"XAnyEvent.Window", unsafe.Offsetof(glfw.XAnyEvent{}.Window), 32},
		{"sizeof XAnyEvent", unsafe.Sizeof(glfw.XAnyEvent{}), 40},

		{"XKeyEvent.Window", unsafe.Offsetof(glfw.XKeyEvent{}.Window), 32},
		{"XKeyEvent.Root", unsafe.Offsetof(glfw.XKeyEvent{}.Root), 40},
		{"XKeyEvent.Subwindow", unsafe.Offsetof(glfw.XKeyEvent{}.Subwindow), 48},
		{"XKeyEvent.Time", unsafe.Offsetof(glfw.XKeyEvent{}.Time), 56},
		{"XKeyEvent.X", unsafe.Offsetof(glfw.XKeyEvent{}.X), 64},
		{"XKeyEvent.Y", unsafe.Offsetof(glfw.XKeyEvent{}.Y), 68},
		{"XKeyEvent.XRoot", unsafe.Offsetof(glfw.XKeyEvent{}.XRoot), 72},
		{"XKeyEvent.YRoot", unsafe.Offsetof(glfw.XKeyEvent{}.YRoot), 76},
		{"XKeyEvent.State", unsafe.Offsetof(glfw.XKeyEvent{}.State), 80},
		{"XKeyEvent.Keycode", unsafe.Offsetof(glfw.XKeyEvent{}.Keycode), 84},
		{"XKeyEvent.SameScreen", unsafe.Offsetof(glfw.XKeyEvent{}.SameScreen), 88},
		{"sizeof XKeyEvent", unsafe.Sizeof(glfw.XKeyEvent{}), 96},

		{"XButtonEvent.Button", unsafe.Offsetof(glfw.XButtonEvent{}.Button), 84},
		{"sizeof XButtonEvent", unsafe.Sizeof(glfw.XButtonEvent{}), 96},

		{"XMotionEvent.IsHint", unsafe.Offsetof(glfw.XMotionEvent{}.IsHint), 84},
		{"sizeof XMotionEvent", unsafe.Sizeof(glfw.XMotionEvent{}), 96},

		{"XCrossingEvent.Mode", unsafe.Offsetof(glfw.XCrossingEvent{}.Mode), 80},
		{"XCrossingEvent.Detail", unsafe.Offsetof(glfw.XCrossingEvent{}.Detail), 84},
		{"XCrossingEvent.Focus", unsafe.Offsetof(glfw.XCrossingEvent{}.Focus), 92},
		{"XCrossingEvent.State", unsafe.Offsetof(glfw.XCrossingEvent{}.State), 96},
		{"sizeof XCrossingEvent", unsafe.Sizeof(glfw.XCrossingEvent{}), 104},

		{"XFocusChangeEvent.Mode", unsafe.Offsetof(glfw.XFocusChangeEvent{}.Mode), 40},
		{"XFocusChangeEvent.Detail", unsafe.Offsetof(glfw.XFocusChangeEvent{}.Detail), 44},
		{"sizeof XFocusChangeEvent", unsafe.Sizeof(glfw.XFocusChangeEvent{}), 48},

		{"XExposeEvent.X", unsafe.Offsetof(glfw.XExposeEvent{}.X), 40},
		{"XExposeEvent.Y", unsafe.Offsetof(glfw.XExposeEvent{}.Y), 44},
		{"XExposeEvent.Width", unsafe.Offsetof(glfw.XExposeEvent{}.Width), 48},
		{"XExposeEvent.Height", unsafe.Offsetof(glfw.XExposeEvent{}.Height), 52},
		{"XExposeEvent.Count", unsafe.Offsetof(glfw.XExposeEvent{}.Count), 56},
		{"sizeof XExposeEvent", unsafe.Sizeof(glfw.XExposeEvent{}), 64},

		{"XVisibilityEvent.State", unsafe.Offsetof(glfw.XVisibilityEvent{}.State), 40},
		{"sizeof XVisibilityEvent", unsafe.Sizeof(glfw.XVisibilityEvent{}), 48},

		{"XDestroyWindowEvent.Event", unsafe.Offsetof(glfw.XDestroyWindowEvent{}.Event), 32},
		{"XDestroyWindowEvent.Window", unsafe.Offsetof(glfw.XDestroyWindowEvent{}.Window), 40},
		{"sizeof XDestroyWindowEvent", unsafe.Sizeof(glfw.XDestroyWindowEvent{}), 48},

		{"XUnmapEvent.Window", unsafe.Offsetof(glfw.XUnmapEvent{}.Window), 40},
		{"sizeof XUnmapEvent", unsafe.Sizeof(glfw.XUnmapEvent{}), 56},

		{"XMapEvent.Window", unsafe.Offsetof(glfw.XMapEvent{}.Window), 40},
		{"sizeof XMapEvent", unsafe.Sizeof(glfw.XMapEvent{}), 56},

		{"XReparentEvent.Window", unsafe.Offsetof(glfw.XReparentEvent{}.Window), 40},
		{"XReparentEvent.Parent", unsafe.Offsetof(glfw.XReparentEvent{}.Parent), 48},
		{"XReparentEvent.X", unsafe.Offsetof(glfw.XReparentEvent{}.X), 56},
		{"XReparentEvent.Y", unsafe.Offsetof(glfw.XReparentEvent{}.Y), 60},
		{"sizeof XReparentEvent", unsafe.Sizeof(glfw.XReparentEvent{}), 72},

		{"XConfigureEvent.Event", unsafe.Offsetof(glfw.XConfigureEvent{}.Event), 32},
		{"XConfigureEvent.Window", unsafe.Offsetof(glfw.XConfigureEvent{}.Window), 40},
		{"XConfigureEvent.X", unsafe.Offsetof(glfw.XConfigureEvent{}.X), 48},
		{"XConfigureEvent.Y", unsafe.Offsetof(glfw.XConfigureEvent{}.Y), 52},
		{"XConfigureEvent.Width", unsafe.Offsetof(glfw.XConfigureEvent{}.Width), 56},
		{"XConfigureEvent.Height", unsafe.Offsetof(glfw.XConfigureEvent{}.Height), 60},
		{"sizeof XConfigureEvent", unsafe.Sizeof(glfw.XConfigureEvent{}), 88},

		{"XPropertyEvent.Window", unsafe.Offsetof(glfw.XPropertyEvent{}.Window), 32},
		{"XPropertyEvent.Atom", unsafe.Offsetof(glfw.XPropertyEvent{}.Atom), 40},
		{"XPropertyEvent.Time", unsafe.Offsetof(glfw.XPropertyEvent{}.Time), 48},
		{"XPropertyEvent.State", unsafe.Offsetof(glfw.XPropertyEvent{}.State), 56},
		{"sizeof XPropertyEvent", unsafe.Sizeof(glfw.XPropertyEvent{}), 64},

		{"XSelectionRequestEvent.Owner", unsafe.Offsetof(glfw.XSelectionRequestEvent{}.Owner), 32},
		{"XSelectionRequestEvent.Requestor", unsafe.Offsetof(glfw.XSelectionRequestEvent{}.Requestor), 40},
		{"XSelectionRequestEvent.Selection", unsafe.Offsetof(glfw.XSelectionRequestEvent{}.Selection), 48},
		{"XSelectionRequestEvent.Target", unsafe.Offsetof(glfw.XSelectionRequestEvent{}.Target), 56},
		{"XSelectionRequestEvent.Property", unsafe.Offsetof(glfw.XSelectionRequestEvent{}.Property), 64},
		{"XSelectionRequestEvent.Time", unsafe.Offsetof(glfw.XSelectionRequestEvent{}.Time), 72},
		{"sizeof XSelectionRequestEvent", unsafe.Sizeof(glfw.XSelectionRequestEvent{}), 80},

		{"XSelectionEvent.Requestor", unsafe.Offsetof(glfw.XSelectionEvent{}.Requestor), 32},
		{"XSelectionEvent.Selection", unsafe.Offsetof(glfw.XSelectionEvent{}.Selection), 40},
		{"XSelectionEvent.Target", unsafe.Offsetof(glfw.XSelectionEvent{}.Target), 48},
		{"XSelectionEvent.Property", unsafe.Offsetof(glfw.XSelectionEvent{}.Property), 56},
		{"XSelectionEvent.Time", unsafe.Offsetof(glfw.XSelectionEvent{}.Time), 64},
		{"sizeof XSelectionEvent", unsafe.Sizeof(glfw.XSelectionEvent{}), 72},

		{"XSelectionClearEvent.Window", unsafe.Offsetof(glfw.XSelectionClearEvent{}.Window), 32},
		{"XSelectionClearEvent.Selection", unsafe.Offsetof(glfw.XSelectionClearEvent{}.Selection), 40},
		{"XSelectionClearEvent.Time", unsafe.Offsetof(glfw.XSelectionClearEvent{}.Time), 48},
		{"sizeof XSelectionClearEvent", unsafe.Sizeof(glfw.XSelectionClearEvent{}), 56},

		{"XClientMessageEvent.Window", unsafe.Offsetof(glfw.XClientMessageEvent{}.Window), 32},
		{"XClientMessageEvent.MessageType", unsafe.Offsetof(glfw.XClientMessageEvent{}.MessageType), 40},
		{"XClientMessageEvent.Format", unsafe.Offsetof(glfw.XClientMessageEvent{}.Format), 48},
		{"XClientMessageEvent.Data", unsafe.Offsetof(glfw.XClientMessageEvent{}.Data), 56},
		{"sizeof XClientMessageEvent", unsafe.Sizeof(glfw.XClientMessageEvent{}), 96},

		{"XGenericEventCookie.Extension", unsafe.Offsetof(glfw.XGenericEventCookie{}.Extension), 32},
		{"XGenericEventCookie.Evtype", unsafe.Offsetof(glfw.XGenericEventCookie{}.Evtype), 36},
		{"XGenericEventCookie.Cookie", unsafe.Offsetof(glfw.XGenericEventCookie{}.Cookie), 40},
		{"XGenericEventCookie.Data", unsafe.Offsetof(glfw.XGenericEventCookie{}.Data), 48},
		{"sizeof XGenericEventCookie", unsafe.Sizeof(glfw.XGenericEventCookie{}), 56},

		{"XErrorEvent.Type", unsafe.Offsetof(glfw.XErrorEvent{}.Type), 0},
		{"XErrorEvent.Display", unsafe.Offsetof(glfw.XErrorEvent{}.Display), 8},
		{"XErrorEvent.Resourceid", unsafe.Offsetof(glfw.XErrorEvent{}.Resourceid), 16},
		{"XErrorEvent.Serial", unsafe.Offsetof(glfw.XErrorEvent{}.Serial), 24},
		{"XErrorEvent.ErrorCode", unsafe.Offsetof(glfw.XErrorEvent{}.ErrorCode), 32},
		{"XErrorEvent.RequestCode", unsafe.Offsetof(glfw.XErrorEvent{}.RequestCode), 33},
		{"XErrorEvent.MinorCode", unsafe.Offsetof(glfw.XErrorEvent{}.MinorCode), 34},
		{"sizeof XErrorEvent", unsafe.Sizeof(glfw.XErrorEvent{}), 40},

		{"XSetWindowAttributes.BackgroundPixmap", unsafe.Offsetof(glfw.XSetWindowAttributes{}.BackgroundPixmap), 0},
		{"XSetWindowAttributes.BackgroundPixel", unsafe.Offsetof(glfw.XSetWindowAttributes{}.BackgroundPixel), 8},
		{"XSetWindowAttributes.BorderPixmap", unsafe.Offsetof(glfw.XSetWindowAttributes{}.BorderPixmap), 16},
		{"XSetWindowAttributes.BorderPixel", unsafe.Offsetof(glfw.XSetWindowAttributes{}.BorderPixel), 24},
		{"XSetWindowAttributes.BitGravity", unsafe.Offsetof(glfw.XSetWindowAttributes{}.BitGravity), 32},
		{"XSetWindowAttributes.WinGravity", unsafe.Offsetof(glfw.XSetWindowAttributes{}.WinGravity), 36},
		{"XSetWindowAttributes.BackingStore", unsafe.Offsetof(glfw.XSetWindowAttributes{}.BackingStore), 40},
		{"XSetWindowAttributes.BackingPlanes", unsafe.Offsetof(glfw.XSetWindowAttributes{}.BackingPlanes), 48},
		{"XSetWindowAttributes.BackingPixel", unsafe.Offsetof(glfw.XSetWindowAttributes{}.BackingPixel), 56},
		{"XSetWindowAttributes.SaveUnder", unsafe.Offsetof(glfw.XSetWindowAttributes{}.SaveUnder), 64},
		{"XSetWindowAttributes.EventMask", unsafe.Offsetof(glfw.XSetWindowAttributes{}.EventMask), 72},
		{"XSetWindowAttributes.DoNotPropagateMask", unsafe.Offsetof(glfw.XSetWindowAttributes{}.DoNotPropagateMask), 80},
		{"XSetWindowAttributes.OverrideRedirect", unsafe.Offsetof(glfw.XSetWindowAttributes{}.OverrideRedirect), 88},
		{"XSetWindowAttributes.Colormap", unsafe.Offsetof(glfw.XSetWindowAttributes{}.Colormap), 96},
		{"XSetWindowAttributes.Cursor", unsafe.Offsetof(glfw.XSetWindowAttributes{}.Cursor), 104},
		{"sizeof XSetWindowAttributes", unsafe.Sizeof(glfw.XSetWindowAttributes{}), 112},

		{"XWindowAttributes.X", unsafe.Offsetof(glfw.XWindowAttributes{}.X), 0},
		{"XWindowAttributes.Y", unsafe.Offsetof(glfw.XWindowAttributes{}.Y), 4},
		{"XWindowAttributes.Width", unsafe.Offsetof(glfw.XWindowAttributes{}.Width), 8},
		{"XWindowAttributes.Height", unsafe.Offsetof(glfw.XWindowAttributes{}.Height), 12},
		{"XWindowAttributes.Depth", unsafe.Offsetof(glfw.XWindowAttributes{}.Depth), 20},
		{"XWindowAttributes.Visual", unsafe.Offsetof(glfw.XWindowAttributes{}.Visual), 24},
		{"XWindowAttributes.Root", unsafe.Offsetof(glfw.XWindowAttributes{}.Root), 32},
		{"XWindowAttributes.MapState", unsafe.Offsetof(glfw.XWindowAttributes{}.MapState), 92},
		{"XWindowAttributes.OverrideRedirect", unsafe.Offsetof(glfw.XWindowAttributes{}.OverrideRedirect), 120},
		{"XWindowAttributes.Screen", unsafe.Offsetof(glfw.XWindowAttributes{}.Screen), 128},
		{"sizeof XWindowAttributes", unsafe.Sizeof(glfw.XWindowAttributes{}), 136},

		{"Visual.Visualid", unsafe.Offsetof(glfw.Visual{}.Visualid), 8},
		{"sizeof Visual", unsafe.Sizeof(glfw.Visual{}), 56},

		{"XVisualInfo.Visual", unsafe.Offsetof(glfw.XVisualInfo{}.Visual), 0},
		{"XVisualInfo.Visualid", unsafe.Offsetof(glfw.XVisualInfo{}.Visualid), 8},
		{"XVisualInfo.Screen", unsafe.Offsetof(glfw.XVisualInfo{}.Screen), 16},
		{"XVisualInfo.Depth", unsafe.Offsetof(glfw.XVisualInfo{}.Depth), 20},
		{"XVisualInfo.Class", unsafe.Offsetof(glfw.XVisualInfo{}.Class), 24},
		{"XVisualInfo.RedMask", unsafe.Offsetof(glfw.XVisualInfo{}.RedMask), 32},
		{"XVisualInfo.ColormapSize", unsafe.Offsetof(glfw.XVisualInfo{}.ColormapSize), 56},
		{"XVisualInfo.BitsPerRGB", unsafe.Offsetof(glfw.XVisualInfo{}.BitsPerRGB), 60},
		{"sizeof XVisualInfo", unsafe.Sizeof(glfw.XVisualInfo{}), 64},

		{"XSizeHints.Flags", unsafe.Offsetof(glfw.XSizeHints{}.Flags), 0},
		{"XSizeHints.X", unsafe.Offsetof(glfw.XSizeHints{}.X), 8},
		{"XSizeHints.Y", unsafe.Offsetof(glfw.XSizeHints{}.Y), 12},
		{"XSizeHints.Width", unsafe.Offsetof(glfw.XSizeHints{}.Width), 16},
		{"XSizeHints.Height", unsafe.Offsetof(glfw.XSizeHints{}.Height), 20},
		{"XSizeHints.MinWidth", unsafe.Offsetof(glfw.XSizeHints{}.MinWidth), 24},
		{"XSizeHints.MinHeight", unsafe.Offsetof(glfw.XSizeHints{}.MinHeight), 28},
		{"XSizeHints.MaxWidth", unsafe.Offsetof(glfw.XSizeHints{}.MaxWidth), 32},
		{"XSizeHints.MaxHeight", unsafe.Offsetof(glfw.XSizeHints{}.MaxHeight), 36},
		{"XSizeHints.WidthInc", unsafe.Offsetof(glfw.XSizeHints{}.WidthInc), 40},
		{"XSizeHints.HeightInc", unsafe.Offsetof(glfw.XSizeHints{}.HeightInc), 44},
		{"XSizeHints.MinAspect", unsafe.Offsetof(glfw.XSizeHints{}.MinAspect), 48},
		{"XSizeHints.MaxAspect", unsafe.Offsetof(glfw.XSizeHints{}.MaxAspect), 56},
		{"XSizeHints.BaseWidth", unsafe.Offsetof(glfw.XSizeHints{}.BaseWidth), 64},
		{"XSizeHints.BaseHeight", unsafe.Offsetof(glfw.XSizeHints{}.BaseHeight), 68},
		{"XSizeHints.WinGravity", unsafe.Offsetof(glfw.XSizeHints{}.WinGravity), 72},
		{"sizeof XSizeHints", unsafe.Sizeof(glfw.XSizeHints{}), 80},

		{"XWMHints.Flags", unsafe.Offsetof(glfw.XWMHints{}.Flags), 0},
		{"XWMHints.Input", unsafe.Offsetof(glfw.XWMHints{}.Input), 8},
		{"XWMHints.InitialState", unsafe.Offsetof(glfw.XWMHints{}.InitialState), 12},
		{"XWMHints.IconPixmap", unsafe.Offsetof(glfw.XWMHints{}.IconPixmap), 16},
		{"sizeof XWMHints", unsafe.Sizeof(glfw.XWMHints{}), 56},

		{"XClassHint.ResName", unsafe.Offsetof(glfw.XClassHint{}.ResName), 0},
		{"XClassHint.ResClass", unsafe.Offsetof(glfw.XClassHint{}.ResClass), 8},
		{"sizeof XClassHint", unsafe.Sizeof(glfw.XClassHint{}), 16},

		{"XIMStyles.CountStyles", unsafe.Offsetof(glfw.XIMStyles{}.CountStyles), 0},
		{"XIMStyles.SupportedStyles", unsafe.Offsetof(glfw.XIMStyles{}.SupportedStyles), 8},
		{"sizeof XIMStyles", unsafe.Sizeof(glfw.XIMStyles{}), 16},

		{"XRectangle.X", unsafe.Offsetof(glfw.XRectangle{}.X), 0},
		{"XRectangle.Y", unsafe.Offsetof(glfw.XRectangle{}.Y), 2},
		{"XRectangle.Width", unsafe.Offsetof(glfw.XRectangle{}.Width), 4},
		{"XRectangle.Height", unsafe.Offsetof(glfw.XRectangle{}.Height), 6},
		{"sizeof XRectangle", unsafe.Sizeof(glfw.XRectangle{}), 8},

		{"XkbDescRec.Dpy", unsafe.Offsetof(glfw.XkbDescRec{}.Dpy), 0},
		{"XkbDescRec.Flags", unsafe.Offsetof(glfw.XkbDescRec{}.Flags), 8},
		{"XkbDescRec.DeviceSpec", unsafe.Offsetof(glfw.XkbDescRec{}.DeviceSpec), 10},
		{"XkbDescRec.MinKeyCode", unsafe.Offsetof(glfw.XkbDescRec{}.MinKeyCode), 12},
		{"XkbDescRec.MaxKeyCode", unsafe.Offsetof(glfw.XkbDescRec{}.MaxKeyCode), 13},
		{"XkbDescRec.Ctrls", unsafe.Offsetof(glfw.XkbDescRec{}.Ctrls), 16},
		{"XkbDescRec.Server", unsafe.Offsetof(glfw.XkbDescRec{}.Server), 24},
		{"XkbDescRec.Map", unsafe.Offsetof(glfw.XkbDescRec{}.Map), 32},
		{"XkbDescRec.Indicators", unsafe.Offsetof(glfw.XkbDescRec{}.Indicators), 40},
		{"XkbDescRec.Names", unsafe.Offsetof(glfw.XkbDescRec{}.Names), 48},
		{"XkbDescRec.Compat", unsafe.Offsetof(glfw.XkbDescRec{}.Compat), 56},
		{"XkbDescRec.Geom", unsafe.Offsetof(glfw.XkbDescRec{}.Geom), 64},
		{"sizeof XkbDescRec", unsafe.Sizeof(glfw.XkbDescRec{}), 72},

		{"XkbNamesRec.Keys", unsafe.Offsetof(glfw.XkbNamesRec{}.Keys), 456},
		{"XkbNamesRec.KeyAliases", unsafe.Offsetof(glfw.XkbNamesRec{}.KeyAliases), 464},
		{"XkbNamesRec.NumKeyAliases", unsafe.Offsetof(glfw.XkbNamesRec{}.NumKeyAliases), 489},
		{"sizeof XkbNamesRec", unsafe.Sizeof(glfw.XkbNamesRec{}), 496},

		{"sizeof XkbKeyNameRec", unsafe.Sizeof(glfw.XkbKeyNameRec{}), 4},
		{"XkbKeyAliasRec.Alias", unsafe.Offsetof(glfw.XkbKeyAliasRec{}.Alias), 4},
		{"sizeof XkbKeyAliasRec", unsafe.Sizeof(glfw.XkbKeyAliasRec{}), 8},

		{"XkbStateRec.Group", unsafe.Offsetof(glfw.XkbStateRec{}.Group), 0},
		{"sizeof XkbStateRec", unsafe.Sizeof(glfw.XkbStateRec{}), 18},

		{"XkbAnyEvent.XkbType", unsafe.Offsetof(glfw.XkbAnyEvent{}.XkbType), 40},
		{"XkbAnyEvent.Device", unsafe.Offsetof(glfw.XkbAnyEvent{}.Device), 44},
		{"sizeof XkbAnyEvent", unsafe.Sizeof(glfw.XkbAnyEvent{}), 48},

		{"XkbStateNotifyEvent.Changed", unsafe.Offsetof(glfw.XkbStateNotifyEvent{}.Changed), 48},
		{"XkbStateNotifyEvent.Group", unsafe.Offsetof(glfw.XkbStateNotifyEvent{}.Group), 52},
		{"sizeof XkbStateNotifyEvent", unsafe.Sizeof(glfw.XkbStateNotifyEvent{}), 104},

		{"XRenderDirectFormat.Alpha", unsafe.Offsetof(glfw.XRenderDirectFormat{}.Alpha), 12},
		{"XRenderDirectFormat.AlphaMask", unsafe.Offsetof(glfw.XRenderDirectFormat{}.AlphaMask), 14},
		{"sizeof XRenderDirectFormat", unsafe.Sizeof(glfw.XRenderDirectFormat{}), 16},

		{"XRenderPictFormat.Type", unsafe.Offsetof(glfw.XRenderPictFormat{}.Type), 8},
		{"XRenderPictFormat.Depth", unsafe.Offsetof(glfw.XRenderPictFormat{}.Depth), 12},
		{"XRenderPictFormat.Direct", unsafe.Offsetof(glfw.XRenderPictFormat{}.Direct), 16},
		{"XRenderPictFormat.Colormap", unsafe.Offsetof(glfw.XRenderPictFormat{}.Colormap), 32},
		{"sizeof XRenderPictFormat", unsafe.Sizeof(glfw.XRenderPictFormat{}), 40},

		{"XRRModeInfo.Width", unsafe.Offsetof(glfw.XRRModeInfo{}.Width), 8},
		{"XRRModeInfo.Height", unsafe.Offsetof(glfw.XRRModeInfo{}.Height), 12},
		{"XRRModeInfo.DotClock", unsafe.Offsetof(glfw.XRRModeInfo{}.DotClock), 16},
		{"XRRModeInfo.HSyncStart", unsafe.Offsetof(glfw.XRRModeInfo{}.HSyncStart), 24},
		{"XRRModeInfo.HTotal", unsafe.Offsetof(glfw.XRRModeInfo{}.HTotal), 32},
		{"XRRModeInfo.VTotal", unsafe.Offsetof(glfw.XRRModeInfo{}.VTotal), 48},
		{"XRRModeInfo.NameLength", unsafe.Offsetof(glfw.XRRModeInfo{}.NameLength), 64},
		{"XRRModeInfo.ModeFlags", unsafe.Offsetof(glfw.XRRModeInfo{}.ModeFlags), 72},
		{"sizeof XRRModeInfo", unsafe.Sizeof(glfw.XRRModeInfo{}), 80},

		{"XRRScreenResources.Timestamp", unsafe.Offsetof(glfw.XRRScreenResources{}.Timestamp), 0},
		{"XRRScreenResources.ConfigTimestamp", unsafe.Offsetof(glfw.XRRScreenResources{}.ConfigTimestamp), 8},
		{"XRRScreenResources.Ncrtc", unsafe.Offsetof(glfw.XRRScreenResources{}.Ncrtc), 16},
		{"XRRScreenResources.Crtcs", unsafe.Offsetof(glfw.XRRScreenResources{}.Crtcs), 24},
		{"XRRScreenResources.Noutput", unsafe.Offsetof(glfw.XRRScreenResources{}.Noutput), 32},
		{"XRRScreenResources.Outputs", unsafe.Offsetof(glfw.XRRScreenResources{}.Outputs), 40},
		{"XRRScreenResources.Nmode", unsafe.Offsetof(glfw.XRRScreenResources{}.Nmode), 48},
		{"XRRScreenResources.Modes", unsafe.Offsetof(glfw.XRRScreenResources{}.Modes), 56},
		{"sizeof XRRScreenResources", unsafe.Sizeof(glfw.XRRScreenResources{}), 64},

		{"XRROutputInfo.Crtc", unsafe.Offsetof(glfw.XRROutputInfo{}.Crtc), 8},
		{"XRROutputInfo.Name", unsafe.Offsetof(glfw.XRROutputInfo{}.Name), 16},
		{"XRROutputInfo.NameLen", unsafe.Offsetof(glfw.XRROutputInfo{}.NameLen), 24},
		{"XRROutputInfo.MmWidth", unsafe.Offsetof(glfw.XRROutputInfo{}.MmWidth), 32},
		{"XRROutputInfo.MmHeight", unsafe.Offsetof(glfw.XRROutputInfo{}.MmHeight), 40},
		{"XRROutputInfo.Connection", unsafe.Offsetof(glfw.XRROutputInfo{}.Connection), 48},
		{"XRROutputInfo.Ncrtc", unsafe.Offsetof(glfw.XRROutputInfo{}.Ncrtc), 52},
		{"XRROutputInfo.Crtcs", unsafe.Offsetof(glfw.XRROutputInfo{}.Crtcs), 56},
		{"XRROutputInfo.Nmode", unsafe.Offsetof(glfw.XRROutputInfo{}.Nmode), 80},
		{"XRROutputInfo.Npreferred", unsafe.Offsetof(glfw.XRROutputInfo{}.Npreferred), 84},
		{"XRROutputInfo.Modes", unsafe.Offsetof(glfw.XRROutputInfo{}.Modes), 88},
		{"sizeof XRROutputInfo", unsafe.Sizeof(glfw.XRROutputInfo{}), 96},

		{"XRRCrtcInfo.X", unsafe.Offsetof(glfw.XRRCrtcInfo{}.X), 8},
		{"XRRCrtcInfo.Y", unsafe.Offsetof(glfw.XRRCrtcInfo{}.Y), 12},
		{"XRRCrtcInfo.Width", unsafe.Offsetof(glfw.XRRCrtcInfo{}.Width), 16},
		{"XRRCrtcInfo.Height", unsafe.Offsetof(glfw.XRRCrtcInfo{}.Height), 20},
		{"XRRCrtcInfo.Mode", unsafe.Offsetof(glfw.XRRCrtcInfo{}.Mode), 24},
		{"XRRCrtcInfo.Rotation", unsafe.Offsetof(glfw.XRRCrtcInfo{}.Rotation), 32},
		{"XRRCrtcInfo.Noutput", unsafe.Offsetof(glfw.XRRCrtcInfo{}.Noutput), 36},
		{"XRRCrtcInfo.Outputs", unsafe.Offsetof(glfw.XRRCrtcInfo{}.Outputs), 40},
		{"sizeof XRRCrtcInfo", unsafe.Sizeof(glfw.XRRCrtcInfo{}), 64},

		{"XineramaScreenInfo.XOrg", unsafe.Offsetof(glfw.XineramaScreenInfo{}.XOrg), 4},
		{"XineramaScreenInfo.YOrg", unsafe.Offsetof(glfw.XineramaScreenInfo{}.YOrg), 6},
		{"XineramaScreenInfo.Width", unsafe.Offsetof(glfw.XineramaScreenInfo{}.Width), 8},
		{"XineramaScreenInfo.Height", unsafe.Offsetof(glfw.XineramaScreenInfo{}.Height), 10},
		{"sizeof XineramaScreenInfo", unsafe.Sizeof(glfw.XineramaScreenInfo{}), 12},

		{"XcursorImage.Xhot", unsafe.Offsetof(glfw.XcursorImage{}.Xhot), 16},
		{"XcursorImage.Yhot", unsafe.Offsetof(glfw.XcursorImage{}.Yhot), 20},
		{"XcursorImage.Delay", unsafe.Offsetof(glfw.XcursorImage{}.Delay), 24},
		{"XcursorImage.Pixels", unsafe.Offsetof(glfw.XcursorImage{}.Pixels), 32},
		{"sizeof XcursorImage", unsafe.Sizeof(glfw.XcursorImage{}), 40},

		{"XIEventMask.MaskLen", unsafe.Offsetof(glfw.XIEventMask{}.MaskLen), 4},
		{"XIEventMask.Mask", unsafe.Offsetof(glfw.XIEventMask{}.Mask), 8},
		{"sizeof XIEventMask", unsafe.Sizeof(glfw.XIEventMask{}), 16},

		{"XIValuatorState.Mask", unsafe.Offsetof(glfw.XIValuatorState{}.Mask), 8},
		{"XIValuatorState.Values", unsafe.Offsetof(glfw.XIValuatorState{}.Values), 16},
		{"sizeof XIValuatorState", unsafe.Sizeof(glfw.XIValuatorState{}), 24},

		{"XIRawEvent.Extension", unsafe.Offsetof(glfw.XIRawEvent{}.Extension), 32},
		{"XIRawEvent.Evtype", unsafe.Offsetof(glfw.XIRawEvent{}.Evtype), 36},
		{"XIRawEvent.Time", unsafe.Offsetof(glfw.XIRawEvent{}.Time), 40},
		{"XIRawEvent.Deviceid", unsafe.Offsetof(glfw.XIRawEvent{}.Deviceid), 48},
		{"XIRawEvent.Detail", unsafe.Offsetof(glfw.XIRawEvent{}.Detail), 56},
		{"XIRawEvent.Flags", unsafe.Offsetof(glfw.XIRawEvent{}.Flags), 60},
		{"XIRawEvent.Valuators", unsafe.Offsetof(glfw.XIRawEvent{}.Valuators), 64},
		{"XIRawEvent.RawValues", unsafe.Offsetof(glfw.XIRawEvent{}.RawValues), 88},
		{"sizeof XIRawEvent", unsafe.Sizeof(glfw.XIRawEvent{}), 96},

		// XSyncValue is {int hi; unsigned int lo}, both 32-bit on every data
		// model. The field order is load-bearing: it is passed to the X Sync
		// counter functions by value, so hi must stay at offset 0.
		{"XSyncValue.Hi", unsafe.Offsetof(glfw.XSyncValue{}.Hi), 0},
		{"XSyncValue.Lo", unsafe.Offsetof(glfw.XSyncValue{}.Lo), 4},
		{"sizeof XSyncValue", unsafe.Sizeof(glfw.XSyncValue{}), 8},
	} {
		if tt.got != tt.want {
			t.Errorf("%s: got %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}
