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

package glfw

import (
	"fmt"
)

// SmokeTestX11 exercises the purego foundation against a live X server: the
// libX11 bindings, platform initialization including the fixed-arity
// variadic XGetIMValues call, the X error handler callback, and the
// fixed-arity variadic XCreateIC call. platformTerminate is not called, as
// its GLX/EGL termination is not implemented yet.
func SmokeTestX11() error {
	if err := platformInit(); err != nil {
		return err
	}

	if _glfw.platformWindow.helperWindowHandle == 0 {
		return fmt.Errorf("glfw: smoke: no helper window")
	}
	if _glfw.platformWindow.UTF8_STRING == 0 {
		return fmt.Errorf("glfw: smoke: UTF8_STRING atom not interned")
	}

	// The error handler callback: destroying an invalid window must invoke
	// the handler with BadWindow.
	grabErrorHandlerX11()
	xDestroyWindow(_glfw.platformWindow.display, 0xdeadbeef)
	xSync(_glfw.platformWindow.display, false)
	errorCode := _glfw.platformWindow.errorCode
	releaseErrorHandlerX11()
	if errorCode == _Success {
		return fmt.Errorf("glfw: smoke: the X error handler was not invoked")
	}

	// The fixed-arity variadic XCreateIC call, when an input method is
	// available (it is not under a bare Xvfb without an IM server).
	if _glfw.platformWindow.im != 0 {
		ic := xCreateIC(_glfw.platformWindow.im,
			"inputStyle", _XIMPreeditNothing|_XIMStatusNothing,
			"clientWindow", _glfw.platformWindow.helperWindowHandle,
			"focusWindow", _glfw.platformWindow.helperWindowHandle,
			0)
		if ic == 0 {
			return fmt.Errorf("glfw: smoke: XCreateIC failed")
		}
		xDestroyIC(ic)
	}

	return nil
}
