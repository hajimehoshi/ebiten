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

// +build ios

package ebitenmobileview

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2/internal/driver"
)

// #cgo CFLAGS: -x objective-c
// #cgo LDFLAGS: -framework Foundation -framework UIKit
//
// #import <UIKit/UIKit.h>
import "C"

var ptrToID = map[int64]int{}

func getIDFromPtr(ptr int64) int {
	if id, ok := ptrToID[ptr]; ok {
		return id
	}
	maxID := 0
	for _, id := range ptrToID {
		if maxID < id {
			maxID = id
		}
	}
	id := maxID + 1
	ptrToID[ptr] = id
	return id
}

func UpdateTouchesOnIOS(phase int, ptr int64, x, y int) {
	switch phase {
	case C.UITouchPhaseBegan, C.UITouchPhaseMoved, C.UITouchPhaseStationary:
		id := getIDFromPtr(ptr)
		touches[driver.TouchID(id)] = position{x, y}
		updateInput()
	case C.UITouchPhaseEnded, C.UITouchPhaseCancelled:
		id := getIDFromPtr(ptr)
		delete(ptrToID, ptr)
		delete(touches, driver.TouchID(id))
		updateInput()
	default:
		panic(fmt.Sprintf("ebitenmobileview: invalid phase: %d", phase))
	}
}
