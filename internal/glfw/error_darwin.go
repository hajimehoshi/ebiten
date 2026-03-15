// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

package glfw

import (
	"errors"
	"fmt"
	"os"
)

var lastError = make(chan error, 1)

func inputError(code ErrorCode, format string, args ...any) {
	err := fmt.Errorf("glfw: "+format+": %w", append(args, code)...)
	select {
	case lastError <- err:
	default:
		fmt.Fprintln(os.Stderr, "GLFW: An uncaught error has occurred:", err)
		fmt.Fprintln(os.Stderr, "GLFW: Please report this bug in the Go package immediately.")
	}
}

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
