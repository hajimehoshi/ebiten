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
	"image"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/internal/ui"
)

var deviceScaleFactorsForTest = []float64{1, 1.1, 1.25, 1.5, 1.75, 2, 2.25, 2.5, 3}

// TestOutsideSizeInDIPReportsRequestedSize tests that a window that was given a size reports that
// size back, even at a device scale factor where the size is not representable in pixels (#2978).
func TestOutsideSizeInDIPReportsRequestedSize(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		for req := 1; req <= 2000; req++ {
			pw, ph := ui.WindowSizeInGLFWPixelsForTest(req, req, s)
			w, h := ui.OutsideSizeInDIPForTest(pw, ph, req, req, false, s)
			if w != float64(req) || h != float64(req) {
				t.Errorf("scale %v: outsideSizeInDIP(%d, %d, %d, %d, false, %v) = (%v, %v); want (%d, %d)", s, pw, ph, req, req, s, w, h, req, req)
			}
		}
	}
}

// TestOutsideSizeInDIPUsesActualSize tests that a window that does not have the size it was given
// reports its actual size, as the specified size at SetSize and the actual window size might not
// match (#1163).
func TestOutsideSizeInDIPUsesActualSize(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		for req := 1; req <= 2000; req++ {
			pw, ph := ui.WindowSizeInGLFWPixelsForTest(req, req, s)
			// One pixel wider than it was asked to be is the smallest deviation the size
			// comparison has to notice.
			aw, ah := pw+1, ph
			w, h := ui.OutsideSizeInDIPForTest(aw, ah, req, req, false, s)
			wantW := ui.DIPFromGLFWPixelForTest(float64(aw), s)
			wantH := ui.DIPFromGLFWPixelForTest(float64(ah), s)
			if w != wantW || h != wantH {
				t.Errorf("scale %v: outsideSizeInDIP(%d, %d, %d, %d, false, %v) = (%v, %v); want (%v, %v)", s, aw, ah, req, req, s, w, h, wantW, wantH)
			}
		}
	}
}

// TestOutsideSizeInDIPInFullscreenUsesActualSize tests that the requested size is not reported in
// fullscreen, where it is the size to restore on leaving fullscreen rather than the current size.
func TestOutsideSizeInDIPInFullscreenUsesActualSize(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		for req := 1; req <= 2000; req++ {
			pw, ph := ui.WindowSizeInGLFWPixelsForTest(req, req, s)
			w, h := ui.OutsideSizeInDIPForTest(pw, ph, req, req, true, s)
			wantW := ui.DIPFromGLFWPixelForTest(float64(pw), s)
			wantH := ui.DIPFromGLFWPixelForTest(float64(ph), s)
			if w != wantW || h != wantH {
				t.Errorf("scale %v: outsideSizeInDIP(%d, %d, %d, %d, true, %v) = (%v, %v); want (%v, %v)", s, pw, ph, req, req, s, w, h, wantW, wantH)
			}
		}
	}
}

// TestWindowSizeToRestoreRestoresCapturedSize tests that a window comes back from fullscreen at
// the exact pixel size it had, even at a device scale factor where that size is not representable
// in device-independent pixels.
func TestWindowSizeToRestoreRestoresCapturedSize(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		m := ui.NewMonitorForTest(image.Rect(0, 0, 1920, 1080), s)
		for px := 1; px <= 2000; px++ {
			// The size in device-independent pixels is the one the framebuffer size callback
			// derives from the window's pixel size, which might not convert back to it.
			dip := int(math.Round(ui.DIPFromGLFWPixelForTest(float64(px), s)))
			w, h := ui.WindowSizeToRestoreForTest(px, px, m, dip, dip, m)
			if w != px || h != px {
				t.Errorf("scale %v: windowSizeToRestore(%d, %d, m, %d, %d, m) = (%d, %d); want (%d, %d)", s, px, px, dip, dip, w, h, px, px)
			}
		}
	}
}

// TestWindowSizeToRestoreOnAnotherMonitorUsesSizeInDIP tests that a size captured on one monitor
// is not restored on another, where the same pixel count is a different apparent size.
func TestWindowSizeToRestoreOnAnotherMonitorUsesSizeInDIP(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		b := image.Rect(0, 0, 1920, 1080)
		origMonitor := ui.NewMonitorForTest(b, s)
		// A monitor with the same bounds and scale factor but a different identity is enough:
		// a captured size is restored only on the very monitor it was captured on.
		m := ui.NewMonitorForTest(b, s)
		for dip := 1; dip <= 2000; dip++ {
			wantW, wantH := ui.WindowSizeInGLFWPixelsForTest(dip, dip, s)
			// One pixel wider than the converted size is the smallest captured size whose
			// restoration would be visible.
			w, h := ui.WindowSizeToRestoreForTest(wantW+1, wantH+1, origMonitor, dip, dip, m)
			if w != wantW || h != wantH {
				t.Errorf("scale %v: windowSizeToRestore(%d, %d, origMonitor, %d, %d, m) = (%d, %d); want (%d, %d)", s, wantW+1, wantH+1, dip, dip, w, h, wantW, wantH)
			}
		}
	}
}

// TestWindowSizeToRestoreWithoutCapturedSizeUsesSizeInDIP tests that a window whose size was not
// captured, as when the platform makes it fullscreen on its own, is restored to the size in
// device-independent pixels.
func TestWindowSizeToRestoreWithoutCapturedSizeUsesSizeInDIP(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		m := ui.NewMonitorForTest(image.Rect(0, 0, 1920, 1080), s)
		for dip := 1; dip <= 2000; dip++ {
			w, h := ui.WindowSizeToRestoreForTest(ui.InvalidSizeForTest, ui.InvalidSizeForTest, nil, dip, dip, m)
			wantW, wantH := ui.WindowSizeInGLFWPixelsForTest(dip, dip, s)
			if w != wantW || h != wantH {
				t.Errorf("scale %v: windowSizeToRestore(invalidSize, invalidSize, nil, %d, %d, m) = (%d, %d); want (%d, %d)", s, dip, dip, w, h, wantW, wantH)
			}
		}
	}
}

var monitorBoundsForTest = []image.Rectangle{
	image.Rect(0, 0, 1920, 1080),
	image.Rect(-2560, -300, 0, 1140),
}

// TestWindowPositionInDIPKeepsRequestedPosition tests that a window that moved to the pixel
// position a requested position produces keeps that position, even at a device scale factor where
// it is not representable in pixels (#2978).
func TestWindowPositionInDIPKeepsRequestedPosition(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		for _, b := range monitorBoundsForTest {
			m := ui.NewMonitorForTest(b, s)
			for req := -1000; req <= 2000; req++ {
				px, py := ui.WindowPositionInGLFWPixelsForTest(req, req, m)
				x, y := ui.WindowPositionInDIPForTest(px, py, req, req, m)
				if x != req || y != req {
					t.Errorf("scale %v, monitor %v: windowPositionInDIP(%d, %d, %d, %d, m) = (%d, %d); want (%d, %d)", s, b, px, py, req, req, x, y, req, req)
				}
			}
		}
	}
}

// TestWindowPositionInDIPUsesActualPosition tests that a window that moved anywhere else uses
// where it is, as the move did not come from a request.
func TestWindowPositionInDIPUsesActualPosition(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		for _, b := range monitorBoundsForTest {
			m := ui.NewMonitorForTest(b, s)
			for req := -1000; req <= 2000; req++ {
				px, py := ui.WindowPositionInGLFWPixelsForTest(req, req, m)
				// One pixel to the right of the stored position is the smallest move the
				// position comparison has to notice.
				ax, ay := px+1, py
				x, y := ui.WindowPositionInDIPForTest(ax, ay, req, req, m)
				wantX := int(math.Round(ui.DIPFromGLFWPixelForTest(float64(ax-b.Min.X), s)))
				wantY := int(math.Round(ui.DIPFromGLFWPixelForTest(float64(ay-b.Min.Y), s)))
				if x != wantX || y != wantY {
					t.Errorf("scale %v, monitor %v: windowPositionInDIP(%d, %d, %d, %d, m) = (%d, %d); want (%d, %d)", s, b, ax, ay, req, req, x, y, wantX, wantY)
				}
			}
		}
	}
}

// TestWindowPositionInDIPIsStable tests that reporting the same position again does not change it.
// A window manager can send a move event for a position the window already has, and the stored
// position must not drift with every one of them.
func TestWindowPositionInDIPIsStable(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		for _, b := range monitorBoundsForTest {
			m := ui.NewMonitorForTest(b, s)
			for px := -1000; px <= 2000; px++ {
				x, y := ui.WindowPositionInDIPForTest(px, px, 0, 0, m)
				x2, y2 := ui.WindowPositionInDIPForTest(px, px, x, y, m)
				if x2 != x || y2 != y {
					t.Errorf("scale %v, monitor %v: windowPositionInDIP(%d, %d, %d, %d, m) = (%d, %d); want (%d, %d)", s, b, px, px, x, y, x2, y2, x, y)
				}
			}
		}
	}
}

// TestWindowSizeInGLFWPixelsIsNearestSize tests that a size in device-independent pixels converts
// to the nearest pixel count. Truncating it instead would make a window that is never larger than
// it was asked to be and can be a whole pixel smaller.
func TestWindowSizeInGLFWPixelsIsNearestSize(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		for dip := 1; dip <= 2000; dip++ {
			w, h := ui.WindowSizeInGLFWPixelsForTest(dip, dip, s)
			want := ui.DIPToGLFWPixelForTest(float64(dip), s)
			if math.Abs(float64(w)-want) > 0.5 || math.Abs(float64(h)-want) > 0.5 {
				t.Errorf("scale %v: windowSizeInGLFWPixels(%d, %d, %v) = (%d, %d); want the nearest integers to %v", s, dip, dip, s, w, h, want)
			}
		}
	}
}

// TestWindowPositionInGLFWPixelsIsNearestPosition tests that a position in device-independent
// pixels converts to the nearest pixel position, as the size does.
func TestWindowPositionInGLFWPixelsIsNearestPosition(t *testing.T) {
	for _, s := range deviceScaleFactorsForTest {
		for _, b := range monitorBoundsForTest {
			m := ui.NewMonitorForTest(b, s)
			for dip := -1000; dip <= 2000; dip++ {
				x, y := ui.WindowPositionInGLFWPixelsForTest(dip, dip, m)
				want := ui.DIPToGLFWPixelForTest(float64(dip), s)
				if math.Abs(float64(x-b.Min.X)-want) > 0.5 || math.Abs(float64(y-b.Min.Y)-want) > 0.5 {
					t.Errorf("scale %v, monitor %v: windowPositionInGLFWPixels(%d, %d, m) = (%d, %d); want the nearest integers to %v relative to the monitor", s, b, dip, dip, x, y, want)
				}
			}
		}
	}
}
