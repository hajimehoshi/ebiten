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

// Package ebitenmobileview offers functions for OpenGL/Metal view of mobiles.
//
// The functions are not intended for public usages.
// There is no guarantee of backward compatibility.
package ebitenmobileview

// #include <stdint.h>
import "C"

import (
	"runtime"
)

type ViewRectSetter interface {
	SetViewRect(x, y, width, height int)
}

func Layout(viewWidth, viewHeight int, viewRectSetter ViewRectSetter) {
	var x, y, width, height C.int
	ebitenLayout(C.int(viewWidth), C.int(viewHeight), &x, &y, &width, &height)
	if viewRectSetter != nil {
		viewRectSetter.SetViewRect(int(x), int(y), int(width), int(height))
	}
}

func Update() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	theState.m.Lock()
	defer theState.m.Unlock()

	return update()
}

func UpdateTouchesOnAndroid(action int, id int, x, y int) {
	ebitenUpdateTouchesOnAndroid((C.int)(action), (C.int)(id), (C.int)(x), (C.int)(y))
}

func UpdateTouchesOnIOS(phase int, ptr int64, x, y int) {
	ebitenUpdateTouchesOnIOS((C.int)(phase), (C.uintptr_t)(ptr), (C.int)(x), (C.int)(y))
}
