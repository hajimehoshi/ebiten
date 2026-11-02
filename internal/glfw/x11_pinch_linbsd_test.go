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

//go:build freebsd || linux || netbsd

package glfw_test

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

// TestXIPinchDeltas tests that the scales XInput reports relative to the beginning of a pinch gesture
// become the changes between consecutive events.
func TestXIPinchDeltas(t *testing.T) {
	const (
		begin  = glfw.XIGesturePinchBeginForTest
		update = glfw.XIGesturePinchUpdateForTest
		end    = glfw.XIGesturePinchEndForTest
	)
	for _, tt := range []struct {
		name    string
		evtypes []int32
		scales  []float64
		want    []float64
	}{
		{
			name:    "magnify and shrink",
			evtypes: []int32{begin, update, update, update, end},
			scales:  []float64{1, 1.25, 1.5, 1.2, 1.2},
			want:    []float64{0, 0.25, 0.2, -0.2, 0},
		},
		{
			// A new gesture starts from 1 again, regardless of how far the previous one went.
			name:    "two gestures",
			evtypes: []int32{begin, update, end, begin, update, end},
			scales:  []float64{1, 2, 2, 1, 1.1, 1.1},
			want:    []float64{0, 1, 0, 0, 0.1, 0},
		},
		{
			// A broken scale is skipped without spoiling the next update.
			name:    "broken scales",
			evtypes: []int32{begin, update, update, update, update},
			scales:  []float64{1, 1.5, 0, math.NaN(), 3},
			want:    []float64{0, 0.5, 0, 0, 1},
		},
	} {
		got := glfw.XIPinchDeltasForTest(tt.evtypes, tt.scales)
		for i := range got {
			if math.Abs(got[i]-tt.want[i]) > 1e-12 {
				t.Errorf("%s: event %d: got %v, want %v", tt.name, i, got[i], tt.want[i])
			}
		}
	}
}

// TestXIRotation tests that the angles XInput reports in degrees become radians, and that only update
// events turn.
func TestXIRotation(t *testing.T) {
	for _, tt := range []struct {
		evtype     int32
		deltaAngle float64
		want       float64
	}{
		{
			evtype:     glfw.XIGesturePinchBeginForTest,
			deltaAngle: 10,
			want:       0,
		},
		{
			evtype:     glfw.XIGesturePinchUpdateForTest,
			deltaAngle: 90,
			want:       math.Pi / 2,
		},
		{
			evtype:     glfw.XIGesturePinchUpdateForTest,
			deltaAngle: -45,
			want:       -math.Pi / 4,
		},
		{
			evtype:     glfw.XIGesturePinchUpdateForTest,
			deltaAngle: math.NaN(),
			want:       0,
		},
		{
			evtype:     glfw.XIGesturePinchUpdateForTest,
			deltaAngle: math.Inf(1),
			want:       0,
		},
		{
			evtype:     glfw.XIGesturePinchEndForTest,
			deltaAngle: 10,
			want:       0,
		},
	} {
		if got := glfw.XIRotationForTest(tt.evtype, tt.deltaAngle); math.Abs(got-tt.want) > 1e-12 {
			t.Errorf("XIRotationForTest(%d, %v): got %v, want %v", tt.evtype, tt.deltaAngle, got, tt.want)
		}
	}
}
