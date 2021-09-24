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

// gamecontrollerdb.txt is downloaded at https://github.com/gabomdq/SDL_GameControllerDB.

// To update the database file, run:
//
//     curl --location --remote-name https://raw.githubusercontent.com/gabomdq/SDL_GameControllerDB/master/gamecontrollerdb.txt

//go:generate file2byteslice -package gamepaddb -input=./gamecontrollerdb.txt -output=./gamecontrollerdb.txt.go -var=gamecontrollerdbTxt

package gamepaddb

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
)

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

	if isIOS {
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
	if _, err := Update(gamecontrollerdbTxt); err != nil {
		panic(err)
	}
	if _, err := Update(additionalGLFWGamepads); err != nil {
		panic(err)
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
	AxisScale  int
	AxisOffset int
	HatState   int
}

var (
	gamepadButtonMappings = map[string]map[driver.StandardGamepadButton]*mapping{}
	gamepadAxisMappings   = map[string]map[driver.StandardGamepadAxis]*mapping{}
	mappingsM             sync.RWMutex
)

func processLine(line string, platform platform) error {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return nil
	}
	if line[0] == '#' {
		return nil
	}
	tokens := strings.Split(line, ",")
	id := tokens[0]
	for _, token := range tokens[2:] {
		if len(token) == 0 {
			continue
		}
		tks := strings.Split(token, ":")
		if tks[0] == "platform" {
			switch tks[1] {
			case "Windows":
				if platform != platformWindows {
					return nil
				}
			case "Mac OS X":
				if platform != platformMacOS {
					return nil
				}
			case "Linux":
				if platform != platformUnix {
					return nil
				}
			case "Android":
				if platform != platformAndroid {
					return nil
				}
			case "iOS":
				if platform != platformIOS {
					return nil
				}
			default:
				return fmt.Errorf("gamepaddb: unexpected platform: %s", tks[1])
			}
			continue
		}

		gb, err := parseMappingElement(tks[1])
		if err != nil {
			return err
		}

		if b, ok := toStandardGamepadButton(tks[0]); ok {
			m, ok := gamepadButtonMappings[id]
			if !ok {
				m = map[driver.StandardGamepadButton]*mapping{}
				gamepadButtonMappings[id] = m
			}
			m[b] = gb
			continue
		}

		if a, ok := toStandardGamepadAxis(tks[0]); ok {
			m, ok := gamepadAxisMappings[id]
			if !ok {
				m = map[driver.StandardGamepadAxis]*mapping{}
				gamepadAxisMappings[id] = m
			}
			m[a] = gb
			continue
		}

		// The buttons like "misc1" are ignored so far.
		// There is no corresponding button in the Web standard gamepad layout.
	}

	return nil
}

func parseMappingElement(str string) (*mapping, error) {
	switch {
	case str[0] == 'a' || strings.HasPrefix(str, "+a") || strings.HasPrefix(str, "-a"):
		var tilda bool
		if str[len(str)-1] == '~' {
			str = str[:len(str)-1]
			tilda = true
		}

		min := -1
		max := 1
		numstr := str[1:]

		if str[0] == '+' {
			numstr = str[2:]
			min = 0
		} else if str[0] == '-' {
			numstr = str[2:]
			max = 0
		}

		scale := 2 / (max - min)
		offset := -(max + min)
		if tilda {
			scale = -scale
			offset = -offset
		}

		index, err := strconv.Atoi(numstr)
		if err != nil {
			return nil, err
		}

		return &mapping{
			Type:       mappingTypeAxis,
			Index:      index,
			AxisScale:  scale,
			AxisOffset: offset,
		}, nil

	case str[0] == 'b':
		index, err := strconv.Atoi(str[1:])
		if err != nil {
			return nil, err
		}
		return &mapping{
			Type:  mappingTypeButton,
			Index: index,
		}, nil

	case str[0] == 'h':
		tokens := strings.Split(str[1:], ".")
		if len(tokens) < 2 {
			return nil, fmt.Errorf("gamepaddb: unexpected hat: %s", str)
		}
		index, err := strconv.Atoi(tokens[0])
		if err != nil {
			return nil, err
		}
		hat, err := strconv.Atoi(tokens[1])
		if err != nil {
			return nil, err
		}
		return &mapping{
			Type:     mappingTypeHat,
			Index:    index,
			HatState: hat,
		}, nil
	}

	return nil, fmt.Errorf("gamepaddb: unepxected mapping: %s", str)
}

func toStandardGamepadButton(str string) (driver.StandardGamepadButton, bool) {
	switch str {
	case "a":
		return driver.StandardGamepadButtonRightBottom, true
	case "b":
		return driver.StandardGamepadButtonRightRight, true
	case "x":
		return driver.StandardGamepadButtonRightLeft, true
	case "y":
		return driver.StandardGamepadButtonRightTop, true
	case "back":
		return driver.StandardGamepadButtonCenterLeft, true
	case "start":
		return driver.StandardGamepadButtonCenterRight, true
	case "guide":
		return driver.StandardGamepadButtonCenterCenter, true
	case "leftshoulder":
		return driver.StandardGamepadButtonFrontTopLeft, true
	case "rightshoulder":
		return driver.StandardGamepadButtonFrontTopRight, true
	case "leftstick":
		return driver.StandardGamepadButtonLeftStick, true
	case "rightstick":
		return driver.StandardGamepadButtonRightStick, true
	case "dpup":
		return driver.StandardGamepadButtonLeftTop, true
	case "dpright":
		return driver.StandardGamepadButtonLeftRight, true
	case "dpdown":
		return driver.StandardGamepadButtonLeftBottom, true
	case "dpleft":
		return driver.StandardGamepadButtonLeftLeft, true
	case "lefttrigger":
		return driver.StandardGamepadButtonFrontBottomLeft, true
	case "righttrigger":
		return driver.StandardGamepadButtonFrontBottomRight, true
	default:
		return 0, false
	}
}

func toStandardGamepadAxis(str string) (driver.StandardGamepadAxis, bool) {
	switch str {
	case "leftx":
		return driver.StandardGamepadAxisLeftStickHorizontal, true
	case "lefty":
		return driver.StandardGamepadAxisLeftStickVertical, true
	case "rightx":
		return driver.StandardGamepadAxisRightStickHorizontal, true
	case "righty":
		return driver.StandardGamepadAxisRightStickVertical, true
	default:
		return 0, false
	}
}

func HasStandardLayoutMapping(id string) bool {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	if _, ok := gamepadButtonMappings[id]; ok {
		return true
	}
	if _, ok := gamepadAxisMappings[id]; ok {
		return true
	}
	return false
}

type GamepadState interface {
	Axis(index int) float64
	Button(index int) bool
	Hat(index int) int
}

func AxisValue(id string, axis driver.StandardGamepadAxis, state GamepadState) float64 {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	mappings, ok := gamepadAxisMappings[id]
	if !ok {
		return 0
	}

	mapping := mappings[axis]
	if mapping == nil {
		return 0
	}

	switch mapping.Type {
	case mappingTypeAxis:
		v := state.Axis(mapping.Index)*float64(mapping.AxisScale) + float64(mapping.AxisOffset)
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

func ButtonValue(id string, button driver.StandardGamepadButton, state GamepadState) float64 {
	mappingsM.RLock()
	defer mappingsM.RUnlock()

	return buttonValue(id, button, state)
}

func buttonValue(id string, button driver.StandardGamepadButton, state GamepadState) float64 {
	mappings, ok := gamepadButtonMappings[id]
	if !ok {
		return 0
	}

	mapping := mappings[button]
	if mapping == nil {
		return 0
	}

	switch mapping.Type {
	case mappingTypeAxis:
		v := state.Axis(mapping.Index)*float64(mapping.AxisScale) + float64(mapping.AxisOffset)
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

func IsButtonPressed(id string, button driver.StandardGamepadButton, state GamepadState) bool {
	// Use XInput's trigger dead zone.
	// See https://source.chromium.org/chromium/chromium/src/+/main:device/gamepad/public/cpp/gamepad.h;l=22-23;drc=6997f8a177359bb99598988ed5e900841984d242
	const threshold = 30.0 / 255.0

	mappingsM.RLock()
	defer mappingsM.RUnlock()

	mappings, ok := gamepadButtonMappings[id]
	if !ok {
		return false
	}

	mapping := mappings[button]
	if mapping == nil {
		return false
	}

	switch mapping.Type {
	case mappingTypeAxis:
		v := buttonValue(id, button, state)
		return v > threshold
	case mappingTypeButton:
		return state.Button(mapping.Index)
	case mappingTypeHat:
		return state.Hat(mapping.Index)&mapping.HatState != 0
	}

	return false
}

// Update adds new gamepad mappings.
// The string must be in the format of SDL_GameControllerDB.
func Update(mapping []byte) (bool, error) {
	if currentPlatform == platformUnknown {
		return false, nil
	}

	// TODO: Implement this (#1557)
	if currentPlatform == platformAndroid || currentPlatform == platformIOS {
		// Note: NOT returning an error, as mappings also do not matter right now.
		return false, nil
	}

	mappingsM.Lock()
	defer mappingsM.Unlock()

	buf := bytes.NewBuffer(mapping)
	r := bufio.NewReader(buf)
	for {
		line, err := r.ReadString('\n')
		if err != nil && err != io.EOF {
			return false, err
		}
		if err := processLine(line, currentPlatform); err != nil {
			return false, err
		}
		if err == io.EOF {
			break
		}
	}

	return true, nil
}
