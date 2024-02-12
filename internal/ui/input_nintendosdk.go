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

//go:build nintendosdk

package ui

// #cgo !darwin LDFLAGS: -Wl,-unresolved-symbols=ignore-all
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
//
// #include "input_nintendosdk.h"
//
// const int kScreenWidth = 1920;
// const int kScreenHeight = 1080;
import "C"

import (
	"github.com/hajimehoshi/ebiten/v2/internal/gamepad"
)

func (u *UserInterface) updateInputState() error {
	var err error
	u.mainThread.Call(func() {
		err = u.updateInputStateImpl()
	})
	return err
}

// updateInputStateImpl must be called from the main thread.
func (u *UserInterface) updateInputStateImpl() error {
	if err := gamepad.Update(); err != nil {
		return err
	}

	C.ebitengine_UpdateTouches()

	u.nativeTouches = u.nativeTouches[:0]
	if n := int(C.ebitengine_GetTouchCount()); n > 0 {
		if cap(u.nativeTouches) < n {
			u.nativeTouches = make([]C.struct_Touch, n)
		} else {
			u.nativeTouches = u.nativeTouches[:n]
		}
		C.ebitengine_GetTouches(&u.nativeTouches[0])
	}

	u.m.Lock()
	defer u.m.Unlock()

	u.inputState.Touches = u.inputState.Touches[:0]
	for _, t := range u.nativeTouches {
		x, y := u.context.clientPositionToLogicalPosition(float64(t.x), float64(t.y), theMonitor.DeviceScaleFactor())
		u.inputState.Touches = append(u.inputState.Touches, Touch{
			ID: TouchID(t.id),
			X:  int(x),
			Y:  int(y),
		})
	}

	return nil
}

func (u *UserInterface) KeyName(key Key) string {
	return ""
}
