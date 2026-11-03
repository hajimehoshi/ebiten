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

package glfw_test

import (
	"math"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

func TestDirectManipulationVtblSlots(t *testing.T) {
	// IUnknown has 3 methods.
	want := map[string]uintptr{
		"IDirectManipulationManager":       3 + 7,
		"IDirectManipulationUpdateManager": 3 + 3,
		"IDirectManipulationViewport":      3 + 28,
		"IDirectManipulationContent":       3 + 8,
	}
	for name, got := range glfw.DirectManipulationVtblSlots() {
		if got != want[name] {
			t.Errorf("%s: got %d slots, want %d", name, got, want[name])
		}
	}
}

func transform(scale, x, y float32) [6]float32 {
	return [6]float32{scale, 0, 0, scale, x, y}
}

func TestDirectManipulationGesture(t *testing.T) {
	type step struct {
		m         [6]float32
		panX      float64
		panY      float64
		pinch     float64
		tolerance float64
	}
	for _, tt := range []struct {
		name  string
		steps []step
	}{
		{
			name: "pan",
			steps: []step{
				{m: transform(1, 0, -10), panY: -10},
				{m: transform(1, 5, -30), panX: 5, panY: -20},
				{m: transform(1, 5, -30)},
			},
		},
		{
			name: "pinch",
			steps: []step{
				{m: transform(1.25, -3, -4), pinch: 0.25},
				{m: transform(1.5, -6, -8), pinch: 0.2, tolerance: 1e-6},
				// The translation during a pinch is not a pan.
				{m: transform(1.5, -20, -20)},
				{m: transform(0.75, 0, 0), pinch: -0.5},
			},
		},
		{
			name: "pan turning into pinch",
			steps: []step{
				{m: transform(1, 2, 0), panX: 2},
				{m: transform(1.1, 3, 1), pinch: 0.1, tolerance: 1e-6},
				// A pinch does not turn back into a pan.
				{m: transform(1, 3, 1), pinch: 1/1.1 - 1, tolerance: 1e-6},
			},
		},
		{
			name: "zero scale",
			steps: []step{
				{m: transform(0, 10, 10)},
				{m: transform(1, 10, 10), panX: 10, panY: 10},
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := glfw.NewDirectManipulationGesture()
			for i, s := range tt.steps {
				panX, panY, pinch := g.Transform(s.m)
				if panX != s.panX || panY != s.panY || math.Abs(pinch-s.pinch) > s.tolerance {
					t.Errorf("step %d: got (%v, %v, %v), want (%v, %v, %v)", i, panX, panY, pinch, s.panX, s.panY, s.pinch)
				}
			}
		})
	}
}

func TestDirectManipulationPolling(t *testing.T) {
	for _, tt := range []struct {
		name         string
		contact      bool
		sinceContact time.Duration
		statuses     []uint32
		want         bool
	}{
		{
			name:     "no contact",
			statuses: []uint32{glfw.DirectManipulationBuilding},
			want:     false,
		},
		{
			name:     "ready without a contact",
			statuses: []uint32{glfw.DirectManipulationReady},
			want:     false,
		},
		{
			name:         "ready after a contact",
			contact:      true,
			sinceContact: 0,
			statuses:     []uint32{glfw.DirectManipulationReady},
			want:         true,
		},
		{
			name:         "ready long after a contact",
			contact:      true,
			sinceContact: glfw.DirectManipulationContactPollingDuration,
			statuses:     []uint32{glfw.DirectManipulationReady},
			want:         false,
		},
		{
			name:         "running",
			contact:      true,
			sinceContact: time.Hour,
			statuses:     []uint32{glfw.DirectManipulationReady, glfw.DirectManipulationRunning},
			want:         true,
		},
		{
			name:         "inertia",
			contact:      true,
			sinceContact: time.Hour,
			statuses:     []uint32{glfw.DirectManipulationRunning, glfw.DirectManipulationInertia},
			want:         true,
		},
		{
			name:         "ready after running soon after a contact",
			contact:      true,
			sinceContact: 0,
			statuses:     []uint32{glfw.DirectManipulationRunning, glfw.DirectManipulationReady},
			want:         false,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := glfw.DirectManipulationPolling(tt.contact, tt.sinceContact, tt.statuses...); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDirectManipulationZoomToRect tests that the float arguments of ZoomToRect reach Direct Manipulation,
// which depends on the calling convention of the architecture.
func TestDirectManipulationZoomToRect(t *testing.T) {
	// The viewport is 1000x1000 pixels. Zooming to a square of 500 pixels doubles the content, and moves the
	// corner of the square to the origin.
	m, err := glfw.DirectManipulationZoomToRect(100, 200, 600, 700)
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Skip("Direct Manipulation is not available")
	}
	want := [6]float32{2, 0, 0, 2, -200, -400}
	near := true
	for i := range m {
		if math.Abs(float64(m[i]-want[i])) > 1e-3 {
			near = false
		}
	}
	if !near {
		t.Errorf("got %v, want %v", *m, want)
	}
}
