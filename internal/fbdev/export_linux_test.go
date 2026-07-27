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

package fbdev

import (
	"unsafe"
)

// NativeWindowSize is the size of the native window type.
const NativeWindowSize = unsafe.Sizeof(nativeWindow{})

// VarScreeninfoSize is the size of the struct FBIOGET_VSCREENINFO fills.
const VarScreeninfoSize = unsafe.Sizeof(fbVarScreeninfo{})

// RefreshRateForTiming returns the refresh rate derived from a mode's timings.
func RefreshRateForTiming(pixclock, xres, leftMargin, rightMargin, hsyncLen, yres, upperMargin, lowerMargin, vsyncLen uint32) int {
	vi := fbVarScreeninfo{
		xres:        xres,
		yres:        yres,
		pixclock:    pixclock,
		leftMargin:  leftMargin,
		rightMargin: rightMargin,
		upperMargin: upperMargin,
		lowerMargin: lowerMargin,
		hsyncLen:    hsyncLen,
		vsyncLen:    vsyncLen,
	}
	return vi.refreshRate()
}
