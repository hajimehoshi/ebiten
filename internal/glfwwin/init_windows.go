// Copyright 2002-2006 Marcus Geelnard
// Copyright 2006-2019 Camilla Löwy
// Copyright 2022 The Ebiten Authors
//
// This software is provided 'as-is', without any express or implied
// warranty. In no event will the authors be held liable for any damages
// arising from the use of this software.
//
// Permission is granted to anyone to use this software for any purpose,
// including commercial applications, and to alter it and redistribute it
// freely, subject to the following restrictions:
//
// 1. The origin of this software must not be misrepresented; you must not
//    claim that you wrote the original software. If you use this software
//    in a product, an acknowledgment in the product documentation would
//    be appreciated but is not required.
//
// 2. Altered source versions must be plainly marked as such, and must not
//    be misrepresented as being the original software.
//
// 3. This notice may not be removed or altered from any source
//    distribution.

package glfwwin

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

	for _, monitor := range _glfw.monitors {
		if len(monitor.originalRamp.Red) != 0 {
			monitor.platformSetGammaRamp(&monitor.originalRamp)
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
			terminate()
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
