// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/hajimehoshi/ebiten/v2/internal/microsoftgdk"
	"github.com/hajimehoshi/ebiten/v2/internal/winver"
)

type platformContextState struct {
	dc       _HDC
	handle   _HGLRC
	interval int
}

type platformLibraryContextState struct {
	inited bool

	EXT_swap_control               bool
	EXT_colorspace                 bool
	ARB_multisample                bool
	ARB_framebuffer_sRGB           bool
	EXT_framebuffer_sRGB           bool
	ARB_pixel_format               bool
	ARB_create_context             bool
	ARB_create_context_profile     bool
	EXT_create_context_es2_profile bool
	ARB_create_context_robustness  bool
	ARB_create_context_no_error    bool
	ARB_context_flush_control      bool
}

func findPixelFormatAttribValue(attribs []int32, values []int32, attrib int32) int32 {
	for i := range attribs {
		if attribs[i] == attrib {
			return values[i]
		}
	}
	return 0
}

func (w *Window) choosePixelFormat(ctxconfig *ctxconfig, fbconfig_ *fbconfig) (int, error) {
	var nativeCount int32
	var attribs []int32

	c, err := _DescribePixelFormat(w.context.platform.dc, 1, uint32(unsafe.Sizeof(_PIXELFORMATDESCRIPTOR{})), nil)
	if err != nil {
		return 0, err
	}
	nativeCount = c

	if _glfw.platformContext.ARB_pixel_format {
		// NOTE: In a Parallels VM WGL_ARB_pixel_format returns fewer pixel formats than
		//       DescribePixelFormat, violating the guarantees of the extension spec
		// HACK: Iterate through the minimum of both counts
		var attrib int32 = _WGL_NUMBER_PIXEL_FORMATS_ARB
		var extensionCount int32
		if err := wglGetPixelFormatAttribivARB(w.context.platform.dc, 1, 0, 1, &attrib, &extensionCount); err != nil {
			return 0, fmt.Errorf("glfw: WGL: failed to retrieve pixel format attribute: %w", err)
		}
		if nativeCount > extensionCount {
			nativeCount = extensionCount
		}

		attribs = append(attribs,
			_WGL_SUPPORT_OPENGL_ARB,
			_WGL_DRAW_TO_WINDOW_ARB,
			_WGL_PIXEL_TYPE_ARB,
			_WGL_ACCELERATION_ARB,
			_WGL_RED_BITS_ARB,
			_WGL_RED_SHIFT_ARB,
			_WGL_GREEN_BITS_ARB,
			_WGL_GREEN_SHIFT_ARB,
			_WGL_BLUE_BITS_ARB,
			_WGL_BLUE_SHIFT_ARB,
			_WGL_ALPHA_BITS_ARB,
			_WGL_ALPHA_SHIFT_ARB,
			_WGL_DEPTH_BITS_ARB,
			_WGL_STENCIL_BITS_ARB,
			_WGL_ACCUM_BITS_ARB,
			_WGL_ACCUM_RED_BITS_ARB,
			_WGL_ACCUM_GREEN_BITS_ARB,
			_WGL_ACCUM_BLUE_BITS_ARB,
			_WGL_ACCUM_ALPHA_BITS_ARB,
			_WGL_AUX_BUFFERS_ARB,
			_WGL_STEREO_ARB,
			_WGL_DOUBLE_BUFFER_ARB)

		if _glfw.platformContext.ARB_multisample {
			attribs = append(attribs, _WGL_SAMPLES_ARB)
		}

		if ctxconfig.client == OpenGLAPI {
			if _glfw.platformContext.ARB_framebuffer_sRGB || _glfw.platformContext.EXT_framebuffer_sRGB {
				attribs = append(attribs, _WGL_FRAMEBUFFER_SRGB_CAPABLE_ARB)
			}
		} else {
			if _glfw.platformContext.EXT_colorspace {
				attribs = append(attribs, _WGL_COLORSPACE_EXT)
			}
		}
	}

	usableConfigs := make([]*fbconfig, 0, nativeCount)
	for i := int32(0); i < nativeCount; i++ {
		var u fbconfig
		pixelFormat := uintptr(i) + 1

		if _glfw.platformContext.ARB_pixel_format {
			// Get pixel format attributes through "modern" extension
			values := make([]int32, len(attribs))
			if err := wglGetPixelFormatAttribivARB(w.context.platform.dc, int32(pixelFormat), 0, uint32(len(attribs)), &attribs[0], &values[0]); err != nil {
				return 0, err
			}

			findAttribValue := func(attrib int32) int32 {
				return findPixelFormatAttribValue(attribs, values, attrib)
			}

			if findAttribValue(_WGL_SUPPORT_OPENGL_ARB) == 0 || findAttribValue(_WGL_DRAW_TO_WINDOW_ARB) == 0 {
				continue
			}

			if findAttribValue(_WGL_PIXEL_TYPE_ARB) != _WGL_TYPE_RGBA_ARB {
				continue
			}

			if findAttribValue(_WGL_ACCELERATION_ARB) == _WGL_NO_ACCELERATION_ARB {
				continue
			}

			if (findAttribValue(_WGL_DOUBLE_BUFFER_ARB) != 0) != fbconfig_.doublebuffer {
				continue
			}

			u.redBits = int(findAttribValue(_WGL_RED_BITS_ARB))
			u.greenBits = int(findAttribValue(_WGL_GREEN_BITS_ARB))
			u.blueBits = int(findAttribValue(_WGL_BLUE_BITS_ARB))
			u.alphaBits = int(findAttribValue(_WGL_ALPHA_BITS_ARB))

			u.depthBits = int(findAttribValue(_WGL_DEPTH_BITS_ARB))
			u.stencilBits = int(findAttribValue(_WGL_STENCIL_BITS_ARB))

			u.accumRedBits = int(findAttribValue(_WGL_ACCUM_RED_BITS_ARB))
			u.accumGreenBits = int(findAttribValue(_WGL_ACCUM_GREEN_BITS_ARB))
			u.accumBlueBits = int(findAttribValue(_WGL_ACCUM_BLUE_BITS_ARB))
			u.accumAlphaBits = int(findAttribValue(_WGL_ACCUM_ALPHA_BITS_ARB))

			u.auxBuffers = int(findAttribValue(_WGL_AUX_BUFFERS_ARB))

			if findAttribValue(_WGL_STEREO_ARB) != 0 {
				u.stereo = true
			}

			if _glfw.platformContext.ARB_multisample {
				u.samples = int(findAttribValue(_WGL_SAMPLES_ARB))
			}

			if ctxconfig.client == OpenGLAPI {
				if _glfw.platformContext.ARB_framebuffer_sRGB || _glfw.platformContext.EXT_framebuffer_sRGB {
					if findAttribValue(_WGL_FRAMEBUFFER_SRGB_CAPABLE_ARB) != 0 {
						u.sRGB = true
					}
				}
			} else {
				if _glfw.platformContext.EXT_colorspace {
					if findAttribValue(_WGL_COLORSPACE_EXT) == _WGL_COLORSPACE_SRGB_EXT {
						u.sRGB = true
					}
				}
			}
		} else {
			// Get pixel format attributes through legacy PFDs

			var pfd _PIXELFORMATDESCRIPTOR
			if _, err := _DescribePixelFormat(w.context.platform.dc, int32(pixelFormat), uint32(unsafe.Sizeof(pfd)), &pfd); err != nil {
				return 0, err
			}

			if pfd.dwFlags&_PFD_DRAW_TO_WINDOW == 0 || pfd.dwFlags&_PFD_SUPPORT_OPENGL == 0 {
				continue
			}

			if pfd.dwFlags&_PFD_GENERIC_ACCELERATED == 0 && pfd.dwFlags&_PFD_GENERIC_FORMAT != 0 {
				continue
			}

			if pfd.iPixelType != _PFD_TYPE_RGBA {
				continue
			}

			if (pfd.dwFlags&_PFD_DOUBLEBUFFER != 0) != fbconfig_.doublebuffer {
				continue
			}

			u.redBits = int(pfd.cRedBits)
			u.greenBits = int(pfd.cGreenBits)
			u.blueBits = int(pfd.cBlueBits)
			u.alphaBits = int(pfd.cAlphaBits)

			u.depthBits = int(pfd.cDepthBits)
			u.stencilBits = int(pfd.cStencilBits)

			u.accumRedBits = int(pfd.cAccumRedBits)
			u.accumGreenBits = int(pfd.cAccumGreenBits)
			u.accumBlueBits = int(pfd.cAccumBlueBits)
			u.accumAlphaBits = int(pfd.cAccumAlphaBits)

			u.auxBuffers = int(pfd.cAuxBuffers)

			if pfd.dwFlags&_PFD_STEREO != 0 {
				u.stereo = true
			}
		}

		u.handle = pixelFormat
		usableConfigs = append(usableConfigs, &u)
	}

	if len(usableConfigs) == 0 {
		return 0, fmt.Errorf("glfw: the driver does not appear to support OpenGL")
	}

	closest := chooseFBConfig(fbconfig_, usableConfigs)
	if closest == nil {
		return 0, fmt.Errorf("glfw: failed to find a suitable pixel format")
	}

	return int(closest.handle), nil
}

func makeContextCurrentWGL(window *Window) error {
	if window != nil {
		if err := wglMakeCurrent(window.context.platform.dc, window.context.platform.handle); err != nil {
			_ = _glfw.contextSlot.set(0)
			return err
		}
		if err := _glfw.contextSlot.set(uintptr(unsafe.Pointer(window))); err != nil {
			return err
		}
	} else {
		if err := wglMakeCurrent(0, 0); err != nil {
			_ = _glfw.contextSlot.set(0)
			return err
		}
		if err := _glfw.contextSlot.set(0); err != nil {
			return err
		}
	}
	return nil
}

func swapBuffersWGL(window *Window) error {
	if window.monitor == nil && winver.IsWindowsVistaOrGreater() {
		// DWM Composition is always enabled on Win8+
		enabled := winver.IsWindows8OrGreater()

		if !enabled {
			var err error
			enabled, err = _DwmIsCompositionEnabled()
			if err != nil {
				return err
			}
		}

		// HACK: Use DwmFlush when desktop composition is enabled
		if enabled {
			for i := 0; i < window.context.platform.interval; i++ {
				// Ignore an error from DWM functions as they might not be implemented e.g. on Proton (#2113).
				_ = _DwmFlush()
			}
		}
	}

	if err := _SwapBuffers(window.context.platform.dc); err != nil {
		return err
	}
	return nil
}

func swapIntervalWGL(window *Window, interval int) error {
	window.context.platform.interval = interval

	if window.monitor == nil && winver.IsWindowsVistaOrGreater() {
		// DWM Composition is always enabled on Win8+
		enabled := winver.IsWindows8OrGreater()

		if !enabled {
			e, err := _DwmIsCompositionEnabled()
			// Ignore an error from DWM functions as they might not be implemented e.g. on Proton (#2113).
			if err == nil {
				enabled = e
			}
		}

		// HACK: Disable WGL swap interval when desktop composition is enabled to
		//       avoid interfering with DWM vsync
		if enabled {
			interval = 0
		}
	}

	if _glfw.platformContext.EXT_swap_control {
		if err := wglSwapIntervalEXT(int32(interval)); err != nil {
			return err
		}
	}
	return nil
}

func extensionSupportedWGL(extension string) bool {
	var extensions string

	if wglGetExtensionsStringARB_Available() {
		extensions = wglGetExtensionsStringARB(wglGetCurrentDC())
	} else if wglGetExtensionsStringEXT_Available() {
		extensions = wglGetExtensionsStringEXT()
	}

	if len(extensions) == 0 {
		return false
	}

	for _, str := range strings.Split(extensions, " ") {
		if extension == str {
			return true
		}
	}
	return false
}

func getProcAddressWGL(procname string) uintptr {
	proc := wglGetProcAddress(procname)
	if proc != 0 {
		return proc
	}
	return opengl32.NewProc(procname).Addr()
}

func destroyContextWGL(window *Window) error {
	if window.context.platform.handle != 0 {
		// Ignore ERROR_BUSY. This happens when the thread is different from the context thread (#2518).
		// This is a known issue of GLFW (glfw/glfw#2239).
		// TODO: Delete the context on an appropriate thread.
		if err := wglDeleteContext(window.context.platform.handle); err != nil && !errors.Is(err, windows.ERROR_BUSY) {
			return err
		}
		window.context.platform.handle = 0
	}
	return nil
}

func initWGL() error {
	if microsoftgdk.IsXbox() {
		return fmt.Errorf("glfw: WGL is not available in Xbox")
	}

	if _glfw.platformContext.inited {
		return nil
	}

	// opengl32.dll must be loaded first. The loading state might affect Windows APIs.
	// This is needed at least before SetPixelFormat.
	if err := opengl32.Load(); err != nil {
		return err
	}

	// NOTE: A dummy context has to be created for opengl32.dll to load the
	//       OpenGL ICD, from which we can then query WGL extensions
	// NOTE: This code will accept the Microsoft GDI ICD; accelerated context
	//       creation failure occurs during manual pixel format enumeration

	dc, err := _GetDC(_glfw.platformWindow.helperWindowHandle)
	if err != nil {
		return err
	}
	pfd := _PIXELFORMATDESCRIPTOR{
		nVersion:   1,
		dwFlags:    _PFD_DRAW_TO_WINDOW | _PFD_SUPPORT_OPENGL | _PFD_DOUBLEBUFFER,
		iPixelType: _PFD_TYPE_RGBA,
		cColorBits: 24,
	}
	pfd.nSize = uint16(unsafe.Sizeof(pfd))

	format, err := _ChoosePixelFormat(dc, &pfd)
	if err != nil {
		return err
	}
	if err := _SetPixelFormat(dc, format, &pfd); err != nil {
		return err
	}

	rc, err := wglCreateContext(dc)
	if err != nil {
		return err
	}

	pdc := wglGetCurrentDC()
	prc := wglGetCurrentContext()

	if err := wglMakeCurrent(dc, rc); err != nil {
		_ = wglMakeCurrent(pdc, prc)
		_ = wglDeleteContext(rc)
		return err
	}

	// NOTE: Functions must be loaded first as they're needed to retrieve the
	//       extension string that tells us whether the functions are supported
	//
	// Interestingly, wglGetProcAddress might return 0 after extensionSupportedWGL is called.
	initWGLExtensionFunctions()

	// NOTE: WGL_ARB_extensions_string and WGL_EXT_extensions_string are not
	//       checked below as we are already using them
	_glfw.platformContext.ARB_multisample = extensionSupportedWGL("WGL_ARB_multisample")
	_glfw.platformContext.ARB_framebuffer_sRGB = extensionSupportedWGL("WGL_ARB_framebuffer_sRGB")
	_glfw.platformContext.EXT_framebuffer_sRGB = extensionSupportedWGL("WGL_EXT_framebuffer_sRGB")
	_glfw.platformContext.ARB_create_context = extensionSupportedWGL("WGL_ARB_create_context")
	_glfw.platformContext.ARB_create_context_profile = extensionSupportedWGL("WGL_ARB_create_context_profile")
	_glfw.platformContext.EXT_create_context_es2_profile = extensionSupportedWGL("WGL_EXT_create_context_es2_profile")
	_glfw.platformContext.ARB_create_context_robustness = extensionSupportedWGL("WGL_ARB_create_context_robustness")
	_glfw.platformContext.ARB_create_context_no_error = extensionSupportedWGL("WGL_ARB_create_context_no_error")
	_glfw.platformContext.EXT_swap_control = extensionSupportedWGL("WGL_EXT_swap_control")
	_glfw.platformContext.EXT_colorspace = extensionSupportedWGL("WGL_EXT_colorspace")
	_glfw.platformContext.ARB_pixel_format = extensionSupportedWGL("WGL_ARB_pixel_format")
	_glfw.platformContext.ARB_context_flush_control = extensionSupportedWGL("WGL_ARB_context_flush_control")

	if err := wglMakeCurrent(pdc, prc); err != nil {
		return err
	}
	if err := wglDeleteContext(rc); err != nil {
		return err
	}
	_glfw.platformContext.inited = true
	return nil
}

func terminateWGL() {
}

func (w *Window) createContextWGL(ctxconfig *ctxconfig, fbconfig *fbconfig) error {
	var share _HGLRC
	if ctxconfig.share != nil {
		share = ctxconfig.share.context.platform.handle
	}

	dc, err := _GetDC(w.platform.handle)
	if err != nil {
		return err
	}
	w.context.platform.dc = dc

	pixelFormat, err := w.choosePixelFormat(ctxconfig, fbconfig)
	if err != nil {
		return err
	}

	var pfd _PIXELFORMATDESCRIPTOR
	if _, err := _DescribePixelFormat(w.context.platform.dc, int32(pixelFormat), uint32(unsafe.Sizeof(pfd)), &pfd); err != nil {
		return err
	}

	if err := _SetPixelFormat(w.context.platform.dc, int32(pixelFormat), &pfd); err != nil {
		return err
	}

	if ctxconfig.client == OpenGLAPI {
		if ctxconfig.forward && !_glfw.platformContext.ARB_create_context {
			return fmt.Errorf("glfw: a forward compatible OpenGL context requested but WGL_ARB_create_context is unavailable: %w", VersionUnavailable)
		}

		if ctxconfig.profile != 0 && !_glfw.platformContext.ARB_create_context_profile {
			return fmt.Errorf("glfw: OpenGL profile requested but WGL_ARB_create_context_profile is unavailable: %w", VersionUnavailable)
		}
	} else {
		if !_glfw.platformContext.ARB_create_context || !_glfw.platformContext.ARB_create_context_profile || !_glfw.platformContext.EXT_create_context_es2_profile {
			return fmt.Errorf("glfw: OpenGL ES requested but WGL_ARB_create_context_es2_profile is unavailable: %w", APIUnavailable)
		}
	}

	if _glfw.platformContext.ARB_create_context {
		var flags int32
		var mask int32
		if ctxconfig.client == OpenGLAPI {
			if ctxconfig.forward {
				flags |= _WGL_CONTEXT_FORWARD_COMPATIBLE_BIT_ARB
			}

			if ctxconfig.profile == OpenGLCoreProfile {
				mask |= _WGL_CONTEXT_CORE_PROFILE_BIT_ARB
			} else if ctxconfig.profile == OpenGLCompatProfile {
				mask |= _WGL_CONTEXT_COMPATIBILITY_PROFILE_BIT_ARB
			}
		} else {
			mask |= _WGL_CONTEXT_ES2_PROFILE_BIT_EXT
		}

		if ctxconfig.debug {
			flags |= _WGL_CONTEXT_DEBUG_BIT_ARB
		}

		var attribs []int32
		if ctxconfig.robustness != 0 {
			if _glfw.platformContext.ARB_create_context_robustness {
				if ctxconfig.robustness == NoResetNotification {
					attribs = append(attribs, _WGL_CONTEXT_RESET_NOTIFICATION_STRATEGY_ARB, _WGL_NO_RESET_NOTIFICATION_ARB)
				} else if ctxconfig.robustness == LoseContextOnReset {
					attribs = append(attribs, _WGL_CONTEXT_RESET_NOTIFICATION_STRATEGY_ARB, _WGL_LOSE_CONTEXT_ON_RESET_ARB)
				}
				flags |= _WGL_CONTEXT_ROBUST_ACCESS_BIT_ARB
			}
		}

		if ctxconfig.release != 0 {
			if _glfw.platformContext.ARB_context_flush_control {
				if ctxconfig.release == ReleaseBehaviorNone {
					attribs = append(attribs, _WGL_CONTEXT_RELEASE_BEHAVIOR_ARB, _WGL_CONTEXT_RELEASE_BEHAVIOR_NONE_ARB)
				} else if ctxconfig.release == ReleaseBehaviorFlush {
					attribs = append(attribs, _WGL_CONTEXT_RELEASE_BEHAVIOR_ARB, _WGL_CONTEXT_RELEASE_BEHAVIOR_FLUSH_ARB)
				}
			}
		}

		if ctxconfig.noerror {
			if _glfw.platformContext.ARB_create_context_no_error {
				attribs = append(attribs, _WGL_CONTEXT_OPENGL_NO_ERROR_ARB, 1)
			}
		}

		// NOTE: Only request an explicitly versioned context when necessary, as
		//       explicitly requesting version 1.0 does not always return the
		//       highest version supported by the driver
		if ctxconfig.major != 1 || ctxconfig.minor != 0 {
			attribs = append(attribs, _WGL_CONTEXT_MAJOR_VERSION_ARB, int32(ctxconfig.major))
			attribs = append(attribs, _WGL_CONTEXT_MINOR_VERSION_ARB, int32(ctxconfig.minor))
		}

		if flags != 0 {
			attribs = append(attribs, _WGL_CONTEXT_FLAGS_ARB, flags)
		}

		if mask != 0 {
			attribs = append(attribs, _WGL_CONTEXT_PROFILE_MASK_ARB, mask)
		}

		attribs = append(attribs, 0, 0)

		var err error
		w.context.platform.handle, err = wglCreateContextAttribsARB(w.context.platform.dc, share, &attribs[0])
		if err != nil {
			return err
		}
	} else {
		var err error
		w.context.platform.handle, err = wglCreateContext(w.context.platform.dc)
		if err != nil {
			return err
		}

		if share != 0 {
			if err := wglShareLists(share, w.context.platform.handle); err != nil {
				return err
			}
		}
	}

	w.context.makeCurrent = makeContextCurrentWGL
	w.context.swapBuffers = swapBuffersWGL
	w.context.swapInterval = swapIntervalWGL
	w.context.extensionSupported = extensionSupportedWGL
	w.context.getProcAddress = getProcAddressWGL
	w.context.destroy = destroyContextWGL

	return nil
}

func getWGLContext(handle *Window) _HGLRC {
	window := handle
	if !_glfw.initialized {
		panic(NotInitialized)
	}
	if window.context.source != NativeContextAPI {
		// TODO: Should this return an error?
		return 0
	}
	return window.context.platform.handle
}
