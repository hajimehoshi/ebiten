// Copyright 2026 The Ebitengine Authors
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

var MotorMagnitude = motorMagnitude

// nativeGamepadForTest is a gamepad backend whose reports a test supplies. Its axes and buttons
// reuse the virtual backend, and it adds the hats that a virtual gamepad never has.
type nativeGamepadForTest struct {
	nativeGamepadVirtual

	hats []int
}

func (g *nativeGamepadForTest) hatCount() int {
	return len(g.hats)
}

func (g *nativeGamepadForTest) hatState(hat int) int {
	if hat < 0 || hat >= len(g.hats) {
		return hatCentered
	}
	return g.hats[hat]
}

// NewGamepadForTest returns a gamepad with the given SDL ID that takes its standard layout from
// gamepaddb, as a device does. The gamepad is not in the gamepad list, and its raw state is written
// with [Gamepad.SetReportForTest].
func NewGamepadForTest(sdlID string) *Gamepad {
	return &Gamepad{
		sdlID:  sdlID,
		native: &nativeGamepadForTest{},
	}
}

// SetReportForTest replaces the gamepad's raw state with one report, as an update from a device does.
func (g *Gamepad) SetReportForTest(axes []float64, buttons []bool, hats []int) {
	withNative(g, func(n *nativeGamepadForTest) {
		n.axes = append(n.axes[:0], axes...)
		n.buttons = append(n.buttons[:0], buttons...)
		n.hats = append(n.hats[:0], hats...)
	})
}
