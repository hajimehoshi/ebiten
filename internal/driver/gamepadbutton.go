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

package driver

type GamepadButton int

const (
	GamepadButton0 GamepadButton = iota
	GamepadButton1
	GamepadButton2
	GamepadButton3
	GamepadButton4
	GamepadButton5
	GamepadButton6
	GamepadButton7
	GamepadButton8
	GamepadButton9
	GamepadButton10
	GamepadButton11
	GamepadButton12
	GamepadButton13
	GamepadButton14
	GamepadButton15
	GamepadButton16
	GamepadButton17
	GamepadButton18
	GamepadButton19
	GamepadButton20
	GamepadButton21
	GamepadButton22
	GamepadButton23
	GamepadButton24
	GamepadButton25
	GamepadButton26
	GamepadButton27
	GamepadButton28
	GamepadButton29
	GamepadButton30
	GamepadButton31
)

const GamepadButtonNum = 32

type StandardGamepadButton int

// https://www.w3.org/TR/gamepad/#remapping
const (
	StandardGamepadButtonRightBottom StandardGamepadButton = iota
	StandardGamepadButtonRightRight
	StandardGamepadButtonRightLeft
	StandardGamepadButtonRightTop
	StandardGamepadButtonFrontTopLeft
	StandardGamepadButtonFrontTopRight
	StandardGamepadButtonFrontBottomLeft
	StandardGamepadButtonFrontBottomRight
	StandardGamepadButtonCenterLeft
	StandardGamepadButtonCenterRight
	StandardGamepadButtonLeftStick
	StandardGamepadButtonRightStick
	StandardGamepadButtonLeftTop
	StandardGamepadButtonLeftBottom
	StandardGamepadButtonLeftLeft
	StandardGamepadButtonLeftRight
	StandardGamepadButtonCenterCenter

	StandardGamepadButtonMax = StandardGamepadButtonCenterCenter
)

type StandardGamepadAxis int

// https://www.w3.org/TR/gamepad/#remapping
const (
	StandardGamepadAxisLeftStickHorizontal StandardGamepadAxis = iota
	StandardGamepadAxisLeftStickVertical
	StandardGamepadAxisRightStickHorizontal
	StandardGamepadAxisRightStickVertical

	StandardGamepadAxisMax = StandardGamepadAxisRightStickVertical
)
