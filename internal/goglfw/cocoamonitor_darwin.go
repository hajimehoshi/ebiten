// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

package goglfw

func (m *Monitor) platformGetMonitorPos() (xpos, ypos int, ok bool) {
	// cocoa_monitor.m:L450
	panic("goglfw: Monitor.platformGetMonitorPos is not implemented yet")
}

func (m *Monitor) platformGetMonitorContentScale() (xscale, yscale float32, err error) {
	// cocoa_monitor.m:L464
	panic("goglfw: Monitor.platformGetMonitorContentScale is not implemented yet")
}

func (m *Monitor) platformGetMonitorWorkarea() (xpos, ypos, width, height int) {
	// cocoa_monitor.m:L486
	panic("goglfw: Monitor.platformGetMonitorWorkarea is not implemented yet")
}

func (m *Monitor) platformAppendVideoModes(monitors []*VidMode) ([]*VidMode, error) {
	// cocoa_monitor.m:L512
	panic("goglfw: Monitor.platformAppendVideoModes is not implemented yet")
}

func (m *Monitor) platformGetVideoMode() *VidMode {
	// cocoa_monitor.m:L552
	panic("goglfw: Monitor.platformGetVideoMode is not implemented yet")
}
