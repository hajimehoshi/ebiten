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

//go:build ebitencbackend
// +build ebitencbackend

package cbackend

// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
//
// #include <stdint.h>
//
// struct Gamepad {
//   int id;
//   char standard;
//   int button_num;
//   int axis_num;
//   char button_pressed[32];
//   float button_values[32];
//   float axis_values[16];
// };
//
// struct Touch {
//   int id;
//   int x;
//   int y;
// };
//
// // UI
// void EbitenInitializeGame();
// void EbitenGetScreenSize(int* width, int* height);
// void EbitenBeginFrame();
// void EbitenEndFrame();
//
// // Input
// int EbitenGetGamepadNum();
// void EbitenGetGamepads(struct Gamepad* gamepads);
// int EbitenGetTouchNum();
// void EbitenGetTouches(struct Touch* touches);
// void EbitenVibrateGamepad(int id, double durationInSeconds, double strongMagnitude, double weakMagnitude);
//
// // Audio
// typedef void (*OnReadCallback)(float* buf, size_t length);
// void EbitenOpenAudio(int sample_rate, int channel_num, OnReadCallback on_read_callback);
// void EbitenCloseAudio();
//
// void EbitenAudioOnReadCallback(float* buf, size_t length);
// static void EbitenOpenAudioProxy(int sample_rate, int channel_num) {
//   EbitenOpenAudio(sample_rate, channel_num, EbitenAudioOnReadCallback);
// }
import "C"

import (
	"reflect"
	"time"
	"unsafe"
)

type Gamepad struct {
	ID            int
	Standard      bool
	ButtonCount   int
	AxisCount     int
	ButtonPressed [32]bool
	ButtonValues  [32]float64
	AxisValues    [16]float64
}

type Touch struct {
	ID int
	X  int
	Y  int
}

func InitializeGame() {
	C.EbitenInitializeGame()
}

func ScreenSize() (int, int) {
	var width, height C.int
	C.EbitenGetScreenSize(&width, &height)
	return int(width), int(height)
}

func BeginFrame() {
	C.EbitenBeginFrame()
}

func EndFrame() {
	C.EbitenEndFrame()
}

var cGamepads []C.struct_Gamepad

func AppendGamepads(gamepads []Gamepad) []Gamepad {
	n := int(C.EbitenGetGamepadNum())
	if cap(cGamepads) < n {
		cGamepads = append(cGamepads, make([]C.struct_Gamepad, n)...)
	} else {
		cGamepads = cGamepads[:n]
	}
	if n > 0 {
		C.EbitenGetGamepads(&cGamepads[0])
	}

	for _, g := range cGamepads {
		gamepad := Gamepad{
			ID:          int(g.id),
			Standard:    g.standard != 0,
			ButtonCount: int(g.button_num),
			AxisCount:   int(g.axis_num),
		}
		for i := 0; i < gamepad.ButtonCount; i++ {
			gamepad.ButtonPressed[i] = g.button_pressed[i] != 0
			gamepad.ButtonValues[i] = float64(g.button_values[i])
		}
		for i := 0; i < gamepad.AxisCount; i++ {
			gamepad.AxisValues[i] = float64(g.axis_values[i])
		}

		gamepads = append(gamepads, gamepad)
	}
	return gamepads
}

var cTouches []C.struct_Touch

func AppendTouches(touches []Touch) []Touch {
	n := int(C.EbitenGetTouchNum())
	cTouches = cTouches[:0]
	if cap(cTouches) < n {
		cTouches = append(cTouches, make([]C.struct_Touch, n)...)
	} else {
		cTouches = cTouches[:n]
	}
	if n > 0 {
		C.EbitenGetTouches(&cTouches[0])
	}

	for _, t := range cTouches {
		touches = append(touches, Touch{
			ID: int(t.id),
			X:  int(t.x),
			Y:  int(t.y),
		})
	}
	return touches
}

func VibrateGamepad(id int, duration time.Duration, strongMagnitude float64, weakMagnitude float64) {
	C.EbitenVibrateGamepad(C.int(id), C.double(float64(duration)/float64(time.Second)), C.double(strongMagnitude), C.double(weakMagnitude))
}

var onReadCallback func(buf []float32)

func OpenAudio(sampleRate, channelNum int, onRead func(buf []float32)) {
	C.EbitenOpenAudioProxy(C.int(sampleRate), C.int(channelNum))
	onReadCallback = onRead
}

func CloseAudio() {
	C.EbitenCloseAudio()
}

//export EbitenAudioOnReadCallback
func EbitenAudioOnReadCallback(buf *C.float, length C.size_t) {
	var s []float32
	h := (*reflect.SliceHeader)(unsafe.Pointer(&s))
	h.Data = uintptr(unsafe.Pointer(buf))
	h.Len = int(length)
	h.Cap = int(length)
	onReadCallback(s)
}
