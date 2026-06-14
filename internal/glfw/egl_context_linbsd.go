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

// getEGLErrorString returns a description of the specified EGL error.
func getEGLErrorString(err int32) string {
	switch err {
	case _EGL_SUCCESS:
		return "Success"
	case _EGL_NOT_INITIALIZED:
		return "EGL is not or could not be initialized"
	case _EGL_BAD_ACCESS:
		return "EGL cannot access a requested resource"
	case _EGL_BAD_ALLOC:
		return "EGL failed to allocate resources for the requested operation"
	case _EGL_BAD_ATTRIBUTE:
		return "An unrecognized attribute or attribute value was passed in the attribute list"
	case _EGL_BAD_CONTEXT:
		return "An EGLContext argument does not name a valid EGL rendering context"
	case _EGL_BAD_CONFIG:
		return "An EGLConfig argument does not name a valid EGL frame buffer configuration"
	case _EGL_BAD_CURRENT_SURFACE:
		return "The current surface of the calling thread is a window, pixel buffer or pixmap that is no longer valid"
	case _EGL_BAD_DISPLAY:
		return "An EGLDisplay argument does not name a valid EGL display connection"
	case _EGL_BAD_SURFACE:
		return "An EGLSurface argument does not name a valid surface configured for GL rendering"
	case _EGL_BAD_MATCH:
		return "Arguments are inconsistent"
	case _EGL_BAD_PARAMETER:
		return "One or more argument values are invalid"
	case _EGL_BAD_NATIVE_PIXMAP:
		return "A NativePixmapType argument does not refer to a valid native pixmap"
	case _EGL_BAD_NATIVE_WINDOW:
		return "A NativeWindowType argument does not refer to a valid native window"
	case _EGL_CONTEXT_LOST:
		return "The application must destroy all contexts and reinitialise"
	default:
		return "ERROR: UNKNOWN EGL ERROR"
	}
}

// getEGLConfigAttrib returns the specified attribute of the specified
// EGLConfig.
func getEGLConfigAttrib(config uintptr, attrib int32) int32 {
	var value int32
	_glfw.platformContext.egl.GetConfigAttrib(_glfw.platformContext.egl.display, config, attrib, &value)
	return value
}

// chooseEGLConfig returns the EGLConfig most closely matching the specified
// hints.
func chooseEGLConfig(ctxconfig *ctxconfig, fbconfig_ *fbconfig) (uintptr, error) {
	egl := &_glfw.platformContext.egl

	var apiBit int32
	if ctxconfig.client == OpenGLESAPI {
		if ctxconfig.major == 1 {
			apiBit = _EGL_OPENGL_ES_BIT
		} else {
			apiBit = _EGL_OPENGL_ES2_BIT
		}
	} else {
		apiBit = _EGL_OPENGL_BIT
	}

	if fbconfig_.stereo {
		return 0, fmt.Errorf("glfw: egl: stereo rendering not supported: %w", FormatUnavailable)
	}

	var nativeCount int32
	egl.GetConfigs(egl.display, nil, 0, &nativeCount)
	if nativeCount == 0 {
		return 0, fmt.Errorf("glfw: egl: no EGLConfigs returned: %w", APIUnavailable)
	}

	nativeConfigs := make([]uintptr, nativeCount)
	egl.GetConfigs(egl.display, &nativeConfigs[0], nativeCount, &nativeCount)
	nativeConfigs = nativeConfigs[:nativeCount]

	usableConfigs := make([]*fbconfig, 0, nativeCount)
	wrongApiAvailable := false

	for _, n := range nativeConfigs {
		var u fbconfig

		// Only consider RGB(A) EGLConfigs
		if getEGLConfigAttrib(n, _EGL_COLOR_BUFFER_TYPE) != _EGL_RGB_BUFFER {
			continue
		}

		// Only consider window EGLConfigs
		if getEGLConfigAttrib(n, _EGL_SURFACE_TYPE)&_EGL_WINDOW_BIT == 0 {
			continue
		}

		// Only consider EGLConfigs with associated Visuals
		visualID := getEGLConfigAttrib(n, _EGL_NATIVE_VISUAL_ID)
		if visualID == 0 {
			continue
		}

		if fbconfig_.transparent {
			var vi _XVisualInfo
			vi.Visualid = _VisualID(visualID)
			var count int32
			visPtr := xGetVisualInfo(_glfw.platformWindow.display, _VisualIDMask, &vi, &count)
			if visPtr != 0 {
				u.transparent = isVisualTransparentX11((*_XVisualInfo)(unsafe.Pointer(visPtr)).Visual)
				xFree(visPtr)
			}
		}

		if getEGLConfigAttrib(n, _EGL_RENDERABLE_TYPE)&apiBit == 0 {
			wrongApiAvailable = true
			continue
		}

		u.redBits = int(getEGLConfigAttrib(n, _EGL_RED_SIZE))
		u.greenBits = int(getEGLConfigAttrib(n, _EGL_GREEN_SIZE))
		u.blueBits = int(getEGLConfigAttrib(n, _EGL_BLUE_SIZE))

		u.alphaBits = int(getEGLConfigAttrib(n, _EGL_ALPHA_SIZE))
		u.depthBits = int(getEGLConfigAttrib(n, _EGL_DEPTH_SIZE))
		u.stencilBits = int(getEGLConfigAttrib(n, _EGL_STENCIL_SIZE))

		u.samples = int(getEGLConfigAttrib(n, _EGL_SAMPLES))
		u.doublebuffer = fbconfig_.doublebuffer

		u.handle = n
		usableConfigs = append(usableConfigs, &u)
	}

	closest := chooseFBConfig(fbconfig_, usableConfigs)
	if closest == nil {
		if wrongApiAvailable {
			if ctxconfig.client == OpenGLESAPI {
				if ctxconfig.major == 1 {
					return 0, fmt.Errorf("glfw: egl: failed to find support for OpenGL ES 1.x: %w", APIUnavailable)
				}
				return 0, fmt.Errorf("glfw: egl: failed to find support for OpenGL ES 2 or later: %w", APIUnavailable)
			}
			return 0, fmt.Errorf("glfw: egl: failed to find support for OpenGL: %w", APIUnavailable)
		}
		return 0, fmt.Errorf("glfw: egl: failed to find a suitable EGLConfig: %w", FormatUnavailable)
	}

	return closest.handle, nil
}

func makeContextCurrentEGL(window *Window) error {
	egl := &_glfw.platformContext.egl
	if window != nil {
		if !egl.MakeCurrent(egl.display,
			window.context.platform.egl.surface,
			window.context.platform.egl.surface,
			window.context.platform.egl.handle) {
			_glfw.currentContext = nil
			return fmt.Errorf("glfw: egl: failed to make context current: %s: %w", getEGLErrorString(egl.GetError()), PlatformError)
		}
		_glfw.currentContext = window
	} else {
		_glfw.currentContext = nil
		if !egl.MakeCurrent(egl.display, _EGL_NO_SURFACE, _EGL_NO_SURFACE, _EGL_NO_CONTEXT) {
			return fmt.Errorf("glfw: egl: failed to clear current context: %s: %w", getEGLErrorString(egl.GetError()), PlatformError)
		}
	}
	return nil
}

func swapBuffersEGL(window *Window) error {
	egl := &_glfw.platformContext.egl
	if window != _glfw.currentContext {
		return fmt.Errorf("glfw: egl: the context must be current on the calling thread when swapping buffers: %w", PlatformError)
	}
	egl.SwapBuffers(egl.display, window.context.platform.egl.surface)
	return nil
}

func swapIntervalEGL(window *Window, interval int) error {
	egl := &_glfw.platformContext.egl
	egl.SwapInterval(egl.display, int32(interval))
	return nil
}

func extensionSupportedEGL(extension string) bool {
	egl := &_glfw.platformContext.egl
	extensions := goString(egl.QueryString(egl.display, _EGL_EXTENSIONS))
	if len(extensions) == 0 {
		return false
	}
	return slices.Contains(strings.Split(extensions, " "), extension)
}

func getProcAddressEGL(procname string) uintptr {
	window := _glfw.currentContext

	if window != nil && window.context.platform.egl.client != 0 {
		if proc, err := purego.Dlsym(window.context.platform.egl.client, procname); err == nil && proc != 0 {
			return proc
		}
	}

	return _glfw.platformContext.egl.GetProcAddress(procname)
}

func destroyContextEGL(window *Window) error {
	egl := &_glfw.platformContext.egl

	// NOTE: Do not unload libGL.so.1 while the X11 display is still open,
	//       as it will make XCloseDisplay segfault
	if window.context.client != OpenGLAPI {
		if window.context.platform.egl.client != 0 {
			_ = purego.Dlclose(window.context.platform.egl.client)
			window.context.platform.egl.client = 0
		}
	}

	if window.context.platform.egl.surface != 0 {
		egl.DestroySurface(egl.display, window.context.platform.egl.surface)
		window.context.platform.egl.surface = _EGL_NO_SURFACE
	}

	if window.context.platform.egl.handle != 0 {
		egl.DestroyContext(egl.display, window.context.platform.egl.handle)
		window.context.platform.egl.handle = _EGL_NO_CONTEXT
	}
	return nil
}

// initEGL initializes EGL.
func initEGL() error {
	egl := &_glfw.platformContext.egl

	if egl.handle != 0 {
		return nil
	}

	sonames := []string{"libEGL.so.1", "libEGL.so"}
	var soname string
	for _, name := range sonames {
		handle, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err == nil {
			egl.handle = handle
			soname = name
			break
		}
	}

	if egl.handle == 0 {
		return fmt.Errorf("glfw: egl: library not found: %w", APIUnavailable)
	}

	egl.prefix = strings.HasPrefix(soname, "lib")

	registerRequired := func(fptr any, name string) bool {
		sym, err := purego.Dlsym(egl.handle, name)
		if err != nil || sym == 0 {
			return false
		}
		purego.RegisterFunc(fptr, sym)
		return true
	}

	if !registerRequired(&egl.GetConfigAttrib, "eglGetConfigAttrib") ||
		!registerRequired(&egl.GetConfigs, "eglGetConfigs") ||
		!registerRequired(&egl.GetDisplay, "eglGetDisplay") ||
		!registerRequired(&egl.GetError, "eglGetError") ||
		!registerRequired(&egl.Initialize, "eglInitialize") ||
		!registerRequired(&egl.Terminate, "eglTerminate") ||
		!registerRequired(&egl.BindAPI, "eglBindAPI") ||
		!registerRequired(&egl.CreateContext, "eglCreateContext") ||
		!registerRequired(&egl.DestroySurface, "eglDestroySurface") ||
		!registerRequired(&egl.DestroyContext, "eglDestroyContext") ||
		!registerRequired(&egl.CreateWindowSurface, "eglCreateWindowSurface") ||
		!registerRequired(&egl.MakeCurrent, "eglMakeCurrent") ||
		!registerRequired(&egl.SwapBuffers, "eglSwapBuffers") ||
		!registerRequired(&egl.SwapInterval, "eglSwapInterval") ||
		!registerRequired(&egl.QueryString, "eglQueryString") ||
		!registerRequired(&egl.GetProcAddress, "eglGetProcAddress") {
		terminateEGL()
		return fmt.Errorf("glfw: egl: failed to load required entry points: %w", PlatformError)
	}

	egl.display = egl.GetDisplay(_glfw.platformWindow.display)
	if egl.display == _EGL_NO_DISPLAY {
		err := fmt.Errorf("glfw: egl: failed to get EGL display: %s: %w", getEGLErrorString(egl.GetError()), APIUnavailable)
		terminateEGL()
		return err
	}

	if !egl.Initialize(egl.display, &egl.major, &egl.minor) {
		err := fmt.Errorf("glfw: egl: failed to initialize EGL: %s: %w", getEGLErrorString(egl.GetError()), APIUnavailable)
		terminateEGL()
		return err
	}

	egl.KHR_create_context = extensionSupportedEGL("EGL_KHR_create_context")
	egl.KHR_create_context_no_error = extensionSupportedEGL("EGL_KHR_create_context_no_error")
	egl.KHR_gl_colorspace = extensionSupportedEGL("EGL_KHR_gl_colorspace")
	egl.KHR_get_all_proc_addresses = extensionSupportedEGL("EGL_KHR_get_all_proc_addresses")
	egl.KHR_context_flush_control = extensionSupportedEGL("EGL_KHR_context_flush_control")
	egl.EXT_present_opaque = extensionSupportedEGL("EGL_EXT_present_opaque")

	return nil
}

// terminateEGL terminates EGL.
func terminateEGL() {
	egl := &_glfw.platformContext.egl

	if egl.display != 0 {
		egl.Terminate(egl.display)
		egl.display = _EGL_NO_DISPLAY
	}

	if egl.handle != 0 {
		_ = purego.Dlclose(egl.handle)
		egl.handle = 0
	}
}

// createContextEGL creates the OpenGL or OpenGL ES context.
func createContextEGL(window *Window, ctxconfig *ctxconfig, fbconfig *fbconfig) error {
	egl := &_glfw.platformContext.egl

	if egl.display == 0 {
		return fmt.Errorf("glfw: egl: API not available: %w", APIUnavailable)
	}

	var share uintptr
	if ctxconfig.share != nil {
		share = ctxconfig.share.context.platform.egl.handle
	}

	config, err := chooseEGLConfig(ctxconfig, fbconfig)
	if err != nil {
		return err
	}

	if ctxconfig.client == OpenGLESAPI {
		if !egl.BindAPI(_EGL_OPENGL_ES_API) {
			return fmt.Errorf("glfw: egl: failed to bind OpenGL ES: %s: %w", getEGLErrorString(egl.GetError()), APIUnavailable)
		}
	} else {
		if !egl.BindAPI(_EGL_OPENGL_API) {
			return fmt.Errorf("glfw: egl: failed to bind OpenGL: %s: %w", getEGLErrorString(egl.GetError()), APIUnavailable)
		}
	}

	var attribs []int32
	setAttrib := func(a, v int32) {
		attribs = append(attribs, a, v)
	}

	if egl.KHR_create_context {
		var mask, flags int32

		if ctxconfig.client == OpenGLAPI {
			if ctxconfig.forward {
				flags |= _EGL_CONTEXT_OPENGL_FORWARD_COMPATIBLE_BIT_KHR
			}

			if ctxconfig.profile == OpenGLCoreProfile {
				mask |= _EGL_CONTEXT_OPENGL_CORE_PROFILE_BIT_KHR
			} else if ctxconfig.profile == OpenGLCompatProfile {
				mask |= _EGL_CONTEXT_OPENGL_COMPATIBILITY_PROFILE_BIT_KHR
			}
		}

		if ctxconfig.debug {
			flags |= _EGL_CONTEXT_OPENGL_DEBUG_BIT_KHR
		}

		if ctxconfig.robustness != 0 {
			if ctxconfig.robustness == NoResetNotification {
				setAttrib(_EGL_CONTEXT_OPENGL_RESET_NOTIFICATION_STRATEGY_KHR, _EGL_NO_RESET_NOTIFICATION_KHR)
			} else if ctxconfig.robustness == LoseContextOnReset {
				setAttrib(_EGL_CONTEXT_OPENGL_RESET_NOTIFICATION_STRATEGY_KHR, _EGL_LOSE_CONTEXT_ON_RESET_KHR)
			}

			flags |= _EGL_CONTEXT_OPENGL_ROBUST_ACCESS_BIT_KHR
		}

		if ctxconfig.major != 1 || ctxconfig.minor != 0 {
			setAttrib(_EGL_CONTEXT_MAJOR_VERSION_KHR, int32(ctxconfig.major))
			setAttrib(_EGL_CONTEXT_MINOR_VERSION_KHR, int32(ctxconfig.minor))
		}

		if ctxconfig.noerror {
			if egl.KHR_create_context_no_error {
				setAttrib(_EGL_CONTEXT_OPENGL_NO_ERROR_KHR, 1)
			}
		}

		if mask != 0 {
			setAttrib(_EGL_CONTEXT_OPENGL_PROFILE_MASK_KHR, mask)
		}

		if flags != 0 {
			setAttrib(_EGL_CONTEXT_FLAGS_KHR, flags)
		}
	} else {
		if ctxconfig.client == OpenGLESAPI {
			setAttrib(_EGL_CONTEXT_CLIENT_VERSION, int32(ctxconfig.major))
		}
	}

	if egl.KHR_context_flush_control {
		if ctxconfig.release == ReleaseBehaviorNone {
			setAttrib(_EGL_CONTEXT_RELEASE_BEHAVIOR_KHR, _EGL_CONTEXT_RELEASE_BEHAVIOR_NONE_KHR)
		} else if ctxconfig.release == ReleaseBehaviorFlush {
			setAttrib(_EGL_CONTEXT_RELEASE_BEHAVIOR_KHR, _EGL_CONTEXT_RELEASE_BEHAVIOR_FLUSH_KHR)
		}
	}

	setAttrib(_EGL_NONE, _EGL_NONE)

	window.context.platform.egl.handle = egl.CreateContext(egl.display, config, share, &attribs[0])

	if window.context.platform.egl.handle == _EGL_NO_CONTEXT {
		return fmt.Errorf("glfw: egl: failed to create context: %s: %w", getEGLErrorString(egl.GetError()), VersionUnavailable)
	}

	// Set up attributes for surface creation
	attribs = attribs[:0]

	if fbconfig.sRGB {
		if egl.KHR_gl_colorspace {
			setAttrib(_EGL_GL_COLORSPACE_KHR, _EGL_GL_COLORSPACE_SRGB_KHR)
		}
	}

	if !fbconfig.doublebuffer {
		setAttrib(_EGL_RENDER_BUFFER, _EGL_SINGLE_BUFFER)
	}

	setAttrib(_EGL_NONE, _EGL_NONE)

	window.context.platform.egl.surface =
		egl.CreateWindowSurface(egl.display, config, window.platform.handle, &attribs[0])
	if window.context.platform.egl.surface == _EGL_NO_SURFACE {
		return fmt.Errorf("glfw: egl: failed to create window surface: %s: %w", getEGLErrorString(egl.GetError()), PlatformError)
	}

	window.context.platform.egl.config = config

	// Load the appropriate client library
	if !egl.KHR_get_all_proc_addresses {
		var sonames []string
		if ctxconfig.client == OpenGLESAPI {
			if ctxconfig.major == 1 {
				sonames = []string{"libGLESv1_CM.so.1", "libGLES_CM.so.1"}
			} else {
				sonames = []string{"libGLESv2.so.2"}
			}
		} else {
			sonames = []string{"libOpenGL.so.0", "libGL.so.1"}
		}

		for _, name := range sonames {
			// HACK: Match presence of lib prefix to increase chance of finding
			//       a matching pair in the jungle that is Win32 EGL/GLES
			if egl.prefix != strings.HasPrefix(name, "lib") {
				continue
			}

			handle, err := purego.Dlopen(name, purego.RTLD_LAZY|purego.RTLD_GLOBAL)
			if err == nil {
				window.context.platform.egl.client = handle
				break
			}
		}

		if window.context.platform.egl.client == 0 {
			return fmt.Errorf("glfw: egl: failed to load client library: %w", APIUnavailable)
		}
	}

	window.context.makeCurrent = makeContextCurrentEGL
	window.context.swapBuffers = swapBuffersEGL
	window.context.swapInterval = swapIntervalEGL
	window.context.extensionSupported = extensionSupportedEGL
	window.context.getProcAddress = getProcAddressEGL
	window.context.destroy = destroyContextEGL

	return nil
}

// chooseVisualEGL returns the Visual and depth of the chosen EGLConfig.
func chooseVisualEGL(wndconfig *wndconfig, ctxconfig *ctxconfig, fbconfig *fbconfig) (visual uintptr, depth int32, err error) {
	native, err := chooseEGLConfig(ctxconfig, fbconfig)
	if err != nil {
		return 0, 0, err
	}

	var visualID int32
	_glfw.platformContext.egl.GetConfigAttrib(_glfw.platformContext.egl.display, native, _EGL_NATIVE_VISUAL_ID, &visualID)

	var desired _XVisualInfo
	desired.Screen = int32(_glfw.platformWindow.screen)
	desired.Visualid = _VisualID(visualID)

	var count int32
	resultPtr := xGetVisualInfo(_glfw.platformWindow.display, _VisualScreenMask|_VisualIDMask, &desired, &count)
	if resultPtr == 0 {
		return 0, 0, fmt.Errorf("glfw: egl: failed to retrieve Visual for EGLConfig: %w", PlatformError)
	}
	result := (*_XVisualInfo)(unsafe.Pointer(resultPtr))

	visual = result.Visual
	depth = result.Depth

	xFree(resultPtr)
	return visual, depth, nil
}
