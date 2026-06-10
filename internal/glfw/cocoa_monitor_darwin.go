// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

// NSScreenNumber key string for device description dictionary.
var nsScreenNumberKey objc.ID

func init() {
	classNSString := objc.GetClass("NSString")
	nsScreenNumberKey = objc.ID(classNSString).Send(sel_alloc)
	nsScreenNumberKey = nsScreenNumberKey.Send(objc.RegisterName("initWithUTF8String:"), "NSScreenNumber\x00")
}

// GammaRamp describes the gamma ramp for a monitor.
type GammaRamp struct {
	Red   []uint16 // A slice of value describing the response of the red channel.
	Green []uint16 // A slice of value describing the response of the green channel.
	Blue  []uint16 // A slice of value describing the response of the blue channel.
}

// macOS-specific helper functions

// modeIsGood checks if a display mode is suitable for use.
func modeIsGood(mode uintptr) bool {
	flags := cgDisplayModeGetIOFlags(mode)
	if flags&kDisplayModeValidFlag == 0 || flags&kDisplayModeSafeFlag == 0 {
		return false
	}
	if flags&kDisplayModeInterlacedFlag != 0 {
		return false
	}
	if flags&kDisplayModeStretchedFlag != 0 {
		return false
	}
	return true
}

// getFallbackRefreshRate queries the I/O registry for the display refresh rate.
// This is needed when CGDisplayModeGetRefreshRate returns 0 (e.g. on ProMotion displays).
func getFallbackRefreshRate(displayID uint32) float64 {
	refreshRate := 60.0

	var it uint32
	if ioServiceGetMatchingServices(0, ioServiceMatching(unsafe.StringData("IOFramebuffer\x00")), &it) != 0 {
		return refreshRate
	}
	defer ioObjectRelease(it)

	for {
		service := ioIteratorNext(it)
		if service == 0 {
			break
		}

		indexKey := cfStringCreateWithCString(0, "IOFramebufferOpenGLIndex", kCFStringEncodingUTF8)
		indexRef := ioRegistryEntryCreateCFProperty(service, indexKey, 0, 0)
		cfRelease(indexKey)
		if indexRef == 0 {
			ioObjectRelease(service)
			continue
		}

		var index uint32
		cfNumberGetValue(indexRef, kCFNumberIntType, unsafe.Pointer(&index))
		cfRelease(indexRef)

		if cgOpenGLDisplayMaskToDisplayID(1<<index) != displayID {
			ioObjectRelease(service)
			continue
		}

		clockKey := cfStringCreateWithCString(0, "IOFBCurrentPixelClock", kCFStringEncodingUTF8)
		clockRef := ioRegistryEntryCreateCFProperty(service, clockKey, 0, 0)
		cfRelease(clockKey)
		countKey := cfStringCreateWithCString(0, "IOFBCurrentPixelCount", kCFStringEncodingUTF8)
		countRef := ioRegistryEntryCreateCFProperty(service, countKey, 0, 0)
		cfRelease(countKey)

		var clock, count uint32
		if clockRef != 0 {
			cfNumberGetValue(clockRef, kCFNumberIntType, unsafe.Pointer(&clock))
			cfRelease(clockRef)
		}
		if countRef != 0 {
			cfNumberGetValue(countRef, kCFNumberIntType, unsafe.Pointer(&count))
			cfRelease(countRef)
		}

		if clock > 0 && count > 0 {
			refreshRate = float64(clock) / float64(count)
		}

		ioObjectRelease(service)
		break
	}

	return refreshRate
}

// vidmodeFromCGDisplayMode converts a CGDisplayMode to a VidMode.
func vidmodeFromCGDisplayMode(mode uintptr, fallbackRefreshRate float64) VidMode {
	w := int(cgDisplayModeGetWidth(mode))
	h := int(cgDisplayModeGetHeight(mode))
	refreshRate := int(math.Round(cgDisplayModeGetRefreshRate(mode)))
	if refreshRate == 0 {
		refreshRate = int(math.Round(fallbackRefreshRate))
	}

	return VidMode{
		Width:       w,
		Height:      h,
		RedBits:     8,
		GreenBits:   8,
		BlueBits:    8,
		RefreshRate: refreshRate,
	}
}

// beginFadeReservation acquires a display fade reservation.
func beginFadeReservation() (uint32, bool) {
	var token uint32
	if cgAcquireDisplayFadeReservation(5.0, &token) == kCGErrorSuccess {
		cgDisplayFade(token, 0.3, 0.0, 1.0, 0.0, 0.0, 0.0, 1)
		return token, true
	}
	return kCGDisplayFadeReservationInvalidToken, false
}

// endFadeReservation fades back in and releases a fade reservation.
func endFadeReservation(token uint32) {
	if token != kCGDisplayFadeReservationInvalidToken {
		cgDisplayFade(token, 0.5, 1.0, 0.0, 0.0, 0.0, 0.0, 0)
		cgReleaseDisplayFadeReservation(token)
	}
}

// getMonitorNameNS retrieves the name of a monitor.
// It tries NSScreen.localizedName first (macOS 10.15+), then falls back to IOKit.
func getMonitorNameNS(displayID uint32) string {
	screens := objc.ID(class_NSScreen).Send(sel_screens)
	count := int(screens.Send(sel_count))
	for i := range count {
		screen := screens.Send(sel_objectAtIndex, i)
		dict := screen.Send(sel_deviceDescription)
		screenNum := dict.Send(sel_objectForKey, nsScreenNumberKey)
		if screenNum == 0 {
			continue
		}
		sid := uint32(screenNum.Send(sel_unsignedIntValue))
		// HACK: Compare unit numbers instead of display IDs to work around
		//       display replacement on machines with automatic graphics switching
		if cgDisplayUnitNumber(sid) == cgDisplayUnitNumber(displayID) {
			if screen.Send(objc.RegisterName("respondsToSelector:"), sel_localizedName) != 0 {
				nameID := screen.Send(sel_localizedName)
				if nameID != 0 {
					utf8Ptr := nameID.Send(sel_UTF8String)
					if utf8Ptr != 0 {
						// Use lengthOfBytesUsingEncoding: to get the UTF-8 byte count.
						// NSString.length returns UTF-16 code units which differs for non-ASCII.
						length := int(nameID.Send(sel_lengthOfBytesUsingEncoding, NSUTF8StringEncoding))
						if length > 0 {
							// Copy the string to avoid dangling pointer
							// when the NSString is released.
							src := unsafe.String((*byte)(unsafe.Pointer(utf8Ptr)), length)
							return string([]byte(src))
						}
					}
				}
			}
			break
		}
	}

	// Fallback: use IOKit to get the display name
	ioServiceName := []byte("IODisplayConnect\x00")
	matching := ioServiceMatching(&ioServiceName[0])
	if matching == 0 {
		return "Display"
	}

	var iterator uint32
	if ioServiceGetMatchingServices(0, matching, &iterator) != 0 {
		return "Display"
	}
	defer ioObjectRelease(iterator)

	targetVendor := cgDisplayVendorNumber(displayID)
	targetModel := cgDisplayModelNumber(displayID)

	var matchedInfo uintptr
	for {
		service := ioIteratorNext(iterator)
		if service == 0 {
			break
		}

		info := ioDisplayCreateInfoDictionary(service, kIODisplayOnlyPreferredName)
		ioObjectRelease(service)
		if info == 0 {
			continue
		}

		vendorIDKey := cfString("DisplayVendorID")
		productIDKey := cfString("DisplayProductID")
		vendorIDRef := cfDictionaryGetValue(info, vendorIDKey)
		productIDRef := cfDictionaryGetValue(info, productIDKey)
		cfRelease(vendorIDKey)
		cfRelease(productIDKey)

		if vendorIDRef == 0 || productIDRef == 0 {
			cfRelease(info)
			continue
		}

		var vendorID, productID uint32
		cfNumberGetValue(vendorIDRef, kCFNumberIntType, unsafe.Pointer(&vendorID))
		cfNumberGetValue(productIDRef, kCFNumberIntType, unsafe.Pointer(&productID))

		if vendorID == targetVendor && productID == targetModel {
			matchedInfo = info
			break
		}

		cfRelease(info)
	}

	if matchedInfo == 0 {
		return "Display"
	}
	defer cfRelease(matchedInfo)

	productNameKey := cfString("DisplayProductName")
	defer cfRelease(productNameKey)
	names := cfDictionaryGetValue(matchedInfo, productNameKey)
	if names == 0 {
		return "Display"
	}

	enUSKey := cfString("en_US")
	defer cfRelease(enUSKey)
	nameRef := cfDictionaryGetValue(names, enUSKey)
	if nameRef == 0 {
		return "Display"
	}

	size := cfStringGetMaximumSizeForEncoding(cfStringGetLength(nameRef), kCFStringEncodingUTF8)
	buf := make([]byte, size+1)
	cfStringGetCString(nameRef, &buf[0], size, kCFStringEncodingUTF8)
	return cStringToGoString(buf)
}

// cStringToGoString converts a null-terminated C string in a byte slice to a Go string.
func cStringToGoString(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

// nsScreenForDisplayID finds the NSScreen (as objc.ID) for a given CGDirectDisplayID.
// It compares unit numbers instead of display IDs to work around display replacement
// on machines with automatic graphics switching.
func nsScreenForDisplayID(displayID uint32) objc.ID {
	unitNumber := cgDisplayUnitNumber(displayID)
	screens := objc.ID(class_NSScreen).Send(sel_screens)
	count := int(screens.Send(sel_count))
	for i := range count {
		screen := screens.Send(sel_objectAtIndex, i)
		dict := screen.Send(sel_deviceDescription)
		screenNum := dict.Send(sel_objectForKey, nsScreenNumberKey)
		if screenNum == 0 {
			continue
		}
		sid := uint32(screenNum.Send(sel_unsignedIntValue))
		if cgDisplayUnitNumber(sid) == unitNumber {
			return screen
		}
	}
	return 0
}

// pollMonitorsNS enumerates displays and updates the global monitor list.
func pollMonitorsNS() error {
	var displayCount uint32
	if cgGetOnlineDisplayList(0, nil, &displayCount) != kCGErrorSuccess {
		return fmt.Errorf("glfw: failed to get online display list: %w", PlatformError)
	}
	if displayCount == 0 {
		return nil
	}

	displays := make([]uint32, displayCount)
	if cgGetOnlineDisplayList(displayCount, &displays[0], &displayCount) != kCGErrorSuccess {
		return fmt.Errorf("glfw: failed to get online display list: %w", PlatformError)
	}

	// Reset screen references for all existing monitors.
	for _, m := range _glfw.monitors {
		m.platform.screen = 0
	}

	disconnected := make([]*Monitor, len(_glfw.monitors))
	copy(disconnected, _glfw.monitors)

	for i := uint32(0); i < displayCount; i++ {
		display := displays[i]

		if cgDisplayIsAsleep(display) != 0 {
			continue
		}

		// HACK: Compare unit numbers instead of display IDs to work around
		//       display replacement on machines with automatic graphics
		//       switching
		unitNumber := cgDisplayUnitNumber(display)

		var alreadyKnown bool
		for j, m := range disconnected {
			if m != nil && m.platform.unitNumber == unitNumber {
				disconnected[j] = nil
				alreadyKnown = true
				// Update the screen reference for the already-known monitor.
				m.platform.screen = nsScreenForDisplayID(display)
				break
			}
		}
		if alreadyKnown {
			continue
		}

		name := getMonitorNameNS(display)

		monitor := &Monitor{
			name: name,
		}
		monitor.platform.displayID = display
		monitor.platform.unitNumber = unitNumber
		monitor.platform.screen = nsScreenForDisplayID(display)

		mode := cgDisplayCopyDisplayMode(display)
		if cgDisplayModeGetRefreshRate(mode) == 0.0 {
			monitor.platform.fallbackRefreshRate = getFallbackRefreshRate(display)
		}
		cfRelease(mode)

		typ := _GLFW_INSERT_LAST

		if err := inputMonitor(monitor, Connected, typ); err != nil {
			return err
		}
	}

	for _, m := range disconnected {
		if m != nil {
			if err := inputMonitor(m, Disconnected, 0); err != nil {
				return err
			}
		}
	}

	return nil
}

// setVideoModeNS finds the closest video mode and switches to it.
func (m *Monitor) setVideoModeNS(desired *VidMode) error {
	best, err := m.chooseVideoMode(desired)
	if err != nil {
		return err
	}
	current := m.platformGetVideoMode()
	if best.equals(current) {
		return nil
	}

	modes := cgDisplayCopyAllDisplayModes(m.platform.displayID, 0)
	if modes == 0 {
		return fmt.Errorf("glfw: failed to copy display modes: %w", PlatformError)
	}
	defer cfRelease(modes)

	count := cfArrayGetCount(modes)
	var native uintptr

	for i := range count {
		dm := cfArrayGetValueAtIndex(modes, i)
		if !modeIsGood(dm) {
			continue
		}

		vm := vidmodeFromCGDisplayMode(dm, m.platform.fallbackRefreshRate)
		if best.equals(&vm) {
			native = dm
			break
		}
	}

	if native != 0 {
		if m.platform.previousMode == 0 {
			m.platform.previousMode = cgDisplayCopyDisplayMode(m.platform.displayID)
		}

		token, hasFade := beginFadeReservation()

		cgDisplaySetDisplayMode(m.platform.displayID, native, 0)

		if hasFade {
			endFadeReservation(token)
		}
	}

	return nil
}

// restoreVideoModeNS restores the previous display mode.
func (m *Monitor) restoreVideoModeNS() {
	if m.platform.previousMode == 0 {
		return
	}

	token, hasFade := beginFadeReservation()

	cgDisplaySetDisplayMode(m.platform.displayID, m.platform.previousMode, 0)

	if hasFade {
		endFadeReservation(token)
	}

	cfRelease(m.platform.previousMode)
	m.platform.previousMode = 0
}

// transformYNS transforms a Y coordinate from Cocoa (bottom-left origin)
// to GLFW (top-left origin) coordinate space.
func transformYNS(y float32) float32 {
	bounds := cgDisplayBounds(cgMainDisplayID())
	return float32(bounds.Height) - y - 1
}

// Platform functions

func (m *Monitor) platformGetMonitorPos() (xpos, ypos int, ok bool) {
	bounds := cgDisplayBounds(m.platform.displayID)
	return int(bounds.X), int(bounds.Y), true
}

func (m *Monitor) platformGetMonitorContentScale() (xscale, yscale float32, err error) {
	if m.platform.screen == 0 {
		return 0, 0, fmt.Errorf("glfw: cannot query content scale without screen: %w", PlatformError)
	}

	points := objc.Send[cocoa.NSRect](m.platform.screen, sel_frame)
	pixels := objc.Send[cocoa.NSRect](m.platform.screen, sel_convertRectToBacking, points)

	return float32(pixels.Size.Width / points.Size.Width),
		float32(pixels.Size.Height / points.Size.Height), nil
}

func (m *Monitor) platformGetMonitorWorkarea() (xpos, ypos, width, height int) {
	screen := m.platform.screen
	if screen == 0 {
		screen = nsScreenForDisplayID(m.platform.displayID)
	}
	if screen == 0 {
		bounds := cgDisplayBounds(m.platform.displayID)
		return int(bounds.X), int(bounds.Y), int(bounds.Width), int(bounds.Height)
	}

	visibleFrame := objc.Send[cgRect](screen, sel_visibleFrame)
	primaryBounds := cgDisplayBounds(cgMainDisplayID())

	xpos = int(visibleFrame.X)
	ypos = int(primaryBounds.Height - visibleFrame.Y - visibleFrame.Height)
	width = int(visibleFrame.Width)
	height = int(visibleFrame.Height)
	return
}

func (m *Monitor) platformAppendVideoModes(monitors []*VidMode) ([]*VidMode, error) {
	origLen := len(monitors)

	modes := cgDisplayCopyAllDisplayModes(m.platform.displayID, 0)
	if modes == 0 {
		return monitors, nil
	}
	defer cfRelease(modes)

	count := cfArrayGetCount(modes)
	for i := range count {
		mode := cfArrayGetValueAtIndex(modes, i)
		if !modeIsGood(mode) {
			continue
		}

		vm := vidmodeFromCGDisplayMode(mode, m.platform.fallbackRefreshRate)
		vmPtr := &vm

		duplicate := false
		for _, existing := range monitors[origLen:] {
			if existing.equals(vmPtr) {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}

		monitors = append(monitors, vmPtr)
	}

	return monitors, nil
}

func (m *Monitor) platformGetVideoMode() *VidMode {
	mode := cgDisplayCopyDisplayMode(m.platform.displayID)
	if mode == 0 {
		return &VidMode{}
	}
	defer cfRelease(mode)

	vm := vidmodeFromCGDisplayMode(mode, m.platform.fallbackRefreshRate)
	return &vm
}

func (m *Monitor) platformGetGammaRamp() (GammaRamp, error) {
	sampleCount := cgDisplayGammaTableCapacity(m.platform.displayID)

	red := make([]float32, sampleCount)
	green := make([]float32, sampleCount)
	blue := make([]float32, sampleCount)
	if cgGetDisplayTransferByTable(m.platform.displayID, sampleCount, &red[0], &green[0], &blue[0], &sampleCount) != kCGErrorSuccess {
		return GammaRamp{}, fmt.Errorf("glfw: failed to get gamma ramp: %w", PlatformError)
	}

	ramp := GammaRamp{
		Red:   make([]uint16, sampleCount),
		Green: make([]uint16, sampleCount),
		Blue:  make([]uint16, sampleCount),
	}
	for i := uint32(0); i < sampleCount; i++ {
		ramp.Red[i] = uint16(red[i] * 65535.0)
		ramp.Green[i] = uint16(green[i] * 65535.0)
		ramp.Blue[i] = uint16(blue[i] * 65535.0)
	}

	return ramp, nil
}

func (m *Monitor) platformSetGammaRamp(ramp *GammaRamp) error {
	size := len(ramp.Red)
	red := make([]float32, size)
	green := make([]float32, size)
	blue := make([]float32, size)
	for i := range size {
		red[i] = float32(ramp.Red[i]) / 65535.0
		green[i] = float32(ramp.Green[i]) / 65535.0
		blue[i] = float32(ramp.Blue[i]) / 65535.0
	}

	if cgSetDisplayTransferByTable(m.platform.displayID, uint32(size), &red[0], &green[0], &blue[0]) != kCGErrorSuccess {
		return fmt.Errorf("glfw: failed to set gamma ramp: %w", PlatformError)
	}

	return nil
}
