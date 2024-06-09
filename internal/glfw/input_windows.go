// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2002-2006 Marcus Geelnard
// SPDX-FileCopyrightText: 2006-2019 Camilla Löwy <elmindreda@glfw.org>
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package glfw

import (
	"fmt"
	"math"
)

const stick = 3

func (w *Window) inputKey(key Key, scancode int, action Action, mods ModifierKey) {
	if key >= 0 && key <= KeyLast {
		var repeated bool

		if action == Release && w.keys[key] == Release {
			return
		}

		if action == Press && w.keys[key] == Press {
			repeated = true
		}

		if action == Release && w.stickyKeys {
			w.keys[key] = stick
		} else {
			w.keys[key] = action
		}

		if repeated {
			action = Repeat
		}
	}

	if !w.lockKeyMods {
		mods &^= ModCapsLock | ModNumLock
	}

	if w.callbacks.key != nil {
		w.callbacks.key(w, key, scancode, action, mods)
	}
}

func (w *Window) inputChar(codepoint rune, mods ModifierKey, plain bool) {
	if codepoint < 32 || (codepoint > 126 && codepoint < 160) {
		return
	}

	if !w.lockKeyMods {
		mods &^= ModCapsLock | ModNumLock
	}

	if w.callbacks.charmods != nil {
		w.callbacks.charmods(w, codepoint, mods)
	}

	if plain {
		if w.callbacks.character != nil {
			w.callbacks.character(w, codepoint)
		}
	}
}

func (w *Window) inputScroll(xoffset, yoffset float64) {
	if w.callbacks.scroll != nil {
		w.callbacks.scroll(w, xoffset, yoffset)
	}
}

func (w *Window) inputMouseClick(button MouseButton, action Action, mods ModifierKey) {
	if button < 0 || button > MouseButtonLast {
		return
	}

	if !w.lockKeyMods {
		mods &^= ModCapsLock | ModNumLock
	}

	if action == Release && w.stickyMouseButtons {
		w.mouseButtons[button] = stick
	} else {
		w.mouseButtons[button] = action
	}

	if w.callbacks.mouseButton != nil {
		w.callbacks.mouseButton(w, button, action, mods)
	}
}

func (w *Window) inputCursorPos(xpos float64, ypos float64) {
	if w.virtualCursorPosX == xpos && w.virtualCursorPosY == ypos {
		return
	}

	w.virtualCursorPosX = xpos
	w.virtualCursorPosY = ypos

	if w.callbacks.cursorPos != nil {
		w.callbacks.cursorPos(w, xpos, ypos)
	}
}

func (w *Window) inputCursorEnter(entered bool) {
	if w.callbacks.cursorEnter != nil {
		w.callbacks.cursorEnter(w, entered)
	}
}

func (w *Window) inputDrop(paths []string) {
	if w.callbacks.drop != nil {
		w.callbacks.drop(w, paths)
	}
}

func (w *Window) centerCursorInContentArea() error {
	width, height, err := w.platformGetWindowSize()
	if err != nil {
		return err
	}
	if err := w.platformSetCursorPos(float64(width/2), float64(height/2)); err != nil {
		return err
	}
	return nil
}

func (w *Window) GetInputMode(mode InputMode) (int, error) {
	if !_glfw.initialized {
		return 0, NotInitialized
	}
	switch mode {
	case CursorMode:
		return w.cursorMode, nil
	case StickyKeysMode:
		return boolToInt(w.stickyKeys), nil
	case StickyMouseButtonsMode:
		return boolToInt(w.stickyMouseButtons), nil
	case LockKeyMods:
		return boolToInt(w.lockKeyMods), nil
	case RawMouseMotion:
		return boolToInt(w.rawMouseMotion), nil
	default:
		return 0, fmt.Errorf("glfw: invalid input mode 0x%08X: %w", mode, InvalidEnum)
	}
}

func (w *Window) SetInputMode(mode InputMode, value int) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	switch mode {
	case CursorMode:
		if value != CursorNormal && value != CursorHidden && value != CursorDisabled {
			return fmt.Errorf("glfw: invalid cursor mode 0x%08X: %w", value, InvalidEnum)
		}

		if w.cursorMode == value {
			return nil
		}

		w.cursorMode = value

		x, y, err := w.platformGetCursorPos()
		if err != nil {
			return err
		}
		w.virtualCursorPosX = x
		w.virtualCursorPosY = y

		if err := w.platformSetCursorMode(value); err != nil {
			return err
		}
		return nil

	case StickyKeysMode:
		if w.stickyKeys == intToBool(value) {
			return nil
		}

		if !intToBool(value) {
			// Release all sticky keys
			for i := Key(0); i <= KeyLast; i++ {
				if w.keys[i] == stick {
					w.keys[i] = Release
				}
			}
		}

		w.stickyKeys = intToBool(value)
		return nil

	case StickyMouseButtonsMode:
		if w.stickyMouseButtons == intToBool(value) {
			return nil
		}

		if !intToBool(value) {
			// Release all sticky mouse buttons
			for i := MouseButton(0); i <= MouseButtonLast; i++ {
				if w.mouseButtons[i] == stick {
					w.mouseButtons[i] = Release
				}
			}
		}

		w.stickyMouseButtons = intToBool(value)
		return nil

	case LockKeyMods:
		w.lockKeyMods = intToBool(value)
		return nil

	case RawMouseMotion:
		if !platformRawMouseMotionSupported() {
			return fmt.Errorf("glfw: raw mouse motion is not supported on this system: %w", PlatformError)
		}

		if w.rawMouseMotion == intToBool(value) {
			return nil
		}

		w.rawMouseMotion = intToBool(value)
		if err := w.platformSetRawMouseMotion(intToBool(value)); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("glfw: invalid input mode 0x%08X: %w", mode, InvalidEnum)
	}
}

func RawMouseMotionSupported() (bool, error) {
	if !_glfw.initialized {
		return false, NotInitialized
	}
	return platformRawMouseMotionSupported(), nil
}

func GetKeyName(key Key, scancode int) (string, error) {
	if !_glfw.initialized {
		return "", NotInitialized
	}

	if key != KeyUnknown {
		if key < KeySpace || key > KeyLast {
			return "", fmt.Errorf("glfw: invalid key %d: %w", key, InvalidEnum)
		}
		if key != KeyKPEqual && (key < KeyKP0 || key > KeyKPAdd) && (key < KeyApostrophe || key > KeyWorld2) {
			return "", nil
		}
		scancode = platformGetKeyScancode(key)
	}

	return platformGetScancodeName(scancode)
}

func GetKeyScancode(key Key) (int, error) {
	if !_glfw.initialized {
		return 0, NotInitialized
	}

	if key < KeySpace || key > KeyLast {
		return 0, fmt.Errorf("glfw: invalid key %d: %w", key, InvalidEnum)
	}

	return platformGetKeyScancode(key), nil
}

func (w *Window) GetKey(key Key) (Action, error) {
	if !_glfw.initialized {
		return 0, NotInitialized
	}

	if key < KeySpace || key > KeyLast {
		return 0, fmt.Errorf("glfw: invalid key %d: %w", key, InvalidEnum)
	}

	if w.keys[key] == stick {
		// Sticky mode: release key now
		w.keys[key] = Release
		return Press, nil
	}

	return w.keys[key], nil
}

func (w *Window) GetMouseButton(button MouseButton) (Action, error) {
	if !_glfw.initialized {
		return 0, NotInitialized
	}

	if button < MouseButton1 || button > MouseButtonLast {
		return 0, fmt.Errorf("glfw: invalid mouse button %d: %w", button, InvalidEnum)
	}

	if w.mouseButtons[button] == stick {
		// Sticky mode: release mouse button now
		w.mouseButtons[button] = Release
		return Press, nil
	}

	return w.mouseButtons[button], nil
}

func (w *Window) GetCursorPos() (xpos, ypos float64, err error) {
	if !_glfw.initialized {
		return 0, 0, NotInitialized
	}

	if w.cursorMode == CursorDisabled {
		return w.virtualCursorPosX, w.virtualCursorPosY, nil
	} else {
		return w.platformGetCursorPos()
	}
}

func (w *Window) SetCursorPos(xpos, ypos float64) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	if xpos != xpos || xpos < -math.MaxFloat64 || xpos > math.MaxFloat64 || ypos != ypos || ypos < -math.MaxFloat64 || ypos > math.MaxFloat64 {
		return fmt.Errorf("glfw: invalid cursor position %f %f: %w", xpos, ypos, InvalidValue)
	}

	if !w.platformWindowFocused() {
		return nil
	}

	if w.cursorMode == CursorDisabled {
		// Only update the accumulated position if the cursor is disabled
		w.virtualCursorPosX = xpos
		w.virtualCursorPosY = ypos
		return nil
	} else {
		// Update system cursor position
		return w.platformSetCursorPos(xpos, ypos)
	}
}

func CreateStandardCursor(shape StandardCursor) (*Cursor, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}

	if shape != ArrowCursor &&
		shape != IBeamCursor &&
		shape != CrosshairCursor &&
		shape != HandCursor &&
		shape != HResizeCursor &&
		shape != VResizeCursor &&
		shape != ResizeNWSECursor &&
		shape != ResizeNESWCursor &&
		shape != ResizeAllCursor &&
		shape != NotAllowedCursor {
		return nil, fmt.Errorf("glfw: invalid standard cursor 0x%08X: %w", shape, InvalidEnum)
	}

	cursor := &Cursor{}
	_glfw.cursors = append(_glfw.cursors, cursor)

	if err := cursor.platformCreateStandardCursor(shape); err != nil {
		_ = cursor.Destroy()
		return nil, err
	}

	return cursor, nil
}

func (c *Cursor) Destroy() error {
	if !_glfw.initialized {
		return NotInitialized
	}

	if c == nil {
		return nil
	}

	// Make sure the cursor is not being used by any window
	for _, window := range _glfw.windows {
		if window.cursor == c {
			if err := window.SetCursor(nil); err != nil {
				return err
			}
		}
	}

	if err := c.platformDestroyCursor(); err != nil {
		return err
	}

	// Unlink cursor from global linked list
	for i, cursor := range _glfw.cursors {
		if cursor == c {
			copy(_glfw.cursors[i:], _glfw.cursors[i+1:])
			_glfw.cursors = _glfw.cursors[:len(_glfw.cursors)-1]
			break
		}
	}

	return nil
}

func (w *Window) SetCursor(cursor *Cursor) error {
	if !_glfw.initialized {
		return NotInitialized
	}

	w.cursor = cursor

	if err := w.platformSetCursor(cursor); err != nil {
		return err
	}
	return nil
}

func (w *Window) SetKeyCallback(cbfun KeyCallback) (KeyCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.key
	w.callbacks.key = cbfun
	return old, nil
}

func (w *Window) SetCharCallback(cbfun CharCallback) (CharCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.character
	w.callbacks.character = cbfun
	return old, nil
}

func (w *Window) SetCharModsCallback(cbfun CharModsCallback) (CharModsCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.charmods
	w.callbacks.charmods = cbfun
	return old, nil
}

func (w *Window) SetMouseButtonCallback(cbfun MouseButtonCallback) (MouseButtonCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.mouseButton
	w.callbacks.mouseButton = cbfun
	return old, nil
}

func (w *Window) SetCursorPosCallback(cbfun CursorPosCallback) (CursorPosCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.cursorPos
	w.callbacks.cursorPos = cbfun
	return old, nil
}

func (w *Window) SetCursorEnterCallback(cbfun CursorEnterCallback) (CursorEnterCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.cursorEnter
	w.callbacks.cursorEnter = cbfun
	return old, nil
}

func (w *Window) SetScrollCallback(cbfun ScrollCallback) (ScrollCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.scroll
	w.callbacks.scroll = cbfun
	return old, nil
}

func (w *Window) SetDropCallback(cbfun DropCallback) (DropCallback, error) {
	if !_glfw.initialized {
		return nil, NotInitialized
	}
	old := w.callbacks.drop
	w.callbacks.drop = cbfun
	return old, nil
}

func (w *Window) SetClipboardString(str string) error {
	if !_glfw.initialized {
		return NotInitialized
	}
	return platformSetClipboardString(str)
}

func GetClipboardString() (string, error) {
	if !_glfw.initialized {
		return "", NotInitialized
	}
	return platformGetClipboardString()
}
