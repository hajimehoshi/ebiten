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

package fbdev_test

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/fbdev"
)

// TestVarScreeninfoSize checks the layout FBIOGET_VSCREENINFO writes into. The
// kernel writes the whole struct, so a short one would be a buffer overflow.
func TestVarScreeninfoSize(t *testing.T) {
	if got, want := fbdev.VarScreeninfoSize, uintptr(160); got != want {
		t.Errorf("size of fb_var_screeninfo: got %d, want %d", got, want)
	}
}

// TestNativeWindowSize checks the native window type the drivers that want one
// expect: two 16-bit dimensions.
func TestNativeWindowSize(t *testing.T) {
	if got, want := fbdev.NativeWindowSize, uintptr(4); got != want {
		t.Errorf("size of the native window: got %d, want %d", got, want)
	}
}

func TestRefreshRateForTiming(t *testing.T) {
	testCases := []struct {
		name     string
		pixclock uint32
		xres     uint32
		left     uint32
		right    uint32
		hsync    uint32
		yres     uint32
		upper    uint32
		lower    uint32
		vsync    uint32
		want     int
	}{
		{
			// 1280x720p60: 74.25 MHz over 1650x750.
			name:     "720p60",
			pixclock: 13468,
			xres:     1280,
			left:     220,
			right:    110,
			hsync:    40,
			yres:     720,
			upper:    20,
			lower:    5,
			vsync:    5,
			want:     60,
		},
		{
			// 1024x768@60: 65 MHz over 1344x806.
			name:     "XGA60",
			pixclock: 15384,
			xres:     1024,
			left:     160,
			right:    24,
			hsync:    136,
			yres:     768,
			upper:    29,
			lower:    3,
			vsync:    6,
			want:     60,
		},
		{
			name:     "unset pixclock",
			pixclock: 0,
			xres:     1280,
			yres:     720,
			want:     0,
		},
		{
			name:     "no timings",
			pixclock: 13468,
			want:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := fbdev.RefreshRateForTiming(tc.pixclock, tc.xres, tc.left, tc.right, tc.hsync, tc.yres, tc.upper, tc.lower, tc.vsync)
			if got != tc.want {
				t.Errorf("refresh rate: got %d, want %d", got, tc.want)
			}
		})
	}
}
