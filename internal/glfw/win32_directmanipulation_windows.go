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

package glfw

import (
	"errors"
	"fmt"
	"math"
	"runtime"
	"structs"
	"syscall"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/winver"
)

// Direct Manipulation is used to receive gestures on a precision touchpad. Without it, Windows
// reports a two-finger pan as WM_MOUSEWHEEL and a pinch as WM_MOUSEWHEEL with MK_CONTROL, which
// cannot be told apart from a mouse wheel turned with the Control key held.
//
// Only contacts whose pointer type is PT_TOUCHPAD are handed to Direct Manipulation (see
// DM_POINTERHITTEST in windowProc). Mice, touch screens, pens, and touchpads that are not precision
// touchpads are not affected and keep sending the same messages as before.
//
// Once a contact is handed to Direct Manipulation, Windows no longer sends WM_MOUSEWHEEL for it, so
// both a pan and a pinch are reported from the content transform of a viewport. The transform is
// polled after each round of message processing instead of registering an
// IDirectManipulationViewportEventHandler, so that no COM object has to be implemented in Go.
//
// For the interfaces, see directmanipulation.idl in https://github.com/wine-mirror/wine.

const (
	_CLSCTX_INPROC_SERVER = 0x1

	_DIRECTMANIPULATION_CONFIGURATION_INTERACTION         = 0x00000001
	_DIRECTMANIPULATION_CONFIGURATION_TRANSLATION_X       = 0x00000002
	_DIRECTMANIPULATION_CONFIGURATION_TRANSLATION_Y       = 0x00000004
	_DIRECTMANIPULATION_CONFIGURATION_SCALING             = 0x00000010
	_DIRECTMANIPULATION_CONFIGURATION_TRANSLATION_INERTIA = 0x00000020

	_DIRECTMANIPULATION_VIEWPORT_OPTIONS_MANUALUPDATE = 0x00000002

	_DIRECTMANIPULATION_RUNNING = 3
	_DIRECTMANIPULATION_INERTIA = 4
)

var (
	_CLSID_DirectManipulationManager = windows.GUID{
		Data1: 0x54e211b6,
		Data2: 0x3650,
		Data3: 0x4f75,
		Data4: [...]byte{0x83, 0x34, 0xfa, 0x35, 0x95, 0x98, 0xe1, 0xc5},
	}
	_IID_IDirectManipulationManager = windows.GUID{
		Data1: 0xfbf5d3b4,
		Data2: 0x70c7,
		Data3: 0x4163,
		Data4: [...]byte{0x93, 0x22, 0x5a, 0x6f, 0x66, 0x0d, 0x6f, 0xbc},
	}
	_IID_IDirectManipulationUpdateManager = windows.GUID{
		Data1: 0xb0ae62fd,
		Data2: 0xbe34,
		Data3: 0x46e7,
		Data4: [...]byte{0x9c, 0xaa, 0xd3, 0x61, 0xfa, 0xcb, 0xb9, 0xcc},
	}
	_IID_IDirectManipulationViewport = windows.GUID{
		Data1: 0x28b85a3d,
		Data2: 0x60a0,
		Data3: 0x48bd,
		Data4: [...]byte{0x9b, 0xa1, 0x5c, 0xe8, 0xd9, 0xea, 0x3a, 0x6d},
	}
	_IID_IDirectManipulationContent = windows.GUID{
		Data1: 0xb89962cb,
		Data2: 0x3d89,
		Data3: 0x442b,
		Data4: [...]byte{0xbb, 0x58, 0x50, 0x98, 0xfa, 0x0f, 0x9f, 0x16},
	}
)

// hresultFailed reports whether r is a failure HRESULT, as FAILED does.
// Success codes other than S_OK, such as S_FALSE, are not failures.
func hresultFailed(r uintptr) bool {
	return int32(uint32(r)) < 0
}

func hresultError(method string, r uintptr) error {
	return fmt.Errorf("glfw: %s failed: %w", method, handleError(windows.Handle(uint32(r))))
}

type _IDirectManipulationManager struct {
	_    structs.HostLayout
	vtbl *_IDirectManipulationManager_Vtbl
}

type _IDirectManipulationManager_Vtbl struct {
	_              structs.HostLayout
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	Activate              uintptr
	Deactivate            uintptr
	RegisterHitTestTarget uintptr
	ProcessInput          uintptr
	GetUpdateManager      uintptr
	CreateViewport        uintptr
	CreateContent         uintptr
}

func (i *_IDirectManipulationManager) Release() {
	_, _, _ = syscall.SyscallN(i.vtbl.Release, uintptr(unsafe.Pointer(i)))
}

func (i *_IDirectManipulationManager) Activate(hwnd windows.HWND) error {
	r, _, _ := syscall.SyscallN(i.vtbl.Activate, uintptr(unsafe.Pointer(i)), uintptr(hwnd))
	if hresultFailed(r) {
		return hresultError("IDirectManipulationManager::Activate", r)
	}
	return nil
}

func (i *_IDirectManipulationManager) Deactivate(hwnd windows.HWND) error {
	r, _, _ := syscall.SyscallN(i.vtbl.Deactivate, uintptr(unsafe.Pointer(i)), uintptr(hwnd))
	if hresultFailed(r) {
		return hresultError("IDirectManipulationManager::Deactivate", r)
	}
	return nil
}

func (i *_IDirectManipulationManager) GetUpdateManager() (*_IDirectManipulationUpdateManager, error) {
	var ptr *_IDirectManipulationUpdateManager
	r, _, _ := syscall.SyscallN(i.vtbl.GetUpdateManager, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(&_IID_IDirectManipulationUpdateManager)), uintptr(unsafe.Pointer(&ptr)))
	if hresultFailed(r) {
		return nil, hresultError("IDirectManipulationManager::GetUpdateManager", r)
	}
	return ptr, nil
}

func (i *_IDirectManipulationManager) CreateViewport(hwnd windows.HWND) (*_IDirectManipulationViewport, error) {
	var ptr *_IDirectManipulationViewport
	r, _, _ := syscall.SyscallN(i.vtbl.CreateViewport, uintptr(unsafe.Pointer(i)), 0, uintptr(hwnd), uintptr(unsafe.Pointer(&_IID_IDirectManipulationViewport)), uintptr(unsafe.Pointer(&ptr)))
	if hresultFailed(r) {
		return nil, hresultError("IDirectManipulationManager::CreateViewport", r)
	}
	return ptr, nil
}

type _IDirectManipulationUpdateManager struct {
	_    structs.HostLayout
	vtbl *_IDirectManipulationUpdateManager_Vtbl
}

type _IDirectManipulationUpdateManager_Vtbl struct {
	_              structs.HostLayout
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	RegisterWaitHandleCallback   uintptr
	UnregisterWaitHandleCallback uintptr
	Update                       uintptr
}

func (i *_IDirectManipulationUpdateManager) Release() {
	_, _, _ = syscall.SyscallN(i.vtbl.Release, uintptr(unsafe.Pointer(i)))
}

func (i *_IDirectManipulationUpdateManager) Update() error {
	r, _, _ := syscall.SyscallN(i.vtbl.Update, uintptr(unsafe.Pointer(i)), 0)
	if hresultFailed(r) {
		return hresultError("IDirectManipulationUpdateManager::Update", r)
	}
	return nil
}

type _IDirectManipulationViewport struct {
	_    structs.HostLayout
	vtbl *_IDirectManipulationViewport_Vtbl
}

type _IDirectManipulationViewport_Vtbl struct {
	_              structs.HostLayout
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	Enable                uintptr
	Disable               uintptr
	SetContact            uintptr
	ReleaseContact        uintptr
	ReleaseAllContacts    uintptr
	GetStatus             uintptr
	GetTag                uintptr
	SetTag                uintptr
	GetViewportRect       uintptr
	SetViewportRect       uintptr
	ZoomToRect            uintptr
	SetViewportTransform  uintptr
	SyncDisplayTransform  uintptr
	GetPrimaryContent     uintptr
	AddContent            uintptr
	RemoveContent         uintptr
	SetViewportOptions    uintptr
	AddConfiguration      uintptr
	RemoveConfiguration   uintptr
	ActivateConfiguration uintptr
	SetManualGesture      uintptr
	SetChaining           uintptr
	AddEventHandler       uintptr
	RemoveEventHandler    uintptr
	SetInputMode          uintptr
	SetUpdateMode         uintptr
	Stop                  uintptr
	Abandon               uintptr
}

func (i *_IDirectManipulationViewport) Release() {
	_, _, _ = syscall.SyscallN(i.vtbl.Release, uintptr(unsafe.Pointer(i)))
}

func (i *_IDirectManipulationViewport) Enable() error {
	r, _, _ := syscall.SyscallN(i.vtbl.Enable, uintptr(unsafe.Pointer(i)))
	if hresultFailed(r) {
		return hresultError("IDirectManipulationViewport::Enable", r)
	}
	return nil
}

func (i *_IDirectManipulationViewport) SetContact(pointerId uint32) error {
	r, _, _ := syscall.SyscallN(i.vtbl.SetContact, uintptr(unsafe.Pointer(i)), uintptr(pointerId))
	if hresultFailed(r) {
		return hresultError("IDirectManipulationViewport::SetContact", r)
	}
	return nil
}

func (i *_IDirectManipulationViewport) GetStatus() (uint32, error) {
	var status uint32
	r, _, _ := syscall.SyscallN(i.vtbl.GetStatus, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(&status)))
	if hresultFailed(r) {
		return 0, hresultError("IDirectManipulationViewport::GetStatus", r)
	}
	return status, nil
}

func (i *_IDirectManipulationViewport) SetViewportRect(viewport *_RECT) error {
	r, _, _ := syscall.SyscallN(i.vtbl.SetViewportRect, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(viewport)))
	if hresultFailed(r) {
		return hresultError("IDirectManipulationViewport::SetViewportRect", r)
	}
	return nil
}

// zoomToRectFunc is the type of IDirectManipulationViewport::ZoomToRect called through purego.
type zoomToRectFunc func(this unsafe.Pointer, left, top, right, bottom float32, animate int32) int32

// newZoomToRectFunc returns a function that calls IDirectManipulationViewport::ZoomToRect of i, or nil
// when the system call can call it.
//
// On 386 and amd64, the system call passes the float arguments where the callee reads them from: on the
// stack on 386, and in both the integer and the float registers on amd64. On arm64, float arguments are
// passed in the float registers, numbered independently of the integer registers, which the system call
// never sets. purego sets them.
func (i *_IDirectManipulationViewport) newZoomToRectFunc() zoomToRectFunc {
	switch runtime.GOARCH {
	case "386", "amd64":
		return nil
	}
	var f zoomToRectFunc
	purego.RegisterFunc(&f, i.vtbl.ZoomToRect)
	return f
}

// ZoomToRect calls IDirectManipulationViewport::ZoomToRect. f is the function newZoomToRectFunc returns.
func (i *_IDirectManipulationViewport) ZoomToRect(f zoomToRectFunc, left, top, right, bottom float32, animate bool) error {
	var a int32
	if animate {
		a = 1
	}
	var r uintptr
	if f != nil {
		r = uintptr(uint32(f(unsafe.Pointer(i), left, top, right, bottom, a)))
	} else {
		r, _, _ = syscall.SyscallN(i.vtbl.ZoomToRect, uintptr(unsafe.Pointer(i)), uintptr(math.Float32bits(left)), uintptr(math.Float32bits(top)), uintptr(math.Float32bits(right)), uintptr(math.Float32bits(bottom)), uintptr(a))
	}
	if hresultFailed(r) {
		return hresultError("IDirectManipulationViewport::ZoomToRect", r)
	}
	return nil
}

func (i *_IDirectManipulationViewport) GetPrimaryContent() (*_IDirectManipulationContent, error) {
	var ptr *_IDirectManipulationContent
	r, _, _ := syscall.SyscallN(i.vtbl.GetPrimaryContent, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(&_IID_IDirectManipulationContent)), uintptr(unsafe.Pointer(&ptr)))
	if hresultFailed(r) {
		return nil, hresultError("IDirectManipulationViewport::GetPrimaryContent", r)
	}
	return ptr, nil
}

func (i *_IDirectManipulationViewport) SetViewportOptions(options uint32) error {
	r, _, _ := syscall.SyscallN(i.vtbl.SetViewportOptions, uintptr(unsafe.Pointer(i)), uintptr(options))
	if hresultFailed(r) {
		return hresultError("IDirectManipulationViewport::SetViewportOptions", r)
	}
	return nil
}

func (i *_IDirectManipulationViewport) ActivateConfiguration(configuration uint32) error {
	r, _, _ := syscall.SyscallN(i.vtbl.ActivateConfiguration, uintptr(unsafe.Pointer(i)), uintptr(configuration))
	if hresultFailed(r) {
		return hresultError("IDirectManipulationViewport::ActivateConfiguration", r)
	}
	return nil
}

func (i *_IDirectManipulationViewport) Abandon() error {
	r, _, _ := syscall.SyscallN(i.vtbl.Abandon, uintptr(unsafe.Pointer(i)))
	if hresultFailed(r) {
		return hresultError("IDirectManipulationViewport::Abandon", r)
	}
	return nil
}

type _IDirectManipulationContent struct {
	_    structs.HostLayout
	vtbl *_IDirectManipulationContent_Vtbl
}

type _IDirectManipulationContent_Vtbl struct {
	_              structs.HostLayout
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	GetContentRect       uintptr
	SetContentRect       uintptr
	GetViewport          uintptr
	GetTag               uintptr
	SetTag               uintptr
	GetOutputTransform   uintptr
	GetContentTransform  uintptr
	SyncContentTransform uintptr
}

func (i *_IDirectManipulationContent) Release() {
	_, _, _ = syscall.SyscallN(i.vtbl.Release, uintptr(unsafe.Pointer(i)))
}

// GetContentTransform returns the 2D affine transform of the content as
// [scaleX, rotationY, rotationX, scaleY, translationX, translationY].
func (i *_IDirectManipulationContent) GetContentTransform() ([6]float32, error) {
	var m [6]float32
	r, _, _ := syscall.SyscallN(i.vtbl.GetContentTransform, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(&m[0])), uintptr(len(m)))
	if hresultFailed(r) {
		return [6]float32{}, hresultError("IDirectManipulationContent::GetContentTransform", r)
	}
	return m, nil
}

// directManipulationViewportSize is the width and the height of the viewport in pixels. Any size
// works, as only the changes of the content transform are used.
const directManipulationViewportSize = 1000

type directManipulation struct {
	manager       *_IDirectManipulationManager
	updateManager *_IDirectManipulationUpdateManager
	viewport      *_IDirectManipulationViewport
	content       *_IDirectManipulationContent

	// zoomToRect is the function to call ZoomToRect of the viewport with. It is registered once, as the
	// method of the viewport does not change.
	zoomToRect zoomToRectFunc

	activated       bool
	uninitializeCOM bool

	gesture directManipulationGesture
}

// newDirectManipulation starts Direct Manipulation for the window. It returns nil without an error
// when Direct Manipulation is not available in this environment.
func newDirectManipulation(hwnd windows.HWND) (dm *directManipulation, err error) {
	// Direct Manipulation handles precision touchpads as of Windows 10.
	if !winver.IsWindows10OrGreater() {
		return nil, nil
	}

	// d is not the named result, as a return statement with an error sets the result to nil
	// before the deferred function releases d.
	d := &directManipulation{
		gesture: newDirectManipulationGesture(),
	}
	dm = d
	defer func() {
		if err != nil {
			d.release(hwnd)
		}
	}()

	// Direct Manipulation is used from the thread that owns the window, in a single-threaded
	// apartment. When COM has already been initialized in another mode on this thread, use it as is.
	// S_FALSE is returned when CoInitializeEx is nested. This is a successful case.
	if err := windows.CoInitializeEx(0, windows.COINIT_APARTMENTTHREADED); err == nil || errors.Is(err, syscall.Errno(windows.S_FALSE)) {
		// CoUninitialize should be called even when CoInitializeEx returns S_FALSE.
		dm.uninitializeCOM = true
	} else if !errors.Is(err, syscall.Errno(windows.RPC_E_CHANGED_MODE)) {
		return nil, fmt.Errorf("glfw: CoInitializeEx failed: %w", err)
	}

	ptr, err := _CoCreateInstance(&_CLSID_DirectManipulationManager, nil, _CLSCTX_INPROC_SERVER, &_IID_IDirectManipulationManager)
	if err != nil {
		return nil, err
	}
	dm.manager = (*_IDirectManipulationManager)(ptr)

	if dm.updateManager, err = dm.manager.GetUpdateManager(); err != nil {
		return nil, err
	}
	if dm.viewport, err = dm.manager.CreateViewport(hwnd); err != nil {
		return nil, err
	}
	dm.zoomToRect = dm.viewport.newZoomToRectFunc()
	if err := dm.viewport.ActivateConfiguration(_DIRECTMANIPULATION_CONFIGURATION_INTERACTION |
		_DIRECTMANIPULATION_CONFIGURATION_TRANSLATION_X |
		_DIRECTMANIPULATION_CONFIGURATION_TRANSLATION_Y |
		_DIRECTMANIPULATION_CONFIGURATION_TRANSLATION_INERTIA |
		_DIRECTMANIPULATION_CONFIGURATION_SCALING); err != nil {
		return nil, err
	}
	// The viewport is not composited by Direct Manipulation. Its transform is updated only by
	// IDirectManipulationUpdateManager::Update.
	if err := dm.viewport.SetViewportOptions(_DIRECTMANIPULATION_VIEWPORT_OPTIONS_MANUALUPDATE); err != nil {
		return nil, err
	}
	if err := dm.viewport.SetViewportRect(&_RECT{
		right:  directManipulationViewportSize,
		bottom: directManipulationViewportSize,
	}); err != nil {
		return nil, err
	}
	if dm.content, err = dm.viewport.GetPrimaryContent(); err != nil {
		return nil, err
	}

	if err := dm.manager.Activate(hwnd); err != nil {
		return nil, err
	}
	dm.activated = true
	if err := dm.viewport.Enable(); err != nil {
		return nil, err
	}
	if err := dm.updateManager.Update(); err != nil {
		return nil, err
	}

	return dm, nil
}

func (d *directManipulation) release(hwnd windows.HWND) {
	// Errors are ignored, as nothing can be done for them while the window is being destroyed.
	if d.viewport != nil {
		_ = d.viewport.Abandon()
	}
	if d.activated {
		_ = d.manager.Deactivate(hwnd)
		d.activated = false
	}
	if d.content != nil {
		d.content.Release()
		d.content = nil
	}
	if d.viewport != nil {
		d.viewport.Release()
		d.viewport = nil
	}
	if d.updateManager != nil {
		d.updateManager.Release()
		d.updateManager = nil
	}
	if d.manager != nil {
		d.manager.Release()
		d.manager = nil
	}
	if d.uninitializeCOM {
		windows.CoUninitialize()
		d.uninitializeCOM = false
	}
}

// disableDirectManipulation stops Direct Manipulation for the window, if it is used.
//
// Direct Manipulation is optional, so a failure of it falls back to the messages without it instead of
// being an error: gestures on a touchpad are reported as WM_MOUSEWHEEL again, and no pinch is reported.
func (w *Window) disableDirectManipulation() {
	if w.platform.directManipulation == nil {
		return
	}
	w.platform.directManipulation.release(w.platform.handle)
	w.platform.directManipulation = nil
}

// update reports the changes of the viewport since the last call as scroll and pinch input.
func (d *directManipulation) update(w *Window) error {
	if err := d.updateManager.Update(); err != nil {
		return err
	}
	status, err := d.viewport.GetStatus()
	if err != nil {
		return err
	}
	m, err := d.content.GetContentTransform()
	if err != nil {
		return err
	}

	panX, panY, pinch := d.gesture.transform(m)
	if panX != 0 || panY != 0 {
		// The transform is in physical pixels. Report device-independent pixels, as macOS does.
		xscale, yscale, err := w.platformGetWindowContentScale()
		if err != nil {
			return err
		}
		panX /= float64(xscale)
		panY /= float64(yscale)
		// Match the wheel amounts of precise scrolling on macOS.
		w.inputScroll(panX*0.1, panY*0.1, panX, panY, ScrollUnitPixel)
	}
	if pinch != 0 {
		w.inputPinch(pinch)
	}

	// Reset the transform when a gesture is over, so that every gesture starts from the identity
	// transform and never reaches the zoom boundaries.
	if d.gesture.active() && status != _DIRECTMANIPULATION_RUNNING && status != _DIRECTMANIPULATION_INERTIA {
		d.gesture = newDirectManipulationGesture()
		if err := d.viewport.ZoomToRect(d.zoomToRect, 0, 0, directManipulationViewportSize, directManipulationViewportSize, false); err != nil {
			return err
		}
	}

	return nil
}

type directManipulationGestureKind int

const (
	directManipulationGestureNone directManipulationGestureKind = iota
	directManipulationGesturePan
	directManipulationGesturePinch
)

// directManipulationGesture turns the content transforms of one gesture into either a pan or a pinch.
type directManipulationGesture struct {
	kind  directManipulationGestureKind
	scale float32
	x     float32
	y     float32
}

func newDirectManipulationGesture() directManipulationGesture {
	return directManipulationGesture{
		scale: 1,
	}
}

func (g *directManipulationGesture) active() bool {
	return g.kind != directManipulationGestureNone
}

// transform takes the content transform and returns the pan in physical pixels and the pinch as the
// scale ratio minus one since the last transform. Either the pan or the pinch is zero.
func (g *directManipulationGesture) transform(m [6]float32) (panX, panY, pinch float64) {
	scale, x, y := m[0], m[4], m[5]

	// Windows can report a zero scale. Ignore it.
	if scale == 0 {
		return 0, 0, 0
	}
	if scale == g.scale && x == g.x && y == g.y {
		return 0, 0, 0
	}

	// A pinch also moves the content around the center of the fingers, so the translation is not
	// reported as a pan during a pinch. A pan can turn into a pinch, as the first changes of a slow
	// pinch can look like a pan, but not vice versa.
	if math.Abs(float64(scale)-1) < 1e-4 {
		if g.kind == directManipulationGestureNone {
			g.kind = directManipulationGesturePan
		}
	} else {
		g.kind = directManipulationGesturePinch
	}

	switch g.kind {
	case directManipulationGesturePan:
		panX = float64(x - g.x)
		panY = float64(y - g.y)
	case directManipulationGesturePinch:
		pinch = float64(scale)/float64(g.scale) - 1
	}

	g.scale = scale
	g.x = x
	g.y = y
	return panX, panY, pinch
}
