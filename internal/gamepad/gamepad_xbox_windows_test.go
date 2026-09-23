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
	// GameInput reports the same device pointer for the same physical device, so an event names its
	// device by index and the tests reuse deviceA and deviceB across connections and disconnections.
	const (
		deviceA = iota
		deviceB
	)

	type event struct {
		device    int
		connected bool
	}

	type frame struct {
		events []event
		want   []int
	}

	tests := []struct {
		name   string
		frames []frame
	}{
		{
			name: "connect",
			frames: []frame{
				{
					events: []event{
						{
							device:    deviceA,
							connected: true,
						},
					},
					want: []int{deviceA},
				},
			},
		},
		{
			name: "disconnect",
			frames: []frame{
				{
					events: []event{
						{
							device:    deviceA,
							connected: true,
						},
					},
					want: []int{deviceA},
				},
				{
					events: []event{
						{
							device:    deviceA,
							connected: false,
						},
					},
					want: nil,
				},
			},
		},
		{
			name: "connect and disconnect in one frame",
			frames: []frame{
				{
					events: []event{
						{
							device:    deviceA,
							connected: true,
						},
						{
							device:    deviceA,
							connected: false,
						},
					},
					want: nil,
				},
			},
		},
		{
			name: "disconnect and reconnect in one frame",
			frames: []frame{
				{
					events: []event{
						{
							device:    deviceA,
							connected: true,
						},
					},
					want: []int{deviceA},
				},
				{
					events: []event{
						{
							device:    deviceA,
							connected: false,
						},
						{
							device:    deviceA,
							connected: true,
						},
					},
					want: []int{deviceA},
				},
			},
		},
		{
			name: "one of two devices disconnects in one frame",
			frames: []frame{
				{
					events: []event{
						{
							device:    deviceA,
							connected: true,
						},
						{
							device:    deviceB,
							connected: true,
						},
						{
							device:    deviceA,
							connected: false,
						},
					},
					want: []int{deviceB},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			devices := []*gamepad.IGameInputDevice{
				deviceA: gamepad.NewXboxDevice(),
				deviceB: gamepad.NewXboxDevice(),
			}
			var n gamepad.XboxGamepads
			var g gamepad.Gamepads
			for i, f := range tc.frames {
				for _, e := range f.events {
					n.DeviceCallback(devices[e.device], e.connected)
				}
				if err := n.Update(&g); err != nil {
					t.Fatalf("frame %d: Update failed: %v", i, err)
				}
				var want []*gamepad.IGameInputDevice
				for _, d := range f.want {
					want = append(want, devices[d])
				}
				got := g.AppendXboxDevices(nil)
				if !slices.Equal(got, want) {
					t.Errorf("frame %d: devices: got %v, want %v", i, got, want)
				}
				// A registered gamepad holds exactly one reference to its device; nothing else does.
				for d, device := range devices {
					want := 0
					if slices.Contains(f.want, d) {
						want = 1
					}
					if got := gamepad.XboxDeviceRefCount(device); got != want {
						t.Errorf("frame %d: device %d: reference count: got %d, want %d", i, d, got, want)
					}
				}
			}
		})
	}
}
