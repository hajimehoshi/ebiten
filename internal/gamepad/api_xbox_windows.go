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

package gamepad

import (
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

var (
	gameInput = windows.NewLazySystemDLL("GameInput.dll")

	procGameInputCreate = gameInput.NewProc("GameInputCreate")
)

type _GameInputCallbackToken uint64

type _GameInputDeviceStatus int32

const (
	_GameInputDeviceNoStatus      _GameInputDeviceStatus = 0x00000000
	_GameInputDeviceConnected     _GameInputDeviceStatus = 0x00000001
	_GameInputDeviceInputEnabled  _GameInputDeviceStatus = 0x00000002
	_GameInputDeviceOutputEnabled _GameInputDeviceStatus = 0x00000004
	_GameInputDeviceRawIoEnabled  _GameInputDeviceStatus = 0x00000008
	_GameInputDeviceAudioCapture  _GameInputDeviceStatus = 0x00000010
	_GameInputDeviceAudioRender   _GameInputDeviceStatus = 0x00000020
	_GameInputDeviceSynchronized  _GameInputDeviceStatus = 0x00000040
	_GameInputDeviceWireless      _GameInputDeviceStatus = 0x00000080
	_GameInputDeviceUserIdle      _GameInputDeviceStatus = 0x00100000
	_GameInputDeviceAnyStatus     _GameInputDeviceStatus = 0x00FFFFFF
)

type _GameInputEnumerationKind int32

const (
	_GameInputNoEnumeration       _GameInputEnumerationKind = 0
	_GameInputAsyncEnumeration    _GameInputEnumerationKind = 1
	_GameInputBlockingEnumeration _GameInputEnumerationKind = 2
)

type _GameInputGamepadButtons int32

const (
	_GameInputGamepadNone            _GameInputGamepadButtons = 0x00000000
	_GameInputGamepadMenu            _GameInputGamepadButtons = 0x00000001
	_GameInputGamepadView            _GameInputGamepadButtons = 0x00000002
	_GameInputGamepadA               _GameInputGamepadButtons = 0x00000004
	_GameInputGamepadB               _GameInputGamepadButtons = 0x00000008
	_GameInputGamepadX               _GameInputGamepadButtons = 0x00000010
	_GameInputGamepadY               _GameInputGamepadButtons = 0x00000020
	_GameInputGamepadDPadUp          _GameInputGamepadButtons = 0x00000040
	_GameInputGamepadDPadDown        _GameInputGamepadButtons = 0x00000080
	_GameInputGamepadDPadLeft        _GameInputGamepadButtons = 0x00000100
	_GameInputGamepadDPadRight       _GameInputGamepadButtons = 0x00000200
	_GameInputGamepadLeftShoulder    _GameInputGamepadButtons = 0x00000400
	_GameInputGamepadRightShoulder   _GameInputGamepadButtons = 0x00000800
	_GameInputGamepadLeftThumbstick  _GameInputGamepadButtons = 0x00001000
	_GameInputGamepadRightThumbstick _GameInputGamepadButtons = 0x00002000
)

type _GameInputKind int32

const (
	_GameInputKindUnknown          _GameInputKind = 0x00000000
	_GameInputKindRawDeviceReport  _GameInputKind = 0x00000001
	_GameInputKindControllerAxis   _GameInputKind = 0x00000002
	_GameInputKindControllerButton _GameInputKind = 0x00000004
	_GameInputKindControllerSwitch _GameInputKind = 0x00000008
	_GameInputKindController       _GameInputKind = 0x0000000E
	_GameInputKindKeyboard         _GameInputKind = 0x00000010
	_GameInputKindMouse            _GameInputKind = 0x00000020
	_GameInputKindTouch            _GameInputKind = 0x00000100
	_GameInputKindMotion           _GameInputKind = 0x00001000
	_GameInputKindArcadeStick      _GameInputKind = 0x00010000
	_GameInputKindFlightStick      _GameInputKind = 0x00020000
	_GameInputKindGamepad          _GameInputKind = 0x00040000
	_GameInputKindRacingWheel      _GameInputKind = 0x00080000
	_GameInputKindUiNavigation     _GameInputKind = 0x01000000
	_GameInputKindAny              _GameInputKind = 0x0FFFFFFF
)

type _GameInputGamepadState struct {
	buttons          _GameInputGamepadButtons
	leftTrigger      float32
	rightTrigger     float32
	leftThumbstickX  float32
	leftThumbstickY  float32
	rightThumbstickX float32
	rightThumbstickY float32
}

type _GameInputRumbleParams struct {
	lowFrequency  float32
	highFrequency float32
	leftTrigger   float32
	rightTrigger  float32
}

func _GameInputCreate() (*_IGameInput, error) {
	var gameInput *_IGameInput
	r, _, _ := procGameInputCreate.Call(uintptr(unsafe.Pointer(&gameInput)))
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("gamepad: GameInputCreate failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return gameInput, nil
}

type _IGameInput struct {
	vtbl *_IGameInput_Vtbl
}

type _IGameInput_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	GetCurrentTimestamp            uintptr
	GetCurrentReading              uintptr
	GetNextReading                 uintptr
	GetPreviousReading             uintptr
	GetTemporalReading             uintptr
	RegisterReadingCallback        uintptr
	RegisterDeviceCallback         uintptr
	RegisterGuideButtonCallback    uintptr
	RegisterKeyboardLayoutCallback uintptr
	StopCallback                   uintptr
	UnregisterCallback             uintptr
	CreateDispatcher               uintptr
	CreateAggregateDevice          uintptr
	FindDeviceFromId               uintptr
	FindDeviceFromObject           uintptr
	FindDeviceFromPlatformHandle   uintptr
	FindDeviceFromPlatformString   uintptr
	EnableOemDeviceSupport         uintptr
	SetFocusPolicy                 uintptr
}

func (i *_IGameInput) GetCurrentReading(inputKind _GameInputKind, device *_IGameInputDevice) (*_IGameInputReading, error) {
	var reading *_IGameInputReading
	r, _, _ := syscall.Syscall6(i.vtbl.GetCurrentReading, 4, uintptr(unsafe.Pointer(i)),
		uintptr(inputKind), uintptr(unsafe.Pointer(device)), uintptr(unsafe.Pointer(&reading)),
		0, 0)
	runtime.KeepAlive(device)
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("gamepad: IGameInput::GetCurrentReading failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return reading, nil
}

func (i *_IGameInput) RegisterDeviceCallback(device *_IGameInputDevice,
	inputKind _GameInputKind,
	statusFilter _GameInputDeviceStatus,
	enumerationKind _GameInputEnumerationKind,
	context unsafe.Pointer,
	callbackFunc uintptr,
	callbackToken *_GameInputCallbackToken) error {
	r, _, _ := syscall.Syscall9(i.vtbl.RegisterDeviceCallback, 8, uintptr(unsafe.Pointer(i)),
		uintptr(unsafe.Pointer(device)), uintptr(inputKind), uintptr(statusFilter),
		uintptr(enumerationKind), uintptr(context), callbackFunc,
		uintptr(unsafe.Pointer(callbackToken)), 0)
	runtime.KeepAlive(device)
	runtime.KeepAlive(callbackToken)
	if uint32(r) != uint32(windows.S_OK) {
		return fmt.Errorf("gamepad: IGameInput::RegisterDeviceCallback failed: %w", handleError(windows.Handle(uint32(r))))
	}
	return nil
}

type _IGameInputDevice struct {
	vtbl *_IGameInputDevice_Vtbl
}

type _IGameInputDevice_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	GetDeviceInfo                   uintptr
	GetDeviceStatus                 uintptr
	GetBatteryState                 uintptr
	CreateForceFeedbackEffect       uintptr
	IsForceFeedbackMotorPoweredOn   uintptr
	SetForceFeedbackMotorGain       uintptr
	SetHapticMotorState             uintptr
	SetRumbleState                  uintptr
	SetInputSynchronizationState    uintptr
	SendInputSynchronizationHint    uintptr
	PowerOff                        uintptr
	CreateRawDeviceReport           uintptr
	GetRawDeviceFeature             uintptr
	SetRawDeviceFeature             uintptr
	SendRawDeviceOutput             uintptr
	ExecuteRawDeviceIoControl       uintptr
	AcquireExclusiveRawDeviceAccess uintptr
	ReleaseExclusiveRawDeviceAccess uintptr
}

func (i *_IGameInputDevice) SetRumbleState(params *_GameInputRumbleParams, timestamp uint64) {
	_, _, _ = syscall.Syscall(i.vtbl.SetRumbleState, 3, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(params)), uintptr(timestamp))
	runtime.KeepAlive(params)
}

type _IGameInputReading struct {
	vtbl *_IGameInputReading_Vtbl
}

type _IGameInputReading_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr

	GetInputKind             uintptr
	GetSequenceNumber        uintptr
	GetTimestamp             uintptr
	GetDevice                uintptr
	GetRawReport             uintptr
	GetControllerAxisCount   uintptr
	GetControllerAxisState   uintptr
	GetControllerButtonCount uintptr
	GetControllerButtonState uintptr
	GetControllerSwitchCount uintptr
	GetControllerSwitchState uintptr
	GetKeyCount              uintptr
	GetKeyState              uintptr
	GetMouseState            uintptr
	GetTouchCount            uintptr
	GetTouchState            uintptr
	GetMotionState           uintptr
	GetArcadeStickState      uintptr
	GetFlightStickState      uintptr
	GetGamepadState          uintptr
	GetRacingWheelState      uintptr
	GetUiNavigationState     uintptr
}

func (i *_IGameInputReading) GetGamepadState() (_GameInputGamepadState, bool) {
	var state _GameInputGamepadState
	r, _, _ := syscall.Syscall(i.vtbl.GetGamepadState, 2, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(&state)), 0)
	return state, int32(r) != 0
}

func (i *_IGameInputReading) Release() uint32 {
	r, _, _ := syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	return uint32(r)
}
