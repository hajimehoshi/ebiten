// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd || openbsd

package cglfw

// #define GLFW_INCLUDE_NONE
// #include "glfw3_unix.h"
//
// void goErrorCB(int code, char* desc);
//
// static void glfwSetErrorCallbackCB() {
//   glfwSetErrorCallback((GLFWerrorfun)goErrorCB);
// }
import "C"

import (
	"fmt"
	"os"
)

// ErrorCode corresponds to an error code.
type ErrorCode int

// Error codes that are translated to panics and the programmer should not
// expect to handle.
const (
	notInitialized   ErrorCode = C.GLFW_NOT_INITIALIZED    // GLFW has not been initialized.
	noCurrentContext ErrorCode = C.GLFW_NO_CURRENT_CONTEXT // No context is current.
	invalidEnum      ErrorCode = C.GLFW_INVALID_ENUM       // One of the enum parameters for the function was given an invalid enum.
	invalidValue     ErrorCode = C.GLFW_INVALID_VALUE      // One of the parameters for the function was given an invalid value.
	outOfMemory      ErrorCode = C.GLFW_OUT_OF_MEMORY      // A memory allocation failed.
	platformError    ErrorCode = C.GLFW_PLATFORM_ERROR     // A platform-specific error occurred that does not match any of the more specific categories.
)

const (
	// APIUnavailable is the error code used when GLFW could not find support
	// for the requested client API on the system.
	//
	// The installed graphics driver does not support the requested client API,
	// or does not support it via the chosen context creation backend. Below
	// are a few examples.
	//
	// Some pre-installed Windows graphics drivers do not support OpenGL. AMD
	// only supports OpenGL ES via EGL, while Nvidia and Intel only supports it
	// via a WGL or GLX extension. OS X does not provide OpenGL ES at all. The
	// Mesa EGL, OpenGL and OpenGL ES libraries do not interface with the
	// Nvidia binary driver.
	APIUnavailable ErrorCode = C.GLFW_API_UNAVAILABLE

	// VersionUnavailable is the error code used when the requested OpenGL or
	// OpenGL ES (including any requested profile or context option) is not
	// available on this machine.
	//
	// The machine does not support your requirements. If your application is
	// sufficiently flexible, downgrade your requirements and try again.
	// Otherwise, inform the user that their machine does not match your
	// requirements.
	//
	// Future invalid OpenGL and OpenGL ES versions, for example OpenGL 4.8 if
	// 5.0 comes out before the 4.x series gets that far, also fail with this
	// error and not GLFW_INVALID_VALUE, because GLFW cannot know what future
	// versions will exist.
	VersionUnavailable ErrorCode = C.GLFW_VERSION_UNAVAILABLE

	// FormatUnavailable is the error code used for both window creation and
	// clipboard querying format errors.
	//
	// If emitted during window creation, the requested pixel format is not
	// supported. This means one or more hard constraints did not match any of
	// the available pixel formats. If your application is sufficiently
	// flexible, downgrade your requirements and try again. Otherwise, inform
	// the user that their machine does not match your requirements.
	//
	// If emitted when querying the clipboard, the contents of the clipboard
	// could not be converted to the requested format. You should ignore the
	// error or report it to the user, as appropriate.
	FormatUnavailable ErrorCode = C.GLFW_FORMAT_UNAVAILABLE
)

func (e ErrorCode) String() string {
	switch e {
	case notInitialized:
		return "NotInitialized"
	case noCurrentContext:
		return "NoCurrentContext"
	case invalidEnum:
		return "InvalidEnum"
	case invalidValue:
		return "InvalidValue"
	case outOfMemory:
		return "OutOfMemory"
	case platformError:
		return "PlatformError"
	case APIUnavailable:
		return "APIUnavailable"
	case VersionUnavailable:
		return "VersionUnavailable"
	case FormatUnavailable:
		return "FormatUnavailable"
	default:
		return fmt.Sprintf("ErrorCode(%d)", e)
	}
}

// Error holds error code and description.
type Error struct {
	Code ErrorCode
	Desc string
}

// Error prints the error code and description in a readable format.
func (e *Error) Error() string {
	return fmt.Sprintf("%s: %s", e.Code.String(), e.Desc)
}

// Note: There are many cryptic caveats to proper error handling here.
// See: https://github.com/go-gl/glfw3/pull/86

// lastError holds the value of the last error.
var lastError = make(chan *Error, 1)

//export goErrorCB
func goErrorCB(code C.int, desc *C.char) {
	err := &Error{ErrorCode(code), C.GoString(desc)}
	select {
	case lastError <- err:
	default:
		fmt.Fprintln(os.Stderr, "GLFW: An uncaught error has occurred:", err)
		fmt.Fprintln(os.Stderr, "GLFW: Please report this bug in the Go package immediately.")
	}
}

// Set the glfw callback internally
func init() {
	C.glfwSetErrorCallbackCB()
}

// fetchErrorIgnoringPlatformError is fetchError igoring platformError.
func fetchErrorIgnoringPlatformError() error {
	select {
	case err := <-lastError:
		if err.Code == platformError {
			fmt.Fprintln(os.Stderr, err)
			return nil
		}
		return err
	default:
		return nil
	}
}
