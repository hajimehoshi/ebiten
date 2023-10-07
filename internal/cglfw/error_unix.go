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
	"errors"
	"fmt"
	"os"
)

// ErrorCode corresponds to an error code.
type ErrorCode int

const (
	NotInitialized     = ErrorCode(0x00010001)
	NoCurrentContext   = ErrorCode(0x00010002)
	InvalidEnum        = ErrorCode(0x00010003)
	InvalidValue       = ErrorCode(0x00010004)
	OutOfMemory        = ErrorCode(0x00010005)
	APIUnavailable     = ErrorCode(0x00010006)
	VersionUnavailable = ErrorCode(0x00010007)
	PlatformError      = ErrorCode(0x00010008)
	FormatUnavailable  = ErrorCode(0x00010009)
	NoWindowContext    = ErrorCode(0x0001000A)
)

func (e ErrorCode) Error() string {
	switch e {
	case NotInitialized:
		return "the GLFW library is not initialized"
	case NoCurrentContext:
		return "there is no current context"
	case InvalidEnum:
		return "invalid argument for enum parameter"
	case InvalidValue:
		return "invalid value for parameter"
	case OutOfMemory:
		return "out of memory"
	case APIUnavailable:
		return "the requested API is unavailable"
	case VersionUnavailable:
		return "the requested API version is unavailable"
	case PlatformError:
		return "a platform-specific error occurred"
	case FormatUnavailable:
		return "the requested format is unavailable"
	case NoWindowContext:
		return "the specified window has no context"
	default:
		return fmt.Sprintf("GLFW error (%d)", e)
	}
}

// Note: There are many cryptic caveats to proper error handling here.
// See: https://github.com/go-gl/glfw3/pull/86

// lastError holds the value of the last error.
var lastError = make(chan error, 1)

//export goErrorCB
func goErrorCB(code C.int, desc *C.char) {
	err := fmt.Errorf("glfw: %s: %w", C.GoString(desc), ErrorCode(code))
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
		if errors.Is(err, PlatformError) {
			fmt.Fprintln(os.Stderr, err)
			return nil
		}
		return err
	default:
		return nil
	}
}
