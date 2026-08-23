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

// TestOutsideSizeInDIPInNativeFullscreenUsesActualSize tests that the requested size is not
// reported in the native fullscreen mode, where it is the size to restore on leaving fullscreen
// rather than the current size.
func TestOutsideSizeInDIPInNativeFullscreenUsesActualSize(t *testing.T) {
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
