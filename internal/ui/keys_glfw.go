// Copyright 2015 Hajime Hoshi
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

// +build !js

package ui

import (
	glfw "github.com/go-gl/glfw3"
)

var glfwKeyCodeToKey = map[glfw.Key]Key{
	glfw.Key0:           Key0,
	glfw.Key1:           Key1,
	glfw.Key2:           Key2,
	glfw.Key3:           Key3,
	glfw.Key4:           Key4,
	glfw.Key5:           Key5,
	glfw.Key6:           Key6,
	glfw.Key7:           Key7,
	glfw.Key8:           Key8,
	glfw.Key9:           Key9,
	glfw.KeyA:           KeyA,
	glfw.KeyB:           KeyB,
	glfw.KeyC:           KeyC,
	glfw.KeyCapsLock:    KeyCapsLock,
	glfw.KeyComma:       KeyComma,
	glfw.KeyD:           KeyD,
	glfw.KeyDelete:      KeyDelete,
	glfw.KeyDown:        KeyDown,
	glfw.KeyE:           KeyE,
	glfw.KeyEnd:         KeyEnd,
	glfw.KeyEnter:       KeyEnter,
	glfw.KeyEscape:      KeyEscape,
	glfw.KeyF:           KeyF,
	glfw.KeyF1:          KeyF1,
	glfw.KeyF10:         KeyF10,
	glfw.KeyF11:         KeyF11,
	glfw.KeyF12:         KeyF12,
	glfw.KeyF2:          KeyF2,
	glfw.KeyF3:          KeyF3,
	glfw.KeyF4:          KeyF4,
	glfw.KeyF5:          KeyF5,
	glfw.KeyF6:          KeyF6,
	glfw.KeyF7:          KeyF7,
	glfw.KeyF8:          KeyF8,
	glfw.KeyF9:          KeyF9,
	glfw.KeyG:           KeyG,
	glfw.KeyH:           KeyH,
	glfw.KeyHome:        KeyHome,
	glfw.KeyI:           KeyI,
	glfw.KeyInsert:      KeyInsert,
	glfw.KeyJ:           KeyJ,
	glfw.KeyK:           KeyK,
	glfw.KeyL:           KeyL,
	glfw.KeyLeft:        KeyLeft,
	glfw.KeyLeftAlt:     KeyLeftAlt,
	glfw.KeyLeftControl: KeyLeftControl,
	glfw.KeyLeftShift:   KeyLeftShift,
	glfw.KeyM:           KeyM,
	glfw.KeyN:           KeyN,
	glfw.KeyO:           KeyO,
	glfw.KeyP:           KeyP,
	glfw.KeyPageDown:    KeyPageDown,
	glfw.KeyPageUp:      KeyPageUp,
	glfw.KeyPeriod:      KeyPeriod,
	glfw.KeyQ:           KeyQ,
	glfw.KeyR:           KeyR,
	glfw.KeyRight:       KeyRight,
	glfw.KeyS:           KeyS,
	glfw.KeySpace:       KeySpace,
	glfw.KeyT:           KeyT,
	glfw.KeyTab:         KeyTab,
	glfw.KeyU:           KeyU,
	glfw.KeyUp:          KeyUp,
	glfw.KeyV:           KeyV,
	glfw.KeyW:           KeyW,
	glfw.KeyX:           KeyX,
	glfw.KeyY:           KeyY,
	glfw.KeyZ:           KeyZ,
}
