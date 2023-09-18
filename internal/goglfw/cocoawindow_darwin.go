// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2023 The Ebitengine Authors

package goglfw

func (w *Window) platformCreateWindow(wndconfig *wndconfig, ctxconfig *ctxconfig, fbconfig *fbconfig) error {
	// cocoa_window.m:L898
	panic("goglfw: Window.platformCreateWindow is not implemented yet")
}

func (w *Window) platformDestroyWindow() error {
	// cocoa_window.m:L968
	panic("goglfw: Window.platformDestroyWindow is not implemented yet")
}

func (w *Window) platformSetWindowTitle(title string) error {
	// cocoa_window.m:L999
	panic("goglfw: Window.platformSetWindowTitle is not implemented yet")
}

func (w *Window) platformSetWindowIcon(images []*Image) error {
	// cocoa_window.m:L1010
	panic("goglfw: Window.platformSetWindowIcon is not implemented yet")
}

func (w *Window) platformGetWindowPos() (xpos, ypos int, err error) {
	// cocoa_window.m:L1016
	panic("goglfw: Window.platformGetWindowPos is not implemented yet")
}

func (w *Window) platformSetWindowPos(xpos, ypos int) error {
	// cocoa_window.m:L1031
	panic("goglfw: Window.platformSetWindowPos is not implemented yet")
}

func (w *Window) platformGetWindowSize() (width, height int, err error) {
	// cocoa_window.m:L1043
	panic("goglfw: Window.platformGetWindowSize is not implemented yet")
}

func (w *Window) platformSetWindowSize(width, height int) error {
	// cocoa_window.m:L1057
	panic("goglfw: Window.platformSetWindowSize is not implemented yet")
}

func (w *Window) platformSetWindowSizeLimits(minwidth, minheight, maxwidth, maxheight int) error {
	// cocoa_window.m:L1079
	panic("goglfw: Window.platformSetWindowSizeLimits is not implemented yet")
}

func (w *Window) platformSetWindowAspectRatio(numer, denom int) error {
	// cocoa_window.m:L1098
	panic("goglfw: Window.platformSetWindowAspectRatio is not implemented yet")
}

func (w *Window) platformGetFramebufferSize() (width, height int, err error) {
	// cocoa_window.m:L1108
	panic("goglfw: Window.platformGetFramebufferSize is not implemented yet")
}

func (w *Window) platformGetWindowFrameSize() (left, top, right, bottom int, err error) {
	// cocoa_window.m:L1123
	panic("goglfw: Window.platformGetWindowFrameSize is not implemented yet")
}

func (w *Window) platformGetWindowContentScale() (xscale, yscale float32, err error) {
	// cocoa_window.m:L1146
	panic("goglfw: Window.platformGetWindowContentScale is not implemented yet")
}

func (w *Window) platformIconifyWindow() {
	// cocoa_window.m:L1162
	panic("goglfw: Window.platformIconifyWindow is not implemented yet")
}

func (w *Window) platformRestoreWindow() {
	// cocoa_window.m:L1169
	panic("goglfw: Window.platformRestoreWindow is not implemented yet")
}

func (w *Window) platformMaximizeWindow() error {
	// cocoa_window.m:L1179
	panic("goglfw: Window.platformMaximizeWindow is not implemented yet")
}

func (w *Window) platformShowWindow() {
	// cocoa_window.m:L1187
	panic("goglfw: Window.platformShowWindow is not implemented yet")
}

func (w *Window) platformHideWindow() {
	// cocoa_window.m:L1194
	panic("goglfw: Window.platformHideWindow is not implemented yet")
}

func (w *Window) platformRequestWindowAttention() {
	// cocoa_window.m:L1201
	panic("goglfw: Window.platformRequestWindowAttention is not implemented yet")
}

func (w *Window) platformFocusWindow() error {
	// cocoa_window.m:L1208
	panic("goglfw: Window.platformFocusWindow is not implemented yet")
}

func (w *Window) platformSetWindowMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) error {
	// cocoa_window.m:L1220
	panic("goglfw: Window.platformSetWindowMonitor is not implemented yet")
}

func (w *Window) platformWindowFocused() bool {
	// cocoa_window.m:L1348
	panic("goglfw: platformWindowFocused is not implemented yet")
}

func (w *Window) platformWindowIconified() bool {
	// cocoa_window.m:L1355
	panic("goglfw: Window.platformWindowIconified is not implemented yet")
}

func (w *Window) platformWindowVisible() bool {
	// cocoa_window.m:L1362
	panic("goglfw: Window.platformWindowVisible is not implemented yet")
}

func (w *Window) platformWindowMaximized() bool {
	// cocoa_window.m:L1369
	panic("goglfw: Window.platformWindowMaximized is not implemented yet")
}

func (w *Window) platformWindowHovered() (bool, error) {
	// cocoa_window.m:L1381
	panic("goglfw: Window.platformWindowHovered is not implemented yet")
}

func (w *Window) platformFramebufferTransparent() bool {
	// cocoa_window.m:L1399
	panic("goglfw: Window.platformFramebufferTransparent is not implemented yet")
}

func (w *Window) platformSetWindowResizable(enabled bool) error {
	// cocoa_window.m:L1406
	panic("goglfw: Window.platformSetWindowResizable is not implemented yet")
}

func (w *Window) platformSetWindowDecorated(enabled bool) error {
	// cocoa_window.m:L1430
	panic("goglfw: Window.platformSetWindowDecorated is not implemented yet")
}

func (w *Window) platformSetWindowFloating(enabled bool) error {
	// cocoa_window.m:L1452
	panic("goglfw: Window.platformSetWindowFloating is not implemented yet")
}

func (w *Window) platformSetWindowMousePassthrough(enabled bool) error {
	panic("goglfw: Window.platformSetWindowMousePassthrough is not implemented yet")
}

func (w *Window) platformGetWindowOpacity() (float32, error) {
	// cocoa_window.m:L1462
	panic("goglfw: Window.platformGetWindowOpacity is not implemented yet")
}

func (w *Window) platformSetWindowOpacity(opacity float32) error {
	// cocoa_window.m:L1469
	panic("goglfw: Window.platformSetWindowOpacity is not implemented yet")
}

func (w *Window) platformSetRawMouseMotion(enabled bool) error {
	return nil
}

func platformRawMouseMotionSupported() bool {
	return false
}

func platformPollEvents() error {
	// cocoa_window.m:L1485
	panic("goglfw: platformPollEvents is not implemented yet")
}

func platformWaitEvents() error {
	// cocoa_window.m:L1507
	panic("goglfw: platformWaitEvents is not implemented yet")
}

func platformWaitEventsTimeout(timeout float64) error {
	// cocoa_window.m:L1528
	panic("goglfw: platformWaitEventsTimeout is not implemented yet")
}

func platformPostEmptyEvent() error {
	// cocoa_window.m:L1548
	panic("goglfw: platformPostEmptyEvent is not implemented yet")
}

func (w *Window) platformGetCursorPos() (xpos, ypos float64, err error) {
	// cocoa_window.m:L1569
	panic("goglfw: Window.platformGetCursorPos is not implemented yet")
}

func (w *Window) platformSetCursorPos(xpos, ypos float64) error {
	// cocoa_window.m:L1585
	panic("goglfw: Window.platformSetCursorPos is not implemented yet")
}

func (w *Window) platformSetCursorMode(mode int) error {
	// cocoa_window.m:L1621
	panic("goglfw: Window.platformSetMode is not implemented yet")
}

func platformGetScancodeName(scancode int) (string, error) {
	// cocoa_window.m:L1629
	panic("goglfw: platformGetScancodeName is not implemented yet")
}

func platformGetKeyScancode(key Key) int {
	// cocoa_window.m:L1678
	panic("goglfw: platformGetKeyScancode is not implemented yet")
}

func (c *Cursor) platformCreateStandardCursor(shape StandardCursor) error {
	// cocoa_window.m:L1727
	panic("goglfw: Cursor.platformCreateStandardCursor is not implemented yet")
}

func (c *Cursor) platformDestroyCursor() error {
	// cocoa_window.m:L1757
	panic("goglfw: Cursor.platformDestroyCursor is not implemented yet")
}

func (w *Window) platformSetCursor(cursor *Cursor) error {
	// cocoa_window.m:L1765
	panic("goglfw: Window.platformSetCursor is not implemented yet")
}

func platformSetClipboardString(str string) error {
	panic("goglfw: platformSetClipboardString is not implemented")
}

func platformGetClipboardString() (string, error) {
	panic("goglfw: platformGetClipboardString is not implemented")
}
