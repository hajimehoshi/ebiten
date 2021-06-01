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
	"sync"
	"unsafe"
)

func Play(sampleRate, channelNum, bitDepthInBytes int) error {
	if msg := C.ebiten_oboe_Play(C.int(sampleRate), C.int(channelNum), C.int(bitDepthInBytes)); msg != nil {
		return fmt.Errorf("oboe: Play failed: %s", C.GoString(msg))
	}
	return nil
}

func Suspend() error {
	if msg := C.ebiten_oboe_Suspend(); msg != nil {
		return fmt.Errorf("oboe: Suspend failed: %s", C.GoString(msg))
	}
	return nil
}

func Resume() error {
	if msg := C.ebiten_oboe_Resume(); msg != nil {
		return fmt.Errorf("oboe: Resume failed: %s", C.GoString(msg))
	}
	return nil
}

type Player struct {
	player    C.PlayerID
	onWritten func()

	// m is the mutex for this player.
	// This is necessary as Close can be invoked from the finalizer goroutine.
	m sync.Mutex
}

func NewPlayer(volume float64, onWritten func()) *Player {
	p := &Player{
		onWritten: onWritten,
	}
	p.player = C.ebiten_oboe_Player_Create(C.double(volume), C.uintptr_t(uintptr(unsafe.Pointer(p))))
	runtime.SetFinalizer(p, (*Player).Close)
	return p
}

//export ebiten_oboe_onWrittenCallback
func ebiten_oboe_onWrittenCallback(player C.uintptr_t) {
	p := (*Player)(unsafe.Pointer(uintptr(player)))
	p.onWritten()
}

func (p *Player) IsPlaying() bool {
	p.m.Lock()
	defer p.m.Unlock()
	return bool(C.ebiten_oboe_Player_IsPlaying(p.player))
}

func (p *Player) AppendBuffer(buf []byte) {
	p.m.Lock()
	defer p.m.Unlock()

	ptr := C.CBytes(buf)
	defer C.free(ptr)

	C.ebiten_oboe_Player_AppendBuffer(p.player, (*C.uint8_t)(ptr), C.int(len(buf)))
}

func (p *Player) Play() error {
	p.m.Lock()
	defer p.m.Unlock()

	if p.player == 0 {
		return fmt.Errorf("oboe: player is already closed at Play")
	}
	C.ebiten_oboe_Player_Play(p.player)
	return nil
}

func (p *Player) Pause() error {
	p.m.Lock()
	defer p.m.Unlock()

	if p.player == 0 {
		return fmt.Errorf("oboe: player is already closed at Pause")
	}
	C.ebiten_oboe_Player_Pause(p.player)
	return nil
}

func (p *Player) SetVolume(volume float64) {
	p.m.Lock()
	defer p.m.Unlock()
	C.ebiten_oboe_Player_SetVolume(p.player, C.double(volume))
}

func (p *Player) Close() error {
	p.m.Lock()
	defer p.m.Unlock()

	runtime.SetFinalizer(p, nil)
	if p.player == 0 {
		return fmt.Errorf("oboe: player is already closed at Close")
	}
	C.ebiten_oboe_Player_Close(p.player)
	p.player = 0
	return nil
}

func (p *Player) UnplayedBufferSize() int {
	p.m.Lock()
	defer p.m.Unlock()
	return int(C.ebiten_oboe_Player_UnplayedBufferSize(p.player))
}
