// Copyright 2016 Hajime Hoshi
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

package ebitenmobileview

import (
	"unicode"
)

func UpdateTouchesOnAndroid(action int, id int, x, y int) {
	switch action {
	case 0x00, 0x05, 0x02: // ACTION_DOWN, ACTION_POINTER_DOWN, ACTION_MOVE
		touches[id] = position{x, y}
		updateInput()
	case 0x01, 0x06: // ACTION_UP, ACTION_POINTER_UP
		delete(touches, id)
		updateInput()
	}
}

// UpdateTouchesOnIOS is a dummy function for backward compatibility.
// UpdateTouchesOnIOS is called from ebiten/mobile package.
func UpdateTouchesOnIOS(phase int, ptr int64, x, y int) {
	panic("ebitenmobileview: updateTouchesOnIOSImpl must not be called on Android")
}

func OnKeyDownOnAndroid(keyCode int, unicodeChar int) {
	key, ok := androidKeyToDriverKey[keyCode]
	if !ok {
		return
	}
	keys[key] = struct{}{}
	if r := rune(unicodeChar); r != 0 && unicode.IsPrint(r) {
		runes = []rune{r}
	}
	updateInput()
}

func OnKeyUpOnAndroid(keyCode int) {
	key, ok := androidKeyToDriverKey[keyCode]
	if !ok {
		return
	}
	delete(keys, key)
	updateInput()
}
