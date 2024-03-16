// Copyright 2021 The Ebiten Authors
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

// gamecontrollerdb.txt is downloaded at https://github.com/mdqinc/SDL_GameControllerDB.

// To update the database file, run:
//
//     curl --location --remote-name https://raw.githubusercontent.com/mdqinc/SDL_GameControllerDB/master/gamecontrollerdb.txt

package gamepaddb

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/hex"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

//go:embed gamecontrollerdb.txt
var gamecontrollerdb_txt []byte

type platform int

const (
	platformUnknown platform = iota
	platformWindows
	platformMacOS
	platformUnix
	platformAndroid
	platformIOS
)

var currentPlatform platform

func init() {
	if runtime.GOOS == "windows" {
		currentPlatform = platformWindows
		return
	}

	if runtime.GOOS == "aix" ||
		runtime.GOOS == "dragonfly" ||
		runtime.GOOS == "freebsd" ||
		runtime.GOOS == "hurd" ||
		runtime.GOOS == "illumos" ||
		runtime.GOOS == "linux" ||
		runtime.GOOS == "netbsd" ||
		runtime.GOOS == "openbsd" ||
		runtime.GOOS == "solaris" {
		currentPlatform = platformUnix
		return
	}

	if runtime.GOOS == "android" {
		currentPlatform = platformAndroid
		return
	}

	if runtime.GOOS == "ios" {
		currentPlatform = platformIOS
		return
	}

	if runtime.GOOS == "darwin" {
		currentPlatform = platformMacOS
		return
	}
}

var additionalGLFWGamepads = []byte(`
78696e70757401000000000000000000,XInput Gamepad (GLFW),platform:Windows,a:b0,b:b1,x:b2,y:b3,leftshoulder:b4,rightshoulder:b5,back:b6,start:b7,leftstick:b8,rightstick:b9,leftx:a0,lefty:a1,rightx:a2,righty:a3,lefttrigger:a4,righttrigger:a5,dpup:h0.1,dpright:h0.2,dpdown:h0.4,dpleft:h0.8,
78696e70757402000000000000000000,XInput Wheel (GLFW),platform:Windows,a:b0,b:b1,x:b2,y:b3,leftshoulder:b4,rightshoulder:b5,back:b6,start:b7,leftstick:b8,rightstick:b9,leftx:a0,lefty:a1,rightx:a2,righty:a3,lefttrigger:a4,righttrigger:a5,dpup:h0.1,dpright:h0.2,dpdown:h0.4,dpleft:h0.8,
78696e70757403000000000000000000,XInput Arcade Stick (GLFW),platform:Windows,a:b0,b:b1,x:b2,y:b3,leftshoulder:b4,rightshoulder:b5,back:b6,start:b7,leftstick:b8,rightstick:b9,leftx:a0,lefty:a1,rightx:a2,righty:a3,lefttrigger:a4,righttrigger:a5,dpup:h0.1,dpright:h0.2,dpdown:h0.4,dpleft:h0.8,
78696e70757404000000000000000000,XInput Flight Stick (GLFW),platform:Windows,a:b0,b:b1,x:b2,y:b3,leftshoulder:b4,rightshoulder:b5,back:b6,start:b7,leftstick:b8,rightstick:b9,leftx:a0,lefty:a1,rightx:a2,righty:a3,lefttrigger:a4,righttrigger:a5,dpup:h0.1,dpright:h0.2,dpdown:h0.4,dpleft:h0.8,
78696e70757405000000000000000000,XInput Dance Pad (GLFW),platform:Windows,a:b0,b:b1,x:b2,y:b3,leftshoulder:b4,rightshoulder:b5,back:b6,start:b7,leftstick:b8,rightstick:b9,leftx:a0,lefty:a1,rightx:a2,righty:a3,lefttrigger:a4,righttrigger:a5,dpup:h0.1,dpright:h0.2,dpdown:h0.4,dpleft:h0.8,
78696e70757406000000000000000000,XInput Guitar (GLFW),platform:Windows,a:b0,b:b1,x:b2,y:b3,leftshoulder:b4,rightshoulder:b5,back:b6,start:b7,leftstick:b8,rightstick:b9,leftx:a0,lefty:a1,rightx:a2,righty:a3,lefttrigger:a4,righttrigger:a5,dpup:h0.1,dpright:h0.2,dpdown:h0.4,dpleft:h0.8,
78696e70757408000000000000000000,XInput Drum Kit (GLFW),platform:Windows,a:b0,b:b1,x:b2,y:b3,leftshoulder:b4,rightshoulder:b5,back:b6,start:b7,leftstick:b8,rightstick:b9,leftx:a0,lefty:a1,rightx:a2,righty:a3,lefttrigger:a4,righttrigger:a5,dpup:h0.1,dpright:h0.2,dpdown:h0.4,dpleft:h0.8,
`)

func init() {
	if err := Update(gamecontrollerdb_txt); err != nil {
		panic(err)
	}
	if err := Update(additionalGLFWGamepads); err != nil {
		panic(err)
	}
}

type MappingType int

const (
	MappingTypeButton MappingType = iota
	MappingTypeAxis
	MappingTypeHat
)

const (
	HatUp    = 1
	HatRight = 2
	HatDown  = 4
	HatLeft  = 8
)

type MappingItem struct {
	Type       MappingType
	Index      int
	AxisScale  float64
	AxisOffset float64
	HatState   int
}

var (
	gamepadNames          = map[string]string{}
	gamepadButtonMappings = map[string]map[StandardButton]MappingItem{}
	gamepadAxisMappings   = map[string]map[StandardAxis]MappingItem{}
	mappingsM             sync.RWMutex
)

func parseLine(line string, platform platform) (id string, name string, buttons map[StandardButton]MappingItem, axes map[StandardAxis]MappingItem, err error) {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return "", "", nil, nil, nil
	}
	if line[0] == '#' {
		return "", "", nil, nil, nil
	}
	tokens := strings.Split(line, ",")
	if len(tokens) < 2 {
		return "", "", nil, nil, fmt.Errorf("gamepaddb: syntax error")
	}

	for _, token := range tokens[2:] {
		if len(token) == 0 {
			continue
		}
		tks := strings.Split(token, ":")
		if len(tks) < 2 {
			return "", "", nil, nil, fmt.Errorf("gamepaddb: syntax error")
		}

		// Note that the platform part is listed in the definition of SDL_GetPlatform.
		if tks[0] == "platform" {
			switch tks[1] {
			case "Windows":
				if platform != platformWindows {
					return "", "", nil, nil, nil
				}
			case "Mac OS X":
				if platform != platformMacOS {
					return "", "", nil, nil, nil
				}
			case "Linux":
				if platform != platformUnix {
					return "", "", nil, nil, nil
				}
			case "Android":
				if platform != platformAndroid {
					return "", "", nil, nil, nil
				}
			case "iOS":
				if platform != platformIOS {
					return "", "", nil, nil, nil
				}
			case "":
				// Allow any platforms
			default:
				return "", "", nil, nil, fmt.Errorf("gamepaddb: unexpected platform: %s", tks[1])
			}
			continue
		}

		gb, err := parseMappingElement(tks[1])
		if err != nil {
			return "", "", nil, nil, err
		}

		if b, ok := toStandardGamepadButton(tks[0]); ok {
			if buttons == nil {
				buttons = map[StandardButton]MappingItem{}
			}
			buttons[b] = gb
			continue
		}

		if a, ok := toStandardGamepadAxis(tks[0]); ok {
			if axes == nil {
				axes = map[StandardAxis]MappingItem{}
			}
			axes[a] = gb
			continue
		}

		// The buttons like "misc1" are ignored so far.
		// There is no corresponding button in the Web standard gamepad layout.
	}

	return tokens[0], tokens[1], buttons, axes, nil
}

func parseMappingElement(str string) (MappingItem, error) {
	switch {
	case str[0] == 'a' || strings.HasPrefix(str, "+a") || strings.HasPrefix(str, "-a"):
		var tilda bool
		if str[len(str)-1] == '~' {
			str = str[:len(str)-1]
			tilda = true
		}

		min := -1.0
		max := 1.0
		numstr := str[1:]

		if str[0] == '+' {
			numstr = str[2:]
			// Only use the positive half, i.e. 0..1.
			min = 0
		} else if str[0] == '-' {
			numstr = str[2:]
			// Only use the negative half, i.e. -1..0,
			// but invert the sense so 0 does not "press" buttons.
			//
			// In other words, this is the same as '+' but with the input axis
			// value reversed.
			//
			// See SDL's source:
			// https://github.com/libsdl-org/SDL/blob/f398d8a42422c049d77c744658f1cd2bb011ed4a/src/joystick/SDL_gamecontroller.c#L960
			min, max = 0, min
		}

		// Map min..max to -1..+1.
		//
		// See SDL's source:
		// https://github.com/libsdl-org/SDL/blob/f398d8a42422c049d77c744658f1cd2bb011ed4a/src/joystick/SDL_gamecontroller.c#L276
		// then simplify assuming output range -1..+1.
		//
		// Yields:
		scale := 2 / (max - min)
		offset := -(max + min) / (max - min)
		if tilda {
			scale = -scale
			offset = -offset
		}

		index, err := strconv.Atoi(numstr)
		if err != nil {
			return MappingItem{}, err
		}

		return MappingItem{
			Type:       MappingTypeAxis,
			Index:      index,
			AxisScale:  scale,
			AxisOffset: offset,
		}, nil

	case str[0] == 'b':
		index, err := strconv.Atoi(str[1:])
		if err != nil {
			return MappingItem{}, err
		}
		return MappingItem{
			Type:  MappingTypeButton,
			Index: index,
		}, nil

	case str[0] == 'h':
		tokens := strings.Split(str[1:], ".")
		if len(tokens) < 2 {
			return MappingItem{}, fmt.Errorf("gamepaddb: unexpected hat: %s", str)
		}
		index, err := strconv.Atoi(tokens[0])
		if err != nil {
			return MappingItem{}, err
		}
		hat, err := strconv.Atoi(tokens[1])
		if err != nil {
			return MappingItem{}, err
		}
		return MappingItem{
			Type:     MappingTypeHat,
			Index:    index,
			HatState: hat,
		}, nil
	}

	return MappingItem{}, fmt.Errorf("gamepaddb: unepxected mapping: %s", str)
}

func toStandardGamepadButton(str string) (StandardButton, bool) {
	switch str {
	case "a":
		return StandardButtonRightBottom, true
	case "b":
		return StandardButtonRightRight, true
	case "x":
		return StandardButtonRightLeft, true
	case "y":
		return StandardButtonRightTop, true
	case "back":
		return StandardButtonCenterLeft, true
	case "start":
		return StandardButtonCenterRight, true
	case "guide":
		return StandardButtonCenterCenter, true
	case "leftshoulder":
		return StandardButtonFrontTopLeft, true
	case "rightshoulder":
		return StandardButtonFrontTopRight, true
	case "leftstick":
		return StandardButtonLeftStick, true
	case "rightstick":
		return StandardButtonRightStick, true
	case "dpup":
		return StandardButtonLeftTop, true
	case "dpright":
		return StandardButtonLeftRight, true
	case "dpdown":
		return StandardButtonLeftBottom, true
	case "dpleft":
		return StandardButtonLeftLeft, true
	case "lefttrigger":
		return StandardButtonFrontBottomLeft, true
	case "righttrigger":
		return StandardButtonFrontBottomRight, true
	default:
		return 0, false
	}
}

func toStandardGamepadAxis(str string) (StandardAxis, bool) {
	switch str {
	case "leftx":
		return StandardAxisLeftStickHorizontal, true
	case "lefty":
		return StandardAxisLeftStickVertical, true
	case "rightx":
		return StandardAxisRightStickHorizontal, true
	case "righty":
		return StandardAxisRightStickVertical, true
	default:
		return 0, false
	}
}

func buttonMapping(id string) map[StandardButton]MappingItem {
	if m, ok := gamepadButtonMappings[id]; ok {
		return m
	}
	if currentPlatform == platformAndroid {
		if addAndroidDefaultMappings(id) {
			return gamepadButtonMappings[id]
		}
	}
	return nil
}

func axisMapping(id string) map[StandardAxis]MappingItem {
	if m, ok := gamepadAxisMappings[id]; ok {
		return m
	}
	if currentPlatform == platformAndroid {
		if addAndroidDefaultMappings(id) {
			return gamepadAxisMappings[id]
		}
	}
	return nil
}

func HasStandardLayoutMapping(id string) bool {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	return buttonMapping(id) != nil || axisMapping(id) != nil
}

type GamepadState interface {
	Axis(index int) float64
	Button(index int) bool
	Hat(index int) int
}

func Name(id string) string {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	return gamepadNames[id]
}

func HasStandardAxis(id string, axis StandardAxis) bool {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	mappings := axisMapping(id)
	if mappings == nil {
		return false
	}
	_, ok := mappings[axis]
	return ok
}

func StandardAxisValue(id string, axis StandardAxis, state GamepadState) float64 {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	mappings := axisMapping(id)
	if mappings == nil {
		return 0
	}

	mapping, ok := mappings[axis]
	if !ok {
		return 0
	}

	switch mapping.Type {
	case MappingTypeAxis:
		v := state.Axis(mapping.Index)*mapping.AxisScale + mapping.AxisOffset
		if v > 1 {
			return 1
		} else if v < -1 {
			return -1
		}
		return v
	case MappingTypeButton:
		if state.Button(mapping.Index) {
			return 1
		} else {
			return -1
		}
	case MappingTypeHat:
		if state.Hat(mapping.Index)&mapping.HatState != 0 {
			return 1
		} else {
			return -1
		}
	}

	return 0
}

func HasStandardButton(id string, button StandardButton) bool {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	mappings := buttonMapping(id)
	if mappings == nil {
		return false
	}
	_, ok := mappings[button]
	return ok
}

func StandardButtonValue(id string, button StandardButton, state GamepadState) float64 {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	return standardButtonValue(id, button, state)
}

func standardButtonValue(id string, button StandardButton, state GamepadState) float64 {
	mappings := buttonMapping(id)
	if mappings == nil {
		return 0
	}

	mapping, ok := mappings[button]
	if !ok {
		return 0
	}

	switch mapping.Type {
	case MappingTypeAxis:
		v := state.Axis(mapping.Index)*mapping.AxisScale + mapping.AxisOffset
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		// Adjust [-1, 1] to [0, 1]
		return (v + 1) / 2
	case MappingTypeButton:
		if state.Button(mapping.Index) {
			return 1
		}
		return 0
	case MappingTypeHat:
		if state.Hat(mapping.Index)&mapping.HatState != 0 {
			return 1
		}
		return 0
	}

	return 0
}

// ButtonPressedThreshold represents the value up to which a button counts as not yet pressed.
// This has been set to match XInput's trigger dead zone.
// See https://source.chromium.org/chromium/chromium/src/+/main:device/gamepad/public/cpp/gamepad.h;l=22-23;drc=6997f8a177359bb99598988ed5e900841984d242
// Note: should be used with >, not >=, comparisons.
const ButtonPressedThreshold = 30.0 / 255.0

func IsStandardButtonPressed(id string, button StandardButton, state GamepadState) bool {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	mappings, ok := gamepadButtonMappings[id]
	if !ok {
		return false
	}

	mapping, ok := mappings[button]
	if !ok {
		return false
	}

	switch mapping.Type {
	case MappingTypeAxis:
		v := standardButtonValue(id, button, state)
		return v > ButtonPressedThreshold
	case MappingTypeButton:
		return state.Button(mapping.Index)
	case MappingTypeHat:
		return state.Hat(mapping.Index)&mapping.HatState != 0
	}

	return false
}

// UnsafeMapping returns the mapping of the gamepad.
// UnsafeMapping is unsafe. The returned values must not be modified.
func UnsafeMapping(id string) (map[StandardButton]MappingItem, map[StandardAxis]MappingItem) {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	return buttonMapping(id), axisMapping(id)
}

// Update adds new gamepad mappings.
// The string must be in the format of SDL_GameControllerDB.
//
// Update works atomically. If an error happens, nothing is updated.
func Update(mappingData []byte) error {
	mappingsM.Lock()
	defer mappingsM.Unlock()

	buf := bytes.NewBuffer(mappingData)
	s := bufio.NewScanner(buf)

	type parsedLine struct {
		id      string
		name    string
		buttons map[StandardButton]MappingItem
		axes    map[StandardAxis]MappingItem
	}
	var lines []parsedLine

	for s.Scan() {
		line := s.Text()
		id, name, buttons, axes, err := parseLine(line, currentPlatform)
		if err != nil {
			return err
		}
		if id != "" {
			lines = append(lines, parsedLine{
				id:      id,
				name:    name,
				buttons: buttons,
				axes:    axes,
			})
		}
	}

	if err := s.Err(); err != nil {
		return err
	}

	for _, l := range lines {
		gamepadNames[l.id] = l.name
		gamepadButtonMappings[l.id] = l.buttons
		gamepadAxisMappings[l.id] = l.axes
	}

	return nil
}

func addAndroidDefaultMappings(id string) bool {
	// See https://github.com/libsdl-org/SDL/blob/120c76c84bbce4c1bfed4e9eb74e10678bd83120/src/joystick/SDL_gamecontroller.c#L468-L568

	const faceButtonMask = ((1 << SDLControllerButtonA) |
		(1 << SDLControllerButtonB) |
		(1 << SDLControllerButtonX) |
		(1 << SDLControllerButtonY))

	idBytes, err := hex.DecodeString(id)
	if err != nil {
		return false
	}
	buttonMask := uint16(idBytes[12]) | (uint16(idBytes[13]) << 8)
	axisMask := uint16(idBytes[14]) | (uint16(idBytes[15]) << 8)
	if buttonMask == 0 && axisMask == 0 {
		return false
	}
	if buttonMask&faceButtonMask == 0 {
		return false
	}

	gamepadButtonMappings[id] = map[StandardButton]MappingItem{}
	gamepadAxisMappings[id] = map[StandardAxis]MappingItem{}

	// For mappings, see mobile/ebitenmobileview/input_android.go.

	if buttonMask&(1<<SDLControllerButtonA) != 0 {
		gamepadButtonMappings[id][StandardButtonRightBottom] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonA,
		}
	}
	if buttonMask&(1<<SDLControllerButtonB) != 0 {
		gamepadButtonMappings[id][StandardButtonRightRight] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonB,
		}
	} else {
		// Use the back button as "B" for easy UI navigation with TV remotes.
		gamepadButtonMappings[id][StandardButtonRightRight] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonBack,
		}
		buttonMask &^= uint16(1) << SDLControllerButtonBack
	}
	if buttonMask&(1<<SDLControllerButtonX) != 0 {
		gamepadButtonMappings[id][StandardButtonRightLeft] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonX,
		}
	}
	if buttonMask&(1<<SDLControllerButtonY) != 0 {
		gamepadButtonMappings[id][StandardButtonRightTop] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonY,
		}
	}
	if buttonMask&(1<<SDLControllerButtonBack) != 0 {
		gamepadButtonMappings[id][StandardButtonCenterLeft] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonBack,
		}
	}
	if buttonMask&(1<<SDLControllerButtonGuide) != 0 {
		// TODO: If SDKVersion >= 30, add this code:
		//
		//     gamepadButtonMappings[id][StandardButtonCenterCenter] = MappingItem{
		//         Type:  mappingTypeButton,
		//         Index: SDLControllerButtonGuide,
		//     }
	}
	if buttonMask&(1<<SDLControllerButtonStart) != 0 {
		gamepadButtonMappings[id][StandardButtonCenterRight] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonStart,
		}
	}
	if buttonMask&(1<<SDLControllerButtonLeftStick) != 0 {
		gamepadButtonMappings[id][StandardButtonLeftStick] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonLeftStick,
		}
	}
	if buttonMask&(1<<SDLControllerButtonRightStick) != 0 {
		gamepadButtonMappings[id][StandardButtonRightStick] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonRightStick,
		}
	}
	if buttonMask&(1<<SDLControllerButtonLeftShoulder) != 0 {
		gamepadButtonMappings[id][StandardButtonFrontTopLeft] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonLeftShoulder,
		}
	}
	if buttonMask&(1<<SDLControllerButtonRightShoulder) != 0 {
		gamepadButtonMappings[id][StandardButtonFrontTopRight] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonRightShoulder,
		}
	}

	if buttonMask&(1<<SDLControllerButtonDpadUp) != 0 {
		gamepadButtonMappings[id][StandardButtonLeftTop] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonDpadUp,
		}
	}
	if buttonMask&(1<<SDLControllerButtonDpadDown) != 0 {
		gamepadButtonMappings[id][StandardButtonLeftBottom] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonDpadDown,
		}
	}
	if buttonMask&(1<<SDLControllerButtonDpadLeft) != 0 {
		gamepadButtonMappings[id][StandardButtonLeftLeft] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonDpadLeft,
		}
	}
	if buttonMask&(1<<SDLControllerButtonDpadRight) != 0 {
		gamepadButtonMappings[id][StandardButtonLeftRight] = MappingItem{
			Type:  MappingTypeButton,
			Index: SDLControllerButtonDpadRight,
		}
	}

	if axisMask&(1<<SDLControllerAxisLeftX) != 0 {
		gamepadAxisMappings[id][StandardAxisLeftStickHorizontal] = MappingItem{
			Type:       MappingTypeAxis,
			Index:      SDLControllerAxisLeftX,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}
	if axisMask&(1<<SDLControllerAxisLeftY) != 0 {
		gamepadAxisMappings[id][StandardAxisLeftStickVertical] = MappingItem{
			Type:       MappingTypeAxis,
			Index:      SDLControllerAxisLeftY,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}
	if axisMask&(1<<SDLControllerAxisRightX) != 0 {
		gamepadAxisMappings[id][StandardAxisRightStickHorizontal] = MappingItem{
			Type:       MappingTypeAxis,
			Index:      SDLControllerAxisRightX,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}
	if axisMask&(1<<SDLControllerAxisRightY) != 0 {
		gamepadAxisMappings[id][StandardAxisRightStickVertical] = MappingItem{
			Type:       MappingTypeAxis,
			Index:      SDLControllerAxisRightY,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}
	if axisMask&(1<<SDLControllerAxisTriggerLeft) != 0 {
		gamepadButtonMappings[id][StandardButtonFrontBottomLeft] = MappingItem{
			Type:       MappingTypeAxis,
			Index:      SDLControllerAxisTriggerLeft,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}
	if axisMask&(1<<SDLControllerAxisTriggerRight) != 0 {
		gamepadButtonMappings[id][StandardButtonFrontBottomRight] = MappingItem{
			Type:       MappingTypeAxis,
			Index:      SDLControllerAxisTriggerRight,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}

	return true
}
