// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2018 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"errors"
)

func terminate() error {
	for _, w := range _glfw.windows {
		if err := w.Destroy(); err != nil {
			return err
		}
	}

	for _, c := range _glfw.cursors {
		if err := c.Destroy(); err != nil {
			return err
		}
	}

	_glfw.monitors = nil

	if err := platformTerminate(); err != nil {
		return err
	}

	_glfw.initialized = false

	if err := _glfw.contextSlot.destroy(); err != nil {
		return err
	}

	return nil
}

func Init() (ferr error) {
	defer func() {
		if ferr != nil {
			// InvalidValue can happen when specific joysticks are used. This issue
			// will be fixed in GLFW 3.3.5. As a temporary fix, ignore this error.
			// See go-gl/glfw#292, go-gl/glfw#324, and glfw/glfw#1763
			// (#1229).
			if errors.Is(ferr, InvalidValue) {
				ferr = nil
				return
			}
			_ = terminate()
		}
	}()

	if _glfw.initialized {
		return nil
	}

	_glfw.hints.init.hatButtons = true

	if err := platformInit(); err != nil {
		return err
	}

	if err := _glfw.contextSlot.create(); err != nil {
		return err
	}

	_glfw.initialized = true

	if err := defaultWindowHints(); err != nil {
		return err
	}
	return nil
}

func Terminate() error {
	if !_glfw.initialized {
		return nil
	}
	if err := terminate(); err != nil {
		return err
	}
	return nil
}
