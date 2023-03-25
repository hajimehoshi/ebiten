// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package goglfw

// #cgo LDFLAGS: -framework CoreGraphics
// #include <CoreGraphics/CoreGraphics.h>
import "C"
import (
	"errors"
	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	"math"
)

func (m *Monitor) platformAppendVideoModes(monitors []*VidMode) ([]*VidMode, error) {
	panic("NOT IMPLEMENTED")
}

func (m *Monitor) GetCocoaMonitor() (uintptr, error) {
	//    _GLFW_REQUIRE_INIT_OR_RETURN(kCGNullDirectDisplay);
	return uintptr(m.platform.displayID), nil
}

func (m *Monitor) platformGetMonitorPos() (xpos, ypos int, ok bool) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	bounds := C.CGDisplayBounds(C.uint(m.platform.displayID))
	return int(bounds.origin.x), int(bounds.origin.y), true
}

func (m *Monitor) platformGetMonitorWorkarea() (xpos, ypos, width, height int) {
	panic("NOT IMPLEMENTED")
}

func (m *Monitor) platformGetMonitorContentScale() (xscale, yscale float32, err error) {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	if m.platform.screen.ID == 0 {
		return 0, 0, errors.New("cocoa: cannot query content scale without screen")
	}
	//@autoreleasepool {
	//
	//    if (!monitor->ns.screen)
	//    {
	//        _glfwInputError(GLFW_PLATFORM_ERROR,
	//                        "Cocoa: Cannot query content scale without screen");
	//    }
	points := m.platform.screen.Frame()
	pixels := m.platform.screen.ConvertRectToBacking(points)
	//    const NSRect points = [monitor->ns.screen frame];
	//    const NSRect pixels = [monitor->ns.screen convertRectToBacking:points];
	//
	//    if (xscale)
	//        *xscale = (float) (pixels.size.width / points.size.width);
	//    if (yscale)
	//        *yscale = (float) (pixels.size.height / points.size.height);
	//
	//    } // autoreleasepool
	return float32(pixels.Size.Width / points.Size.Width), float32(pixels.Size.Height / points.Size.Height), nil
}

func (m *Monitor) platformGetVideoMode() *VidMode {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	//    @autoreleasepool {
	//
	native := _CGDisplayCopyDisplayMode(m.platform.displayID)
	mode := vidmodeFromCGDisplayMode(native, m.platform.fallbackRefreshRate)
	_CGDisplayModeRelease(native)
	//    CGDisplayModeRef native = CGDisplayCopyDisplayMode(monitor->ns.displayID);
	//    *mode = vidmodeFromCGDisplayMode(native, monitor->ns.fallbackRefreshRate);
	//    CGDisplayModeRelease(native);
	//
	//    } // autoreleasepool
	return &mode
}

func platformPollMonitors() error {
	var displayCount uint32

	//	  uint32_t displayCount;
	_CGGetOnlineDisplayList(0, nil, &displayCount)
	//    CGGetOnlineDisplayList(0, NULL, &displayCount);
	displays := make([]_CGDirectDisplayID, displayCount)
	//    CGDirectDisplayID* displays = _glfw_calloc(displayCount, sizeof(CGDirectDisplayID));
	_CGGetOnlineDisplayList(displayCount, &displays[0], &displayCount)
	//    CGGetOnlineDisplayList(displayCount, displays, &displayCount);
	//
	//    for (int i = 0;  i < _glfw.monitorCount;  i++)
	//        _glfw.monitors[i]->ns.screen = nil;
	//
	var disconnected = make([]*Monitor, len(_glfw.monitors))
	//    _GLFWmonitor** disconnected = NULL;
	//    uint32_t disconnectedCount = _glfw.monitorCount;
	copy(disconnected, _glfw.monitors)
	//    if (disconnectedCount)
	//    {
	//        disconnected = _glfw_calloc(_glfw.monitorCount, sizeof(_GLFWmonitor*));
	//        memcpy(disconnected,
	//               _glfw.monitors,
	//               _glfw.monitorCount * sizeof(_GLFWmonitor*));
	//    }
	//
	for i := uint32(0); i < displayCount; i++ {
		//    for (uint32_t i = 0;  i < displayCount;  i++)
		//    {
		if _CGDisplayIsAsleep(displays[i]) {
			continue
		}
		//        if (CGDisplayIsAsleep(displays[i]))
		//            continue;
		unitNumber := _CGDisplayUnitNumber(displays[i])
		var screen cocoa.NSScreen
		//        const uint32_t unitNumber = CGDisplayUnitNumber(displays[i]);
		//        NSScreen* screen = nil;
		screens := cocoa.NSScreen_screens()
		count := screens.Count()
		//        for (screen in [NSScreen screens])
		//        {
		//            NSNumber* screenNumber = [screen deviceDescription][@"NSScreenNumber"];
		for k := cocoa.NSUInteger(0); k < count; k++ {
			screen = cocoa.NSScreen{ID: screens.ObjectAtIndex(k)}
			screenNumber := cocoa.NSNumber{ID: screen.DeviceDescription().ObjectForKey(cocoa.NSString_alloc().InitWithUTF8String("NSScreenNumber").ID)}
			// HACK: Compare unit numbers instead of display IDs to work around
			//       display replacement on machines with automatic graphics
			//       switching
			if _CGDisplayUnitNumber(_CGDirectDisplayID(screenNumber.UnsignedIntValue())) == unitNumber {
				break
			}
			//            if (CGDisplayUnitNumber([screenNumber unsignedIntValue]) == unitNumber)
			//                break;
			//        }
		}
		// HACK: Compare unit numbers instead of display IDs to work around
		//       display replacement on machines with automatic graphics
		//       switching
		var j int
		for j = 0; j < len(disconnected); j++ {
			//        uint32_t j;
			//        for (j = 0;  j < disconnectedCount;  j++)
			//        {
			if disconnected[j] != nil && disconnected[j].platform.unitNumber == unitNumber {
				disconnected[j].platform.screen = screen
				disconnected[j] = nil
				break
			}
			//            if (disconnected[j] && disconnected[j]->ns.unitNumber == unitNumber)
			//            {
			//                disconnected[j]->ns.screen = screen;
			//                disconnected[j] = NULL;
			//                break;
			//            }
			//        }
		}
		if j < len(disconnected) {
			continue
		}
		//        if (j < disconnectedCount)
		//            continue;
		//
		//        const CGSize size = CGDisplayScreenSize(displays[i]);
		name := "" // TODO: getMonitorName
		//        char* name = getMonitorName(displays[i], screen);
		//        if (!name)
		//            continue;
		//
		monitor := &Monitor{name: name}
		monitor.platform.displayID = displays[i]
		monitor.platform.unitNumber = unitNumber
		monitor.platform.screen = screen
		//        _GLFWmonitor* monitor = _glfwAllocMonitor(name, size.width, size.height);
		//        monitor->ns.displayID  = displays[i];
		//        monitor->ns.unitNumber = unitNumber;
		//        monitor->ns.screen     = screen;
		//
		//        _glfw_free(name);
		//
		//        CGDisplayModeRef mode = CGDisplayCopyDisplayMode(displays[i]);
		//        if (CGDisplayModeGetRefreshRate(mode) == 0.0)
		//            monitor->ns.fallbackRefreshRate = getFallbackRefreshRate(displays[i]);
		//        CGDisplayModeRelease(mode);
		//
		err := inputMonitor(monitor, Connected, _GLFW_INSERT_LAST)
		if err != nil {
			return err
		}
		//        _glfwInputMonitor(monitor, GLFW_CONNECTED, _GLFW_INSERT_LAST);
		//    }
	}
	for _, monitor := range disconnected {
		if monitor != nil {
			err := inputMonitor(monitor, Disconnected, 0)
			if err != nil {
				return err
			}
		}
	}

	//    for (uint32_t i = 0;  i < disconnectedCount;  i++)
	//    {
	//        if (disconnected[i])
	//            _glfwInputMonitor(disconnected[i], GLFW_DISCONNECTED, 0);
	//    }
	disconnected = nil
	displays = nil
	//    _glfw_free(disconnected);
	//    _glfw_free(displays);
	return nil
}

// Convert Core Graphics display mode to GLFW video mode
func vidmodeFromCGDisplayMode(mode _CGDisplayModeRef, fallbackRefreshRate float64) (result VidMode) {
	result.Width = int(_CGDisplayModeGetWidth(mode))
	result.Height = int(_CGDisplayModeGetHeight(mode))
	result.RefreshRate = int(math.Round(_CGDisplayModeGetRefreshRate(mode)))
	if result.RefreshRate == 0 {
		result.RefreshRate = (int)(math.Round(fallbackRefreshRate))
	}
	// TODO: ...
	//#if MAC_OS_X_VERSION_MAX_ALLOWED <= 101100
	//    CFStringRef format = CGDisplayModeCopyPixelEncoding(mode);
	//    if (CFStringCompare(format, CFSTR(IO16BitDirectPixels), 0) == 0)
	//    {
	//        result.redBits = 5;
	//        result.greenBits = 5;
	//        result.blueBits = 5;
	//    }
	//    else
	//#endif /* MAC_OS_X_VERSION_MAX_ALLOWED */
	{
		result.RedBits = 8
		result.GreenBits = 8
		result.BlueBits = 8
	}
	//#if MAC_OS_X_VERSION_MAX_ALLOWED <= 101100
	//    CFRelease(format);
	//#endif /* MAC_OS_X_VERSION_MAX_ALLOWED */
	//    return result;
	return result
}
