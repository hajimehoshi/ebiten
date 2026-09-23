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

//go:build !android && !ios && !js && !nintendosdk && !playstation5

package ui_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/glfw"
	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestGLFWInputScrollUnits(t *testing.T) {
	for _, tt := range []struct {
		name         string
		unit         glfw.ScrollUnit
		scrollDeltaX float64
		scrollDeltaY float64
		wantX        float64
		wantY        float64
	}{
		{
			name:         "pixel",
			unit:         glfw.ScrollUnitPixel,
			scrollDeltaX: 12.5,
			scrollDeltaY: -7,
			wantX:        12.5,
			wantY:        -7,
		},
		{
			name:         "notch",
			unit:         glfw.ScrollUnitNotch,
			scrollDeltaX: 1,
			scrollDeltaY: -2,
			wantX:        100,
			wantY:        -200,
		},
		{
			name:         "line",
			unit:         glfw.ScrollUnitLine,
			scrollDeltaX: 3,
			scrollDeltaY: -1.5,
			wantX:        100,
			wantY:        -50,
		},
	} {
		i := ui.NewGLFWInputForTest()
		i.HandleScrollForTest(1, -2, tt.scrollDeltaX, tt.scrollDeltaY, tt.unit)
		var s ui.InputState
		i.ReadForTest(&s)
		if s.WheelX != 1 || s.WheelY != -2 {
			t.Errorf("%s: wheel: got (%v, %v), want (1, -2)", tt.name, s.WheelX, s.WheelY)
		}
		if s.ScrollDeltaX != tt.wantX || s.ScrollDeltaY != tt.wantY {
			t.Errorf("%s: scroll: got (%v, %v), want (%v, %v)", tt.name, s.ScrollDeltaX, s.ScrollDeltaY, tt.wantX, tt.wantY)
		}
	}
}

// TestGLFWInputScrollPages tests that page-unit scrolling reaches a tick's input snapshot once the
// content size is known, including a tick run outside the regular frame.
func TestGLFWInputScrollPages(t *testing.T) {
	i := ui.NewGLFWInputForTest()
	i.HandleScrollForTest(0.5, -1, 0.5, -1, glfw.ScrollUnitPage)

	// Without a content size, the pages stay pending rather than being lost.
	var s ui.InputState
	i.ReadForTest(&s)
	if s.WheelX != 0.5 || s.WheelY != -1 {
		t.Errorf("wheel: got (%v, %v), want (0.5, -1)", s.WheelX, s.WheelY)
	}
	if s.ScrollDeltaX != 0 || s.ScrollDeltaY != 0 {
		t.Errorf("scroll before the content size is known: got (%v, %v), want (0, 0)", s.ScrollDeltaX, s.ScrollDeltaY)
	}

	i.SetContentSizeForTest(640, 480)
	i.ReadForTest(&s)
	if s.ScrollDeltaX != 320 || s.ScrollDeltaY != -480 {
		t.Errorf("scroll after the content size is known: got (%v, %v), want (320, -480)", s.ScrollDeltaX, s.ScrollDeltaY)
	}

	// The pages are consumed by the tick that reported them.
	i.ReadForTest(&s)
	if s.ScrollDeltaX != 0 || s.ScrollDeltaY != 0 {
		t.Errorf("scroll at the next tick: got (%v, %v), want (0, 0)", s.ScrollDeltaX, s.ScrollDeltaY)
	}

	// A page scrolled between ticks is converted at the next tick, whichever frame runs it.
	i.HandleScrollForTest(0, -2, 0, -2, glfw.ScrollUnitPage)
	i.ReadForTest(&s)
	if s.ScrollDeltaX != 0 || s.ScrollDeltaY != -960 {
		t.Errorf("scroll at a later tick: got (%v, %v), want (0, -960)", s.ScrollDeltaX, s.ScrollDeltaY)
	}
}

// TestGLFWInputScrollAnomalyDropsBothChannels tests that a wheel value dropped as anomalous (#3390)
// drops its scroll amount too.
func TestGLFWInputScrollAnomalyDropsBothChannels(t *testing.T) {
	i := ui.NewGLFWInputForTest()
	i.HandleScrollForTest(0, 1, 0, 1, glfw.ScrollUnitNotch)
	// A sudden spike right after a regular value is dropped.
	i.HandleScrollForTest(0, 60, 0, 60, glfw.ScrollUnitNotch)

	var s ui.InputState
	i.ReadForTest(&s)
	if s.WheelY != 1 {
		t.Errorf("wheel: got %v, want 1", s.WheelY)
	}
	if s.ScrollDeltaY != 100 {
		t.Errorf("scroll: got %v, want 100", s.ScrollDeltaY)
	}
}
