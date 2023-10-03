// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"sort"
)

func abs(x int) uint {
	if x < 0 {
		return uint(-x)
	}
	return uint(x)
}

func (v *VidMode) equals(other *VidMode) bool {
	if v.RedBits+v.GreenBits+v.BlueBits != other.RedBits+other.GreenBits+other.BlueBits {
		return false
	}

	if v.Width != other.Width {
		return false
	}

	if v.Height != other.Height {
		return false
	}

	if v.RefreshRate != other.RefreshRate {
		return false
	}

	return true
}

func (m *Monitor) refreshVideoModes() error {
	m.modes = m.modes[:0]
	modes, err := m.platformAppendVideoModes(m.modes)
	if err != nil {
		return err
	}
	sort.Slice(modes, func(i, j int) bool {
		a := modes[i]
		b := modes[j]
		abpp := a.RedBits + a.GreenBits + a.BlueBits
		bbpp := b.RedBits + b.GreenBits + b.BlueBits
		if abpp != bbpp {
			return abpp < bbpp
		}
		aarea := a.Width * a.Height
		barea := b.Width * b.Height
		if aarea != barea {
			return aarea < barea
		}
		if a.Width != b.Width {
			return a.Width < b.Width
		}
		return a.RefreshRate < b.RefreshRate
	})
	m.modes = modes
	return nil
}

func inputMonitor(monitor *Monitor, action PeripheralEvent, placement int) error {
	switch action {
	case Connected:
		switch placement {
		case _GLFW_INSERT_FIRST:
			_glfw.monitors = append(_glfw.monitors, nil)
			copy(_glfw.monitors[1:], _glfw.monitors)
			_glfw.monitors[0] = monitor
		case _GLFW_INSERT_LAST:
			_glfw.monitors = append(_glfw.monitors, monitor)
		}
	case Disconnected:
		for _, window := range _glfw.windows {
			if window.monitor == monitor {
				width, height, err := window.platformGetWindowSize()
				if err != nil {
					return err
				}
				if err := window.platformSetWindowMonitor(nil, 0, 0, width, height, 0); err != nil {
					return err
				}
				xoff, yoff, _, _, err := window.platformGetWindowFrameSize()
				if err != nil {
					return err
				}
				if err := window.platformSetWindowPos(xoff, yoff); err != nil {
					return err
				}
			}
		}
		for i, m := range _glfw.monitors {
			if m == monitor {
				copy(_glfw.monitors[i:], _glfw.monitors[i+1:])
				_glfw.monitors[len(_glfw.monitors)-1] = nil
				_glfw.monitors = _glfw.monitors[:len(_glfw.monitors)-1]
				break
			}
		}
	}

	if _glfw.callbacks.monitor != nil {
		_glfw.callbacks.monitor(monitor, action)
	}

	return nil
}

func (m *Monitor) inputMonitorWindow(window *Window) {
	m.window = window
}

func (m *Monitor) chooseVideoMode(desired *VidMode) (*VidMode, error) {
	if err := m.refreshVideoModes(); err != nil {
		return nil, err
	}

	// math.MaxUint was added at Go 1.17. See https://github.com/golang/go/issues/28538
	const (
		intSize = 32 << (^uint(0) >> 63)
		maxUint = 1<<intSize - 1
	)

	var (
		leastColorDiff uint = maxUint
		leastSizeDiff  uint = maxUint
		leastRateDiff  uint = maxUint
	)

	var closest *VidMode
	for _, v := range m.modes {
		var colorDiff uint
		if desired.RedBits != DontCare {
			colorDiff += abs(v.RedBits - desired.RedBits)
		}
		if desired.GreenBits != DontCare {
			colorDiff += abs(v.GreenBits - desired.GreenBits)
		}
		if desired.BlueBits != DontCare {
			colorDiff += abs(v.BlueBits - desired.BlueBits)
		}

		sizeDiff := abs((v.Width-desired.Width)*(v.Width-desired.Width) +
			(v.Height-desired.Height)*(v.Height-desired.Height))

		var rateDiff uint
		if desired.RefreshRate != DontCare {
			rateDiff = abs(v.RefreshRate - desired.RefreshRate)
		} else {
			rateDiff = maxUint - uint(v.RefreshRate)
		}

		if colorDiff < leastColorDiff ||
			colorDiff == leastColorDiff && sizeDiff < leastSizeDiff ||
			colorDiff == leastColorDiff && sizeDiff == leastSizeDiff && rateDiff < leastRateDiff {
			closest = v
			leastColorDiff = colorDiff
			leastSizeDiff = sizeDiff
			leastRateDiff = rateDiff
		}
	}

	return closest, nil
}

func splitBPP(bpp int) (red, green, blue int) {
	// We assume that by 32 the user really meant 24
	if bpp == 32 {
		bpp = 24
	}

	// Convert "bits per pixel" to red, green & blue sizes
	red = bpp / 3
	green = bpp / 3
	blue = bpp / 3
	delta := bpp - (red * 3)
	if delta >= 1 {
		green++
	}
	if delta == 2 {
		red++
	}
	return
}

// GLFW public APIs

func GetMonitors() ([]*Monitor, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	return _glfw.monitors, nil
}

func GetPrimaryMonitor() (*Monitor, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	if len(_glfw.monitors) == 0 {
		return nil, nil
	}
	return _glfw.monitors[0], nil
}

func (m *Monitor) GetPos() (xpos, ypos int, err error) {
	if !_glfw.initialized {
		return 0, 0, NotInitialized
	}
	xpos, ypos, ok := m.platformGetMonitorPos()
	if !ok {
		return 0, 0, nil
	}
	return xpos, ypos, nil
}

func (m *Monitor) GetWorkarea() (xpos, ypos, width, height int, err error) {
	if !_glfw.initialized {
		return 0, 0, 0, 0, NotInitialized
	}
	xpos, ypos, width, height = m.platformGetMonitorWorkarea()
	return
}

// GetPhysicalSize is not implemented.

func (m *Monitor) GetContentScale() (xscale, yscale float32, err error) {
	if !_glfw.initialized {
		return 0, 0, NotInitialized
	}
	xscale, yscale, err = m.platformGetMonitorContentScale()
	return
}

func (m *Monitor) GetName() (string, error) {
	if !_glfw.initialized {
		return "", NotInitialized
	}
	return m.name, nil
}

// SetUserPointer is not implemented.
// GetUserPointer is not implemented.

func SetMonitorCallback(cbfun MonitorCallback) (MonitorCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := _glfw.callbacks.monitor
	_glfw.callbacks.monitor = cbfun
	return old, nil
}

func (m *Monitor) GetVideoModes() ([]*VidMode, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	return m.modes, nil
}

func (m *Monitor) GetVideoMode() (*VidMode, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	return m.platformGetVideoMode(), nil
}

// SetGamma is not implemented.
// GetGammaRamp is not implemented.
// SetGammaRamp is not implemented.
