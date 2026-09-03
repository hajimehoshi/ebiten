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

//go:generate go run gen.go
//go:generate gofmt -s -w .

package gamepaddb

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

type platform int

const (
	platformUnknown platform = iota
	platformWindows
	platformDarwin
	platformUnix
	platformAndroid
)

func currentPlatform() platform {
	switch runtime.GOOS {
	case "windows":
		return platformWindows
	case "aix", "dragonfly", "freebsd", "hurd", "illumos", "linux", "netbsd", "openbsd", "solaris":
		return platformUnix
	case "android":
		return platformAndroid
	case "ios":
		return platformDarwin
	case "darwin":
		return platformDarwin
	default:
		return platformUnknown
	}
}

type mappingType int

const (
	mappingTypeButton mappingType = iota
	mappingTypeAxis
	mappingTypeHat
)

const (
	HatUp    = 1
	HatRight = 2
	HatDown  = 4
	HatLeft  = 8
)

type mapping struct {
	Type       mappingType
	Index      int
	AxisScale  float64
	AxisOffset float64
	HatState   int
}

var (
	gamepadNames          = map[string]string{}
	gamepadButtonMappings = map[string]map[StandardButton]mapping{}
	gamepadAxisMappings   = map[string]map[StandardAxis]mapping{}
	mappingsM             sync.Mutex
)

func parseLine(line string, platform platform) (id string, name string, buttons map[StandardButton]mapping, axes map[StandardAxis]mapping, err error) {
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
				if platform != platformDarwin {
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
				if platform != platformDarwin {
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
				buttons = map[StandardButton]mapping{}
			}
			buttons[b] = gb
			continue
		}

		if a, ok := toStandardGamepadAxis(tks[0]); ok {
			if axes == nil {
				axes = map[StandardAxis]mapping{}
			}
			axes[a] = gb
			continue
		}

		// The buttons like "misc1" are ignored so far.
		// There is no corresponding button in the Web standard gamepad layout.
	}

	return tokens[0], tokens[1], buttons, axes, nil
}

func parseMappingElement(str string) (mapping, error) {
	if len(str) == 0 {
		return mapping{}, fmt.Errorf("gamepaddb: unexpected empty mapping")
	}
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
			return mapping{}, err
		}

		return mapping{
			Type:       mappingTypeAxis,
			Index:      index,
			AxisScale:  scale,
			AxisOffset: offset,
		}, nil

	case str[0] == 'b':
		index, err := strconv.Atoi(str[1:])
		if err != nil {
			return mapping{}, err
		}
		return mapping{
			Type:  mappingTypeButton,
			Index: index,
		}, nil

	case str[0] == 'h':
		tokens := strings.Split(str[1:], ".")
		if len(tokens) < 2 {
			return mapping{}, fmt.Errorf("gamepaddb: unexpected hat: %s", str)
		}
		index, err := strconv.Atoi(tokens[0])
		if err != nil {
			return mapping{}, err
		}
		hat, err := strconv.Atoi(tokens[1])
		if err != nil {
			return mapping{}, err
		}
		return mapping{
			Type:     mappingTypeHat,
			Index:    index,
			HatState: hat,
		}, nil
	}

	return mapping{}, fmt.Errorf("gamepaddb: unexpected mapping: %s", str)
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

// buttonMappings returns the button mappings for the given id.
// The caller must hold mappingsM, as this can add the Android default mappings.
func buttonMappings(id string) map[StandardButton]mapping {
	if m, ok := gamepadButtonMappings[id]; ok {
		return m
	}
	if currentPlatform() == platformAndroid {
		if addAndroidDefaultMappings(id) {
			return gamepadButtonMappings[id]
		}
	}
	return nil
}

// axisMappings returns the axis mappings for the given id.
// The caller must hold mappingsM, as this can add the Android default mappings.
func axisMappings(id string) map[StandardAxis]mapping {
	if m, ok := gamepadAxisMappings[id]; ok {
		return m
	}
	if currentPlatform() == platformAndroid {
		if addAndroidDefaultMappings(id) {
			return gamepadAxisMappings[id]
		}
	}
	return nil
}

// Mapping is the standard-layout mapping of one standard axis or button of a gamepad.
// The zero Mapping maps nothing.
//
// Evaluating a Mapping takes no lock of this package, so a caller can hold its own gamepad lock
// while doing so and read one consistent gamepad state.
type Mapping struct {
	mapping mapping

	// hasStandardLayout reports that the gamepad has a standard layout mapping. This can be true even
	// when the resolved axis or button is not a part of it.
	hasStandardLayout bool

	// mapped reports that mapping is meaningful.
	mapped bool
}

// HasStandardLayout reports whether the gamepad has a standard layout mapping.
func (m Mapping) HasStandardLayout() bool {
	return m.hasStandardLayout
}

// IsMapped reports whether the standard axis or button is mapped to an input of the gamepad.
func (m Mapping) IsMapped() bool {
	return m.mapped
}

// StandardAxisMapping returns the mapping of the standard axis for the gamepad id.
func StandardAxisMapping(id string, axis StandardAxis) Mapping {
	mappingsM.Lock()
	defer mappingsM.Unlock()

	buttons := buttonMappings(id)
	axes := axisMappings(id)
	m := Mapping{
		hasStandardLayout: buttons != nil || axes != nil,
	}
	if a, ok := axes[axis]; ok {
		m.mapping = a
		m.mapped = true
	}
	return m
}

// StandardButtonMapping returns the mapping of the standard button for the gamepad id.
func StandardButtonMapping(id string, button StandardButton) Mapping {
	mappingsM.Lock()
	defer mappingsM.Unlock()

	buttons := buttonMappings(id)
	axes := axisMappings(id)
	m := Mapping{
		hasStandardLayout: buttons != nil || axes != nil,
	}
	if b, ok := buttons[button]; ok {
		m.mapping = b
		m.mapped = true
	}
	return m
}

// AxisValue returns the value of the standard axis in the range -1 to 1, or 0 when the axis is not
// mapped.
func (m Mapping) AxisValue(state GamepadState) float64 {
	if !m.mapped {
		return 0
	}
	return standardAxisValue(m.mapping, state)
}

// ButtonValue returns the value of the standard button in the range 0 to 1, or 0 when the button is
// not mapped.
func (m Mapping) ButtonValue(state GamepadState) float64 {
	if !m.mapped {
		return 0
	}
	return standardButtonValue(m.mapping, state)
}

// IsButtonPressed reports whether the standard button is pressed. An unmapped button is not pressed.
func (m Mapping) IsButtonPressed(state GamepadState) bool {
	if !m.mapped {
		return false
	}
	return isStandardButtonPressed(m.mapping, state)
}

func HasStandardLayoutMapping(id string) bool {
	mappingsM.Lock()
	defer mappingsM.Unlock()

	return buttonMappings(id) != nil || axisMappings(id) != nil
}

// GamepadState represents a gamepad state given by a gamepad driver.
//
// The methods must not be called while mappingsM is held.
type GamepadState interface {
	IsAxisReady(index int) bool
	Axis(index int) float64
	Button(index int) bool
	Hat(index int) int
}

func Name(id string) string {
	mappingsM.Lock()
	defer mappingsM.Unlock()

	return gamepadNames[id]
}

// standardAxisValue calculates the value for the given mapping, which is passed by value.
func standardAxisValue(mapping mapping, state GamepadState) float64 {
	switch mapping.Type {
	case mappingTypeAxis:
		if !state.IsAxisReady(mapping.Index) {
			return 0
		}
		v := state.Axis(mapping.Index)*mapping.AxisScale + mapping.AxisOffset
		if v > 1 {
			return 1
		} else if v < -1 {
			return -1
		}
		return v
	case mappingTypeButton:
		if state.Button(mapping.Index) {
			return 1
		} else {
			return -1
		}
	case mappingTypeHat:
		if state.Hat(mapping.Index)&mapping.HatState != 0 {
			return 1
		} else {
			return -1
		}
	}

	return 0
}

// standardButtonValue calculates the value for the given mapping, which is passed by value.
func standardButtonValue(mapping mapping, state GamepadState) float64 {
	switch mapping.Type {
	case mappingTypeAxis:
		if !state.IsAxisReady(mapping.Index) {
			return 0
		}
		v := state.Axis(mapping.Index)*mapping.AxisScale + mapping.AxisOffset
		if v > 1 {
			v = 1
		} else if v < -1 {
			v = -1
		}
		// Adjust [-1, 1] to [0, 1]
		return (v + 1) / 2
	case mappingTypeButton:
		if state.Button(mapping.Index) {
			return 1
		}
		return 0
	case mappingTypeHat:
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

// isStandardButtonPressed reports whether the button is pressed for the given mapping, which is
// passed by value.
func isStandardButtonPressed(mapping mapping, state GamepadState) bool {
	switch mapping.Type {
	case mappingTypeAxis:
		v := standardButtonValue(mapping, state)
		return v > ButtonPressedThreshold
	case mappingTypeButton:
		return state.Button(mapping.Index)
	case mappingTypeHat:
		return state.Hat(mapping.Index)&mapping.HatState != 0
	}

	return false
}

// Update adds new gamepad mappings.
// The string must be in the format of SDL_GameControllerDB.
//
// Update works atomically. If an error happens, nothing is updated.
func Update(mappingData []byte) error {
	mappingsM.Lock()
	defer mappingsM.Unlock()

	type parsedLine struct {
		id      string
		name    string
		buttons map[StandardButton]mapping
		axes    map[StandardAxis]mapping
	}
	var lines []parsedLine

	for line := range bytes.Lines(mappingData) {
		id, name, buttons, axes, err := parseLine(string(line), currentPlatform())
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

	for _, l := range lines {
		// A line that maps neither a button nor an axis defines no mapping. Registering it would make
		// a mapping exist with no content, and would drop an existing mapping for the same ID.
		if l.buttons == nil && l.axes == nil {
			continue
		}
		gamepadNames[l.id] = l.name
		gamepadButtonMappings[l.id] = l.buttons
		gamepadAxisMappings[l.id] = l.axes
	}

	return nil
}

// addAndroidDefaultMappings adds default mappings for the given Android gamepad ID
// and reports whether the mappings were added.
//
// The gamepad database has entries for Android gamepads, but Android devices are too
// varied for the database to cover them all. An Android gamepad ID is generated when
// the device is added, from the button and axis masks that the system reports, so a
// mapping for a device missing from the database can be derived from its ID.
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
	if len(idBytes) < 16 {
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

	gamepadButtonMappings[id] = map[StandardButton]mapping{}
	gamepadAxisMappings[id] = map[StandardAxis]mapping{}

	// For mappings, see mobile/ebitenmobileview/input_android.go.

	if buttonMask&(1<<SDLControllerButtonA) != 0 {
		gamepadButtonMappings[id][StandardButtonRightBottom] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonA,
		}
	}
	if buttonMask&(1<<SDLControllerButtonB) != 0 {
		gamepadButtonMappings[id][StandardButtonRightRight] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonB,
		}
	} else {
		// Use the back button as "B" for easy UI navigation with TV remotes.
		gamepadButtonMappings[id][StandardButtonRightRight] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonBack,
		}
		buttonMask &^= uint16(1) << SDLControllerButtonBack
	}
	if buttonMask&(1<<SDLControllerButtonX) != 0 {
		gamepadButtonMappings[id][StandardButtonRightLeft] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonX,
		}
	}
	if buttonMask&(1<<SDLControllerButtonY) != 0 {
		gamepadButtonMappings[id][StandardButtonRightTop] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonY,
		}
	}
	if buttonMask&(1<<SDLControllerButtonBack) != 0 {
		gamepadButtonMappings[id][StandardButtonCenterLeft] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonBack,
		}
	}
	if buttonMask&(1<<SDLControllerButtonGuide) != 0 {
		// TODO: If SDKVersion >= 30, add this code:
		//
		//     gamepadButtonMappings[id][StandardButtonCenterCenter] = mapping{
		//         Type:  mappingTypeButton,
		//         Index: SDLControllerButtonGuide,
		//     }
	}
	if buttonMask&(1<<SDLControllerButtonStart) != 0 {
		gamepadButtonMappings[id][StandardButtonCenterRight] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonStart,
		}
	}
	if buttonMask&(1<<SDLControllerButtonLeftStick) != 0 {
		gamepadButtonMappings[id][StandardButtonLeftStick] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonLeftStick,
		}
	}
	if buttonMask&(1<<SDLControllerButtonRightStick) != 0 {
		gamepadButtonMappings[id][StandardButtonRightStick] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonRightStick,
		}
	}
	if buttonMask&(1<<SDLControllerButtonLeftShoulder) != 0 {
		gamepadButtonMappings[id][StandardButtonFrontTopLeft] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonLeftShoulder,
		}
	}
	if buttonMask&(1<<SDLControllerButtonRightShoulder) != 0 {
		gamepadButtonMappings[id][StandardButtonFrontTopRight] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonRightShoulder,
		}
	}

	if buttonMask&(1<<SDLControllerButtonDpadUp) != 0 {
		gamepadButtonMappings[id][StandardButtonLeftTop] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonDpadUp,
		}
	}
	if buttonMask&(1<<SDLControllerButtonDpadDown) != 0 {
		gamepadButtonMappings[id][StandardButtonLeftBottom] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonDpadDown,
		}
	}
	if buttonMask&(1<<SDLControllerButtonDpadLeft) != 0 {
		gamepadButtonMappings[id][StandardButtonLeftLeft] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonDpadLeft,
		}
	}
	if buttonMask&(1<<SDLControllerButtonDpadRight) != 0 {
		gamepadButtonMappings[id][StandardButtonLeftRight] = mapping{
			Type:  mappingTypeButton,
			Index: SDLControllerButtonDpadRight,
		}
	}

	if axisMask&(1<<SDLControllerAxisLeftX) != 0 {
		gamepadAxisMappings[id][StandardAxisLeftStickHorizontal] = mapping{
			Type:       mappingTypeAxis,
			Index:      SDLControllerAxisLeftX,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}
	if axisMask&(1<<SDLControllerAxisLeftY) != 0 {
		gamepadAxisMappings[id][StandardAxisLeftStickVertical] = mapping{
			Type:       mappingTypeAxis,
			Index:      SDLControllerAxisLeftY,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}
	if axisMask&(1<<SDLControllerAxisRightX) != 0 {
		gamepadAxisMappings[id][StandardAxisRightStickHorizontal] = mapping{
			Type:       mappingTypeAxis,
			Index:      SDLControllerAxisRightX,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}
	if axisMask&(1<<SDLControllerAxisRightY) != 0 {
		gamepadAxisMappings[id][StandardAxisRightStickVertical] = mapping{
			Type:       mappingTypeAxis,
			Index:      SDLControllerAxisRightY,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}
	if axisMask&(1<<SDLControllerAxisTriggerLeft) != 0 {
		gamepadButtonMappings[id][StandardButtonFrontBottomLeft] = mapping{
			Type:       mappingTypeAxis,
			Index:      SDLControllerAxisTriggerLeft,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}
	if axisMask&(1<<SDLControllerAxisTriggerRight) != 0 {
		gamepadButtonMappings[id][StandardButtonFrontBottomRight] = mapping{
			Type:       mappingTypeAxis,
			Index:      SDLControllerAxisTriggerRight,
			AxisScale:  1,
			AxisOffset: 0,
		}
	}

	return true
}
