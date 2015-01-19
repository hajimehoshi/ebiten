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

// Package gamepad offers abstract gamepad buttons and configuration.
package gamepad

import (
	"github.com/hajimehoshi/ebiten"
)

// A StdButton represents a standard gamepad button.
// See also: http://www.w3.org/TR/gamepad/
//    [UL0]            [UR0]
//    [UL1]            [UR1]
//
//    [LU]     [CC]     [RU]
//  [LL][LR] [CL][CR] [RL][RR]
//    [LD]              [RD]
//         [AL]    [AR]
type StdButton int

const (
	StdButtonNone StdButton = iota
	StdButtonLL
	StdButtonLR
	StdButtonLU
	StdButtonLD
	StdButtonCL
	StdButtonCC
	StdButtonCR
	StdButtonRL
	StdButtonRR
	StdButtonRU
	StdButtonRD
	StdButtonUL0
	StdButtonUL1
	StdButtonUR0
	StdButtonUR1
	StdButtonAL
	StdButtonAR
)

const threshold = 0.75

type axis struct {
	id       int
	positive bool
}

type Configuration struct {
	current StdButton
	buttons map[StdButton]ebiten.GamepadButton
	axes    map[StdButton]axis
}

func (c *Configuration) Scan(index int, b StdButton) bool {
	if c.buttons == nil {
		c.buttons = map[StdButton]ebiten.GamepadButton{}
	}
	if c.axes == nil {
		c.axes = map[StdButton]axis{}
	}

	delete(c.buttons, b)
	delete(c.axes, b)

	ebn := ebiten.GamepadButton(ebiten.GamepadButtonNum(index))
	for eb := ebiten.GamepadButton(0); eb < ebn; eb++ {
		if ebiten.IsGamepadButtonPressed(index, eb) {
			c.buttons[b] = eb
			return true
		}
	}
	an := ebiten.GamepadAxisNum(index)
	for a := 0; a < an; a++ {
		v := ebiten.GamepadAxis(index, an)
		// Check v <= 1.0 because there is a bug that a button returns an axis value wrongly and the value may be over 1.
		if threshold <= v && v <= 1.0 {
			c.axes[b] = axis{a, true}
			return true
		}
		if -1.0 <= v && v <= -threshold {
			c.axes[b] = axis{a, false}
			return true
		}
	}
	return false
}

func (c *Configuration) IsButtonPressed(id int, b StdButton) bool {
	if c.buttons == nil {
		c.buttons = map[StdButton]ebiten.GamepadButton{}
	}
	if c.axes == nil {
		c.axes = map[StdButton]axis{}
	}

	bb, ok := c.buttons[b]
	if ok {
		return ebiten.IsGamepadButtonPressed(0, bb)
	}
	a, ok := c.axes[b]
	if ok {
		v := ebiten.GamepadAxis(0, a.id)
		if a.positive {
			return threshold <= v && v <= 1.0
		} else {
			return -1.0 <= v && v <= -threshold
		}
	}
	return false
}
