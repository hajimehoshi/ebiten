// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2021 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/winver"
)

func monitorCallback(handle _HMONITOR, dc _HDC, rect *_RECT, monitor *Monitor /* _LPARAM */) uintptr /* _BOOL */ {
	if mi, ok := _GetMonitorInfoW_Ex(handle); ok {
		if windows.UTF16ToString(mi.szDevice[:]) == monitor.platform.adapterName {
			monitor.platform.handle = handle
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

	monitor := &Monitor{
		name: name,
	}

	if adapter.StateFlags&_DISPLAY_DEVICE_MODESPRUNED != 0 {
		monitor.platform.modesPruned = true
	}

	monitor.platform.adapterName = adapterDeviceName
	if display != nil {
		monitor.platform.displayName = windows.UTF16ToString(display.DeviceName[:])
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
	if microsoftgdk.IsXbox() {
		return nil
	}

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
				if monitor != nil && monitor.platform.displayName == windows.UTF16ToString(display.DeviceName[:]) {
					disconnected[i] = nil
					err := _EnumDisplayMonitors(0, nil, monitorCallbackPtr, _LPARAM(unsafe.Pointer(_glfw.monitors[i])))
					if err != nil {
						return err
					}

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
				if monitor != nil && monitor.platform.displayName == windows.UTF16ToString(adapter.DeviceName[:]) {
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
	switch _ChangeDisplaySettingsExW(m.platform.adapterName, &dm, 0, _CDS_FULLSCREEN, nil) {
	case _DISP_CHANGE_SUCCESSFUL:
		m.platform.modeChanged = true
		return nil
	case _DISP_CHANGE_BADDUALVIEW:
		return errors.New("glfw: the system uses DualView at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_BADFLAGS:
		return errors.New("glfw: invalid flags at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_BADMODE:
		return errors.New("glfw: graphics mode not supported at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_BADPARAM:
		return errors.New("glfw: invalid parameter at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_FAILED:
		return errors.New("glfw: graphics mode failed at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_NOTUPDATED:
		return errors.New("glfw: failed to write to registry at Monitor.setVideoModeWin32")
	case _DISP_CHANGE_RESTART:
		return errors.New("glfw: computer restart required at Monitor.setVideoModeWin32")
	default:
		return errors.New("glfw: unknown error at Monitor.setVideoModeWin32")
	}
}

func (m *Monitor) restoreVideoModeWin32() {
	if m.platform.modeChanged {
		_ChangeDisplaySettingsExW(m.platform.adapterName, nil, 0, _CDS_FULLSCREEN, nil)
		m.platform.modeChanged = false
	}
}

func getMonitorContentScaleWin32(handle _HMONITOR) (xscale, yscale float32, err error) {
	var xdpi, ydpi uint32

	if winver.IsWindows8Point1OrGreater() {
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
	if microsoftgdk.IsXbox() {
		return 0, 0, true
	}

	dm, ok := _EnumDisplaySettingsExW(m.platform.adapterName, _ENUM_CURRENT_SETTINGS, _EDS_ROTATEDMODE)
	if !ok {
		return 0, 0, false
	}
	return int(dm.dmPosition.x), int(dm.dmPosition.y), true
}

func (m *Monitor) platformGetMonitorContentScale() (xscale, yscale float32, err error) {
	if microsoftgdk.IsXbox() {
		return 1, 1, nil
	}

	return getMonitorContentScaleWin32(m.platform.handle)
}

func (m *Monitor) platformGetMonitorWorkarea() (xpos, ypos, width, height int) {
	if microsoftgdk.IsXbox() {
		w, h := microsoftgdk.MonitorResolution()
		return 0, 0, w, h
	}

	mi, ok := _GetMonitorInfoW(m.platform.handle)
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
		dm, ok := _EnumDisplaySettingsW(m.platform.adapterName, modeIndex)
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

		if m.platform.modesPruned {
			// Skip modes not supported by the connected displays
			if _ChangeDisplaySettingsExW(m.platform.adapterName, &dm, 0, _CDS_TEST, nil) != _DISP_CHANGE_SUCCESSFUL {
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
	if microsoftgdk.IsXbox() {
		return m.modes[0]
	}

	dm, _ := _EnumDisplaySettingsW(m.platform.adapterName, _ENUM_CURRENT_SETTINGS)
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

func (m *Monitor) in32Adapter() (string, error) {
	if !_glfw.initialized {
		return "", NotInitialized
	}
	return m.platform.adapterName, nil
}

func (m *Monitor) win32Monitor() (string, error) {
	if !_glfw.initialized {
		return "", NotInitialized
	}
	return m.platform.displayName, nil
}
