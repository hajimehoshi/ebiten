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
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

type dinputObjectType int

const (
	dinputObjectTypeAxis dinputObjectType = iota
	dinputObjectTypeSlider
	dinputObjectTypeButton
	dinputObjectTypePOV
)

var dinputObjectDataFormats = []_DIOBJECTDATAFORMAT{
	{&_GUID_XAxis, _DIJOFS_X, _DIDFT_AXIS | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, _DIDOI_ASPECTPOSITION},
	{&_GUID_YAxis, _DIJOFS_Y, _DIDFT_AXIS | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, _DIDOI_ASPECTPOSITION},
	{&_GUID_ZAxis, _DIJOFS_Z, _DIDFT_AXIS | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, _DIDOI_ASPECTPOSITION},
	{&_GUID_RxAxis, _DIJOFS_RX, _DIDFT_AXIS | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, _DIDOI_ASPECTPOSITION},
	{&_GUID_RyAxis, _DIJOFS_RY, _DIDFT_AXIS | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, _DIDOI_ASPECTPOSITION},
	{&_GUID_RzAxis, _DIJOFS_RZ, _DIDFT_AXIS | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, _DIDOI_ASPECTPOSITION},
	{&_GUID_Slider, _DIJOFS_SLIDER(0), _DIDFT_AXIS | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, _DIDOI_ASPECTPOSITION},
	{&_GUID_Slider, _DIJOFS_SLIDER(1), _DIDFT_AXIS | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, _DIDOI_ASPECTPOSITION},
	{&_GUID_POV, _DIJOFS_POV(0), _DIDFT_POV | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{&_GUID_POV, _DIJOFS_POV(1), _DIDFT_POV | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{&_GUID_POV, _DIJOFS_POV(2), _DIDFT_POV | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{&_GUID_POV, _DIJOFS_POV(3), _DIDFT_POV | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(0), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(1), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(2), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(3), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(4), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(5), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(6), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(7), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(8), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(9), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(10), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(11), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(12), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(13), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(14), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(15), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(16), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(17), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(18), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(19), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(20), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(21), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(22), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(23), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(24), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(25), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(26), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(27), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(28), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(29), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(30), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
	{nil, _DIJOFS_BUTTON(31), _DIDFT_BUTTON | _DIDFT_OPTIONAL | _DIDFT_ANYINSTANCE, 0},
}

var xinputButtons = []uint16{
	_XINPUT_GAMEPAD_A,
	_XINPUT_GAMEPAD_B,
	_XINPUT_GAMEPAD_X,
	_XINPUT_GAMEPAD_Y,
	_XINPUT_GAMEPAD_LEFT_SHOULDER,
	_XINPUT_GAMEPAD_RIGHT_SHOULDER,
	_XINPUT_GAMEPAD_BACK,
	_XINPUT_GAMEPAD_START,
	_XINPUT_GAMEPAD_LEFT_THUMB,
	_XINPUT_GAMEPAD_RIGHT_THUMB,
}

type nativeGamepadsDesktop struct {
	dinput8    windows.Handle
	dinput8API *_IDirectInput8W
	xinput     windows.Handle

	procDirectInput8Create    uintptr
	procXInputGetCapabilities uintptr
	procXInputGetState        uintptr

	origWndProc         uintptr
	wndProcCallback     uintptr
	enumDevicesCallback uintptr
	enumObjectsCallback uintptr

	nativeWindow  windows.HWND
	deviceChanged atomic.Bool
	err           error
}

type dinputObject struct {
	objectType dinputObjectType
	index      int
}

type enumObjectsContext struct {
	device      *_IDirectInputDevice8W
	objects     []dinputObject
	axisCount   int
	sliderCount int
	buttonCount int
	povCount    int
}

func (g *nativeGamepadsDesktop) init(gamepads *gamepads) error {
	// As there is no guarantee that the DLL exists, NewLazySystemDLL is not available.
	// TODO: Is there a 'system' version of LoadLibrary?
	if h, err := windows.LoadLibrary("dinput8.dll"); err == nil {
		g.dinput8 = h

		p, err := windows.GetProcAddress(h, "DirectInput8Create")
		if err != nil {
			return err
		}
		g.procDirectInput8Create = p
	}

	// TODO: Loading xinput1_4.dll or xinput9_1_0.dll should be enough.
	// See https://source.chromium.org/chromium/chromium/src/+/main:device/gamepad/xinput_data_fetcher_win.cc;l=75-84;drc=643cdf61903e99f27c3d80daee67e217e9d280e0
	for _, dll := range []string{
		"xinput1_4.dll",
		"xinput1_3.dll",
		"xinput9_1_0.dll",
		"xinput1_2.dll",
		"xinput1_1.dll",
	} {
		if h, err := windows.LoadLibrary(dll); err == nil {
			g.xinput = h
			{
				p, err := windows.GetProcAddress(h, "XInputGetCapabilities")
				if err != nil {
					return err
				}
				g.procXInputGetCapabilities = p
			}
			{
				p, err := windows.GetProcAddress(h, "XInputGetState")
				if err != nil {
					return err
				}
				g.procXInputGetState = p
			}
			break
		}
	}

	if g.dinput8 != 0 {
		// TODO: Use _GetModuleHandleExW to align with GLFW v3.3.8.
		m, err := _GetModuleHandleW()
		if err != nil {
			return err
		}

		var api *_IDirectInput8W
		if err := g.directInput8Create(m, _DIRECTINPUT_VERSION, &_IID_IDirectInput8W, &api, nil); err != nil {
			return err
		}
		g.dinput8API = api

		if err := g.detectConnection(gamepads); err != nil {
			return err
		}
	}

	return nil
}

func (g *nativeGamepadsDesktop) directInput8Create(hinst uintptr, dwVersion uint32, riidltf *windows.GUID, ppvOut **_IDirectInput8W, punkOuter unsafe.Pointer) error {
	r, _, _ := syscall.Syscall6(g.procDirectInput8Create, 5,
		hinst, uintptr(dwVersion), uintptr(unsafe.Pointer(riidltf)), uintptr(unsafe.Pointer(ppvOut)), uintptr(punkOuter),
		0)
	if uint32(r) != _DI_OK {
		return fmt.Errorf("gamepad: DirectInput8Create failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

func (g *nativeGamepadsDesktop) xinputGetCapabilities(dwUserIndex uint32, dwFlags uint32, pCapabilities *_XINPUT_CAPABILITIES) error {
	// XInputGetCapabilities doesn't call SetLastError and returns an error code directly.
	r, _, _ := syscall.Syscall(g.procXInputGetCapabilities, 3,
		uintptr(dwUserIndex), uintptr(dwFlags), uintptr(unsafe.Pointer(pCapabilities)))
	if e := syscall.Errno(uint32(r)); e != windows.ERROR_SUCCESS {
		return fmt.Errorf("gamepad: XInputGetCapabilities failed: %w", e)
	}
	return nil
}

func (g *nativeGamepadsDesktop) xinputGetState(dwUserIndex uint32, pState *_XINPUT_STATE) error {
	// XInputGetState doesn't call SetLastError and returns an error code directly.
	r, _, _ := syscall.Syscall(g.procXInputGetState, 2,
		uintptr(dwUserIndex), uintptr(unsafe.Pointer(pState)), 0)
	if e := syscall.Errno(uint32(r)); e != windows.ERROR_SUCCESS {
		return fmt.Errorf("gamepad: XInputGetCapabilities failed: %w", e)
	}
	return nil
}

func (g *nativeGamepadsDesktop) detectConnection(gamepads *gamepads) error {
	if g.dinput8 != 0 {
		if g.enumDevicesCallback == 0 {
			g.enumDevicesCallback = windows.NewCallback(g.dinput8EnumDevicesCallback)
		}
		if err := g.dinput8API.EnumDevices(_DI8DEVCLASS_GAMECTRL, g.enumDevicesCallback, unsafe.Pointer(gamepads), _DIEDFL_ALLDEVICES); err != nil {
			return err
		}
		if g.err != nil {
			return g.err
		}
	}
	if g.xinput != 0 {
		const xuserMaxCount = 4

		for i := 0; i < xuserMaxCount; i++ {
			if gamepads.find(func(g *Gamepad) bool {
				n := g.native.(*nativeGamepadDesktop)
				return n.dinputDevice == nil && n.xinputIndex == i
			}) != nil {
				continue
			}

			var xic _XINPUT_CAPABILITIES
			if err := g.xinputGetCapabilities(uint32(i), 0, &xic); err != nil {
				if !errors.Is(err, windows.ERROR_DEVICE_NOT_CONNECTED) {
					return err
				}
				continue
			}

			sdlID := fmt.Sprintf("78696e707574%02x000000000000000000", xic.subType&0xff)
			name := "Unknown XInput Device"
			switch xic.subType {
			case _XINPUT_DEVSUBTYPE_GAMEPAD:
				if xic.flags&_XINPUT_CAPS_WIRELESS != 0 {
					name = "Wireless Xbox Controller"
				} else {
					name = "Xbox Controller"
				}
			case _XINPUT_DEVSUBTYPE_WHEEL:
				name = "XInput Wheel"
			case _XINPUT_DEVSUBTYPE_ARCADE_STICK:
				name = "XInput Arcade Stick"
			case _XINPUT_DEVSUBTYPE_FLIGHT_STICK:
				name = "XInput Flight Stick"
			case _XINPUT_DEVSUBTYPE_DANCE_PAD:
				name = "XInput Dance Pad"
			case _XINPUT_DEVSUBTYPE_GUITAR:
				name = "XInput Guitar"
			case _XINPUT_DEVSUBTYPE_DRUM_KIT:
				name = "XInput Drum Kit"
			}

			gp := gamepads.add(name, sdlID)
			gp.native = &nativeGamepadDesktop{
				xinputIndex: i,
			}
		}
	}
	return nil
}

func (g *nativeGamepadsDesktop) dinput8EnumDevicesCallback(lpddi *_DIDEVICEINSTANCEW, pvRef unsafe.Pointer) uintptr {
	gamepads := (*gamepads)(pvRef)

	if g.err != nil {
		return _DIENUM_STOP
	}

	s, err := supportsXInput(lpddi.guidProduct)
	if err != nil {
		g.err = err
		return _DIENUM_STOP
	}
	if s {
		return _DIENUM_CONTINUE
	}

	var device *_IDirectInputDevice8W
	if err := g.dinput8API.CreateDevice(&lpddi.guidInstance, &device, nil); err != nil {
		g.err = err
		return _DIENUM_STOP
	}

	// lpddi.guidInstance is not reliable as a unique identity when the same multiple devices are connected (#3046).
	// Use HID Path instead.
	getDInputPath := func(device *_IDirectInputDevice8W) (string, error) {
		prop := _DIPROPGUIDANDPATH{
			diph: _DIPROPHEADER{
				dwHeaderSize: uint32(unsafe.Sizeof(_DIPROPHEADER{})),
				dwSize:       uint32(unsafe.Sizeof(_DIPROPGUIDANDPATH{})),
				dwHow:        _DIPH_DEVICE,
			},
		}
		if err := device.GetProperty(_DIPROP_GUIDANDPATH, &prop.diph); err != nil {
			return "", err
		}
		return windows.UTF16ToString(prop.wszPath[:]), nil
	}
	dinputPath, err := getDInputPath(device)
	if err != nil {
		g.err = err
		device.Release()
		return _DIENUM_STOP
	}

	var findErr error
	if gamepads.find(func(g *Gamepad) bool {
		// A DInput device can be nil when the device is an XInput device (#3047).
		d := g.native.(*nativeGamepadDesktop).dinputDevice
		if d == nil {
			return false
		}
		path, err := getDInputPath(d)
		if err != nil {
			findErr = err
			return true
		}
		return path == dinputPath
	}) != nil {
		if findErr != nil {
			g.err = findErr
			device.Release()
			return _DIENUM_STOP
		}
		device.Release()
		return _DIENUM_CONTINUE
	}

	dataFormat := _DIDATAFORMAT{
		dwSize:     uint32(unsafe.Sizeof(_DIDATAFORMAT{})),
		dwObjSize:  uint32(unsafe.Sizeof(_DIOBJECTDATAFORMAT{})),
		dwFlags:    _DIDFT_ABSAXIS,
		dwDataSize: uint32(unsafe.Sizeof(_DIJOYSTATE{})),
		dwNumObjs:  uint32(len(dinputObjectDataFormats)),
		rgodf:      &dinputObjectDataFormats[0],
	}
	if err := device.SetDataFormat(&dataFormat); err != nil {
		g.err = err
		device.Release()
		return _DIENUM_STOP
	}

	dc := _DIDEVCAPS{
		dwSize: uint32(unsafe.Sizeof(_DIDEVCAPS{})),
	}
	if err := device.GetCapabilities(&dc); err != nil {
		g.err = err
		device.Release()
		return _DIENUM_STOP
	}

	dipd := _DIPROPDWORD{
		diph: _DIPROPHEADER{
			dwSize:       uint32(unsafe.Sizeof(_DIPROPDWORD{})),
			dwHeaderSize: uint32(unsafe.Sizeof(_DIPROPHEADER{})),
			dwHow:        _DIPH_DEVICE,
		},
		dwData: _DIPROPAXISMODE_ABS,
	}
	if err := device.SetProperty(_DIPROP_AXISMODE, &dipd.diph); err != nil {
		g.err = err
		device.Release()
		return _DIENUM_STOP
	}

	ctx := enumObjectsContext{
		device: device,
	}
	if g.enumObjectsCallback == 0 {
		g.enumObjectsCallback = windows.NewCallback(g.dinputDevice8EnumObjectsCallback)
	}
	if err := device.EnumObjects(g.enumObjectsCallback, unsafe.Pointer(&ctx), _DIDFT_AXIS|_DIDFT_BUTTON|_DIDFT_POV); err != nil {
		g.err = err
		device.Release()
		return _DIENUM_STOP
	}

	sort.Slice(ctx.objects, func(i, j int) bool {
		if ctx.objects[i].objectType != ctx.objects[j].objectType {
			return ctx.objects[i].objectType < ctx.objects[j].objectType
		}
		return ctx.objects[i].index < ctx.objects[j].index
	})

	name := windows.UTF16ToString(lpddi.tszInstanceName[:])
	var sdlID string
	if string(lpddi.guidProduct.Data4[2:8]) == "PIDVID" {
		// This seems different from the current SDL implementation.
		// Probably guidProduct includes the vendor and the product information, but this works.
		// From the game controller database, the 'version' part seems always 0.
		sdlID = fmt.Sprintf("03000000%02x%02x0000%02x%02x000000000000",
			byte(lpddi.guidProduct.Data1),
			byte(lpddi.guidProduct.Data1>>8),
			byte(lpddi.guidProduct.Data1>>16),
			byte(lpddi.guidProduct.Data1>>24))
	} else {
		bs := []byte(name)
		if len(bs) < 12 {
			bs = append(bs, make([]byte, 12-len(bs))...)
		}
		sdlID = fmt.Sprintf("05000000%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x",
			bs[0], bs[1], bs[2], bs[3], bs[4], bs[5], bs[6], bs[7], bs[8], bs[9], bs[10], bs[11])
	}

	gp := gamepads.add(name, sdlID)
	gp.native = &nativeGamepadDesktop{
		dinputDevice:  device,
		dinputObjects: ctx.objects,
		dinputPath:    dinputPath,
		dinputAxes:    make([]float64, ctx.axisCount+ctx.sliderCount),
		dinputButtons: make([]bool, ctx.buttonCount),
		dinputHats:    make([]int, ctx.povCount),
	}

	return _DIENUM_CONTINUE
}

func supportsXInput(guid windows.GUID) (bool, error) {
	var count uint32
	if r, err := _GetRawInputDeviceList(nil, &count); err != nil {
		return false, err
	} else if r != 0 {
		return false, nil
	}

	if count == 0 {
		return false, nil
	}

	ridl := make([]_RAWINPUTDEVICELIST, count)
	if _, err := _GetRawInputDeviceList(&ridl[0], &count); err != nil {
		return false, err
	}

	for i := 0; i < int(count); i++ {
		if ridl[i].dwType != _RIM_TYPEHID {
			continue
		}

		rdi := _RID_DEVICE_INFO{
			cbSize: uint32(unsafe.Sizeof(_RID_DEVICE_INFO{})),
		}
		size := uint32(unsafe.Sizeof(rdi))
		if _, err := _GetRawInputDeviceInfoW(ridl[i].hDevice, _RIDI_DEVICEINFO, unsafe.Pointer(&rdi), &size); err != nil {
			// GetRawInputDeviceInfoW can return an error (#2603).
			continue
		}

		if uint32(rdi.hid.dwVendorId)|(uint32(rdi.hid.dwProductId)<<16) != guid.Data1 {
			continue
		}

		var name [256]uint16
		size = uint32(unsafe.Sizeof(name))
		if _, err := _GetRawInputDeviceInfoW(ridl[i].hDevice, _RIDI_DEVICENAME, unsafe.Pointer(&name[0]), &size); err != nil {
			return false, err
		}

		if strings.Contains(windows.UTF16ToString(name[:]), "IG_") {
			return true, nil
		}
	}

	return false, nil
}

func (g *nativeGamepadsDesktop) dinputDevice8EnumObjectsCallback(lpddoi *_DIDEVICEOBJECTINSTANCEW, pvRef unsafe.Pointer) uintptr {
	ctx := (*enumObjectsContext)(pvRef)

	switch {
	case _DIDFT_GETTYPE(lpddoi.dwType)&_DIDFT_AXIS != 0:
		var index int
		switch lpddoi.guidType {
		case _GUID_Slider:
			index = ctx.sliderCount
		case _GUID_XAxis:
			index = 0
		case _GUID_YAxis:
			index = 1
		case _GUID_ZAxis:
			index = 2
		case _GUID_RxAxis:
			index = 3
		case _GUID_RyAxis:
			index = 4
		case _GUID_RzAxis:
			index = 5
		default:
			return _DIENUM_CONTINUE
		}

		dipr := _DIPROPRANGE{
			diph: _DIPROPHEADER{
				dwSize:       uint32(unsafe.Sizeof(_DIPROPRANGE{})),
				dwHeaderSize: uint32(unsafe.Sizeof(_DIPROPHEADER{})),
				dwObj:        lpddoi.dwType,
				dwHow:        _DIPH_BYID,
			},
			lMin: -32768,
			lMax: 32767,
		}
		if err := ctx.device.SetProperty(_DIPROP_RANGE, &dipr.diph); err != nil {
			return _DIENUM_CONTINUE
		}

		var objectType dinputObjectType
		if lpddoi.guidType == _GUID_Slider {
			objectType = dinputObjectTypeSlider
			ctx.sliderCount++
		} else {
			objectType = dinputObjectTypeAxis
			ctx.axisCount++
		}
		ctx.objects = append(ctx.objects, dinputObject{
			objectType: objectType,
			index:      index,
		})
	case _DIDFT_GETTYPE(lpddoi.dwType)&_DIDFT_BUTTON != 0:
		ctx.objects = append(ctx.objects, dinputObject{
			objectType: dinputObjectTypeButton,
			index:      ctx.buttonCount,
		})
		ctx.buttonCount++
	case _DIDFT_GETTYPE(lpddoi.dwType)&_DIDFT_POV != 0:
		ctx.objects = append(ctx.objects, dinputObject{
			objectType: dinputObjectTypePOV,
			index:      ctx.povCount,
		})
		ctx.povCount++
	}

	return _DIENUM_CONTINUE
}

func (g *nativeGamepadsDesktop) update(gamepads *gamepads) error {
	if g.err != nil {
		return g.err
	}
	if g.origWndProc == 0 {
		if g.wndProcCallback == 0 {
			g.wndProcCallback = windows.NewCallback(g.wndProc)
		}
		// Note that a Win32API GetActiveWindow doesn't work on Xbox.
		h, err := _SetWindowLongPtrW(g.nativeWindow, _GWL_WNDPROC, g.wndProcCallback)
		if err != nil {
			return err
		}
		g.origWndProc = h
	}

	if g.deviceChanged.Load() {
		if err := g.detectConnection(gamepads); err != nil {
			g.err = err
		}
		g.deviceChanged.Store(false)
	}

	return nil
}

func (g *nativeGamepadsDesktop) wndProc(hWnd uintptr, uMsg uint32, wParam, lParam uintptr) uintptr {
	switch uMsg {
	case _WM_DEVICECHANGE:
		g.deviceChanged.Store(true)
	}
	return _CallWindowProcW(g.origWndProc, hWnd, uMsg, wParam, lParam)
}

func (g *nativeGamepadsDesktop) setNativeWindow(nativeWindow uintptr) {
	g.nativeWindow = windows.HWND(nativeWindow)
}

type nativeGamepadDesktop struct {
	dinputDevice  *_IDirectInputDevice8W
	dinputObjects []dinputObject
	dinputPath    string
	dinputAxes    []float64
	dinputButtons []bool
	dinputHats    []int

	xinputIndex int
	xinputState _XINPUT_STATE
}

func (*nativeGamepadDesktop) hasOwnStandardLayoutMapping() bool {
	return false
}

func (*nativeGamepadDesktop) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	return nil
}

func (*nativeGamepadDesktop) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	return nil
}

func (g *nativeGamepadDesktop) usesDInput() bool {
	return g.dinputDevice != nil
}

func (g *nativeGamepadDesktop) update(gamepads *gamepads) (err error) {
	var disconnected bool
	defer func() {
		if !disconnected && err == nil {
			return
		}
		gamepads.remove(func(gamepad *Gamepad) bool {
			return gamepad.native == g
		})
		if g.dinputDevice != nil {
			g.dinputDevice.Release()
		}
	}()

	if g.usesDInput() {
		if err := g.dinputDevice.Poll(); err != nil {
			if !errors.Is(err, handleError(_DIERR_NOTACQUIRED)) && !errors.Is(err, handleError(_DIERR_INPUTLOST)) {
				return err
			}
		}

		var state _DIJOYSTATE
		if err := g.dinputDevice.GetDeviceState(uint32(unsafe.Sizeof(state)), unsafe.Pointer(&state)); err != nil {
			if !errors.Is(err, handleError(_DIERR_NOTACQUIRED)) && !errors.Is(err, handleError(_DIERR_INPUTLOST)) {
				return err
			}
			// Acquire can return an error just after a gamepad is disconnected. Ignore the error.
			_ = g.dinputDevice.Acquire()
			if err := g.dinputDevice.Poll(); err != nil {
				if !errors.Is(err, handleError(_DIERR_NOTACQUIRED)) && !errors.Is(err, handleError(_DIERR_INPUTLOST)) {
					return err
				}
			}
			if err := g.dinputDevice.GetDeviceState(uint32(unsafe.Sizeof(state)), unsafe.Pointer(&state)); err != nil {
				if !errors.Is(err, handleError(_DIERR_NOTACQUIRED)) && !errors.Is(err, handleError(_DIERR_INPUTLOST)) {
					return err
				}
				disconnected = true
				return nil
			}
		}

		var ai, bi, hi int
		for _, obj := range g.dinputObjects {
			switch obj.objectType {
			case dinputObjectTypeAxis:
				var v int32
				switch obj.index {
				case 0:
					v = state.lX
				case 1:
					v = state.lY
				case 2:
					v = state.lZ
				case 3:
					v = state.lRx
				case 4:
					v = state.lRy
				case 5:
					v = state.lRz
				}
				g.dinputAxes[ai] = (float64(v) + 0.5) / 32767.5
				ai++
			case dinputObjectTypeSlider:
				v := state.rglSlider[obj.index]
				g.dinputAxes[ai] = (float64(v) + 0.5) / 32767.5
				ai++
			case dinputObjectTypeButton:
				v := (state.rgbButtons[obj.index] & 0x80) != 0
				g.dinputButtons[bi] = v
				bi++
			case dinputObjectTypePOV:
				stateIndex := state.rgdwPOV[obj.index] / (45 * _DI_DEGREES)
				v := hatCentered
				switch stateIndex {
				case 0:
					v = hatUp
				case 1:
					v = hatRightUp
				case 2:
					v = hatRight
				case 3:
					v = hatRightDown
				case 4:
					v = hatDown
				case 5:
					v = hatLeftDown
				case 6:
					v = hatLeft
				case 7:
					v = hatLeftUp
				}
				g.dinputHats[hi] = v
				hi++
			}
		}
		return nil
	}

	var state _XINPUT_STATE
	if err := gamepads.native.(*nativeGamepadsDesktop).xinputGetState(uint32(g.xinputIndex), &state); err != nil {
		if !errors.Is(err, windows.ERROR_DEVICE_NOT_CONNECTED) {
			return err
		}
		disconnected = true
		return nil
	}
	g.xinputState = state
	return nil
}

func (g *nativeGamepadDesktop) axisCount() int {
	if g.usesDInput() {
		return len(g.dinputAxes)
	}
	return 6
}

func (g *nativeGamepadDesktop) buttonCount() int {
	if g.usesDInput() {
		return len(g.dinputButtons)
	}
	return len(xinputButtons)
}

func (g *nativeGamepadDesktop) hatCount() int {
	if g.usesDInput() {
		return len(g.dinputHats)
	}
	return 1
}

func (g *nativeGamepadDesktop) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
}

func (g *nativeGamepadDesktop) axisValue(axis int) float64 {
	if g.usesDInput() {
		if axis < 0 || axis >= len(g.dinputAxes) {
			return 0
		}
		return g.dinputAxes[axis]
	}

	var v float64
	switch axis {
	case 0:
		v = (float64(g.xinputState.Gamepad.sThumbLX) + 0.5) / 32767.5
	case 1:
		v = -(float64(g.xinputState.Gamepad.sThumbLY) + 0.5) / 32767.5
	case 2:
		v = (float64(g.xinputState.Gamepad.sThumbRX) + 0.5) / 32767.5
	case 3:
		v = -(float64(g.xinputState.Gamepad.sThumbRY) + 0.5) / 32767.5
	case 4:
		v = float64(g.xinputState.Gamepad.bLeftTrigger)/127.5 - 1.0
	case 5:
		v = float64(g.xinputState.Gamepad.bRightTrigger)/127.5 - 1.0
	}
	return v
}

func (g *nativeGamepadDesktop) isButtonPressed(button int) bool {
	if g.usesDInput() {
		if button < 0 || button >= len(g.dinputButtons) {
			return false
		}
		return g.dinputButtons[button]
	}

	if button < 0 || button >= len(xinputButtons) {
		return false
	}
	return g.xinputState.Gamepad.wButtons&xinputButtons[button] != 0
}

func (g *nativeGamepadDesktop) buttonValue(button int) float64 {
	if g.isButtonPressed(button) {
		return 1
	}
	return 0
}

func (g *nativeGamepadDesktop) hatState(hat int) int {
	if g.usesDInput() {
		if hat < 0 || hat >= len(g.dinputHats) {
			return 0
		}
		return g.dinputHats[hat]
	}

	if hat != 0 {
		return 0
	}
	var v int
	if g.xinputState.Gamepad.wButtons&_XINPUT_GAMEPAD_DPAD_UP != 0 {
		v |= hatUp
	}
	if g.xinputState.Gamepad.wButtons&_XINPUT_GAMEPAD_DPAD_RIGHT != 0 {
		v |= hatRight
	}
	if g.xinputState.Gamepad.wButtons&_XINPUT_GAMEPAD_DPAD_DOWN != 0 {
		v |= hatDown
	}
	if g.xinputState.Gamepad.wButtons&_XINPUT_GAMEPAD_DPAD_LEFT != 0 {
		v |= hatLeft
	}

	// Treat invalid combinations as neither being pressed
	// while preserving what data can be preserved
	if (v&hatRight) != 0 && (v&hatLeft) != 0 {
		v &^= hatRight | hatLeft
	}
	if (v&hatUp) != 0 && (v&hatDown) != 0 {
		v &^= hatUp | hatDown
	}

	return v
}

func (g *nativeGamepadDesktop) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	// TODO: Implement this (#1452)
}
