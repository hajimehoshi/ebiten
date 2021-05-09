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

package oboe

// Disable AAudio (#1634).
// AAudio doesn't care about plugging in/out of a headphone.
// See https://github.com/google/oboe/blob/master/docs/notes/disconnect.md

// #cgo CXXFLAGS: -std=c++17 -DOBOE_ENABLE_AAUDIO=0
// #cgo LDFLAGS: -llog -lOpenSLES -static-libstdc++
//
// #include "binding_android.h"
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

func Suspend() error {
	if msg := C.Suspend(); msg != nil {
		return fmt.Errorf("oboe: Suspend failed: %s", C.GoString(msg))
	}
	return nil
}

func Resume() error {
	if msg := C.Resume(); msg != nil {
		return fmt.Errorf("oboe: Resume failed: %s", C.GoString(msg))
	}
	return nil
}

type Player struct {
	player    C.PlayerID
	onWritten func()
}

func NewPlayer(sampleRate, channelNum, bitDepthInBytes int, volume float64, onWritten func()) *Player {
	p := &Player{
		onWritten: onWritten,
	}
	p.player = C.Player_Create(C.int(sampleRate), C.int(channelNum), C.int(bitDepthInBytes), C.double(volume), C.uintptr_t(uintptr(unsafe.Pointer(p))))
	runtime.SetFinalizer(p, (*Player).Close)
	return p
}

//export onWrittenCallback
func onWrittenCallback(player C.uintptr_t) {
	p := (*Player)(unsafe.Pointer(uintptr(player)))
	p.onWritten()
}

func (p *Player) IsPlaying() bool {
	return bool(C.Player_IsPlaying(p.player))
}

func (p *Player) AppendBuffer(buf []byte) {
	ptr := C.CBytes(buf)
	defer C.free(ptr)

	C.Player_AppendBuffer(p.player, (*C.uint8_t)(ptr), C.int(len(buf)))
}

func (p *Player) Play() error {
	if msg := C.Player_Play(p.player); msg != nil {
		return fmt.Errorf("oboe: Player_Play failed: %s", C.GoString(msg))
	}
	return nil
}

func (p *Player) Pause() error {
	if msg := C.Player_Pause(p.player); msg != nil {
		return fmt.Errorf("oboe: Player_Pause failed: %s", C.GoString(msg))
	}
	return nil
}

func (p *Player) SetVolume(volume float64) {
	C.Player_SetVolume(p.player, C.double(volume))
}

func (p *Player) Close() error {
	runtime.SetFinalizer(p, nil)
	if msg := C.Player_Close(p.player); msg != nil {
		return fmt.Errorf("oboe: Player_Close failed: %s", C.GoString(msg))
	}
	return nil
}

func (p *Player) UnplayedBufferSize() int {
	return int(C.Player_UnplayedBufferSize(p.player))
}
