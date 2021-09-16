// Copyright 2018 The Ebiten Authors
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

//go:build !js
// +build !js

package glfw

import (
	"fmt"
)

type (
	Action           int
	ErrorCode        int
	Hint             int
	InputMode        int
	Joystick         int
	JoystickHatState int
	Key              int
	ModifierKey      int
	MouseButton      int
	PeripheralEvent  int
	StandardCursor   int
)

const (
	DontCare = -1
	False    = 0
	True     = 1
)

const (
	Release = Action(0)
	Press   = Action(1)
	Repeat  = Action(2)
)

const (
	ModShift   = ModifierKey(0x0001)
	ModControl = ModifierKey(0x0002)
	ModAlt     = ModifierKey(0x0004)
)

const (
	MouseButtonLeft   = MouseButton(0)
	MouseButtonRight  = MouseButton(1)
	MouseButtonMiddle = MouseButton(2)
)

const (
	Joystick1  = Joystick(0)
	Joystick2  = Joystick(1)
	Joystick3  = Joystick(2)
	Joystick4  = Joystick(3)
	Joystick5  = Joystick(4)
	Joystick6  = Joystick(5)
	Joystick7  = Joystick(6)
	Joystick8  = Joystick(7)
	Joystick9  = Joystick(8)
	Joystick10 = Joystick(9)
	Joystick11 = Joystick(10)
	Joystick12 = Joystick(11)
	Joystick13 = Joystick(12)
	Joystick14 = Joystick(13)
	Joystick15 = Joystick(14)
	Joystick16 = Joystick(15)
)

const (
	AutoIconify            = Hint(0x00020006)
	ClientAPI              = Hint(0x00022001)
	ContextVersionMajor    = Hint(0x00022002)
	ContextVersionMinor    = Hint(0x00022003)
	Decorated              = Hint(0x00020005)
	Floating               = Hint(0x00020007)
	Focused                = Hint(0x00020001)
	FocusOnShow            = Hint(0x0002000C)
	Iconified              = Hint(0x00020002)
	Maximized              = Hint(0x00020008)
	Resizable              = Hint(0x00020003)
	TransparentFramebuffer = Hint(0x0002000A)
	Visible                = Hint(0x00020004)
)

const (
	CursorMode             = InputMode(0x00033001)
	StickyKeysMode         = InputMode(0x00033002)
	StickyMouseButtonsMode = InputMode(0x00033003)
)

const (
	CursorDisabled = 0x00034003
	CursorHidden   = 0x00034002
	CursorNormal   = 0x00034001
	NoAPI          = 0
	OpenGLAPI      = 0x00030001
)

const (
	NotInitialized     = ErrorCode(0x00010001)
	NoCurrentContext   = ErrorCode(0x00010002)
	InvalidEnum        = ErrorCode(0x00010003)
	InvalidValue       = ErrorCode(0x00010004)
	OutOfMemory        = ErrorCode(0x00010005)
	APIUnavailable     = ErrorCode(0x00010006)
	VersionUnavailable = ErrorCode(0x00010007)
	PlatformError      = ErrorCode(0x00010008)
	FormatUnavailable  = ErrorCode(0x00010009)
	NoWindowContext    = ErrorCode(0x0001000A)
)

func (e ErrorCode) String() string {
	switch e {
	case NotInitialized:
		return "not initialized"
	case NoCurrentContext:
		return "no current context"
	case InvalidEnum:
		return "invalid enum"
	case InvalidValue:
		return "invalid value"
	case OutOfMemory:
		return "out of memory"
	case APIUnavailable:
		return "API unavailable"
	case VersionUnavailable:
		return "version unavailable"
	case PlatformError:
		return "platform error"
	case FormatUnavailable:
		return "format unavailable"
	case NoWindowContext:
		return "no window context"
	default:
		return fmt.Sprintf("GLFW error code (%d)", e)
	}
}

const (
	ArrowCursor     = StandardCursor(0x00036001)
	IBeamCursor     = StandardCursor(0x00036002)
	CrosshairCursor = StandardCursor(0x00036003)
	HandCursor      = StandardCursor(0x00036004)
	HResizeCursor   = StandardCursor(0x00036005)
	VResizeCursor   = StandardCursor(0x00036006)
)

const (
	HatCentered  = JoystickHatState(0)
	HatUp        = JoystickHatState(1)
	HatRight     = JoystickHatState(2)
	HatDown      = JoystickHatState(4)
	HatLeft      = JoystickHatState(8)
	HatRightUp   = HatRight | HatUp
	HatRightDown = HatRight | HatDown
	HatLeftUp    = HatLeft | HatUp
	HatLeftDown  = HatLeft | HatDown
)
