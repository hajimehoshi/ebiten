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

// pollMonitorsX11 polls for changes in the set of connected monitors.
func pollMonitorsX11() error {
	if _glfw.platformWindow.randr.available && !_glfw.platformWindow.randr.monitorBroken {
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

		outputs := unsafe.Slice((*_RROutput)(unsafe.Pointer(sr.Outputs)), int(sr.Noutput))
		for i := range outputs {
			oiPtr := randr.GetOutputInfo(display, srPtr, outputs[i])
			oi := (*_XRROutputInfo)(unsafe.Pointer(oiPtr))

			if oi.Connection != _RR_Connected || oi.Crtc == _None {
				randr.FreeOutputInfo(oiPtr)
				continue
			}

			j := 0
			for ; j < len(disconnected); j++ {
				if disconnected[j] != nil && disconnected[j].platform.output == outputs[i] {
					disconnected[j] = nil
					break
				}
			}
			if j < len(disconnected) {
				randr.FreeOutputInfo(oiPtr)
				continue
			}

			ciPtr := randr.GetCrtcInfo(display, srPtr, oi.Crtc)
			ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))

			monitor := &Monitor{name: goString(oi.Name)}
			monitor.platform.output = outputs[i]
			monitor.platform.crtc = oi.Crtc

			for j := range screens {
				if int32(screens[j].XOrg) == ci.X &&
					int32(screens[j].YOrg) == ci.Y &&
					int32(screens[j].Width) == int32(ci.Width) &&
					int32(screens[j].Height) == int32(ci.Height) {
					monitor.platform.index = j
					break
				}
			}

			placement := _GLFW_INSERT_LAST
			if monitor.platform.output == primary {
				placement = _GLFW_INSERT_FIRST
			}

			err := inputMonitor(monitor, Connected, placement)

			randr.FreeOutputInfo(oiPtr)
			randr.FreeCrtcInfo(ciPtr)

			if err != nil {
				return err
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

	return inputMonitor(&Monitor{name: "Display"}, Connected, _GLFW_INSERT_FIRST)
}

// setVideoModeX11 sets the current video mode for the specified monitor.
func setVideoModeX11(monitor *Monitor, desired *VidMode) error {
	if _glfw.platformWindow.randr.available && !_glfw.platformWindow.randr.monitorBroken {
		display := _glfw.platformWindow.display
		randr := &_glfw.platformWindow.randr

		best, err := monitor.chooseVideoMode(desired)
		if err != nil {
			return err
		}
		current := monitor.platformGetVideoMode()
		if current != nil && current.equals(best) {
			return nil
		}

		srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
		sr := (*_XRRScreenResources)(unsafe.Pointer(srPtr))
		ciPtr := randr.GetCrtcInfo(display, srPtr, monitor.platform.crtc)
		ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))
		oiPtr := randr.GetOutputInfo(display, srPtr, monitor.platform.output)
		oi := (*_XRROutputInfo)(unsafe.Pointer(oiPtr))

		native := _RRMode(_None)

		modes := unsafe.Slice((*_RRMode)(unsafe.Pointer(oi.Modes)), int(oi.Nmode))
		for i := range modes {
			mi := getModeInfo(sr, modes[i])
			if !modeIsGood(mi) {
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

		randr.FreeOutputInfo(oiPtr)
		randr.FreeCrtcInfo(ciPtr)
		randr.FreeScreenResources(srPtr)
	}
	return nil
}

// restoreVideoModeX11 restores the saved (original) video mode for the
// specified monitor.
func restoreVideoModeX11(monitor *Monitor) {
	if _glfw.platformWindow.randr.available && !_glfw.platformWindow.randr.monitorBroken {
		if monitor.platform.oldMode == _None {
			return
		}

		display := _glfw.platformWindow.display
		randr := &_glfw.platformWindow.randr

		srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
		ciPtr := randr.GetCrtcInfo(display, srPtr, monitor.platform.crtc)
		ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))

		randr.SetCrtcConfig(display,
			srPtr, monitor.platform.crtc,
			_CurrentTime,
			ci.X, ci.Y,
			monitor.platform.oldMode,
			ci.Rotation,
			(*_RROutput)(unsafe.Pointer(ci.Outputs)),
			ci.Noutput)

		randr.FreeCrtcInfo(ciPtr)
		randr.FreeScreenResources(srPtr)

		monitor.platform.oldMode = _None
	}
}

func (m *Monitor) platformGetMonitorPos() (xpos, ypos int, ok bool) {
	if !_glfw.platformWindow.randr.available || _glfw.platformWindow.randr.monitorBroken {
		return 0, 0, false
	}

	display := _glfw.platformWindow.display
	randr := &_glfw.platformWindow.randr

	srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
	defer randr.FreeScreenResources(srPtr)

	ciPtr := randr.GetCrtcInfo(display, srPtr, m.platform.crtc)
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

	if _glfw.platformWindow.randr.available && !_glfw.platformWindow.randr.monitorBroken {
		randr := &_glfw.platformWindow.randr

		srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
		sr := (*_XRRScreenResources)(unsafe.Pointer(srPtr))
		ciPtr := randr.GetCrtcInfo(display, srPtr, m.platform.crtc)
		ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))

		areaX = ci.X
		areaY = ci.Y

		mi := getModeInfo(sr, ci.Mode)

		if ci.Rotation == _RR_Rotate_90 || ci.Rotation == _RR_Rotate_270 {
			areaWidth = int32(mi.Height)
			areaHeight = int32(mi.Width)
		} else {
			areaWidth = int32(mi.Width)
			areaHeight = int32(mi.Height)
		}

		randr.FreeCrtcInfo(ciPtr)
		randr.FreeScreenResources(srPtr)
	} else {
		areaWidth = xDisplayWidth(display, int32(_glfw.platformWindow.screen))
		areaHeight = xDisplayHeight(display, int32(_glfw.platformWindow.screen))
	}

	if _glfw.platformWindow.NET_WORKAREA != 0 && _glfw.platformWindow.NET_CURRENT_DESKTOP != 0 {
		var extentsPtr, desktopPtr uintptr
		extentCount := getWindowPropertyX11(_glfw.platformWindow.root,
			_glfw.platformWindow.NET_WORKAREA,
			_XA_CARDINAL,
			&extentsPtr)

		if getWindowPropertyX11(_glfw.platformWindow.root,
			_glfw.platformWindow.NET_CURRENT_DESKTOP,
			_XA_CARDINAL,
			&desktopPtr) > 0 {
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

		if extentsPtr != 0 {
			xFree(extentsPtr)
		}
		if desktopPtr != 0 {
			xFree(desktopPtr)
		}
	}

	return int(areaX), int(areaY), int(areaWidth), int(areaHeight)
}

func (m *Monitor) platformAppendVideoModes(monitors []*VidMode) ([]*VidMode, error) {
	result := monitors

	if _glfw.platformWindow.randr.available && !_glfw.platformWindow.randr.monitorBroken {
		display := _glfw.platformWindow.display
		randr := &_glfw.platformWindow.randr

		srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
		sr := (*_XRRScreenResources)(unsafe.Pointer(srPtr))
		ciPtr := randr.GetCrtcInfo(display, srPtr, m.platform.crtc)
		ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))
		oiPtr := randr.GetOutputInfo(display, srPtr, m.platform.output)
		oi := (*_XRROutputInfo)(unsafe.Pointer(oiPtr))

		modes := unsafe.Slice((*_RRMode)(unsafe.Pointer(oi.Modes)), int(oi.Nmode))
		for i := range modes {
			mi := getModeInfo(sr, modes[i])
			if !modeIsGood(mi) {
				continue
			}

			mode := vidmodeFromModeInfo(mi, ci)

			// Skip duplicate modes
			duplicate := false
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

		randr.FreeOutputInfo(oiPtr)
		randr.FreeCrtcInfo(ciPtr)
		randr.FreeScreenResources(srPtr)
	} else {
		result = append(result, m.platformGetVideoMode())
	}

	return result, nil
}

func (m *Monitor) platformGetVideoMode() *VidMode {
	display := _glfw.platformWindow.display

	var mode VidMode

	if _glfw.platformWindow.randr.available && !_glfw.platformWindow.randr.monitorBroken {
		randr := &_glfw.platformWindow.randr

		srPtr := randr.GetScreenResourcesCurrent(display, _glfw.platformWindow.root)
		sr := (*_XRRScreenResources)(unsafe.Pointer(srPtr))

		ciPtr := randr.GetCrtcInfo(display, srPtr, m.platform.crtc)
		if ciPtr != 0 {
			ci := (*_XRRCrtcInfo)(unsafe.Pointer(ciPtr))
			// mi can be nil if the monitor has been disconnected
			if mi := getModeInfo(sr, ci.Mode); mi != nil {
				mode = *vidmodeFromModeInfo(mi, ci)
			}

			randr.FreeCrtcInfo(ciPtr)
		}

		randr.FreeScreenResources(srPtr)
	} else {
		mode.Width = int(xDisplayWidth(display, int32(_glfw.platformWindow.screen)))
		mode.Height = int(xDisplayHeight(display, int32(_glfw.platformWindow.screen)))
		mode.RefreshRate = 0

		mode.RedBits, mode.GreenBits, mode.BlueBits =
			splitBPP(int(xDefaultDepth(display, int32(_glfw.platformWindow.screen))))
	}

	return &mode
}
