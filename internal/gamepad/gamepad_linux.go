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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/hajimehoshi/ebiten/v2/internal/gamepaddb"
)

const dirName = "/dev/input"

var reEvent = regexp.MustCompile(`^event[0-9]+$`)

func isBitSet(s []byte, bit int) bool {
	return s[bit/8]&(1<<(bit%8)) != 0
}

type nativeGamepadsImpl struct {
	inotify int
	watch   int
}

func newNativeGamepadsImpl() nativeGamepads {
	return &nativeGamepadsImpl{}
}

func (g *nativeGamepadsImpl) init(gamepads *gamepads) error {
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

	inotify, err := unix.InotifyInit1(unix.IN_NONBLOCK | unix.IN_CLOEXEC)
	if err != nil {
		return fmt.Errorf("gamepad: InotifyInit1 failed: %w", err)
	}
	g.inotify = inotify

	if g.inotify > 0 {
		// Register for IN_ATTRIB to get notified when udev is done.
		// This works well in practice but the true way is libudev.
		watch, err := unix.InotifyAddWatch(g.inotify, dirName, unix.IN_CREATE|unix.IN_ATTRIB|unix.IN_DELETE)
		if err != nil {
			return fmt.Errorf("gamepad: InotifyAddWatch failed: %w", err)
		}
		g.watch = watch
	}

	ents, err := os.ReadDir(dirName)
	if err != nil {
		return fmt.Errorf("gamepad: ReadDir(%s) failed: %w", dirName, err)
	}
	for _, ent := range ents {
		if ent.IsDir() {
			continue
		}
		if !reEvent.MatchString(ent.Name()) {
			continue
		}
		if err := g.openGamepad(gamepads, filepath.Join(dirName, ent.Name())); err != nil {
			return err
		}
	}

	return nil
}

func (*nativeGamepadsImpl) openGamepad(gamepads *gamepads, path string) (err error) {
	if gamepads.find(func(gamepad *Gamepad) bool {
		return gamepad.native.(*nativeGamepadImpl).path == path
	}) != nil {
		return nil
	}

	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK, 0)
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
	defer func() {
		if err != nil {
			_ = unix.Close(fd)
		}
	}()

	evBits := make([]byte, (unix.EV_CNT+7)/8)
	keyBits := make([]byte, (_KEY_CNT+7)/8)
	absBits := make([]byte, (_ABS_CNT+7)/8)
	var id input_id
	if err := ioctl(fd, _EVIOCGBIT(0, uint(len(evBits))), unsafe.Pointer(&evBits[0])); err != nil {
		return fmt.Errorf("gamepad: ioctl for evBits failed: %w", err)
	}
	if err := ioctl(fd, _EVIOCGBIT(unix.EV_KEY, uint(len(keyBits))), unsafe.Pointer(&keyBits[0])); err != nil {
		return fmt.Errorf("gamepad: ioctl for keyBits failed: %w", err)
	}
	if err := ioctl(fd, _EVIOCGBIT(unix.EV_ABS, uint(len(absBits))), unsafe.Pointer(&absBits[0])); err != nil {
		return fmt.Errorf("gamepad: ioctl for absBits failed: %w", err)
	}
	if err := ioctl(fd, _EVIOCGID(), unsafe.Pointer(&id)); err != nil {
		return fmt.Errorf("gamepad: ioctl for an ID failed: %w", err)
	}

	if !isBitSet(evBits, unix.EV_ABS) {
		if err := unix.Close(fd); err != nil {
			return err
		}

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

	n := &nativeGamepadImpl{
		path: path,
		fd:   fd,
	}
	gp := gamepads.add(name, sdlID)
	gp.native = n
	runtime.AddCleanup(gp, func(n *nativeGamepadImpl) {
		n.close()
	}, n)

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
			// Write the hat index both for the X and the Y hat axis.
			// That way, the hat can be referenced using either axis, which is used by the code building hatMappingInput.
			n.absMap[code] = hatCount
			code++
			n.absMap[code] = hatCount
			hatCount++
			continue
		}
		if err := ioctl(n.fd, uint(_EVIOCGABS(uint(code))), unsafe.Pointer(&n.absInfo[code])); err != nil {
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
		return err
	}

	return nil
}

func (g *nativeGamepadsImpl) update(gamepads *gamepads) error {
	if g.inotify <= 0 {
		return nil
	}

	buf := make([]byte, 16384)
	n, err := unix.Read(g.inotify, buf[:])
	if err != nil {
		if err == unix.EAGAIN {
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
		name := unix.ByteSliceToString(buf[16 : 16+e.Len-1]) // len includes the null terminate.
		buf = buf[16+e.Len:]
		if !reEvent.MatchString(name) {
			continue
		}

		path := filepath.Join(dirName, name)
		if e.Mask&(unix.IN_CREATE|unix.IN_ATTRIB) != 0 {
			if err := g.openGamepad(gamepads, path); err != nil {
				return err
			}
			continue
		}
		if e.Mask&unix.IN_DELETE != 0 {
			if gp := gamepads.find(func(gamepad *Gamepad) bool {
				return gamepad.native.(*nativeGamepadImpl).path == path
			}); gp != nil {
				gp.native.(*nativeGamepadImpl).close()
				gamepads.remove(func(gamepad *Gamepad) bool {
					return gamepad == gp
				})
			}
			continue
		}
	}

	return nil
}

type nativeGamepadImpl struct {
	fd      int
	path    string
	keyMap  [_KEY_CNT - _BTN_MISC]int
	absMap  [_ABS_CNT]int
	absInfo [_ABS_CNT]input_absinfo
	dropped bool

	axes    [_ABS_CNT]float64
	buttons [_KEY_CNT - _BTN_MISC]bool
	hats    [4]int

	axisCount_   int
	buttonCount_ int
	hatCount_    int

	stdAxisMap   map[gamepaddb.StandardAxis]mappingInput
	stdButtonMap map[gamepaddb.StandardButton]mappingInput
}

func (g *nativeGamepadImpl) close() {
	if g.fd != 0 {
		_ = unix.Close(g.fd)
	}
	g.fd = 0
}

func (g *nativeGamepadImpl) update(gamepad *gamepads) error {
	if g.fd == 0 {
		return nil
	}

	for {
		buf := make([]byte, unsafe.Sizeof(input_event{}))
		// TODO: Should the returned byte count be cared?
		if _, err := unix.Read(g.fd, buf); err != nil {
			if err == unix.EAGAIN {
				break
			}
			// Disconnected
			if err == unix.ENODEV {
				g.close()
				return nil
			}
			return fmt.Errorf("gamepad: Read failed: %w", err)
		}

		const (
			offsetTyp   = unsafe.Offsetof(input_event{}.typ)
			offsetCode  = unsafe.Offsetof(input_event{}.code)
			offsetValue = unsafe.Offsetof(input_event{}.value)
		)
		// time is not used.
		e := input_event{
			typ:   uint16(buf[offsetTyp]) | uint16(buf[offsetTyp+1])<<8,
			code:  uint16(buf[offsetCode]) | uint16(buf[offsetCode+1])<<8,
			value: int32(buf[offsetValue]) | int32(buf[offsetValue+1])<<8 | int32(buf[offsetValue+2])<<16 | int32(buf[offsetValue+3])<<24,
		}

		if e.typ == unix.EV_SYN {
			switch e.code {
			case _SYN_DROPPED:
				g.dropped = true
			case _SYN_REPORT:
				g.dropped = false
				if err := g.pollAbsState(); err != nil {
					return fmt.Errorf("gamepad: poll absolute state: %w", err)
				}
			}
		}
		if g.dropped {
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
	return nil
}

func (g *nativeGamepadImpl) pollAbsState() error {
	for code := 0; code < _ABS_CNT; code++ {
		if g.absMap[code] < 0 {
			continue
		}
		if err := ioctl(g.fd, uint(_EVIOCGABS(uint(code))), unsafe.Pointer(&g.absInfo[code])); err != nil {
			return fmt.Errorf("gamepad: ioctl for an abs at pollAbsState failed: %w", err)
		}
		g.handleAbsEvent(code, g.absInfo[code].value)
	}
	return nil
}

func (g *nativeGamepadImpl) handleAbsEvent(code int, value int32) {
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

	// D-pad can be analog or digital. Prefer digital one.
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
	// TODO: Implement this (#1452)
}
