package glfwwin

func platformGetKeyScancode(key Key) int {
	return _glfw.state.scancodes[key]
}

func (w *Window) platformCreateWindow(wndconfig *wndconfig, ctxconfig *ctxconfig, fbconfig *fbconfig) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformDestroyWindow() error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowTitle(title string) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) updateWindowStyles() error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowAspectRatio(numer, denom int) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetFramebufferSize() (width, height int, err error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetWindowOpacity() (float32, error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformIconifyWindow() {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformRestoreWindow() {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformMaximizeWindow() error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformShowWindow() {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformHideWindow() {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformWindowIconified() bool {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformWindowVisible() bool {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformWindowMaximized() bool {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformWindowHovered() (bool, error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformFramebufferTransparent() bool {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowResizable(enabled bool) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowDecorated(enabled bool) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowFloating(enabled bool) error {
	panic("NOT IMPLEMENTED")
}

func platformPollEvents() error {
	panic("NOT IMPLEMENTED")
}

func platformWaitEvents() error {
	panic("NOT IMPLEMENTED")
}

func platformWaitEventsTimeout(timeout float64) error {
	panic("NOT IMPLEMENTED")
}

func platformPostEmptyEvent() error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformRequestWindowAttention() {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformFocusWindow() error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowOpacity(opacity float32) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetWindowContentScale() (xscale, yscale float32, err error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowSizeLimits(minwidth, minheight, maxwidth, maxheight int) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowIcon(images []*Image) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetWindowPos() (xpos, ypos int, err error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetWindowSize() (width, height int, err error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetCursorPos(f float64, f2 float64) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowSize(width, height int) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetCursorPos() (xpos, ypos float64, err error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetCursorMode(mode int) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetCursor(cursor *Cursor) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformGetWindowFrameSize() (left, top, right, bottom int, err error) {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformSetWindowPos(xpos, ypos int) error {
	panic("NOT IMPLEMENTED")
}

func platformRawMouseMotionSupported() bool {
	panic("NOT IMPLEMENTED")
	return true
}

func (w *Window) platformSetRawMouseMotion(enabled bool) error {
	panic("NOT IMPLEMENTED")
}

func (w *Window) platformWindowFocused() bool {
	panic("NOT IMPLEMENTED")
}

func (c *Cursor) platformCreateCursor(image *Image, xhot, yhot int) error {
	panic("NOT IMPLEMENTED")
}

func (c *Cursor) platformCreateStandardCursor(shape StandardCursor) error {
	panic("NOT IMPLEMENTED")
}

func (c *Cursor) platformDestroyCursor() error {
	panic("NOT IMPLEMENTED")
}

func platformSetClipboardString(str string) error {
	panic("glfwwin: platformSetClipboardString is not implemented")
}

func platformGetClipboardString() (string, error) {
	panic("glfwwin: platformGetClipboardString is not implemented")
}

func (w *Window) GetCocoaWindow() (uintptr, error) {
	panic("NOT IMPLEMENTED")
}
