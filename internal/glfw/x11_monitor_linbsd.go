// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

//go:build freebsd || linux || netbsd

package glfw

import (
	"math"
	"unsafe"
)

// modeIsGood reports whether the display mode should be included in
// enumeration.
func modeIsGood(mi *_XRRModeInfo) bool {
	return mi.ModeFlags&_RR_Interlace == 0
}

// calculateRefreshRate calculates the refresh rate, in Hz, from the
// specified RandR mode info.
func calculateRefreshRate(mi *_XRRModeInfo) int {
	if mi.HTotal != 0 && mi.VTotal != 0 {
		return int(math.Round(float64(mi.DotClock) / (float64(mi.HTotal) * float64(mi.VTotal))))
	}
	return 0
}

// getModeInfo returns the mode info for a RandR mode XID.
func getModeInfo(sr *_XRRScreenResources, id _RRMode) *_XRRModeInfo {
	modes := unsafe.Slice((*_XRRModeInfo)(unsafe.Pointer(sr.Modes)), int(sr.Nmode))
	for i := range modes {
		if modes[i].ID == id {
			return &modes[i]
		}
	}
	return nil
}

// vidmodeFromModeInfo converts RandR mode info to a GLFW video mode.
func vidmodeFromModeInfo(mi *_XRRModeInfo, ci *_XRRCrtcInfo) *VidMode {
	var mode VidMode

	if ci.Rotation == _RR_Rotate_90 || ci.Rotation == _RR_Rotate_270 {
		mode.Width = int(mi.Height)
		mode.Height = int(mi.Width)
	} else {
		mode.Width = int(mi.Width)
		mode.Height = int(mi.Height)
	}

	mode.RefreshRate = calculateRefreshRate(mi)

	mode.RedBits, mode.GreenBits, mode.BlueBits =
		splitBPP(int(xDefaultDepth(_glfw.platformWindow.display, int32(_glfw.platformWindow.screen))))

	return &mode
}

// getCrtcInfoX11 returns the RandR CRTC info for the given CRTC, or 0 when
// the CRTC no longer exists.
//
// The X error handler must be installed for this request, as the default
// handler exits the process for an XID that the X server has already
// destroyed (#3094).
func getCrtcInfoX11(srPtr uintptr, crtc _RRCrtc) uintptr {
	grabErrorHandlerX11()
	defer releaseErrorHandlerX11()
	return _glfw.platformWindow.randr.GetCrtcInfo(_glfw.platformWindow.display, srPtr, crtc)
}

// getOutputInfoX11 returns the RandR output info for the given output, or 0
// when the output no longer exists.
//
// The X error handler must be installed for this request, as the default
// handler exits the process for an XID that the X server has already
// destroyed (#3094).
func getOutputInfoX11(srPtr uintptr, output _RROutput) uintptr {
	grabErrorHandlerX11()
	defer releaseErrorHandlerX11()
	return _glfw.platformWindow.randr.GetOutputInfo(_glfw.platformWindow.display, srPtr, output)
}

// pollMonitorsX11 polls for changes in the set of connected monitors.
func pollMonitorsX11() error {
	if !_glfw.platformWindow.randr.available || _glfw.platformWindow.randr.monitorBroken {
		return inputMonitor(&Monitor{name: "Display"}, Connected, _GLFW_INSERT_FIRST)
	}

	display := _glfw.platformWindow.display
	randr := &_glfw.platformWindow.randr

	srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
	defer randr.FreeScreenResources(srPtr)
	sr := (*_XRRScreenResources)(unsafe.Pointer(srPtr))

	primary := randr.GetOutputPrimary(display, _glfw.platformWindow.root)

	var screens []_XineramaScreenInfo
	if _glfw.platformWindow.xinerama.available {
		var screenCount int32
		screensPtr := _glfw.platformWindow.xinerama.QueryScreens(display, &screenCount)
		if screensPtr != 0 {
			defer xFree(screensPtr)
			screens = unsafe.Slice((*_XineramaScreenInfo)(unsafe.Pointer(screensPtr)), int(screenCount))
		}
	}

	disconnected := make([]*Monitor, len(_glfw.monitors))
	copy(disconnected, _glfw.monitors)

	type connection struct {
		monitor   *Monitor
		placement int
	}
	var connections []connection

	outputs := unsafe.Slice((*_RROutput)(unsafe.Pointer(sr.Outputs)), int(sr.Noutput))
	for i := range outputs {
		oiPtr := getOutputInfoX11(srPtr, outputs[i])
		if oiPtr == 0 {
			continue
		}
		oi := (*_XRROutputInfo)(unsafe.Pointer(oiPtr))

		if oi.Connection != _RR_Connected || oi.Crtc == _None {
			randr.FreeOutputInfo(oiPtr)
			continue
		}

		ciPtr := getCrtcInfoX11(srPtr, oi.Crtc)
		if ciPtr == 0 {
			randr.FreeOutputInfo(oiPtr)
			continue
		}
		ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))

		// An output that is already known keeps its monitor, but its CRTC can
		// have been reassigned since the last poll.
		var monitor *Monitor
		for j := range disconnected {
			if disconnected[j] != nil && disconnected[j].platform.output == outputs[i] {
				monitor = disconnected[j]
				disconnected[j] = nil
				break
			}
		}
		known := monitor != nil
		if !known {
			monitor = &Monitor{name: goString(oi.Name)}
			monitor.platform.output = outputs[i]
		}
		monitor.platform.crtc = oi.Crtc

		monitor.platform.index = 0
		for j := range screens {
			if int32(screens[j].XOrg) == ci.X &&
				int32(screens[j].YOrg) == ci.Y &&
				int32(screens[j].Width) == int32(ci.Width) &&
				int32(screens[j].Height) == int32(ci.Height) {
				monitor.platform.index = j
				break
			}
		}

		randr.FreeOutputInfo(oiPtr)
		randr.FreeCrtcInfo(ciPtr)

		if known {
			continue
		}

		placement := _GLFW_INSERT_LAST
		if monitor.platform.output == primary {
			placement = _GLFW_INSERT_FIRST
		}
		connections = append(connections, connection{
			monitor:   monitor,
			placement: placement,
		})
	}

	// Report the disconnections before the connections. A monitor callback
	// enumerates the monitors, and must not reach a monitor whose CRTC the X
	// server has already destroyed (#3094).
	for _, monitor := range disconnected {
		if monitor != nil {
			if err := inputMonitor(monitor, Disconnected, 0); err != nil {
				return err
			}
		}
	}

	for _, c := range connections {
		if err := inputMonitor(c.monitor, Connected, c.placement); err != nil {
			return err
		}
	}

	return nil
}

// setVideoModeX11 sets the current video mode for the specified monitor.
func setVideoModeX11(monitor *Monitor, desired *VidMode) error {
	if !_glfw.platformWindow.randr.available || _glfw.platformWindow.randr.monitorBroken {
		return nil
	}

	display := _glfw.platformWindow.display
	randr := &_glfw.platformWindow.randr

	best, err := monitor.chooseVideoMode(desired)
	if err != nil {
		return err
	}
	if best == nil {
		return nil
	}
	current := monitor.platformGetVideoMode()
	if current != nil && current.equals(best) {
		return nil
	}

	srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
	defer randr.FreeScreenResources(srPtr)
	sr := (*_XRRScreenResources)(unsafe.Pointer(srPtr))

	ciPtr := getCrtcInfoX11(srPtr, monitor.platform.crtc)
	if ciPtr == 0 {
		return nil
	}
	defer randr.FreeCrtcInfo(ciPtr)
	ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))

	oiPtr := getOutputInfoX11(srPtr, monitor.platform.output)
	if oiPtr == 0 {
		return nil
	}
	defer randr.FreeOutputInfo(oiPtr)
	oi := (*_XRROutputInfo)(unsafe.Pointer(oiPtr))

	native := _RRMode(_None)

	modes := unsafe.Slice((*_RRMode)(unsafe.Pointer(oi.Modes)), int(oi.Nmode))
	for i := range modes {
		mi := getModeInfo(sr, modes[i])
		if mi == nil || !modeIsGood(mi) {
			continue
		}

		mode := vidmodeFromModeInfo(mi, ci)
		if best.equals(mode) {
			native = mi.ID
			break
		}
	}

	if native != _None {
		if monitor.platform.oldMode == _None {
			monitor.platform.oldMode = ci.Mode
		}

		outputs := ci.Outputs
		randr.SetCrtcConfig(display,
			srPtr, monitor.platform.crtc,
			_CurrentTime,
			ci.X, ci.Y,
			native,
			ci.Rotation,
			(*_RROutput)(unsafe.Pointer(outputs)),
			ci.Noutput)
	}

	return nil
}

// restoreVideoModeX11 restores the saved (original) video mode for the
// specified monitor.
func restoreVideoModeX11(monitor *Monitor) {
	if !_glfw.platformWindow.randr.available || _glfw.platformWindow.randr.monitorBroken {
		return
	}

	if monitor.platform.oldMode == _None {
		return
	}

	display := _glfw.platformWindow.display
	randr := &_glfw.platformWindow.randr

	srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
	defer randr.FreeScreenResources(srPtr)

	ciPtr := getCrtcInfoX11(srPtr, monitor.platform.crtc)
	if ciPtr == 0 {
		return
	}
	defer randr.FreeCrtcInfo(ciPtr)
	ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))

	randr.SetCrtcConfig(display,
		srPtr, monitor.platform.crtc,
		_CurrentTime,
		ci.X, ci.Y,
		monitor.platform.oldMode,
		ci.Rotation,
		(*_RROutput)(unsafe.Pointer(ci.Outputs)),
		ci.Noutput)

	monitor.platform.oldMode = _None
}

func (m *Monitor) platformGetMonitorPos() (xpos, ypos int, ok bool) {
	if !_glfw.platformWindow.randr.available || _glfw.platformWindow.randr.monitorBroken {
		return 0, 0, false
	}

	display := _glfw.platformWindow.display
	randr := &_glfw.platformWindow.randr

	srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
	defer randr.FreeScreenResources(srPtr)

	ciPtr := getCrtcInfoX11(srPtr, m.platform.crtc)
	if ciPtr == 0 {
		return 0, 0, false
	}
	defer randr.FreeCrtcInfo(ciPtr)

	ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))
	return int(ci.X), int(ci.Y), true
}

func (m *Monitor) platformGetMonitorContentScale() (xscale, yscale float32, err error) {
	return _glfw.platformWindow.contentScaleX, _glfw.platformWindow.contentScaleY, nil
}

func (m *Monitor) platformGetMonitorWorkarea() (xpos, ypos, width, height int) {
	display := _glfw.platformWindow.display

	var areaX, areaY, areaWidth, areaHeight int32
	var areaFromCrtc bool

	if _glfw.platformWindow.randr.available && !_glfw.platformWindow.randr.monitorBroken {
		randr := &_glfw.platformWindow.randr

		srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
		defer randr.FreeScreenResources(srPtr)
		sr := (*_XRRScreenResources)(unsafe.Pointer(srPtr))

		if ciPtr := getCrtcInfoX11(srPtr, m.platform.crtc); ciPtr != 0 {
			defer randr.FreeCrtcInfo(ciPtr)
			ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))

			// mi can be nil if the monitor has been disconnected.
			if mi := getModeInfo(sr, ci.Mode); mi != nil {
				areaX = ci.X
				areaY = ci.Y

				if ci.Rotation == _RR_Rotate_90 || ci.Rotation == _RR_Rotate_270 {
					areaWidth = int32(mi.Height)
					areaHeight = int32(mi.Width)
				} else {
					areaWidth = int32(mi.Width)
					areaHeight = int32(mi.Height)
				}
				areaFromCrtc = true
			}
		}
	}

	if !areaFromCrtc {
		areaWidth = xDisplayWidth(display, int32(_glfw.platformWindow.screen))
		areaHeight = xDisplayHeight(display, int32(_glfw.platformWindow.screen))
	}

	if _glfw.platformWindow.NET_WORKAREA != 0 && _glfw.platformWindow.NET_CURRENT_DESKTOP != 0 {
		var extentsPtr, desktopPtr uintptr
		extentCount := getWindowPropertyX11(_glfw.platformWindow.root,
			_glfw.platformWindow.NET_WORKAREA,
			_XA_CARDINAL,
			&extentsPtr)
		if extentsPtr != 0 {
			defer xFree(extentsPtr)
		}

		desktopCount := getWindowPropertyX11(_glfw.platformWindow.root,
			_glfw.platformWindow.NET_CURRENT_DESKTOP,
			_XA_CARDINAL,
			&desktopPtr)
		if desktopPtr != 0 {
			defer xFree(desktopPtr)
		}

		if desktopCount > 0 {
			desktop := *(*_Culong)(unsafe.Pointer(desktopPtr))
			if extentCount >= 4 && desktop < extentCount/4 {
				extents := unsafe.Slice((*_Culong)(unsafe.Pointer(extentsPtr)), int(extentCount))

				globalX := int32(extents[desktop*4+0])
				globalY := int32(extents[desktop*4+1])
				globalWidth := int32(extents[desktop*4+2])
				globalHeight := int32(extents[desktop*4+3])

				if areaX < globalX {
					areaWidth -= globalX - areaX
					areaX = globalX
				}

				if areaY < globalY {
					areaHeight -= globalY - areaY
					areaY = globalY
				}

				if areaX+areaWidth > globalX+globalWidth {
					areaWidth = globalX - areaX + globalWidth
				}
				if areaY+areaHeight > globalY+globalHeight {
					areaHeight = globalY - areaY + globalHeight
				}
			}
		}
	}

	return int(areaX), int(areaY), int(areaWidth), int(areaHeight)
}

func (m *Monitor) platformAppendVideoModes(monitors []*VidMode) ([]*VidMode, error) {
	result := monitors

	if !_glfw.platformWindow.randr.available || _glfw.platformWindow.randr.monitorBroken {
		return append(result, m.platformGetVideoMode()), nil
	}

	display := _glfw.platformWindow.display
	randr := &_glfw.platformWindow.randr

	srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
	defer randr.FreeScreenResources(srPtr)
	sr := (*_XRRScreenResources)(unsafe.Pointer(srPtr))

	// The CRTC and the output are gone when the monitor has been disconnected.
	ciPtr := getCrtcInfoX11(srPtr, m.platform.crtc)
	if ciPtr == 0 {
		return result, nil
	}
	defer randr.FreeCrtcInfo(ciPtr)
	ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))

	oiPtr := getOutputInfoX11(srPtr, m.platform.output)
	if oiPtr == 0 {
		return result, nil
	}
	defer randr.FreeOutputInfo(oiPtr)
	oi := (*_XRROutputInfo)(unsafe.Pointer(oiPtr))

	modes := unsafe.Slice((*_RRMode)(unsafe.Pointer(oi.Modes)), int(oi.Nmode))
	for i := range modes {
		mi := getModeInfo(sr, modes[i])
		if mi == nil || !modeIsGood(mi) {
			continue
		}

		mode := vidmodeFromModeInfo(mi, ci)

		// Skip duplicate modes
		var duplicate bool
		for _, existing := range result {
			if existing.equals(mode) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}

		result = append(result, mode)
	}

	return result, nil
}

func (m *Monitor) platformGetVideoMode() *VidMode {
	display := _glfw.platformWindow.display

	var mode VidMode

	if !_glfw.platformWindow.randr.available || _glfw.platformWindow.randr.monitorBroken {
		mode.Width = int(xDisplayWidth(display, int32(_glfw.platformWindow.screen)))
		mode.Height = int(xDisplayHeight(display, int32(_glfw.platformWindow.screen)))
		mode.RefreshRate = 0

		mode.RedBits, mode.GreenBits, mode.BlueBits =
			splitBPP(int(xDefaultDepth(display, int32(_glfw.platformWindow.screen))))

		return &mode
	}

	randr := &_glfw.platformWindow.randr

	srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
	defer randr.FreeScreenResources(srPtr)
	sr := (*_XRRScreenResources)(unsafe.Pointer(srPtr))

	ciPtr := getCrtcInfoX11(srPtr, m.platform.crtc)
	if ciPtr != 0 {
		defer randr.FreeCrtcInfo(ciPtr)
		ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))
		// mi can be nil if the monitor has been disconnected
		if mi := getModeInfo(sr, ci.Mode); mi != nil {
			mode = *vidmodeFromModeInfo(mi, ci)
		}
	}

	return &mode
}
