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

package gamepad_test

import (
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

func TestXboxDeviceEvents(t *testing.T) {
	// GameInput reports the same device pointer for the same physical device, so the tests reuse
	// deviceA and deviceB across connections and disconnections.
	deviceA := &gamepad.IGameInputDevice{}
	deviceB := &gamepad.IGameInputDevice{}

	type event struct {
		device    *gamepad.IGameInputDevice
		connected bool
	}

	type frame struct {
		events []event
		want   []*gamepad.IGameInputDevice
	}

	tests := []struct {
		name   string
		frames []frame
	}{
		{
			name: "connect",
			frames: []frame{
				{
					events: []event{{deviceA, true}},
					want:   []*gamepad.IGameInputDevice{deviceA},
				},
			},
		},
		{
			name: "disconnect",
			frames: []frame{
				{
					events: []event{{deviceA, true}},
					want:   []*gamepad.IGameInputDevice{deviceA},
				},
				{
					events: []event{{deviceA, false}},
					want:   nil,
				},
			},
		},
		{
			name: "connect and disconnect in one frame",
			frames: []frame{
				{
					events: []event{{deviceA, true}, {deviceA, false}},
					want:   nil,
				},
			},
		},
		{
			name: "disconnect and reconnect in one frame",
			frames: []frame{
				{
					events: []event{{deviceA, true}},
					want:   []*gamepad.IGameInputDevice{deviceA},
				},
				{
					events: []event{{deviceA, false}, {deviceA, true}},
					want:   []*gamepad.IGameInputDevice{deviceA},
				},
			},
		},
		{
			name: "one of two devices disconnects in one frame",
			frames: []frame{
				{
					events: []event{{deviceA, true}, {deviceB, true}, {deviceA, false}},
					want:   []*gamepad.IGameInputDevice{deviceB},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var n gamepad.XboxGamepads
			var g gamepad.Gamepads
			for i, f := range tc.frames {
				for _, e := range f.events {
					n.DeviceCallback(e.device, e.connected)
				}
				if err := n.Update(&g); err != nil {
					t.Fatalf("frame %d: Update failed: %v", i, err)
				}
				got := g.AppendXboxDevices(nil)
				if !slices.Equal(got, f.want) {
					t.Errorf("frame %d: devices: got %v, want %v", i, got, f.want)
				}
			}
		})
	}
}
