// Copyright 2022 The Ebiten Authors
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

package gamepad

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	_DI_OK           = 0
	_DI_NOEFFECT     = _SI_FALSE
	_DI_PROPNOEFFECT = _SI_FALSE

	_DI_DEGREES = 100

	_DI8DEVCLASS_GAMECTRL = 4

	_DIDFT_ABSAXIS     = 0x00000002
	_DIDFT_AXIS        = 0x00000003
	_DIDFT_BUTTON      = 0x0000000C
	_DIDFT_POV         = 0x00000010
	_DIDFT_OPTIONAL    = 0x80000000
	_DIDFT_ANYINSTANCE = 0x00FFFF00

	_DIDOI_ASPECTPOSITION = 0x00000100

	_DIEDFL_ALLDEVICES = 0x00000000

	_DIENUM_STOP     = 0
	_DIENUM_CONTINUE = 1

	_DIERR_INPUTLOST   = windows.SEVERITY_ERROR<<31 | windows.FACILITY_WIN32<<16 | windows.ERROR_READ_FAULT
	_DIERR_NOTACQUIRED = windows.SEVERITY_ERROR<<31 | windows.FACILITY_WIN32<<16 | windows.ERROR_INVALID_ACCESS

	_DIJOFS_X  = uint32(unsafe.Offsetof(_DIJOYSTATE{}.lX))
	_DIJOFS_Y  = uint32(unsafe.Offsetof(_DIJOYSTATE{}.lY))
	_DIJOFS_Z  = uint32(unsafe.Offsetof(_DIJOYSTATE{}.lZ))
	_DIJOFS_RX = uint32(unsafe.Offsetof(_DIJOYSTATE{}.lRx))
	_DIJOFS_RY = uint32(unsafe.Offsetof(_DIJOYSTATE{}.lRy))
	_DIJOFS_RZ = uint32(unsafe.Offsetof(_DIJOYSTATE{}.lRz))

	_DIPH_DEVICE = 0
	_DIPH_BYID   = 2

	_DIPROP_AXISMODE    = 2
	_DIPROP_GUIDANDPATH = 12
	_DIPROP_RANGE       = 4

	_DIPROPAXISMODE_ABS = 0

	_DIRECTINPUT_VERSION = 0x0800

	_GWL_WNDPROC = -4

	_MAX_PATH = 260

	_RIDI_DEVICEINFO = 0x2000000b
	_RIDI_DEVICENAME = 0x20000007

	_RIM_TYPEHID = 2

	_SI_FALSE = 1

	_WM_DEVICECHANGE = 0x0219

	_XINPUT_CAPS_WIRELESS = 0x0002

	_XINPUT_DEVSUBTYPE_GAMEPAD      = 0x01
	_XINPUT_DEVSUBTYPE_WHEEL        = 0x02
	_XINPUT_DEVSUBTYPE_ARCADE_STICK = 0x03
	_XINPUT_DEVSUBTYPE_FLIGHT_STICK = 0x04
	_XINPUT_DEVSUBTYPE_DANCE_PAD    = 0x05
	_XINPUT_DEVSUBTYPE_GUITAR       = 0x06
	_XINPUT_DEVSUBTYPE_DRUM_KIT     = 0x08

	_XINPUT_GAMEPAD_DPAD_UP        = 0x0001
	_XINPUT_GAMEPAD_DPAD_DOWN      = 0x0002
	_XINPUT_GAMEPAD_DPAD_LEFT      = 0x0004
	_XINPUT_GAMEPAD_DPAD_RIGHT     = 0x0008
	_XINPUT_GAMEPAD_START          = 0x0010
	_XINPUT_GAMEPAD_BACK           = 0x0020
	_XINPUT_GAMEPAD_LEFT_THUMB     = 0x0040
	_XINPUT_GAMEPAD_RIGHT_THUMB    = 0x0080
	_XINPUT_GAMEPAD_LEFT_SHOULDER  = 0x0100
	_XINPUT_GAMEPAD_RIGHT_SHOULDER = 0x0200
	_XINPUT_GAMEPAD_A              = 0x1000
	_XINPUT_GAMEPAD_B              = 0x2000
	_XINPUT_GAMEPAD_X              = 0x4000
	_XINPUT_GAMEPAD_Y              = 0x8000
)

func _DIDFT_GETTYPE(n uint32) byte {
	return byte(n)
}

func _DIJOFS_SLIDER(n int) uint32 {
	return uint32(unsafe.Offsetof(_DIJOYSTATE{}.rglSlider) + uintptr(n)*unsafe.Sizeof(int32(0)))
}

func _DIJOFS_POV(n int) uint32 {
	return uint32(unsafe.Offsetof(_DIJOYSTATE{}.rgdwPOV) + uintptr(n)*unsafe.Sizeof(uint32(0)))
}

func _DIJOFS_BUTTON(n int) uint32 {
	return uint32(unsafe.Offsetof(_DIJOYSTATE{}.rgbButtons) + uintptr(n))
}

var (
	_IID_IDirectInput8W = windows.GUID{Data1: 0xbf798031, Data2: 0x483a, Data3: 0x4da2, Data4: [...]byte{0xaa, 0x99, 0x5d, 0x64, 0xed, 0x36, 0x97, 0x00}}
	_GUID_XAxis         = windows.GUID{Data1: 0xa36d02e0, Data2: 0xc9f3, Data3: 0x11cf, Data4: [...]byte{0xbf, 0xc7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	_GUID_YAxis         = windows.GUID{Data1: 0xa36d02e1, Data2: 0xc9f3, Data3: 0x11cf, Data4: [...]byte{0xbf, 0xc7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	_GUID_ZAxis         = windows.GUID{Data1: 0xa36d02e2, Data2: 0xc9f3, Data3: 0x11cf, Data4: [...]byte{0xbf, 0xc7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	_GUID_RxAxis        = windows.GUID{Data1: 0xa36d02f4, Data2: 0xc9f3, Data3: 0x11cf, Data4: [...]byte{0xbf, 0xc7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	_GUID_RyAxis        = windows.GUID{Data1: 0xa36d02f5, Data2: 0xc9f3, Data3: 0x11cf, Data4: [...]byte{0xbf, 0xc7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	_GUID_RzAxis        = windows.GUID{Data1: 0xa36d02e3, Data2: 0xc9f3, Data3: 0x11cf, Data4: [...]byte{0xbf, 0xc7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	_GUID_Slider        = windows.GUID{Data1: 0xa36d02e4, Data2: 0xc9f3, Data3: 0x11cf, Data4: [...]byte{0xbf, 0xc7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
	_GUID_POV           = windows.GUID{Data1: 0xa36d02f2, Data2: 0xc9f3, Data3: 0x11cf, Data4: [...]byte{0xbf, 0xc7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
)

var (
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	user32   = windows.NewLazySystemDLL("user32.dll")

	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	procCallWindowProcW        = user32.NewProc("CallWindowProcW")
	procGetRawInputDeviceInfoW = user32.NewProc("GetRawInputDeviceInfoW")
	procGetRawInputDeviceList  = user32.NewProc("GetRawInputDeviceList")

	procSetWindowLongW    = user32.NewProc("SetWindowLongW")    // 32-Bit Windows version.
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW") // 64-Bit Windows version.
)

func _GetModuleHandleW() (uintptr, error) {
	m, _, e := procGetModuleHandleW.Call(0)
	if m == 0 {
		if e != nil && e != windows.ERROR_SUCCESS {
			return 0, fmt.Errorf("gamepad: GetModuleHandleW failed: %w", e)
		}
		return 0, fmt.Errorf("gamepad: GetModuleHandleW returned 0")
	}
	return m, nil
}

func _CallWindowProcW(lpPrevWndFunc uintptr, hWnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	r, _, _ := procCallWindowProcW.Call(lpPrevWndFunc, hWnd, uintptr(msg), wParam, lParam)
	return r
}

func _GetRawInputDeviceInfoW(hDevice windows.Handle, uiCommand uint32, pData unsafe.Pointer, pcb *uint32) (uint32, error) {
	r, _, e := procGetRawInputDeviceInfoW.Call(uintptr(hDevice), uintptr(uiCommand), uintptr(pData), uintptr(unsafe.Pointer(pcb)))
	if uint32(r) == ^uint32(0) {
		if e != nil && e != windows.ERROR_SUCCESS {
			return 0, fmt.Errorf("gamepad: GetRawInputDeviceInfoW failed: %w", e)
		}
		return 0, fmt.Errorf("gamepad: GetRawInputDeviceInfoW returned -1")
	}
	return uint32(r), nil
}

func _GetRawInputDeviceList(pRawInputDeviceList *_RAWINPUTDEVICELIST, puiNumDevices *uint32) (uint32, error) {
	r, _, e := procGetRawInputDeviceList.Call(uintptr(unsafe.Pointer(pRawInputDeviceList)), uintptr(unsafe.Pointer(puiNumDevices)), unsafe.Sizeof(_RAWINPUTDEVICELIST{}))
	if uint32(r) == ^uint32(0) {
		if e != nil && e != windows.ERROR_SUCCESS {
			return 0, fmt.Errorf("gamepad: GetRawInputDeviceList failed: %w", e)
		}
		return 0, fmt.Errorf("gamepad: GetRawInputDeviceList returned -1")
	}
	return uint32(r), nil
}

func _SetWindowLongPtrW(hWnd windows.HWND, nIndex int32, dwNewLong uintptr) (uintptr, error) {
	var p *windows.LazyProc
	if procSetWindowLongPtrW.Find() == nil {
		// 64-Bit Windows.
		p = procSetWindowLongPtrW
	} else {
		// 32-Bit Windows.
		p = procSetWindowLongW
	}
	h, _, e := p.Call(uintptr(hWnd), uintptr(nIndex), dwNewLong)
	if h == 0 {
		if e != nil && e != windows.ERROR_SUCCESS {
			return 0, fmt.Errorf("gamepad: SetWindowLongPtrW failed: %w", e)
		}
		return 0, fmt.Errorf("gamepad: SetWindowLongPtrW returned 0")
	}
	return h, nil
}

type _DIDATAFORMAT struct {
	dwSize     uint32
	dwObjSize  uint32
	dwFlags    uint32
	dwDataSize uint32
	dwNumObjs  uint32
	rgodf      *_DIOBJECTDATAFORMAT
}

type _DIDEVCAPS struct {
	dwSize                uint32
	dwFlags               uint32
	dwDevType             uint32
	dwAxes                uint32
	dwButtons             uint32
	dwPOVs                uint32
	dwFFSamplePeriod      uint32
	dwFFMinTimeResolution uint32
	dwFirmwareRevision    uint32
	dwHardwareRevision    uint32
	dwFFDriverVersion     uint32
}

type _DIDEVICEINSTANCEW struct {
	dwSize          uint32
	guidInstance    windows.GUID
	guidProduct     windows.GUID
	dwDevType       uint32
	tszInstanceName [_MAX_PATH]uint16
	tszProductName  [_MAX_PATH]uint16
	guidFFDriver    windows.GUID
	wUsagePage      uint16
	wUsage          uint16
}

type _DIDEVICEOBJECTINSTANCEW struct {
	dwSize              uint32
	guidType            windows.GUID
	dwOfs               uint32
	dwType              uint32
	dwFlags             uint32
	tszName             [_MAX_PATH]uint16
	dwFFMaxForce        uint32
	dwFFForceResolution uint32
	wCollectionNumber   uint16
	wDesignatorIndex    uint16
	wUsagePage          uint16
	wUsage              uint16
	dwDimension         uint32
	wExponent           uint16
	wReserved           uint16
}

type _DIJOYSTATE struct {
	lX         int32
	lY         int32
	lZ         int32
	lRx        int32
	lRy        int32
	lRz        int32
	rglSlider  [2]int32
	rgdwPOV    [4]uint32
	rgbButtons [32]byte
}

type _DIOBJECTDATAFORMAT struct {
	pguid   *windows.GUID
	dwOfs   uint32
	dwType  uint32
	dwFlags uint32
}

type _DIPROPDWORD struct {
	diph   _DIPROPHEADER
	dwData uint32
}

type _DIPROPGUIDANDPATH struct {
	diph      _DIPROPHEADER
	guidClass windows.GUID
	wszPath   [_MAX_PATH]uint16
}

type _DIPROPHEADER struct {
	dwSize       uint32
	dwHeaderSize uint32
	dwObj        uint32
	dwHow        uint32
}

type _DIPROPRANGE struct {
	diph _DIPROPHEADER
	lMin int32
	lMax int32
}

type _IDirectInput8W struct {
	vtbl *_IDirectInput8W_Vtbl
}

type _IDirectInput8W_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	CreateDevice           uintptr
	EnumDevices            uintptr
	GetDeviceStatus        uintptr
	RunControlPanel        uintptr
	Initialize             uintptr
	FindDevice             uintptr
	EnumDevicesBySemantics uintptr
	ConfigureDevices       uintptr
}

func (d *_IDirectInput8W) CreateDevice(rguid *windows.GUID, lplpDirectInputDevice **_IDirectInputDevice8W, pUnkOuter unsafe.Pointer) error {
	r, _, _ := syscall.Syscall6(d.vtbl.CreateDevice, 4,
		uintptr(unsafe.Pointer(d)),
		uintptr(unsafe.Pointer(rguid)), uintptr(unsafe.Pointer(lplpDirectInputDevice)), uintptr(pUnkOuter),
		0, 0)
	if uint32(r) != _DI_OK {
		return fmt.Errorf("gamepad: IDirectInput8::CreateDevice failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (d *_IDirectInput8W) EnumDevices(dwDevType uint32, lpCallback uintptr, pvRef unsafe.Pointer, dwFlags uint32) error {
	r, _, _ := syscall.Syscall6(d.vtbl.EnumDevices, 5,
		uintptr(unsafe.Pointer(d)),
		uintptr(dwDevType), lpCallback, uintptr(pvRef), uintptr(dwFlags),
		0)
	if uint32(r) != _DI_OK {
		return fmt.Errorf("gamepad: IDirectInput8::EnumDevices failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

type _IDirectInputDevice8W struct {
	vtbl *_IDirectInputDevice8W_Vtbl
}

type _IDirectInputDevice8W_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	GetCapabilities          uintptr
	EnumObjects              uintptr
	GetProperty              uintptr
	SetProperty              uintptr
	Acquire                  uintptr
	Unacquire                uintptr
	GetDeviceState           uintptr
	GetDeviceData            uintptr
	SetDataFormat            uintptr
	SetEventNotification     uintptr
	SetCooperativeLevel      uintptr
	GetObjectInfo            uintptr
	GetDeviceInfo            uintptr
	RunControlPanel          uintptr
	Initialize               uintptr
	CreateEffect             uintptr
	EnumEffects              uintptr
	GetEffectInfo            uintptr
	GetForceFeedbackState    uintptr
	SendForceFeedbackCommand uintptr
	EnumCreatedEffectObjects uintptr
	Escape                   uintptr
	Poll                     uintptr
	SendDeviceData           uintptr
	EnumEffectsInFile        uintptr
	WriteEffectToFile        uintptr
	BuildActionMap           uintptr
	SetActionMap             uintptr
	GetImageInfo             uintptr
}

func (d *_IDirectInputDevice8W) Acquire() error {
	r, _, _ := syscall.Syscall(d.vtbl.Acquire, 1, uintptr(unsafe.Pointer(d)), 0, 0)
	if uint32(r) != _DI_OK && uint32(r) != _SI_FALSE {
		return fmt.Errorf("gamepad: IDirectInputDevice8::Acquire failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (d *_IDirectInputDevice8W) EnumObjects(lpCallback uintptr, pvRef unsafe.Pointer, dwFlags uint32) error {
	r, _, _ := syscall.Syscall6(d.vtbl.EnumObjects, 4,
		uintptr(unsafe.Pointer(d)),
		lpCallback, uintptr(pvRef), uintptr(dwFlags),
		0, 0)
	if uint32(r) != _DI_OK {
		return fmt.Errorf("gamepad: IDirectInputDevice8::EnumObjects failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (d *_IDirectInputDevice8W) GetCapabilities(lpDIDevCaps *_DIDEVCAPS) error {
	r, _, _ := syscall.Syscall(d.vtbl.GetCapabilities, 2, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(lpDIDevCaps)), 0)
	if uint32(r) != _DI_OK {
		return fmt.Errorf("gamepad: IDirectInputDevice8::GetCapabilities failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (d *_IDirectInputDevice8W) GetDeviceState(cbData uint32, lpvData unsafe.Pointer) error {
	r, _, _ := syscall.Syscall(d.vtbl.GetDeviceState, 3, uintptr(unsafe.Pointer(d)), uintptr(cbData), uintptr(lpvData))
	if uint32(r) != _DI_OK {
		return fmt.Errorf("gamepad: IDirectInputDevice8::GetDeviceState failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (d *_IDirectInputDevice8W) GetProperty(rguidProp uintptr, pdiph *_DIPROPHEADER) error {
	r, _, _ := syscall.Syscall(d.vtbl.GetProperty, 3, uintptr(unsafe.Pointer(d)), rguidProp, uintptr(unsafe.Pointer(pdiph)))
	if uint32(r) != _DI_OK {
		return fmt.Errorf("gamepad: IDirectInputDevice8::GetProperty failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (d *_IDirectInputDevice8W) Poll() error {
	r, _, _ := syscall.Syscall(d.vtbl.Poll, 1, uintptr(unsafe.Pointer(d)), 0, 0)
	if uint32(r) != _DI_OK && uint32(r) != _DI_NOEFFECT {
		return fmt.Errorf("gamepad: IDirectInputDevice8::Poll failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (d *_IDirectInputDevice8W) Release() uint32 {
	r, _, _ := syscall.Syscall(d.vtbl.Release, 1, uintptr(unsafe.Pointer(d)), 0, 0)
	return uint32(r)
}

func (d *_IDirectInputDevice8W) SetDataFormat(lpdf *_DIDATAFORMAT) error {
	r, _, _ := syscall.Syscall(d.vtbl.SetDataFormat, 2, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(lpdf)), 0)
	if uint32(r) != _DI_OK {
		return fmt.Errorf("gamepad: IDirectInputDevice8::SetDataFormat failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (d *_IDirectInputDevice8W) SetProperty(rguidProp uintptr, pdiph *_DIPROPHEADER) error {
	r, _, _ := syscall.Syscall(d.vtbl.SetProperty, 3, uintptr(unsafe.Pointer(d)), rguidProp, uintptr(unsafe.Pointer(pdiph)))
	if uint32(r) != _DI_OK && uint32(r) != _DI_PROPNOEFFECT {
		return fmt.Errorf("gamepad: IDirectInputDevice8::SetProperty failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

type _RID_DEVICE_INFO struct {
	cbSize uint32
	dwType uint32
	hid    _RID_DEVICE_INFO_HID // Originally, this member is a union.
}

type _RID_DEVICE_INFO_HID struct {
	dwVendorId      uint32
	dwProductId     uint32
	dwVersionNumber uint32
	usUsagePage     uint16
	usUsage         uint16
	_               uint32 // A padding adjusting with the size of RID_DEVICE_INFO_KEYBOARD
	_               uint32 // A padding adjusting with the size of RID_DEVICE_INFO_KEYBOARD
}

type _RAWINPUTDEVICELIST struct {
	hDevice windows.Handle
	dwType  uint32
}

type _XINPUT_CAPABILITIES struct {
	typ       byte
	subType   byte
	flags     uint16
	gamepad   _XINPUT_GAMEPAD
	vibration _XINPUT_VIBRATION
}

type _XINPUT_GAMEPAD struct {
	wButtons      uint16
	bLeftTrigger  byte
	bRightTrigger byte
	sThumbLX      int16
	sThumbLY      int16
	sThumbRX      int16
	sThumbRY      int16
}

type _XINPUT_STATE struct {
	dwPacketNumber uint32
	Gamepad        _XINPUT_GAMEPAD
}

type _XINPUT_VIBRATION struct {
	wLeftMotorSpeed  uint16
	wRightMotorSpeed uint16
}
