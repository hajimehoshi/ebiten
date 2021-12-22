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
//
// // Audio
// // TODO: Implement mixing on Go side and reduce the API.
// typedef void (*OnWrittenCallback)(int id);
// void EbitenOpenAudio(int sample_rate, int channel_num, int bit_depth_in_bytes);
// void EbitenCloseAudio();
// int EbitenCreateAudioPlayer(OnWrittenCallback on_written_callback);
// void EbitenAudioPlayerPlay(int id);
// void EbitenAudioPlayerPause(int id);
// void EbitenAudioPlayerWrite(int id, uint8_t* data, int length);
// void EbitenAudioPlayerClose(int id, int immediately);
// double EbitenAudioPlayerGetVolume(int id);
// void EbitenAudioPlayerSetVolume(int id, double volume);
// int EbitenAudioPlayerGetUnplayedBufferSize(int id);
// float EbitenAudioBufferSizeInSeconds();
//
// void EbitenAudioPlayerOnWrittenCallback(int id);
// static int EbitenCreateAudioPlayerProxy() {
//   return EbitenCreateAudioPlayer(EbitenAudioPlayerOnWrittenCallback);
// }
import "C"

import (
	"runtime"
	"sync"
	"unsafe"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
)

type Gamepad struct {
	ID            driver.GamepadID
	Standard      bool
	ButtonNum     int
	AxisNum       int
	ButtonPressed [32]bool
	ButtonValues  [32]float64
	AxisValues    [16]float64
}

type Touch struct {
	ID driver.TouchID
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
			ID:        driver.GamepadID(g.id),
			Standard:  g.standard != 0,
			ButtonNum: int(g.button_num),
			AxisNum:   int(g.axis_num),
		}
		for i := 0; i < gamepad.ButtonNum; i++ {
			gamepad.ButtonPressed[i] = g.button_pressed[i] != 0
			gamepad.ButtonValues[i] = float64(g.button_values[i])
		}
		for i := 0; i < gamepad.AxisNum; i++ {
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
			ID: driver.TouchID(t.id),
			X:  int(t.x),
			Y:  int(t.y),
		})
	}
	return touches
}

func OpenAudio(sampleRate, channelNum, bitDepthInBytes int) {
	C.EbitenOpenAudio(C.int(sampleRate), C.int(channelNum), C.int(bitDepthInBytes))
}

func CloseAudio() {
	C.EbitenCloseAudio()
}

func CreateAudioPlayer(onWritten func()) *AudioPlayer {
	id := C.EbitenCreateAudioPlayerProxy()
	p := &AudioPlayer{
		id: id,
	}
	onWrittenCallbacksM.Lock()
	defer onWrittenCallbacksM.Unlock()
	onWrittenCallbacks[id] = onWritten
	return p
}

func AudioBufferSizeInSeconds() float64 {
	return float64(C.EbitenAudioBufferSizeInSeconds())
}

type AudioPlayer struct {
	id C.int
}

func (p *AudioPlayer) Play() {
	C.EbitenAudioPlayerPlay(p.id)
}

func (p *AudioPlayer) Pause() {
	C.EbitenAudioPlayerPause(p.id)
}

func (p *AudioPlayer) Write(buf []byte) {
	C.EbitenAudioPlayerWrite(p.id, (*C.uint8_t)(unsafe.Pointer(&buf[0])), C.int(len(buf)))
	runtime.KeepAlive(buf)
}

func (p *AudioPlayer) Close(immediately bool) {
	var i C.int
	if immediately {
		i = 1
	}
	C.EbitenAudioPlayerClose(p.id, i)

	onWrittenCallbacksM.Lock()
	defer onWrittenCallbacksM.Unlock()
	delete(onWrittenCallbacks, p.id)
}

func (p *AudioPlayer) Volume() float64 {
	return float64(C.EbitenAudioPlayerGetVolume(p.id))
}

func (p *AudioPlayer) SetVolume(volume float64) {
	C.EbitenAudioPlayerSetVolume(p.id, C.double(volume))
}

func (p *AudioPlayer) UnplayedBufferSize() int {
	return int(C.EbitenAudioPlayerGetUnplayedBufferSize(p.id))
}

var (
	onWrittenCallbacks  = map[C.int]func(){}
	onWrittenCallbacksM sync.Mutex
)

//export EbitenAudioPlayerOnWrittenCallback
func EbitenAudioPlayerOnWrittenCallback(id C.int) {
	onWrittenCallbacksM.Lock()
	defer onWrittenCallbacksM.Unlock()
	if c, ok := onWrittenCallbacks[id]; ok {
		c()
	}
}
