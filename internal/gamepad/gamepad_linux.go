// Copyright 2022 The Ebiten Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !android && !nintendosdk && !playstation5

package gamepad

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
	"github.com/hajimehoshi/ebiten/v2/internal/mathutil"
)

const dirName = "/dev/input"

var reEvent = regexp.MustCompile(`^event[0-9]+$`)

func isBitSet(s []byte, bit int) bool {
	return s[bit/8]&(1<<(bit%8)) != 0
}

// openDevice opens path, retrying on EINTR.
func openDevice(path string, flags int) (int, error) {
	for {
		fd, err := unix.Open(path, flags, 0)
		if err != unix.EINTR {
			return fd, err
		}
	}
}

// isDisconnectError reports whether err indicates that the device was removed.
func isDisconnectError(err error) bool {
	// Some drivers report EIO instead of ENODEV on removal.
	return errors.Is(err, unix.ENODEV) || errors.Is(err, unix.EIO)
}

type nativeGamepadsImpl struct {
	inotifyPlus1 int
	watch        int

	// pendingTouch holds the touch surface nodes whose gamepad node has not been opened yet, keyed
	// by the controller's uniq string. A node's gamepad can appear before or after it, so whichever
	// is opened second does the pairing.
	pendingTouch map[string]*touchNode
}

func newNativeGamepadsImpl() nativeGamepads {
	return &nativeGamepadsImpl{}
}

func (g *nativeGamepadsImpl) init(gamepads *gamepads) (err error) {
	// Check the existence of the directory `dirName`.
	var stat unix.Stat_t
	if err := unix.Stat(dirName, &stat); err != nil {
		if err == unix.ENOENT {
			return nil
		}
		// `/dev/input` might not be accessible in some environments (#3057).
		if err == unix.EACCES {
			return nil
		}
		return fmt.Errorf("gamepad: Stat failed: %w", err)
	}
	if stat.Mode&unix.S_IFDIR == 0 {
		return nil
	}

	// Another program of the same user can exhaust the inotify limits (#3304). That only costs
	// hotplug detection, so it is not fatal here, and GLFW ignores the same errors.
	if inotify, err := unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC); err == nil {
		g.inotifyPlus1 = inotify + 1

		// Register for IN_ATTRIB to get notified when udev is done.
		// This works well in practice but the true way is libudev.
		if watch, err := unix.InotifyAddWatch(g.inotifyPlus1-1, dirName, unix.IN_CREATE|unix.IN_ATTRIB|unix.IN_DELETE); err == nil {
			g.watch = watch
		} else {
			_ = unix.Close(g.inotifyPlus1 - 1)
			g.inotifyPlus1 = 0
		}
	}
	defer func() {
		if err != nil && g.inotifyPlus1 != 0 {
			_ = unix.Close(g.inotifyPlus1 - 1)
			g.inotifyPlus1 = 0
		}
	}()

	ents, err := os.ReadDir(dirName)
	if err != nil {
		// Directory enumeration can fail even when Stat succeeds.
		// Keep any active inotify watch for later gamepad detection.
		return nil
	}
	for _, ent := range ents {
		if ent.IsDir() {
			continue
		}
		if !reEvent.MatchString(ent.Name()) {
			continue
		}
		if err := g.openDevice(gamepads, filepath.Join(dirName, ent.Name())); err != nil {
			return err
		}
	}

	return nil
}

// isOpen reports whether the node at path is already open as a gamepad, as the touch surface of a
// gamepad, or as a pending touch surface.
func (g *nativeGamepadsImpl) isOpen(gamepads *gamepads, path string) bool {
	if gamepads.find(func(gamepad *Gamepad) bool {
		n := gamepad.native.(*nativeGamepadImpl)
		return n.path == path || (n.touch != nil && n.touch.path == path)
	}) != nil {
		return true
	}
	for _, t := range g.pendingTouch {
		if t.path == path {
			return true
		}
	}
	return false
}

// openDevice opens the event node at path and, by what it is, adds it as a gamepad, attaches it as
// the touch surface of its gamepad, or closes it again.
func (g *nativeGamepadsImpl) openDevice(gamepads *gamepads, path string) error {
	if g.isOpen(gamepads, path) {
		return nil
	}

	// Rumble requires write access to upload and play force feedback effects.
	// Fall back to read-only when write access is not permitted: the gamepad
	// still works, without rumble.
	writable := true
	fd, err := openDevice(path, unix.O_RDWR|unix.O_NONBLOCK|unix.O_CLOEXEC)
	if err == unix.EACCES || err == unix.EPERM {
		writable = false
		fd, err = openDevice(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC)
	}
	if err != nil {
		if err == unix.EACCES {
			return nil
		}
		// This happens with the Snap sandbox.
		if err == unix.EPERM {
			return nil
		}
		// This happens just after a disconnection.
		if err == unix.ENOENT {
			return nil
		}
		return fmt.Errorf("gamepad: Open failed: %w", err)
	}
	owned := true
	defer func() {
		if owned {
			_ = unix.Close(fd)
		}
	}()

	evBits := make([]byte, (unix.EV_CNT+7)/8)
	keyBits := make([]byte, (_KEY_CNT+7)/8)
	absBits := make([]byte, (_ABS_CNT+7)/8)
	var id input_id
	// The device can be removed between the open and the ioctls below. Such a
	// device is skipped; the deferred Close releases the fd.
	if err := ioctl(fd, _EVIOCGBIT(0, uint(len(evBits))), unsafe.Pointer(&evBits[0])); err != nil {
		if isDisconnectError(err) {
			return nil
		}
		return fmt.Errorf("gamepad: ioctl for evBits failed: %w", err)
	}
	if err := ioctl(fd, _EVIOCGBIT(unix.EV_KEY, uint(len(keyBits))), unsafe.Pointer(&keyBits[0])); err != nil {
		if isDisconnectError(err) {
			return nil
		}
		return fmt.Errorf("gamepad: ioctl for keyBits failed: %w", err)
	}
	if err := ioctl(fd, _EVIOCGBIT(unix.EV_ABS, uint(len(absBits))), unsafe.Pointer(&absBits[0])); err != nil {
		if isDisconnectError(err) {
			return nil
		}
		return fmt.Errorf("gamepad: ioctl for absBits failed: %w", err)
	}
	if err := ioctl(fd, _EVIOCGID(), unsafe.Pointer(&id)); err != nil {
		if isDisconnectError(err) {
			return nil
		}
		return fmt.Errorf("gamepad: ioctl for an ID failed: %w", err)
	}
	// A node without properties is fine; the property bits then stay clear.
	propBits := make([]byte, (_INPUT_PROP_CNT+7)/8)
	_ = ioctl(fd, _EVIOCGPROP(uint(len(propBits))), unsafe.Pointer(&propBits[0]))

	kind := classifyEvdev(evBits, keyBits, absBits, propBits)
	if kind == evdevKindOther {
		owned = false
		return unix.Close(fd)
	}

	// The uniq string, the controller's address on the kernel drivers of interest, is what ties a
	// touch surface node to its gamepad node. Many devices have none.
	uniq := ""
	cuniq := make([]byte, 256)
	if err := ioctl(fd, _EVIOCGUNIQ(uint(len(cuniq))), unsafe.Pointer(&cuniq[0])); err == nil {
		uniq = unix.ByteSliceToString(cuniq)
	}

	if kind == evdevKindTouchSurface {
		// A touch surface without a uniq cannot be paired with a gamepad, so it is not a
		// controller's touch surface.
		if uniq == "" {
			owned = false
			return unix.Close(fd)
		}
		t, err := newTouchNode(fd, path, uniq, id)
		if err != nil {
			if isDisconnectError(err) {
				return nil
			}
			return err
		}
		owned = false
		g.attachTouch(gamepads, t)
		return nil
	}

	cname := make([]byte, 256)
	name := "Unknown"
	// TODO: Is it OK to ignore the error here?
	if err := ioctl(fd, uint(_EVIOCGNAME(uint(len(cname)))), unsafe.Pointer(&cname[0])); err == nil {
		name = unix.ByteSliceToString(cname)
	}

	var sdlID string
	if id.vendor != 0 && id.product != 0 && id.version != 0 {
		sdlID = fmt.Sprintf("%02x%02x0000%02x%02x0000%02x%02x0000%02x%02x0000",
			byte(id.bustype), byte(id.bustype>>8),
			byte(id.vendor), byte(id.vendor>>8),
			byte(id.product), byte(id.product>>8),
			byte(id.version), byte(id.version>>8))
	} else {
		bs := []byte(name)
		if len(bs) < 12 {
			bs = append(bs, make([]byte, 12-len(bs))...)
		}
		sdlID = fmt.Sprintf("%02x%02x0000%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x",
			byte(id.bustype), byte(id.bustype>>8),
			bs[0], bs[1], bs[2], bs[3], bs[4], bs[5], bs[6], bs[7], bs[8], bs[9], bs[10], bs[11])
	}

	supportsRumble := false
	if writable && isBitSet(evBits, unix.EV_FF) {
		ffBits := make([]byte, (_FF_CNT+7)/8)
		if err := ioctl(fd, _EVIOCGBIT(unix.EV_FF, uint(len(ffBits))), unsafe.Pointer(&ffBits[0])); err == nil {
			supportsRumble = isBitSet(ffBits, _FF_RUMBLE)
		}
	}

	n := &nativeGamepadImpl{
		path:           path,
		fdPlus1:        fd + 1,
		uniq:           uniq,
		id:             id,
		supportsRumble: supportsRumble,
		effectID:       -1,
	}

	var axisCount int
	var buttonCount int
	var hatCount int
	for i := range n.keyMap {
		n.keyMap[i] = -1
	}
	for i := range n.absMap {
		n.absMap[i] = -1
	}
	for code := _BTN_MISC; code < _KEY_CNT; code++ {
		if !isBitSet(keyBits, code) {
			continue
		}
		n.keyMap[code-_BTN_MISC] = buttonCount
		buttonCount++
	}
	for code := 0; code < _ABS_CNT; code++ {
		if !isBitSet(absBits, code) {
			continue
		}
		if code >= _ABS_HAT0X && code <= _ABS_HAT3Y {
			// Write the hat index for both the X and the Y hat axes.
			// That way, the hat can be referenced using either axis, which is used by the code building hatMappingInput.
			n.absMap[code] = hatCount
			code++
			n.absMap[code] = hatCount
			hatCount++
			continue
		}
		if err := ioctl(n.fdPlus1-1, uint(_EVIOCGABS(uint(code))), unsafe.Pointer(&n.absInfo[code])); err != nil {
			if isDisconnectError(err) {
				return nil
			}
			return fmt.Errorf("gamepad: ioctl for an abs at openGamepad failed: %w", err)
		}
		n.absMap[code] = axisCount
		axisCount++
	}

	n.axisCount_ = axisCount
	n.buttonCount_ = buttonCount
	n.hatCount_ = hatCount

	n.computeStandardLayout(id.vendor)

	if err := n.pollAbsState(); err != nil {
		if isDisconnectError(err) {
			return nil
		}
		return err
	}

	// The gamepad's touch surface may have been opened first.
	if t := g.pendingTouch[uniq]; t != nil && uniq != "" && t.id.vendor == id.vendor && t.id.product == id.product {
		n.touch = t
		delete(g.pendingTouch, uniq)
	}

	owned = false
	gp := gamepads.add(name, sdlID)
	gp.native = n
	n.cleanup = runtime.AddCleanup(gp, func(n *nativeGamepadImpl) {
		n.close()
	}, n)

	return nil
}

// attachTouch gives a touch surface node to the gamepad of the same controller, or keeps it until
// that gamepad is opened. A gamepad that already has a touch surface keeps the one it has.
func (g *nativeGamepadsImpl) attachTouch(gamepads *gamepads, t *touchNode) {
	if gp := gamepads.find(func(gamepad *Gamepad) bool {
		n := gamepad.native.(*nativeGamepadImpl)
		return n.touch == nil && n.uniq == t.uniq && n.id.vendor == t.id.vendor && n.id.product == t.id.product
	}); gp != nil {
		withNative(gp, func(n *nativeGamepadImpl) {
			n.touch = t
		})
		return
	}
	if g.pendingTouch == nil {
		g.pendingTouch = map[string]*touchNode{}
	}
	if old := g.pendingTouch[t.uniq]; old != nil {
		old.close()
	}
	g.pendingTouch[t.uniq] = t
}

// removeDevice drops the node at path, whichever of a gamepad, an attached touch surface, or a
// pending touch surface it is open as.
func (g *nativeGamepadsImpl) removeDevice(gamepads *gamepads, path string) {
	if gp := gamepads.find(func(gamepad *Gamepad) bool {
		return gamepad.native.(*nativeGamepadImpl).path == path
	}); gp != nil {
		// Lock the gamepad so the close cannot race with a
		// concurrent Vibrate using the file descriptor.
		withNative(gp, func(n *nativeGamepadImpl) {
			n.close()
		})
		gamepads.remove(func(gamepad *Gamepad) bool {
			return gamepad == gp
		})
		return
	}
	if gp := gamepads.find(func(gamepad *Gamepad) bool {
		n := gamepad.native.(*nativeGamepadImpl)
		return n.touch != nil && n.touch.path == path
	}); gp != nil {
		withNative(gp, func(n *nativeGamepadImpl) {
			n.touch.close()
			n.touch = nil
		})
		return
	}
	for uniq, t := range g.pendingTouch {
		if t.path == path {
			t.close()
			delete(g.pendingTouch, uniq)
			return
		}
	}
}

func (g *nativeGamepadsImpl) update(gamepads *gamepads) error {
	if g.inotifyPlus1 == 0 {
		return nil
	}

	buf := make([]byte, 16384)
	n, err := unix.Read(g.inotifyPlus1-1, buf[:])
	if err != nil {
		// EINTR means the read was interrupted before any event arrived.
		// Retry at the next update instead of reporting an error.
		if err == unix.EAGAIN || err == unix.EINTR {
			return nil
		}
		return fmt.Errorf("gamepad: Read failed: %w", err)
	}
	buf = buf[:n]

	for len(buf) > 0 {
		e := unix.InotifyEvent{
			Wd:     int32(buf[0]) | int32(buf[1])<<8 | int32(buf[2])<<16 | int32(buf[3])<<24,
			Mask:   uint32(buf[4]) | uint32(buf[5])<<8 | uint32(buf[6])<<16 | uint32(buf[7])<<24,
			Cookie: uint32(buf[8]) | uint32(buf[9])<<8 | uint32(buf[10])<<16 | uint32(buf[11])<<24,
			Len:    uint32(buf[12]) | uint32(buf[13])<<8 | uint32(buf[14])<<16 | uint32(buf[15])<<24,
		}
		if e.Len == 0 {
			buf = buf[16:]
			continue
		}
		name := unix.ByteSliceToString(buf[16 : 16+e.Len-1]) // len includes the null terminator.
		buf = buf[16+e.Len:]
		if !reEvent.MatchString(name) {
			continue
		}

		path := filepath.Join(dirName, name)
		if e.Mask&(unix.IN_CREATE|unix.IN_ATTRIB) != 0 {
			if err := g.openDevice(gamepads, path); err != nil {
				return err
			}
			continue
		}
		if e.Mask&unix.IN_DELETE != 0 {
			g.removeDevice(gamepads, path)
			continue
		}
	}

	return nil
}

// readInputEvent reads one event from an event node. ok is false when no event is pending.
func readInputEvent(fd int) (e input_event, ok bool, err error) {
	buf := make([]byte, unsafe.Sizeof(input_event{}))
	// TODO: Should the returned byte count be cared about?
	if _, err := unix.Read(fd, buf); err != nil {
		// EINTR means no event was read. Retry at the next update instead of
		// treating this as an error and dropping the device.
		if err == unix.EAGAIN || err == unix.EINTR {
			return input_event{}, false, nil
		}
		return input_event{}, false, fmt.Errorf("gamepad: Read failed: %w", err)
	}

	const (
		offsetTyp   = unsafe.Offsetof(input_event{}.typ)
		offsetCode  = unsafe.Offsetof(input_event{}.code)
		offsetValue = unsafe.Offsetof(input_event{}.value)
	)
	// time is not used.
	return input_event{
		typ:   uint16(buf[offsetTyp]) | uint16(buf[offsetTyp+1])<<8,
		code:  uint16(buf[offsetCode]) | uint16(buf[offsetCode+1])<<8,
		value: int32(buf[offsetValue]) | int32(buf[offsetValue+1])<<8 | int32(buf[offsetValue+2])<<16 | int32(buf[offsetValue+3])<<24,
	}, true, nil
}

type nativeGamepadImpl struct {
	fdPlus1 int
	path    string

	// uniq and id identify the controller; a touch surface node with the same values is the
	// controller's touchpad. touch is that node once it is paired, or nil.
	uniq  string
	id    input_id
	touch *touchNode

	keyMap  [_KEY_CNT - _BTN_MISC]int
	absMap  [_ABS_CNT]int
	absInfo [_ABS_CNT]input_absinfo
	dropped bool

	supportsRumble bool
	effectID       int16

	axes    [_ABS_CNT]float64
	buttons [_KEY_CNT - _BTN_MISC]bool
	hats    [4]int

	axisCount_   int
	buttonCount_ int
	hatCount_    int

	stdAxisMap   map[gamepaddb.StandardAxis]mappingInput
	stdButtonMap map[gamepaddb.StandardButton]mappingInput

	cleanup runtime.Cleanup
}

func (g *nativeGamepadImpl) close() {
	g.cleanup.Stop()
	if g.touch != nil {
		g.touch.close()
		g.touch = nil
	}
	if g.fdPlus1 == 0 {
		return
	}
	_ = unix.Close(g.fdPlus1 - 1)
	g.fdPlus1 = 0
}

func (g *nativeGamepadImpl) update(gamepad *gamepads) (err error) {
	if g.fdPlus1 == 0 {
		return nil
	}

	defer func() {
		if err == nil {
			return
		}
		g.close()
		// A removed device is not an error; the inotify IN_DELETE event drops
		// it from the list.
		if isDisconnectError(err) {
			err = nil
		}
	}()

	for {
		e, ok, err := readInputEvent(g.fdPlus1 - 1)
		if err != nil {
			return err
		}
		if !ok {
			break
		}

		if e.typ == unix.EV_SYN && e.code == _SYN_DROPPED {
			g.dropped = true
		}
		if g.dropped {
			// Ignore events through the next SYN_REPORT, then restore the device state.
			if e.typ == unix.EV_SYN && e.code == _SYN_REPORT {
				if err := g.pollAbsState(); err != nil {
					return fmt.Errorf("gamepad: poll absolute state: %w", err)
				}
				if err := g.pollKeyState(); err != nil {
					return fmt.Errorf("gamepad: poll key state: %w", err)
				}
				g.dropped = false
			}
			continue
		}

		switch e.typ {
		case unix.EV_KEY:
			if int(e.code-_BTN_MISC) < len(g.keyMap) {
				idx := g.keyMap[e.code-_BTN_MISC]
				if idx < 0 {
					continue
				}
				g.buttons[idx] = e.value != 0
			}
		case unix.EV_ABS:
			g.handleAbsEvent(int(e.code), e.value)
		}
	}

	// The touch surface is an extra: a failure on its node costs the surface, not the gamepad. Its
	// removal drops it from the list either way.
	if g.touch != nil {
		if err := g.touch.update(); err != nil {
			g.touch.close()
			g.touch = nil
		}
	}
	return nil
}

func (g *nativeGamepadImpl) pollKeyState() error {
	var keyBits [(_KEY_CNT + 7) / 8]byte
	if err := ioctl(g.fdPlus1-1, _EVIOCGKEY(uint(len(keyBits))), unsafe.Pointer(&keyBits[0])); err != nil {
		return fmt.Errorf("gamepad: ioctl for keys at pollKeyState failed: %w", err)
	}
	for code, index := range g.keyMap {
		if index >= 0 {
			g.buttons[index] = isBitSet(keyBits[:], code+_BTN_MISC)
		}
	}
	return nil
}

func (g *nativeGamepadImpl) pollAbsState() error {
	for code := range _ABS_CNT {
		if g.absMap[code] < 0 {
			continue
		}
		if err := ioctl(g.fdPlus1-1, uint(_EVIOCGABS(uint(code))), unsafe.Pointer(&g.absInfo[code])); err != nil {
			return fmt.Errorf("gamepad: ioctl for an abs at pollAbsState failed: %w", err)
		}
		g.handleAbsEvent(code, g.absInfo[code].value)
	}
	return nil
}

func (g *nativeGamepadImpl) handleAbsEvent(code int, value int32) {
	if code < 0 || code >= len(g.absMap) {
		return
	}
	index := g.absMap[code]
	if index < 0 {
		return
	}

	if code >= _ABS_HAT0X && code <= _ABS_HAT3Y {
		axis := (code - _ABS_HAT0X) % 2

		switch axis {
		case 0:
			switch {
			case value < 0:
				g.hats[index] |= hatLeft
				g.hats[index] &^= hatRight
			case value > 0:
				g.hats[index] &^= hatLeft
				g.hats[index] |= hatRight
			default:
				g.hats[index] &^= hatLeft | hatRight
			}
		case 1:
			switch {
			case value < 0:
				g.hats[index] |= hatUp
				g.hats[index] &^= hatDown
			case value > 0:
				g.hats[index] &^= hatUp
				g.hats[index] |= hatDown
			default:
				g.hats[index] &^= hatUp | hatDown
			}
		}
		return
	}

	info := g.absInfo[code]
	v := float64(value)
	if r := float64(info.maximum) - float64(info.minimum); r != 0 {
		v = (v - float64(info.minimum)) / r
		v = v*2 - 1
	}
	g.axes[index] = v
}

func (g *nativeGamepadImpl) computeStandardLayout(vendor uint16) {
	g.stdAxisMap = map[gamepaddb.StandardAxis]mappingInput{}
	g.stdButtonMap = map[gamepaddb.StandardButton]mappingInput{}

	// NOTE: assignments to the same value are in exact reverse order as SDL2,
	// so we can just overwrite rather than checking.

	// BTN_GAMEPAD implies that the kernel module implements standard mapping.
	if b := g.keyMap[_BTN_GAMEPAD-_BTN_MISC]; b < 0 {
		return
	}

	// A and B buttons go by name.
	if b := g.keyMap[_BTN_A-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonRightBottom] = buttonMappingInput{g: g, button: b}
	}
	if b := g.keyMap[_BTN_B-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonRightRight] = buttonMappingInput{g: g, button: b}
	}
	if vendor == 0x054c /* USB_VENDOR_SONY */ {
		// Sony uses WEST/NORTH buttons.
		if b := g.keyMap[_BTN_WEST-_BTN_MISC]; b >= 0 {
			g.stdButtonMap[gamepaddb.StandardButtonRightLeft] = buttonMappingInput{g: g, button: b}
		}
		if b := g.keyMap[_BTN_NORTH-_BTN_MISC]; b >= 0 {
			g.stdButtonMap[gamepaddb.StandardButtonRightTop] = buttonMappingInput{g: g, button: b}
		}
	} else {
		// Xbox uses X/Y buttons.
		// Note that this is the opposite assignment following the WEST/NORTH mappings,
		// and contradicts Linux kernel documentation which states
		// that buttons are always assigned by physical location.
		// However, it matches actual Xbox gamepads, and SDL2 has the same logic.
		if b := g.keyMap[_BTN_X-_BTN_MISC]; b >= 0 {
			g.stdButtonMap[gamepaddb.StandardButtonRightLeft] = buttonMappingInput{g: g, button: b}
		}
		if b := g.keyMap[_BTN_Y-_BTN_MISC]; b >= 0 {
			g.stdButtonMap[gamepaddb.StandardButtonRightTop] = buttonMappingInput{g: g, button: b}
		}
	}

	// Center and thumb buttons.
	if b := g.keyMap[_BTN_SELECT-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonCenterLeft] = buttonMappingInput{g: g, button: b}
	}
	if b := g.keyMap[_BTN_START-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonCenterRight] = buttonMappingInput{g: g, button: b}
	}
	if b := g.keyMap[_BTN_THUMBL-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonLeftStick] = buttonMappingInput{g: g, button: b}
	}
	if b := g.keyMap[_BTN_THUMBR-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonRightStick] = buttonMappingInput{g: g, button: b}
	}
	if b := g.keyMap[_BTN_MODE-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonCenterCenter] = buttonMappingInput{g: g, button: b}
	}

	// Shoulder buttons can be analog or digital. Prefer digital ones.
	if h := g.absMap[_ABS_HAT1Y]; h >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonFrontTopLeft] = hatMappingInput{g: g, hat: h, direction: hatDown}
	}
	if h := g.absMap[_ABS_HAT1X]; h >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonFrontTopRight] = hatMappingInput{g: g, hat: h, direction: hatRight}
	}
	if b := g.keyMap[_BTN_TL-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonFrontTopLeft] = buttonMappingInput{g: g, button: b}
	}
	if b := g.keyMap[_BTN_TR-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonFrontTopRight] = buttonMappingInput{g: g, button: b}
	}

	// Triggers can be analog or digital. Prefer analog ones.
	if b := g.keyMap[_BTN_TL2-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonFrontBottomLeft] = buttonMappingInput{g: g, button: b}
	}
	if b := g.keyMap[_BTN_TR2-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonFrontBottomRight] = buttonMappingInput{g: g, button: b}
	}
	if a := g.absMap[_ABS_Z]; a >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonFrontBottomLeft] = axisMappingInput{g: g, axis: a}
	}
	if a := g.absMap[_ABS_RZ]; a >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonFrontBottomRight] = axisMappingInput{g: g, axis: a}
	}
	if h := g.absMap[_ABS_HAT2Y]; h >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonFrontBottomLeft] = hatMappingInput{g: g, hat: h, direction: hatDown}
	}
	if h := g.absMap[_ABS_HAT2X]; h >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonFrontBottomRight] = hatMappingInput{g: g, hat: h, direction: hatRight}
	}

	// D-pad can be analog or digital. Prefer the digital one.
	if h := g.absMap[_ABS_HAT0X]; h >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonLeftLeft] = hatMappingInput{g: g, hat: h, direction: hatLeft}
		g.stdButtonMap[gamepaddb.StandardButtonLeftRight] = hatMappingInput{g: g, hat: h, direction: hatRight}
	}
	if h := g.absMap[_ABS_HAT0Y]; h >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonLeftTop] = hatMappingInput{g: g, hat: h, direction: hatUp}
		g.stdButtonMap[gamepaddb.StandardButtonLeftBottom] = hatMappingInput{g: g, hat: h, direction: hatDown}
	}
	if b := g.keyMap[_BTN_DPAD_UP-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonLeftTop] = buttonMappingInput{g: g, button: b}
	}
	if b := g.keyMap[_BTN_DPAD_DOWN-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonLeftBottom] = buttonMappingInput{g: g, button: b}
	}
	if b := g.keyMap[_BTN_DPAD_LEFT-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonLeftLeft] = buttonMappingInput{g: g, button: b}
	}
	if b := g.keyMap[_BTN_DPAD_RIGHT-_BTN_MISC]; b >= 0 {
		g.stdButtonMap[gamepaddb.StandardButtonLeftRight] = buttonMappingInput{g: g, button: b}
	}

	// Left stick.
	if a := g.absMap[_ABS_X]; a >= 0 {
		g.stdAxisMap[gamepaddb.StandardAxisLeftStickHorizontal] = axisMappingInput{g: g, axis: a}
	}
	if a := g.absMap[_ABS_Y]; a >= 0 {
		g.stdAxisMap[gamepaddb.StandardAxisLeftStickVertical] = axisMappingInput{g: g, axis: a}
	}

	// Right stick.
	if a := g.absMap[_ABS_RX]; a >= 0 {
		g.stdAxisMap[gamepaddb.StandardAxisRightStickHorizontal] = axisMappingInput{g: g, axis: a}
	}
	if a := g.absMap[_ABS_RY]; a >= 0 {
		g.stdAxisMap[gamepaddb.StandardAxisRightStickVertical] = axisMappingInput{g: g, axis: a}
	}
}

func (g *nativeGamepadImpl) hasOwnStandardLayoutMapping() bool {
	return len(g.stdAxisMap) != 0 || len(g.stdButtonMap) != 0
}

func (g *nativeGamepadImpl) standardAxisInOwnMapping(axis gamepaddb.StandardAxis) mappingInput {
	return g.stdAxisMap[axis]
}

func (g *nativeGamepadImpl) standardButtonInOwnMapping(button gamepaddb.StandardButton) mappingInput {
	return g.stdButtonMap[button]
}

func (g *nativeGamepadImpl) axisCount() int {
	return g.axisCount_
}

func (g *nativeGamepadImpl) buttonCount() int {
	return g.buttonCount_
}

func (g *nativeGamepadImpl) hatCount() int {
	return g.hatCount_
}

func (g *nativeGamepadImpl) isAxisReady(axis int) bool {
	return axis >= 0 && axis < g.axisCount()
}

func (g *nativeGamepadImpl) axisValue(axis int) float64 {
	if axis < 0 || axis >= g.axisCount_ {
		return 0
	}
	return g.axes[axis]
}

func (g *nativeGamepadImpl) isButtonPressed(button int) bool {
	if button < 0 || button >= g.buttonCount_ {
		return false
	}
	return g.buttons[button]
}

func (g *nativeGamepadImpl) buttonValue(button int) float64 {
	if g.isButtonPressed(button) {
		return 1
	}
	return 0
}

func (g *nativeGamepadImpl) hatState(hat int) int {
	if hat < 0 || hat >= g.hatCount_ {
		return hatCentered
	}
	return g.hats[hat]
}

func (g *nativeGamepadImpl) vibrate(duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	if !g.supportsRumble || g.fdPlus1 == 0 {
		return
	}

	strongMagnitude = mathutil.Clamp01(strongMagnitude)
	weakMagnitude = mathutil.Clamp01(weakMagnitude)

	if strongMagnitude <= 0 && weakMagnitude <= 0 {
		g.writeFFEvent(0)
		return
	}

	if duration <= 0 {
		return
	}

	// The kernel stops the effect once the replay length has passed, so no
	// duration tracking is needed here. A replay length of 0 would play the
	// effect with no time limit, so keep it at least 1.
	ms := duration.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	if ms > 0xffff {
		ms = 0xffff
	}

	// An ID of -1 lets the kernel assign an ID to a new effect. Uploading
	// with the ID of an already uploaded effect updates the effect in place.
	effect := ff_effect{
		typ: _FF_RUMBLE,
		id:  g.effectID,
	}
	effect.replay.length = uint16(ms)
	effect.u.rumble.strong_magnitude = motorMagnitude(strongMagnitude)
	effect.u.rumble.weak_magnitude = motorMagnitude(weakMagnitude)

	if err := ioctl(g.fdPlus1-1, _EVIOCSFF(), unsafe.Pointer(&effect)); err != nil {
		return
	}
	g.effectID = effect.id

	g.writeFFEvent(1)
}

// writeFFEvent starts (value 1) or stops (value 0) playing the uploaded force
// feedback effect. It does nothing when no effect has been uploaded.
func (g *nativeGamepadImpl) writeFFEvent(value int32) {
	if g.effectID < 0 {
		return
	}
	e := input_event{
		typ:   unix.EV_FF,
		code:  uint16(g.effectID),
		value: value,
	}
	_, _ = unix.Write(g.fdPlus1-1, unsafe.Slice((*byte)(unsafe.Pointer(&e)), int(unsafe.Sizeof(e))))
}
