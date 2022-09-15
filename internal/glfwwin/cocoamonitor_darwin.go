package glfwwin

func (m *Monitor) platformAppendVideoModes(monitors []*VidMode) ([]*VidMode, error) {
	panic("NOT IMPLEMENTED")
}

func (m *Monitor) GetCocoaMonitor() (uintptr, error) {
	panic("NOT IMPLEMENTED")
}

func (m *Monitor) platformGetMonitorPos() (xpos, ypos int, ok bool) {
	panic("NOT IMPLEMENTED")
}

func (m *Monitor) platformGetMonitorWorkarea() (xpos, ypos, width, height int) {
	panic("NOT IMPLEMENTED")
}

func (m *Monitor) platformGetMonitorContentScale() (xscale, yscale float32, err error) {
	panic("NOT IMPLEMENTED")
}

func (m *Monitor) platformGetVideoMode() *VidMode {
	panic("NOT IMPLEMENTED")
}
