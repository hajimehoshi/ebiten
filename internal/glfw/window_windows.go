// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2012 Torsten Walluhn <tw@mad-cad.net>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"unsafe"
)

func (w *Window) inputWindowFocus(focused bool) {
	if w.callbacks.focus != nil {
		w.callbacks.focus(w, focused)
	}

	if !focused {
		for key := Key(0); key <= KeyLast; key++ {
			if w.keys[key] == Press {
				scancode := platformGetKeyScancode(key)
				w.inputKey(key, scancode, Release, 0)
			}
		}
		for button := MouseButton(0); button <= MouseButtonLast; button++ {
			if w.mouseButtons[button] == Press {
				w.inputMouseClick(button, Release, 0)
			}
		}
	}
}

func (w *Window) inputWindowPos(x, y int) {
	if w.callbacks.pos != nil {
		w.callbacks.pos(w, x, y)
	}
}

func (w *Window) inputWindowSize(width, height int) {
	if w.callbacks.size != nil {
		w.callbacks.size(w, width, height)
	}
}

func (w *Window) inputWindowIconify(iconified bool) {
	if w.callbacks.iconify != nil {
		w.callbacks.iconify(w, iconified)
	}
}

func (w *Window) inputWindowMaximize(maximized bool) {
	if w.callbacks.maximize != nil {
		w.callbacks.maximize(w, maximized)
	}
}

func (w *Window) inputFramebufferSize(width, height int) {
	if w.callbacks.fbsize != nil {
		w.callbacks.fbsize(w, width, height)
	}
}

func (w *Window) inputWindowContentScale(xscale, yscale float32) {
	if w.callbacks.scale != nil {
		w.callbacks.scale(w, xscale, yscale)
	}
}

func (w *Window) inputWindowDamage() {
	if w.callbacks.refresh != nil {
		w.callbacks.refresh(w)
	}
}

func (w *Window) inputWindowCloseRequest() {
	w.shouldClose = true

	if w.callbacks.close != nil {
		w.callbacks.close(w)
	}
}

func (w *Window) inputWindowMonitor(monitor *Monitor) {
	w.monitor = monitor
}

func CreateWindow(width, height int, title string, monitor *Monitor, share *Window) (window *Window, ferr error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}

	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("glfw: invalid window size %dx%d: %w", width, height, InvalidValue)
	}

	fbconfig := _glfw.hints.framebuffer
	ctxconfig := _glfw.hints.context
	wndconfig := _glfw.hints.window

	wndconfig.width = width
	wndconfig.height = height
	wndconfig.title = title
	ctxconfig.share = share

	if err := checkValidContextConfig(&ctxconfig); err != nil {
		return nil, err
	}

	window = &Window{
		videoMode: VidMode{
			Width:       width,
			Height:      height,
			RedBits:     fbconfig.redBits,
			GreenBits:   fbconfig.greenBits,
			BlueBits:    fbconfig.blueBits,
			RefreshRate: _glfw.hints.refreshRate,
		},

		monitor:          monitor,
		resizable:        wndconfig.resizable,
		decorated:        wndconfig.decorated,
		autoIconify:      wndconfig.autoIconify,
		floating:         wndconfig.floating,
		focusOnShow:      wndconfig.focusOnShow,
		mousePassthrough: wndconfig.mousePassthrough,
		cursorMode:       CursorNormal,

		doublebuffer: fbconfig.doublebuffer,

		minwidth:  DontCare,
		minheight: DontCare,
		maxwidth:  DontCare,
		maxheight: DontCare,
		numer:     DontCare,
		denom:     DontCare,
	}
	defer func() {
		if ferr != nil {
			_ = window.Destroy()
		}
	}()
	_glfw.windows = append(_glfw.windows, window)

	// Open the actual window and create its context
	if err := window.platformCreateWindow(&wndconfig, &ctxconfig, &fbconfig); err != nil {
		return nil, err
	}

	return window, nil
}

func defaultWindowHints() error {
	if !_glfw.initialized {
		return NotInitialized
	}

	// The default is OpenGL with minimum version 1.0
	_glfw.hints.context = ctxconfig{
		client: NoAPI, // This is different from the original GLFW, which uses OpenGLAPI by default.
		source: NativeContextAPI,
		major:  1,
		minor:  0,
	}

	// The default is a focused, visible, resizable window with decorations
	_glfw.hints.window = wndconfig{
		resizable:    true,
		visible:      true,
		decorated:    true,
		focused:      true,
		autoIconify:  true,
		centerCursor: true,
		focusOnShow:  true,
	}

	// The default is 24 bits of color, 24 bits of depth and 8 bits of stencil,
	// double buffered
	_glfw.hints.framebuffer = fbconfig{
		redBits:      8,
		greenBits:    8,
		blueBits:     8,
		alphaBits:    8,
		depthBits:    24,
		stencilBits:  8,
		doublebuffer: true,
	}

	// The default is to select the highest available refresh rate
	_glfw.hints.refreshRate = DontCare

	return nil
}

func WindowHint(hint Hint, value int) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	switch hint {
	case RedBits:
		_glfw.hints.framebuffer.redBits = value
	case GreenBits:
		_glfw.hints.framebuffer.greenBits = value
	case BlueBits:
		_glfw.hints.framebuffer.blueBits = value
	case AlphaBits:
		_glfw.hints.framebuffer.alphaBits = value
	case DepthBits:
		_glfw.hints.framebuffer.depthBits = value
	case StencilBits:
		_glfw.hints.framebuffer.stencilBits = value
	case AccumRedBits:
		_glfw.hints.framebuffer.accumRedBits = value
	case AccumGreenBits:
		_glfw.hints.framebuffer.accumGreenBits = value
	case AccumBlueBits:
		_glfw.hints.framebuffer.accumBlueBits = value
	case AccumAlphaBits:
		_glfw.hints.framebuffer.accumAlphaBits = value
	case AuxBuffers:
		_glfw.hints.framebuffer.auxBuffers = value
	case Stereo:
		_glfw.hints.framebuffer.stereo = intToBool(value)
	case DoubleBuffer:
		_glfw.hints.framebuffer.doublebuffer = intToBool(value)
	case TransparentFramebuffer:
		_glfw.hints.framebuffer.transparent = intToBool(value)
	case Samples:
		_glfw.hints.framebuffer.samples = value
	case SRGBCapable:
		_glfw.hints.framebuffer.sRGB = intToBool(value)
	case Resizable:
		_glfw.hints.window.resizable = intToBool(value)
	case Decorated:
		_glfw.hints.window.decorated = intToBool(value)
	case Focused:
		_glfw.hints.window.focused = intToBool(value)
	case AutoIconify:
		_glfw.hints.window.autoIconify = intToBool(value)
	case Floating:
		_glfw.hints.window.floating = intToBool(value)
	case Maximized:
		_glfw.hints.window.maximized = intToBool(value)
	case Visible:
		_glfw.hints.window.visible = intToBool(value)
	case ScaleToMonitor:
		_glfw.hints.window.scaleToMonitor = intToBool(value)
	case CenterCursor:
		_glfw.hints.window.centerCursor = intToBool(value)
	case FocusOnShow:
		_glfw.hints.window.focusOnShow = intToBool(value)
	case MousePassthrough:
		_glfw.hints.window.mousePassthrough = intToBool(value)
	case ClientAPI:
		_glfw.hints.context.client = value
	case ContextCreationAPI:
		_glfw.hints.context.source = value
	case ContextVersionMajor:
		_glfw.hints.context.major = value
	case ContextVersionMinor:
		_glfw.hints.context.minor = value
	case ContextRobustness:
		_glfw.hints.context.robustness = value
	case OpenGLForwardCompat:
		_glfw.hints.context.forward = intToBool(value)
	case OpenGLDebugContext:
		_glfw.hints.context.debug = intToBool(value)
	case ContextNoError:
		_glfw.hints.context.noerror = intToBool(value)
	case OpenGLProfile:
		_glfw.hints.context.profile = value
	case ContextReleaseBehavior:
		_glfw.hints.context.release = value
	case RefreshRate:
		_glfw.hints.refreshRate = value
	default:
		return fmt.Errorf("glfw: invalid window hint 0x%08X: %w", hint, InvalidEnum)
	}
	return nil
}

// WindowHintString is not implemented.
func WindowHintString(hint Hint, value string) error {
	// Do nothing.
	return nil
}

func (w *Window) Destroy() error {
	if !_glfw.initialized {
		return NotInitialized
	}

	// Allow closing of NULL (to match the behavior of free)
	if w == nil {
		return nil
	}

	// Clear all callbacks to avoid exposing a half torn-down w object
	// TODO: Clear w.callbacks

	// The w's context must not be current on another thread when the
	// w is destroyed
	current, err := _glfw.contextSlot.get()
	if err != nil {
		return err
	}
	if uintptr(unsafe.Pointer(w)) == current {
		if err := (*Window)(nil).MakeContextCurrent(); err != nil {
			return err
		}
	}

	for i, window := range _glfw.windows {
		if window == w {
			copy(_glfw.windows[i:], _glfw.windows[i+1:])
			_glfw.windows[len(_glfw.windows)-1] = nil
			_glfw.windows = _glfw.windows[:len(_glfw.windows)-1]
			break
		}
	}

	if err := w.platformDestroyWindow(); err != nil {
		return err
	}

	return nil
}

func (w *Window) ShouldClose() (bool, error) {
	if !_glfw.initialized {
		return false, NotInitialized
	}
	return w.shouldClose, nil
}

func (w *Window) SetShouldClose(value bool) error {
	if !_glfw.initialized {
		return NotInitialized
	}
	w.shouldClose = value
	return nil
}

func (w *Window) SetTitle(title string) error {
	if !_glfw.initialized {
		return NotInitialized
	}
	if err := w.platformSetWindowTitle(title); err != nil {
		return err
	}
	return nil
}

func (w *Window) SetIcon(images []image.Image) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	gimgs := make([]*Image, len(images))
	for i, img := range images {
		b := img.Bounds()
		m := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
		gimgs[i] = &Image{
			Width:  b.Dx(),
			Height: b.Dy(),
			Pixels: m.Pix,
		}
	}

	if err := w.platformSetWindowIcon(gimgs); err != nil {
		return err
	}
	return nil
}

func (w *Window) GetPos() (xpos, ypos int, err error) {
	if !_glfw.initialized {
		return 0, 0, NotInitialized
	}
	return w.platformGetWindowPos()
}

func (w *Window) SetPos(xpos, ypos int) error {
	if !_glfw.initialized {
		return NotInitialized
	}
	if w.monitor != nil {
		return nil
	}
	if err := w.platformSetWindowPos(xpos, ypos); err != nil {
		return err
	}
	return nil
}

func (w *Window) GetSize() (width, height int, err error) {
	if !_glfw.initialized {
		return 0, 0, NotInitialized
	}
	return w.platformGetWindowSize()
}

func (w *Window) SetSize(width, height int) error {
	if !_glfw.initialized {
		return NotInitialized
	}
	w.videoMode.Width = width
	w.videoMode.Height = height
	if err := w.platformSetWindowSize(width, height); err != nil {
		return err
	}
	return nil
}

func (w *Window) SetSizeLimits(minwidth, minheight, maxwidth, maxheight int) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	if minwidth != DontCare && minheight != DontCare {
		if minwidth < 0 || minheight < 0 {
			return fmt.Errorf("glfw: invalid window minimum size %dx%d: %w", minwidth, minheight, InvalidValue)
		}
	}

	if maxwidth != DontCare && maxheight != DontCare {
		if maxwidth < 0 || maxheight < 0 || maxwidth < minwidth || maxheight < minheight {
			return fmt.Errorf("glfw: invalid window maximum size %dx%d: %w", maxwidth, maxheight, InvalidValue)
		}
	}

	w.minwidth = minwidth
	w.minheight = minheight
	w.maxwidth = maxwidth
	w.maxheight = maxheight

	if w.monitor != nil || !w.resizable {
		return nil
	}

	if err := w.platformSetWindowSizeLimits(minwidth, minheight, maxwidth, maxheight); err != nil {
		return err
	}

	return nil
}

func (w *Window) SetAspectRatio(numer, denom int) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	if numer != DontCare && denom != DontCare {
		if numer <= 0 || denom <= 0 {
			return fmt.Errorf("glfw: invalid window aspect ratio %d:%d: %w", numer, denom, InvalidValue)
		}
	}

	w.numer = numer
	w.denom = denom

	if w.monitor != nil || !w.resizable {
		return nil
	}

	if err := w.platformSetWindowAspectRatio(numer, denom); err != nil {
		return err
	}
	return nil
}

func (w *Window) GetFramebufferSize() (width, height int, err error) {
	if !_glfw.initialized {
		return 0, 0, NotInitialized
	}
	return w.platformGetFramebufferSize()
}

func (w *Window) GetFrameSize() (left, top, right, bottom int, err error) {
	if !_glfw.initialized {
		return 0, 0, 0, 0, NotInitialized
	}
	return w.platformGetWindowFrameSize()
}

func (w *Window) GetContentScale() (xscale, yscale float32, err error) {
	if !_glfw.initialized {
		return 0, 0, NotInitialized
	}
	return w.platformGetWindowContentScale()
}

func (w *Window) GetOpacity() (float32, error) {
	if !_glfw.initialized {
		return 0, NotInitialized
	}
	return w.platformGetWindowOpacity()
}

func (w *Window) SetOpacity(opacity float32) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	if opacity != opacity || opacity < 0 || opacity > 1 {
		return fmt.Errorf("glfw: invalid window opacity %f: %w", opacity, InvalidValue)
	}

	if err := w.platformSetWindowOpacity(opacity); err != nil {
		return err
	}
	return nil
}

func (w *Window) Iconify() error {
	if !_glfw.initialized {
		return NotInitialized
	}
	w.platformIconifyWindow()
	return nil
}

func (w *Window) Restore() error {
	if !_glfw.initialized {
		return NotInitialized
	}
	w.platformRestoreWindow()
	return nil
}

func (w *Window) Maximize() error {
	if !_glfw.initialized {
		return NotInitialized
	}
	if w.monitor != nil {
		return nil
	}
	if err := w.platformMaximizeWindow(); err != nil {
		return err
	}
	return nil
}

func (w *Window) Show() error {
	if !_glfw.initialized {
		return NotInitialized
	}
	if w.monitor != nil {
		return nil
	}

	w.platformShowWindow()

	if w.focusOnShow {
		if err := w.platformFocusWindow(); err != nil {
			return err
		}
	}
	return nil
}

func (w *Window) RequestAttention() error {
	if !_glfw.initialized {
		return NotInitialized
	}
	w.platformRequestWindowAttention()
	return nil
}

func (w *Window) Hide() error {
	if !_glfw.initialized {
		return NotInitialized
	}

	if w.monitor != nil {
		return nil
	}

	w.platformHideWindow()
	return nil
}

func (w *Window) Focus() error {
	if !_glfw.initialized {
		return NotInitialized
	}
	if err := w.platformFocusWindow(); err != nil {
		return err
	}
	return nil
}

func (w *Window) GetAttrib(attrib Hint) (int, error) {
	if !_glfw.initialized {
		return 0, NotInitialized
	}

	switch attrib {
	case Focused:
		return boolToInt(w.platformWindowFocused()), nil
	case Iconified:
		return boolToInt(w.platformWindowIconified()), nil
	case Visible:
		return boolToInt(w.platformWindowVisible()), nil
	case Maximized:
		return boolToInt(w.platformWindowMaximized()), nil
	case Hovered:
		b, err := w.platformWindowHovered()
		if err != nil {
			return 0, err
		}
		return boolToInt(b), nil
	case FocusOnShow:
		return boolToInt(w.focusOnShow), nil
	case MousePassthrough:
		return boolToInt(w.mousePassthrough), nil
	case TransparentFramebuffer:
		return boolToInt(w.platformFramebufferTransparent()), nil
	case Resizable:
		return boolToInt(w.resizable), nil
	case Decorated:
		return boolToInt(w.decorated), nil
	case Floating:
		return boolToInt(w.floating), nil
	case AutoIconify:
		return boolToInt(w.autoIconify), nil
	case ClientAPI:
		return w.context.client, nil
	case ContextCreationAPI:
		return w.context.source, nil
	case ContextVersionMajor:
		return w.context.major, nil
	case ContextVersionMinor:
		return w.context.minor, nil
	case ContextRevision:
		return w.context.revision, nil
	case ContextRobustness:
		return w.context.robustness, nil
	case OpenGLForwardCompat:
		return boolToInt(w.context.forward), nil
	case OpenGLDebugContext:
		return boolToInt(w.context.debug), nil
	case OpenGLProfile:
		return w.context.profile, nil
	case ContextReleaseBehavior:
		return w.context.release, nil
	case ContextNoError:
		return boolToInt(w.context.noerror), nil
	default:
		return 0, fmt.Errorf("glfw: invalid window attribute 0x%08X: %w", attrib, InvalidEnum)
	}
}

func (w *Window) SetAttrib(attrib Hint, value int) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	bValue := intToBool(value)

	switch attrib {
	case AutoIconify:
		w.autoIconify = bValue
		return nil
	case Resizable:
		if w.resizable == bValue {
			return nil
		}
		w.resizable = bValue
		if w.monitor == nil {
			if err := w.platformSetWindowResizable(bValue); err != nil {
				return nil
			}
		}
		return nil
	case Decorated:
		if w.decorated == bValue {
			return nil
		}
		w.decorated = bValue
		if w.monitor == nil {
			if err := w.platformSetWindowDecorated(bValue); err != nil {
				return err
			}
		}
		return nil
	case Floating:
		if w.floating == bValue {
			return nil
		}
		w.floating = bValue
		if w.monitor == nil {
			if err := w.platformSetWindowFloating(bValue); err != nil {
				return err
			}
		}
		return nil
	case FocusOnShow:
		w.focusOnShow = bValue
		return nil
	case MousePassthrough:
		w.mousePassthrough = bValue
		if err := w.platformSetWindowMousePassthrough(bValue); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("glfw: invalid window attribute 0x%08X: %w", attrib, InvalidEnum)
	}
}

func (w *Window) GetMonitor() (*Monitor, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	return w.monitor, nil
}

func (w *Window) SetMonitor(monitor *Monitor, xpos, ypos, width, height, refreshRate int) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	if width <= 0 || height <= 0 {
		return fmt.Errorf("glfw: invalid window size %dx%d: %w", width, height, InvalidValue)
	}

	if refreshRate < 0 && refreshRate != DontCare {
		return fmt.Errorf("glfw: invalid refresh rate %d: %w", refreshRate, InvalidValue)
	}

	w.videoMode.Width = width
	w.videoMode.Height = height
	w.videoMode.RefreshRate = refreshRate

	if err := w.platformSetWindowMonitor(monitor, xpos, ypos, width, height, refreshRate); err != nil {
		return err
	}
	return nil
}

func (w *Window) SetUserPointer(pointer unsafe.Pointer) error {
	if !_glfw.initialized {
		return NotInitialized
	}
	w.userPointer = pointer
	return nil
}

func (w *Window) GetUserPointer() (unsafe.Pointer, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	return w.userPointer, nil
}

func (w *Window) SetPosCallback(cbfun PosCallback) (PosCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.pos
	w.callbacks.pos = cbfun
	return old, nil
}

func (w *Window) SetSizeCallback(cbfun SizeCallback) (SizeCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.size
	w.callbacks.size = cbfun
	return old, nil
}

func (w *Window) SetCloseCallback(cbfun CloseCallback) (CloseCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.close
	w.callbacks.close = cbfun
	return old, nil
}

func (w *Window) SetRefreshCallback(cbfun RefreshCallback) (RefreshCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.refresh
	w.callbacks.refresh = cbfun
	return old, nil
}

func (w *Window) SetFocusCallback(cbfun FocusCallback) (FocusCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.focus
	w.callbacks.focus = cbfun
	return old, nil
}

func (w *Window) SetIconifyCallback(cbfun IconifyCallback) (IconifyCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.iconify
	w.callbacks.iconify = cbfun
	return old, nil
}

func (w *Window) SetMaximizeCallback(cbfun MaximizeCallback) (MaximizeCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.maximize
	w.callbacks.maximize = cbfun
	return old, nil
}

func (w *Window) SetFramebufferSizeCallback(cbfun FramebufferSizeCallback) (FramebufferSizeCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.fbsize
	w.callbacks.fbsize = cbfun
	return old, nil
}

func (w *Window) SetContentScaleCallback(cbfun ContentScaleCallback) (ContentScaleCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.scale
	w.callbacks.scale = cbfun
	return old, nil
}

func PollEvents() error {
	if !_glfw.initialized {
		return NotInitialized
	}
	if err := platformPollEvents(); err != nil {
		return err
	}
	return nil
}

func WaitEvents() error {
	if !_glfw.initialized {
		return NotInitialized
	}
	if err := platformWaitEvents(); err != nil {
		return err
	}
	return nil
}

func WaitEventsTimeout(timeout float64) error {
	if !_glfw.initialized {
		return NotInitialized
	}
	if timeout != timeout || timeout < 0.0 || timeout > math.MaxFloat64 {
		return fmt.Errorf("glfw: invalid time %f: %w", timeout, InvalidValue)
	}
	if err := platformWaitEventsTimeout(timeout); err != nil {
		return err
	}
	return nil
}

func PostEmptyEvent() error {
	if !_glfw.initialized {
		return NotInitialized
	}
	if err := platformPostEmptyEvent(); err != nil {
		return err
	}
	return nil
}
