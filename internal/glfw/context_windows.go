// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unsafe"

	"github.com/ebitengine/purego"
)

func checkValidContextConfig(ctxconfig *ctxconfig) error {
	if ctxconfig.source != NativeContextAPI &&
		ctxconfig.source != EGLContextAPI &&
		ctxconfig.source != OSMesaContextAPI {
		return fmt.Errorf("glfw: invalid context creation API 0x%08X: %w", ctxconfig.source, InvalidEnum)
	}

	if ctxconfig.client != NoAPI &&
		ctxconfig.client != OpenGLAPI &&
		ctxconfig.client != OpenGLESAPI {
		return fmt.Errorf("glfw: invalid client API 0x%08X: %w", ctxconfig.client, InvalidEnum)
	}

	if ctxconfig.share != nil {
		if ctxconfig.client == NoAPI || ctxconfig.share.context.client == NoAPI {
			return NoWindowContext
		}
		if ctxconfig.source != ctxconfig.share.context.source {
			return fmt.Errorf("glfw: context creation APIs do not match between contexts: %w", InvalidEnum)
		}
	}

	if ctxconfig.client == OpenGLAPI {
		if (ctxconfig.major < 1 || ctxconfig.minor < 0) ||
			(ctxconfig.major == 1 && ctxconfig.minor > 5) ||
			(ctxconfig.major == 2 && ctxconfig.minor > 1) ||
			(ctxconfig.major == 3 && ctxconfig.minor > 3) {
			// OpenGL 1.0 is the smallest valid version
			// OpenGL 1.x series ended with version 1.5
			// OpenGL 2.x series ended with version 2.1
			// OpenGL 3.x series ended with version 3.3
			// For now, let everything else through

			return fmt.Errorf("glfw: invalid OpenGL version %d.%d: %w", ctxconfig.major, ctxconfig.minor, InvalidValue)
		}

		if ctxconfig.profile != 0 {
			if ctxconfig.profile != OpenGLCoreProfile && ctxconfig.profile != OpenGLCompatProfile {
				return fmt.Errorf("glfw: invalid OpenGL profile 0x%08X: %w", ctxconfig.profile, InvalidEnum)
			}

			if ctxconfig.major <= 2 || (ctxconfig.major == 3 && ctxconfig.minor < 2) {
				// Desktop OpenGL context profiles are only defined for version 3.2
				// and above

				return fmt.Errorf("glfw: context profiles are only defined for OpenGL version 3.2 and above: %w", InvalidValue)
			}
		}

		if ctxconfig.forward && ctxconfig.major <= 2 {
			// Forward-compatible contexts are only defined for OpenGL version 3.0 and above
			return fmt.Errorf("glfw: forward-compatibility is only defined for OpenGL version 3.0 and above: %w", InvalidValue)
		}
	} else if ctxconfig.client == OpenGLESAPI {
		if ctxconfig.major < 1 || ctxconfig.minor < 0 ||
			(ctxconfig.major == 1 && ctxconfig.minor > 1) ||
			(ctxconfig.major == 2 && ctxconfig.minor > 0) {
			// OpenGL ES 1.0 is the smallest valid version
			// OpenGL ES 1.x series ended with version 1.1
			// OpenGL ES 2.x series ended with version 2.0
			// For now, let everything else through

			return fmt.Errorf("glfw: invalid OpenGL ES version %d.%d: %w", ctxconfig.major, ctxconfig.minor, InvalidValue)
		}
	}

	if ctxconfig.robustness != 0 {
		if ctxconfig.robustness != NoResetNotification && ctxconfig.robustness != LoseContextOnReset {
			return fmt.Errorf("glfw: invalid context robustness mode 0x%08X: %w", ctxconfig.robustness, InvalidEnum)
		}
	}

	if ctxconfig.release != 0 {
		if ctxconfig.release != ReleaseBehaviorNone && ctxconfig.release != ReleaseBehaviorFlush {
			return fmt.Errorf("glfw: invalid context release behavior 0x%08X: %w", ctxconfig.release, InvalidEnum)
		}
	}

	return nil
}

func chooseFBConfig(desired *fbconfig, alternatives []*fbconfig) *fbconfig {
	leastMissing := math.MaxInt32
	leastColorDiff := math.MaxInt32
	leastExtraDiff := math.MaxInt32

	var closest *fbconfig
	for _, current := range alternatives {
		if desired.stereo && !current.stereo {
			// Stereo is a hard constraint
			continue
		}

		// Count number of missing buffers
		missing := 0

		if desired.alphaBits > 0 && current.alphaBits == 0 {
			missing++
		}

		if desired.depthBits > 0 && current.depthBits == 0 {
			missing++
		}

		if desired.stencilBits > 0 && current.stencilBits == 0 {
			missing++
		}

		if desired.auxBuffers > 0 &&
			current.auxBuffers < desired.auxBuffers {
			missing += desired.auxBuffers - current.auxBuffers
		}

		if desired.samples > 0 && current.samples == 0 {
			// Technically, several multisampling buffers could be
			// involved, but that's a lower level implementation detail and
			// not important to us here, so we count them as one
			missing++
		}

		if desired.transparent != current.transparent {
			missing++
		}

		// These polynomials make many small channel size differences matter
		// less than one large channel size difference

		// Calculate color channel size difference value
		colorDiff := 0

		if desired.redBits != DontCare {
			colorDiff += (desired.redBits - current.redBits) *
				(desired.redBits - current.redBits)
		}

		if desired.greenBits != DontCare {
			colorDiff += (desired.greenBits - current.greenBits) *
				(desired.greenBits - current.greenBits)
		}

		if desired.blueBits != DontCare {
			colorDiff += (desired.blueBits - current.blueBits) *
				(desired.blueBits - current.blueBits)
		}

		// Calculate non-color channel size difference value
		extraDiff := 0

		if desired.alphaBits != DontCare {
			extraDiff += (desired.alphaBits - current.alphaBits) *
				(desired.alphaBits - current.alphaBits)
		}

		if desired.depthBits != DontCare {
			extraDiff += (desired.depthBits - current.depthBits) *
				(desired.depthBits - current.depthBits)
		}

		if desired.stencilBits != DontCare {
			extraDiff += (desired.stencilBits - current.stencilBits) *
				(desired.stencilBits - current.stencilBits)
		}

		if desired.accumRedBits != DontCare {
			extraDiff += (desired.accumRedBits - current.accumRedBits) *
				(desired.accumRedBits - current.accumRedBits)
		}

		if desired.accumGreenBits != DontCare {
			extraDiff += (desired.accumGreenBits - current.accumGreenBits) *
				(desired.accumGreenBits - current.accumGreenBits)
		}

		if desired.accumBlueBits != DontCare {
			extraDiff += (desired.accumBlueBits - current.accumBlueBits) *
				(desired.accumBlueBits - current.accumBlueBits)
		}

		if desired.accumAlphaBits != DontCare {
			extraDiff += (desired.accumAlphaBits - current.accumAlphaBits) *
				(desired.accumAlphaBits - current.accumAlphaBits)
		}

		if desired.samples != DontCare {
			extraDiff += (desired.samples - current.samples) *
				(desired.samples - current.samples)
		}

		if desired.sRGB && !current.sRGB {
			extraDiff++
		}

		// Figure out if the current one is better than the best one found so far
		// Least number of missing buffers is the most important heuristic,
		// then color buffer size match and lastly size match for other buffers

		if missing < leastMissing {
			closest = current
		} else if missing == leastMissing {
			if (colorDiff < leastColorDiff) || (colorDiff == leastColorDiff && extraDiff < leastExtraDiff) {
				closest = current
			}
		}

		if current == closest {
			leastMissing = missing
			leastColorDiff = colorDiff
			leastExtraDiff = extraDiff
		}
	}

	return closest
}

func (w *Window) refreshContextAttribs(ctxconfig *ctxconfig) (ferr error) {
	const (
		GL_COLOR_BUFFER_BIT                    = 0x00004000
		GL_CONTEXT_COMPATIBILITY_PROFILE_BIT   = 0x00000002
		GL_CONTEXT_CORE_PROFILE_BIT            = 0x00000001
		GL_CONTEXT_FLAG_DEBUG_BIT              = 0x00000002
		GL_CONTEXT_FLAG_FORWARD_COMPATIBLE_BIT = 0x00000001
		GL_CONTEXT_FLAG_NO_ERROR_BIT_KHR       = 0x00000008
		GL_CONTEXT_FLAGS                       = 0x821E
		GL_CONTEXT_PROFILE_MASK                = 0x9126
		GL_CONTEXT_RELEASE_BEHAVIOR            = 0x82FB
		GL_CONTEXT_RELEASE_BEHAVIOR_FLUSH      = 0x82FC
		GL_LOSE_CONTEXT_ON_RESET_ARB           = 0x8252
		GL_NO_RESET_NOTIFICATION_ARB           = 0x8261
		GL_NONE                                = 0
		GL_RESET_NOTIFICATION_STRATEGY_ARB     = 0x8256
		GL_VERSION                             = 0x1F02
	)

	w.context.source = ctxconfig.source
	w.context.client = OpenGLAPI

	// In Ebitengine, only one window is created.
	// Always assume that the current context is not set.
	defer func() {
		err := (*Window)(nil).MakeContextCurrent()
		if ferr == nil {
			ferr = err
		}
	}()
	if err := w.MakeContextCurrent(); err != nil {
		return err
	}

	getIntegerv := w.context.getProcAddress("glGetIntegerv")
	getString := w.context.getProcAddress("glGetString")
	if getIntegerv == 0 || getString == 0 {
		return fmt.Errorf("glfw: entry point retrieval is broken: %w", PlatformError)
	}

	r, _, _ := purego.SyscallN(getString, GL_VERSION)
	version := bytePtrToString((*byte)(unsafe.Pointer(r)))
	if version == "" {
		if ctxconfig.client == OpenGLAPI {
			return fmt.Errorf("glfw: OpenGL version string retrieval is broken: %w", PlatformError)
		} else {
			return fmt.Errorf("glfw: OpenGL ES version string retrieval is broken: %w", PlatformError)
		}
	}

	for _, prefix := range []string{
		"OpenGL ES-CM ",
		"OpenGL ES-CL ",
		"OpenGL ES "} {
		if strings.HasPrefix(version, prefix) {
			version = version[len(prefix):]
			w.context.client = OpenGLESAPI
			break
		}
	}

	m := regexp.MustCompile(`^(\d+)(\.(\d+)(\.(\d+))?)?`).FindStringSubmatch(version)
	if m == nil {
		if w.context.client == OpenGLAPI {
			return fmt.Errorf("glfw: no version found in OpenGL version string: %w", PlatformError)
		} else {
			return fmt.Errorf("glfw: no version found in OpenGL ES version string: %w", PlatformError)
		}
	}
	w.context.major, _ = strconv.Atoi(m[1])
	w.context.minor, _ = strconv.Atoi(m[3])
	w.context.revision, _ = strconv.Atoi(m[5])

	if w.context.major < ctxconfig.major || (w.context.major == ctxconfig.major && w.context.minor < ctxconfig.minor) {
		// The desired OpenGL version is greater than the actual version
		// This only happens if the machine lacks {GLX|WGL}_ARB_create_context
		// /and/ the user has requested an OpenGL version greater than 1.0

		// For API consistency, we emulate the behavior of the
		// {GLX|WGL}_ARB_create_context extension and fail here

		if w.context.client == OpenGLAPI {
			return fmt.Errorf("glfw: requested OpenGL version %d.%d, got version %d.%d: %w", ctxconfig.major, ctxconfig.minor, w.context.major, w.context.minor, VersionUnavailable)
		} else {
			return fmt.Errorf("glfw: requested OpenGL ES version %d.%d, got version %d.%d: %w", ctxconfig.major, ctxconfig.minor, w.context.major, w.context.minor, VersionUnavailable)
		}
	}

	if w.context.major >= 3 {
		// OpenGL 3.0+ uses a different function for extension string retrieval
		// We cache it here instead of in glfwExtensionSupported mostly to alert
		// users as early as possible that their build may be broken

		glGetStringi := w.context.getProcAddress("glGetStringi")
		if glGetStringi == 0 {
			return fmt.Errorf("glfw: entry point retrieval is broken: %w", PlatformError)
		}
	}

	if w.context.client == OpenGLAPI {
		// Read back context flags (OpenGL 3.0 and above)
		if w.context.major >= 3 {
			var flags int32
			_, _, _ = purego.SyscallN(getIntegerv, GL_CONTEXT_FLAGS, uintptr(unsafe.Pointer(&flags)))

			if flags&GL_CONTEXT_FLAG_FORWARD_COMPATIBLE_BIT != 0 {
				w.context.forward = true
			}

			if flags&GL_CONTEXT_FLAG_DEBUG_BIT != 0 {
				w.context.debug = true
			} else {
				ok, err := w.ExtensionSupported("GL_ARB_debug_output")
				if err != nil {
					return err
				}
				if ok && ctxconfig.debug {
					// HACK: This is a workaround for older drivers (pre KHR_debug)
					//       not setting the debug bit in the context flags for
					//       debug contexts
					w.context.debug = true
				}
			}

			if flags&GL_CONTEXT_FLAG_NO_ERROR_BIT_KHR != 0 {
				w.context.noerror = true
			}
		}

		// Read back OpenGL context profile (OpenGL 3.2 and above)
		if w.context.major >= 4 || (w.context.major == 3 && w.context.minor >= 2) {
			var mask int32
			_, _, _ = purego.SyscallN(getIntegerv, GL_CONTEXT_PROFILE_MASK, uintptr(unsafe.Pointer(&mask)))

			if mask&GL_CONTEXT_COMPATIBILITY_PROFILE_BIT != 0 {
				w.context.profile = OpenGLCompatProfile
			} else if mask&GL_CONTEXT_CORE_PROFILE_BIT != 0 {
				w.context.profile = OpenGLCoreProfile
			} else {
				ok, err := w.ExtensionSupported("GL_ARB_compatibility")
				if err != nil {
					return err
				}
				if ok {
					// HACK: This is a workaround for the compatibility profile bit
					//       not being set in the context flags if an OpenGL 3.2+
					//       context was created without having requested a specific
					//       version
					w.context.profile = OpenGLCompatProfile
				}
			}
		}

		// Read back robustness strategy
		ok, err := w.ExtensionSupported("GL_ARB_robustness")
		if err != nil {
			return err
		}
		if ok {
			// NOTE: We avoid using the context flags for detection, as they are
			//       only present from 3.0 while the extension applies from 1.1

			var strategy int32
			_, _, _ = purego.SyscallN(getIntegerv, GL_RESET_NOTIFICATION_STRATEGY_ARB, uintptr(unsafe.Pointer(&strategy)))

			if strategy == GL_LOSE_CONTEXT_ON_RESET_ARB {
				w.context.robustness = LoseContextOnReset
			} else if strategy == GL_NO_RESET_NOTIFICATION_ARB {
				w.context.robustness = NoResetNotification
			}
		}
	} else {
		// Read back robustness strategy
		ok, err := w.ExtensionSupported("GL_EXT_robustness")
		if err != nil {
			return err
		}
		if ok {
			// NOTE: The values of these constants match those of the OpenGL ARB
			//       one, so we can reuse them here

			var strategy int32
			_, _, _ = purego.SyscallN(getIntegerv, GL_RESET_NOTIFICATION_STRATEGY_ARB, uintptr(unsafe.Pointer(&strategy)))

			if strategy == GL_LOSE_CONTEXT_ON_RESET_ARB {
				w.context.robustness = LoseContextOnReset
			} else if strategy == GL_NO_RESET_NOTIFICATION_ARB {
				w.context.robustness = NoResetNotification
			}
		}
	}

	ok, err := w.ExtensionSupported("GL_KHR_context_flush_control")
	if err != nil {
		return err
	}
	if ok {
		var behavior int32
		_, _, _ = purego.SyscallN(getIntegerv, GL_CONTEXT_RELEASE_BEHAVIOR, uintptr(unsafe.Pointer(&behavior)))

		if behavior == GL_NONE {
			w.context.release = ReleaseBehaviorNone
		} else if behavior == GL_CONTEXT_RELEASE_BEHAVIOR_FLUSH {
			w.context.release = ReleaseBehaviorFlush
		}
	}

	// Clearing the front buffer to black to avoid garbage pixels left over from
	// previous uses of our bit of VRAM
	glClear := w.context.getProcAddress("glClear")
	_, _, _ = purego.SyscallN(glClear, GL_COLOR_BUFFER_BIT)

	if w.doublebuffer {
		if err := w.context.swapBuffers(w); err != nil {
			return err
		}
	}

	return nil
}

func (w *Window) MakeContextCurrent() error {
	if !_glfw.initialized {
		return NotInitialized
	}

	ptr, err := _glfw.contextSlot.get()
	if err != nil {
		return err
	}
	previous := (*Window)(unsafe.Pointer(ptr))

	if w != nil && w.context.client == NoAPI {
		return fmt.Errorf("glfw: cannot make current with a window that has no OpenGL or OpenGL ES context: %w", NoWindowContext)
	}

	if previous != nil {
		if w == nil || w.context.source != previous.context.source {
			if err := previous.context.makeCurrent(nil); err != nil {
				return err
			}
		}
	}

	if w != nil {
		if err := w.context.makeCurrent(w); err != nil {
			return err
		}
	}
	return nil
}

func GetCurrentContext() (*Window, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	ptr, err := _glfw.contextSlot.get()
	if err != nil {
		return nil, err
	}
	return (*Window)(unsafe.Pointer(ptr)), nil
}

func (w *Window) SwapBuffers() error {
	if !_glfw.initialized {
		return NotInitialized
	}

	if w.context.client == NoAPI {
		return fmt.Errorf("glfw: cannot swap buffers of a window that has no OpenGL or OpenGL ES context: %w", NoWindowContext)
	}

	if err := w.context.swapBuffers(w); err != nil {
		return err
	}
	return nil
}

func (w *Window) SwapInterval(interval int) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	if err := w.context.swapInterval(w, interval); err != nil {
		return err
	}
	return nil
}

func (w *Window) ExtensionSupported(extension string) (bool, error) {
	const (
		GL_EXTENSIONS     = 0x1F03
		GL_NUM_EXTENSIONS = 0x821D
	)

	if !_glfw.initialized {
		return false, NotInitialized
	}

	if w.context.major >= 3 {
		// Check if extension is in the modern OpenGL extensions string list

		glGetIntegerv := w.context.getProcAddress("glGetIntegerv")
		var count int32
		_, _, _ = purego.SyscallN(glGetIntegerv, GL_NUM_EXTENSIONS, uintptr(unsafe.Pointer(&count)))

		glGetStringi := w.context.getProcAddress("glGetStringi")
		for i := 0; i < int(count); i++ {
			r, _, _ := purego.SyscallN(glGetStringi, GL_EXTENSIONS, uintptr(i))
			if r == 0 {
				return false, fmt.Errorf("glfw: extension string retrieval is broken: %w", PlatformError)
			}

			en := bytePtrToString((*byte)(unsafe.Pointer(r)))
			if en == extension {
				return true, nil
			}
		}
	} else {
		// Check if extension is in the old style OpenGL extensions string

		glGetString := w.context.getProcAddress("glGetString")
		r, _, _ := purego.SyscallN(glGetString, GL_EXTENSIONS)
		if r == 0 {
			return false, fmt.Errorf("glfw: extension string retrieval is broken: %w", PlatformError)
		}

		extensions := bytePtrToString((*byte)(unsafe.Pointer(r)))
		for _, str := range strings.Split(extensions, " ") {
			if str == extension {
				return true, nil
			}
		}
	}

	// Check if extension is in the platform-specific string
	return w.context.extensionSupported(extension), nil
}

// bytePtrToString takes a pointer to a sequence of text and returns the corresponding string.
// If the pointer is nil, it returns the empty string. It assumes that the text sequence is
// terminated at a zero byte; if the zero byte is not present, the program may crash.
// It is copied from golang.org/x/sys/windows/syscall.go for use on macOS, Linux and Windows
func bytePtrToString(p *byte) string {
	if p == nil {
		return ""
	}
	if *p == 0 {
		return ""
	}

	// Find NUL terminator.
	n := 0
	for ptr := unsafe.Pointer(p); *(*byte)(ptr) != 0; n++ {
		ptr = unsafe.Add(ptr, 1)
	}

	return unsafe.String(p, n)
}
