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
)

// GamepadButton represents a gamepad button.
type GamepadButton = driver.GamepadButton

// GamepadButtons
const (
	GamepadButton0   GamepadButton = driver.GamepadButton0
	GamepadButton1   GamepadButton = driver.GamepadButton1
	GamepadButton2   GamepadButton = driver.GamepadButton2
	GamepadButton3   GamepadButton = driver.GamepadButton3
	GamepadButton4   GamepadButton = driver.GamepadButton4
	GamepadButton5   GamepadButton = driver.GamepadButton5
	GamepadButton6   GamepadButton = driver.GamepadButton6
	GamepadButton7   GamepadButton = driver.GamepadButton7
	GamepadButton8   GamepadButton = driver.GamepadButton8
	GamepadButton9   GamepadButton = driver.GamepadButton9
	GamepadButton10  GamepadButton = driver.GamepadButton10
	GamepadButton11  GamepadButton = driver.GamepadButton11
	GamepadButton12  GamepadButton = driver.GamepadButton12
	GamepadButton13  GamepadButton = driver.GamepadButton13
	GamepadButton14  GamepadButton = driver.GamepadButton14
	GamepadButton15  GamepadButton = driver.GamepadButton15
	GamepadButton16  GamepadButton = driver.GamepadButton16
	GamepadButton17  GamepadButton = driver.GamepadButton17
	GamepadButton18  GamepadButton = driver.GamepadButton18
	GamepadButton19  GamepadButton = driver.GamepadButton19
	GamepadButton20  GamepadButton = driver.GamepadButton20
	GamepadButton21  GamepadButton = driver.GamepadButton21
	GamepadButton22  GamepadButton = driver.GamepadButton22
	GamepadButton23  GamepadButton = driver.GamepadButton23
	GamepadButton24  GamepadButton = driver.GamepadButton24
	GamepadButton25  GamepadButton = driver.GamepadButton25
	GamepadButton26  GamepadButton = driver.GamepadButton26
	GamepadButton27  GamepadButton = driver.GamepadButton27
	GamepadButton28  GamepadButton = driver.GamepadButton28
	GamepadButton29  GamepadButton = driver.GamepadButton29
	GamepadButton30  GamepadButton = driver.GamepadButton30
	GamepadButton31  GamepadButton = driver.GamepadButton31
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
