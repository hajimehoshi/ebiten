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

package ui_test

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

func TestVirtualKeyboardOffset(t *testing.T) {
	const (
		screenWidth  = 640
		screenHeight = 480
	)

	testCases := []struct {
		name               string
		offscreenWidth     float64
		offscreenHeight    float64
		caretBounds        image.Rectangle
		caretKnown         bool
		visibleRegion      image.Rectangle
		visibleRegionKnown bool
		wantScale          float64
		wantOffsetY        float64
	}{
		{
			name:               "no session",
			offscreenWidth:     640,
			offscreenHeight:    480,
			visibleRegion:      image.Rect(0, 0, 640, 300),
			visibleRegionKnown: true,
			wantScale:          1,
			wantOffsetY:        0,
		},
		{
			name:            "region unknown",
			offscreenWidth:  640,
			offscreenHeight: 480,
			caretBounds:     image.Rect(100, 400, 101, 420),
			caretKnown:      true,
			wantScale:       1,
			wantOffsetY:     0,
		},
		{
			name:               "caret above the keyboard",
			offscreenWidth:     640,
			offscreenHeight:    480,
			caretBounds:        image.Rect(100, 100, 101, 120),
			caretKnown:         true,
			visibleRegion:      image.Rect(0, 0, 640, 300),
			visibleRegionKnown: true,
			wantScale:          1,
			wantOffsetY:        0,
		},
		{
			name:               "caret behind the keyboard",
			offscreenWidth:     640,
			offscreenHeight:    480,
			caretBounds:        image.Rect(100, 400, 101, 420),
			caretKnown:         true,
			visibleRegion:      image.Rect(0, 0, 640, 300),
			visibleRegionKnown: true,
			wantScale:          1,
			wantOffsetY:        -120,
		},
		{
			// The exposed strip at the bottom equals the shift, and must not be taller
			// than the covered region, or it would show outside the keyboard.
			name:               "caret at the very bottom",
			offscreenWidth:     640,
			offscreenHeight:    480,
			caretBounds:        image.Rect(0, 460, 1, 480),
			caretKnown:         true,
			visibleRegion:      image.Rect(0, 0, 640, 300),
			visibleRegionKnown: true,
			wantScale:          1,
			wantOffsetY:        -180,
		},
		{
			// The caret is in logical units, so the shift scales with the screen.
			name:               "scaled screen",
			offscreenWidth:     320,
			offscreenHeight:    240,
			caretBounds:        image.Rect(0, 200, 1, 220),
			caretKnown:         true,
			visibleRegion:      image.Rect(0, 0, 640, 300),
			visibleRegionKnown: true,
			wantScale:          2,
			wantOffsetY:        -140,
		},
		{
			// The letterbox offset is kept: the shift is added to it. Here the offscreen
			// is centered at 120, so the caret bottom sits at 340 and the shift is -40.
			name:               "letterboxed screen",
			offscreenWidth:     640,
			offscreenHeight:    240,
			caretBounds:        image.Rect(0, 200, 1, 220),
			caretKnown:         true,
			visibleRegion:      image.Rect(0, 0, 640, 300),
			visibleRegionKnown: true,
			wantScale:          1,
			wantOffsetY:        80,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ui.SetVirtualKeyboardStateForTest(tc.caretBounds, tc.caretKnown, tc.visibleRegion, tc.visibleRegionKnown)
			defer ui.SetVirtualKeyboardStateForTest(image.Rectangle{}, false, image.Rectangle{}, false)

			scale, _, offsetY := ui.ScreenScaleAndOffsetsForTest(screenWidth, screenHeight, tc.offscreenWidth, tc.offscreenHeight)
			if scale != tc.wantScale {
				t.Errorf("scale: got %v, want %v", scale, tc.wantScale)
			}
			if offsetY != tc.wantOffsetY {
				t.Errorf("offsetY: got %v, want %v", offsetY, tc.wantOffsetY)
			}
		})
	}
}

// TestVirtualKeyboardOffsetPutsCaretAtTheEdge confirms the invariant the shift exists for: a
// caret behind the keyboard ends up exactly at the edge of the visible region.
func TestVirtualKeyboardOffsetPutsCaretAtTheEdge(t *testing.T) {
	const (
		screenWidth     = 640
		screenHeight    = 480
		offscreenWidth  = 320
		offscreenHeight = 240
	)

	caretBounds := image.Rect(0, 200, 1, 220)
	visibleRegion := image.Rect(0, 0, 640, 300)

	ui.SetVirtualKeyboardStateForTest(caretBounds, true, visibleRegion, true)
	defer ui.SetVirtualKeyboardStateForTest(image.Rectangle{}, false, image.Rectangle{}, false)

	scale, _, offsetY := ui.ScreenScaleAndOffsetsForTest(screenWidth, screenHeight, offscreenWidth, offscreenHeight)
	if got, want := float64(caretBounds.Max.Y)*scale+offsetY, float64(visibleRegion.Max.Y); got != want {
		t.Errorf("the caret bottom on the screen: got %v, want %v", got, want)
	}
}
