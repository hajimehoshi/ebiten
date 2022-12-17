// SPDX-License-Identifier: Zlib
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfwwin

import (
	"fmt"
	cf "github.com/hajimehoshi/ebiten/v2/internal/corefoundation"
)

// Initialize OpenGL support
func initNSGL() error {
	if _glfw.context.framework != 0 {
		return nil
	}

	_glfw.context.framework = cf.CFBundleGetBundleWithIdentifier(cf.CFStringCreateWithCString(cf.KCFAllocatorDefault, []byte("com.apple.opengl\x00"), cf.KCFStringEncodingUTF8))

	if _glfw.context.framework == 0 {
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
	return nil
}

//GLFWbool _glfwCreateContextNSGL(_GLFWwindow* window,
//                                const _GLFWctxconfig* ctxconfig,
//                                const _GLFWfbconfig* fbconfig)
//{

//
//#define ADD_ATTRIB(a) \
//{ \
//    assert((size_t) index < sizeof(attribs) / sizeof(attribs[0])); \
//    attribs[index++] = a; \
//}
//#define SET_ATTRIB(a, v) { ADD_ATTRIB(a); ADD_ATTRIB(v); }
//
//    NSOpenGLPixelFormatAttribute attribs[40];
//    int index = 0;
//
//    ADD_ATTRIB(NSOpenGLPFAAccelerated);
//    ADD_ATTRIB(NSOpenGLPFAClosestPolicy);
//
//    if (ctxconfig->nsgl.offline)
//    {
//        ADD_ATTRIB(NSOpenGLPFAAllowOfflineRenderers);
//        // NOTE: This replaces the NSSupportsAutomaticGraphicsSwitching key in
//        //       Info.plist for unbundled applications
//        // HACK: This assumes that NSOpenGLPixelFormat will remain
//        //       a straightforward wrapper of its CGL counterpart
//        ADD_ATTRIB(kCGLPFASupportsAutomaticGraphicsSwitching);
//    }
//
//#if MAC_OS_X_VERSION_MAX_ALLOWED >= 101000
//    if (ctxconfig->major >= 4)
//    {
//        SET_ATTRIB(NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion4_1Core);
//    }
//    else
//#endif /*MAC_OS_X_VERSION_MAX_ALLOWED*/
//    if (ctxconfig->major >= 3)
//    {
//        SET_ATTRIB(NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion3_2Core);
//    }
//
//    if (ctxconfig->major <= 2)
//    {
//        if (fbconfig->auxBuffers != GLFW_DONT_CARE)
//            SET_ATTRIB(NSOpenGLPFAAuxBuffers, fbconfig->auxBuffers);
//
//        if (fbconfig->accumRedBits != GLFW_DONT_CARE &&
//            fbconfig->accumGreenBits != GLFW_DONT_CARE &&
//            fbconfig->accumBlueBits != GLFW_DONT_CARE &&
//            fbconfig->accumAlphaBits != GLFW_DONT_CARE)
//        {
//            const int accumBits = fbconfig->accumRedBits +
//                                  fbconfig->accumGreenBits +
//                                  fbconfig->accumBlueBits +
//                                  fbconfig->accumAlphaBits;
//
//            SET_ATTRIB(NSOpenGLPFAAccumSize, accumBits);
//        }
//    }
//
//    if (fbconfig->redBits != GLFW_DONT_CARE &&
//        fbconfig->greenBits != GLFW_DONT_CARE &&
//        fbconfig->blueBits != GLFW_DONT_CARE)
//    {
//        int colorBits = fbconfig->redBits +
//                        fbconfig->greenBits +
//                        fbconfig->blueBits;
//
//        // macOS needs non-zero color size, so set reasonable values
//        if (colorBits == 0)
//            colorBits = 24;
//        else if (colorBits < 15)
//            colorBits = 15;
//
//        SET_ATTRIB(NSOpenGLPFAColorSize, colorBits);
//    }
//
//    if (fbconfig->alphaBits != GLFW_DONT_CARE)
//        SET_ATTRIB(NSOpenGLPFAAlphaSize, fbconfig->alphaBits);
//
//    if (fbconfig->depthBits != GLFW_DONT_CARE)
//        SET_ATTRIB(NSOpenGLPFADepthSize, fbconfig->depthBits);
//
//    if (fbconfig->stencilBits != GLFW_DONT_CARE)
//        SET_ATTRIB(NSOpenGLPFAStencilSize, fbconfig->stencilBits);
//
//    if (fbconfig->stereo)
//    {
//#if MAC_OS_X_VERSION_MAX_ALLOWED >= 101200
//        _glfwInputError(GLFW_FORMAT_UNAVAILABLE,
//                        "NSGL: Stereo rendering is deprecated");
//        return GLFW_FALSE;
//#else
//        ADD_ATTRIB(NSOpenGLPFAStereo);
//#endif
//    }
//
//    if (fbconfig->doublebuffer)
//        ADD_ATTRIB(NSOpenGLPFADoubleBuffer);
//
//    if (fbconfig->samples != GLFW_DONT_CARE)
//    {
//        if (fbconfig->samples == 0)
//        {
//            SET_ATTRIB(NSOpenGLPFASampleBuffers, 0);
//        }
//        else
//        {
//            SET_ATTRIB(NSOpenGLPFASampleBuffers, 1);
//            SET_ATTRIB(NSOpenGLPFASamples, fbconfig->samples);
//        }
//    }
//
//    // NOTE: All NSOpenGLPixelFormats on the relevant cards support sRGB
//    //       framebuffer, so there's no need (and no way) to request it
//
//    ADD_ATTRIB(0);
//
//#undef ADD_ATTRIB
//#undef SET_ATTRIB
//
//    window->context.nsgl.pixelFormat =
//        [[NSOpenGLPixelFormat alloc] initWithAttributes:attribs];
//    if (window->context.nsgl.pixelFormat == nil)
//    {
//        _glfwInputError(GLFW_FORMAT_UNAVAILABLE,
//                        "NSGL: Failed to find a suitable pixel format");
//        return GLFW_FALSE;
//    }
//
//    NSOpenGLContext* share = nil;
//
//    if (ctxconfig->share)
//        share = ctxconfig->share->context.nsgl.object;
//
//    window->context.nsgl.object =
//        [[NSOpenGLContext alloc] initWithFormat:window->context.nsgl.pixelFormat
//                                   shareContext:share];
//    if (window->context.nsgl.object == nil)
//    {
//        _glfwInputError(GLFW_VERSION_UNAVAILABLE,
//                        "NSGL: Failed to create OpenGL context");
//        return GLFW_FALSE;
//    }
//
//    if (fbconfig->transparent)
//    {
//        GLint opaque = 0;
//        [window->context.nsgl.object setValues:&opaque
//                                  forParameter:NSOpenGLContextParameterSurfaceOpacity];
//    }
//
//    [window->ns.view setWantsBestResolutionOpenGLSurface:window->ns.retina];
//
//    [window->context.nsgl.object setView:window->ns.view];
//
//    window->context.makeCurrent = makeContextCurrentNSGL;
//    window->context.swapBuffers = swapBuffersNSGL;
//    window->context.swapInterval = swapIntervalNSGL;
//    window->context.extensionSupported = extensionSupportedNSGL;
//    window->context.getProcAddress = getProcAddressNSGL;
//    window->context.destroy = destroyContextNSGL;
//
//    return GLFW_TRUE;
//}
