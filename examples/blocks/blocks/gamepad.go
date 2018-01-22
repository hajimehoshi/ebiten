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

// +build example

package blocks

import (
	"fmt"

	"github.com/hajimehoshi/ebiten"
)

type virtualGamepadButton int

const (
	virtualGamepadButtonLeft virtualGamepadButton = iota
	virtualGamepadButtonRight
	virtualGamepadButtonDown
	virtualGamepadButtonButtonA
	virtualGamepadButtonButtonB
)

var virtualGamepadButtons = []virtualGamepadButton{
	virtualGamepadButtonLeft,
	virtualGamepadButtonRight,
	virtualGamepadButtonDown,
	virtualGamepadButtonButtonA,
	virtualGamepadButtonButtonB,
}

const threshold = 0.75

type axis struct {
	id       int
	positive bool
}

type gamepadConfig struct {
	current         virtualGamepadButton
	buttons         map[virtualGamepadButton]ebiten.GamepadButton
	axes            map[virtualGamepadButton]axis
	assignedButtons map[ebiten.GamepadButton]struct{}
	assignedAxes    map[axis]struct{}
}

func (c *gamepadConfig) initializeIfNeeded() {
	if c.buttons == nil {
		c.buttons = map[virtualGamepadButton]ebiten.GamepadButton{}
	}
	if c.axes == nil {
		c.axes = map[virtualGamepadButton]axis{}
	}
	if c.assignedButtons == nil {
		c.assignedButtons = map[ebiten.GamepadButton]struct{}{}
	}
	if c.assignedAxes == nil {
		c.assignedAxes = map[axis]struct{}{}
	}
}

func (c *gamepadConfig) Reset() {
	c.buttons = nil
	c.axes = nil
	c.assignedButtons = nil
	c.assignedAxes = nil
}

// Scan scans the current input state and assigns the given virtual gamepad button b
// to the current (pysical) pressed buttons of the gamepad.
func (c *gamepadConfig) Scan(gamepadID int, b virtualGamepadButton) bool {
	c.initializeIfNeeded()

	delete(c.buttons, b)
	delete(c.axes, b)

	ebn := ebiten.GamepadButton(ebiten.GamepadButtonNum(gamepadID))
	for eb := ebiten.GamepadButton(0); eb < ebn; eb++ {
		if _, ok := c.assignedButtons[eb]; ok {
			continue
		}
		if ebiten.IsGamepadButtonPressed(gamepadID, eb) {
			c.buttons[b] = eb
			c.assignedButtons[eb] = struct{}{}
			return true
		}
	}

	an := ebiten.GamepadAxisNum(gamepadID)
	for a := 0; a < an; a++ {
		v := ebiten.GamepadAxis(gamepadID, a)
		// Check |v| < 1.0 because
		// 1) there is a bug that a button returns an axis value wrongly
		// and the value may be over 1.
		// 2) just 1.0 or -1.0 values are ignored since PS4's L2/R2 keys take
		// -1.0 by default.
		if threshold <= v && v < 1.0 {
			if _, ok := c.assignedAxes[axis{a, true}]; !ok {
				c.axes[b] = axis{a, true}
				c.assignedAxes[axis{a, true}] = struct{}{}
				return true
			}
		}
		if -1.0 < v && v <= -threshold {
			if _, ok := c.assignedAxes[axis{a, false}]; !ok {
				c.axes[b] = axis{a, false}
				c.assignedAxes[axis{a, false}] = struct{}{}
				return true
			}
		}
	}

	return false
}

// IsButtonPressed returns a boolean value indicating whether
// the given virtual button b is pressed.
func (c *gamepadConfig) IsButtonPressed(b virtualGamepadButton) bool {
	c.initializeIfNeeded()

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

// Name returns the pysical button's name for the given virtual button.
func (c *gamepadConfig) ButtonName(b virtualGamepadButton) string {
	c.initializeIfNeeded()

	bb, ok := c.buttons[b]
	if ok {
		return fmt.Sprintf("Button %d", bb)
	}

	a, ok := c.axes[b]
	if ok {
		if a.positive {
			return fmt.Sprintf("Axis %d+", a.id)
		} else {
			return fmt.Sprintf("Axis %d-", a.id)
		}
	}

	return ""
}
