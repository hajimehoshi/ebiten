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

//go:build !ebitenginecbackend && !ebitencbackend
// +build !ebitenginecbackend,!ebitencbackend

package gamepad

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gameInput = windows.NewLazySystemDLL("GameInput.dll")

	procGameInputCreate = gameInput.NewProc("GameInputCreate")
)

const _APP_LOCAL_DEVICE_ID_SIZE = 32

type _GameInputCallbackToken uint64

type _GameInputDeviceCapabilities int32

const (
	_GameInputDeviceCapabilityNone            _GameInputDeviceCapabilities = 0x00000000
	_GameInputDeviceCapabilityAudio           _GameInputDeviceCapabilities = 0x00000001
	_GameInputDeviceCapabilityPluginModule    _GameInputDeviceCapabilities = 0x00000002
	_GameInputDeviceCapabilityPowerOff        _GameInputDeviceCapabilities = 0x00000004
	_GameInputDeviceCapabilitySynchronization _GameInputDeviceCapabilities = 0x00000008
	_GameInputDeviceCapabilityWireless        _GameInputDeviceCapabilities = 0x00000010
)

type _GameInputDeviceFamily int32

const (
	_GameInputFamilyVirtual   _GameInputDeviceFamily = -1
	_GameInputFamilyAggregate _GameInputDeviceFamily = 0
	_GameInputFamilyXboxOne   _GameInputDeviceFamily = 1
	_GameInputFamilyXbox360   _GameInputDeviceFamily = 2
	_GameInputFamilyHid       _GameInputDeviceFamily = 3
	_GameInputFamilyI8042     _GameInputDeviceFamily = 4
)

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

type _GameInputFeedbackAxes int32

const (
	_GameInputFeedbackAxisNone     _GameInputFeedbackAxes = 0x00000000
	_GameInputFeedbackAxisLinearX  _GameInputFeedbackAxes = 0x00000001
	_GameInputFeedbackAxisLinearY  _GameInputFeedbackAxes = 0x00000002
	_GameInputFeedbackAxisLinearZ  _GameInputFeedbackAxes = 0x00000004
	_GameInputFeedbackAxisAngularX _GameInputFeedbackAxes = 0x00000008
	_GameInputFeedbackAxisAngularY _GameInputFeedbackAxes = 0x00000010
	_GameInputFeedbackAxisAngularZ _GameInputFeedbackAxes = 0x00000020
	_GameInputFeedbackAxisNormal   _GameInputFeedbackAxes = 0x00000040
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

type _GameInputLabel int32

const (
	_GameInputLabelUnknown                  _GameInputLabel = -1
	_GameInputLabelNone                     _GameInputLabel = 0
	_GameInputLabelXboxGuide                _GameInputLabel = 1
	_GameInputLabelXboxBack                 _GameInputLabel = 2
	_GameInputLabelXboxStart                _GameInputLabel = 3
	_GameInputLabelXboxMenu                 _GameInputLabel = 4
	_GameInputLabelXboxView                 _GameInputLabel = 5
	_GameInputLabelXboxA                    _GameInputLabel = 7
	_GameInputLabelXboxB                    _GameInputLabel = 8
	_GameInputLabelXboxX                    _GameInputLabel = 9
	_GameInputLabelXboxY                    _GameInputLabel = 10
	_GameInputLabelXboxDPadUp               _GameInputLabel = 11
	_GameInputLabelXboxDPadDown             _GameInputLabel = 12
	_GameInputLabelXboxDPadLeft             _GameInputLabel = 13
	_GameInputLabelXboxDPadRight            _GameInputLabel = 14
	_GameInputLabelXboxLeftShoulder         _GameInputLabel = 15
	_GameInputLabelXboxLeftTrigger          _GameInputLabel = 16
	_GameInputLabelXboxLeftStickButton      _GameInputLabel = 17
	_GameInputLabelXboxRightShoulder        _GameInputLabel = 18
	_GameInputLabelXboxRightTrigger         _GameInputLabel = 19
	_GameInputLabelXboxRightStickButton     _GameInputLabel = 20
	_GameInputLabelXboxPaddle1              _GameInputLabel = 21
	_GameInputLabelXboxPaddle2              _GameInputLabel = 22
	_GameInputLabelXboxPaddle3              _GameInputLabel = 23
	_GameInputLabelXboxPaddle4              _GameInputLabel = 24
	_GameInputLabelLetterA                  _GameInputLabel = 25
	_GameInputLabelLetterB                  _GameInputLabel = 26
	_GameInputLabelLetterC                  _GameInputLabel = 27
	_GameInputLabelLetterD                  _GameInputLabel = 28
	_GameInputLabelLetterE                  _GameInputLabel = 29
	_GameInputLabelLetterF                  _GameInputLabel = 30
	_GameInputLabelLetterG                  _GameInputLabel = 31
	_GameInputLabelLetterH                  _GameInputLabel = 32
	_GameInputLabelLetterI                  _GameInputLabel = 33
	_GameInputLabelLetterJ                  _GameInputLabel = 34
	_GameInputLabelLetterK                  _GameInputLabel = 35
	_GameInputLabelLetterL                  _GameInputLabel = 36
	_GameInputLabelLetterM                  _GameInputLabel = 37
	_GameInputLabelLetterN                  _GameInputLabel = 38
	_GameInputLabelLetterO                  _GameInputLabel = 39
	_GameInputLabelLetterP                  _GameInputLabel = 40
	_GameInputLabelLetterQ                  _GameInputLabel = 41
	_GameInputLabelLetterR                  _GameInputLabel = 42
	_GameInputLabelLetterS                  _GameInputLabel = 43
	_GameInputLabelLetterT                  _GameInputLabel = 44
	_GameInputLabelLetterU                  _GameInputLabel = 45
	_GameInputLabelLetterV                  _GameInputLabel = 46
	_GameInputLabelLetterW                  _GameInputLabel = 47
	_GameInputLabelLetterX                  _GameInputLabel = 48
	_GameInputLabelLetterY                  _GameInputLabel = 49
	_GameInputLabelLetterZ                  _GameInputLabel = 50
	_GameInputLabelNumber0                  _GameInputLabel = 51
	_GameInputLabelNumber1                  _GameInputLabel = 52
	_GameInputLabelNumber2                  _GameInputLabel = 53
	_GameInputLabelNumber3                  _GameInputLabel = 54
	_GameInputLabelNumber4                  _GameInputLabel = 55
	_GameInputLabelNumber5                  _GameInputLabel = 56
	_GameInputLabelNumber6                  _GameInputLabel = 57
	_GameInputLabelNumber7                  _GameInputLabel = 58
	_GameInputLabelNumber8                  _GameInputLabel = 59
	_GameInputLabelNumber9                  _GameInputLabel = 60
	_GameInputLabelArrowUp                  _GameInputLabel = 61
	_GameInputLabelArrowUpRight             _GameInputLabel = 62
	_GameInputLabelArrowRight               _GameInputLabel = 63
	_GameInputLabelArrowDownRight           _GameInputLabel = 64
	_GameInputLabelArrowDown                _GameInputLabel = 65
	_GameInputLabelArrowDownLLeft           _GameInputLabel = 66
	_GameInputLabelArrowLeft                _GameInputLabel = 67
	_GameInputLabelArrowUpLeft              _GameInputLabel = 68
	_GameInputLabelArrowUpDown              _GameInputLabel = 69
	_GameInputLabelArrowLeftRight           _GameInputLabel = 70
	_GameInputLabelArrowUpDownLeftRight     _GameInputLabel = 71
	_GameInputLabelArrowClockwise           _GameInputLabel = 72
	_GameInputLabelArrowCounterClockwise    _GameInputLabel = 73
	_GameInputLabelArrowReturn              _GameInputLabel = 74
	_GameInputLabelIconBranding             _GameInputLabel = 75
	_GameInputLabelIconHome                 _GameInputLabel = 76
	_GameInputLabelIconMenu                 _GameInputLabel = 77
	_GameInputLabelIconCross                _GameInputLabel = 78
	_GameInputLabelIconCircle               _GameInputLabel = 79
	_GameInputLabelIconSquare               _GameInputLabel = 80
	_GameInputLabelIconTriangle             _GameInputLabel = 81
	_GameInputLabelIconStar                 _GameInputLabel = 82
	_GameInputLabelIconDPadUp               _GameInputLabel = 83
	_GameInputLabelIconDPadDown             _GameInputLabel = 84
	_GameInputLabelIconDPadLeft             _GameInputLabel = 85
	_GameInputLabelIconDPadRight            _GameInputLabel = 86
	_GameInputLabelIconDialClockwise        _GameInputLabel = 87
	_GameInputLabelIconDialCounterClockwise _GameInputLabel = 88
	_GameInputLabelIconSliderLeftRight      _GameInputLabel = 89
	_GameInputLabelIconSliderUpDown         _GameInputLabel = 90
	_GameInputLabelIconWheelUpDown          _GameInputLabel = 91
	_GameInputLabelIconPlus                 _GameInputLabel = 92
	_GameInputLabelIconMinus                _GameInputLabel = 93
	_GameInputLabelIconSuspension           _GameInputLabel = 94
	_GameInputLabelHome                     _GameInputLabel = 95
	_GameInputLabelGuide                    _GameInputLabel = 96
	_GameInputLabelMode                     _GameInputLabel = 97
	_GameInputLabelSelect                   _GameInputLabel = 98
	_GameInputLabelMenu                     _GameInputLabel = 99
	_GameInputLabelView                     _GameInputLabel = 100
	_GameInputLabelBack                     _GameInputLabel = 101
	_GameInputLabelStart                    _GameInputLabel = 102
	_GameInputLabelOptions                  _GameInputLabel = 103
	_GameInputLabelShare                    _GameInputLabel = 104
	_GameInputLabelUp                       _GameInputLabel = 105
	_GameInputLabelDown                     _GameInputLabel = 106
	_GameInputLabelLeft                     _GameInputLabel = 107
	_GameInputLabelRight                    _GameInputLabel = 108
	_GameInputLabelLB                       _GameInputLabel = 109
	_GameInputLabelLT                       _GameInputLabel = 110
	_GameInputLabelLSB                      _GameInputLabel = 111
	_GameInputLabelL1                       _GameInputLabel = 112
	_GameInputLabelL2                       _GameInputLabel = 113
	_GameInputLabelL3                       _GameInputLabel = 114
	_GameInputLabelRB                       _GameInputLabel = 115
	_GameInputLabelRT                       _GameInputLabel = 116
	_GameInputLabelRSB                      _GameInputLabel = 117
	_GameInputLabelR1                       _GameInputLabel = 118
	_GameInputLabelR2                       _GameInputLabel = 119
	_GameInputLabelR3                       _GameInputLabel = 120
	_GameInputLabelP1                       _GameInputLabel = 121
	_GameInputLabelP2                       _GameInputLabel = 122
	_GameInputLabelP3                       _GameInputLabel = 123
	_GameInputLabelP4                       _GameInputLabel = 124
)

type _GameInputLocation int32

const (
	_GameInputLocationUnknown  _GameInputLocation = -1
	_GameInputLocationChassis  _GameInputLocation = 0
	_GameInputLocationDisplay  _GameInputLocation = 1
	_GameInputLocationAxis     _GameInputLocation = 2
	_GameInputLocationButton   _GameInputLocation = 3
	_GameInputLocationSwitch   _GameInputLocation = 4
	_GameInputLocationKey      _GameInputLocation = 5
	_GameInputLocationTouchPad _GameInputLocation = 6
)

type _GameInputRumbleMotors int32

const (
	_GameInputRumbleNone          _GameInputRumbleMotors = 0x00000000
	_GameInputRumbleLowFrequency  _GameInputRumbleMotors = 0x00000001
	_GameInputRumbleHighFrequency _GameInputRumbleMotors = 0x00000002
	_GameInputRumbleLeftTrigger   _GameInputRumbleMotors = 0x00000004
	_GameInputRumbleRightTrigger  _GameInputRumbleMotors = 0x00000008
)

type _GameInputSwitchKind int32

const (
	_GameInputUnknownSwitchKind _GameInputSwitchKind = -1
	_GameInput2WaySwitch        _GameInputSwitchKind = 0
	_GameInput4WaySwitch        _GameInputSwitchKind = 1
	_GameInput8WaySwitch        _GameInputSwitchKind = 2
)

type _APP_LOCAL_DEVICE_ID struct {
	value [_APP_LOCAL_DEVICE_ID_SIZE]byte
}

type _GameInputArcadeStickInfo struct {
	// TODO: Implement this
}

type _GameInputControllerAxisInfo struct {
	mappedInputKinds  _GameInputKind
	label             _GameInputLabel
	isContinuous      bool // Assume that the byte size of C++'s bool is 1.
	isNonlinear       bool
	isQuantized       bool
	hasRestValue      bool
	restValue         float32
	resolution        uint64
	legacyDInputIndex uint16
	legacyHidIndex    uint16
	rawReportIndex    uint32
	inputReport       *_GameInputRawDeviceReportInfo
	inputReportItem   *_GameInputRawDeviceReportItemInfo
}

type _GameInputControllerButtonInfo struct {
	mappedInputKinds  _GameInputKind
	label             _GameInputLabel
	legacyDInputIndex uint16
	legacyHidIndex    uint16
	rawReportIndex    uint32
	inputReport       *_GameInputRawDeviceReportInfo
	inputReportItem   *_GameInputRawDeviceReportItemInfo
}

type _GameInputControllerSwitchInfo struct {
	mappedInputKinds  _GameInputKind
	label             _GameInputLabel
	positionLabels    [9]_GameInputLabel
	kind              _GameInputSwitchKind
	legacyDInputIndex uint16
	legacyHidIndex    uint16
	rawReportIndex    uint32
	inputReport       *_GameInputRawDeviceReportInfo
	inputReportItem   *_GameInputRawDeviceReportItemInfo
}

type _GameInputDeviceInfo struct {
	infoSize                 uint32
	vendorId                 uint16
	productId                uint16
	revisionNumber           uint16
	interfaceNumber          uint8
	collectionNumber         uint8
	usage                    _GameInputUsage
	hardwareVersion          _GameInputVersion
	firmwareVersion          _GameInputVersion
	deviceId                 _APP_LOCAL_DEVICE_ID
	deviceRootId             _APP_LOCAL_DEVICE_ID
	deviceFamily             _GameInputDeviceFamily
	capabilities             _GameInputDeviceCapabilities
	supportedInput           _GameInputKind
	supportedRumbleMotors    _GameInputRumbleMotors
	inputReportCount         uint32
	outputReportCount        uint32
	featureReportCount       uint32
	controllerAxisCount      uint32
	controllerButtonCount    uint32
	controllerSwitchCount    uint32
	touchPointCount          uint32
	touchSensorCount         uint32
	forceFeedbackMotorCount  uint32
	hapticFeedbackMotorCount uint32
	deviceStringCount        uint32
	deviceDescriptorSize     uint32
	inputReportInfo          *_GameInputRawDeviceReportInfo
	outputReportInfo         *_GameInputRawDeviceReportInfo
	featureReportInfo        *_GameInputRawDeviceReportInfo
	controllerAxisInfo       *_GameInputControllerAxisInfo
	controllerButtonInfo     *_GameInputControllerButtonInfo
	controllerSwitchInfo     *_GameInputControllerSwitchInfo
	keyboardInfo             *_GameInputKeyboardInfo
	mouseInfo                *_GameInputMouseInfo
	touchSensorInfo          *_GameInputTouchSensorInfo
	motionInfo               *_GameInputMotionInfo
	arcadeStickInfo          *_GameInputArcadeStickInfo
	flightStickInfo          *_GameInputFlightStickInfo
	gamepadInfo              *_GameInputGamepadInfo
	racingWheelInfo          *_GameInputRacingWheelInfo
	uiNavigationInfo         *_GameInputUiNavigationInfo
	forceFeedbackMotorInfo   *_GameInputForceFeedbackMotorInfo
	hapticFeedbackMotorInfo  *_GameInputHapticFeedbackMotorInfo
	displayName              *_GameInputString
	deviceStrings            *_GameInputString
	deviceDescriptorData     unsafe.Pointer
}

type _GameInputFlightStickInfo struct {
	// TODO: Implement this
}

type _GameInputForceFeedbackMotorInfo struct {
	supportedAxes                     _GameInputFeedbackAxes
	location                          _GameInputLocation
	locationId                        uint32
	maxSimultaneousEffects            uint32
	isConstantEffectSupported         bool
	isRampEffectSupported             bool
	isSineWaveEffectSupported         bool
	isSquareWaveEffectSupported       bool
	isTriangleWaveEffectSupported     bool
	isSawtoothUpWaveEffectSupported   bool
	isSawtoothDownWaveEffectSupported bool
	isSpringEffectSupported           bool
	isFrictionEffectSupported         bool
	isDamperEffectSupported           bool
	isInertiaEffectSupported          bool
}

type _GameInputHapticFeedbackMotorInfo struct {
	mappedRumbleMotors _GameInputRumbleMotors
	location           _GameInputLocation
	locationId         uint32
	waveformCount      uint32
	waveformInfo       *_GameInputHapticWaveformInfo
}

type _GameInputHapticWaveformInfo struct {
	usage                  _GameInputUsage
	isDurationSupported    bool
	isIntensitySupported   bool
	isRepeatSupported      bool
	isRepeatDelaySupported bool
	defaultDuration        uint64
}

type _GameInputGamepadInfo struct {
	menuButtonLabel            _GameInputLabel
	viewButtonLabel            _GameInputLabel
	aButtonLabel               _GameInputLabel
	bButtonLabel               _GameInputLabel
	xButtonLabel               _GameInputLabel
	yButtonLabel               _GameInputLabel
	dpadUpLabel                _GameInputLabel
	dpadDownLabel              _GameInputLabel
	dpadLeftLabel              _GameInputLabel
	dpadRightLabel             _GameInputLabel
	leftShoulderButtonLabel    _GameInputLabel
	rightShoulderButtonLabel   _GameInputLabel
	leftThumbstickButtonLabel  _GameInputLabel
	rightThumbstickButtonLabel _GameInputLabel
}

type _GameInputGamepadState struct {
	buttons          _GameInputGamepadButtons
	leftTrigger      float32
	rightTrigger     float32
	leftThumbstickX  float32
	leftThumbstickY  float32
	rightThumbstickX float32
	rightThumbstickY float32
}

type _GameInputKeyboardInfo struct {
	// TODO: Implement this
}

type _GameInputMotionInfo struct {
	// TODO: Implement this
}

type _GameInputMouseInfo struct {
	// TODO: Implement this
}

type _GameInputRacingWheelInfo struct {
	// TODO: Implement this
}

type _GameInputRawDeviceItemCollectionInfo struct {
	// TODO: Implement this
}

type _GameInputRawDeviceReportInfo struct {
	// TODO: Implement this
}

type _GameInputRawDeviceReportItemInfo struct {
	// TODO: Implement this
}

type _GameInputRumbleParams struct {
	lowFrequency  float32
	highFrequency float32
	leftTrigger   float32
	rightTrigger  float32
}

type _GameInputString struct {
	sizeInBytes    uint32
	codePointCount uint32
	data           *byte
}

type _GameInputTouchSensorInfo struct {
	// TODO: Implement this
}

type _GameInputUiNavigationInfo struct {
	// TODO: Implement this
}

type _GameInputUsage struct {
	page uint16
	id   uint16
}

type _GameInputVersion struct {
	major    uint16
	minor    uint16
	build    uint16
	revision uint16
}

func _GameInputCreate() (*_IGameInput, error) {
	var gameInput *_IGameInput
	r, _, _ := procGameInputCreate.Call(uintptr(unsafe.Pointer(&gameInput)))
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("gamepad: GameInputCreate failed: HRESULT(%d)", uint32(r))
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
		return nil, fmt.Errorf("gamepad: IGameInput::GetCurrentReading failed: HRESULT(%d)", uint32(r))
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
		return fmt.Errorf("gamepad: IGameInput::RegisterDeviceCallback failed: HRESULT(%d)", uint32(r))
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

func (i *_IGameInputDevice) GetDeviceInfo() *_GameInputDeviceInfo {
	r, _, _ := syscall.Syscall(i.vtbl.GetDeviceInfo, 1, uintptr(unsafe.Pointer(i)), 0, 0)
	// The lifetime of the returned value is the same as i.
	return (*_GameInputDeviceInfo)(unsafe.Pointer(r))
}

func (i *_IGameInputDevice) SetRumbleState(params *_GameInputRumbleParams, timestamp uint64) {
	syscall.Syscall(i.vtbl.SetRumbleState, 3, uintptr(unsafe.Pointer(i)), uintptr(unsafe.Pointer(params)), uintptr(timestamp))
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

func (i *_IGameInputReading) Release() {
	syscall.Syscall(i.vtbl.Release, 1, uintptr(unsafe.Pointer(i)), 0, 0)
}
