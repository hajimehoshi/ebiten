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
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

func TestWheelScrollAmount(t *testing.T) {
	for _, tt := range []struct {
		name       string
		notches    float64
		setting    uint32
		wantAmount float64
		wantUnit   glfw.ScrollUnit
	}{
		{
			name:       "default lines",
			notches:    1,
			setting:    3,
			wantAmount: 3,
			wantUnit:   glfw.ScrollUnitLine,
		},
		{
			name:       "fractional notches",
			notches:    0.25,
			setting:    3,
			wantAmount: 0.75,
			wantUnit:   glfw.ScrollUnitLine,
		},
		{
			name:       "custom lines",
			notches:    -2,
			setting:    5,
			wantAmount: -10,
			wantUnit:   glfw.ScrollUnitLine,
		},
		{
			name:       "no scrolling",
			notches:    1,
			setting:    0,
			wantAmount: 0,
			wantUnit:   glfw.ScrollUnitLine,
		},
		{
			name:       "page scrolling",
			notches:    -1.5,
			setting:    glfw.WheelPageScroll,
			wantAmount: -1.5,
			wantUnit:   glfw.ScrollUnitPage,
		},
	} {
		amount, unit := glfw.WheelScrollAmount(tt.notches, tt.setting)
		if amount != tt.wantAmount || unit != tt.wantUnit {
			t.Errorf("%s: got (%v, %v), want (%v, %v)", tt.name, amount, unit, tt.wantAmount, tt.wantUnit)
		}
	}
}
