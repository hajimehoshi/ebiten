// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2026 The Ebitengine Authors

//go:build freebsd || linux || netbsd

package glfw

import (
	"fmt"
	"slices"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
)

// getGLXFBConfigAttrib returns the specified attribute of the specified
// GLXFBConfig.
func getGLXFBConfigAttrib(fbconfig uintptr, attrib int32) int32 {
	var value int32
	_glfw.platformContext.glx.GetFBConfigAttrib(_glfw.platformWindow.display, fbconfig, attrib, &value)
	return value
}

// chooseGLXFBConfig returns the GLXFBConfig most closely matching the
// specified hints.
func chooseGLXFBConfig(desired *fbconfig) (uintptr, error) {
	glx := &_glfw.platformContext.glx

	trustWindowBit := true

	// HACK: This is a (hopefully temporary) workaround for Chromium
	//       (VirtualBox GL) not setting the window bit on any GLXFBConfigs
	vendor := goString(glx.GetClientString(_glfw.platformWindow.display, _GLX_VENDOR))
	if vendor == "Chromium" {
		trustWindowBit = false
	}

	var nativeCount int32
	nativeConfigsPtr := glx.GetFBConfigs(_glfw.platformWindow.display, int32(_glfw.platformWindow.screen), &nativeCount)
	if nativeConfigsPtr == 0 || nativeCount == 0 {
		return 0, fmt.Errorf("glfw: glx: no GLXFBConfigs returned: %w", APIUnavailable)
	}
	nativeConfigs := unsafe.Slice((*uintptr)(unsafe.Pointer(nativeConfigsPtr)), int(nativeCount))

	usableConfigs := make([]*fbconfig, 0, nativeCount)

	for _, n := range nativeConfigs {
		var u fbconfig

		// Only consider RGBA GLXFBConfigs
		if getGLXFBConfigAttrib(n, _GLX_RENDER_TYPE)&_GLX_RGBA_BIT == 0 {
			continue
		}

		// Only consider window GLXFBConfigs
		if getGLXFBConfigAttrib(n, _GLX_DRAWABLE_TYPE)&_GLX_WINDOW_BIT == 0 {
			if trustWindowBit {
				continue
			}
		}

		if intToBool(int(getGLXFBConfigAttrib(n, _GLX_DOUBLEBUFFER))) != desired.doublebuffer {
			continue
		}

		if desired.transparent {
			viPtr := glx.GetVisualFromFBConfig(_glfw.platformWindow.display, n)
			if viPtr != 0 {
				u.transparent = isVisualTransparentX11((*_XVisualInfo)(unsafe.Pointer(viPtr)).Visual)
				xFree(viPtr)
			}
		}

		u.redBits = int(getGLXFBConfigAttrib(n, _GLX_RED_SIZE))
		u.greenBits = int(getGLXFBConfigAttrib(n, _GLX_GREEN_SIZE))
		u.blueBits = int(getGLXFBConfigAttrib(n, _GLX_BLUE_SIZE))

		u.alphaBits = int(getGLXFBConfigAttrib(n, _GLX_ALPHA_SIZE))
		u.depthBits = int(getGLXFBConfigAttrib(n, _GLX_DEPTH_SIZE))
		u.stencilBits = int(getGLXFBConfigAttrib(n, _GLX_STENCIL_SIZE))

		u.accumRedBits = int(getGLXFBConfigAttrib(n, _GLX_ACCUM_RED_SIZE))
		u.accumGreenBits = int(getGLXFBConfigAttrib(n, _GLX_ACCUM_GREEN_SIZE))
		u.accumBlueBits = int(getGLXFBConfigAttrib(n, _GLX_ACCUM_BLUE_SIZE))
		u.accumAlphaBits = int(getGLXFBConfigAttrib(n, _GLX_ACCUM_ALPHA_SIZE))

		u.auxBuffers = int(getGLXFBConfigAttrib(n, _GLX_AUX_BUFFERS))

		if getGLXFBConfigAttrib(n, _GLX_STEREO) != 0 {
			u.stereo = true
		}

		if glx.ARB_multisample {
			u.samples = int(getGLXFBConfigAttrib(n, _GLX_SAMPLES))
		}

		if glx.ARB_framebuffer_sRGB || glx.EXT_framebuffer_sRGB {
			u.sRGB = intToBool(int(getGLXFBConfigAttrib(n, _GLX_FRAMEBUFFER_SRGB_CAPABLE_ARB)))
		}

		u.handle = n
		usableConfigs = append(usableConfigs, &u)
	}

	closest := chooseFBConfig(desired, usableConfigs)

	xFree(nativeConfigsPtr)

	if closest == nil {
		return 0, fmt.Errorf("glfw: glx: failed to find a suitable GLXFBConfig: %w", FormatUnavailable)
	}
	return closest.handle, nil
}

// createLegacyContextGLX creates an OpenGL context using the legacy API.
func createLegacyContextGLX(window *Window, fbconfig uintptr, share uintptr) uintptr {
	return _glfw.platformContext.glx.CreateNewContext(_glfw.platformWindow.display,
		fbconfig,
		_GLX_RGBA_TYPE,
		share,
		true)
}

func makeContextCurrentGLX(window *Window) error {
	glx := &_glfw.platformContext.glx
	if window != nil {
		if !glx.MakeCurrent(_glfw.platformWindow.display,
			window.context.platform.glx.window,
			window.context.platform.glx.handle) {
			_glfw.currentContext = nil
			return fmt.Errorf("glfw: glx: failed to make context current: %w", PlatformError)
		}
		_glfw.currentContext = window
	} else {
		_glfw.currentContext = nil
		if !glx.MakeCurrent(_glfw.platformWindow.display, _None, 0) {
			return fmt.Errorf("glfw: glx: failed to clear current context: %w", PlatformError)
		}
	}
	return nil
}

func swapBuffersGLX(window *Window) error {
	_glfw.platformContext.glx.SwapBuffers(_glfw.platformWindow.display, window.context.platform.glx.window)
	window.signalFrameSyncCounter()
	return nil
}

func swapIntervalGLX(window *Window, interval int) error {
	glx := &_glfw.platformContext.glx
	switch {
	case glx.EXT_swap_control:
		glx.SwapIntervalEXT(_glfw.platformWindow.display, window.context.platform.glx.window, int32(interval))
	case glx.MESA_swap_control:
		glx.SwapIntervalMESA(int32(interval))
	case glx.SGI_swap_control:
		if interval > 0 {
			glx.SwapIntervalSGI(int32(interval))
		}
	}
	return nil
}

func extensionSupportedGLX(extension string) bool {
	extensions := goString(_glfw.platformContext.glx.QueryExtensionsString(_glfw.platformWindow.display, int32(_glfw.platformWindow.screen)))
	if len(extensions) == 0 {
		return false
	}
	return slices.Contains(strings.Split(extensions, " "), extension)
}

func getProcAddressGLX(procname string) uintptr {
	glx := &_glfw.platformContext.glx
	if glx.GetProcAddress != nil {
		return glx.GetProcAddress(procname)
	}
	if glx.GetProcAddressARB != nil {
		return glx.GetProcAddressARB(procname)
	}
	// NOTE: glvnd provides GLX 1.4, so this can only happen with libGL
	sym, err := purego.Dlsym(glx.handle, procname)
	if err != nil {
		return 0
	}
	return sym
}

func destroyContextGLX(window *Window) error {
	glx := &_glfw.platformContext.glx

	if window.context.platform.glx.window != 0 {
		glx.DestroyWindow(_glfw.platformWindow.display, window.context.platform.glx.window)
		window.context.platform.glx.window = _None
	}

	if window.context.platform.glx.handle != 0 {
		glx.DestroyContext(_glfw.platformWindow.display, window.context.platform.glx.handle)
		window.context.platform.glx.handle = 0
	}
	return nil
}

// initGLX initializes GLX.
func initGLX() error {
	glx := &_glfw.platformContext.glx

	if glx.handle != 0 {
		return nil
	}

	handle, err := openX11Library("libGLX.so.0", "libGL.so.1", "libGL.so")
	if err != nil {
		return fmt.Errorf("glfw: glx: failed to load GLX: %w", APIUnavailable)
	}
	glx.handle = handle

	registerRequired := func(fptr any, name string) bool {
		sym, err := purego.Dlsym(handle, name)
		if err != nil || sym == 0 {
			return false
		}
		purego.RegisterFunc(fptr, sym)
		return true
	}

	if !registerRequired(&glx.GetFBConfigs, "glXGetFBConfigs") ||
		!registerRequired(&glx.GetFBConfigAttrib, "glXGetFBConfigAttrib") ||
		!registerRequired(&glx.GetClientString, "glXGetClientString") ||
		!registerRequired(&glx.QueryExtension, "glXQueryExtension") ||
		!registerRequired(&glx.QueryVersion, "glXQueryVersion") ||
		!registerRequired(&glx.DestroyContext, "glXDestroyContext") ||
		!registerRequired(&glx.MakeCurrent, "glXMakeCurrent") ||
		!registerRequired(&glx.SwapBuffers, "glXSwapBuffers") ||
		!registerRequired(&glx.QueryExtensionsString, "glXQueryExtensionsString") ||
		!registerRequired(&glx.CreateNewContext, "glXCreateNewContext") ||
		!registerRequired(&glx.CreateWindow, "glXCreateWindow") ||
		!registerRequired(&glx.DestroyWindow, "glXDestroyWindow") ||
		!registerRequired(&glx.GetVisualFromFBConfig, "glXGetVisualFromFBConfig") {
		return fmt.Errorf("glfw: glx: failed to load required entry points: %w", PlatformError)
	}

	// NOTE: Unlike GLX 1.3 entry points these are not required to be present
	if sym, err := purego.Dlsym(handle, "glXGetProcAddress"); err == nil && sym != 0 {
		purego.RegisterFunc(&glx.GetProcAddress, sym)
	}
	if sym, err := purego.Dlsym(handle, "glXGetProcAddressARB"); err == nil && sym != 0 {
		purego.RegisterFunc(&glx.GetProcAddressARB, sym)
	}

	if !glx.QueryExtension(_glfw.platformWindow.display, &glx.errorBase, &glx.eventBase) {
		return fmt.Errorf("glfw: glx: GLX extension not found: %w", APIUnavailable)
	}

	if !glx.QueryVersion(_glfw.platformWindow.display, &glx.major, &glx.minor) {
		return fmt.Errorf("glfw: glx: failed to query GLX version: %w", APIUnavailable)
	}

	if glx.major == 1 && glx.minor < 3 {
		return fmt.Errorf("glfw: glx: GLX version 1.3 is required: %w", APIUnavailable)
	}

	if extensionSupportedGLX("GLX_EXT_swap_control") {
		if proc := getProcAddressGLX("glXSwapIntervalEXT"); proc != 0 {
			purego.RegisterFunc(&glx.SwapIntervalEXT, proc)
			glx.EXT_swap_control = true
		}
	}

	if extensionSupportedGLX("GLX_SGI_swap_control") {
		if proc := getProcAddressGLX("glXSwapIntervalSGI"); proc != 0 {
			purego.RegisterFunc(&glx.SwapIntervalSGI, proc)
			glx.SGI_swap_control = true
		}
	}

	if extensionSupportedGLX("GLX_MESA_swap_control") {
		if proc := getProcAddressGLX("glXSwapIntervalMESA"); proc != 0 {
			purego.RegisterFunc(&glx.SwapIntervalMESA, proc)
			glx.MESA_swap_control = true
		}
	}

	if extensionSupportedGLX("GLX_ARB_multisample") {
		glx.ARB_multisample = true
	}

	if extensionSupportedGLX("GLX_ARB_framebuffer_sRGB") {
		glx.ARB_framebuffer_sRGB = true
	}

	if extensionSupportedGLX("GLX_EXT_framebuffer_sRGB") {
		glx.EXT_framebuffer_sRGB = true
	}

	if extensionSupportedGLX("GLX_ARB_create_context") {
		if proc := getProcAddressGLX("glXCreateContextAttribsARB"); proc != 0 {
			purego.RegisterFunc(&glx.CreateContextAttribsARB, proc)
			glx.ARB_create_context = true
		}
	}

	if extensionSupportedGLX("GLX_ARB_create_context_robustness") {
		glx.ARB_create_context_robustness = true
	}

	if extensionSupportedGLX("GLX_ARB_create_context_profile") {
		glx.ARB_create_context_profile = true
	}

	if extensionSupportedGLX("GLX_EXT_create_context_es2_profile") {
		glx.EXT_create_context_es2_profile = true
	}

	if extensionSupportedGLX("GLX_ARB_create_context_no_error") {
		glx.ARB_create_context_no_error = true
	}

	if extensionSupportedGLX("GLX_ARB_context_flush_control") {
		glx.ARB_context_flush_control = true
	}

	return nil
}

// terminateGLX terminates GLX.
func terminateGLX() {
	// NOTE: This function must not call any X11 functions, as it is called
	//       after XCloseDisplay (see platformTerminate for details)

	glx := &_glfw.platformContext.glx
	if glx.handle != 0 {
		_ = purego.Dlclose(glx.handle)
		glx.handle = 0
	}
}

// createContextGLX creates the OpenGL or OpenGL ES context.
func createContextGLX(window *Window, ctxconfig *ctxconfig, fbconfig *fbconfig) error {
	glx := &_glfw.platformContext.glx

	var share uintptr
	if ctxconfig.share != nil {
		share = ctxconfig.share.context.platform.glx.handle
	}

	native, err := chooseGLXFBConfig(fbconfig)
	if err != nil {
		return err
	}

	if ctxconfig.client == OpenGLESAPI {
		if !glx.ARB_create_context ||
			!glx.ARB_create_context_profile ||
			!glx.EXT_create_context_es2_profile {
			return fmt.Errorf("glfw: glx: OpenGL ES requested but GLX_EXT_create_context_es2_profile is unavailable: %w", APIUnavailable)
		}
	}

	if ctxconfig.forward {
		if !glx.ARB_create_context {
			return fmt.Errorf("glfw: glx: forward compatibility requested but GLX_ARB_create_context_profile is unavailable: %w", VersionUnavailable)
		}
	}

	if ctxconfig.profile != 0 {
		if !glx.ARB_create_context || !glx.ARB_create_context_profile {
			return fmt.Errorf("glfw: glx: an OpenGL profile requested but GLX_ARB_create_context_profile is unavailable: %w", VersionUnavailable)
		}
	}

	grabErrorHandlerX11()

	if glx.ARB_create_context {
		var attribs []int32
		var mask, flags int32

		setAttrib := func(a, v int32) {
			attribs = append(attribs, a, v)
		}

		if ctxconfig.client == OpenGLAPI {
			if ctxconfig.forward {
				flags |= _GLX_CONTEXT_FORWARD_COMPATIBLE_BIT_ARB
			}

			if ctxconfig.profile == OpenGLCoreProfile {
				mask |= _GLX_CONTEXT_CORE_PROFILE_BIT_ARB
			} else if ctxconfig.profile == OpenGLCompatProfile {
				mask |= _GLX_CONTEXT_COMPATIBILITY_PROFILE_BIT_ARB
			}
		} else {
			mask |= _GLX_CONTEXT_ES2_PROFILE_BIT_EXT
		}

		if ctxconfig.debug {
			flags |= _GLX_CONTEXT_DEBUG_BIT_ARB
		}

		if ctxconfig.robustness != 0 {
			if glx.ARB_create_context_robustness {
				if ctxconfig.robustness == NoResetNotification {
					setAttrib(_GLX_CONTEXT_RESET_NOTIFICATION_STRATEGY_ARB, _GLX_NO_RESET_NOTIFICATION_ARB)
				} else if ctxconfig.robustness == LoseContextOnReset {
					setAttrib(_GLX_CONTEXT_RESET_NOTIFICATION_STRATEGY_ARB, _GLX_LOSE_CONTEXT_ON_RESET_ARB)
				}

				flags |= _GLX_CONTEXT_ROBUST_ACCESS_BIT_ARB
			}
		}

		if ctxconfig.release != 0 {
			if glx.ARB_context_flush_control {
				if ctxconfig.release == ReleaseBehaviorNone {
					setAttrib(_GLX_CONTEXT_RELEASE_BEHAVIOR_ARB, _GLX_CONTEXT_RELEASE_BEHAVIOR_NONE_ARB)
				} else if ctxconfig.release == ReleaseBehaviorFlush {
					setAttrib(_GLX_CONTEXT_RELEASE_BEHAVIOR_ARB, _GLX_CONTEXT_RELEASE_BEHAVIOR_FLUSH_ARB)
				}
			}
		}

		if ctxconfig.noerror {
			if glx.ARB_create_context_no_error {
				setAttrib(_GLX_CONTEXT_OPENGL_NO_ERROR_ARB, 1)
			}
		}

		// NOTE: Only request an explicitly versioned context when necessary, as
		//       explicitly requesting version 1.0 does not always return the
		//       highest version supported by the driver
		if ctxconfig.major != 1 || ctxconfig.minor != 0 {
			setAttrib(_GLX_CONTEXT_MAJOR_VERSION_ARB, int32(ctxconfig.major))
			setAttrib(_GLX_CONTEXT_MINOR_VERSION_ARB, int32(ctxconfig.minor))
		}

		if mask != 0 {
			setAttrib(_GLX_CONTEXT_PROFILE_MASK_ARB, mask)
		}

		if flags != 0 {
			setAttrib(_GLX_CONTEXT_FLAGS_ARB, flags)
		}

		setAttrib(_None, _None)

		window.context.platform.glx.handle =
			glx.CreateContextAttribsARB(_glfw.platformWindow.display,
				native,
				share,
				true,
				&attribs[0])

		// HACK: This is a fallback for broken versions of the Mesa
		//       implementation of GLX_ARB_create_context_profile that fail
		//       default 1.0 context creation with a GLXBadProfileARB error in
		//       violation of the extension spec
		if window.context.platform.glx.handle == 0 {
			if _glfw.platformWindow.errorCode == int(glx.errorBase)+_GLXBadProfileARB &&
				ctxconfig.client == OpenGLAPI &&
				ctxconfig.profile == OpenGLAnyProfile &&
				!ctxconfig.forward {
				window.context.platform.glx.handle = createLegacyContextGLX(window, native, share)
			}
		}
	} else {
		window.context.platform.glx.handle = createLegacyContextGLX(window, native, share)
	}

	releaseErrorHandlerX11()

	if window.context.platform.glx.handle == 0 {
		return inputErrorX11(VersionUnavailable, "GLX: Failed to create context")
	}

	window.context.platform.glx.window =
		glx.CreateWindow(_glfw.platformWindow.display, native, window.platform.handle, nil)
	if window.context.platform.glx.window == 0 {
		return fmt.Errorf("glfw: glx: failed to create window: %w", PlatformError)
	}

	window.context.makeCurrent = makeContextCurrentGLX
	window.context.swapBuffers = swapBuffersGLX
	window.context.swapInterval = swapIntervalGLX
	window.context.extensionSupported = extensionSupportedGLX
	window.context.getProcAddress = getProcAddressGLX
	window.context.destroy = destroyContextGLX

	return nil
}

// chooseVisualGLX returns the Visual and depth of the chosen GLXFBConfig.
func chooseVisualGLX(wndconfig *wndconfig, ctxconfig *ctxconfig, fbconfig *fbconfig) (visual uintptr, depth int32, err error) {
	native, err := chooseGLXFBConfig(fbconfig)
	if err != nil {
		return 0, 0, err
	}

	resultPtr := _glfw.platformContext.glx.GetVisualFromFBConfig(_glfw.platformWindow.display, native)
	if resultPtr == 0 {
		return 0, 0, fmt.Errorf("glfw: glx: failed to retrieve Visual for GLXFBConfig: %w", PlatformError)
	}
	result := (*_XVisualInfo)(unsafe.Pointer(resultPtr))

	visual = result.Visual
	depth = result.Depth

	xFree(resultPtr)
	return visual, depth, nil
}
