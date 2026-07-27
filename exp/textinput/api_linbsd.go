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

//go:build (freebsd || (linux && !android) || netbsd) && !nintendosdk && !playstation5

package textinput

import (
	"sync"

	"github.com/ebitengine/purego"
)

// xPoint mirrors XPoint.
type xPoint struct {
	x int16
	y int16
}

var (
	xFree func(data uintptr) int32

	// XVaCreateNestedList and XSetICValues are variadic in C; both are bound
	// with the single key-value shape used by this file.
	xVaCreateNestedList func(dummy int32, name *byte, value *xPoint, term uintptr) uintptr
	xSetICValues        func(ic uintptr, name string, value uintptr, term uintptr) uintptr
	xmbResetIC          func(ic uintptr) uintptr
)

// ensureX11 loads the libX11 functions on first use and reports whether they
// are available.
var ensureX11 = sync.OnceValue(func() bool {
	var lib uintptr
	for _, name := range []string{"libX11.so.6", "libX11.so"} {
		l, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			lib = l
			break
		}
	}
	if lib == 0 {
		return false
	}
	purego.RegisterLibFunc(&xFree, lib, "XFree")
	purego.RegisterLibFunc(&xVaCreateNestedList, lib, "XVaCreateNestedList")
	purego.RegisterLibFunc(&xSetICValues, lib, "XSetICValues")
	purego.RegisterLibFunc(&xmbResetIC, lib, "XmbResetIC")
	return true
})

// x11IMESpotLocation is the XNSpotLocation value, and x11SpotLocationName is
// the XNSpotLocation key. Both live at package level because the nested list
// handed to XSetICValues stores pointers to them, so they must stay valid
// beyond the XVaCreateNestedList call. Both are used on the main thread only.
var (
	x11IMESpotLocation  xPoint
	x11SpotLocationName = []byte("spotLocation\x00")
)

// setIMESpotLocation sets the XIM spot location of the given X input context:
// the position the input method places its preedit and candidate windows at.
//
// setIMESpotLocation must be called from the main thread.
func setIMESpotLocation(ic uintptr, x, y int) {
	if ic == 0 || !ensureX11() {
		return
	}
	x11IMESpotLocation = xPoint{x: int16(x), y: int16(y)}
	list := xVaCreateNestedList(0, &x11SpotLocationName[0], &x11IMESpotLocation, 0)
	if list == 0 {
		return
	}
	xSetICValues(ic, "preeditAttributes", list, 0)
	xFree(list)
}

// discardIMEComposition discards the composition the input method holds for
// the given X input context, if any.
//
// discardIMEComposition must be called from the main thread.
func discardIMEComposition(ic uintptr) {
	if ic == 0 || !ensureX11() {
		return
	}
	if result := xmbResetIC(ic); result != 0 {
		xFree(result)
	}
}
