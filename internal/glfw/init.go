// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2018 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build darwin || freebsd || linux || netbsd || windows

package glfw

import (
	"errors"
	"slices"
)

func terminate() error {
	// Clear global callbacks before destroying windows to prevent
	// callbacks from firing during teardown.
	_glfw.callbacks.monitor = nil

	if err := destroyWindows(); err != nil {
		return err
	}

	if err := destroyCursors(); err != nil {
		return err
	}

	_glfw.monitors = nil

	if err := platformTerminate(); err != nil {
		return err
	}

	_glfw.initialized = false

	return nil
}

// destroyWindows destroys every window. [Window.Destroy] removes the window
// from _glfw.windows, so the iteration runs over a snapshot to avoid skipping
// entries.
func destroyWindows() error {
	for _, w := range slices.Clone(_glfw.windows) {
		if err := w.Destroy(); err != nil {
			return err
		}
	}
	return nil
}

// destroyCursors destroys every cursor. [Cursor.Destroy] removes the cursor
// from _glfw.cursors, so the iteration runs over a snapshot to avoid skipping
// an entry or revisiting one, the latter of which would free a cursor twice.
func destroyCursors() error {
	for _, c := range slices.Clone(_glfw.cursors) {
		if err := c.Destroy(); err != nil {
			return err
		}
	}
	return nil
}

func Init() (err error) {
	defer func() {
		if err != nil {
			// InvalidValue can happen when specific joysticks are used. This issue
			// will be fixed in GLFW 3.3.5. As a temporary fix, ignore this error.
			// See go-gl/glfw#292, go-gl/glfw#324, and glfw/glfw#1763
			// (#1229).
			if errors.Is(err, InvalidValue) {
				err = nil
				return
			}
			_ = terminate()
		}
	}()

	if _glfw.initialized {
		return nil
	}

	_glfw = library{}
	_glfw.hints.init.hatButtons = true

	if err := platformInit(); err != nil {
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
