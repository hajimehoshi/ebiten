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

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

func monitorCallback(handle _HMONITOR, dc _HDC, rect *_RECT, data _LPARAM) uintptr /* _BOOL */ {
	if mi, ok := _GetMonitorInfoW_Ex(handle); ok {
		monitor := (*Monitor)(unsafe.Pointer(data))
		if windows.UTF16ToString(mi.szDevice[:]) == monitor.win32.adapterName {
			monitor.win32.handle = handle
		}
	}
	return 1
}

var monitorCallbackPtr = windows.NewCallbackCDecl(monitorCallback)

func createMonitor(adapter *_DISPLAY_DEVICEW, display *_DISPLAY_DEVICEW) (*Monitor, error) {
	var name string
	if display != nil {
		name = windows.UTF16ToString(display.DeviceString[:])
	} else {
		name = windows.UTF16ToString(adapter.DeviceString[:])
	}
	if name == "" {
		return nil, nil
	}

	adapterDeviceName := windows.UTF16ToString(adapter.DeviceName[:])
	dm, ok := _EnumDisplaySettingsW(adapterDeviceName, _ENUM_CURRENT_SETTINGS)
	if !ok {
		return nil, nil
	}

	var widthMM, heightMM int
	dc, err := _CreateDCW("DISPLAY", adapterDeviceName, "", nil)
	if err != nil {
		return nil, err
	}
	if _IsWindows8Point1OrGreater() {
		widthMM = int(_GetDeviceCaps(dc, _HORZSIZE))
		heightMM = int(_GetDeviceCaps(dc, _VERTSIZE))
	} else {
		widthMM = int(float64(dm.dmPelsWidth) * 25.4 / float64(_GetDeviceCaps(dc, _LOGPIXELSX)))
		heightMM = int(float64(dm.dmPelsHeight) * 25.4 / float64(_GetDeviceCaps(dc, _LOGPIXELSY)))
	}
	if err := _DeleteDC(dc); err != nil {
		return nil, err
	}

	monitor := &Monitor{
		name:     name,
		widthMM:  widthMM,
		heightMM: heightMM,
	}

	if adapter.StateFlags&_DISPLAY_DEVICE_MODESPRUNED != 0 {
		monitor.win32.modesPruned = true
	}

	monitor.win32.adapterName = adapterDeviceName
	if display != nil {
		monitor.win32.displayName = windows.UTF16ToString(display.DeviceName[:])
	}

	rect := _RECT{
		left:   dm.dmPosition.x,
		top:    dm.dmPosition.y,
		right:  dm.dmPosition.x + int32(dm.dmPelsWidth),
		bottom: dm.dmPosition.y + int32(dm.dmPelsHeight),
	}
	if err := _EnumDisplayMonitors(0, &rect, monitorCallbackPtr, _LPARAM(unsafe.Pointer(monitor))); err != nil {
		return nil, err
	}
	return monitor, nil
}

func pollMonitorsWin32() error {
	disconnected := make([]*Monitor, len(_glfw.monitors))
	copy(disconnected, _glfw.monitors)

adapterLoop:
	for adapterIndex := uint32(0); ; adapterIndex++ {
		adapter, ok := _EnumDisplayDevicesW("", adapterIndex, 0)
		if !ok {
			break
		}
		if adapter.StateFlags&_DISPLAY_DEVICE_ACTIVE == 0 {
			continue
		}

		typ := _GLFW_INSERT_LAST
		if adapter.StateFlags&_DISPLAY_DEVICE_PRIMARY_DEVICE != 0 {
			typ = _GLFW_INSERT_FIRST
		}

		var found bool
	displayLoop:
		for displayIndex := uint32(0); ; displayIndex++ {
			display, ok := _EnumDisplayDevicesW(windows.UTF16ToString(adapter.DeviceName[:]), displayIndex, 0)
			if !ok {
				break
			}
			found = true
			if display.StateFlags&_DISPLAY_DEVICE_ACTIVE == 0 {
				continue
			}

			for i, monitor := range disconnected {
				if monitor != nil && monitor.win32.displayName == windows.UTF16ToString(display.DeviceName[:]) {
					disconnected[i] = nil
					_EnumDisplayMonitors(0, nil, monitorCallbackPtr, _LPARAM(unsafe.Pointer(_glfw.monitors[i])))
					continue displayLoop
				}
			}

			monitor, err := createMonitor(&adapter, &display)
			if err != nil {
				return err
			}
			if monitor == nil {
				return nil
			}

			if err := inputMonitor(monitor, Connected, typ); err != nil {
				return err
			}
			typ = _GLFW_INSERT_LAST
		}

		// HACK: If an active adapter does not have any display devices
		//       (as sometimes happens), add it directly as a monitor
		if !found {
			for i, monitor := range disconnected {
				if monitor != nil && monitor.win32.displayName == windows.UTF16ToString(adapter.DeviceName[:]) {
					disconnected[i] = nil
					continue adapterLoop
				}
			}

			monitor, err := createMonitor(&adapter, nil)
			if err != nil {
				return err
			}
			if monitor == nil {
				return nil
			}

			if err := inputMonitor(monitor, Connected, typ); err != nil {
				return err
			}
		}
	}

	for _, monitor := range disconnected {
		if monitor != nil {
			if err := inputMonitor(monitor, Disconnected, 0); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *Monitor) setVideoModeWin32(desired *VidMode) error {
	best, err := m.chooseVideoMode(desired)
	if err != nil {
		return err
	}
	current := m.platformGetVideoMode()
	if best.equals(current) {
		return nil
	}

	dm := _DEVMODEW{
		dmFields:           _DM_PELSWIDTH | _DM_PELSHEIGHT | _DM_BITSPERPEL | _DM_DISPLAYFREQUENCY,
		dmPelsWidth:        uint32(best.Width),
		dmPelsHeight:       uint32(best.Height),
		dmBitsPerPel:       uint32(best.RedBits + best.GreenBits + best.BlueBits),
		dmDisplayFrequency: uint32(best.RefreshRate),
	}
	dm.dmSize = uint16(unsafe.Sizeof(dm))
	if dm.dmBitsPerPel < 15 || dm.dmBitsPerPel >= 24 {
		dm.dmBitsPerPel = 32
	}
	switch _ChangeDisplaySettingsExW(m.win32.adapterName, &dm, 0, _CDS_FULLSCREEN, nil) {
	case _DISP_CHANGE_SUCCESSFUL:
		m.win32.modeChanged = true
		return nil
	case _DISP_CHANGE_BADDUALVIEW:
		return errors.New("glfwwin: the system uses DualView at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_BADFLAGS:
		return errors.New("glfwwin: invalid flags at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_BADMODE:
		return errors.New("glfwwin: graphics mode not supported at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_BADPARAM:
		return errors.New("glfwwin: invalid parameter at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_FAILED:
		return errors.New("glfwwin: graphics mode failed at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_NOTUPDATED:
		return errors.New("glfwwin: failed to write to registry at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_RESTART:
		return errors.New("glfwwin: computer restart required at Monitor.setVideoModeWin32")
	default:
		return errors.New("glfwwin: unknown error at Monitor.setVideoModeWin32")
	}
}

func (m *Monitor) restoreVideoModeWin32() {
	if m.win32.modeChanged {
		_ChangeDisplaySettingsExW(m.win32.adapterName, nil, 0, _CDS_FULLSCREEN, nil)
		m.win32.modeChanged = false
	}
}

func getMonitorContentScaleWin32(handle _HMONITOR) (xscale, yscale float32, err error) {
	var xdpi, ydpi uint32

	if _IsWindows8Point1OrGreater() {
		var err error
		xdpi, ydpi, err = _GetDpiForMonitor(handle, _MDT_EFFECTIVE_DPI)
		if err != nil {
			return 0, 0, err
		}
	} else {
		dc, err := _GetDC(0)
		if err != nil {
			return 0, 0, err
		}
		defer _ReleaseDC(0, dc)

		xdpi = uint32(_GetDeviceCaps(dc, _LOGPIXELSX))
		ydpi = uint32(_GetDeviceCaps(dc, _LOGPIXELSY))
	}

	xscale = float32(xdpi) / _USER_DEFAULT_SCREEN_DPI
	yscale = float32(ydpi) / _USER_DEFAULT_SCREEN_DPI
	return
}

func (m *Monitor) platformGetMonitorPos() (xpos, ypos int, ok bool) {
	dm, ok := _EnumDisplaySettingsExW(m.win32.adapterName, _ENUM_CURRENT_SETTINGS, _EDS_ROTATEDMODE)
	if !ok {
		return 0, 0, false
	}
	return int(dm.dmPosition.x), int(dm.dmPosition.y), true
}

func (m *Monitor) platformGetMonitorContentScale() (xscale, yscale float32, err error) {
	return getMonitorContentScaleWin32(m.win32.handle)
}

func (m *Monitor) platformGetMonitorWorkarea() (xpos, ypos, width, height int) {
	mi, ok := _GetMonitorInfoW(m.win32.handle)
	if !ok {
		return 0, 0, 0, 0
	}
	xpos = int(mi.rcWork.left)
	ypos = int(mi.rcWork.top)
	width = int(mi.rcWork.right - mi.rcWork.left)
	height = int(mi.rcWork.bottom - mi.rcWork.top)
	return
}

func (m *Monitor) platformAppendVideoModes(monitors []*VidMode) ([]*VidMode, error) {
	origLen := len(monitors)
loop:
	for modeIndex := uint32(0); ; modeIndex++ {
		dm, ok := _EnumDisplaySettingsW(m.win32.adapterName, modeIndex)
		if !ok {
			break
		}

		// Skip modes with less than 15 BPP
		if dm.dmBitsPerPel < 15 {
			continue
		}

		r, g, b := splitBPP(int(dm.dmBitsPerPel))
		mode := &VidMode{
			Width:       int(dm.dmPelsWidth),
			Height:      int(dm.dmPelsHeight),
			RefreshRate: int(dm.dmDisplayFrequency),
			RedBits:     r,
			GreenBits:   g,
			BlueBits:    b,
		}

		// Skip duplicate modes
		for _, m := range monitors[origLen:] {
			if m.equals(mode) {
				continue loop
			}
		}

		if m.win32.modesPruned {
			// Skip modes not supported by the connected displays
			if _ChangeDisplaySettingsExW(m.win32.adapterName, &dm, 0, _CDS_TEST, nil) != _DISP_CHANGE_SUCCESSFUL {
				continue
			}
		}

		monitors = append(monitors, mode)
	}

	if len(monitors) == origLen {
		// HACK: Report the current mode if no valid modes were found
		monitors = append(monitors, m.platformGetVideoMode())
	}

	return monitors, nil
}

func (m *Monitor) platformGetVideoMode() *VidMode {
	dm, _ := _EnumDisplaySettingsW(m.win32.adapterName, _ENUM_CURRENT_SETTINGS)
	r, g, b := splitBPP(int(dm.dmBitsPerPel))
	return &VidMode{
		Width:       int(dm.dmPelsWidth),
		Height:      int(dm.dmPelsHeight),
		RefreshRate: int(dm.dmDisplayFrequency),
		RedBits:     r,
		GreenBits:   g,
		BlueBits:    b,
	}
}

func (m *Monitor) platformGetGammaRamp(ramp *GammaRamp) {
	panic("glfwwin: platformGetGammaRamp is not implemented")
}

func (m *Monitor) platformSetGammaRamp(ramp *GammaRamp) {
	panic("glfwwin: platformSetGammaRamp is not implemented")
}

func (m *Monitor) in32Adapter() (string, error) {
	if !_glfw.initialized {
		return "", NotInitialized
	}
	return m.win32.adapterName, nil
}

func (m *Monitor) win32Monitor() (string, error) {
	if !_glfw.initialized {
		return "", NotInitialized
	}
	return m.win32.displayName, nil
}
