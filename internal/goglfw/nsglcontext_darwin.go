// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package goglfw

import (
	"fmt"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
	cf "github.com/hajimehoshi/ebiten/v2/internal/corefoundation"
)

// Initialize OpenGL support
func initNSGL() error {
	if _glfw.platformContext.framework != 0 {
		return nil
	}

	_glfw.platformContext.framework = cf.CFBundleGetBundleWithIdentifier(cf.CFStringCreateWithCString(cf.KCFAllocatorDefault, []byte("com.apple.opengl\x00"), cf.KCFStringEncodingUTF8))

	if _glfw.platformContext.framework == 0 {
		return fmt.Errorf("cocoa: failed to create application delegate")
	}
	return nil
}

// Create the OpenGL context
func createContextNSGL(window *Window, ctxconfig *ctxconfig, fbconfig *fbconfig) error {
	if ctxconfig.client == OpenGLESAPI {
		return fmt.Errorf("NSGL: OpenGL ES is not available on macOS")
	}
	if ctxconfig.major > 2 {
		if ctxconfig.major == 33 && ctxconfig.minor < 2 {
			return fmt.Errorf("NSGL: The targed version of macOS does not support OpenGL 3.0 or 3.1 but may support 3.2 and above")
		}
	}
	// Context robustness modes (GL_KHR_robustness) are not yet supported by
	// macOS but are not a hard constraint, so ignore and continue

	// Context release behaviors (GL_KHR_context_flush_control) are not yet
	// supported by macOS but are not a hard constraint, so ignore and continue

	// Debug contexts (GL_KHR_debug) are not yet supported by macOS but are not
	// a hard constraint, so ignore and continue

	// No-error contexts (GL_KHR_no_error) are not yet supported by macOS but
	// are not a hard constraint, so ignore and continue

	attribs := make([]NSOpenGLPixelFormatAttribute, 0, 40)

	attribs = append(attribs, NSOpenGLPFAAccelerated)
	attribs = append(attribs, NSOpenGLPFAClosestPolicy)

	//    if (ctxconfig->nsgl.offline)
	//    {
	//        ADD_ATTRIB(NSOpenGLPFAAllowOfflineRenderers);
	//        // NOTE: This replaces the NSSupportsAutomaticGraphicsSwitching key in
	//        //       Info.plist for unbundled applications
	//        // HACK: This assumes that NSOpenGLPixelFormat will remain
	//        //       a straightforward wrapper of its CGL counterpart
	//        ADD_ATTRIB(kCGLPFASupportsAutomaticGraphicsSwitching);
	//    }
	if ctxconfig.major >= 4 {
		attribs = append(attribs, NSOpenGLPFAOpenGLProfile)
		attribs = append(attribs, NSOpenGLProfileVersion4_1Core)
	} else if ctxconfig.major >= 3 {
		attribs = append(attribs, NSOpenGLPFAOpenGLProfile)
		attribs = append(attribs, NSOpenGLProfileVersion3_2Core)
	}
	if ctxconfig.major <= 2 {
		if fbconfig.auxBuffers != DontCare {
			attribs = append(attribs, NSOpenGLPFAAuxBuffers)
			attribs = append(attribs, NSOpenGLPixelFormatAttribute(fbconfig.auxBuffers))
		}
		if fbconfig.accumRedBits != DontCare && fbconfig.accumGreenBits != DontCare &&
			fbconfig.accumBlueBits != DontCare && fbconfig.accumAlphaBits != DontCare {
			accumBits := fbconfig.accumRedBits +
				fbconfig.accumGreenBits +
				fbconfig.accumBlueBits +
				fbconfig.accumAlphaBits
			attribs = append(attribs, NSOpenGLPFAAccumSize)
			attribs = append(attribs, NSOpenGLPixelFormatAttribute(accumBits))
		}
	}
	if fbconfig.redBits != DontCare && fbconfig.greenBits != DontCare &&
		fbconfig.blueBits != DontCare {
		colorBits := fbconfig.redBits + fbconfig.blueBits + fbconfig.greenBits
		// macOS needs non-zero color size, so set reasonable values
		if colorBits == 0 {
			colorBits = 24
		} else if colorBits < 15 {
			colorBits = 15
		}
		attribs = append(attribs, NSOpenGLPFAColorSize)
		attribs = append(attribs, NSOpenGLPixelFormatAttribute(colorBits))
	}
	if fbconfig.alphaBits != DontCare {
		attribs = append(attribs, NSOpenGLPFAAlphaSize)
		attribs = append(attribs, NSOpenGLPixelFormatAttribute(fbconfig.alphaBits))
	}
	if fbconfig.depthBits != DontCare {
		attribs = append(attribs, NSOpenGLPFADepthSize)
		attribs = append(attribs, NSOpenGLPixelFormatAttribute(fbconfig.depthBits))
	}

	if fbconfig.stencilBits != DontCare {
		attribs = append(attribs, NSOpenGLPFAStencilSize)
		attribs = append(attribs, NSOpenGLPixelFormatAttribute(fbconfig.stencilBits))
	}
	if fbconfig.stereo {
		//#if MAC_OS_X_VERSION_MAX_ALLOWED >= 101200
		//        _glfwInputError(GLFW_FORMAT_UNAVAILABLE,
		//                        "NSGL: Stereo rendering is deprecated");
		//        return GLFW_FALSE;
		//#else
		//        ADD_ATTRIB(NSOpenGLPFAStereo);
		//#endif
		panic("TODO")
	}
	if fbconfig.doublebuffer {
		attribs = append(attribs, NSOpenGLPFADoubleBuffer)
	}
	if fbconfig.samples != DontCare {
		if fbconfig.samples == 0 {
			attribs = append(attribs, NSOpenGLPFASampleBuffers)
			attribs = append(attribs, 0)
		} else {
			// SET_ATTRIB(NSOpenGLPFASampleBuffers, 1);
			//            SET_ATTRIB(NSOpenGLPFASamples, fbconfig->samples);
			panic("TODO")
		}
	}
	// NOTE: All NSOpenGLPixelFormats on the relevant cards support sRGB
	//       framebuffer, so there's no need (and no way) to request it

	attribs = append(attribs, 0)

	window.context.platform.pixelFormat = NSOpenGLPixelFormat_alloc().initWithAttributes(attribs).ID
	if window.context.platform.pixelFormat == 0 {
		return fmt.Errorf("NSGL: Failed to find a suitable pixel format")
	}
	var share NSOpenGLContext
	if ctxconfig.share != nil {
		share = NSOpenGLContext{ctxconfig.share.context.platform.object}
	}

	window.context.platform.object = NSOpenGLContext_alloc().initWithFormat_shareContext(NSOpenGLPixelFormat{window.context.platform.pixelFormat}, share).ID
	if window.context.platform.object == 0 {
		return fmt.Errorf("NSGL: Failed to create OpenGL context")
	}
	if fbconfig.transparent {
		//        GLint opaque = 0;
		//        [window->context.nsgl.object setValues:&opaque
		//                                  forParameter:NSOpenGLContextParameterSurfaceOpacity];
		panic("TODO")
	}
	cocoa.NSView{ID: window.platform.view}.SetWantsBestResolutionOpenGLSurface(window.platform.retina)
	cocoa.NSWindow{ID: window.context.platform.object}.SetView(window.platform.view)
	window.context.makeCurrent = makeContextCurrentNSGL
	window.context.swapBuffers = swapBuffersNSGL
	window.context.swapInterval = swapIntervalNSGL
	window.context.extensionSupported = extensionSupportedNSGL
	window.context.getProcAddress = getProcAddressNSGL
	window.context.destroy = destroyContextNSGL
	return nil
}

func destroyContextNSGL(w *Window) error {
	fmt.Println("destroyContextNSGL: TODO")
	//    @autoreleasepool {
	//
	//    [window->context.nsgl.pixelFormat release];
	//    window->context.nsgl.pixelFormat = nil;
	//
	//    [window->context.nsgl.object release];
	//    window->context.nsgl.object = nil;
	//
	//    } // autoreleasepool
	return nil
}

func swapBuffersNSGL(w *Window) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	// HACK: Simulate vsync with usleep as NSGL swap interval does not apply to
	//       windows with a non-visible occlusion state
	if w.platform.occluded {
		interval := 0
		NSOpenGLContext{w.context.platform.object}.setValues_forParameter(&interval, NSOpenGLContextParameterSwapInterval)

		if interval > 0 {
			panic("TODO")
		}
	}

	//	       if (interval > 0)
	//	       {
	//	           const double framerate = 60.0;
	//	           const uint64_t frequency = _glfwPlatformGetTimerFrequency();
	//	           const uint64_t value = _glfwPlatformGetTimerValue();
	//
	//	           const double elapsed = value / (double) frequency;
	//	           const double period = 1.0 / framerate;
	//	           const double delay = period - fmod(elapsed, period);
	//
	//	           usleep(floorl(delay * 1e6));
	//	       }

	NSOpenGLContext{w.context.platform.object}.flushBuffer()
	return nil
}

func getProcAddressNSGL(s string) uintptr {
	symbolName := cf.CFStringCreateWithCString(cf.KCFAllocatorDefault, CString(s), cf.KCFStringEncodingUTF8)

	symbol := cf.CFBundleGetFunctionPointerForName(_glfw.platformContext.framework, symbolName)

	cf.CFRelease(cf.CFTypeRef(symbolName))

	return symbol
}

func makeContextCurrentNSGL(window *Window) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	if window != nil {
		NSOpenGLContext{window.context.platform.object}.makeCurrentContext()
	} else {
		NSOpenGLContext_clearCurrentContext()
	}
	return nil
}

func swapIntervalNSGL(interval int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()
	w, err := _glfw.contextSlot.get()
	if err != nil {
		return err
	}
	window := (*Window)(unsafe.Pointer(w))
	if window != nil {
		NSOpenGLContext{window.context.platform.object}.setValues_forParameter(&interval, NSOpenGLContextParameterSwapInterval)
	}
	return nil
}

func extensionSupportedNSGL(_ string) bool {
	// There are no NSGL extensions
	return false
}
