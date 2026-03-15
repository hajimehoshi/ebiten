// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2012 The glfw3-go Authors
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

package glfw

import (
	"fmt"
	"unsafe"
)

func (m *Monitor) GetCocoaMonitor() (uintptr, error) {
	return uintptr(m.platform.displayID), nil
}

func (w *Window) GetCocoaWindow() (uintptr, error) {
	return uintptr(w.platform.object), nil
}

func (w *Window) GetNSGLContext() (unsafe.Pointer, error) {
	if w.context.source != NativeContextAPI {
		return nil, fmt.Errorf("glfw: window has no NSGL context: %w", NoWindowContext)
	}
	return unsafe.Pointer(w.context.platform.object), nil
}
