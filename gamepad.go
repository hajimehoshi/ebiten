// Copyright 2015 Hajime Hoshi
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

package ebiten

import (
	"github.com/hajimehoshi/ebiten/v2/internal/driver"
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

// GamepadButton represents a gamepad button.
type GamepadButton = gamepad.Button

// GamepadButtons
const (
	GamepadButton0   GamepadButton = gamepad.Button0
	GamepadButton1   GamepadButton = gamepad.Button1
	GamepadButton2   GamepadButton = gamepad.Button2
	GamepadButton3   GamepadButton = gamepad.Button3
	GamepadButton4   GamepadButton = gamepad.Button4
	GamepadButton5   GamepadButton = gamepad.Button5
	GamepadButton6   GamepadButton = gamepad.Button6
	GamepadButton7   GamepadButton = gamepad.Button7
	GamepadButton8   GamepadButton = gamepad.Button8
	GamepadButton9   GamepadButton = gamepad.Button9
	GamepadButton10  GamepadButton = gamepad.Button10
	GamepadButton11  GamepadButton = gamepad.Button11
	GamepadButton12  GamepadButton = gamepad.Button12
	GamepadButton13  GamepadButton = gamepad.Button13
	GamepadButton14  GamepadButton = gamepad.Button14
	GamepadButton15  GamepadButton = gamepad.Button15
	GamepadButton16  GamepadButton = gamepad.Button16
	GamepadButton17  GamepadButton = gamepad.Button17
	GamepadButton18  GamepadButton = gamepad.Button18
	GamepadButton19  GamepadButton = gamepad.Button19
	GamepadButton20  GamepadButton = gamepad.Button20
	GamepadButton21  GamepadButton = gamepad.Button21
	GamepadButton22  GamepadButton = gamepad.Button22
	GamepadButton23  GamepadButton = gamepad.Button23
	GamepadButton24  GamepadButton = gamepad.Button24
	GamepadButton25  GamepadButton = gamepad.Button25
	GamepadButton26  GamepadButton = gamepad.Button26
	GamepadButton27  GamepadButton = gamepad.Button27
	GamepadButton28  GamepadButton = gamepad.Button28
	GamepadButton29  GamepadButton = gamepad.Button29
	GamepadButton30  GamepadButton = gamepad.Button30
	GamepadButton31  GamepadButton = gamepad.Button31
	GamepadButtonMax GamepadButton = GamepadButton31
)

// StandardGamepadButton represents a gamepad button in the standard layout.
//
// The layout and the button values are based on the web standard.
// See https://www.w3.org/TR/gamepad/#remapping.
type StandardGamepadButton = driver.StandardGamepadButton

// StandardGamepadButtons
const (
	StandardGamepadButtonRightBottom      StandardGamepadButton = driver.StandardGamepadButtonRightBottom
	StandardGamepadButtonRightRight       StandardGamepadButton = driver.StandardGamepadButtonRightRight
	StandardGamepadButtonRightLeft        StandardGamepadButton = driver.StandardGamepadButtonRightLeft
	StandardGamepadButtonRightTop         StandardGamepadButton = driver.StandardGamepadButtonRightTop
	StandardGamepadButtonFrontTopLeft     StandardGamepadButton = driver.StandardGamepadButtonFrontTopLeft
	StandardGamepadButtonFrontTopRight    StandardGamepadButton = driver.StandardGamepadButtonFrontTopRight
	StandardGamepadButtonFrontBottomLeft  StandardGamepadButton = driver.StandardGamepadButtonFrontBottomLeft
	StandardGamepadButtonFrontBottomRight StandardGamepadButton = driver.StandardGamepadButtonFrontBottomRight
	StandardGamepadButtonCenterLeft       StandardGamepadButton = driver.StandardGamepadButtonCenterLeft
	StandardGamepadButtonCenterRight      StandardGamepadButton = driver.StandardGamepadButtonCenterRight
	StandardGamepadButtonLeftStick        StandardGamepadButton = driver.StandardGamepadButtonLeftStick
	StandardGamepadButtonRightStick       StandardGamepadButton = driver.StandardGamepadButtonRightStick
	StandardGamepadButtonLeftTop          StandardGamepadButton = driver.StandardGamepadButtonLeftTop
	StandardGamepadButtonLeftBottom       StandardGamepadButton = driver.StandardGamepadButtonLeftBottom
	StandardGamepadButtonLeftLeft         StandardGamepadButton = driver.StandardGamepadButtonLeftLeft
	StandardGamepadButtonLeftRight        StandardGamepadButton = driver.StandardGamepadButtonLeftRight
	StandardGamepadButtonCenterCenter     StandardGamepadButton = driver.StandardGamepadButtonCenterCenter
	StandardGamepadButtonMax              StandardGamepadButton = StandardGamepadButtonCenterCenter
)

// StandardGamepadAxis represents a gamepad axis in the standard layout.
//
// The layout and the button values are based on the web standard.
// See https://www.w3.org/TR/gamepad/#remapping.
type StandardGamepadAxis = driver.StandardGamepadAxis

// StandardGamepadAxes
const (
	StandardGamepadAxisLeftStickHorizontal  StandardGamepadAxis = driver.StandardGamepadAxisLeftStickHorizontal
	StandardGamepadAxisLeftStickVertical    StandardGamepadAxis = driver.StandardGamepadAxisLeftStickVertical
	StandardGamepadAxisRightStickHorizontal StandardGamepadAxis = driver.StandardGamepadAxisRightStickHorizontal
	StandardGamepadAxisRightStickVertical   StandardGamepadAxis = driver.StandardGamepadAxisRightStickVertical
	StandardGamepadAxisMax                  StandardGamepadAxis = StandardGamepadAxisRightStickVertical
)
