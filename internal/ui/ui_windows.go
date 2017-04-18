// Copyright 2016 Hajime Hoshi
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

// +build !js

package ui

// TODO: Use golang.org/x/sys/windows (NewLazyDLL) instead of cgo.

// #cgo LDFLAGS: -lgdi32
//
// #include <windows.h>
//
// static char* getDPI(int* dpi) {
//   HDC dc = GetWindowDC(0);
//   *dpi = GetDeviceCaps(dc, LOGPIXELSX);
//   if (!ReleaseDC(0, dc)) {
//     return "ReleaseDC failed";
//   }
//   return "";
// }
//
// static int getCaptionHeight() {
//   return GetSystemMetrics(SM_CYCAPTION);
// }
//
import "C"

func deviceScale() float64 {
	dpi := C.int(0)
	if errmsg := C.GoString(C.getDPI(&dpi)); errmsg != "" {
		panic(errmsg)
	}
	return float64(dpi) / 96
}

func glfwScale() float64 {
	return deviceScale()
}

func adjustWindowPosition(x, y int) (int, int) {
	// As the video width/height might be wrong,
	// adjust x/y at least to enable to handle the window (#328)
	if x < 0 {
		x = 0
	}
	t := int(C.getCaptionHeight())
	if y < t {
		y = t
	}
	return x, y
}
