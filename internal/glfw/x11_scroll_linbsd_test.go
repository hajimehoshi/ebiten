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
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
)

func TestXIScrollAxisOffset(t *testing.T) {
	axes := []glfw.XIScrollAxis{
		glfw.NewXIScrollAxis(2, glfw.XIScrollTypeVertical, 15),
		glfw.NewXIScrollAxis(3, glfw.XIScrollTypeHorizontal, 15),
	}
	for _, tt := range []struct {
		name   string
		number int32
		delta  float64
		wantX  float64
		wantY  float64
	}{
		{
			name:   "one notch down",
			number: 2,
			delta:  15,
			wantX:  0,
			wantY:  -1,
		},
		{
			name:   "half a notch down",
			number: 2,
			delta:  7.5,
			wantX:  0,
			wantY:  -0.5,
		},
		{
			name:   "one notch up",
			number: 2,
			delta:  -15,
			wantX:  0,
			wantY:  1,
		},
		{
			name:   "one notch left",
			number: 3,
			delta:  -15,
			wantX:  1,
			wantY:  0,
		},
		{
			name:   "one notch right",
			number: 3,
			delta:  15,
			wantX:  -1,
			wantY:  0,
		},
		{
			name:   "unknown axis",
			number: 7,
			delta:  100,
			wantX:  0,
			wantY:  0,
		},
	} {
		x, y := glfw.XIScrollAxisOffset(axes, tt.number, tt.delta)
		if x != tt.wantX || y != tt.wantY {
			t.Errorf("%s: got (%v, %v), want (%v, %v)", tt.name, x, y, tt.wantX, tt.wantY)
		}
	}
}

func TestXIScrollAxisOffsetInvertedIncrement(t *testing.T) {
	axes := []glfw.XIScrollAxis{glfw.NewXIScrollAxis(0, glfw.XIScrollTypeVertical, -15)}
	if x, y := glfw.XIScrollAxisOffset(axes, 0, -15); x != 0 || y != -1 {
		t.Errorf("got (%v, %v), want (0, -1)", x, y)
	}
}

func TestXIScrollAxisOffsetZeroIncrement(t *testing.T) {
	axes := []glfw.XIScrollAxis{glfw.NewXIScrollAxis(0, glfw.XIScrollTypeVertical, 0)}
	if x, y := glfw.XIScrollAxisOffset(axes, 0, 15); x != 0 || y != 0 {
		t.Errorf("got (%v, %v), want (0, 0)", x, y)
	}
}

// TestXIPendingScroll tests that the scroll offsets of a raw motion event reach the device motion
// event of the same input report, and only that one.
func TestXIPendingScroll(t *testing.T) {
	pending := glfw.XIPendingScrolls{}

	// The device motion event of the same report, at the same time, takes the offsets once.
	glfw.XIRecordPendingScroll(pending, 9, 1000, 0, -1)
	if x, y := glfw.XITakePendingScroll(pending, 9, 1000); x != 0 || y != -1 {
		t.Errorf("same report: got (%v, %v), want (0, -1)", x, y)
	}
	if x, y := glfw.XITakePendingScroll(pending, 9, 1000); x != 0 || y != 0 {
		t.Errorf("taken again: got (%v, %v), want (0, 0)", x, y)
	}

	// A device motion event of a later report does not take the offsets of an earlier raw motion
	// event, whose own device motion event was delivered elsewhere.
	glfw.XIRecordPendingScroll(pending, 9, 1000, 0, -1)
	if x, y := glfw.XITakePendingScroll(pending, 9, 1016); x != 0 || y != 0 {
		t.Errorf("later report: got (%v, %v), want (0, 0)", x, y)
	}

	// A raw motion event replaces the offsets of the device's previous one.
	glfw.XIRecordPendingScroll(pending, 9, 1000, 0, -1)
	glfw.XIRecordPendingScroll(pending, 9, 1016, 1, 0)
	if x, y := glfw.XITakePendingScroll(pending, 9, 1016); x != 1 || y != 0 {
		t.Errorf("replaced: got (%v, %v), want (1, 0)", x, y)
	}

	// Two reports in the same millisecond: the second raw motion event, without scrolling, replaces
	// the first, whose device motion event was delivered elsewhere.
	glfw.XIRecordPendingScroll(pending, 9, 1000, 0, -1)
	glfw.XIRecordPendingScroll(pending, 9, 1000, 0, 0)
	if x, y := glfw.XITakePendingScroll(pending, 9, 1000); x != 0 || y != 0 {
		t.Errorf("same timestamp, replaced by no scrolling: got (%v, %v), want (0, 0)", x, y)
	}

	// Devices are independent.
	glfw.XIRecordPendingScroll(pending, 9, 1000, 0, -1)
	if x, y := glfw.XITakePendingScroll(pending, 10, 1000); x != 0 || y != 0 {
		t.Errorf("another device: got (%v, %v), want (0, 0)", x, y)
	}
	if x, y := glfw.XITakePendingScroll(pending, 9, 1000); x != 0 || y != -1 {
		t.Errorf("the device after another: got (%v, %v), want (0, -1)", x, y)
	}
}
