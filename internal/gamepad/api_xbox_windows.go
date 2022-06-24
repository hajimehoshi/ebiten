// Copyright 2022 The Ebitengine Authors
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

//go:build !ebitencbackend
// +build !ebitencbackend

package gamepad

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	gameInput = windows.NewLazySystemDLL("GameInput.dll")

	procGameInputCreate = gameInput.NewProc("GameInputCreate")
)

func _GameInputCreate() (*_IGameInput, error) {
	var gameInput *_IGameInput
	r, _, _ := procGameInputCreate.Call(uintptr(unsafe.Pointer(&gameInput)))
	if uint32(r) != uint32(windows.S_OK) {
		return nil, fmt.Errorf("gamepad: GameInputCreate failed: HRESULT(%d)", uint32(r))
	}
	return gameInput, nil
}

type _IGameInput struct {
	vtbl *_IGameInput_Vtbl
}

type _IGameInput_Vtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
}
