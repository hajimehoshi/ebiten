// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"fmt"
	"math"
	"time"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"

	"github.com/hajimehoshi/ebiten/v2/internal/cocoa"
)

func initNSGL() error {
	return nil
}

func terminateNSGL() {
}

func (w *Window) createContextNSGL(ctxconfig *ctxconfig, fbconfig_ *fbconfig) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if ctxconfig.client == OpenGLESAPI {
		return fmt.Errorf("glfw: NSGL does not support OpenGL ES: %w", APIUnavailable)
	}

	if ctxconfig.major == 3 && ctxconfig.minor < 2 {
		return fmt.Errorf("glfw: NSGL does not support OpenGL version 3.0 or 3.1: %w", VersionUnavailable)
	}

	if ctxconfig.major >= 3 && !ctxconfig.forward {
		return fmt.Errorf("glfw: NSGL OpenGL 3.2+ requires a forward-compatible core profile: %w", VersionUnavailable)
	}

	if ctxconfig.major >= 3 && ctxconfig.profile != OpenGLCoreProfile {
		return fmt.Errorf("glfw: NSGL OpenGL 3.2+ requires a core profile: %w", VersionUnavailable)
	}

	// Build the pixel format attributes array.
	var attribs [40]uint32
	idx := 0

	addAttrib := func(a uint32) {
		attribs[idx] = a
		idx++
	}
	addAttribVal := func(a, v uint32) {
		attribs[idx] = a
		idx++
		attribs[idx] = v
		idx++
	}

	addAttrib(NSOpenGLPFAAccelerated)
	addAttrib(NSOpenGLPFAClosestPolicy)

	if ctxconfig.major >= 4 {
		addAttribVal(NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion4_1Core)
	} else if ctxconfig.major >= 3 {
		addAttribVal(NSOpenGLPFAOpenGLProfile, NSOpenGLProfileVersion3_2Core)
	}

	if fbconfig_.doublebuffer {
		addAttrib(NSOpenGLPFADoubleBuffer)
	}

	if fbconfig_.redBits != DontCare && fbconfig_.greenBits != DontCare && fbconfig_.blueBits != DontCare {
		colorBits := fbconfig_.redBits + fbconfig_.greenBits + fbconfig_.blueBits
		// macOS needs non-zero color size, so set reasonable values
		if colorBits == 0 {
			colorBits = 24
		} else if colorBits < 15 {
			colorBits = 15
		}
		addAttribVal(NSOpenGLPFAColorSize, uint32(colorBits))
	}

	if fbconfig_.alphaBits > 0 {
		addAttribVal(NSOpenGLPFAAlphaSize, uint32(fbconfig_.alphaBits))
	}

	if fbconfig_.depthBits > 0 {
		addAttribVal(NSOpenGLPFADepthSize, uint32(fbconfig_.depthBits))
	}

	if fbconfig_.stencilBits > 0 {
		addAttribVal(NSOpenGLPFAStencilSize, uint32(fbconfig_.stencilBits))
	}

	if ctxconfig.major <= 2 {
		if fbconfig_.auxBuffers != DontCare {
			addAttribVal(NSOpenGLPFAAuxBuffers, uint32(fbconfig_.auxBuffers))
		}

		if fbconfig_.accumRedBits != DontCare &&
			fbconfig_.accumGreenBits != DontCare &&
			fbconfig_.accumBlueBits != DontCare &&
			fbconfig_.accumAlphaBits != DontCare {
			accumBits := fbconfig_.accumRedBits + fbconfig_.accumGreenBits + fbconfig_.accumBlueBits + fbconfig_.accumAlphaBits
			addAttribVal(NSOpenGLPFAAccumSize, uint32(accumBits))
		}
	}

	if fbconfig_.samples > 0 {
		addAttribVal(NSOpenGLPFASampleBuffers, 1)
		addAttribVal(NSOpenGLPFASamples, uint32(fbconfig_.samples))
	}

	if fbconfig_.stereo {
		return fmt.Errorf("glfw: NSGL stereo rendering is deprecated: %w", FormatUnavailable)
	}

	// Terminate the attributes list.
	addAttrib(0)

	// Create the pixel format.
	pixelFormat := objc.ID(classNSOpenGLPixelFormat).Send(selAlloc).Send(selInitWithAttributes, uintptr(unsafe.Pointer(&attribs[0])))
	if pixelFormat == 0 {
		return fmt.Errorf("glfw: NSGL: failed to find a suitable pixel format: %w", FormatUnavailable)
	}

	// Create the OpenGL context.
	var share objc.ID
	if ctxconfig.share != nil {
		share = ctxconfig.share.context.platform.object
	}

	context := objc.ID(classNSOpenGLContext).Send(selAlloc).Send(selInitWithFormatShareContext, uintptr(pixelFormat), uintptr(share))
	if context == 0 {
		pixelFormat.Send(selRelease)
		return fmt.Errorf("glfw: NSGL: failed to create OpenGL context: %w", VersionUnavailable)
	}

	w.context.platform.object = context
	w.context.platform.pixelFormat = pixelFormat

	// Set surface opacity for transparent windows.
	if fbconfig_.transparent {
		var opacity int32 = 0
		context.Send(selSetValuesForParameter, uintptr(unsafe.Pointer(&opacity)), uintptr(NSOpenGLCPSurfaceOpacity))
	}

	// Enable retina support.
	if w.platform.retina {
		w.platform.view.Send(selSetWantsBestResolutionOpenGLSurface, true)
	}

	// Set the view on the context.
	context.Send(selSetView, uintptr(w.platform.view))

	w.context.makeCurrent = makeContextCurrentNSGL
	w.context.swapBuffers = swapBuffersNSGL
	w.context.swapInterval = swapIntervalNSGL
	w.context.extensionSupported = extensionSupportedNSGL
	w.context.getProcAddress = getProcAddressNSGL
	w.context.destroy = destroyContextNSGL

	return nil
}

func makeContextCurrentNSGL(window *Window) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if window != nil {
		window.context.platform.object.Send(selMakeCurrentContext)
		if err := _glfw.contextSlot.set(uintptr(unsafe.Pointer(window))); err != nil {
			return err
		}
	} else {
		objc.ID(classNSOpenGLContext).Send(selClearCurrentContext)
		if err := _glfw.contextSlot.set(0); err != nil {
			return err
		}
	}
	return nil
}

func swapBuffersNSGL(window *Window) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	// HACK: Simulate vsync with usleep as NSGL swap interval does not apply to
	// windows with a non-visible occlusion state.
	if window.platform.occluded {
		var interval int32
		window.context.platform.object.Send(selGetValuesForParameter, uintptr(unsafe.Pointer(&interval)), uintptr(NSOpenGLCPSwapInterval))
		if interval > 0 {
			const framerate = 60.0
			elapsed := float64(time.Now().UnixNano()) / float64(time.Second)
			period := 1.0 / framerate
			delay := period - math.Mod(elapsed, period)

			time.Sleep(time.Duration(delay * float64(time.Second)))
		}
	}

	window.context.platform.object.Send(selFlushBuffer)
	return nil
}

func swapIntervalNSGL(window *Window, interval int) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	value := int32(interval)
	window.context.platform.object.Send(selSetValuesForParameter, uintptr(unsafe.Pointer(&value)), uintptr(NSOpenGLCPSwapInterval))
	return nil
}

func extensionSupportedNSGL(extension string) bool {
	return false
}

func getProcAddressNSGL(procname string) uintptr {
	proc, err := purego.Dlsym(openGLFramework, procname)
	if err != nil {
		return 0
	}
	return proc
}

func destroyContextNSGL(window *Window) error {
	pool := cocoa.NSAutoreleasePool_new()
	defer pool.Release()

	if window.context.platform.pixelFormat != 0 {
		window.context.platform.pixelFormat.Send(selRelease)
		window.context.platform.pixelFormat = 0
	}

	if window.context.platform.object != 0 {
		window.context.platform.object.Send(selClearDrawable)
		window.context.platform.object.Send(selRelease)
		window.context.platform.object = 0
	}

	return nil
}
