// Copyright 2018 The Ebiten Authors
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

package devicescale

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
import "C"

import (
	"fmt"
	"syscall"
)

var (
	user32 = syscall.NewLazyDLL("user32")
)

var (
	procSetProcessDPIAware = user32.NewProc("SetProcessDPIAware")
)

func setProcessDPIAware() error {
	r, _, e := syscall.Syscall(procSetProcessDPIAware.Addr(), 0, 0, 0, 0)
	if e != 0 {
		return fmt.Errorf("devicescale: SetProcessDPIAware failed: error code: %d", e)
	}
	if r == 0 {
		return fmt.Errorf("devicescale: SetProcessDPIAware failed: returned value: %d", r)
	}
	return nil
}

func impl() float64 {
	if err := setProcessDPIAware(); err != nil {
		panic(err)
	}
	dpi := C.int(0)
	if errmsg := C.GoString(C.getDPI(&dpi)); errmsg != "" {
		panic(errmsg)
	}
	return float64(dpi) / 96
}
