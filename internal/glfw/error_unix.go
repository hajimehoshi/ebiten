// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd || openbsd

package glfw

// #define GLFW_INCLUDE_NONE
// #include "glfw3_unix.h"
//
// void goErrorCB(int code, char* desc);
//
// #cgo noescape glfwSetErrorCallbackCB
// static void glfwSetErrorCallbackCB() {
//   glfwSetErrorCallback((GLFWerrorfun)goErrorCB);
// }
import "C"

import (
	"errors"
	"fmt"
	"os"
)

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

// fetchErrorIgnoringPlatformError is fetchError ignoring platformError.
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
